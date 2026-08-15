package adplatform

import (
	"context"
	"fmt"
	"math/rand"
)

// ConversionEvent is the normalized shape passed to every ad platform client,
// independent of that platform's own wire format.
type ConversionEvent struct {
	EventID    string // dedup key for the ad platform's event/pixel matching
	ClickID    string
	EventName  string
	UserID     string
	OccurredAt string // RFC3339
	CampaignID string
	Properties map[string]any
}

type Result struct {
	Success        bool
	ExternalStatus string
	RawRequest     string // the request body we would have sent, kept for observability
	Error          string
}

type Client interface {
	Name() string
	SendConversion(ctx context.Context, ev ConversionEvent) (Result, error)
}

func ForPlatform(platform string) (Client, error) {
	switch platform {
	case "meta":
		return MetaClient{}, nil
	case "google":
		return GoogleClient{}, nil
	case "tiktok":
		return TikTokClient{}, nil
	default:
		return nil, fmt.Errorf("unknown ad platform: %s", platform)
	}
}

// simulateOutcome stands in for the real HTTP call. It occasionally reports a
// transient failure so the retry path (asynq) is exercised end to end. Swap
// each client's SendConversion body for a real HTTP POST + real credentials
// to go live.
func simulateOutcome(raw string) Result {
	if rand.Float64() < 0.15 {
		return Result{Success: false, ExternalStatus: "stubbed_error", RawRequest: raw, Error: "simulated transient error from ad platform"}
	}
	return Result{Success: true, ExternalStatus: "stubbed_ok", RawRequest: raw}
}
