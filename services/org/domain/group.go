package domain

import (
	"time"

	"github.com/google/uuid"
)

// Group represents a named group within an organization.
// Groups may be permanent (long-lived communities) or ephemeral (short-lived
// sessions such as card games). Ephemeral groups require an ExpiresAt TTL and
// are automatically deleted once that time passes.
type Group struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	Name        string
	Description string
	Ephemeral   bool
	ExpiresAt   *time.Time // nil for permanent groups
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// NewGroup creates a Group with validation.
func NewGroup(orgID, appID, createdBy uuid.UUID, name, description string, ephemeral bool, expiresAt *time.Time) (*Group, error) {
	if orgID == uuid.Nil {
		return nil, ErrInvalidInput("orgID must not be nil")
	}
	if appID == uuid.Nil {
		return nil, ErrInvalidInput("appID must not be nil")
	}
	if createdBy == uuid.Nil {
		return nil, ErrInvalidInput("createdBy must not be nil")
	}
	if name == "" {
		return nil, ErrInvalidInput("group name must not be empty")
	}
	if ephemeral && expiresAt == nil {
		return nil, ErrInvalidInput("ephemeral groups must have an expires_at")
	}
	return &Group{
		ID:          uuid.New(),
		OrgID:       orgID,
		AppID:       appID,
		Name:        name,
		Description: description,
		Ephemeral:   ephemeral,
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// GroupMember represents a user's membership in a group.
type GroupMember struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	AppID   uuid.UUID
}
