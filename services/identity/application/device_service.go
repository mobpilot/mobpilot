package application

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	appports "github.com/mobpilot/mobpilot/services/identity/application/ports"
	"github.com/mobpilot/mobpilot/services/identity/domain"
	domainports "github.com/mobpilot/mobpilot/services/identity/domain/ports"
)

// DeviceService implements appports.DeviceTokenUseCase.
type DeviceService struct {
	devices domainports.DeviceTokenRepository
}

var _ appports.DeviceTokenUseCase = (*DeviceService)(nil)

func NewDeviceService(devices domainports.DeviceTokenRepository) *DeviceService {
	return &DeviceService{devices: devices}
}

func (s *DeviceService) RegisterDevice(ctx context.Context, cmd appports.RegisterDeviceCommand) (*domain.DeviceToken, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "DeviceService.RegisterDevice")
	defer span.End()

	dt, err := domain.NewDeviceToken(
		cmd.UserID, cmd.AppID,
		cmd.Platform, cmd.TokenType,
		cmd.Token, cmd.DeviceID,
	)
	if err != nil {
		return nil, err
	}
	if err := s.devices.Upsert(ctx, dt); err != nil {
		return nil, fmt.Errorf("DeviceService.RegisterDevice upsert: %w", err)
	}
	return dt, nil
}

func (s *DeviceService) UnregisterDevice(ctx context.Context, cmd appports.UnregisterDeviceCommand) error {
	ctx, span := otel.Tracer("identity").Start(ctx, "DeviceService.UnregisterDevice")
	defer span.End()

	if err := s.devices.Delete(ctx, cmd.AppID, cmd.DeviceID); err != nil {
		return fmt.Errorf("DeviceService.UnregisterDevice: %w", err)
	}
	return nil
}
