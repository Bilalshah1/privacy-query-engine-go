// Package db sets up the Postgres connection used to run the
// whitelisted query templates. This is the only place the true,
// un-noised data lives outside of an active request's memory.
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config holds Postgres connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Connect opens a Postgres connection pool and verifies it with a
// ping. It sets conservative pool limits suitable for an analytics
// query service that mostly runs short, indexed aggregate queries.
func Connect(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db: failed to ping database: %w", err)
	}

	return db, nil
}