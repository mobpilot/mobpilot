package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	natsjs "github.com/nats-io/nats.go/jetstream"

	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

// OutboxWorker polls the outbox table for unpublished events and publishes them
// to NATS JetStream. It bypasses the domain EventPublisher port intentionally —
// the outbox represents events that were already emitted but may not have been
// delivered (e.g. on process restart).
type OutboxWorker struct {
	queries  *sqlcorg.Queries
	js       natsjs.JetStream
	logger   *slog.Logger
	interval time.Duration
}

func NewOutboxWorker(queries *sqlcorg.Queries, js natsjs.JetStream, logger *slog.Logger) *OutboxWorker {
	return &OutboxWorker{
		queries:  queries,
		js:       js,
		logger:   logger,
		interval: 2 * time.Second,
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *OutboxWorker) poll(ctx context.Context) {
	rows, err := w.queries.GetUnpublishedOutboxEvents(ctx, 100)
	if err != nil {
		w.logger.WarnContext(ctx, "outbox poll failed", "err", err)
		return
	}
	for _, row := range rows {
		if err := w.publishRow(ctx, row); err != nil {
			w.logger.WarnContext(ctx, "outbox publish failed",
				"event_id", row.ID,
				"event_type", row.EventType,
				"err", err,
			)
			continue
		}
		if err := w.queries.MarkOutboxEventPublished(ctx, row.ID); err != nil {
			w.logger.WarnContext(ctx, "outbox mark published failed",
				"event_id", row.ID,
				"err", err,
			)
		}
	}
}

func (w *OutboxWorker) publishRow(ctx context.Context, row sqlcorg.GetUnpublishedOutboxEventsRow) error {
	// Re-wrap in the standard envelope so consumers get consistent structure.
	envelope := map[string]any{
		"event_id":    row.ID,
		"event_type":  row.EventType,
		"occurred_at": row.CreatedAt,
		"data":        json.RawMessage(row.Payload),
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	subject := "mobpilot.events." + row.EventType
	if _, err := w.js.Publish(ctx, subject, payload); err != nil {
		return fmt.Errorf("nats publish %s: %w", subject, err)
	}
	return nil
}

// InsertOutboxEvent is a helper for writing an event to the outbox within a
// transaction. Use this from repository methods that need transactional event
// publishing.
func InsertOutboxEvent(ctx context.Context, q *sqlcorg.Queries, aggregateID uuid.UUID, eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("InsertOutboxEvent marshal: %w", err)
	}
	if err := q.InsertOutboxEvent(ctx, sqlcorg.InsertOutboxEventParams{
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payload,
	}); err != nil {
		return fmt.Errorf("InsertOutboxEvent persist: %w", err)
	}
	return nil
}
