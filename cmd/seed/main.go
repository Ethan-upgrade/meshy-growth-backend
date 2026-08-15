package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
)

func main() {
	cfg := config.Load()
	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	gatewayURL := getEnv("EVENT_GATEWAY_URL", "http://event-gateway:8080")
	expServiceURL := getEnv("EXPERIMENT_SERVICE_URL", "http://experiment-service:8080")

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHTTP(client, gatewayURL, 60*time.Second)
	waitForHTTP(client, expServiceURL, 60*time.Second)

	campaignIDs := seedCampaigns(conn)
	seedExperiment(client, expServiceURL)

	rand.Seed(time.Now().UnixNano())

	eventNames := []string{"page_view", "signup", "add_to_cart", "purchase"}
	platforms := []string{"meta", "google", "tiktok"}

	const numUsers = 200
	log.Printf("seeding traffic for %d users...", numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("demo-user-%d", i)
		platform := platforms[rand.Intn(len(platforms))]
		campaignID := campaignIDs[platform]
		clickID := fmt.Sprintf("%s-click-%d", platform, i)

		assignExperiment(client, gatewayURL, userID)

		numEvents := 1 + rand.Intn(len(eventNames))
		for j := 0; j < numEvents; j++ {
			sendEvent(client, gatewayURL, eventNames[j], userID, clickID, platform, campaignID)
			time.Sleep(20 * time.Millisecond)
		}
	}
	log.Println("seeding complete")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func waitForHTTP(client *http.Client, baseURL string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(time.Second)
	}
	log.Printf("warning: %s did not become healthy within %s", baseURL, timeout)
}

func seedCampaigns(conn *sql.DB) map[string]int64 {
	campaigns := []struct {
		Name       string
		Platform   string
		ExternalID string
	}{
		{"Meta Demo Campaign", "meta", "meta-ext-001"},
		{"Google Demo Campaign", "google", "google-ext-001"},
		{"TikTok Demo Campaign", "tiktok", "tiktok-ext-001"},
	}
	ids := map[string]int64{}
	for _, c := range campaigns {
		var id int64
		err := conn.QueryRow(`SELECT id FROM campaigns WHERE external_campaign_id = ?`, c.ExternalID).Scan(&id)
		if err == sql.ErrNoRows {
			res, err := conn.Exec(
				`INSERT INTO campaigns (name, ad_platform, external_campaign_id, attribution_window_days, is_active) VALUES (?, ?, ?, 7, TRUE)`,
				c.Name, c.Platform, c.ExternalID)
			if err != nil {
				log.Fatalf("seed campaign: %v", err)
			}
			id, _ = res.LastInsertId()
		} else if err != nil {
			log.Fatalf("query campaign: %v", err)
		}
		ids[c.Platform] = id
	}
	return ids
}

func seedExperiment(client *http.Client, expServiceURL string) {
	body := map[string]any{
		"key":                 "checkout_button_color",
		"name":                "Checkout Button Color Test",
		"traffic_allocation":  100,
		"variants": []map[string]any{
			{"key": "control", "name": "Blue Button", "weight": 1},
			{"key": "treatment", "name": "Green Button", "weight": 1},
		},
	}
	postJSON(client, expServiceURL+"/v1/experiments", body)
}

func assignExperiment(client *http.Client, gatewayURL, userID string) {
	url := fmt.Sprintf("%s/v1/experiments/checkout_button_color/assign?user_id=%s", gatewayURL, userID)
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func sendEvent(client *http.Client, gatewayURL, eventName, userID, clickID, platform string, campaignID int64) {
	body := map[string]any{
		"event_name":  eventName,
		"user_id":     userID,
		"click_id":    clickID,
		"ad_platform": platform,
		"campaign_id": campaignID,
		"properties":  map[string]any{"value": rand.Intn(100)},
	}
	postJSON(client, gatewayURL+"/v1/events", body)
}

func postJSON(client *http.Client, url string, body map[string]any) {
	payload, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("POST %s error: %v", url, err)
		return
	}
	defer resp.Body.Close()
}
