package ports

import (
	"context"

	"github.com/google/uuid"
)

// RealtimeMessage is the payload published to a Centrifugo channel.
type RealtimeMessage struct {
	Channel string
	Data    map[string]any
}

// RealtimePort publishes messages to connected WebSocket/SSE clients via
// Centrifugo. Implements in-app real-time delivery.
type RealtimePort interface {
	Publish(ctx context.Context, msg RealtimeMessage) error
}

// GroupQueryPort fetches group membership from the org service.
type GroupQueryPort interface {
	ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*GroupMember, error)
}

// GroupMember is a thin data transfer type (not the domain aggregate).
type GroupMember struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	AppID   uuid.UUID
}
