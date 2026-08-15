package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"meshy-growth-backend/internal/adplatform"
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

	processedProducer := kafkapkg.NewProducer(cfg.KafkaBrokers, "events.processed")
	defer processedProducer.Close()

	qClient := queue.NewClient(cfg.RedisAddr)
	defer qClient.Close()

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeConversionRetry, func(ctx context.Context, t *asynq.Task) error {
		var p queue.ConversionRetryPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		return retryConversion(ctx, conn, p.ConversionID)
	})

	server := queue.NewServer(cfg.RedisAddr, 5)
	go func() {
		if err := server.Run(mux); err != nil {
			log.Fatalf("asynq server: %v", err)
		}
	}()

	consumer := kafkapkg.NewConsumer(cfg.KafkaBrokers, "events.raw", "conversion-worker")
	defer consumer.Close()

	log.Println("conversion-worker started, consuming events.raw")
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
		if err := handleEvent(context.Background(), conn, qClient, processedProducer, ev); err != nil {
			log.Printf("handleEvent error, will retry from last committed offset: %v", err)
			continue
		}
		if err := consumer.CommitMessages(context.Background(), msg); err != nil {
			log.Printf("commit error: %v", err)
		}
	}

	log.Println("shutting down gracefully")
	server.Shutdown()
}

func handleEvent(ctx context.Context, conn *sql.DB, qClient *asynq.Client, processed *kafkapkg.Producer, ev models.Event) error {
	props, _ := json.Marshal(ev.Properties)
	res, err := conn.ExecContext(ctx,
		`INSERT INTO events (event_name, user_id, device_id, click_id, ad_platform, campaign_id, properties, occurred_at, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.EventName, ev.UserID, ev.DeviceID, ev.ClickID, ev.AdPlatform, ev.CampaignID, props, ev.OccurredAt, ev.ReceivedAt)
	if err != nil {
		return err
	}
	eventID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	ev.ID = eventID

	publishProcessed(ctx, processed, ev, eventID)

	if ev.ClickID == "" || ev.CampaignID == 0 || ev.AdPlatform == "" {
		// No attribution info on this event, nothing to convert.
		return nil
	}

	convRes, err := conn.ExecContext(ctx,
		`INSERT INTO conversions (event_id, ad_platform, campaign_id, click_id, status)
		 VALUES (?, ?, ?, ?, 'pending')`,
		eventID, ev.AdPlatform, ev.CampaignID, ev.ClickID)
	if err != nil {
		return err
	}
	conversionID, err := convRes.LastInsertId()
	if err != nil {
		return err
	}

	if err := sendConversion(ctx, conn, conversionID, ev); err != nil {
		log.Printf("conversion %d send failed, scheduling retry: %v", conversionID, err)
		return enqueueRetry(qClient, conversionID)
	}
	return nil
}

func sendConversion(ctx context.Context, conn *sql.DB, conversionID int64, ev models.Event) error {
	client, err := adplatform.ForPlatform(ev.AdPlatform)
	if err != nil {
		return err
	}
	res, sendErr := client.SendConversion(ctx, adplatform.ConversionEvent{
		EventID:    fmt.Sprintf("%d", ev.ID),
		ClickID:    ev.ClickID,
		EventName:  ev.EventName,
		UserID:     ev.UserID,
		OccurredAt: ev.OccurredAt.Format(time.RFC3339),
		CampaignID: fmt.Sprintf("%d", ev.CampaignID),
		Properties: ev.Properties,
	})

	if sendErr != nil || !res.Success {
		errMsg := res.Error
		if sendErr != nil {
			errMsg = sendErr.Error()
		}
		if _, updErr := conn.ExecContext(ctx,
			`UPDATE conversions SET status='failed', attempt_count=attempt_count+1, last_error=?, callback_payload=? WHERE id=?`,
			errMsg, res.RawRequest, conversionID); updErr != nil {
			return updErr
		}
		if sendErr != nil {
			return sendErr
		}
		return errors.New(errMsg)
	}

	_, err = conn.ExecContext(ctx,
		`UPDATE conversions SET status='success', attempt_count=attempt_count+1, callback_payload=? WHERE id=?`,
		res.RawRequest, conversionID)
	return err
}

func retryConversion(ctx context.Context, conn *sql.DB, conversionID int64) error {
	var eventID int64
	var adPlatform, clickID string
	var campaignID int64
	if err := conn.QueryRowContext(ctx,
		`SELECT event_id, ad_platform, campaign_id, click_id FROM conversions WHERE id = ?`, conversionID,
	).Scan(&eventID, &adPlatform, &campaignID, &clickID); err != nil {
		return err
	}

	var propsRaw sql.NullString
	var eventName, userID string
	var occurredAt time.Time
	if err := conn.QueryRowContext(ctx,
		`SELECT event_name, user_id, properties, occurred_at FROM events WHERE id = ?`, eventID,
	).Scan(&eventName, &userID, &propsRaw, &occurredAt); err != nil {
		return err
	}
	var props map[string]any
	if propsRaw.Valid && propsRaw.String != "" {
		_ = json.Unmarshal([]byte(propsRaw.String), &props)
	}

	ev := models.Event{
		ID:         eventID,
		EventName:  eventName,
		UserID:     userID,
		ClickID:    clickID,
		AdPlatform: adPlatform,
		CampaignID: campaignID,
		Properties: props,
		OccurredAt: occurredAt,
	}
	return sendConversion(ctx, conn, conversionID, ev)
}

func publishProcessed(ctx context.Context, p *kafkapkg.Producer, ev models.Event, eventID int64) {
	ev.ID = eventID
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := p.Publish(pctx, ev.UserID, payload); err != nil {
		log.Printf("publish processed event failed: %v", err)
	}
}

func enqueueRetry(qClient *asynq.Client, conversionID int64) error {
	payload, err := json.Marshal(queue.ConversionRetryPayload{ConversionID: conversionID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(queue.TypeConversionRetry, payload)
	_, err = qClient.Enqueue(task,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		asynq.ProcessIn(10*time.Second),
	)
	return err
}
