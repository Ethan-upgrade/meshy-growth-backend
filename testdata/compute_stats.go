//go:build ignore

// compute_stats.go computes control/treatment conversion aggregates from the
// real Udacity "Analyze A/B Test Results" dataset (ab_data.csv, sourced from
// https://github.com/30lm32/ml-ab-testing) so the numbers can be used as an
// independent, real-world test vector for internal/stats functions.
//
// Run with: go run compute_stats.go
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
)

func main() {
	f, err := os.Open("ab_data.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("header:", header)

	var controlAssigned, controlConverted, treatmentAssigned, treatmentConverted int64
	seen := map[string]bool{}
	dupes := 0
	mismatched := 0 // group/landing_page inconsistent rows (known data quality issue in this dataset)

	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		userID := rec[0]
		group := rec[2]
		landingPage := rec[3]
		converted := rec[4] == "1"

		if seen[userID] {
			dupes++
			continue
		}
		seen[userID] = true

		if (group == "control" && landingPage != "old_page") || (group == "treatment" && landingPage != "new_page") {
			mismatched++
			continue
		}

		switch group {
		case "control":
			controlAssigned++
			if converted {
				controlConverted++
			}
		case "treatment":
			treatmentAssigned++
			if converted {
				treatmentConverted++
			}
		}
	}

	fmt.Printf("duplicate user_id rows skipped: %d\n", dupes)
	fmt.Printf("mismatched group/landing_page rows skipped: %d\n", mismatched)
	fmt.Printf("control:   assigned=%d converted=%d rate=%.4f\n", controlAssigned, controlConverted, float64(controlConverted)/float64(controlAssigned))
	fmt.Printf("treatment: assigned=%d converted=%d rate=%.4f\n", treatmentAssigned, treatmentConverted, float64(treatmentConverted)/float64(treatmentAssigned))
}
