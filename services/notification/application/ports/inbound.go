package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/mobpilot/mobpilot/services/notification/domain"
)

// SendToUserCommand triggers a notification for a single user.
type SendToUserCommand struct {
	UserID    uuid.UUID
	AppID     uuid.UUID
	OrgID     *uuid.UUID
	GroupID   *uuid.UUID
	EventType string
	Title     string
	Body      string
	Data      map[string]any
}

// SendToGroupCommand fan-outs a notification to all members of a group.
type SendToGroupCommand struct {
	GroupID   uuid.UUID
	AppID     uuid.UUID
	OrgID     uuid.UUID
	EventType string
	Title     string
	Body      string
	Data      map[string]any
}

// MarkReadCommand marks a notification as read by its owner.
type MarkReadCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

// NotificationUseCase defines the inbound application port.
type NotificationUseCase interface {
	SendToUser(ctx context.Context, cmd SendToUserCommand) (*domain.Notification, error)
	SendToGroup(ctx context.Context, cmd SendToGroupCommand) error
	MarkRead(ctx context.Context, cmd MarkReadCommand) error
	ListForUser(ctx context.Context, userID, appID uuid.UUID, limit, offset int) ([]*domain.Notification, error)
}

// UpsertPreferenceCommand sets a notification preference for a user.
type UpsertPreferenceCommand struct {
	UserID    uuid.UUID
	AppID     uuid.UUID
	Channel   domain.Channel
	EventType string
	Enabled   bool
}

// PreferenceUseCase manages per-user channel preferences.
type PreferenceUseCase interface {
	GetPreferences(ctx context.Context, userID, appID uuid.UUID) ([]*domain.NotificationPreference, error)
	UpsertPreference(ctx context.Context, cmd UpsertPreferenceCommand) error
}
