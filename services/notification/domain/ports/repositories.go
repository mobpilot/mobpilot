package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/mobpilot/mobpilot/services/notification/domain"
)

// NotificationRepository persists notification records.
type NotificationRepository interface {
	Save(ctx context.Context, n *domain.Notification) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	ListForUser(ctx context.Context, userID, appID uuid.UUID, limit, offset int) ([]*domain.Notification, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus) error
	MarkRead(ctx context.Context, id uuid.UUID) error
}

// PreferenceRepository persists per-user notification preferences.
type PreferenceRepository interface {
	Upsert(ctx context.Context, p *domain.NotificationPreference) error
	FindByUser(ctx context.Context, userID, appID uuid.UUID) ([]*domain.NotificationPreference, error)
	IsEnabled(ctx context.Context, userID, appID uuid.UUID, channel domain.Channel, eventType string) (bool, error)
}
