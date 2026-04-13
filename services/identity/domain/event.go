package domain

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the marker interface for all events raised by aggregates.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// UserCreatedEvent is raised when a new user profile is persisted for the first time.
type UserCreatedEvent struct {
	UserID      uuid.UUID
	AppID       uuid.UUID
	EventTimeAt time.Time
}

func (e UserCreatedEvent) EventType() string    { return "identity.user.created" }
func (e UserCreatedEvent) OccurredAt() time.Time { return e.EventTimeAt }
