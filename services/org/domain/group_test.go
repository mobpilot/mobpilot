package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/org/domain"
)

func TestNewGroup(t *testing.T) {
	appID     := uuid.New()
	createdBy := uuid.New()
	expires   := time.Now().Add(time.Hour)

	tests := []struct {
		name        string
		orgID       uuid.UUID
		groupName   string
		desc        string
		ephemeral   bool
		expiresAt   *time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid permanent group",
			orgID:     uuid.New(),
			groupName: "Engineering",
			desc:      "The engineering team",
			ephemeral: false,
			wantErr:   false,
		},
		{
			name:      "valid ephemeral group with expiry",
			orgID:     uuid.New(),
			groupName: "Card Game",
			desc:      "Quick card game session",
			ephemeral: true,
			expiresAt: &expires,
			wantErr:   false,
		},
		{
			name:        "ephemeral group missing expiry",
			orgID:       uuid.New(),
			groupName:   "Game",
			ephemeral:   true,
			expiresAt:   nil,
			wantErr:     true,
			errContains: "ephemeral groups must have an expires_at",
		},
		{
			name:        "nil orgID",
			orgID:       uuid.Nil,
			groupName:   "Engineering",
			wantErr:     true,
			errContains: "orgID must not be nil",
		},
		{
			name:        "empty name",
			orgID:       uuid.New(),
			groupName:   "",
			wantErr:     true,
			errContains: "group name must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := domain.NewGroup(tt.orgID, appID, createdBy, tt.groupName, tt.desc, tt.ephemeral, tt.expiresAt)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.groupName, group.Name)
			assert.Equal(t, tt.desc, group.Description)
			assert.Equal(t, tt.orgID, group.OrgID)
			assert.Equal(t, tt.ephemeral, group.Ephemeral)
			assert.NotEqual(t, uuid.Nil, group.ID)
		})
	}
}
