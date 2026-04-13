package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

// MemberRepo is the PostgreSQL adapter implementing domain/ports.MemberRepository.
type MemberRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcorg.Queries
}

var _ domainports.MemberRepository = (*MemberRepo)(nil)

func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{pool: pool, queries: sqlcorg.New(pool)}
}

func (r *MemberRepo) Add(ctx context.Context, member *domain.OrgMember) error {
	var invitedBy pgtype.UUID
	if member.InvitedBy != nil {
		invitedBy = pgtype.UUID{Bytes: [16]byte(*member.InvitedBy), Valid: true}
	}
	if err := r.queries.AddOrgMember(ctx, sqlcorg.AddOrgMemberParams{
		OrgID:     member.OrgID,
		UserID:    member.UserID,
		Role:      string(member.Role),
		InvitedBy: invitedBy,
		JoinedAt:  member.JoinedAt,
	}); err != nil {
		return fmt.Errorf("MemberRepo.Add: %w", mapPgError(err))
	}
	return nil
}

func (r *MemberRepo) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	if err := r.queries.RemoveOrgMember(ctx, sqlcorg.RemoveOrgMemberParams{
		OrgID:  orgID,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("MemberRepo.Remove: %w", err)
	}
	return nil
}

func (r *MemberRepo) FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error) {
	row, err := r.queries.GetOrgMember(ctx, sqlcorg.GetOrgMemberParams{
		OrgID:  orgID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMemberNotFound
		}
		return nil, fmt.Errorf("MemberRepo.FindByOrgAndUser: %w", err)
	}
	return rowToMember(row), nil
}

func (r *MemberRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error) {
	rows, err := r.queries.ListOrgMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("MemberRepo.ListByOrg: %w", err)
	}
	members := make([]*domain.OrgMember, len(rows))
	for i, row := range rows {
		members[i] = rowToMember(row)
	}
	return members, nil
}

func rowToMember(row sqlcorg.OrgMember) *domain.OrgMember {
	m := &domain.OrgMember{
		OrgID:    row.OrgID,
		UserID:   row.UserID,
		Role:     domain.MemberRole(row.Role),
		JoinedAt: row.JoinedAt,
	}
	if row.InvitedBy.Valid {
		id := uuid.UUID(row.InvitedBy.Bytes)
		m.InvitedBy = &id
	}
	return m
}
