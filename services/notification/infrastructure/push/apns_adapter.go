package push

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"

	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// APNsAdapter sends push notifications via Apple Push Notification service.
type APNsAdapter struct {
	client *apns2.Client
	topic  string // bundle ID / topic
}

// NewAPNsAdapter creates an APNs adapter using token-based auth (p8 key).
// keyID: APNs key ID (10-char string from Apple Developer portal).
// teamID: Apple Developer team ID.
// keyPEM: Contents of the .p8 private key file.
// topic: App bundle ID (e.g. "com.example.myapp").
// production: true for production APNs, false for sandbox.
func NewAPNsAdapter(keyID, teamID, topic string, keyPEM []byte, production bool) (*APNsAdapter, error) {
	authKey, err := token.AuthKeyFromBytes(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("APNsAdapter: parse auth key: %w", err)
	}
	t := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}
	client := apns2.NewTokenClient(t)
	if production {
		client = client.Production()
	} else {
		client = client.Development()
	}
	return &APNsAdapter{client: client, topic: topic}, nil
}

type apnsPayload struct {
	Aps  apnsAps        `json:"aps"`
	Data map[string]any `json:"data,omitempty"`
}

type apnsAps struct {
	Alert apnsAlert `json:"alert"`
	Sound string    `json:"sound,omitempty"`
}

type apnsAlert struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (a *APNsAdapter) Send(_ context.Context, msg domainports.PushMessage) error {
	data := map[string]any{}
	for k, v := range msg.Data {
		data[k] = v
	}

	payload := apnsPayload{
		Aps: apnsAps{
			Alert: apnsAlert{Title: msg.Title, Body: msg.Body},
			Sound: "default",
		},
		Data: data,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("APNsAdapter: marshal payload: %w", err)
	}

	n := &apns2.Notification{
		DeviceToken: msg.Token,
		Topic:       a.topic,
		Payload:     payloadBytes,
	}

	resp, err := a.client.Push(n)
	if err != nil {
		return fmt.Errorf("APNsAdapter: push: %w", err)
	}
	if !resp.Sent() {
		return fmt.Errorf("APNsAdapter: not sent: %s (%s)", resp.Reason, resp.ApnsID)
	}
	return nil
}
