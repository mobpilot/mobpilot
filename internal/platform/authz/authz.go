package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Known namespaces and relations:
//   Organization: owner, admin, member
//   App: owner, viewer, editor
//   Post: author, viewer
//   Group: member, admin

// Checker provides fine-grained RBAC checks backed by Ory Keto.
type Checker interface {
	// Check returns true if the subject has the given relation on the object.
	Check(ctx context.Context, subject, relation, object string) (bool, error)
	// WriteRelation creates a relationship tuple.
	WriteRelation(ctx context.Context, subject, relation, object string) error
	// DeleteRelation removes a relationship tuple.
	DeleteRelation(ctx context.Context, subject, relation, object string) error
}

// ketoChecker implements Checker using the Ory Keto REST API.
type ketoChecker struct {
	readURL  string
	writeURL string
	client   *http.Client
}

// NewKetoChecker creates a Checker backed by the Ory Keto read and write APIs.
// readURL is the Keto public/read endpoint (e.g., http://keto:4466).
// writeURL is the Keto admin/write endpoint (e.g., http://keto:4467).
func NewKetoChecker(readURL, writeURL string) Checker {
	return &ketoChecker{
		readURL:  strings.TrimRight(readURL, "/"),
		writeURL: strings.TrimRight(writeURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// relationTuple is the JSON representation of a Keto relationship tuple.
type relationTuple struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	SubjectID string `json:"subject_id"`
}

// parseObject splits an "Namespace:objectID" string into namespace and object ID.
// For example, "Organization:some-uuid" returns ("Organization", "some-uuid").
func parseObject(object string) (namespace, objectID string, err error) {
	idx := strings.Index(object, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("authz: invalid object format %q: expected Namespace:objectID", object)
	}
	return object[:idx], object[idx+1:], nil
}

// Check returns true if the subject has the given relation on the object.
// The object must be in "Namespace:objectID" format.
func (k *ketoChecker) Check(ctx context.Context, subject, relation, object string) (bool, error) {
	namespace, objectID, err := parseObject(object)
	if err != nil {
		return false, fmt.Errorf("ketoChecker.Check: %w", err)
	}

	tuple := relationTuple{
		Namespace: namespace,
		Object:    objectID,
		Relation:  relation,
		SubjectID: subject,
	}

	body, err := json.Marshal(tuple)
	if err != nil {
		return false, fmt.Errorf("ketoChecker.Check: marshal body: %w", err)
	}

	reqURL := k.readURL + "/relation-tuples/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("ketoChecker.Check: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("ketoChecker.Check: execute request: %w", err)
	}
	defer resp.Body.Close()

	// Keto returns 200 with {"allowed": true/false} for check requests.
	// A 403 also means not allowed.
	if resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("ketoChecker.Check: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Allowed bool `json:"allowed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("ketoChecker.Check: decode response: %w", err)
	}

	return result.Allowed, nil
}

// WriteRelation creates a relationship tuple in Keto.
// The object must be in "Namespace:objectID" format.
func (k *ketoChecker) WriteRelation(ctx context.Context, subject, relation, object string) error {
	namespace, objectID, err := parseObject(object)
	if err != nil {
		return fmt.Errorf("ketoChecker.WriteRelation: %w", err)
	}

	tuple := relationTuple{
		Namespace: namespace,
		Object:    objectID,
		Relation:  relation,
		SubjectID: subject,
	}

	body, err := json.Marshal(tuple)
	if err != nil {
		return fmt.Errorf("ketoChecker.WriteRelation: marshal body: %w", err)
	}

	reqURL := k.writeURL + "/admin/relation-tuples"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ketoChecker.WriteRelation: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("ketoChecker.WriteRelation: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ketoChecker.WriteRelation: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}

// DeleteRelation removes a relationship tuple from Keto.
// The object must be in "Namespace:objectID" format.
func (k *ketoChecker) DeleteRelation(ctx context.Context, subject, relation, object string) error {
	namespace, objectID, err := parseObject(object)
	if err != nil {
		return fmt.Errorf("ketoChecker.DeleteRelation: %w", err)
	}

	reqURL := k.writeURL + "/admin/relation-tuples"
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("ketoChecker.DeleteRelation: create request: %w", err)
	}

	q := url.Values{}
	q.Set("namespace", namespace)
	q.Set("object", objectID)
	q.Set("relation", relation)
	q.Set("subject_id", subject)
	req.URL.RawQuery = q.Encode()

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("ketoChecker.DeleteRelation: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ketoChecker.DeleteRelation: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}
