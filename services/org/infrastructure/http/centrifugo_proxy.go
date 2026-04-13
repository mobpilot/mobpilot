package http

import (
	"encoding/json"
	"net/http"
	"strings"

	domainports "github.com/mobpilot/mobpilot/services/org/domain/ports"
)

// centrifugoProxyRequest is the JSON body that Centrifugo sends to the
// subscribe proxy endpoint.
type centrifugoProxyRequest struct {
	UserID    string `json:"user"`
	Channel   string `json:"channel"`
	Transport string `json:"transport"`
}

// centrifugoProxyHandler handles POST /internal/centrifugo/subscribe.
// Centrifugo calls this for channels in the "group:" and "user:" namespaces
// (ephemeral-group channels use JWT token auth and never reach this endpoint).
//
// The endpoint is NOT behind Hydra auth — it is server-to-server, verified via
// the X-Centrifugo-Proxy-Secret header.
type centrifugoProxyHandler struct {
	authz        domainports.AuthzPort
	proxySecret  string
}

// NewCentrifugoProxyHandler creates the Centrifugo subscribe proxy handler.
func NewCentrifugoProxyHandler(authz domainports.AuthzPort, proxySecret string) *centrifugoProxyHandler {
	return &centrifugoProxyHandler{authz: authz, proxySecret: proxySecret}
}

func (h *centrifugoProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verify server-to-server secret.
	if r.Header.Get("X-Centrifugo-Proxy-Secret") != h.proxySecret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req centrifugoProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	allowed, err := h.checkSubscription(r, req)
	if err != nil {
		// On error, deny the subscription.
		writeCentrifugoError(w, 500, "internal error")
		return
	}
	if !allowed {
		writeCentrifugoError(w, 403, "not authorized")
		return
	}

	// Allow — return empty result object.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"result":{}}`))
}

func (h *centrifugoProxyHandler) checkSubscription(r *http.Request, req centrifugoProxyRequest) (bool, error) {
	channel := req.Channel
	userID := req.UserID

	switch {
	case strings.HasPrefix(channel, "group:"):
		groupID := strings.TrimPrefix(channel, "group:")
		return h.authz.Check(r.Context(), userID, "member", "Group:"+groupID)

	case strings.HasPrefix(channel, "user:"):
		// User personal channel: only the user themselves can subscribe.
		targetUserID := strings.TrimPrefix(channel, "user:")
		return userID == targetUserID, nil

	default:
		// Unknown channel namespace — deny.
		return false, nil
	}
}

// writeCentrifugoError writes a Centrifugo-compatible error response.
// Centrifugo requires HTTP 200 even for denied subscriptions; the error is
// conveyed in the JSON body.
func writeCentrifugoError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Centrifugo protocol: always 200
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
