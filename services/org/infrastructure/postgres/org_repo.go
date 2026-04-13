package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

// OrgRepo is the PostgreSQL adapter implementing domain/ports.OrgRepository.
type OrgRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcorg.Queries
}

var _ domainports.OrgRepository = (*OrgRepo)(nil)

func NewOrgRepo(pool *pgxpool.Pool) *OrgRepo {
	return &OrgRepo{pool: pool, queries: sqlcorg.New(pool)}
}

func (r *OrgRepo) Create(ctx context.Context, org *domain.Organization) error {
	if err := r.queries.CreateOrganization(ctx, sqlcorg.CreateOrganizationParams{
		ID:          org.ID,
		Name:        org.Name,
		Slug:        org.Slug,
		OwnerUserID: org.OwnerUserID,
		Plan:        org.Plan,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("OrgRepo.Create: %w", mapPgError(err))
	}
	return nil
}

func (r *OrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	row, err := r.queries.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrgNotFound
		}
		return nil, fmt.Errorf("OrgRepo.FindByID: %w", err)
	}
	return rowToOrg(row), nil
}

func (r *OrgRepo) FindBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	row, err := r.queries.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrgNotFound
		}
		return nil, fmt.Errorf("OrgRepo.FindBySlug: %w", err)
	}
	return rowToOrg(row), nil
}

func (r *OrgRepo) Update(ctx context.Context, org *domain.Organization) error {
	if err := r.queries.UpdateOrganization(ctx, sqlcorg.UpdateOrganizationParams{
		ID:        org.ID,
		Name:      org.Name,
		Plan:      org.Plan,
		UpdatedAt: org.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("OrgRepo.Update: %w", mapPgError(err))
	}
	return nil
}

func rowToOrg(row sqlcorg.Organization) *domain.Organization {
	org := &domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Slug:        row.Slug,
		OwnerUserID: row.OwnerUserID,
		Plan:        row.Plan,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.SuspendedAt.Valid {
		t := row.SuspendedAt.Time
		org.SuspendedAt = &t
	}
	return org
}

// ─── error mapping ────────────────────────────────────────────────────────────

const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
)

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			if pgErr.ConstraintName == "uq_organizations_slug" || pgErr.ConstraintName == "organizations_slug_key" {
				return domain.ErrSlugTaken
			}
			return domain.ErrAlreadyExists
		case pgErrForeignKeyViolation:
			return domain.ErrInvalidInput("referenced resource does not exist")
		}
	}
	return err
}
