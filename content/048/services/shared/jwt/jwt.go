// Package jwt issues and verifies tenant ingest tokens.
//
// Every tenant gets one long-lived HS256 JWT with claim `tid` = tenant id.
// The same secret is shared between the control plane (auth-service, for
// signing) and the tenant's auth-proxy pod (for verification). Rotation is a
// chart upgrade with a new secret — see docs/upgrades.md.
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const Issuer = "obs-saas"

type TenantClaims struct {
	TenantID string `json:"tid"`
	jwtv5.RegisteredClaims
}

// Issue mints an HS256 token valid for `ttl`. Pass ttl<=0 for non-expiring
// ingest tokens (we still include issued-at so rotation is observable).
func Issue(secret []byte, tenantID string, ttl time.Duration) (string, error) {
	if len(secret) < 32 {
		return "", errors.New("jwt: secret must be at least 32 bytes")
	}
	if tenantID == "" {
		return "", errors.New("jwt: tenantID required")
	}
	now := time.Now()
	claims := TenantClaims{
		TenantID: tenantID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:   Issuer,
			Subject:  tenantID,
			IssuedAt: jwtv5.NewNumericDate(now),
		},
	}
	if ttl > 0 {
		claims.ExpiresAt = jwtv5.NewNumericDate(now.Add(ttl))
	}
	t := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

// Verify checks signature, issuer and `tid == expected`. Returns the claims on
// success. `expected==""` skips the tid check (useful in the control plane).
func Verify(secret []byte, expected, token string) (*TenantClaims, error) {
	var claims TenantClaims
	t, err := jwtv5.ParseWithClaims(token, &claims, func(t *jwtv5.Token) (any, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	}, jwtv5.WithIssuer(Issuer))
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("jwt: invalid token")
	}
	if expected != "" && claims.TenantID != expected {
		return nil, fmt.Errorf("jwt: tid mismatch")
	}
	return &claims, nil
}
