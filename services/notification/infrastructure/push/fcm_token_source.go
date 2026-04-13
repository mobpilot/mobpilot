package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	fcmScope        = "https://www.googleapis.com/auth/firebase.messaging"
	googleTokenURL  = "https://oauth2.googleapis.com/token"
	tokenExpirySecs = 3600
	// Refresh the token 5 minutes before it expires.
	tokenRefreshLead = 5 * time.Minute
)

// serviceAccountJSON is the structure of a Google service account JSON key file.
type serviceAccountJSON struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	TokenURI                string `json:"token_uri"`
}

// cachedToken holds an access token alongside its expiry.
type cachedToken struct {
	value     string
	expiresAt time.Time
}

// ServiceAccountTokenSource fetches OAuth2 access tokens for the FCM v1 API
// using a Google service account JSON key file. Tokens are cached and refreshed
// transparently.
type ServiceAccountTokenSource struct {
	privateKey  *rsa.PrivateKey
	privateKeyID string
	clientEmail  string
	tokenURI     string
	client       *http.Client

	mu    sync.Mutex
	cache *cachedToken
}

// NewServiceAccountTokenSource parses the given service account JSON bytes and
// returns a TokenSource ready to issue FCM access tokens.
func NewServiceAccountTokenSource(saJSON []byte) (*ServiceAccountTokenSource, error) {
	var sa serviceAccountJSON
	if err := json.Unmarshal(saJSON, &sa); err != nil {
		return nil, fmt.Errorf("parse service account JSON: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("expected service_account key type, got %q", sa.Type)
	}

	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private_key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = googleTokenURL
	}

	return &ServiceAccountTokenSource{
		privateKey:   rsaKey,
		privateKeyID: sa.PrivateKeyID,
		clientEmail:  sa.ClientEmail,
		tokenURI:     tokenURI,
		client:       &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Token returns a valid FCM OAuth2 access token, fetching a new one if the
// cached token has expired (or is about to expire).
func (s *ServiceAccountTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cache != nil && time.Until(s.cache.expiresAt) > tokenRefreshLead {
		return s.cache.value, nil
	}

	token, expiry, err := s.fetchToken(ctx)
	if err != nil {
		return "", err
	}
	s.cache = &cachedToken{value: token, expiresAt: expiry}
	return token, nil
}

// fetchToken creates a self-signed JWT and exchanges it for a Google access token.
func (s *ServiceAccountTokenSource) fetchToken(ctx context.Context) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(time.Duration(tokenExpirySecs) * time.Second)

	// Build a JWT signed with the service account's private key (RS256).
	claims := jwt.MapClaims{
		"iss":   s.clientEmail,
		"sub":   s.clientEmail,
		"scope": fcmScope,
		"aud":   s.tokenURI,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = s.privateKeyID

	assertion, err := tok.SignedString(s.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT: %w", err)
	}

	// Exchange the JWT for a Google OAuth2 access token.
	body := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURI,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, respBody)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("empty access_token in response")
	}

	actualExpiry := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, actualExpiry, nil
}
