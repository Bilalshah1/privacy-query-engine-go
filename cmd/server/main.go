// Command server starts the privacy query engine's HTTP API,
// wiring together Postgres (true data), Redis (daily epsilon
// budget), and Kafka (audit log) as described in the project README.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Bilalshah1/privacy-query-engine/internal/api"
	"github.com/Bilalshah1/privacy-query-engine/internal/audit"
	"github.com/Bilalshah1/privacy-query-engine/internal/budget"
	"github.com/Bilalshah1/privacy-query-engine/internal/config"
	"github.com/Bilalshah1/privacy-query-engine/internal/db"
)

func main() {
	cfg := config.Load()

	// --- Postgres: where the true, un-noised data lives ---
	pg, err := db.Connect(db.Config{
		Host:     cfg.PostgresHost,
		Port:     cfg.PostgresPort,
		User:     cfg.PostgresUser,
		Password: cfg.PostgresPassword,
		DBName:   cfg.PostgresDB,
		SSLMode:  cfg.PostgresSSLMode,
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pg.Close()
	log.Println("connected to postgres")

	// --- Redis: daily epsilon budget tracking ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("connected to redis")

	budgetTracker := budget.NewTracker(redisClient, cfg.DailyEpsilonBudget)

	// --- Kafka: append-only audit log ---
	auditLogger := audit.NewLogger(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() {
		if err := auditLogger.Close(); err != nil {
			log.Printf("error closing audit logger: %v", err)
		}
	}()
	log.Printf("audit logger configured for topic %q on brokers %v", cfg.KafkaTopic, cfg.KafkaBrokers)

	// --- HTTP server ---
	handler := api.NewServer(pg, budgetTracker, auditLogger, cfg.DefaultDelta)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received, draining in-flight requests...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("shutdown complete")
	}
}