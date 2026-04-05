package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appports "github.com/mobpilot/mobpilot/services/identity/application/ports"
	"github.com/mobpilot/mobpilot/services/identity/domain"
	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
)

// Handler wires the identity HTTP API.
type Handler struct {
	users     appports.UserUseCase
	devices   appports.DeviceTokenUseCase
	artifacts appports.ArtifactUseCase
}

func NewHandler(users appports.UserUseCase, devices appports.DeviceTokenUseCase, artifacts appports.ArtifactUseCase) *Handler {
	return &Handler{users: users, devices: devices, artifacts: artifacts}
}

// Mount registers all routes on the given chi router.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/v1/identity/me", h.getMe)
	r.Patch("/v1/identity/me", h.updateMe)
	r.Get("/v1/identity/users/{userID}", h.getUser)

	r.Post("/v1/identity/devices", h.registerDevice)
	r.Delete("/v1/identity/devices/{deviceID}", h.unregisterDevice)

	r.Get("/v1/identity/me/artifacts", h.listArtifacts)
	r.Put("/v1/identity/me/artifacts/{key}", h.putArtifact)
	r.Get("/v1/identity/me/artifacts/{key}", h.getArtifact)
	r.Delete("/v1/identity/me/artifacts/{key}", h.deleteArtifact)
}

// ─── User profile ─────────────────────────────────────────────────────────────

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	userID := mustParseUUID(claims.Subject)
	appID := mustParseUUID(claims.AppID)

	user, err := h.users.GetMe(r.Context(), appports.GetMeCommand{UserID: userID, AppID: appID})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	userID := mustParseUUID(claims.Subject)
	appID := mustParseUUID(claims.AppID)

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.users.UpdateMe(r.Context(), appports.UpdateMeCommand{
		UserID: userID,
		AppID:  appID,
		Patch: domain.ProfilePatch{
			DisplayName: req.DisplayName,
			AvatarURL:   req.AvatarURL,
			Bio:         req.Bio,
			Location:    req.Location,
			WebsiteURL:  req.WebsiteURL,
		},
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid userID")
		return
	}
	appID := mustParseUUID(claims.AppID)

	user, err := h.users.GetUser(r.Context(), userID, appID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userToResponse(user))
}

// ─── Device tokens ────────────────────────────────────────────────────────────

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dt, err := h.devices.RegisterDevice(r.Context(), appports.RegisterDeviceCommand{
		UserID:    mustParseUUID(claims.Subject),
		AppID:     mustParseUUID(claims.AppID),
		Platform:  domain.Platform(req.Platform),
		TokenType: domain.TokenType(req.TokenType),
		Token:     req.Token,
		DeviceID:  req.DeviceID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": dt.ID.String()})
}

func (h *Handler) unregisterDevice(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	if err := h.devices.UnregisterDevice(r.Context(), appports.UnregisterDeviceCommand{
		AppID:    mustParseUUID(claims.AppID),
		DeviceID: deviceID,
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Artifacts ────────────────────────────────────────────────────────────────

func (h *Handler) listArtifacts(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	artifacts, err := h.artifacts.List(r.Context(), mustParseUUID(claims.Subject), mustParseUUID(claims.AppID))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]artifactResponse, len(artifacts))
	for i, a := range artifacts {
		items[i] = artifactToResponse(a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) putArtifact(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	key := chi.URLParam(r, "key")

	var value any
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	a, err := h.artifacts.Put(r.Context(), appports.PutArtifactCommand{
		UserID: mustParseUUID(claims.Subject),
		AppID:  mustParseUUID(claims.AppID),
		Key:    key,
		Value:  value,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, artifactToResponse(a))
}

func (h *Handler) getArtifact(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	a, err := h.artifacts.Get(r.Context(), appports.GetArtifactCommand{
		UserID: mustParseUUID(claims.Subject),
		AppID:  mustParseUUID(claims.AppID),
		Key:    chi.URLParam(r, "key"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, artifactToResponse(a))
}

func (h *Handler) deleteArtifact(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.artifacts.Delete(r.Context(), appports.DeleteArtifactCommand{
		UserID: mustParseUUID(claims.Subject),
		AppID:  mustParseUUID(claims.AppID),
		Key:    chi.URLParam(r, "key"),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Request / response types ─────────────────────────────────────────────────

type updateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
	Location    *string `json:"location"`
	WebsiteURL  *string `json:"website_url"`
}

type registerDeviceRequest struct {
	Platform  string `json:"platform"`
	TokenType string `json:"token_type"`
	Token     string `json:"token"`
	DeviceID  string `json:"device_id"`
}

type userResponse struct {
	ID          string         `json:"id"`
	AppID       string         `json:"app_id"`
	DisplayName string         `json:"display_name"`
	AvatarURL   string         `json:"avatar_url"`
	Bio         string         `json:"bio"`
	Location    string         `json:"location"`
	WebsiteURL  string         `json:"website_url"`
	Settings    map[string]any `json:"settings"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

func userToResponse(u *domain.User) userResponse {
	settings := u.Settings
	if settings == nil {
		settings = make(map[string]any)
	}
	return userResponse{
		ID:          u.ID.String(),
		AppID:       u.AppID.String(),
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		Location:    u.Location,
		WebsiteURL:  u.WebsiteURL,
		Settings:    settings,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type artifactResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Value     any    `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

func artifactToResponse(a *domain.Artifact) artifactResponse {
	return artifactResponse{
		ID:        a.ID.String(),
		Key:       a.Key,
		Value:     a.Value,
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

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
	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrArtifactNotFound),
		errors.Is(err, domain.ErrDeviceNotFound):
		writeProblem(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrNotAuthorized):
		writeProblem(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeProblem(w, http.StatusConflict, err.Error())
	case domain.IsInvalidInput(err):
		writeProblem(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "internal server error")
	}
}

func problemType(status int) string {
	switch status {
	case http.StatusNotFound:
		return "urn:mobpilot:identity:not-found"
	case http.StatusUnauthorized:
		return "urn:mobpilot:auth:unauthorized"
	case http.StatusForbidden:
		return "urn:mobpilot:auth:forbidden"
	case http.StatusConflict:
		return "urn:mobpilot:identity:conflict"
	case http.StatusUnprocessableEntity:
		return "urn:mobpilot:identity:invalid-input"
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
