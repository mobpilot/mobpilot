package http

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CentrifugoTokenIssuer signs Centrifugo connection and channel subscription
// JWTs using HMAC-SHA256. It implements application.TokenIssuer.
type CentrifugoTokenIssuer struct {
	secret []byte
}

// NewCentrifugoTokenIssuer creates a token issuer from the given HMAC secret.
func NewCentrifugoTokenIssuer(secret string) *CentrifugoTokenIssuer {
	return &CentrifugoTokenIssuer{secret: []byte(secret)}
}

// IssueConnectionToken returns a connection JWT for the given user. Connection
// tokens are valid for one hour; the client should refresh before expiry.
func (i *CentrifugoTokenIssuer) IssueConnectionToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := t.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("CentrifugoTokenIssuer.IssueConnectionToken: %w", err)
	}
	return token, nil
}

// IssueGroupToken returns a channel subscription JWT for an ephemeral group
// channel (channel = "ephemeral-group:<groupID>"). The token expires when the
// group expires — clients cannot re-subscribe after that point.
func (i *CentrifugoTokenIssuer) IssueGroupToken(userID, groupID uuid.UUID, expiresAt time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":     userID.String(),
		"channel": "ephemeral-group:" + groupID.String(),
		"exp":     expiresAt.Unix(),
		"info": map[string]string{
			"groupId": groupID.String(),
			"role":    "member",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := t.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("CentrifugoTokenIssuer.IssueGroupToken: %w", err)
	}
	return token, nil
}
