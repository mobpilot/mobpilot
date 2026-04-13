package nats

import (
	"context"
	"fmt"
	"time"

	natsjs "github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName = "MOBPILOT_EVENTS"
	SubjectAll = "mobpilot.events.>"
)

// EnsureStream creates the MOBPILOT_EVENTS stream idempotently. Both the org
// and notification services call this at startup so neither depends on the
// other having run first.
func EnsureStream(ctx context.Context, js natsjs.JetStream) error {
	cfg := natsjs.StreamConfig{
		Name:     StreamName,
		Subjects: []string{SubjectAll},
		Storage:  natsjs.FileStorage,
		MaxAge:   72 * time.Hour,
	}
	_, err := js.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("nats.EnsureStream: %w", err)
	}
	return nil
}
