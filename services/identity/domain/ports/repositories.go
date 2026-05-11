package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/mobpilot/mobpilot/services/identity/domain"
)

// UserRepository is the outbound persistence port for User aggregates.
type UserRepository interface {
	// Upsert creates or replaces the user profile (idempotent on kratosID+appID).
	Upsert(ctx context.Context, user *domain.User) error
	// FindByID returns the user profile for the given Kratos identity ID in an app.
	FindByID(ctx context.Context, id, appID uuid.UUID) (*domain.User, error)
	// Update persists changes to an existing user profile.
	Update(ctx context.Context, user *domain.User) error
}

// DeviceTokenRepository is the outbound persistence port for DeviceToken.
type DeviceTokenRepository interface {
	// Upsert creates or updates the token for (appID, deviceID).
	Upsert(ctx context.Context, token *domain.DeviceToken) error
	// Delete removes a device token by device ID within an app.
	Delete(ctx context.Context, appID uuid.UUID, deviceID string) error
	// FindByUser returns all active tokens for a user in an app.
	FindByUser(ctx context.Context, userID, appID uuid.UUID) ([]*domain.DeviceToken, error)
}

// ArtifactRepository is the outbound persistence port for Artifact.
type ArtifactRepository interface {
	// Upsert creates or replaces an artifact (idempotent on userID+appID+key).
	Upsert(ctx context.Context, artifact *domain.Artifact) error
	// FindByKey retrieves an artifact by key for a user in an app.
	FindByKey(ctx context.Context, userID, appID uuid.UUID, key string) (*domain.Artifact, error)
	// List returns all artifacts for a user in an app.
	List(ctx context.Context, userID, appID uuid.UUID) ([]*domain.Artifact, error)
	// Delete removes an artifact by key.
	Delete(ctx context.Context, userID, appID uuid.UUID, key string) error
}

// EventPublisher publishes domain events (e.g. to NATS JetStream).
// Implementation lives in infrastructure/nats.
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.Event) error
}
