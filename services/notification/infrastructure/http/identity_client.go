package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// IdentityClient fetches device tokens from the identity service HTTP API.
// It implements domain/ports.DeviceQueryPort.
type IdentityClient struct {
	baseURL string // e.g. "http://identity:8080"
	apiKey  string // internal service-to-service API key
	client  *http.Client
}

var _ domainports.DeviceQueryPort = (*IdentityClient)(nil)

func NewIdentityClient(baseURL, apiKey string) *IdentityClient {
	return &IdentityClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type deviceTokenResponse struct {
	Items []struct {
		TokenType string `json:"token_type"`
		Token     string `json:"token"`
		Platform  string `json:"platform"`
	} `json:"items"`
}

func (c *IdentityClient) FindTokensByUser(ctx context.Context, userID, appID string) ([]*domain.DeviceToken, error) {
	url := fmt.Sprintf("%s/internal/users/%s/device-tokens?app_id=%s", c.baseURL, userID, appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("IdentityClient.FindTokensByUser: create request: %w", err)
	}
	req.Header.Set("X-Internal-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IdentityClient.FindTokensByUser: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IdentityClient.FindTokensByUser: status %d", resp.StatusCode)
	}

	var body deviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("IdentityClient.FindTokensByUser: decode: %w", err)
	}

	tokens := make([]*domain.DeviceToken, len(body.Items))
	for i, item := range body.Items {
		tokens[i] = &domain.DeviceToken{
			TokenType: item.TokenType,
			Token:     item.Token,
			Platform:  item.Platform,
		}
	}
	return tokens, nil
}
