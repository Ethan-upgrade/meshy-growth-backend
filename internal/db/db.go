package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Connect opens a MySQL connection, retrying while the container is still
// starting up (docker-compose does not guarantee readiness, only start order).
func Connect(dsn string) (*sql.DB, error) {
	var conn *sql.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		conn, err = sql.Open("mysql", dsn)
		if err == nil {
			if pingErr := conn.Ping(); pingErr == nil {
				return conn, nil
			} else {
				err = pingErr
				conn.Close()
			}
		}
		log.Printf("mysql not ready (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("mysql connect failed after retries: %w", err)
}

func Migrate(conn *sql.DB, migrationsPath string) error {
	driver, err := mysqlmigrate.WithInstance(conn, &mysqlmigrate.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "mysql", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
