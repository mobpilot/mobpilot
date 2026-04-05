package domain

import (
	"time"

	"github.com/google/uuid"
)

// Artifact is an arbitrary key/value entry in a user's personal cloud storage.
// Used for cross-app settings backup and app-specific data sync.
type Artifact struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AppID     uuid.UUID
	Key       string
	Value     any // serialised as JSONB
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewArtifact creates an Artifact.
func NewArtifact(userID, appID uuid.UUID, key string, value any) (*Artifact, error) {
	if key == "" {
		return nil, ErrInvalidInput("key must not be empty")
	}
	now := time.Now().UTC()
	return &Artifact{
		ID:        uuid.New(),
		UserID:    userID,
		AppID:     appID,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
