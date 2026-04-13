package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

// GroupRepo is the PostgreSQL adapter implementing domain/ports.GroupRepository.
type GroupRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcorg.Queries
}

var _ domainports.GroupRepository = (*GroupRepo)(nil)

func NewGroupRepo(pool *pgxpool.Pool) *GroupRepo {
	return &GroupRepo{pool: pool, queries: sqlcorg.New(pool)}
}

func (r *GroupRepo) Create(ctx context.Context, group *domain.Group) error {
	var expiresAt pgtype.Timestamptz
	if group.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *group.ExpiresAt, Valid: true}
	}
	if err := r.queries.CreateGroup(ctx, sqlcorg.CreateGroupParams{
		ID:          group.ID,
		OrgID:       group.OrgID,
		AppID:       group.AppID,
		Name:        group.Name,
		Description: group.Description,
		Ephemeral:   group.Ephemeral,
		ExpiresAt:   expiresAt,
		CreatedBy:   group.CreatedBy,
		CreatedAt:   group.CreatedAt,
	}); err != nil {
		return fmt.Errorf("GroupRepo.Create: %w", mapPgError(err))
	}
	return nil
}

func (r *GroupRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Group, error) {
	row, err := r.queries.GetGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("GroupRepo.FindByID: %w", err)
	}
	return rowToGroup(row), nil
}

func (r *GroupRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Group, error) {
	rows, err := r.queries.ListGroupsByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("GroupRepo.ListByOrg: %w", err)
	}
	groups := make([]*domain.Group, len(rows))
	for i, row := range rows {
		groups[i] = rowToGroup(row)
	}
	return groups, nil
}

func (r *GroupRepo) ListExpired(ctx context.Context, now time.Time) ([]*domain.Group, error) {
	rows, err := r.queries.ListExpiredGroups(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("GroupRepo.ListExpired: %w", err)
	}
	groups := make([]*domain.Group, len(rows))
	for i, row := range rows {
		groups[i] = rowToGroup(row)
	}
	return groups, nil
}

func (r *GroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteGroup(ctx, id); err != nil {
		return fmt.Errorf("GroupRepo.Delete: %w", err)
	}
	return nil
}

func (r *GroupRepo) AddMember(ctx context.Context, groupID, userID, appID uuid.UUID) error {
	if err := r.queries.AddGroupMember(ctx, sqlcorg.AddGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
		AppID:   appID,
	}); err != nil {
		return fmt.Errorf("GroupRepo.AddMember: %w", mapPgError(err))
	}
	return nil
}

func (r *GroupRepo) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	if err := r.queries.RemoveGroupMember(ctx, sqlcorg.RemoveGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("GroupRepo.RemoveMember: %w", err)
	}
	return nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, groupID uuid.UUID) ([]*domain.GroupMember, error) {
	rows, err := r.queries.ListGroupMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("GroupRepo.ListMembers: %w", err)
	}
	members := make([]*domain.GroupMember, len(rows))
	for i, row := range rows {
		members[i] = &domain.GroupMember{
			GroupID: row.GroupID,
			UserID:  row.UserID,
			AppID:   row.AppID,
		}
	}
	return members, nil
}

// rowToGroup maps a sqlc Group row to a domain Group.
func rowToGroup(row sqlcorg.Group) *domain.Group {
	g := &domain.Group{
		ID:          row.ID,
		OrgID:       row.OrgID,
		AppID:       row.AppID,
		Name:        row.Name,
		Description: row.Description,
		Ephemeral:   row.Ephemeral,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		g.ExpiresAt = &t
	}
	return g
}
