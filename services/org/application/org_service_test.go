package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/org/application"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
)

// ─── Mock repositories ──────────────────────────────────────────────────────

type mockOrgRepo struct {
	orgs map[uuid.UUID]*domain.Organization
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{orgs: make(map[uuid.UUID]*domain.Organization)}
}

func (m *mockOrgRepo) Create(ctx context.Context, org *domain.Organization) error {
	for _, existing := range m.orgs {
		if existing.Slug == org.Slug {
			return domain.ErrSlugTaken
		}
	}
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return nil, domain.ErrOrgNotFound
	}
	return org, nil
}

func (m *mockOrgRepo) FindBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	for _, org := range m.orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, domain.ErrOrgNotFound
}

func (m *mockOrgRepo) Update(ctx context.Context, org *domain.Organization) error {
	m.orgs[org.ID] = org
	return nil
}

type mockMemberRepo struct {
	members map[string]*domain.OrgMember // key: orgID:userID
}

func newMockMemberRepo() *mockMemberRepo {
	return &mockMemberRepo{members: make(map[string]*domain.OrgMember)}
}

func (m *mockMemberRepo) Add(ctx context.Context, member *domain.OrgMember) error {
	key := member.OrgID.String() + ":" + member.UserID.String()
	if _, exists := m.members[key]; exists {
		return domain.ErrAlreadyExists
	}
	m.members[key] = member
	return nil
}

func (m *mockMemberRepo) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	key := orgID.String() + ":" + userID.String()
	delete(m.members, key)
	return nil
}

func (m *mockMemberRepo) FindByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error) {
	key := orgID.String() + ":" + userID.String()
	member, ok := m.members[key]
	if !ok {
		return nil, domain.ErrMemberNotFound
	}
	return member, nil
}

func (m *mockMemberRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error) {
	var result []*domain.OrgMember
	prefix := orgID.String() + ":"
	for key, member := range m.members {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, member)
		}
	}
	return result, nil
}

type mockPublisher struct {
	published []domain.Event
}

func (m *mockPublisher) Publish(ctx context.Context, events []domain.Event) error {
	m.published = append(m.published, events...)
	return nil
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestOrgService_CreateOrg(t *testing.T) {
	orgRepo := newMockOrgRepo()
	memberRepo := newMockMemberRepo()
	publisher := &mockPublisher{}
	svc := application.NewOrgService(orgRepo, memberRepo, publisher)

	ctx := context.Background()
	ownerID := uuid.New()

	org, err := svc.CreateOrg(ctx, appports.CreateOrgCommand{
		Name:        "Acme Inc",
		Slug:        "acme-inc",
		OwnerUserID: ownerID,
	})
	require.NoError(t, err)

	assert.Equal(t, "Acme Inc", org.Name)
	assert.Equal(t, "acme-inc", org.Slug)
	assert.Equal(t, ownerID, org.OwnerUserID)
	assert.Equal(t, "free", org.Plan)

	// Owner should be added as a member.
	assert.Len(t, memberRepo.members, 1)
	for _, m := range memberRepo.members {
		assert.Equal(t, domain.RoleOwner, m.Role)
		assert.Equal(t, ownerID, m.UserID)
	}

	// Event should be published.
	require.Len(t, publisher.published, 1)
	assert.Equal(t, "org.organization.created", publisher.published[0].EventType())
}

func TestOrgService_CreateOrg_DuplicateSlug(t *testing.T) {
	orgRepo := newMockOrgRepo()
	memberRepo := newMockMemberRepo()
	publisher := &mockPublisher{}
	svc := application.NewOrgService(orgRepo, memberRepo, publisher)

	ctx := context.Background()
	ownerID := uuid.New()

	_, err := svc.CreateOrg(ctx, appports.CreateOrgCommand{
		Name:        "Acme Inc",
		Slug:        "acme",
		OwnerUserID: ownerID,
	})
	require.NoError(t, err)

	_, err = svc.CreateOrg(ctx, appports.CreateOrgCommand{
		Name:        "Another Acme",
		Slug:        "acme",
		OwnerUserID: uuid.New(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSlugTaken)
}

func TestOrgService_CreateOrg_ValidationErrors(t *testing.T) {
	orgRepo := newMockOrgRepo()
	memberRepo := newMockMemberRepo()
	publisher := &mockPublisher{}
	svc := application.NewOrgService(orgRepo, memberRepo, publisher)

	ctx := context.Background()

	_, err := svc.CreateOrg(ctx, appports.CreateOrgCommand{
		Name:        "",
		Slug:        "acme",
		OwnerUserID: uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, domain.IsInvalidInput(err))
}

func TestOrgService_GetOrg(t *testing.T) {
	orgRepo := newMockOrgRepo()
	memberRepo := newMockMemberRepo()
	publisher := &mockPublisher{}
	svc := application.NewOrgService(orgRepo, memberRepo, publisher)

	ctx := context.Background()

	created, err := svc.CreateOrg(ctx, appports.CreateOrgCommand{
		Name:        "Acme",
		Slug:        "acme",
		OwnerUserID: uuid.New(),
	})
	require.NoError(t, err)

	found, err := svc.GetOrg(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Acme", found.Name)
}

func TestOrgService_GetOrg_NotFound(t *testing.T) {
	orgRepo := newMockOrgRepo()
	memberRepo := newMockMemberRepo()
	publisher := &mockPublisher{}
	svc := application.NewOrgService(orgRepo, memberRepo, publisher)

	_, err := svc.GetOrg(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrOrgNotFound)
}
