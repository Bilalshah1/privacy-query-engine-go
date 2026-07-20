// Package budget tracks each user's daily differential-privacy budget
// (epsilon) in Redis, using an atomic Lua script for the
// check-and-deduct step so concurrent requests can't overspend it.
package budget

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed budget.lua
var checkAndDeductScript string

// Tracker enforces a per-user daily epsilon budget backed by Redis.
type Tracker struct {
	client      *redis.Client
	script      *redis.Script
	dailyBudget float64
}

// NewTracker creates a budget tracker. dailyBudget is the total
// epsilon each user is allowed to spend per day (the README's example
// system uses 10.0).
func NewTracker(client *redis.Client, dailyBudget float64) *Tracker {
	return &Tracker{
		client:      client,
		script:      redis.NewScript(checkAndDeductScript),
		dailyBudget: dailyBudget,
	}
}

// Result is the outcome of a budget check-and-deduct attempt.
type Result struct {
	Allowed        bool
	SpentSoFar     float64
	RemainingAfter float64
}

func keyFor(userID string, now time.Time) string {
	return fmt.Sprintf("budget:%s:%s", userID, now.UTC().Format("2006-01-02"))
}

func secondsUntilNextUTCMidnight(now time.Time) int64 {
	now = now.UTC()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return int64(tomorrow.Sub(now).Seconds())
}

// CheckAndDeduct atomically checks whether userID has at least
// epsilon remaining in today's budget, and if so, deducts it.
func (t *Tracker) CheckAndDeduct(ctx context.Context, userID string, epsilon float64) (Result, error) {
	if epsilon <= 0 {
		return Result{}, fmt.Errorf("budget: epsilon must be > 0, got %f", epsilon)
	}

	now := time.Now()
	key := keyFor(userID, now)
	ttl := secondsUntilNextUTCMidnight(now)

	raw, err := t.script.Run(ctx, t.client, []string{key}, epsilon, t.dailyBudget, ttl).Result()
	if err != nil {
		return Result{}, fmt.Errorf("budget: redis script failed: %w", err)
	}

	values, ok := raw.([]interface{})
	if !ok || len(values) != 3 {
		return Result{}, fmt.Errorf("budget: unexpected script result shape: %#v", raw)
	}

	allowed := toInt64(values[0]) == 1
	spent := toFloat64(values[1])
	remaining := toFloat64(values[2])

	return Result{
		Allowed:        allowed,
		SpentSoFar:     spent,
		RemainingAfter: remaining,
	}, nil
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	case float64:
		return n
	case int64:
		return float64(n)
	default:
		return 0
	}
}