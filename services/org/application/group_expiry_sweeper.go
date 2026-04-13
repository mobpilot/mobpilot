package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/mobpilot/mobpilot/services/org/domain"
	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// GroupExpirySweeper periodically deletes ephemeral groups whose expires_at has passed.
// It cleans up Keto relations (for any permanent-style cleanup), deletes group_members,
// deletes the group row, and publishes a GroupDeletedEvent.
type GroupExpirySweeper struct {
	groups    domainports.GroupRepository
	publisher domainports.EventPublisher
	authz     domainports.AuthzPort
	logger    *slog.Logger
	interval  time.Duration
}

func NewGroupExpirySweeper(
	groups domainports.GroupRepository,
	publisher domainports.EventPublisher,
	authz domainports.AuthzPort,
	logger *slog.Logger,
) *GroupExpirySweeper {
	return &GroupExpirySweeper{
		groups:    groups,
		publisher: publisher,
		authz:     authz,
		logger:    logger,
		interval:  60 * time.Second,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (s *GroupExpirySweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *GroupExpirySweeper) sweep(ctx context.Context) {
	expired, err := s.groups.ListExpired(ctx, time.Now().UTC())
	if err != nil {
		s.logger.ErrorContext(ctx, "group expiry sweep: list expired", "err", err)
		return
	}
	for _, g := range expired {
		s.deleteGroup(ctx, g)
	}
}

func (s *GroupExpirySweeper) deleteGroup(ctx context.Context, g *domain.Group) {
	// For permanent-style cleanup: remove any Keto relations.
	// Ephemeral groups don't write Keto tuples, but we clean up defensively.
	members, err := s.groups.ListMembers(ctx, g.ID)
	if err != nil {
		s.logger.WarnContext(ctx, "group expiry: list members", "group_id", g.ID, "err", err)
	}
	for _, m := range members {
		_ = s.authz.DeleteRelation(ctx, m.UserID.String(), "member", "Group:"+g.ID.String())
	}
	_ = s.authz.DeleteRelation(ctx, g.CreatedBy.String(), "admin", "Group:"+g.ID.String())

	if err := s.groups.Delete(ctx, g.ID); err != nil {
		s.logger.ErrorContext(ctx, "group expiry: delete group", "group_id", g.ID, "err", err)
		return
	}

	_ = s.publisher.Publish(ctx, []domain.DomainEvent{domain.GroupDeletedEvent{
		GroupID:     g.ID,
		OrgID:       g.OrgID,
		AppID:       g.AppID,
		OccurredAt_: time.Now().UTC(),
	}})

	s.logger.InfoContext(ctx, "expired group deleted", "group_id", g.ID, "org_id", g.OrgID)
}
