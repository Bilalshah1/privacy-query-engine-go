package api

// QueryRequest is the JSON body for POST /query, matching the
// original project's contract:
//
//	{"query": "count_users_by_age", "epsilon": 1.0, "mechanism": "laplace"}
type QueryRequest struct {
	Query     string  `json:"query"`
	Epsilon   float64 `json:"epsilon"`
	Mechanism string  `json:"mechanism"` // "laplace" or "gaussian"
	// Delta is only used for the Gaussian mechanism. If omitted, the
	// server's configured default delta is used.
	Delta float64 `json:"delta,omitempty"`
}

// ResultRow is one noisy row returned to the caller.
type ResultRow struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// QueryResponseMeta mirrors the "meta" block from the original
// project's example response.
type QueryResponseMeta struct {
	Mechanism       string  `json:"mechanism"`
	EpsilonUsed     float64 `json:"epsilon_used"`
	DeltaUsed       float64 `json:"delta_used,omitempty"`
	BudgetRemaining float64 `json:"budget_remaining"`
	DPGuarantee     string  `json:"dp_guarantee"`
	RowCount        int     `json:"row_count"`
}

// QueryResponse is the successful JSON response body for POST /query.
type QueryResponse struct {
	OK   bool              `json:"ok"`
	Data []ResultRow       `json:"data"`
	Meta QueryResponseMeta `json:"meta"`
}

// ErrorResponse is returned for any non-2xx response.
type ErrorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}