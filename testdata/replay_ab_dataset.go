//go:build ignore

// replay_ab_dataset.go is a live integration test/tool for meshy-growth-backend.
// It replays a real sample of the Udacity "Analyze A/B Test Results" dataset
// (testdata/ab_data.csv, sourced from https://github.com/30lm32/ml-ab-testing)
// through the actual running HTTP services (event-gateway + experiment-service),
// then reads back the computed stats from experiment-service and checks that
// they exactly match the counts we fed in.
//
// Usage (with `docker compose up -d` already running):
//
//	go run replay_ab_dataset.go -sample 4000
//
// It creates a dedicated experiment "real_dataset_replay_<timestamp>" so runs
// are idempotent/repeatable and do not collide with the demo seed data.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type row struct {
	userID      string
	group       string
	landingPage string
	converted   bool
}

func main() {
	sample := flag.Int("sample", 4000, "number of (deduplicated, consistent) rows to replay")
	gatewayURL := flag.String("gateway", "http://localhost:8081", "event-gateway base URL")
	expServiceURL := flag.String("exp", "http://localhost:8082", "experiment-service base URL")
	flag.Parse()

	rows := loadDataset("ab_data.csv")
	log.Printf("loaded %d clean rows from ab_data.csv", len(rows))

	rand.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	if *sample < len(rows) {
		rows = rows[:*sample]
	}

	expKey := fmt.Sprintf("real_dataset_replay_%d", time.Now().Unix())
	client := &http.Client{Timeout: 10 * time.Second}

	createExperiment(client, *expServiceURL, expKey)

	// Bucketing (internal/experiment.PickVariant) is a deterministic hash of
	// (expKey, userID), so we cannot force a dataset row's original group to
	// land in the matching live variant. Instead we record whichever variant
	// the live system actually assigns for each replayed user, and use the
	// dataset's own `converted` flag for that row as ground truth for whether
	// a purchase event should be sent. Expected per-variant totals are then
	// accumulated from the *live* assignment responses, giving an exact,
	// deterministic oracle to check the live experiment-service stats against.
	want := map[string]struct{ assigned, converted int }{}
	for _, r := range rows {
		userID := r.group + "-" + r.userID // dedupe across control/treatment id collisions

		variant := assignExperiment(client, *gatewayURL, expKey, userID)
		if variant == "" {
			continue
		}
		w := want[variant]
		w.assigned++
		if r.converted {
			sendPurchaseEvent(client, *gatewayURL, userID)
			w.converted++
		}
		want[variant] = w
	}

	log.Println("waiting for Kafka pipeline to drain...")
	time.Sleep(8 * time.Second)

	stats := fetchStats(client, *expServiceURL, expKey)
	printAndVerify(stats, want)
}

func loadDataset(path string) []row {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil {
		log.Fatal(err)
	}

	seen := map[string]bool{}
	var out []row
	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		userID, group, landingPage, convertedStr := rec[0], rec[2], rec[3], rec[4]
		if seen[userID] {
			continue
		}
		seen[userID] = true
		if (group == "control" && landingPage != "old_page") || (group == "treatment" && landingPage != "new_page") {
			continue
		}
		out = append(out, row{userID: userID, group: group, landingPage: landingPage, converted: convertedStr == "1"})
	}
	return out
}

func createExperiment(client *http.Client, baseURL, key string) {
	body := map[string]any{
		"key":                key,
		"name":               "Real dataset replay",
		"traffic_allocation": 100,
		"variants": []map[string]any{
			{"key": "control", "name": "Control (old_page)", "weight": 1},
			{"key": "treatment", "name": "Treatment (new_page)", "weight": 1},
		},
	}
	payload, _ := json.Marshal(body)
	resp, err := client.Post(baseURL+"/v1/experiments", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("create experiment: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("create experiment: unexpected status %d", resp.StatusCode)
	}
	log.Printf("created experiment %q", key)
}


type assignResponse struct {
	Variant *string `json:"variant"`
}

func assignExperiment(client *http.Client, gatewayURL, expKey, userID string) string {
	url := fmt.Sprintf("%s/v1/experiments/%s/assign?user_id=%s", gatewayURL, expKey, userID)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("assign error for %s: %v", userID, err)
		return ""
	}
	defer resp.Body.Close()
	var ar assignResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		log.Printf("decode assign response for %s: %v", userID, err)
		return ""
	}
	if ar.Variant == nil {
		return ""
	}
	return *ar.Variant
}

func sendPurchaseEvent(client *http.Client, gatewayURL, userID string) {
	body := map[string]any{
		"event_name": "purchase",
		"user_id":    userID,
	}
	payload, _ := json.Marshal(body)
	resp, err := client.Post(gatewayURL+"/v1/events", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("send event error for %s: %v", userID, err)
		return
	}
	defer resp.Body.Close()
}

type variantStats struct {
	VariantKey      string   `json:"variant_key"`
	Assigned        int64    `json:"assigned"`
	Converted       int64    `json:"converted"`
	ConversionRate  float64  `json:"conversion_rate"`
	PValueVsControl *float64 `json:"p_value_vs_control,omitempty"`
}

type statsResponse struct {
	Variants []variantStats `json:"variants"`
	SRMCheck *struct {
		ChiSquare float64 `json:"chi_square"`
		PValue    float64 `json:"p_value"`
		Flagged   bool    `json:"flagged"`
	} `json:"srm_check,omitempty"`
}

func fetchStats(client *http.Client, baseURL, key string) statsResponse {
	url := fmt.Sprintf("%s/v1/experiments/%s/stats?goal_event=purchase", baseURL, key)
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("fetch stats: %v", err)
	}
	defer resp.Body.Close()
	var s statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		log.Fatalf("decode stats: %v", err)
	}
	return s
}

func printAndVerify(s statsResponse, want map[string]struct{ assigned, converted int }) {
	fmt.Println("=== Live experiment-service response ===")
	raw, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(raw))

	fmt.Println("\n=== Expected (from live assignment responses fed back into the replay) ===")
	for variant, w := range want {
		rate := 0.0
		if w.assigned > 0 {
			rate = float64(w.converted) / float64(w.assigned)
		}
		fmt.Printf("%s: assigned=%d converted=%d rate=%.4f\n", variant, w.assigned, w.converted, rate)
	}

	ok := true
	for _, v := range s.Variants {
		w, present := want[v.VariantKey]
		if !present {
			fmt.Printf("MISMATCH: live response has variant %q not in expected map\n", v.VariantKey)
			ok = false
			continue
		}
		if v.Assigned != int64(w.assigned) {
			fmt.Printf("MISMATCH: %s assigned = %d, want %d\n", v.VariantKey, v.Assigned, w.assigned)
			ok = false
		}
		if v.Converted != int64(w.converted) {
			fmt.Printf("MISMATCH: %s converted = %d, want %d\n", v.VariantKey, v.Converted, w.converted)
			ok = false
		}
	}
	if ok {
		fmt.Println("\nPASS: end-to-end replay through live event-gateway + experiment-service matches expected counts exactly.")
	} else {
		fmt.Println("\nFAIL: see mismatches above.")
		os.Exit(1)
	}
}


