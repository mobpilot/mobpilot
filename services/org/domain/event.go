package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event is the marker interface for all events raised by aggregates.
type Event interface {
	EventType() string
	OccurredAt() time.Time
}

// OrgCreatedEvent is raised when a new organization is created.
type OrgCreatedEvent struct {
	OrgID       uuid.UUID
	OwnerUserID uuid.UUID
	Slug        string
	At time.Time
}

func (e OrgCreatedEvent) EventType() string     { return "org.organization.created" }
func (e OrgCreatedEvent) OccurredAt() time.Time { return e.At }

// MemberInvitedEvent is raised when a user is invited to an organization.
type MemberInvitedEvent struct {
	OrgID       uuid.UUID
	Email       string
	Role        MemberRole
	At time.Time
}

func (e MemberInvitedEvent) EventType() string     { return "org.member.invited" }
func (e MemberInvitedEvent) OccurredAt() time.Time { return e.At }

// MemberJoinedEvent is raised when a user accepts an invitation and becomes a member.
type MemberJoinedEvent struct {
	OrgID       uuid.UUID
	UserID      uuid.UUID
	Role        MemberRole
	At time.Time
}

func (e MemberJoinedEvent) EventType() string     { return "org.member.joined" }
func (e MemberJoinedEvent) OccurredAt() time.Time { return e.At }

// GroupCreatedEvent is raised when a new group is created.
type GroupCreatedEvent struct {
	GroupID     uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	CreatedBy   uuid.UUID
	Ephemeral   bool
	ExpiresAt   *time.Time
	At time.Time
}

func (e GroupCreatedEvent) EventType() string     { return "org.group.created" }
func (e GroupCreatedEvent) OccurredAt() time.Time { return e.At }

// MemberAddedToGroupEvent is raised when a user is added to a group.
type MemberAddedToGroupEvent struct {
	GroupID     uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	UserID      uuid.UUID
	At time.Time
}

func (e MemberAddedToGroupEvent) EventType() string     { return "org.group.member_added" }
func (e MemberAddedToGroupEvent) OccurredAt() time.Time { return e.At }

// MemberRemovedFromGroupEvent is raised when a user is removed from a group.
type MemberRemovedFromGroupEvent struct {
	GroupID     uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	UserID      uuid.UUID
	At time.Time
}

func (e MemberRemovedFromGroupEvent) EventType() string     { return "org.group.member_removed" }
func (e MemberRemovedFromGroupEvent) OccurredAt() time.Time { return e.At }

// GroupDeletedEvent is raised when a group is deleted (including ephemeral expiry).
type GroupDeletedEvent struct {
	GroupID     uuid.UUID
	OrgID       uuid.UUID
	AppID       uuid.UUID
	At time.Time
}

func (e GroupDeletedEvent) EventType() string     { return "org.group.deleted" }
func (e GroupDeletedEvent) OccurredAt() time.Time { return e.At }
