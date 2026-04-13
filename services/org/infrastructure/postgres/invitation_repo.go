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

// InvitationRepo is the PostgreSQL adapter implementing domain/ports.InvitationRepository.
type InvitationRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcorg.Queries
}

var _ domainports.InvitationRepository = (*InvitationRepo)(nil)

func NewInvitationRepo(pool *pgxpool.Pool) *InvitationRepo {
	return &InvitationRepo{pool: pool, queries: sqlcorg.New(pool)}
}

func (r *InvitationRepo) Create(ctx context.Context, inv *domain.Invitation) error {
	if err := r.queries.CreateInvitation(ctx, sqlcorg.CreateInvitationParams{
		ID:        inv.ID,
		OrgID:     inv.OrgID,
		Email:     inv.Email,
		Role:      string(inv.Role),
		Token:     inv.Token,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}); err != nil {
		return fmt.Errorf("InvitationRepo.Create: %w", mapPgError(err))
	}
	return nil
}

func (r *InvitationRepo) FindByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	row, err := r.queries.GetInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInviteNotFound
		}
		return nil, fmt.Errorf("InvitationRepo.FindByToken: %w", err)
	}
	inv := &domain.Invitation{
		ID:        row.ID,
		OrgID:     row.OrgID,
		Email:     row.Email,
		Role:      domain.MemberRole(row.Role),
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
	if row.AcceptedAt.Valid {
		t := row.AcceptedAt.Time
		inv.AcceptedAt = &t
	}
	return inv, nil
}

func (r *InvitationRepo) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.MarkInvitationAccepted(ctx, id); err != nil {
		return fmt.Errorf("InvitationRepo.MarkAccepted: %w", err)
	}
	return nil
}
