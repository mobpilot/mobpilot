package application

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// RoleService implements appports.RoleUseCase.
type RoleService struct {
	roles domainports.RoleRepository
}

// Compile-time interface check.
var _ appports.RoleUseCase = (*RoleService)(nil)

func NewRoleService(roles domainports.RoleRepository) *RoleService {
	return &RoleService{roles: roles}
}

func (s *RoleService) CreateRole(ctx context.Context, cmd appports.CreateRoleCommand) (*domain.Role, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "RoleService.CreateRole")
	defer span.End()

	role, err := domain.NewRole(cmd.OrgID, cmd.Name, cmd.Permissions)
	if err != nil {
		return nil, fmt.Errorf("RoleService.CreateRole new role: %w", err)
	}
	if err := s.roles.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("RoleService.CreateRole persist: %w", err)
	}
	return role, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, cmd appports.DeleteRoleCommand) error {
	ctx, span := otel.Tracer("org").Start(ctx, "RoleService.DeleteRole")
	defer span.End()

	if err := s.roles.Delete(ctx, cmd.RoleID); err != nil {
		return fmt.Errorf("RoleService.DeleteRole: %w", err)
	}
	return nil
}

func (s *RoleService) ListRoles(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	ctx, span := otel.Tracer("org").Start(ctx, "RoleService.ListRoles")
	defer span.End()

	roles, err := s.roles.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("RoleService.ListRoles: %w", err)
	}
	return roles, nil
}
