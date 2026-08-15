package queue

import (
	"github.com/hibiken/asynq"
)

const (
	TypeConversionRetry = "conversion:retry"
	TypeDailyRollup      = "pipeline:daily_rollup"
)

type ConversionRetryPayload struct {
	ConversionID int64 `json:"conversion_id"`
}

func NewClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
}

func NewServer(redisAddr string, concurrency int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{Concurrency: concurrency},
	)
}

func NewScheduler(redisAddr string) *asynq.Scheduler {
	return asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr}, nil)
}
