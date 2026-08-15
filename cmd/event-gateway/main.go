package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
	"meshy-growth-backend/internal/experiment"
	"meshy-growth-backend/internal/httpserver"
	"meshy-growth-backend/internal/kafka"
	"meshy-growth-backend/internal/models"
)

type incomingEvent struct {
	EventName  string         `json:"event_name" binding:"required"`
	UserID     string         `json:"user_id" binding:"required"`
	DeviceID   string         `json:"device_id"`
	ClickID    string         `json:"click_id"`
	AdPlatform string         `json:"ad_platform"`
	CampaignID int64          `json:"campaign_id"`
	Properties map[string]any `json:"properties"`
	OccurredAt *time.Time     `json:"occurred_at"`
}

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	producer := kafka.NewProducer(cfg.KafkaBrokers, "events.raw")
	defer producer.Close()

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	r.POST("/v1/events", func(c *gin.Context) {
		var in incomingEvent
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		occurredAt := time.Now().UTC()
		if in.OccurredAt != nil {
			occurredAt = *in.OccurredAt
		}
		ev := models.Event{
			EventName:  in.EventName,
			UserID:     in.UserID,
			DeviceID:   in.DeviceID,
			ClickID:    in.ClickID,
			AdPlatform: in.AdPlatform,
			CampaignID: in.CampaignID,
			Properties: in.Properties,
			OccurredAt: occurredAt,
			ReceivedAt: time.Now().UTC(),
		}
		if err := publishEvent(c.Request.Context(), producer, ev); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
	})

	r.GET("/v1/experiments/:key/assign", func(c *gin.Context) {
		expKey := c.Param("key")
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		variantKey, err := assignVariant(c.Request.Context(), conn, expKey, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if variantKey == "" {
			// User falls outside the experiment's traffic_allocation (holdout).
			c.JSON(http.StatusOK, gin.H{"experiment_key": expKey, "variant": nil, "in_experiment": false})
			return
		}
		ev := models.Event{
			EventName:  "experiment_exposure",
			UserID:     userID,
			Properties: map[string]any{"experiment_key": expKey, "variant": variantKey},
			OccurredAt: time.Now().UTC(),
			ReceivedAt: time.Now().UTC(),
		}
		_ = publishEvent(c.Request.Context(), producer, ev)
		c.JSON(http.StatusOK, gin.H{"experiment_key": expKey, "variant": variantKey, "in_experiment": true})
	})

	log.Printf("event-gateway listening on %s", cfg.HTTPAddr)
	if err := httpserver.Run(cfg.HTTPAddr, r); err != nil {
		log.Fatal(err)
	}
}

func publishEvent(ctx context.Context, p *kafka.Producer, ev models.Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return p.Publish(ctx, ev.UserID, payload)
}

func assignVariant(ctx context.Context, conn *sql.DB, expKey, userID string) (string, error) {
	var experimentID int64
	var status string
	var trafficAllocation int
	if err := conn.QueryRowContext(ctx,
		"SELECT id, status, traffic_allocation FROM experiments WHERE exp_key = ?", expKey,
	).Scan(&experimentID, &status, &trafficAllocation); err != nil {
		return "", err
	}

	if int(experiment.Bucket(expKey+":holdout", userID)%100) >= trafficAllocation {
		return "", nil
	}

	rows, err := conn.QueryContext(ctx,
		"SELECT id, variant_key, weight FROM experiment_variants WHERE experiment_id = ?", experimentID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var variants []experiment.Variant
	variantIDs := map[string]int64{}
	for rows.Next() {
		var id int64
		var key string
		var weight int
		if err := rows.Scan(&id, &key, &weight); err != nil {
			return "", err
		}
		variants = append(variants, experiment.Variant{Key: key, Weight: weight})
		variantIDs[key] = id
	}
	if len(variants) == 0 {
		return "", sql.ErrNoRows
	}

	variantKey := experiment.PickVariant(expKey, userID, variants)

	_, err = conn.ExecContext(ctx,
		`INSERT INTO experiment_assignments (experiment_id, variant_id, user_id)
		 VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE variant_id = variant_id`,
		experimentID, variantIDs[variantKey], userID)
	if err != nil {
		return "", err
	}

	return variantKey, nil
}
