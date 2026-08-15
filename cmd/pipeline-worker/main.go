package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
	kafkapkg "meshy-growth-backend/internal/kafka"
	"meshy-growth-backend/internal/models"
	"meshy-growth-backend/internal/queue"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	// Demo cadence: reconciles the daily_funnel_stats table from raw events
	// every 5 minutes. In production this would run once a day (e.g. "0 1 * * *").
	scheduler := queue.NewScheduler(cfg.RedisAddr)
	if _, err := scheduler.Register("@every 5m", asynq.NewTask(queue.TypeDailyRollup, nil)); err != nil {
		log.Fatalf("register scheduler: %v", err)
	}
	go func() {
		if err := scheduler.Run(); err != nil {
			log.Fatalf("scheduler run: %v", err)
		}
	}()

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeDailyRollup, func(ctx context.Context, t *asynq.Task) error {
		return runDailyRollup(ctx, conn)
	})
	server := queue.NewServer(cfg.RedisAddr, 2)
	go func() {
		if err := server.Run(mux); err != nil {
			log.Fatalf("asynq server: %v", err)
		}
	}()

	consumer := kafkapkg.NewConsumer(cfg.KafkaBrokers, "events.processed", "pipeline-worker")
	defer consumer.Close()

	log.Println("pipeline-worker started, consuming events.processed")
	for {
		if ctx.Err() != nil {
			break
		}
		msg, err := consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("kafka read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		var ev models.Event
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("bad event payload, dropping: %v", err)
			_ = consumer.CommitMessages(context.Background(), msg)
			continue
		}
		if err := incrementFunnelStat(context.Background(), conn, ev); err != nil {
			log.Printf("incrementFunnelStat error, will retry from last committed offset: %v", err)
			continue
		}
		if err := consumer.CommitMessages(context.Background(), msg); err != nil {
			log.Printf("commit error: %v", err)
		}
	}

	log.Println("shutting down gracefully")
	scheduler.Shutdown()
	server.Shutdown()
}

func incrementFunnelStat(ctx context.Context, conn *sql.DB, ev models.Event) error {
	_, err := conn.ExecContext(ctx,
		`INSERT INTO daily_funnel_stats (stat_date, campaign_id, event_name, event_count)
		 VALUES (DATE(?), ?, ?, 1)
		 ON DUPLICATE KEY UPDATE event_count = event_count + 1`,
		ev.OccurredAt, ev.CampaignID, ev.EventName)
	return err
}

func runDailyRollup(ctx context.Context, conn *sql.DB) error {
	log.Println("running daily rollup reconciliation")
	_, err := conn.ExecContext(ctx, `
		INSERT INTO daily_funnel_stats (stat_date, campaign_id, event_name, event_count)
		SELECT DATE(occurred_at), campaign_id, event_name, COUNT(*)
		FROM events
		GROUP BY DATE(occurred_at), campaign_id, event_name
		ON DUPLICATE KEY UPDATE event_count = VALUES(event_count)
	`)
	return err
}
