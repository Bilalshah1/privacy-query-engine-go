// Package config loads runtime configuration from environment
// variables, with sane defaults for local development.
package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port string

	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	KafkaBrokers []string
	KafkaTopic   string

	// DailyEpsilonBudget is the total privacy budget (epsilon) each
	// user is allotted per calendar day, per the README's design.
	DailyEpsilonBudget float64

	// DefaultDelta is the failure-probability parameter used for the
	// Gaussian mechanism when a request doesn't specify one.
	DefaultDelta float64
}

func Load() Config {
	return Config{
		Port: envOr("PORT", "3000"),

		PostgresHost:     envOr("POSTGRES_HOST", "localhost"),
		PostgresPort:     envOrInt("POSTGRES_PORT", 5432),
		PostgresUser:     envOr("POSTGRES_USER", "postgres"),
		PostgresPassword: envOr("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       envOr("POSTGRES_DB", "privacy_query_engine"),
		PostgresSSLMode:  envOr("POSTGRES_SSLMODE", "disable"),

		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword: envOr("REDIS_PASSWORD", ""),
		RedisDB:       envOrInt("REDIS_DB", 0),

		KafkaBrokers: strings.Split(envOr("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   envOr("KAFKA_TOPIC", "dp-audit-log"),

		DailyEpsilonBudget: envOrFloat("DAILY_EPSILON_BUDGET", 10.0),
		DefaultDelta:       envOrFloat("DEFAULT_DELTA", 0.00001),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}