package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mobpilot/mobpilot/services/identity/domain"
	domainports "github.com/mobpilot/mobpilot/services/identity/domain/ports"
)

// EventPublisher publishes domain events to NATS JetStream.
type EventPublisher struct {
	js jetstream.JetStream
}

var _ domainports.EventPublisher = (*EventPublisher)(nil)

func NewEventPublisher(js jetstream.JetStream) *EventPublisher {
	return &EventPublisher{js: js}
}

func (p *EventPublisher) Publish(ctx context.Context, events []domain.Event) error {
	for _, e := range events {
		payload, err := json.Marshal(map[string]any{
			"event_type":  e.EventType(),
			"occurred_at": e.OccurredAt(),
			"data":        e,
		})
		if err != nil {
			return fmt.Errorf("nats.EventPublisher marshal %s: %w", e.EventType(), err)
		}
		subject := "mobpilot.events." + e.EventType()
		if _, err := p.js.Publish(ctx, subject, payload); err != nil {
			return fmt.Errorf("nats.EventPublisher publish %s: %w", subject, err)
		}
	}
	return nil
}
