package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/mobpilot/mobpilot/services/identity/domain"
)

// UserUseCase is the inbound port for user profile operations.
// HTTP handlers and Connect-RPC handlers depend on this interface, not on
// the concrete UserService.
type UserUseCase interface {
	// GetMe returns the current user's profile, creating it if it does not exist.
	GetMe(ctx context.Context, cmd GetMeCommand) (*domain.User, error)
	// UpdateMe applies a partial update to the current user's profile.
	UpdateMe(ctx context.Context, cmd UpdateMeCommand) (*domain.User, error)
	// GetUser returns a public user profile.
	GetUser(ctx context.Context, userID, appID uuid.UUID) (*domain.User, error)
}

// DeviceTokenUseCase is the inbound port for push notification token management.
type DeviceTokenUseCase interface {
	// RegisterDevice registers or refreshes a push token for the caller's device.
	RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (*domain.DeviceToken, error)
	// UnregisterDevice removes a device token.
	UnregisterDevice(ctx context.Context, cmd UnregisterDeviceCommand) error
	// FindDevicesByUser returns all device tokens for a user (internal/service-to-service use).
	FindDevicesByUser(ctx context.Context, userID, appID uuid.UUID) ([]*domain.DeviceToken, error)
}

// ArtifactUseCase is the inbound port for artifact (key/value) storage.
type ArtifactUseCase interface {
	// Put creates or replaces an artifact.
	Put(ctx context.Context, cmd PutArtifactCommand) (*domain.Artifact, error)
	// Get retrieves an artifact by key.
	Get(ctx context.Context, cmd GetArtifactCommand) (*domain.Artifact, error)
	// List returns all artifacts for the caller in the current app.
	List(ctx context.Context, userID, appID uuid.UUID) ([]*domain.Artifact, error)
	// Delete removes an artifact.
	Delete(ctx context.Context, cmd DeleteArtifactCommand) error
}

// ─── Commands ────────────────────────────────────────────────────────────────

type GetMeCommand struct {
	UserID uuid.UUID // from JWT sub
	AppID  uuid.UUID // from JWT mobpilot_app_id
}

type UpdateMeCommand struct {
	UserID uuid.UUID
	AppID  uuid.UUID
	Patch  domain.ProfilePatch
}

type RegisterDeviceCommand struct {
	UserID    uuid.UUID
	AppID     uuid.UUID
	Platform  domain.Platform
	TokenType domain.TokenType
	Token     string
	DeviceID  string
}

type UnregisterDeviceCommand struct {
	AppID    uuid.UUID
	DeviceID string
}

type PutArtifactCommand struct {
	UserID uuid.UUID
	AppID  uuid.UUID
	Key    string
	Value  any
}

type GetArtifactCommand struct {
	UserID uuid.UUID
	AppID  uuid.UUID
	Key    string
}

type DeleteArtifactCommand struct {
	UserID uuid.UUID
	AppID  uuid.UUID
	Key    string
}
