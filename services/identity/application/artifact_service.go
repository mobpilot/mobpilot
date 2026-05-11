package application

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/identity/application/ports"
	"github.com/mobpilot/mobpilot/services/identity/domain"
	domainports "github.com/mobpilot/mobpilot/services/identity/domain/ports"
)

// ArtifactService implements appports.ArtifactUseCase.
type ArtifactService struct {
	artifacts domainports.ArtifactRepository
}

var _ appports.ArtifactUseCase = (*ArtifactService)(nil)

func NewArtifactService(artifacts domainports.ArtifactRepository) *ArtifactService {
	return &ArtifactService{artifacts: artifacts}
}

func (s *ArtifactService) Put(ctx context.Context, cmd appports.PutArtifactCommand) (*domain.Artifact, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "ArtifactService.Put")
	defer span.End()

	a, err := domain.NewArtifact(cmd.UserID, cmd.AppID, cmd.Key, cmd.Value)
	if err != nil {
		return nil, err
	}
	if err := s.artifacts.Upsert(ctx, a); err != nil {
		return nil, fmt.Errorf("ArtifactService.Put: %w", err)
	}
	return a, nil
}

func (s *ArtifactService) Get(ctx context.Context, cmd appports.GetArtifactCommand) (*domain.Artifact, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "ArtifactService.Get")
	defer span.End()

	a, err := s.artifacts.FindByKey(ctx, cmd.UserID, cmd.AppID, cmd.Key)
	if err != nil {
		return nil, fmt.Errorf("ArtifactService.Get: %w", err)
	}
	return a, nil
}

func (s *ArtifactService) List(ctx context.Context, userID, appID uuid.UUID) ([]*domain.Artifact, error) {
	ctx, span := otel.Tracer("identity").Start(ctx, "ArtifactService.List")
	defer span.End()

	items, err := s.artifacts.List(ctx, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("ArtifactService.List: %w", err)
	}
	return items, nil
}

func (s *ArtifactService) Delete(ctx context.Context, cmd appports.DeleteArtifactCommand) error {
	ctx, span := otel.Tracer("identity").Start(ctx, "ArtifactService.Delete")
	defer span.End()

	if err := s.artifacts.Delete(ctx, cmd.UserID, cmd.AppID, cmd.Key); err != nil {
		return fmt.Errorf("ArtifactService.Delete: %w", err)
	}
	return nil
}
