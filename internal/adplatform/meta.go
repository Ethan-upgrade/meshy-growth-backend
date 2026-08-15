package adplatform

import (
	"context"
	"encoding/json"
)

// MetaClient stub-implements the Meta Conversions API request shape.
// https://developers.facebook.com/docs/marketing-api/conversions-api
type MetaClient struct{}

func (MetaClient) Name() string { return "meta" }

func (MetaClient) SendConversion(ctx context.Context, ev ConversionEvent) (Result, error) {
	payload := map[string]any{
		"data": []map[string]any{
			{
				"event_name":    ev.EventName,
				"event_id":      ev.EventID,
				"event_time":    ev.OccurredAt,
				"action_source": "website",
				"user_data": map[string]any{
					"external_id": ev.UserID,
					"fbc":         ev.ClickID,
				},
				"custom_data": ev.Properties,
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return simulateOutcome(string(raw)), nil
}
