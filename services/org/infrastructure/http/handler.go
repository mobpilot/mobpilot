package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
	appports "github.com/mobpilot/mobpilot/services/org/application/ports"
	"github.com/mobpilot/mobpilot/services/org/domain"
)

// Handler wires the org HTTP API.
type Handler struct {
	orgs         appports.OrgUseCase
	members      appports.MemberUseCase
	groups       appports.GroupUseCase
	roles        appports.RoleUseCase
	tokenIssuer  *CentrifugoTokenIssuer
	proxyHandler *centrifugoProxyHandler
}

func NewHandler(
	orgs appports.OrgUseCase,
	members appports.MemberUseCase,
	groups appports.GroupUseCase,
	roles appports.RoleUseCase,
	tokenIssuer *CentrifugoTokenIssuer,
	proxyHandler *centrifugoProxyHandler,
) *Handler {
	return &Handler{
		orgs:         orgs,
		members:      members,
		groups:       groups,
		roles:        roles,
		tokenIssuer:  tokenIssuer,
		proxyHandler: proxyHandler,
	}
}

// Mount registers all routes on the given chi router (requires Bearer token auth).
func (h *Handler) Mount(r chi.Router) {
	r.Post("/v1/org/organizations", h.createOrg)
	r.Get("/v1/org/organizations/{orgID}", h.getOrg)

	r.Get("/v1/org/organizations/{orgID}/members", h.listMembers)
	r.Post("/v1/org/organizations/{orgID}/members/invite", h.inviteMember)
	r.Post("/v1/org/invitations/accept", h.acceptInvitation)
	r.Delete("/v1/org/organizations/{orgID}/members/{userID}", h.removeMember)

	r.Post("/v1/org/organizations/{orgID}/groups", h.createGroup)
	r.Get("/v1/org/organizations/{orgID}/groups", h.listGroups)
	r.Post("/v1/org/groups/{groupID}/members", h.addGroupMember)
	r.Get("/v1/org/groups/{groupID}/members", h.listGroupMembers)
	r.Delete("/v1/org/groups/{groupID}/members/{userID}", h.removeGroupMember)
	r.Get("/v1/org/groups/{groupID}/subscription-token", h.groupSubscriptionToken)

	r.Get("/v1/org/centrifugo/token", h.centrifugoConnectionToken)

	r.Post("/v1/org/organizations/{orgID}/roles", h.createRole)
	r.Get("/v1/org/organizations/{orgID}/roles", h.listRoles)
	r.Delete("/v1/org/roles/{roleID}", h.deleteRole)
}

// MountInternal registers server-to-server routes that are NOT behind Hydra
// Bearer auth. The Centrifugo subscribe proxy verifies a separate shared
// secret header; group member lookup is protected by the X-Internal-Api-Key
// header (same mechanism used by the identity service).
func (h *Handler) MountInternal(r chi.Router, internalAPIKey string) {
	if h.proxyHandler != nil {
		r.Post("/internal/centrifugo/subscribe", h.proxyHandler.ServeHTTP)
	}
	// Guard all other internal routes with the shared API key.
	r.Group(func(r chi.Router) {
		if internalAPIKey != "" {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Header.Get("X-Internal-Api-Key") != internalAPIKey {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					next.ServeHTTP(w, req)
				})
			})
		}
		r.Get("/internal/groups/{groupID}/members", h.internalListGroupMembers)
	})
}

func (h *Handler) internalListGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid groupID")
		return
	}
	members, err := h.groups.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": membersToResponse(members)})
}

// ─── Organizations ───────────────────────────────────────────────────────────

func (h *Handler) createOrg(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := h.orgs.CreateOrg(r.Context(), appports.CreateOrgCommand{
		Name:        req.Name,
		Slug:        req.Slug,
		OwnerUserID: mustParseUUID(claims.Subject),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, orgToResponse(org))
}

func (h *Handler) getOrg(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}

	org, err := h.orgs.GetOrg(r.Context(), orgID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orgToResponse(org))
}

