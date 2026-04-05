package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knobo/driftbase/services/identity/domain"
	domainports "github.com/knobo/driftbase/services/identity/domain/ports"
	sqlcidentity "github.com/knobo/driftbase/services/identity/infrastructure/postgres/sqlc"
)

// ArtifactRepo is the PostgreSQL adapter implementing domain/ports.ArtifactRepository.
type ArtifactRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcidentity.Queries
}

var _ domainports.ArtifactRepository = (*ArtifactRepo)(nil)

func NewArtifactRepo(pool *pgxpool.Pool) *ArtifactRepo {
	return &ArtifactRepo{pool: pool, queries: sqlcidentity.New(pool)}
}

func (r *ArtifactRepo) Upsert(ctx context.Context, a *domain.Artifact) error {
	value, err := json.Marshal(a.Value)
	if err != nil {
		return fmt.Errorf("ArtifactRepo.Upsert marshal: %w", err)
	}
	if err := r.queries.UpsertArtifact(ctx, sqlcidentity.UpsertArtifactParams{
		ID:        a.ID,
		UserID:    a.UserID,
		AppID:     a.AppID,
		Key:       a.Key,
		Value:     value,
		CreatedAt: a.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("ArtifactRepo.Upsert: %w", mapPgError(err))
	}
	return nil
}

func (r *ArtifactRepo) FindByKey(ctx context.Context, userID, appID uuid.UUID, key string) (*domain.Artifact, error) {
	row, err := r.queries.GetArtifact(ctx, sqlcidentity.GetArtifactParams{
		UserID: userID,
		AppID:  appID,
		Key:    key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrArtifactNotFound
		}
		return nil, fmt.Errorf("ArtifactRepo.FindByKey: %w", err)
	}

	var value any
	if row.Value != nil {
		_ = json.Unmarshal(row.Value, &value)
	}
	return &domain.Artifact{
		ID:        row.ID,
		UserID:    row.UserID,
		AppID:     row.AppID,
		Key:       row.Key,
		Value:     value,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *ArtifactRepo) List(ctx context.Context, userID, appID uuid.UUID) ([]*domain.Artifact, error) {
	rows, err := r.queries.ListArtifacts(ctx, sqlcidentity.ListArtifactsParams{
		UserID: userID,
		AppID:  appID,
	})
	if err != nil {
		return nil, fmt.Errorf("ArtifactRepo.List: %w", err)
	}
	artifacts := make([]*domain.Artifact, len(rows))
	for i, row := range rows {
		var value any
		if row.Value != nil {
			_ = json.Unmarshal(row.Value, &value)
		}
		artifacts[i] = &domain.Artifact{
			ID:        row.ID,
			UserID:    row.UserID,
			AppID:     row.AppID,
			Key:       row.Key,
			Value:     value,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return artifacts, nil
}

func (r *ArtifactRepo) Delete(ctx context.Context, userID, appID uuid.UUID, key string) error {
	if err := r.queries.DeleteArtifact(ctx, sqlcidentity.DeleteArtifactParams{
		UserID: userID,
		AppID:  appID,
		Key:    key,
	}); err != nil {
		return fmt.Errorf("ArtifactRepo.Delete: %w", err)
	}
	return nil
}
