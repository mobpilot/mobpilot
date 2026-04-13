package ports

import (
	"context"

	"github.com/mobpilot/mobpilot/services/notification/domain"
)

// PushMessage is a push notification payload.
type PushMessage struct {
	Token    string
	Platform string // "android" | "ios"
	Title    string
	Body     string
	Data     map[string]string
}

// PushPort sends push notifications to a device.
// Implementations dispatch to FCM or APNs based on token type.
type PushPort interface {
	Send(ctx context.Context, tokenType string, msg PushMessage) error
}

// DeviceQueryPort fetches push tokens for a user from the identity service.
type DeviceQueryPort interface {
	FindTokensByUser(ctx context.Context, userID, appID string) ([]*domain.DeviceToken, error)
}
