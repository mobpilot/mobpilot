package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/services/org/domain"
)

func TestNewRole(t *testing.T) {
	tests := []struct {
		name        string
		orgID       uuid.UUID
		roleName    string
		permissions []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid role",
			orgID:       uuid.New(),
			roleName:    "editor",
			permissions: []string{"posts:write", "posts:read"},
			wantErr:     false,
		},
		{
			name:        "nil orgID",
			orgID:       uuid.Nil,
			roleName:    "editor",
			permissions: []string{},
			wantErr:     true,
			errContains: "orgID must not be nil",
		},
		{
			name:        "empty name",
			orgID:       uuid.New(),
			roleName:    "",
			permissions: []string{},
			wantErr:     true,
			errContains: "role name must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := domain.NewRole(tt.orgID, tt.roleName, tt.permissions)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.roleName, role.Name)
			assert.Equal(t, tt.permissions, role.Permissions)
			assert.Equal(t, tt.orgID, role.OrgID)
		})
	}
}
