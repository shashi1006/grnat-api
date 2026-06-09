// Package jwt provides JWT creation and validation for the ReadyGeneration API.
package jwt

import (
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are the custom JWT claims.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	OrgID  *uuid.UUID `json:"oid,omitempty"`
	Role   string    `json:"role"`
	gojwt.RegisteredClaims
}

// Manager handles JWT signing and verification.
type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewManager creates a JWT Manager.
func NewManager(secret, issuer string, ttlSeconds int) *Manager {
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}
	return &Manager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// Sign generates a signed JWT for the given user.
func (m *Manager) Sign(userID uuid.UUID, orgID *uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning the claims.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenStr, &Claims{}, func(t *gojwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token expired")
		}
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// TTL returns the token lifetime.
func (m *Manager) TTL() time.Duration { return m.ttl }
