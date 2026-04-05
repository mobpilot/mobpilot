package application

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/google/uuid"
	appports "github.com/knobo/driftbase/services/identity/application/ports"
	"github.com/knobo/driftbase/services/identity/domain"
	domainports "github.com/knobo/driftbase/services/identity/domain/ports"
)

// UserService implements appports.UserUseCase.
type UserService struct {
	users     domainports.UserRepository
	publisher domainports.EventPublisher
}

// Compile-time interface check.
var _ appports.UserUseCase = (*UserService)(nil)

func NewUserService(users domainports.UserRepository, publisher domainports.EventPublisher) *UserService {
	return &UserService{users: users, publisher: publisher}
}

func (s *UserService) GetMe(ctx context.Context, cmd appports.GetMeCommand) (*domain.User, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "UserService.GetMe")
	defer span.End()
	span.SetAttributes(
		attribute.String("user.id", cmd.UserID.String()),
		attribute.String("app.id", cmd.AppID.String()),
	)

	user, err := s.users.FindByID(ctx, cmd.UserID, cmd.AppID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("UserService.GetMe: %w", err)
	}

	// Auto-provision profile on first login.
	user, err = domain.NewUser(cmd.UserID, cmd.AppID, "")
	if err != nil {
		return nil, fmt.Errorf("UserService.GetMe new user: %w", err)
	}
	if err := s.users.Upsert(ctx, user); err != nil {
		return nil, fmt.Errorf("UserService.GetMe upsert: %w", err)
	}
	if err := s.publisher.Publish(ctx, user.PopEvents()); err != nil {
		// Non-fatal: log but do not fail the request.
		// TODO: use transactional outbox for guaranteed delivery.
		_ = err
	}
	return user, nil
}

func (s *UserService) UpdateMe(ctx context.Context, cmd appports.UpdateMeCommand) (*domain.User, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "UserService.UpdateMe")
	defer span.End()

	user, err := s.users.FindByID(ctx, cmd.UserID, cmd.AppID)
	if err != nil {
		return nil, fmt.Errorf("UserService.UpdateMe find: %w", err)
	}

	user.UpdateProfile(cmd.Patch)

	if err := s.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("UserService.UpdateMe update: %w", err)
	}
	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, userID, appID uuid.UUID) (*domain.User, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "UserService.GetUser")
	defer span.End()

	user, err := s.users.FindByID(ctx, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetUser: %w", err)
	}
	return user, nil
}