// ─── Members ─────────────────────────────────────────────────────────────────

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}
	members, err := h.members.ListMembers(r.Context(), orgID)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]memberResponse, len(members))
	for i, m := range members {
		items[i] = memberToResponse(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) inviteMember(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inv, err := h.members.InviteMember(r.Context(), appports.InviteMemberCommand{
		OrgID:     orgID,
		Email:     req.Email,
		Role:      domain.MemberRole(req.Role),
		InviterID: mustParseUUID(claims.Subject),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": inv.ID.String(), "token": inv.Token})
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req acceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	member, err := h.members.AcceptInvitation(r.Context(), appports.AcceptInvitationCommand{
		Token:  req.Token,
		UserID: mustParseUUID(claims.Subject),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memberToResponse(member))
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid userID")
		return
	}

	if err := h.members.RemoveMember(r.Context(), appports.RemoveMemberCommand{
		OrgID:    orgID,
		UserID:   userID,
		CallerID: mustParseUUID(claims.Subject),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Groups ──────────────────────────────────────────────────────────────────

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	group, err := h.groups.CreateGroup(r.Context(), appports.CreateGroupCommand{
		OrgID:       orgID,
		AppID:       mustParseUUID(claims.AppID),
		Name:        req.Name,
		Description: req.Description,
		Ephemeral:   req.Ephemeral,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   mustParseUUID(claims.Subject),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, groupToResponse(group))
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}
	groups, err := h.groups.ListGroups(r.Context(), orgID)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]groupResponse, len(groups))
	for i, g := range groups {
		items[i] = groupToResponse(g)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) addGroupMember(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid groupID")
		return
	}

	var req addGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if err := h.groups.AddGroupMember(r.Context(), appports.AddGroupMemberCommand{
		GroupID: groupID,
		UserID:  userID,
		AppID:   mustParseUUID(claims.AppID),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listGroupMembers(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid groupID")
		return
	}
	members, err := h.groups.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": membersToResponse(members)})
}

func (h *Handler) removeGroupMember(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid groupID")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid userID")
		return
	}

	if err := h.groups.RemoveGroupMember(r.Context(), appports.RemoveGroupMemberCommand{
		GroupID: groupID,
		UserID:  userID,
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) groupSubscriptionToken(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid groupID")
		return
	}
	token, err := h.groups.IssueSubscriptionToken(r.Context(), appports.IssueSubscriptionTokenCommand{
		GroupID: groupID,
		UserID:  mustParseUUID(claims.Subject),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// centrifugoConnectionToken issues a short-lived Centrifugo connection JWT.
// Clients use this to establish a WebSocket or SSE connection to Centrifugo.
func (h *Handler) centrifugoConnectionToken(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if h.tokenIssuer == nil {
		writeProblem(w, http.StatusServiceUnavailable, "Centrifugo not configured — set CENTRIFUGO_TOKEN_SECRET")
		return
	}
	userID := mustParseUUID(claims.Subject)
	token, err := h.tokenIssuer.IssueConnectionToken(userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "token signing failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ─── Roles ───────────────────────────────────────────────────────────────────

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}

	var req createRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.roles.CreateRole(r.Context(), appports.CreateRoleCommand{
		OrgID:       orgID,
		Name:        req.Name,
		Permissions: req.Permissions,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, roleToResponse(role))
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid orgID")
		return
	}
	roles, err := h.roles.ListRoles(r.Context(), orgID)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]roleResponse, len(roles))
	for i, role := range roles {
		items[i] = roleToResponse(role)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid roleID")
		return
	}

	if err := h.roles.DeleteRole(r.Context(), appports.DeleteRoleCommand{
		RoleID: roleID,
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Request / response types ────────────────────────────────────────────────

type createOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type inviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

type createGroupRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Ephemeral   bool       `json:"ephemeral"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type addGroupMemberRequest struct {
	UserID string `json:"user_id"`
}

type createRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type orgResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	OwnerUserID string  `json:"owner_user_id"`
	Plan        string  `json:"plan"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	SuspendedAt *string `json:"suspended_at,omitempty"`
}

func orgToResponse(o *domain.Organization) orgResponse {
	resp := orgResponse{
		ID:          o.ID.String(),
		Name:        o.Name,
		Slug:        o.Slug,
		OwnerUserID: o.OwnerUserID.String(),
		Plan:        o.Plan,
		CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   o.UpdatedAt.Format(time.RFC3339),
	}
	if o.SuspendedAt != nil {
		s := o.SuspendedAt.Format(time.RFC3339)
		resp.SuspendedAt = &s
	}
	return resp
}

type memberResponse struct {
	OrgID    string `json:"org_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

func memberToResponse(m *domain.OrgMember) memberResponse {
	return memberResponse{
		OrgID:    m.OrgID.String(),
		UserID:   m.UserID.String(),
		Role:     string(m.Role),
		JoinedAt: m.JoinedAt.Format(time.RFC3339),
	}
}

type groupResponse struct {
	ID          string  `json:"id"`
	OrgID       string  `json:"org_id"`
	AppID       string  `json:"app_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Ephemeral   bool    `json:"ephemeral"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func groupToResponse(g *domain.Group) groupResponse {
	resp := groupResponse{
		ID:          g.ID.String(),
		OrgID:       g.OrgID.String(),
		AppID:       g.AppID.String(),
		Name:        g.Name,
		Description: g.Description,
		Ephemeral:   g.Ephemeral,
		CreatedAt:   g.CreatedAt.Format(time.RFC3339),
	}
	if g.ExpiresAt != nil {
		s := g.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp
}

type groupMemberResponse struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	AppID   string `json:"app_id"`
}

func membersToResponse(members []*domain.GroupMember) []groupMemberResponse {
	items := make([]groupMemberResponse, len(members))
	for i, m := range members {
		items[i] = groupMemberResponse{
			GroupID: m.GroupID.String(),
			UserID:  m.UserID.String(),
			AppID:   m.AppID.String(),
		}
	}
	return items
}

type roleResponse struct {
	ID          string   `json:"id"`
	OrgID       string   `json:"org_id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
}

func roleToResponse(r *domain.Role) roleResponse {
	perms := r.Permissions
	if perms == nil {
		perms = []string{}
	}
	return roleResponse{
		ID:          r.ID.String(),
		OrgID:       r.OrgID.String(),
		Name:        r.Name,
		Permissions: perms,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   problemType(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrgNotFound),
		errors.Is(err, domain.ErrMemberNotFound),
		errors.Is(err, domain.ErrGroupNotFound),
		errors.Is(err, domain.ErrRoleNotFound),
		errors.Is(err, domain.ErrInviteNotFound):
		writeProblem(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrNotAuthorized):
		writeProblem(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrSlugTaken):
		writeProblem(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInviteExpired):
		writeProblem(w, http.StatusGone, err.Error())
	case domain.IsInvalidInput(err):
		writeProblem(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "internal server error")
	}
}

func problemType(status int) string {
	switch status {
	case http.StatusNotFound:
		return "urn:mobpilot:org:not-found"
	case http.StatusUnauthorized:
		return "urn:mobpilot:auth:unauthorized"
	case http.StatusForbidden:
		return "urn:mobpilot:auth:forbidden"
	case http.StatusConflict:
		return "urn:mobpilot:org:conflict"
	case http.StatusGone:
		return "urn:mobpilot:org:expired"
	case http.StatusUnprocessableEntity:
		return "urn:mobpilot:org:invalid-input"
	default:
		return "urn:mobpilot:internal-error"
	}
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
