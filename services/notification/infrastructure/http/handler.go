package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
	appports "github.com/mobpilot/mobpilot/services/notification/application/ports"
	"github.com/mobpilot/mobpilot/services/notification/domain"
)

// Handler wires the notification HTTP API.
type Handler struct {
	notifications appports.NotificationUseCase
	preferences   appports.PreferenceUseCase
}

func NewHandler(notifications appports.NotificationUseCase, preferences appports.PreferenceUseCase) *Handler {
	return &Handler{notifications: notifications, preferences: preferences}
}

// Mount registers all routes on the given chi router.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/v1/notifications", h.listNotifications)
	r.Post("/v1/notifications/{notificationID}/read", h.markRead)
	r.Get("/v1/notifications/preferences", h.getPreferences)
	r.Put("/v1/notifications/preferences", h.upsertPreference)
}

// ─── Notifications ────────────────────────────────────────────────────────────

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	userID := mustParseUUID(claims.Subject)
	appID := mustParseUUID(claims.AppID)

	ns, err := h.notifications.ListForUser(r.Context(), userID, appID, limit, offset)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]notificationResponse, len(ns))
	for i, n := range ns {
		items[i] = notificationToResponse(n)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	nid, err := uuid.Parse(chi.URLParam(r, "notificationID"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid notificationID")
		return
	}

	if err := h.notifications.MarkRead(r.Context(), appports.MarkReadCommand{
		NotificationID: nid,
		UserID:         mustParseUUID(claims.Subject),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Preferences ──────────────────────────────────────────────────────────────

func (h *Handler) getPreferences(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	userID := mustParseUUID(claims.Subject)
	appID := mustParseUUID(claims.AppID)

	prefs, err := h.preferences.GetPreferences(r.Context(), userID, appID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal server error")
		return
	}
	items := make([]preferenceResponse, len(prefs))
	for i, p := range prefs {
		items[i] = preferenceResponse{
			Channel:   string(p.Channel),
			EventType: p.EventType,
			Enabled:   p.Enabled,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) upsertPreference(w http.ResponseWriter, r *http.Request) {
	claims := httpmw.ClaimsFromContext(r.Context())
	if claims == nil {
		writeProblem(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req upsertPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.preferences.UpsertPreference(r.Context(), appports.UpsertPreferenceCommand{
		UserID:    mustParseUUID(claims.Subject),
		AppID:     mustParseUUID(claims.AppID),
		Channel:   domain.Channel(req.Channel),
		EventType: req.EventType,
		Enabled:   req.Enabled,
	}); err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Request / response types ────────────────────────────────────────────────

type upsertPreferenceRequest struct {
	Channel   string `json:"channel"`
	EventType string `json:"event_type"`
	Enabled   bool   `json:"enabled"`
}

type notificationResponse struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	AppID          string         `json:"app_id"`
	GroupID        *string        `json:"group_id,omitempty"`
	OrgID          *string        `json:"org_id,omitempty"`
	Title          string         `json:"title"`
	Body           string         `json:"body"`
	Data           map[string]any `json:"data"`
	DeliveryStatus string         `json:"delivery_status"`
	ReadAt         *string        `json:"read_at,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

func notificationToResponse(n *domain.Notification) notificationResponse {
	resp := notificationResponse{
		ID:             n.ID.String(),
		UserID:         n.UserID.String(),
		AppID:          n.AppID.String(),
		Title:          n.Title,
		Body:           n.Body,
		Data:           n.Data,
		DeliveryStatus: string(n.DeliveryStatus),
		CreatedAt:      n.CreatedAt.Format(time.RFC3339),
	}
	if n.GroupID != nil {
		s := n.GroupID.String()
		resp.GroupID = &s
	}
	if n.OrgID != nil {
		s := n.OrgID.String()
		resp.OrgID = &s
	}
	if n.ReadAt != nil {
		s := n.ReadAt.Format(time.RFC3339)
		resp.ReadAt = &s
	}
	return resp
}

type preferenceResponse struct {
	Channel   string `json:"channel"`
	EventType string `json:"event_type"`
	Enabled   bool   `json:"enabled"`
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
		"type":   "urn:mobpilot:notification:error",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotificationNotFound):
		writeProblem(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrNotAuthorized):
		writeProblem(w, http.StatusForbidden, err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "internal server error")
	}
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
