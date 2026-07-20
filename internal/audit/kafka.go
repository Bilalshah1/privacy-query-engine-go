// Package audit writes an immutable, append-only record of every
// query attempt (successful or rejected) to a Kafka topic. This is
// the "privacy accountant": not a person, but a durable log that lets
// a privacy officer later compute exactly how much composed privacy
// loss any user has accumulated against any query template.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// Outcome is the result of a query attempt, recorded for audit
// purposes regardless of whether the query succeeded.
type Outcome string

const (
	OutcomeSuccess        Outcome = "success"
	OutcomeBudgetExceeded Outcome = "budget_exceeded"
	OutcomeError          Outcome = "error"
)

// Record is one audit log entry. Note that RawQuery is never stored —
// only its hash — so the audit trail can prove what ran without
// persisting a second copy of potentially sensitive query text or
// schema details.
type Record struct {
	ID            string  `json:"id"`
	Timestamp     string  `json:"timestamp"`
	UserID        string  `json:"user_id"`
	QueryTemplate string  `json:"query_template"`
	QueryHash     string  `json:"query_hash"`
	Mechanism     string  `json:"mechanism"`
	Epsilon       float64 `json:"epsilon"`
	Delta         float64 `json:"delta,omitempty"`
	BudgetBefore  float64 `json:"budget_before"`
	BudgetAfter   float64 `json:"budget_after"`
	Outcome       Outcome `json:"outcome"`
	ErrorMessage  string  `json:"error_message,omitempty"`
}

// Logger writes Records to the "dp-audit-log" Kafka topic.
type Logger struct {
	writer *kafka.Writer
}

// NewLogger creates a fire-and-forget Kafka audit logger. Writes are
// asynchronous (kafka.Writer batches internally) so that emitting an
// audit record never adds meaningful latency to the query response —
// the record is written to the topic in the background after the
// caller already has their answer.
func NewLogger(brokers []string, topic string) *Logger {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		Async:        true, // fire-and-forget: don't block the request path
		BatchTimeout: 50 * time.Millisecond,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				// Async writes can't return an error to the original
				// caller, since the response has already been sent.
				// The best we can do is log it loudly so it shows up
				// in ops monitoring rather than vanishing silently.
				log.Printf("audit: failed to write %d message(s) to kafka: %v", len(messages), err)
			}
		},
	}
	return &Logger{writer: w}
}

// Close flushes and closes the underlying Kafka writer. Call this on
// graceful shutdown.
func (l *Logger) Close() error {
	return l.writer.Close()
}

// HashQuery returns a SHA-256 hex digest of the raw SQL. We store this
// instead of the raw SQL text so the audit trail can later prove
// exactly which query ran (any observer can hash a candidate query
// template and compare) without persisting the schema/query text
// itself in a second, potentially less-guarded location.
func HashQuery(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

// Log writes one audit record. It never blocks on the network; Kafka
// writes happen asynchronously via the writer's internal batching, so
// a slow or momentarily unavailable broker cannot slow down or fail
// the query response.
func (l *Logger) Log(ctx context.Context, r Record) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Timestamp == "" {
		r.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	payload, err := json.Marshal(r)
	if err != nil {
		log.Printf("audit: failed to marshal record: %v", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(r.UserID),
		Value: payload,
		Time:  time.Now(),
	}

	// WriteMessages on an Async writer enqueues the message and
	// returns immediately; delivery/failure is reported later via the
	// Completion callback set in NewLogger, not via this call's error.
	if err := l.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("audit: failed to enqueue message: %v", err)
	}
}