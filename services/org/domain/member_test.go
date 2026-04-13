package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/mobpilot/mobpilot/services/org/domain"
)

func TestValidRole(t *testing.T) {
	tests := []struct {
		role  domain.MemberRole
		valid bool
	}{
		{domain.RoleOwner, true},
		{domain.RoleAdmin, true},
		{domain.RoleMember, true},
		{domain.MemberRole("superadmin"), false},
		{domain.MemberRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.valid, domain.ValidRole(tt.role))
		})
	}
}

func TestInvitation_IsExpired(t *testing.T) {
	inv := &domain.Invitation{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "test@example.com",
		Role:      domain.RoleMember,
		Token:     "abc123",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	assert.True(t, inv.IsExpired())

	inv.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	assert.False(t, inv.IsExpired())
}

func TestInvitation_IsAccepted(t *testing.T) {
	inv := &domain.Invitation{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	assert.False(t, inv.IsAccepted())

	now := time.Now().UTC()
	inv.AcceptedAt = &now
	assert.True(t, inv.IsAccepted())
}
