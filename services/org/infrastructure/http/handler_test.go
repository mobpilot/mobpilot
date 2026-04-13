package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
	orghttp "github.com/mobpilot/mobpilot/services/org/infrastructure/http"
)

// ─── Stub use cases ─────────────────────────────────────────────────────────

type stubOrgUseCase struct {
	createFunc func(ctx context.Context, cmd appports.CreateOrgCommand) (*domain.Organization, error)
	getFunc    func(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
}

func (s *stubOrgUseCase) CreateOrg(ctx context.Context, cmd appports.CreateOrgCommand) (*domain.Organization, error) {
	return s.createFunc(ctx, cmd)
}

func (s *stubOrgUseCase) GetOrg(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return s.getFunc(ctx, id)
}

type stubMemberUseCase struct{}

func (s *stubMemberUseCase) InviteMember(ctx context.Context, cmd appports.InviteMemberCommand) (*domain.Invitation, error) {
	return nil, nil
}
func (s *stubMemberUseCase) AcceptInvitation(ctx context.Context, cmd appports.AcceptInvitationCommand) (*domain.OrgMember, error) {
	return nil, nil
}
func (s *stubMemberUseCase) RemoveMember(ctx context.Context, cmd appports.RemoveMemberCommand) error {
	return nil
}
func (s *stubMemberUseCase) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgMember, error) {
	return nil, nil
}

type stubGroupUseCase struct{}

func (s *stubGroupUseCase) CreateGroup(ctx context.Context, cmd appports.CreateGroupCommand) (*domain.Group, error) {
	return nil, nil
}
func (s *stubGroupUseCase) AddGroupMember(ctx context.Context, cmd appports.AddGroupMemberCommand) error {
	return nil
}
func (s *stubGroupUseCase) RemoveGroupMember(ctx context.Context, cmd appports.RemoveGroupMemberCommand) error {
	return nil
}
func (s *stubGroupUseCase) ListGroups(ctx context.Context, orgID uuid.UUID) ([]*domain.Group, error) {
	return nil, nil
}
func (s *stubGroupUseCase) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*domain.GroupMember, error) {
	return nil, nil
}
func (s *stubGroupUseCase) IssueSubscriptionToken(ctx context.Context, cmd appports.IssueSubscriptionTokenCommand) (string, error) {
	return "", nil
}

type stubRoleUseCase struct{}

func (s *stubRoleUseCase) CreateRole(ctx context.Context, cmd appports.CreateRoleCommand) (*domain.Role, error) {
	return nil, nil
}
func (s *stubRoleUseCase) DeleteRole(ctx context.Context, cmd appports.DeleteRoleCommand) error {
	return nil
}
func (s *stubRoleUseCase) ListRoles(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	return nil, nil
}

// ─── Test helpers ────────────────────────────────────────────────────────────

func newTestRouter(orgUC appports.OrgUseCase) *chi.Mux {
	r := chi.NewRouter()
	h := orghttp.NewHandler(orgUC, &stubMemberUseCase{}, &stubGroupUseCase{}, &stubRoleUseCase{}, nil, nil)
	h.Mount(r)
	return r
}

func withAuthContext(r *http.Request, subject, appID string) *http.Request {
	claims := &httpmw.Claims{
		Subject: subject,
		AppID:   appID,
	}
	ctx := httpmw.WithClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestHandler_CreateOrg_Success(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()

	orgUC := &stubOrgUseCase{
		createFunc: func(ctx context.Context, cmd appports.CreateOrgCommand) (*domain.Organization, error) {
			org, err := domain.NewOrganization(cmd.Name, cmd.Slug, cmd.OwnerUserID)
			if err != nil {
				return nil, err
			}
			org.ID = orgID
			return org, nil
		},
	}

	r := newTestRouter(orgUC)
	body, _ := json.Marshal(map[string]string{"name": "Acme", "slug": "acme"})
	req := httptest.NewRequest(http.MethodPost, "/v1/org/organizations", bytes.NewReader(body))
	req = withAuthContext(req, ownerID.String(), uuid.New().String())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, orgID.String(), resp["id"])
	assert.Equal(t, "Acme", resp["name"])
	assert.Equal(t, "acme", resp["slug"])
}

func TestHandler_CreateOrg_Unauthenticated(t *testing.T) {
	orgUC := &stubOrgUseCase{}
	r := newTestRouter(orgUC)

	body, _ := json.Marshal(map[string]string{"name": "Acme", "slug": "acme"})
	req := httptest.NewRequest(http.MethodPost, "/v1/org/organizations", bytes.NewReader(body))
	// No auth context added.

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetOrg_NotFound(t *testing.T) {
	orgUC := &stubOrgUseCase{
		getFunc: func(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
			return nil, domain.ErrOrgNotFound
		},
	}

	r := newTestRouter(orgUC)
	orgID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/org/organizations/"+orgID.String(), nil)
	req = withAuthContext(req, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/problem+json")
}

func TestHandler_CreateOrg_SlugConflict(t *testing.T) {
	orgUC := &stubOrgUseCase{
		createFunc: func(ctx context.Context, cmd appports.CreateOrgCommand) (*domain.Organization, error) {
			return nil, domain.ErrSlugTaken
		},
	}

	r := newTestRouter(orgUC)
	body, _ := json.Marshal(map[string]string{"name": "Acme", "slug": "taken"})
	req := httptest.NewRequest(http.MethodPost, "/v1/org/organizations", bytes.NewReader(body))
	req = withAuthContext(req, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}
