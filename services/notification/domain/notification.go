package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryStatus tracks where a notification is in its lifecycle.
type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryFailed    DeliveryStatus = "failed"
)

// Notification is the core aggregate — one record per recipient per event.
type Notification struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AppID          uuid.UUID
	GroupID        *uuid.UUID
	OrgID          *uuid.UUID
	Title          string
	Body           string
	Data           map[string]any
	DeliveryStatus DeliveryStatus
	ReadAt         *time.Time
	CreatedAt      time.Time
}

func NewNotification(userID, appID uuid.UUID, title, body string) *Notification {
	return &Notification{
		ID:             uuid.New(),
		UserID:         userID,
		AppID:          appID,
		Title:          title,
		Body:           body,
		Data:           map[string]any{},
		DeliveryStatus: DeliveryPending,
		CreatedAt:      time.Now().UTC(),
	}
}

func (n *Notification) MarkDelivered() { n.DeliveryStatus = DeliveryDelivered }
func (n *Notification) MarkFailed()    { n.DeliveryStatus = DeliveryFailed }

func (n *Notification) MarkRead() {
	if n.ReadAt == nil {
		now := time.Now().UTC()
		n.ReadAt = &now
	}
}
