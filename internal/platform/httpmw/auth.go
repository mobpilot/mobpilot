package httpmw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const (
	ctxKeySubject contextKey = "subject"
	ctxKeyAppID   contextKey = "app_id"
	ctxKeyScopes  contextKey = "scopes"
)

// Claims holds the validated JWT claims extracted by the auth middleware.
type Claims struct {
	Subject string
	AppID   string
	Scopes  []string
}

// ClaimsFromContext retrieves the validated claims from the request context.
// Returns nil if the request was not authenticated.
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(ctxKeySubject).(*Claims)
	return c
}

// WithClaims injects claims into a context. Exported for testing.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, ctxKeySubject, c)
}

// introspectResponse mirrors the Hydra token introspection response.
type introspectResponse struct {
	Active    bool   `json:"active"`
	Subject   string `json:"sub"`
	Scope     string `json:"scope"`
	Ext       struct {
		AppID string `json:"mobpilot_app_id"`
	} `json:"ext"`
}

// introspectCache caches valid introspection results briefly to reduce Hydra load.
type introspectCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	claims    *Claims
	expiresAt time.Time
}

var tokenCache = &introspectCache{entries: make(map[string]cacheEntry)}

func (c *introspectCache) get(token string) (*Claims, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[token]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, token)
		return nil, false
	}
	return e.claims, true
}

func (c *introspectCache) set(token string, claims *Claims, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[token] = cacheEntry{claims: claims, expiresAt: time.Now().Add(ttl)}
}

// BearerAuth returns an HTTP middleware that validates OAuth2 Bearer tokens
// via Hydra's token introspection endpoint.
// hydraAdminURL: e.g. "http://hydra:4445"
func BearerAuth(hydraAdminURL string) func(http.Handler) http.Handler {
	introspectURL := strings.TrimRight(hydraAdminURL, "/") + "/admin/oauth2/introspect"
	client := &http.Client{Timeout: 5 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeProblem(w, http.StatusUnauthorized, "missing or invalid Authorization header")
				return
			}

			claims, ok := tokenCache.get(token)
			if !ok {
				var err error
				claims, err = introspect(r.Context(), client, introspectURL, token)
				if err != nil {
					writeProblem(w, http.StatusUnauthorized, "token validation failed")
					return
				}
				tokenCache.set(token, claims, 30*time.Second)
			}

			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func introspect(ctx context.Context, client *http.Client, url, token string) (*Claims, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		strings.NewReader("token="+token))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspect request: %w", err)
	}
	defer resp.Body.Close()

	var ir introspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return nil, fmt.Errorf("introspect decode: %w", err)
	}
	if !ir.Active {
		return nil, fmt.Errorf("token inactive")
	}

	scopes := strings.Fields(ir.Scope)
	return &Claims{
		Subject: ir.Subject,
		AppID:   ir.Ext.AppID,
		Scopes:  scopes,
	}, nil
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "urn:mobpilot:auth:unauthorized",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
