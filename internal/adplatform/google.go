package adplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GoogleClient stub-implements the Google Ads click conversion upload shape.
// https://developers.google.com/google-ads/api/docs/conversions/upload-clicks
type GoogleClient struct{}

func (GoogleClient) Name() string { return "google" }

func (GoogleClient) SendConversion(ctx context.Context, ev ConversionEvent) (Result, error) {
	// Real API requires "yyyy-MM-dd HH:mm:ss+HH:MM" (space-separated, explicit
	// offset), not RFC3339 ("T"/"Z"). Reformat if parseable; fall back to the
	// raw value so a stub call never fails on a formatting edge case.
	conversionDateTime := ev.OccurredAt
	if t, err := time.Parse(time.RFC3339, ev.OccurredAt); err == nil {
		conversionDateTime = t.Format("2006-01-02 15:04:05-07:00")
	}
	payload := map[string]any{
		"conversions": []map[string]any{
			{
				"gclid": ev.ClickID,
				// Real conversion_action is a resource name
				// "customers/{customer_id}/conversionActions/{conversion_action_id}";
				// we don't track a Google customer id in this prototype, so the
				// customer segment is a placeholder.
				"conversion_action":    fmt.Sprintf("customers/000-000-0000/conversionActions/%s", ev.CampaignID),
				"conversion_date_time": conversionDateTime,
				"user_identifiers":     []map[string]any{{"third_party_user_id": ev.UserID}},
				"custom_variables":     ev.Properties,
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return simulateOutcome(string(raw)), nil
}
