package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

// RoleRepo is the PostgreSQL adapter implementing domain/ports.RoleRepository.
type RoleRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcorg.Queries
}

var _ domainports.RoleRepository = (*RoleRepo)(nil)

func NewRoleRepo(pool *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{pool: pool, queries: sqlcorg.New(pool)}
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	if err := r.queries.CreateRole(ctx, sqlcorg.CreateRoleParams{
		ID:          role.ID,
		OrgID:       role.OrgID,
		Name:        role.Name,
		Permissions: role.Permissions,
		CreatedAt:   role.CreatedAt,
	}); err != nil {
		return fmt.Errorf("RoleRepo.Create: %w", mapPgError(err))
	}
	return nil
}

func (r *RoleRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	row, err := r.queries.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, fmt.Errorf("RoleRepo.FindByID: %w", err)
	}
	return &domain.Role{
		ID:          row.ID,
		OrgID:       row.OrgID,
		Name:        row.Name,
		Permissions: row.Permissions,
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (r *RoleRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	rows, err := r.queries.ListRolesByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("RoleRepo.ListByOrg: %w", err)
	}
	roles := make([]*domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = &domain.Role{
			ID:          row.ID,
			OrgID:       row.OrgID,
			Name:        row.Name,
			Permissions: row.Permissions,
			CreatedAt:   row.CreatedAt,
		}
	}
	return roles, nil
}

func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("RoleRepo.Delete: %w", err)
	}
	return nil
}
