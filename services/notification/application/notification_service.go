package application

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	appports "github.com/mobpilot/mobpilot/services/notification/application/ports"
	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// fanOutConcurrency limits concurrent per-member goroutines in SendToGroup.
const fanOutConcurrency = 10

// NotificationService implements appports.NotificationUseCase.
type NotificationService struct {
	notifications domainports.NotificationRepository
	preferences   domainports.PreferenceRepository
	push          domainports.PushPort
	email         domainports.EmailPort
	realtime      domainports.RealtimePort
	groups        domainports.GroupQueryPort
	devices       domainports.DeviceQueryPort
	logger        *slog.Logger
}

var _ appports.NotificationUseCase = (*NotificationService)(nil)

func NewNotificationService(
	notifications domainports.NotificationRepository,
	preferences domainports.PreferenceRepository,
	push domainports.PushPort,
	email domainports.EmailPort,
	realtime domainports.RealtimePort,
	groups domainports.GroupQueryPort,
	devices domainports.DeviceQueryPort,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		preferences:   preferences,
		push:          push,
		email:         email,
		realtime:      realtime,
		groups:        groups,
		devices:       devices,
		logger:        logger,
	}
}

// SendToUser delivers a notification to a single user via all enabled channels.
func (s *NotificationService) SendToUser(ctx context.Context, cmd appports.SendToUserCommand) (*domain.Notification, error) {
	n := domain.NewNotification(cmd.UserID, cmd.AppID, cmd.Title, cmd.Body)
	n.GroupID = cmd.GroupID
	n.OrgID = cmd.OrgID
	if cmd.Data != nil {
		n.Data = cmd.Data
	}

	if err := s.notifications.Save(ctx, n); err != nil {
		return nil, fmt.Errorf("NotificationService.SendToUser save: %w", err)
	}

	s.deliverToUser(ctx, n, cmd.EventType)
	return n, nil
}

// SendToGroup fan-outs a notification to all members of a group.
// Step 1: publish real-time to the group channel (instant, online clients).
// Step 2: persist + push to each member individually (offline clients).
func (s *NotificationService) SendToGroup(ctx context.Context, cmd appports.SendToGroupCommand) error {
	// 1. Real-time broadcast to the group channel (best-effort).
	data := cmd.Data
	if data == nil {
		data = map[string]any{}
	}
	data["title"] = cmd.Title
	data["body"] = cmd.Body

	if err := s.realtime.Publish(ctx, domainports.RealtimeMessage{
		Channel: "group:" + cmd.GroupID.String(),
		Data:    data,
	}); err != nil {
		s.logger.WarnContext(ctx, "realtime publish failed", "group_id", cmd.GroupID, "err", err)
	}

	// 2. Fetch members and fan-out push/email per member.
	members, err := s.groups.ListGroupMembers(ctx, cmd.GroupID)
	if err != nil {
		return fmt.Errorf("NotificationService.SendToGroup list members: %w", err)
	}

	sem := make(chan struct{}, fanOutConcurrency)
	g, gCtx := errgroup.WithContext(ctx)

	for _, m := range members {
		m := m // capture
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()
			memberCmd := appports.SendToUserCommand{
				UserID:    m.UserID,
				AppID:     m.AppID,
				OrgID:     &cmd.OrgID,
				GroupID:   &cmd.GroupID,
				EventType: cmd.EventType,
				Title:     cmd.Title,
				Body:      cmd.Body,
				Data:      cmd.Data,
			}
			if _, err := s.SendToUser(gCtx, memberCmd); err != nil {
				s.logger.WarnContext(gCtx, "per-member notification failed",
					"user_id", m.UserID, "err", err)
			}
			return nil // non-fatal: don't abort the whole fan-out
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("NotificationService.SendToGroup: %w", err)
	}
	return nil
}

// MarkRead marks a notification as read by its owner.
func (s *NotificationService) MarkRead(ctx context.Context, cmd appports.MarkReadCommand) error {
	n, err := s.notifications.FindByID(ctx, cmd.NotificationID)
	if err != nil {
		return fmt.Errorf("NotificationService.MarkRead find: %w", err)
	}
	if n.UserID != cmd.UserID {
		return domain.ErrNotAuthorized
	}
	if err := s.notifications.MarkRead(ctx, cmd.NotificationID); err != nil {
		return fmt.Errorf("NotificationService.MarkRead: %w", err)
	}
	return nil
}

// ListForUser returns paginated notifications for a user.
func (s *NotificationService) ListForUser(ctx context.Context, userID, appID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	items, err := s.notifications.ListForUser(ctx, userID, appID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationService.ListForUser: %w", err)
	}
	return items, nil
}

// deliverToUser sends a notification via push and email based on preferences.
// Errors are logged but never propagated — delivery failures are non-fatal.
func (s *NotificationService) deliverToUser(ctx context.Context, n *domain.Notification, eventType string) {
	s.tryPush(ctx, n, eventType)
	s.tryEmail(ctx, n, eventType)
}

func (s *NotificationService) tryPush(ctx context.Context, n *domain.Notification, eventType string) {
	enabled, err := s.preferences.IsEnabled(ctx, n.UserID, n.AppID, domain.ChannelPush, eventType)
	if err != nil {
		// Preference lookup failed — default to enabled so the user is not silently skipped.
		s.logger.WarnContext(ctx, "push: preference lookup failed, defaulting to enabled",
			"user_id", n.UserID, "err", err)
	} else if !enabled {
		return
	}

	tokens, err := s.devices.FindTokensByUser(ctx, n.UserID.String(), n.AppID.String())
	if err != nil {
		s.logger.WarnContext(ctx, "push: device lookup failed", "user_id", n.UserID, "err", err)
		return
	}

	strData := make(map[string]string, len(n.Data))
	for k, v := range n.Data {
		if sv, ok := v.(string); ok {
			strData[k] = sv
		}
	}
	strData["notification_id"] = n.ID.String()

	for _, dt := range tokens {
		msg := domainports.PushMessage{
			Token:    dt.Token,
			Platform: dt.Platform,
			Title:    n.Title,
			Body:     n.Body,
			Data:     strData,
		}
		if err := s.push.Send(ctx, dt.TokenType, msg); err != nil {
			s.logger.WarnContext(ctx, "push send failed",
				"user_id", n.UserID, "token_type", dt.TokenType, "err", err)
		}
	}
}

func (s *NotificationService) tryEmail(ctx context.Context, n *domain.Notification, eventType string) {
	enabled, err := s.preferences.IsEnabled(ctx, n.UserID, n.AppID, domain.ChannelEmail, eventType)
	if err != nil {
		// Email is opt-in: on lookup failure default to disabled (safe).
		s.logger.WarnContext(ctx, "email: preference lookup failed, defaulting to disabled",
			"user_id", n.UserID, "err", err)
		return
	}
	if !enabled {
		return
	}

	emailAddr, _ := n.Data["email"].(string)
	if emailAddr == "" {
		return
	}

	if err := s.email.Send(ctx, domainports.EmailMessage{
		To:      emailAddr,
		Subject: n.Title,
		Body:    n.Body,
	}); err != nil {
		s.logger.WarnContext(ctx, "email send failed", "user_id", n.UserID, "err", err)
	}
}
