package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Bilalshah1/privacy-query-engine/internal/audit"
	"github.com/Bilalshah1/privacy-query-engine/internal/budget"
	"github.com/Bilalshah1/privacy-query-engine/internal/dp"
	"github.com/Bilalshah1/privacy-query-engine/internal/query"
)

// Handler bundles everything a request needs: the database to run
// true queries against, the budget tracker to enforce daily epsilon
// limits, the audit logger to record every attempt, and the default
// delta to use for the Gaussian mechanism when a caller doesn't
// specify one.
type Handler struct {
	DB           *sql.DB
	Budget       *budget.Tracker
	Audit        *audit.Logger
	DefaultDelta float64
}

// writeJSON writes a JSON body with the given status code.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError writes a JSON error response in the {"ok": false, "error": "..."} shape.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{OK: false, Error: message})
}

// HandleQuery implements POST /query — the entire pipeline described
// in the README: check-and-deduct the caller's privacy budget, run
// the true (whitelisted) query, add calibrated noise, log the
// outcome, and return only the noisy result.
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	userID := userIDFromContext(r.Context())
	ctx := r.Context()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Mechanism != "laplace" && req.Mechanism != "gaussian" {
		writeError(w, http.StatusBadRequest, `mechanism must be "laplace" or "gaussian"`)
		return
	}
	if req.Epsilon <= 0 {
		writeError(w, http.StatusBadRequest, "epsilon must be > 0")
		return
	}

	// Step 1: the query name must be one of the whitelisted templates.
	// No arbitrary SQL — see internal/query/templates.go for why.
	tmpl, err := query.Lookup(req.Query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	delta := req.Delta
	if delta == 0 {
		delta = h.DefaultDelta
	}

	// Step 2: atomically check-and-deduct the caller's daily epsilon
	// budget. This must happen BEFORE running the query — if the
	// budget check ran after, a burst of concurrent requests could
	// all pass the check before any of them deduct, overspending the
	// budget (the exact race condition the Lua script exists to
	// prevent, so we don't want to give it a chance to matter here).
	budgetResult, err := h.Budget.CheckAndDeduct(ctx, userID, req.Epsilon)
	if err != nil {
		h.logAudit(ctx, userID, tmpl, req, delta, 0, 0, audit.OutcomeError, err.Error())
		writeError(w, http.StatusInternalServerError, "budget check failed: "+err.Error())
		return
	}
	if !budgetResult.Allowed {
		h.logAudit(ctx, userID, tmpl, req, delta, budgetResult.SpentSoFar, budgetResult.RemainingAfter, audit.OutcomeBudgetExceeded, "")
		writeError(w, http.StatusTooManyRequests, fmt.Sprintf(
			"insufficient privacy budget: requested %.4f, %.4f remaining today",
			req.Epsilon, budgetResult.RemainingAfter,
		))
		return
	}

	// Step 3: run the true (un-noised) query. This value never
	// reaches the caller directly — only after noise is added below.
	rows, err := query.Execute(ctx, h.DB, tmpl.Name)
	if err != nil {
		h.logAudit(ctx, userID, tmpl, req, delta, budgetResult.SpentSoFar, budgetResult.RemainingAfter, audit.OutcomeError, err.Error())
		writeError(w, http.StatusInternalServerError, "query execution failed: "+err.Error())
		return
	}

	// Step 4: add calibrated noise, server-side, before anything is
	// serialized into the response.
	noisyRows := make([]ResultRow, 0, len(rows))
	for _, row := range rows {
		var noisyValue float64
		switch req.Mechanism {
		case "laplace":
			noisyValue = dp.AddLaplaceNoise(row.Value, tmpl.Sensitivity, req.Epsilon)
		case "gaussian":
			noisyValue = dp.AddGaussianNoise(row.Value, tmpl.Sensitivity, req.Epsilon, delta)
		}
		noisyRows = append(noisyRows, ResultRow{Label: row.Label, Value: noisyValue})
	}

	// Step 5: log success to the audit trail (fire-and-forget; does
	// not block this response).
	h.logAudit(ctx, userID, tmpl, req, delta, budgetResult.SpentSoFar, budgetResult.RemainingAfter, audit.OutcomeSuccess, "")

	guarantee := fmt.Sprintf("ε-DP with ε=%.4f", req.Epsilon)
	if req.Mechanism == "gaussian" {
		guarantee = fmt.Sprintf("(ε, δ)-DP with ε=%.4f, δ=%.8f", req.Epsilon, delta)
	}

	writeJSON(w, http.StatusOK, QueryResponse{
		OK:   true,
		Data: noisyRows,
		Meta: QueryResponseMeta{
			Mechanism:       req.Mechanism,
			EpsilonUsed:     req.Epsilon,
			DeltaUsed:       delta,
			BudgetRemaining: budgetResult.RemainingAfter,
			DPGuarantee:     guarantee,
			RowCount:        len(noisyRows),
		},
	})
}

// logAudit is a small helper to keep the Record construction out of
// the main handler flow above.
func (h *Handler) logAudit(
	ctx context.Context,
	userID string,
	tmpl query.Template,
	req QueryRequest,
	delta float64,
	budgetBefore, budgetAfter float64,
	outcome audit.Outcome,
	errMsg string,
) {
	h.Audit.Log(ctx, audit.Record{
		UserID:        userID,
		QueryTemplate: tmpl.Name,
		QueryHash:     audit.HashQuery(tmpl.SQL),
		Mechanism:     req.Mechanism,
		Epsilon:       req.Epsilon,
		Delta:         delta,
		BudgetBefore:  budgetBefore,
		BudgetAfter:   budgetAfter,
		Outcome:       outcome,
		ErrorMessage:  errMsg,
	})
}