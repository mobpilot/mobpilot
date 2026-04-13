package application_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/notification/application"
	appports "github.com/mobpilot/mobpilot/services/notification/application/ports"
	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// ─── Stubs ────────────────────────────────────────────────────────────────────

type stubNotificationRepo struct {
	mu       sync.Mutex
	saved    []*domain.Notification
	findByID map[uuid.UUID]*domain.Notification
	marked   []uuid.UUID
}

func (r *stubNotificationRepo) Save(_ context.Context, n *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, n)
	if r.findByID == nil {
		r.findByID = map[uuid.UUID]*domain.Notification{}
	}
	r.findByID[n.ID] = n
	return nil
}

func (r *stubNotificationRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.findByID[id]; ok {
		return n, nil
	}
	return nil, domain.ErrNotificationNotFound
}

func (r *stubNotificationRepo) ListForUser(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saved, nil
}

func (r *stubNotificationRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.DeliveryStatus) error {
	return nil
}

func (r *stubNotificationRepo) MarkRead(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.marked = append(r.marked, id)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type stubPreferenceRepo struct {
	// channel → enabled; missing keys fall back to default
	prefs map[domain.Channel]bool
}

func (r *stubPreferenceRepo) IsEnabled(_ context.Context, _, _ uuid.UUID, ch domain.Channel, _ string) (bool, error) {
	if v, ok := r.prefs[ch]; ok {
		return v, nil
	}
	// default: push on, email off
	return ch == domain.ChannelPush, nil
}

func (r *stubPreferenceRepo) Upsert(_ context.Context, _ *domain.NotificationPreference) error {
	return nil
}

func (r *stubPreferenceRepo) FindByUser(_ context.Context, _, _ uuid.UUID) ([]*domain.NotificationPreference, error) {
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type stubPush struct {
	calls []domainports.PushMessage
	err   error
}

func (p *stubPush) Send(_ context.Context, _ string, msg domainports.PushMessage) error {
	p.calls = append(p.calls, msg)
	return p.err
}

// ─────────────────────────────────────────────────────────────────────────────

type stubEmail struct {
	calls []domainports.EmailMessage
}

func (e *stubEmail) Send(_ context.Context, msg domainports.EmailMessage) error {
	e.calls = append(e.calls, msg)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type stubRealtime struct {
	calls []domainports.RealtimeMessage
	err   error
}

func (rt *stubRealtime) Publish(_ context.Context, msg domainports.RealtimeMessage) error {
	rt.calls = append(rt.calls, msg)
	return rt.err
}

// ─────────────────────────────────────────────────────────────────────────────

type stubGroups struct {
	members []*domainports.GroupMember
}

func (g *stubGroups) ListGroupMembers(_ context.Context, _ uuid.UUID) ([]*domainports.GroupMember, error) {
	return g.members, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type stubDevices struct {
	tokens []*domain.DeviceToken
}

func (d *stubDevices) FindTokensByUser(_ context.Context, _, _ string) ([]*domain.DeviceToken, error) {
	return d.tokens, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newSvc(
	repo *stubNotificationRepo,
	prefs *stubPreferenceRepo,
	push *stubPush,
	email *stubEmail,
	rt *stubRealtime,
	groups *stubGroups,
	devices *stubDevices,
) *application.NotificationService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return application.NewNotificationService(repo, prefs, push, email, rt, groups, devices, logger)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestSendToUser_SavesNotification(t *testing.T) {
	repo := &stubNotificationRepo{}
	svc := newSvc(repo, &stubPreferenceRepo{}, &stubPush{}, &stubEmail{}, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	userID := uuid.New()
	appID := uuid.New()

	n, err := svc.SendToUser(context.Background(), appports.SendToUserCommand{
		UserID: userID, AppID: appID,
		EventType: "test.event",
		Title:     "Hello", Body: "World",
	})
	require.NoError(t, err)
	assert.Equal(t, userID, n.UserID)
	assert.Equal(t, appID, n.AppID)
	assert.Equal(t, "Hello", n.Title)
	assert.Len(t, repo.saved, 1)
}

func TestSendToUser_PushSentWhenEnabled(t *testing.T) {
	repo := &stubNotificationRepo{}
	prefs := &stubPreferenceRepo{prefs: map[domain.Channel]bool{domain.ChannelPush: true}}
	push := &stubPush{}
	devices := &stubDevices{tokens: []*domain.DeviceToken{
		{Token: "tok1", TokenType: "fcm", Platform: "android"},
	}}
	svc := newSvc(repo, prefs, push, &stubEmail{}, &stubRealtime{}, &stubGroups{}, devices)

	_, err := svc.SendToUser(context.Background(), appports.SendToUserCommand{
		UserID: uuid.New(), AppID: uuid.New(),
		EventType: "test.event",
		Title:     "T", Body: "B",
	})
	require.NoError(t, err)
	assert.Len(t, push.calls, 1)
	assert.Equal(t, "tok1", push.calls[0].Token)
}

func TestSendToUser_PushNotSentWhenDisabled(t *testing.T) {
	prefs := &stubPreferenceRepo{prefs: map[domain.Channel]bool{domain.ChannelPush: false}}
	push := &stubPush{}
	svc := newSvc(&stubNotificationRepo{}, prefs, push, &stubEmail{}, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	_, err := svc.SendToUser(context.Background(), appports.SendToUserCommand{
		UserID: uuid.New(), AppID: uuid.New(),
		EventType: "test.event",
		Title:     "T", Body: "B",
	})
	require.NoError(t, err)
	assert.Empty(t, push.calls)
}

func TestSendToUser_EmailSentWhenEnabled(t *testing.T) {
	prefs := &stubPreferenceRepo{prefs: map[domain.Channel]bool{
		domain.ChannelPush:  false,
		domain.ChannelEmail: true,
	}}
	email := &stubEmail{}
	svc := newSvc(&stubNotificationRepo{}, prefs, &stubPush{}, email, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	_, err := svc.SendToUser(context.Background(), appports.SendToUserCommand{
		UserID: uuid.New(), AppID: uuid.New(),
		EventType: "test.event",
		Title:     "Subject", Body: "Body text",
		Data: map[string]any{"email": "user@example.com"},
	})
	require.NoError(t, err)
	require.Len(t, email.calls, 1)
	assert.Equal(t, "user@example.com", email.calls[0].To)
	assert.Equal(t, "Subject", email.calls[0].Subject)
}

func TestSendToUser_EmailNotSentByDefault(t *testing.T) {
	// Default: email disabled for ChannelEmail (stubPreferenceRepo falls back to off)
	email := &stubEmail{}
	svc := newSvc(&stubNotificationRepo{}, &stubPreferenceRepo{}, &stubPush{}, email, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	_, err := svc.SendToUser(context.Background(), appports.SendToUserCommand{
		UserID: uuid.New(), AppID: uuid.New(),
		Title: "T", Body: "B",
		Data: map[string]any{"email": "user@example.com"},
	})
	require.NoError(t, err)
	assert.Empty(t, email.calls)
}

func TestSendToGroup_PublishesToRealtime(t *testing.T) {
	rt := &stubRealtime{}
	groupID := uuid.New()
	svc := newSvc(&stubNotificationRepo{}, &stubPreferenceRepo{}, &stubPush{}, &stubEmail{}, rt,
		&stubGroups{members: []*domainports.GroupMember{}}, &stubDevices{})

	err := svc.SendToGroup(context.Background(), appports.SendToGroupCommand{
		GroupID: groupID, AppID: uuid.New(), OrgID: uuid.New(),
		EventType: "test.event",
		Title:     "Group msg", Body: "Hello group",
	})
	require.NoError(t, err)
	require.Len(t, rt.calls, 1)
	assert.Equal(t, "group:"+groupID.String(), rt.calls[0].Channel)
}

func TestSendToGroup_FansOutToEachMember(t *testing.T) {
	repo := &stubNotificationRepo{}
	push := &stubPush{}
	appID := uuid.New()
	groupID := uuid.New()
	member1 := uuid.New()
	member2 := uuid.New()

	prefs := &stubPreferenceRepo{prefs: map[domain.Channel]bool{domain.ChannelPush: false}}
	groups := &stubGroups{members: []*domainports.GroupMember{
		{GroupID: groupID, UserID: member1, AppID: appID},
		{GroupID: groupID, UserID: member2, AppID: appID},
	}}
	svc := newSvc(repo, prefs, push, &stubEmail{}, &stubRealtime{}, groups, &stubDevices{})

	err := svc.SendToGroup(context.Background(), appports.SendToGroupCommand{
		GroupID: groupID, AppID: appID, OrgID: uuid.New(),
		EventType: "test.event",
		Title:     "Group msg", Body: "Hi all",
	})
	require.NoError(t, err)
	// One notification saved per member
	assert.Len(t, repo.saved, 2)
	userIDs := []uuid.UUID{repo.saved[0].UserID, repo.saved[1].UserID}
	assert.Contains(t, userIDs, member1)
	assert.Contains(t, userIDs, member2)
}

func TestSendToGroup_RealtimeFailureDoesNotAbort(t *testing.T) {
	rt := &stubRealtime{err: errors.New("centrifugo unreachable")}
	groupID := uuid.New()
	member := uuid.New()
	appID := uuid.New()

	prefs := &stubPreferenceRepo{prefs: map[domain.Channel]bool{domain.ChannelPush: false}}
	groups := &stubGroups{members: []*domainports.GroupMember{
		{GroupID: groupID, UserID: member, AppID: appID},
	}}
	repo := &stubNotificationRepo{}
	svc := newSvc(repo, prefs, &stubPush{}, &stubEmail{}, rt, groups, &stubDevices{})

	// Should not return an error even if Centrifugo is down
	err := svc.SendToGroup(context.Background(), appports.SendToGroupCommand{
		GroupID: groupID, AppID: appID, OrgID: uuid.New(),
		EventType: "test.event",
		Title:     "T", Body: "B",
	})
	require.NoError(t, err)
	// Member still gets their notification saved
	assert.Len(t, repo.saved, 1)
}

func TestMarkRead_AllowsOwner(t *testing.T) {
	userID := uuid.New()
	repo := &stubNotificationRepo{}
	n := domain.NewNotification(userID, uuid.New(), "T", "B")
	_ = repo.Save(context.Background(), n)

	svc := newSvc(repo, &stubPreferenceRepo{}, &stubPush{}, &stubEmail{}, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	err := svc.MarkRead(context.Background(), appports.MarkReadCommand{
		NotificationID: n.ID,
		UserID:         userID,
	})
	require.NoError(t, err)
	assert.Contains(t, repo.marked, n.ID)
}

func TestMarkRead_RejectsNonOwner(t *testing.T) {
	ownerID := uuid.New()
	repo := &stubNotificationRepo{}
	n := domain.NewNotification(ownerID, uuid.New(), "T", "B")
	_ = repo.Save(context.Background(), n)

	svc := newSvc(repo, &stubPreferenceRepo{}, &stubPush{}, &stubEmail{}, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	err := svc.MarkRead(context.Background(), appports.MarkReadCommand{
		NotificationID: n.ID,
		UserID:         uuid.New(), // different user
	})
	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestMarkRead_NotFound(t *testing.T) {
	svc := newSvc(&stubNotificationRepo{}, &stubPreferenceRepo{}, &stubPush{}, &stubEmail{}, &stubRealtime{}, &stubGroups{}, &stubDevices{})

	err := svc.MarkRead(context.Background(), appports.MarkReadCommand{
		NotificationID: uuid.New(),
		UserID:         uuid.New(),
	})
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}
