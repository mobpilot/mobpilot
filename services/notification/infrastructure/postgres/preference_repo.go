package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
	sqlcn "github.com/mobpilot/mobpilot/services/notification/infrastructure/postgres/sqlc"
)

// PreferenceRepo is the PostgreSQL adapter implementing domain/ports.PreferenceRepository.
type PreferenceRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcn.Queries
}

var _ domainports.PreferenceRepository = (*PreferenceRepo)(nil)

func NewPreferenceRepo(pool *pgxpool.Pool) *PreferenceRepo {
	return &PreferenceRepo{pool: pool, queries: sqlcn.New(pool)}
}

func (r *PreferenceRepo) Upsert(ctx context.Context, p *domain.NotificationPreference) error {
	if err := r.queries.UpsertNotificationPreference(ctx, sqlcn.UpsertNotificationPreferenceParams{
		UserID:    p.UserID,
		AppID:     p.AppID,
		Channel:   string(p.Channel),
		EventType: p.EventType,
		Enabled:   p.Enabled,
	}); err != nil {
		return fmt.Errorf("PreferenceRepo.Upsert: %w", err)
	}
	return nil
}

func (r *PreferenceRepo) FindByUser(ctx context.Context, userID, appID uuid.UUID) ([]*domain.NotificationPreference, error) {
	rows, err := r.queries.GetNotificationPreferences(ctx, sqlcn.GetNotificationPreferencesParams{
		UserID: userID,
		AppID:  appID,
	})
	if err != nil {
		return nil, fmt.Errorf("PreferenceRepo.FindByUser: %w", err)
	}
	prefs := make([]*domain.NotificationPreference, len(rows))
	for i, row := range rows {
		prefs[i] = &domain.NotificationPreference{
			UserID:    row.UserID,
			AppID:     row.AppID,
			Channel:   domain.Channel(row.Channel),
			EventType: row.EventType,
			Enabled:   row.Enabled,
		}
	}
	return prefs, nil
}

// IsEnabled checks whether a user has push/email/in_app enabled for a given
// event type. Falls back to wildcard "*" if no specific preference exists.
// Default for push is enabled, default for email is disabled.
func (r *PreferenceRepo) IsEnabled(ctx context.Context, userID, appID uuid.UUID, channel domain.Channel, eventType string) (bool, error) {
	enabled, err := r.queries.GetPreferenceEnabled(ctx, sqlcn.GetPreferenceEnabledParams{
		UserID:    userID,
		AppID:     appID,
		Channel:   string(channel),
		EventType: eventType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No explicit preference — return default.
			return channel == domain.ChannelPush, nil // push on, email off
		}
		return false, fmt.Errorf("PreferenceRepo.IsEnabled: %w", err)
	}
	return enabled, nil
}
