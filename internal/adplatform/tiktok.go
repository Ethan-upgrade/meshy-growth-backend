package adplatform

import (
	"context"
	"encoding/json"
)

// TikTokClient stub-implements the TikTok Events API request shape.
// https://business-api.tiktok.com/portal/docs?id=1741601162187777
type TikTokClient struct{}

func (TikTokClient) Name() string { return "tiktok" }

func (TikTokClient) SendConversion(ctx context.Context, ev ConversionEvent) (Result, error) {
	payload := map[string]any{
		"event_source":    "web",
		"event_source_id": ev.CampaignID,
		"data": []map[string]any{
			{
				"event":     ev.EventName,
				"event_id":  ev.EventID,
				"timestamp": ev.OccurredAt,
				"context": map[string]any{
					"user": map[string]any{
						"external_id": ev.UserID,
					},
					"ad": map[string]any{
						"ttclid": ev.ClickID,
					},
				},
				"properties": ev.Properties,
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return simulateOutcome(string(raw)), nil
}
