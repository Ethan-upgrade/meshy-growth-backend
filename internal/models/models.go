package models

import "time"

type Event struct {
	ID         int64          `json:"id"`
	EventName  string         `json:"event_name"`
	UserID     string         `json:"user_id"`
	DeviceID   string         `json:"device_id"`
	ClickID    string         `json:"click_id"`
	AdPlatform string         `json:"ad_platform"`
	CampaignID int64          `json:"campaign_id"`
	Properties map[string]any `json:"properties"`
	OccurredAt time.Time      `json:"occurred_at"`
	ReceivedAt time.Time      `json:"received_at"`
}

type Conversion struct {
	ID              int64     `json:"id"`
	EventID         int64     `json:"event_id"`
	AdPlatform      string    `json:"ad_platform"`
	CampaignID      int64     `json:"campaign_id"`
	ClickID         string    `json:"click_id"`
	Status          string    `json:"status"`
	AttemptCount    int       `json:"attempt_count"`
	LastError       string    `json:"last_error"`
	CallbackPayload string    `json:"callback_payload"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Campaign struct {
	ID                    int64  `json:"id"`
	Name                  string `json:"name"`
	AdPlatform            string `json:"ad_platform"`
	ExternalCampaignID    string `json:"external_campaign_id"`
	AttributionWindowDays int    `json:"attribution_window_days"`
	IsActive              bool   `json:"is_active"`
}

type Experiment struct {
	ID                int64     `json:"id"`
	Key               string    `json:"key"`
	Name              string    `json:"name"`
	Status            string    `json:"status"`
	TrafficAllocation int       `json:"traffic_allocation"`
	CreatedAt         time.Time `json:"created_at"`
}

type ExperimentVariant struct {
	ID           int64  `json:"id"`
	ExperimentID int64  `json:"experiment_id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Weight       int    `json:"weight"`
}

type ExperimentAssignment struct {
	ID           int64     `json:"id"`
	ExperimentID int64     `json:"experiment_id"`
	VariantID    int64     `json:"variant_id"`
	UserID       string    `json:"user_id"`
	AssignedAt   time.Time `json:"assigned_at"`
}

type DailyFunnelStat struct {
	Date       string `json:"date"`
	CampaignID int64  `json:"campaign_id"`
	EventName  string `json:"event_name"`
	Count      int64  `json:"count"`
}
