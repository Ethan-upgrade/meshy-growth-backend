package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr             string
	MySQLDSN             string
	RedisAddr            string
	KafkaBrokers         []string
	MigrationsPath       string
	ExperimentServiceURL string
}

func Load() Config {
	return Config{
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:             getEnv("MYSQL_DSN", "app:app@tcp(mysql:3306)/growth?parseTime=true&multiStatements=true"),
		RedisAddr:            getEnv("REDIS_ADDR", "redis:6379"),
		KafkaBrokers:         strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		MigrationsPath:       getEnv("MIGRATIONS_PATH", "/app/migrations"),
		ExperimentServiceURL: getEnv("EXPERIMENT_SERVICE_URL", "http://experiment-service:8080"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
