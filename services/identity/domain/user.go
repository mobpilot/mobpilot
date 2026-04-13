package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is the aggregate root for a user profile in the context of one app.
// The underlying Kratos identity is identified by ID (== Kratos identity ID).
type User struct {
	ID          uuid.UUID
	AppID       uuid.UUID
	DisplayName string
	AvatarURL   string
	Bio         string
	Location    string
	WebsiteURL  string
	Settings    map[string]any // app-specific settings backup (JSONB)
	CreatedAt   time.Time
	UpdatedAt   time.Time

	events []DomainEvent
}

// NewUser creates a User aggregate and raises UserCreated.
func NewUser(kratosID, appID uuid.UUID, displayName string) (*User, error) {
	if kratosID == uuid.Nil {
		return nil, ErrInvalidInput("kratosID must not be nil")
	}
	if appID == uuid.Nil {
		return nil, ErrInvalidInput("appID must not be nil")
	}
	now := time.Now().UTC()
	u := &User{
		ID:          kratosID,
		AppID:       appID,
		DisplayName: displayName,
		Settings:    make(map[string]any),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	u.raise(UserCreatedEvent{UserID: u.ID, AppID: u.AppID, EventTimeAt: now})
	return u, nil
}

// UpdateProfile applies non-zero fields from the patch to the user.
func (u *User) UpdateProfile(patch ProfilePatch) {
	if patch.DisplayName != nil {
		u.DisplayName = *patch.DisplayName
	}
	if patch.AvatarURL != nil {
		u.AvatarURL = *patch.AvatarURL
	}
	if patch.Bio != nil {
		u.Bio = *patch.Bio
	}
	if patch.Location != nil {
		u.Location = *patch.Location
	}
	if patch.WebsiteURL != nil {
		u.WebsiteURL = *patch.WebsiteURL
	}
	u.UpdatedAt = time.Now().UTC()
}

// ProfilePatch carries optional fields for a partial update.
type ProfilePatch struct {
	DisplayName *string
	AvatarURL   *string
	Bio         *string
	Location    *string
	WebsiteURL  *string
}

// PopEvents drains accumulated domain events. Called by the application layer
// after successfully persisting the aggregate.
func (u *User) PopEvents() []DomainEvent {
	evts := u.events
	u.events = nil
	return evts
}

func (u *User) raise(e DomainEvent) {
	u.events = append(u.events, e)
}
