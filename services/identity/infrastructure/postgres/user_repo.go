package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knobo/driftbase/services/identity/domain"
	domainports "github.com/knobo/driftbase/services/identity/domain/ports"
	sqlcidentity "github.com/knobo/driftbase/services/identity/infrastructure/postgres/sqlc"
)

// UserRepo is the PostgreSQL adapter implementing domain/ports.UserRepository.
type UserRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcidentity.Queries
}

var _ domainports.UserRepository = (*UserRepo)(nil)

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool, queries: sqlcidentity.New(pool)}
}

func (r *UserRepo) Upsert(ctx context.Context, user *domain.User) error {
	settings, err := json.Marshal(user.Settings)
	if err != nil {
		return fmt.Errorf("UserRepo.Upsert marshal settings: %w", err)
	}

	if err := r.queries.UpsertUserProfile(ctx, sqlcidentity.UpsertUserProfileParams{
		ID:          user.ID,
		AppID:       user.AppID,
		DisplayName: user.DisplayName,
		AvatarUrl:   user.AvatarURL,
		Bio:         user.Bio,
		Location:    user.Location,
		WebsiteUrl:  user.WebsiteURL,
		Settings:    settings,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("UserRepo.Upsert: %w", mapPgError(err))
	}
	return nil
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	settings, err := json.Marshal(user.Settings)
	if err != nil {
		return fmt.Errorf("UserRepo.Update marshal settings: %w", err)
	}

	if err := r.queries.UpdateUserProfile(ctx, sqlcidentity.UpdateUserProfileParams{
		ID:          user.ID,
		AppID:       user.AppID,
		DisplayName: user.DisplayName,
		AvatarUrl:   user.AvatarURL,
		Bio:         user.Bio,
		Location:    user.Location,
		WebsiteUrl:  user.WebsiteURL,
		Settings:    settings,
		UpdatedAt:   time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("UserRepo.Update: %w", mapPgError(err))
	}
	return nil
}

func (r *UserRepo) FindByID(ctx context.Context, id, appID uuid.UUID) (*domain.User, error) {
	row, err := r.queries.GetUserProfile(ctx, sqlcidentity.GetUserProfileParams{
		ID:    id,
		AppID: appID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo.FindByID: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		settings = make(map[string]any)
	}

	return &domain.User{
		ID:          row.ID,
		AppID:       row.AppID,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl,
		Bio:         row.Bio,
		Location:    row.Location,
		WebsiteURL:  row.WebsiteUrl,
		Settings:    settings,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// ─── error mapping ────────────────────────────────────────────────────────────

const (
	pgErrUniqueViolation    = "23505"
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
			return domain.ErrAlreadyExists
		case pgErrForeignKeyViolation:
			return domain.ErrInvalidInput("referenced resource does not exist")
		}
	}
	return err
}
