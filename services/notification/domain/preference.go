package domain

import "github.com/google/uuid"

// Channel is the delivery channel for a notification.
type Channel string

const (
	ChannelPush  Channel = "push"
	ChannelEmail Channel = "email"
	ChannelInApp Channel = "in_app"
)

// NotificationPreference controls whether a user receives notifications on a
// given channel for a given event type. EventType "*" acts as a default for
// all event types not explicitly configured.
type NotificationPreference struct {
	UserID    uuid.UUID
	AppID     uuid.UUID
	Channel   Channel
	EventType string // e.g. "org.group.member_added" or "*"
	Enabled   bool
}

// DeviceToken is a lightweight reference to a push token stored by the
// identity service. The notification service never writes these — it only
// reads them via DeviceQueryPort.
type DeviceToken struct {
	UserID    uuid.UUID
	AppID     uuid.UUID
	TokenType string // "fcm" | "apns"
	Token     string
	Platform  string // "android" | "ios" | "web"
}

// GroupMember is a lightweight reference to a group membership stored by
// the org service. Read via GroupQueryPort.
type GroupMember struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
	AppID   uuid.UUID
}
