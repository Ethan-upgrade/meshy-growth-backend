package main

import (
	"log"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
)

func main() {
	cfg := config.Load()
	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied successfully")
}
