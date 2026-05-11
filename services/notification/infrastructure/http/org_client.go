package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// OrgClient fetches group membership from the org service HTTP API.
// It implements domain/ports.GroupQueryPort.
type OrgClient struct {
	baseURL string // e.g. "http://org:8081"
	apiKey  string // internal service-to-service API key
	client  *http.Client
}

var _ domainports.GroupQueryPort = (*OrgClient)(nil)

func NewOrgClient(baseURL, apiKey string) *OrgClient {
	return &OrgClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type groupMembersResponse struct {
	Items []struct {
		GroupID string `json:"group_id"`
		UserID  string `json:"user_id"`
		AppID   string `json:"app_id"`
	} `json:"items"`
}

func (c *OrgClient) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*domainports.GroupMember, error) {
	url := fmt.Sprintf("%s/internal/groups/%s/members", c.baseURL, groupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("OrgClient.ListGroupMembers: create request: %w", err)
	}
	req.Header.Set("X-Internal-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OrgClient.ListGroupMembers: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OrgClient.ListGroupMembers: status %d", resp.StatusCode)
	}

	var body groupMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("OrgClient.ListGroupMembers: decode: %w", err)
	}

	members := make([]*domainports.GroupMember, 0, len(body.Items))
	for _, item := range body.Items {
		gid, err := uuid.Parse(item.GroupID)
		if err != nil {
			continue
		}
		uid, err := uuid.Parse(item.UserID)
		if err != nil {
			continue
		}
		aid, err := uuid.Parse(item.AppID)
		if err != nil {
			continue
		}
		members = append(members, &domainports.GroupMember{
			GroupID: gid,
			UserID:  uid,
			AppID:   aid,
		})
	}
	return members, nil
}
