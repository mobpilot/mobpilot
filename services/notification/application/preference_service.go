package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/notification/application/ports"
	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// PreferenceService implements appports.PreferenceUseCase.
type PreferenceService struct {
	preferences domainports.PreferenceRepository
}

var _ appports.PreferenceUseCase = (*PreferenceService)(nil)

func NewPreferenceService(preferences domainports.PreferenceRepository) *PreferenceService {
	return &PreferenceService{preferences: preferences}
}

func (s *PreferenceService) GetPreferences(ctx context.Context, userID, appID uuid.UUID) ([]*domain.NotificationPreference, error) {
	prefs, err := s.preferences.FindByUser(ctx, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("PreferenceService.GetPreferences: %w", err)
	}
	return prefs, nil
}

func (s *PreferenceService) UpsertPreference(ctx context.Context, cmd appports.UpsertPreferenceCommand) error {
	p := &domain.NotificationPreference{
		UserID:    cmd.UserID,
		AppID:     cmd.AppID,
		Channel:   cmd.Channel,
		EventType: cmd.EventType,
		Enabled:   cmd.Enabled,
	}
	if err := s.preferences.Upsert(ctx, p); err != nil {
		return fmt.Errorf("PreferenceService.UpsertPreference: %w", err)
	}
	return nil
}
