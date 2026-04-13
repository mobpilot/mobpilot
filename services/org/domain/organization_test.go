package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/org/domain"
)

func TestNewOrganization(t *testing.T) {
	tests := []struct {
		name        string
		orgName     string
		slug        string
		ownerID     uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid organization",
			orgName: "Acme Inc",
			slug:    "acme-inc",
			ownerID: uuid.New(),
			wantErr: false,
		},
		{
			name:        "empty name",
			orgName:     "",
			slug:        "acme",
			ownerID:     uuid.New(),
			wantErr:     true,
			errContains: "name must not be empty",
		},
		{
			name:        "empty slug",
			orgName:     "Acme",
			slug:        "",
			ownerID:     uuid.New(),
			wantErr:     true,
			errContains: "slug must not be empty",
		},
		{
			name:        "nil owner",
			orgName:     "Acme",
			slug:        "acme",
			ownerID:     uuid.Nil,
			wantErr:     true,
			errContains: "ownerUserID must not be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, err := domain.NewOrganization(tt.orgName, tt.slug, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.True(t, domain.IsInvalidInput(err))
				assert.Nil(t, org)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.orgName, org.Name)
			assert.Equal(t, tt.slug, org.Slug)
			assert.Equal(t, tt.ownerID, org.OwnerUserID)
			assert.Equal(t, "free", org.Plan)
			assert.NotEqual(t, uuid.Nil, org.ID)
			assert.False(t, org.CreatedAt.IsZero())
		})
	}
}

func TestOrganization_PopEvents(t *testing.T) {
	ownerID := uuid.New()
	org, err := domain.NewOrganization("Test Org", "test-org", ownerID)
	require.NoError(t, err)

	events := org.PopEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "org.organization.created", events[0].EventType())

	// Second call should return empty.
	assert.Empty(t, org.PopEvents())
}

func TestOrganization_Update(t *testing.T) {
	org, err := domain.NewOrganization("Original", "original", uuid.New())
	require.NoError(t, err)

	originalUpdated := org.UpdatedAt
	newName := "Updated Name"
	newPlan := "pro"

	org.Update(domain.OrgPatch{
		Name: &newName,
		Plan: &newPlan,
	})

	assert.Equal(t, "Updated Name", org.Name)
	assert.Equal(t, "pro", org.Plan)
	assert.True(t, org.UpdatedAt.After(originalUpdated) || org.UpdatedAt.Equal(originalUpdated))
}

func TestOrganization_Update_PartialPatch(t *testing.T) {
	org, err := domain.NewOrganization("Original", "original", uuid.New())
	require.NoError(t, err)

	newName := "Only Name Changed"
	org.Update(domain.OrgPatch{
		Name: &newName,
	})

	assert.Equal(t, "Only Name Changed", org.Name)
	assert.Equal(t, "free", org.Plan) // Plan unchanged.
}
