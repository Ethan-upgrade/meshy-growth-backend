package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
	"meshy-growth-backend/internal/httpserver"
)

func main() {
	cfg := config.Load()
	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	defer inspector.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	r.GET("/v1/events/recent", func(c *gin.Context) {
		rows, err := conn.Query(`SELECT id, event_name, user_id, click_id, ad_platform, campaign_id, occurred_at, received_at
			FROM events ORDER BY received_at DESC LIMIT 100`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type row struct {
			ID         int64     `json:"id"`
			EventName  string    `json:"event_name"`
			UserID     string    `json:"user_id"`
			ClickID    string    `json:"click_id"`
			AdPlatform string    `json:"ad_platform"`
			CampaignID int64     `json:"campaign_id"`
			OccurredAt time.Time `json:"occurred_at"`
			ReceivedAt time.Time `json:"received_at"`
		}
		out := []row{}
		for rows.Next() {
			var rw row
			if err := rows.Scan(&rw.ID, &rw.EventName, &rw.UserID, &rw.ClickID, &rw.AdPlatform, &rw.CampaignID, &rw.OccurredAt, &rw.ReceivedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, rw)
		}
		c.JSON(http.StatusOK, out)
	})

	r.GET("/v1/conversions/summary", func(c *gin.Context) {
		rows, err := conn.Query(`SELECT status, COUNT(*) FROM conversions GROUP BY status`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		summary := map[string]int64{}
		for rows.Next() {
			var status string
			var count int64
			if err := rows.Scan(&status, &count); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			summary[status] = count
		}
		c.JSON(http.StatusOK, summary)
	})

	r.GET("/v1/conversions/recent", func(c *gin.Context) {
		rows, err := conn.Query(`SELECT id, ad_platform, campaign_id, click_id, status, attempt_count, last_error, updated_at
			FROM conversions ORDER BY updated_at DESC LIMIT 100`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type row struct {
			ID           int64     `json:"id"`
			AdPlatform   string    `json:"ad_platform"`
			CampaignID   int64     `json:"campaign_id"`
			ClickID      string    `json:"click_id"`
			Status       string    `json:"status"`
			AttemptCount int       `json:"attempt_count"`
			LastError    string    `json:"last_error"`
			UpdatedAt    time.Time `json:"updated_at"`
		}
		out := []row{}
		for rows.Next() {
			var rw row
			var lastError sql.NullString
			if err := rows.Scan(&rw.ID, &rw.AdPlatform, &rw.CampaignID, &rw.ClickID, &rw.Status, &rw.AttemptCount, &lastError, &rw.UpdatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			rw.LastError = lastError.String
			out = append(out, rw)
		}
		c.JSON(http.StatusOK, out)
	})

	r.GET("/v1/queue/status", func(c *gin.Context) {
		info, err := inspector.GetQueueInfo("default")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"pending":   info.Pending,
			"active":    info.Active,
			"scheduled": info.Scheduled,
			"retry":     info.Retry,
			"archived":  info.Archived,
			"completed": info.Completed,
			"processed": info.Processed,
			"failed":    info.Failed,
		})
	})

	r.GET("/v1/campaigns", func(c *gin.Context) {
		rows, err := conn.Query(`SELECT id, name, ad_platform, external_campaign_id, attribution_window_days, is_active FROM campaigns ORDER BY id DESC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type row struct {
			ID                    int64  `json:"id"`
			Name                  string `json:"name"`
			AdPlatform            string `json:"ad_platform"`
			ExternalCampaignID    string `json:"external_campaign_id"`
			AttributionWindowDays int    `json:"attribution_window_days"`
			IsActive              bool   `json:"is_active"`
		}
		out := []row{}
		for rows.Next() {
			var rw row
			if err := rows.Scan(&rw.ID, &rw.Name, &rw.AdPlatform, &rw.ExternalCampaignID, &rw.AttributionWindowDays, &rw.IsActive); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, rw)
		}
		c.JSON(http.StatusOK, out)
	})

	r.POST("/v1/campaigns", func(c *gin.Context) {
		var req struct {
			Name                  string `json:"name" binding:"required"`
			AdPlatform            string `json:"ad_platform" binding:"required"`
			ExternalCampaignID    string `json:"external_campaign_id" binding:"required"`
			AttributionWindowDays int    `json:"attribution_window_days"`
			IsActive              bool   `json:"is_active"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.AttributionWindowDays == 0 {
			req.AttributionWindowDays = 7
		}
		res, err := conn.Exec(
			`INSERT INTO campaigns (name, ad_platform, external_campaign_id, attribution_window_days, is_active) VALUES (?, ?, ?, ?, ?)`,
			req.Name, req.AdPlatform, req.ExternalCampaignID, req.AttributionWindowDays, req.IsActive)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id})
	})

	r.GET("/v1/experiments", proxyGet(httpClient, cfg.ExperimentServiceURL+"/v1/experiments"))
	r.GET("/v1/experiments/:key", func(c *gin.Context) {
		proxyGet(httpClient, cfg.ExperimentServiceURL+"/v1/experiments/"+c.Param("key"))(c)
	})
	r.GET("/v1/experiments/:key/stats", func(c *gin.Context) {
		url := cfg.ExperimentServiceURL + "/v1/experiments/" + c.Param("key") + "/stats"
		if goal := c.Query("goal_event"); goal != "" {
			url += "?goal_event=" + goal
		}
		proxyGet(httpClient, url)(c)
	})

	log.Printf("admin-api listening on %s", cfg.HTTPAddr)
	if err := httpserver.Run(cfg.HTTPAddr, r); err != nil {
		log.Fatal(err)
	}
}

func proxyGet(client *http.Client, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := client.Get(url)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(resp.StatusCode, "application/json", body)
	}
}
