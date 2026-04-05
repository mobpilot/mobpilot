package domain

import (
	"time"

	"github.com/google/uuid"
)

// Platform is the mobile platform for a device.
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWeb     Platform = "web"
)

// TokenType distinguishes between push token providers.
type TokenType string

const (
	TokenTypeFCM  TokenType = "fcm"
	TokenTypeAPNs TokenType = "apns"
)

// DeviceToken represents a push notification token registered by a device.
type DeviceToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AppID     uuid.UUID
	Platform  Platform
	TokenType TokenType
	Token     string
	DeviceID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDeviceToken validates and creates a DeviceToken.
func NewDeviceToken(userID, appID uuid.UUID, platform Platform, tokenType TokenType, token, deviceID string) (*DeviceToken, error) {
	if token == "" {
		return nil, ErrInvalidInput("token must not be empty")
	}
	if deviceID == "" {
		return nil, ErrInvalidInput("deviceID must not be empty")
	}
	now := time.Now().UTC()
	return &DeviceToken{
		ID:        uuid.New(),
		UserID:    userID,
		AppID:     appID,
		Platform:  platform,
		TokenType: tokenType,
		Token:     token,
		DeviceID:  deviceID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
