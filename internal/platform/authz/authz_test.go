package authz_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobpilot/mobpilot/internal/platform/authz"
)

func TestKetoChecker_Check_Allowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/relation-tuples/check", r.URL.Path)

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, "Organization", body["namespace"])
		assert.Equal(t, "org-123", body["object"])
		assert.Equal(t, "member", body["relation"])
		assert.Equal(t, "user-456", body["subject_id"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"allowed": true})
	}))
	defer server.Close()

	checker := authz.NewKetoChecker(server.URL, server.URL)
	allowed, err := checker.Check(context.Background(), "user-456", "member", "Organization:org-123")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestKetoChecker_Check_Denied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"allowed": false})
	}))
	defer server.Close()

	checker := authz.NewKetoChecker(server.URL, server.URL)
	allowed, err := checker.Check(context.Background(), "user-456", "admin", "Organization:org-123")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestKetoChecker_Check_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	checker := authz.NewKetoChecker(server.URL, server.URL)
	allowed, err := checker.Check(context.Background(), "user-456", "owner", "Organization:org-123")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestKetoChecker_WriteRelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/admin/relation-tuples", r.URL.Path)

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, "App", body["namespace"])
		assert.Equal(t, "app-789", body["object"])
		assert.Equal(t, "editor", body["relation"])
		assert.Equal(t, "user-456", body["subject_id"])

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	checker := authz.NewKetoChecker(server.URL, server.URL)
	err := checker.WriteRelation(context.Background(), "user-456", "editor", "App:app-789")
	require.NoError(t, err)
}

func TestKetoChecker_DeleteRelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/admin/relation-tuples", r.URL.Path)

		q := r.URL.Query()
		assert.Equal(t, "Group", q.Get("namespace"))
		assert.Equal(t, "grp-111", q.Get("object"))
		assert.Equal(t, "member", q.Get("relation"))
		assert.Equal(t, "user-222", q.Get("subject_id"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	checker := authz.NewKetoChecker(server.URL, server.URL)
	err := checker.DeleteRelation(context.Background(), "user-222", "member", "Group:grp-111")
	require.NoError(t, err)
}

func TestKetoChecker_InvalidObjectFormat(t *testing.T) {
	checker := authz.NewKetoChecker("http://localhost", "http://localhost")

	_, err := checker.Check(context.Background(), "user", "rel", "no-colon")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid object format")

	err = checker.WriteRelation(context.Background(), "user", "rel", "no-colon")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid object format")

	err = checker.DeleteRelation(context.Background(), "user", "rel", "no-colon")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid object format")
}
