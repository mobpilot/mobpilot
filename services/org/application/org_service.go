package application

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// OrgService implements appports.OrgUseCase.
type OrgService struct {
	orgs      domainports.OrgRepository
	members   domainports.MemberRepository
	publisher domainports.EventPublisher
	logger    *slog.Logger
}

// Compile-time interface check.
var _ appports.OrgUseCase = (*OrgService)(nil)

func NewOrgService(
	orgs domainports.OrgRepository,
	members domainports.MemberRepository,
	publisher domainports.EventPublisher,
	logger *slog.Logger,
) *OrgService {
	return &OrgService{orgs: orgs, members: members, publisher: publisher, logger: logger}
}

func (s *OrgService) CreateOrg(ctx context.Context, cmd appports.CreateOrgCommand) (*domain.Organization, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "OrgService.CreateOrg")
	defer span.End()
	span.SetAttributes(
		attribute.String("org.slug", cmd.Slug),
		attribute.String("owner.id", cmd.OwnerUserID.String()),
	)

	org, err := domain.NewOrganization(cmd.Name, cmd.Slug, cmd.OwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("OrgService.CreateOrg new org: %w", err)
	}

	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("OrgService.CreateOrg persist: %w", err)
	}

	// Add the owner as a member with role "owner".
	owner := &domain.OrgMember{
		OrgID:    org.ID,
		UserID:   cmd.OwnerUserID,
		Role:     domain.RoleOwner,
		JoinedAt: org.CreatedAt,
	}
	if err := s.members.Add(ctx, owner); err != nil {
		return nil, fmt.Errorf("OrgService.CreateOrg add owner: %w", err)
	}

	if err := s.publisher.Publish(ctx, org.PopEvents()); err != nil {
		s.logger.WarnContext(ctx, "OrgService.CreateOrg: publish events failed", "err", err)
	}
	return org, nil
}

func (s *OrgService) GetOrg(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "OrgService.GetOrg")
	defer span.End()

	org, err := s.orgs.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("OrgService.GetOrg: %w", err)
	}
	return org, nil
}
