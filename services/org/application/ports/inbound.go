package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mobpilot/mobpilot/services/org/domain"
)

// OrgUseCase is the inbound port for organization operations.
type OrgUseCase interface {
	// CreateOrg creates a new organization.
	CreateOrg(ctx context.Context, cmd CreateOrgCommand) (*domain.Organization, error)
	// GetOrg returns an organization by ID.
	GetOrg(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
}

// MemberUseCase is the inbound port for membership operations.
type MemberUseCase interface {
	// InviteMember creates an invitation to join an organization.
	InviteMember(ctx context.Context, cmd InviteMemberCommand) (*domain.Invitation, error)
	// AcceptInvitation resolves an invitation token and adds the user as a member.
	AcceptInvitation(ctx context.Context, cmd AcceptInvitationCommand) (*domain.OrgMember, error)
	// RemoveMember removes a member from an organization.
	RemoveMember(ctx context.Context, cmd RemoveMemberCommand) error
	// ListMembers returns all members of an organization.
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error)
}

// GroupUseCase is the inbound port for group operations.
type GroupUseCase interface {
	// CreateGroup creates a new group in an organization.
	CreateGroup(ctx context.Context, cmd CreateGroupCommand) (*domain.Group, error)
	// AddGroupMember adds a user to a group.
	AddGroupMember(ctx context.Context, cmd AddGroupMemberCommand) error
	// RemoveGroupMember removes a user from a group.
	RemoveGroupMember(ctx context.Context, cmd RemoveGroupMemberCommand) error
	// ListGroups returns all groups in an organization.
	ListGroups(ctx context.Context, orgID uuid.UUID) ([]*domain.Group, error)
	// ListGroupMembers returns all members of a group.
	ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*domain.GroupMember, error)
	// IssueSubscriptionToken returns a signed Centrifugo subscription JWT for
	// an ephemeral group channel. Only valid when the user is already a member.
	IssueSubscriptionToken(ctx context.Context, cmd IssueSubscriptionTokenCommand) (string, error)
}

// RoleUseCase is the inbound port for role operations.
type RoleUseCase interface {
	// CreateRole creates a new role in an organization.
	CreateRole(ctx context.Context, cmd CreateRoleCommand) (*domain.Role, error)
	// DeleteRole removes a role from an organization.
	DeleteRole(ctx context.Context, cmd DeleteRoleCommand) error
	// ListRoles returns all roles in an organization.
	ListRoles(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error)
}

// ─── Commands ────────────────────────────────────────────────────────────────

type CreateOrgCommand struct {
	Name        string
	Slug        string
	OwnerUserID uuid.UUID
}

type InviteMemberCommand struct {
	OrgID     uuid.UUID
	Email     string
	Role      domain.MemberRole
	InviterID uuid.UUID
}

type AcceptInvitationCommand struct {
	Token  string
	UserID uuid.UUID
}

type RemoveMemberCommand struct {
	OrgID    uuid.UUID
	UserID   uuid.UUID
	CallerID uuid.UUID
}

type CreateGroupCommand struct {
	OrgID       uuid.UUID
	AppID       uuid.UUID
	Name        string
	Description string
	Ephemeral   bool
	ExpiresAt   *time.Time
	CreatedBy   uuid.UUID
}

type AddGroupMemberCommand struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	AppID   uuid.UUID
}

type RemoveGroupMemberCommand struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
}

// GroupMember mirrors domain.GroupMember for use in use-case responses.
type GroupMember struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	AppID   uuid.UUID
}

type IssueSubscriptionTokenCommand struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
}

type CreateRoleCommand struct {
	OrgID       uuid.UUID
	Name        string
	Permissions []string
}

type DeleteRoleCommand struct {
	OrgID  uuid.UUID
	RoleID uuid.UUID
}
