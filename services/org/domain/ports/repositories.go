package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mobpilot/mobpilot/services/org/domain"
)

// OrgRepository is the outbound persistence port for Organization aggregates.
type OrgRepository interface {
	// Create persists a new organization.
	Create(ctx context.Context, org *domain.Organization) error
	// FindByID returns an organization by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	// FindBySlug returns an organization by its slug.
	FindBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	// Update persists changes to an existing organization.
	Update(ctx context.Context, org *domain.Organization) error
}

// MemberRepository is the outbound persistence port for OrgMember.
type MemberRepository interface {
	// Add inserts a member into an organization.
	Add(ctx context.Context, member *domain.OrgMember) error
	// Remove deletes a member from an organization.
	Remove(ctx context.Context, orgID, userID uuid.UUID) error
	// FindByOrgAndUser returns a member by org and user IDs.
	FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error)
	// ListByOrg returns all members of an organization.
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error)
}

// InvitationRepository is the outbound persistence port for Invitation.
type InvitationRepository interface {
	// Create persists a new invitation.
	Create(ctx context.Context, inv *domain.Invitation) error
	// FindByToken returns an invitation by its token.
	FindByToken(ctx context.Context, token string) (*domain.Invitation, error)
	// MarkAccepted marks an invitation as accepted.
	MarkAccepted(ctx context.Context, id uuid.UUID) error
}

// GroupRepository is the outbound persistence port for Group.
type GroupRepository interface {
	// Create persists a new group.
	Create(ctx context.Context, group *domain.Group) error
	// FindByID returns a group by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Group, error)
	// ListByOrg returns all groups in an organization.
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Group, error)
	// ListExpired returns all ephemeral groups whose expires_at has passed.
	ListExpired(ctx context.Context, now time.Time) ([]*domain.Group, error)
	// Delete removes a group and its members.
	Delete(ctx context.Context, id uuid.UUID) error
	// AddMember adds a user to a group.
	AddMember(ctx context.Context, groupID, userID, appID uuid.UUID) error
	// RemoveMember removes a user from a group.
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	// ListMembers returns all members of a group.
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]*domain.GroupMember, error)
}

// RoleRepository is the outbound persistence port for Role.
type RoleRepository interface {
	// Create persists a new role.
	Create(ctx context.Context, role *domain.Role) error
	// FindByID returns a role by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	// ListByOrg returns all roles in an organization.
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error)
	// Delete removes a role.
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventPublisher publishes domain events (e.g. to NATS JetStream).
type EventPublisher interface {
	Publish(ctx context.Context, events []domain.DomainEvent) error
}

// AuthzPort is the outbound port for authorization (Ory Keto ReBAC).
// Objects must be in "Namespace:objectID" format (e.g. "Group:some-uuid").
type AuthzPort interface {
	// Check returns true if the subject has the given relation on the object.
	Check(ctx context.Context, subject, relation, object string) (bool, error)
	// WriteRelation creates a relationship tuple.
	WriteRelation(ctx context.Context, subject, relation, object string) error
	// DeleteRelation removes a relationship tuple.
	DeleteRelation(ctx context.Context, subject, relation, object string) error
}
