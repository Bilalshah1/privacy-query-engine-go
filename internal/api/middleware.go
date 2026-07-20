package api

import (
	"context"
	"net/http"
)

// contextKey avoids collisions with other packages' context keys.
type contextKey string

const userIDContextKey contextKey = "userID"

// requireUserID is middleware that extracts the X-User-ID header and
// stores it in the request context. There's no real authentication
// here — matching the original project's design, this header is
// trusted as-is. It exists so the budget tracker and audit log have
// something to key on, not as an access-control mechanism.
func requireUserID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeError(w, http.StatusBadRequest, "missing required X-User-ID header")
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// userIDFromContext retrieves the user ID stored by requireUserID.
// Only ever called after requireUserID has run, so the empty-string
// fallback here should be unreachable in practice.
func userIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDContextKey).(string)
	return v
}