package domain

import (
	"time"

	"github.com/google/uuid"
)

// MemberRole is the role of a user in an organization.
type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
)

// ValidRole reports whether r is a known role.
func ValidRole(r MemberRole) bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	}
	return false
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Role      MemberRole
	InvitedBy *uuid.UUID
	JoinedAt  time.Time
}

// Invitation represents a pending invitation to join an organization.
type Invitation struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Email      string
	Role       MemberRole
	Token      string
	ExpiresAt  time.Time
	AcceptedAt *time.Time
	CreatedAt  time.Time
}

// IsExpired reports whether the invitation has passed its expiration time.
func (i *Invitation) IsExpired() bool {
	return time.Now().UTC().After(i.ExpiresAt)
}

// IsAccepted reports whether the invitation has already been accepted.
func (i *Invitation) IsAccepted() bool {
	return i.AcceptedAt != nil
}
