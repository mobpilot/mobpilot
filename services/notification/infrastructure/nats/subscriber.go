package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	natsjs "github.com/nats-io/nats.go/jetstream"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/notification/application/ports"
)

const (
	consumerName   = "notification-processor"
	filterSubject  = "mobpilot.events.org.group.>"
	streamName     = "MOBPILOT_EVENTS"
)

// Subscriber listens for org group events on NATS JetStream and dispatches
// them to the notification application service.
type Subscriber struct {
	js           natsjs.JetStream
	notification appports.NotificationUseCase
	logger       *slog.Logger
}

func NewSubscriber(js natsjs.JetStream, notification appports.NotificationUseCase, logger *slog.Logger) *Subscriber {
	return &Subscriber{js: js, notification: notification, logger: logger}
}

// Run creates (or resumes) the durable consumer and starts consuming messages.
// Blocks until ctx is cancelled.
func (s *Subscriber) Run(ctx context.Context) {
	consumerCfg := natsjs.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: filterSubject,
		AckPolicy:     natsjs.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		DeliverPolicy: natsjs.DeliverNewPolicy,
		// Exponential backoff between redelivery attempts so a consistently
		// failing handler doesn't immediately exhaust all 5 retries.
		BackOff: []time.Duration{
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
			5 * time.Minute,
		},
	}

	consumer, err := s.js.CreateOrUpdateConsumer(ctx, streamName, consumerCfg)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create NATS consumer", "err", err)
		return
	}

	s.logger.InfoContext(ctx, "notification subscriber started", "filter", filterSubject)

	cc, err := consumer.Consume(func(msg natsjs.Msg) {
		if err := s.handle(ctx, msg); err != nil {
			s.logger.WarnContext(ctx, "event handling failed, will redeliver",
				"subject", msg.Subject(), "err", err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to start consumer", "err", err)
		return
	}
	defer cc.Stop()

	<-ctx.Done()
	s.logger.InfoContext(ctx, "notification subscriber stopped")
}

// envelope mirrors the structure written by the org outbox worker.
type envelope struct {
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

type memberAddedData struct {
	GroupID string `json:"group_id"`
	OrgID   string `json:"org_id"`
	AppID   string `json:"app_id"`
	UserID  string `json:"user_id"`
}

type groupDeletedData struct {
	GroupID string `json:"group_id"`
	OrgID   string `json:"org_id"`
	AppID   string `json:"app_id"`
}

func (s *Subscriber) handle(ctx context.Context, msg natsjs.Msg) error {
	var env envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		// Malformed message — ack to avoid blocking the queue.
		s.logger.WarnContext(ctx, "malformed event envelope, skipping", "subject", msg.Subject())
		return nil
	}

	switch env.EventType {
	case "org.group.member_added":
		return s.handleMemberAdded(ctx, env)
	case "org.group.deleted":
		return s.handleGroupDeleted(ctx, env)
	default:
		// Unknown event type — ack silently.
		return nil
	}
}

func (s *Subscriber) handleMemberAdded(ctx context.Context, env envelope) error {
	var data memberAddedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil // skip malformed
	}

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return nil
	}
	appID, err := uuid.Parse(data.AppID)
	if err != nil {
		return nil
	}
	groupID, _ := uuid.Parse(data.GroupID)
	orgID, _ := uuid.Parse(data.OrgID)

	cmd := appports.SendToUserCommand{
		UserID:    userID,
		AppID:     appID,
		GroupID:   &groupID,
		OrgID:     &orgID,
		EventType: env.EventType,
		Title:     "You joined a group",
		Body:      "Welcome! You've been added to a new group.",
		Data:      map[string]any{"group_id": data.GroupID},
	}

	if _, err := s.notification.SendToUser(ctx, cmd); err != nil {
		return fmt.Errorf("subscriber.handleMemberAdded: %w", err)
	}
	return nil
}

func (s *Subscriber) handleGroupDeleted(ctx context.Context, env envelope) error {
	var data groupDeletedData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil
	}

	groupID, err := uuid.Parse(data.GroupID)
	if err != nil {
		return nil
	}
	appID, err := uuid.Parse(data.AppID)
	if err != nil {
		return nil
	}
	orgID, _ := uuid.Parse(data.OrgID)

	cmd := appports.SendToGroupCommand{
		GroupID:   groupID,
		AppID:     appID,
		OrgID:     orgID,
		EventType: env.EventType,
		Title:     "Group ended",
		Body:      "The group session has ended.",
		Data:      map[string]any{"group_id": data.GroupID},
	}

	if err := s.notification.SendToGroup(ctx, cmd); err != nil {
		return fmt.Errorf("subscriber.handleGroupDeleted: %w", err)
	}
	return nil
}
