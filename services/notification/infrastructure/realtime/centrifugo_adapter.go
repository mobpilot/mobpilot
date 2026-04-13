package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// CentrifugoAdapter publishes messages to Centrifugo channels via the HTTP API.
// It implements domain/ports.RealtimePort.
type CentrifugoAdapter struct {
	apiURL string // e.g. "http://centrifugo:8080/api"
	apiKey string
	client *http.Client
}

var _ domainports.RealtimePort = (*CentrifugoAdapter)(nil)

func NewCentrifugoAdapter(apiURL, apiKey string) *CentrifugoAdapter {
	return &CentrifugoAdapter{
		apiURL: apiURL,
		apiKey: apiKey,
		client: &http.Client{},
	}
}

type centrifugoPublishRequest struct {
	Channel string         `json:"channel"`
	Data    map[string]any `json:"data"`
}

func (a *CentrifugoAdapter) Publish(ctx context.Context, msg domainports.RealtimeMessage) error {
	payload, err := json.Marshal(centrifugoPublishRequest{
		Channel: msg.Channel,
		Data:    msg.Data,
	})
	if err != nil {
		return fmt.Errorf("CentrifugoAdapter.Publish: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL+"/publish", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("CentrifugoAdapter.Publish: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("CentrifugoAdapter.Publish: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CentrifugoAdapter.Publish: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// NoopRealtimePort discards all publish calls. Used in tests or when
// Centrifugo is not configured.
type NoopRealtimePort struct{}

var _ domainports.RealtimePort = (*NoopRealtimePort)(nil)

func (*NoopRealtimePort) Publish(_ context.Context, _ domainports.RealtimeMessage) error { return nil }
