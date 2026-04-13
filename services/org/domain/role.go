package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a named set of permissions within an organization.
type Role struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	Name        string
	Permissions []string
	CreatedAt   time.Time
}

// NewRole creates a Role.
func NewRole(orgID uuid.UUID, name string, permissions []string) (*Role, error) {
	if orgID == uuid.Nil {
		return nil, ErrInvalidInput("orgID must not be nil")
	}
	if name == "" {
		return nil, ErrInvalidInput("role name must not be empty")
	}
	return &Role{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Permissions: permissions,
		CreatedAt:   time.Now().UTC(),
	}, nil
}
