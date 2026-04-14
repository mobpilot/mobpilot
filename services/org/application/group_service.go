package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// GroupService implements appports.GroupUseCase.
type GroupService struct {
	groups    domainports.GroupRepository
	publisher domainports.EventPublisher
	authz     domainports.AuthzPort
	tokens    TokenIssuer
	logger    *slog.Logger
}

// TokenIssuer issues signed Centrifugo subscription JWTs.
type TokenIssuer interface {
	IssueGroupToken(userID, groupID uuid.UUID, expiresAt time.Time) (string, error)
}

// Compile-time interface check.
var _ appports.GroupUseCase = (*GroupService)(nil)

func NewGroupService(
	groups domainports.GroupRepository,
	publisher domainports.EventPublisher,
	authz domainports.AuthzPort,
	tokens TokenIssuer,
	logger *slog.Logger,
) *GroupService {
	return &GroupService{
		groups:    groups,
		publisher: publisher,
		authz:     authz,
		tokens:    tokens,
		logger:    logger,
	}
}

func (s *GroupService) CreateGroup(ctx context.Context, cmd appports.CreateGroupCommand) (*domain.Group, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.CreateGroup")
	defer span.End()

	group, err := domain.NewGroup(cmd.OrgID, cmd.AppID, cmd.CreatedBy, cmd.Name, cmd.Description, cmd.Ephemeral, cmd.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("GroupService.CreateGroup new group: %w", err)
	}
	if err := s.groups.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("GroupService.CreateGroup persist: %w", err)
	}

	// Write Keto admin relation for permanent groups only.
	if !group.Ephemeral {
		if err := s.authz.WriteRelation(ctx, cmd.CreatedBy.String(), "admin", "Group:"+group.ID.String()); err != nil {
			s.logger.WarnContext(ctx, "GroupService.CreateGroup: keto write admin relation failed",
				"group_id", group.ID, "err", err)
		}
	}

	_ = s.publisher.Publish(ctx, []domain.Event{domain.GroupCreatedEvent{
		GroupID:     group.ID,
		OrgID:       group.OrgID,
		AppID:       group.AppID,
		CreatedBy:   group.CreatedBy,
		Ephemeral:   group.Ephemeral,
		ExpiresAt:   group.ExpiresAt,
		At: time.Now().UTC(),
	}})

	return group, nil
}

func (s *GroupService) AddGroupMember(ctx context.Context, cmd appports.AddGroupMemberCommand) error {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.AddGroupMember")
	defer span.End()

	// Fetch group to know if it is ephemeral.
	group, err := s.groups.FindByID(ctx, cmd.GroupID)
	if err != nil {
		return fmt.Errorf("GroupService.AddGroupMember find group: %w", err)
	}

	if err := s.groups.AddMember(ctx, cmd.GroupID, cmd.UserID, cmd.AppID); err != nil {
		return fmt.Errorf("GroupService.AddGroupMember: %w", err)
	}

	// Write Keto member relation for permanent groups only.
	if !group.Ephemeral {
		if err := s.authz.WriteRelation(ctx, cmd.UserID.String(), "member", "Group:"+cmd.GroupID.String()); err != nil {
			s.logger.WarnContext(ctx, "GroupService.AddGroupMember: keto write member relation failed",
				"group_id", cmd.GroupID, "user_id", cmd.UserID, "err", err)
		}
	}

	_ = s.publisher.Publish(ctx, []domain.Event{domain.MemberAddedToGroupEvent{
		GroupID:     cmd.GroupID,
		OrgID:       group.OrgID,
		AppID:       cmd.AppID,
		UserID:      cmd.UserID,
		At: time.Now().UTC(),
	}})

	return nil
}

func (s *GroupService) RemoveGroupMember(ctx context.Context, cmd appports.RemoveGroupMemberCommand) error {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.RemoveGroupMember")
	defer span.End()

	group, err := s.groups.FindByID(ctx, cmd.GroupID)
	if err != nil {
		return fmt.Errorf("GroupService.RemoveGroupMember find group: %w", err)
	}

	if err := s.groups.RemoveMember(ctx, cmd.GroupID, cmd.UserID); err != nil {
		return fmt.Errorf("GroupService.RemoveGroupMember: %w", err)
	}

	// Delete Keto member relation for permanent groups only.
	if !group.Ephemeral {
		if err := s.authz.DeleteRelation(ctx, cmd.UserID.String(), "member", "Group:"+cmd.GroupID.String()); err != nil {
			s.logger.WarnContext(ctx, "GroupService.RemoveGroupMember: keto delete member relation failed",
				"group_id", cmd.GroupID, "user_id", cmd.UserID, "err", err)
		}
	}

	_ = s.publisher.Publish(ctx, []domain.Event{domain.MemberRemovedFromGroupEvent{
		GroupID:     cmd.GroupID,
		OrgID:       group.OrgID,
		AppID:       group.AppID,
		UserID:      cmd.UserID,
		At: time.Now().UTC(),
	}})

	return nil
}

func (s *GroupService) ListGroups(ctx context.Context, orgID uuid.UUID) ([]*domain.Group, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.ListGroups")
	defer span.End()

	groups, err := s.groups.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("GroupService.ListGroups: %w", err)
	}
	return groups, nil
}

func (s *GroupService) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*domain.GroupMember, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.ListGroupMembers")
	defer span.End()

	members, err := s.groups.ListMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("GroupService.ListGroupMembers: %w", err)
	}
	return members, nil
}

// IssueSubscriptionToken issues a signed Centrifugo subscription JWT for an
// ephemeral group channel. The user must already be a member of the group.
func (s *GroupService) IssueSubscriptionToken(ctx context.Context, cmd appports.IssueSubscriptionTokenCommand) (string, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "GroupService.IssueSubscriptionToken")
	defer span.End()

	group, err := s.groups.FindByID(ctx, cmd.GroupID)
	if err != nil {
		return "", fmt.Errorf("GroupService.IssueSubscriptionToken find group: %w", err)
	}
	if !group.Ephemeral {
		return "", domain.ErrInvalidInput("subscription tokens are only for ephemeral groups")
	}
	if group.ExpiresAt == nil || group.ExpiresAt.Before(time.Now().UTC()) {
		return "", domain.ErrGroupNotFound
	}

	// Verify membership.
	members, err := s.groups.ListMembers(ctx, cmd.GroupID)
	if err != nil {
		return "", fmt.Errorf("GroupService.IssueSubscriptionToken list members: %w", err)
	}
	isMember := false
	for _, m := range members {
		if m.UserID == cmd.UserID {
			isMember = true
			break
		}
	}
	if !isMember {
		return "", domain.ErrNotAuthorized
	}

	token, err := s.tokens.IssueGroupToken(cmd.UserID, cmd.GroupID, *group.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("GroupService.IssueSubscriptionToken sign: %w", err)
	}
	return token, nil
}
