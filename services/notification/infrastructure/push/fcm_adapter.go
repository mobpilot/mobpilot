package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// FCMAdapter sends push notifications via Firebase Cloud Messaging v1 HTTP API.
// It requires a pre-fetched OAuth2 access token (rotate via a token source).
type FCMAdapter struct {
	projectID   string
	tokenSource TokenSource
	client      *http.Client
}

// TokenSource returns a valid FCM access token. Implementations should cache
// and refresh tokens (e.g. using golang.org/x/oauth2/google).
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

func NewFCMAdapter(projectID string, tokenSource TokenSource) *FCMAdapter {
	return &FCMAdapter{
		projectID:   projectID,
		tokenSource: tokenSource,
		client:      &http.Client{},
	}
}

func (a *FCMAdapter) Send(ctx context.Context, msg domainports.PushMessage) error {
	token, err := a.tokenSource.Token(ctx)
	if err != nil {
		return fmt.Errorf("FCMAdapter: get token: %w", err)
	}

	payload := map[string]any{
		"message": map[string]any{
			"token": msg.Token,
			"notification": map[string]string{
				"title": msg.Title,
				"body":  msg.Body,
			},
			"data": msg.Data,
		},
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", a.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("FCMAdapter: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("FCMAdapter: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FCMAdapter: unexpected status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}
