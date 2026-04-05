package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/identity/domain"
	domainports "github.com/mobpilot/mobpilot/services/identity/domain/ports"
	sqlcidentity "github.com/mobpilot/mobpilot/services/identity/infrastructure/postgres/sqlc"
)

// DeviceRepo is the PostgreSQL adapter implementing domain/ports.DeviceTokenRepository.
type DeviceRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcidentity.Queries
}

var _ domainports.DeviceTokenRepository = (*DeviceRepo)(nil)

func NewDeviceRepo(pool *pgxpool.Pool) *DeviceRepo {
	return &DeviceRepo{pool: pool, queries: sqlcidentity.New(pool)}
}

func (r *DeviceRepo) Upsert(ctx context.Context, dt *domain.DeviceToken) error {
	if err := r.queries.UpsertDeviceToken(ctx, sqlcidentity.UpsertDeviceTokenParams{
		ID:        dt.ID,
		UserID:    dt.UserID,
		AppID:     dt.AppID,
		Platform:  string(dt.Platform),
		TokenType: string(dt.TokenType),
		Token:     dt.Token,
		DeviceID:  dt.DeviceID,
		CreatedAt: dt.CreatedAt,
		UpdatedAt: dt.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("DeviceRepo.Upsert: %w", mapPgError(err))
	}
	return nil
}

func (r *DeviceRepo) Delete(ctx context.Context, appID uuid.UUID, deviceID string) error {
	if err := r.queries.DeleteDeviceToken(ctx, sqlcidentity.DeleteDeviceTokenParams{
		AppID:    appID,
		DeviceID: deviceID,
	}); err != nil {
		return fmt.Errorf("DeviceRepo.Delete: %w", err)
	}
	return nil
}

func (r *DeviceRepo) FindByUser(ctx context.Context, userID, appID uuid.UUID) ([]*domain.DeviceToken, error) {
	rows, err := r.queries.GetDeviceTokensByUser(ctx, sqlcidentity.GetDeviceTokensByUserParams{
		UserID: userID,
		AppID:  appID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("DeviceRepo.FindByUser: %w", err)
	}

	tokens := make([]*domain.DeviceToken, len(rows))
	for i, row := range rows {
		tokens[i] = &domain.DeviceToken{
			ID:        row.ID,
			UserID:    row.UserID,
			AppID:     row.AppID,
			Platform:  domain.Platform(row.Platform),
			TokenType: domain.TokenType(row.TokenType),
			Token:     row.Token,
			DeviceID:  row.DeviceID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return tokens, nil
}
