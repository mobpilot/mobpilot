package domain

import (
	"time"

	"github.com/google/uuid"
)

// Organization is the aggregate root for a tenant organization.
type Organization struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	OwnerUserID uuid.UUID
	Plan        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	SuspendedAt *time.Time

	events []Event
}

// NewOrganization creates an Organization aggregate and raises OrgCreatedEvent.
func NewOrganization(name, slug string, ownerUserID uuid.UUID) (*Organization, error) {
	if name == "" {
		return nil, ErrInvalidInput("name must not be empty")
	}
	if slug == "" {
		return nil, ErrInvalidInput("slug must not be empty")
	}
	if ownerUserID == uuid.Nil {
		return nil, ErrInvalidInput("ownerUserID must not be nil")
	}
	now := time.Now().UTC()
	org := &Organization{
		ID:          uuid.New(),
		Name:        name,
		Slug:        slug,
		OwnerUserID: ownerUserID,
		Plan:        "free",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	org.raise(OrgCreatedEvent{
		OrgID:       org.ID,
		OwnerUserID: ownerUserID,
		Slug:        slug,
		At: now,
	})
	return org, nil
}

// UpdateOrg applies non-zero fields from the patch.
type OrgPatch struct {
	Name *string
	Plan *string
}

func (o *Organization) Update(patch OrgPatch) {
	if patch.Name != nil {
		o.Name = *patch.Name
	}
	if patch.Plan != nil {
		o.Plan = *patch.Plan
	}
	o.UpdatedAt = time.Now().UTC()
}

// PopEvents drains accumulated domain events.
func (o *Organization) PopEvents() []Event {
	evts := o.events
	o.events = nil
	return evts
}

func (o *Organization) raise(e Event) {
	o.events = append(o.events, e)
}
