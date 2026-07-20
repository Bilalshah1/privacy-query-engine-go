package api

import (
	"database/sql"
	"net/http"

	"github.com/Bilalshah1/privacy-query-engine/internal/audit"
	"github.com/Bilalshah1/privacy-query-engine/internal/budget"
)

// NewServer builds the HTTP handler (router + middleware) for the
// whole application. cmd/server/main.go is responsible for actually
// starting an http.Server with this handler and for graceful
// shutdown.
func NewServer(db *sql.DB, budgetTracker *budget.Tracker, auditLogger *audit.Logger, defaultDelta float64) http.Handler {
	h := &Handler{
		DB:           db,
		Budget:       budgetTracker,
		Audit:        auditLogger,
		DefaultDelta: defaultDelta,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/query", requireUserID(h.HandleQuery))
	mux.HandleFunc("/health", handleHealth)

	return mux
}

// handleHealth is a simple liveness check with no dependencies on
// Redis/Postgres/Kafka being reachable — useful for container
// orchestration health probes.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}