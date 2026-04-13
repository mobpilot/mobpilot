package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/org/application"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
)

// ─── Mock invitation repo ───────────────────────────────────────────────────

type mockInvitationRepo struct {
	invitations map[uuid.UUID]*domain.Invitation
}

func newMockInvitationRepo() *mockInvitationRepo {
	return &mockInvitationRepo{invitations: make(map[uuid.UUID]*domain.Invitation)}
}

func (m *mockInvitationRepo) Create(ctx context.Context, inv *domain.Invitation) error {
	m.invitations[inv.ID] = inv
	return nil
}

func (m *mockInvitationRepo) FindByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	for _, inv := range m.invitations {
		if inv.Token == token {
			return inv, nil
		}
	}
	return nil, domain.ErrInviteNotFound
}

func (m *mockInvitationRepo) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok {
		return domain.ErrInviteNotFound
	}
	now := time.Now().UTC()
	inv.AcceptedAt = &now
	return nil
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestMemberService_InviteMember(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	ctx := context.Background()
	orgID := uuid.New()
	inviterID := uuid.New()

	inv, err := svc.InviteMember(ctx, appports.InviteMemberCommand{
		OrgID:     orgID,
		Email:     "alice@example.com",
		Role:      domain.RoleMember,
		InviterID: inviterID,
	})
	require.NoError(t, err)

	assert.Equal(t, orgID, inv.OrgID)
	assert.Equal(t, "alice@example.com", inv.Email)
	assert.Equal(t, domain.RoleMember, inv.Role)
	assert.NotEmpty(t, inv.Token)
	assert.False(t, inv.ExpiresAt.IsZero())

	// Event should be published.
	require.Len(t, publisher.published, 1)
	assert.Equal(t, "org.member.invited", publisher.published[0].EventType())
}

func TestMemberService_InviteMember_InvalidEmail(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	_, err := svc.InviteMember(context.Background(), appports.InviteMemberCommand{
		OrgID:     uuid.New(),
		Email:     "",
		Role:      domain.RoleMember,
		InviterID: uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, domain.IsInvalidInput(err))
}

func TestMemberService_InviteMember_InvalidRole(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	_, err := svc.InviteMember(context.Background(), appports.InviteMemberCommand{
		OrgID:     uuid.New(),
		Email:     "alice@example.com",
		Role:      domain.MemberRole("superadmin"),
		InviterID: uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, domain.IsInvalidInput(err))
}

func TestMemberService_AcceptInvitation(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	ctx := context.Background()
	orgID := uuid.New()

	// Create an invitation.
	inv, err := svc.InviteMember(ctx, appports.InviteMemberCommand{
		OrgID:     orgID,
		Email:     "bob@example.com",
		Role:      domain.RoleAdmin,
		InviterID: uuid.New(),
	})
	require.NoError(t, err)

	// Accept it.
	userID := uuid.New()
	member, err := svc.AcceptInvitation(ctx, appports.AcceptInvitationCommand{
		Token:  inv.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	assert.Equal(t, orgID, member.OrgID)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, domain.RoleAdmin, member.Role)

	// MemberJoined event should be published (invite + accept = 2 events).
	require.Len(t, publisher.published, 2)
	assert.Equal(t, "org.member.joined", publisher.published[1].EventType())
}

func TestMemberService_AcceptInvitation_Expired(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	ctx := context.Background()

	// Manually add an expired invitation.
	expiredInv := &domain.Invitation{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "expired@example.com",
		Role:      domain.RoleMember,
		Token:     "expired-token",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	invRepo.invitations[expiredInv.ID] = expiredInv

	_, err := svc.AcceptInvitation(ctx, appports.AcceptInvitationCommand{
		Token:  "expired-token",
		UserID: uuid.New(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInviteExpired)
}

func TestMemberService_AcceptInvitation_NotFound(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	_, err := svc.AcceptInvitation(context.Background(), appports.AcceptInvitationCommand{
		Token:  "nonexistent",
		UserID: uuid.New(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInviteNotFound)
}

func TestMemberService_ListMembers(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	ctx := context.Background()
	orgID := uuid.New()

	// Add some members directly.
	for i := 0; i < 3; i++ {
		_ = memberRepo.Add(ctx, &domain.OrgMember{
			OrgID:    orgID,
			UserID:   uuid.New(),
			Role:     domain.RoleMember,
			JoinedAt: time.Now().UTC(),
		})
	}

	members, err := svc.ListMembers(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

func TestMemberService_RemoveMember(t *testing.T) {
	memberRepo := newMockMemberRepo()
	invRepo := newMockInvitationRepo()
	publisher := &mockPublisher{}
	svc := application.NewMemberService(memberRepo, invRepo, publisher)

	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	_ = memberRepo.Add(ctx, &domain.OrgMember{
		OrgID:    orgID,
		UserID:   userID,
		Role:     domain.RoleMember,
		JoinedAt: time.Now().UTC(),
	})

	err := svc.RemoveMember(ctx, appports.RemoveMemberCommand{
		OrgID:    orgID,
		UserID:   userID,
		CallerID: uuid.New(),
	})
	require.NoError(t, err)

	// Member should be gone.
	members, _ := svc.ListMembers(ctx, orgID)
	assert.Empty(t, members)
}
