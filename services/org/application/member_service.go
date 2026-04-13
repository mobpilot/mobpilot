package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// MemberService implements appports.MemberUseCase.
type MemberService struct {
	members     domainports.MemberRepository
	invitations domainports.InvitationRepository
	publisher   domainports.EventPublisher
}

// Compile-time interface check.
var _ appports.MemberUseCase = (*MemberService)(nil)

func NewMemberService(
	members domainports.MemberRepository,
	invitations domainports.InvitationRepository,
	publisher domainports.EventPublisher,
) *MemberService {
	return &MemberService{members: members, invitations: invitations, publisher: publisher}
}

func (s *MemberService) InviteMember(ctx context.Context, cmd appports.InviteMemberCommand) (*domain.Invitation, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "MemberService.InviteMember")
	defer span.End()
	span.SetAttributes(
		attribute.String("org.id", cmd.OrgID.String()),
		attribute.String("invite.email", cmd.Email),
	)

	if cmd.Email == "" {
		return nil, domain.ErrInvalidInput("email must not be empty")
	}
	if !domain.ValidRole(cmd.Role) {
		return nil, domain.ErrInvalidInput("invalid role")
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("MemberService.InviteMember generate token: %w", err)
	}

	now := time.Now().UTC()
	inv := &domain.Invitation{
		ID:        uuid.New(),
		OrgID:     cmd.OrgID,
		Email:     cmd.Email,
		Role:      cmd.Role,
		Token:     token,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: now,
	}

	if err := s.invitations.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("MemberService.InviteMember persist: %w", err)
	}

	events := []domain.DomainEvent{
		domain.MemberInvitedEvent{
			OrgID:       cmd.OrgID,
			Email:       cmd.Email,
			Role:        cmd.Role,
			OccurredAt_: now,
		},
	}
	if err := s.publisher.Publish(ctx, events); err != nil {
		_ = err
	}
	return inv, nil
}

func (s *MemberService) AcceptInvitation(ctx context.Context, cmd appports.AcceptInvitationCommand) (*domain.OrgMember, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "MemberService.AcceptInvitation")
	defer span.End()

	inv, err := s.invitations.FindByToken(ctx, cmd.Token)
	if err != nil {
		return nil, fmt.Errorf("MemberService.AcceptInvitation find: %w", err)
	}
	if inv.IsAccepted() {
		return nil, domain.ErrAlreadyExists
	}
	if inv.IsExpired() {
		return nil, domain.ErrInviteExpired
	}

	now := time.Now().UTC()
	member := &domain.OrgMember{
		OrgID:     inv.OrgID,
		UserID:    cmd.UserID,
		Role:      inv.Role,
		InvitedBy: nil, // could track this
		JoinedAt:  now,
	}
	if err := s.members.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("MemberService.AcceptInvitation add member: %w", err)
	}
	if err := s.invitations.MarkAccepted(ctx, inv.ID); err != nil {
		return nil, fmt.Errorf("MemberService.AcceptInvitation mark accepted: %w", err)
	}

	events := []domain.DomainEvent{
		domain.MemberJoinedEvent{
			OrgID:       inv.OrgID,
			UserID:      cmd.UserID,
			Role:        inv.Role,
			OccurredAt_: now,
		},
	}
	if err := s.publisher.Publish(ctx, events); err != nil {
		_ = err
	}
	return member, nil
}

func (s *MemberService) RemoveMember(ctx context.Context, cmd appports.RemoveMemberCommand) error {
	ctx, span := otel.Tracer("org").Start(ctx, "MemberService.RemoveMember")
	defer span.End()

	if err := s.members.Remove(ctx, cmd.OrgID, cmd.UserID); err != nil {
		return fmt.Errorf("MemberService.RemoveMember: %w", err)
	}
	return nil
}

func (s *MemberService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "MemberService.ListMembers")
	defer span.End()

	members, err := s.members.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("MemberService.ListMembers: %w", err)
	}
	return members, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
