// End-to-end JWT flow: issue a token as the auth-service would, then verify
// it the way the per-tenant auth-proxy does. Catches signing/verification
// drift between the two places that share the secret.
package integration_test

import (
	"testing"
	"time"

	jwtpkg "github.com/obs-saas/shared/jwt"
)

func TestIssuedTokenVerifiesWithSharedSecret(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef" // pragma: allowlist secret
	const tid = "t-01HXYZABC"

	tok, err := jwtpkg.Issue([]byte(secret), tid, 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := jwtpkg.Verify([]byte(secret), tid, tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.TenantID != tid {
		t.Fatalf("tid mismatch: %s", claims.TenantID)
	}
}

func TestCrossTenantTokenRejected(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef" // pragma: allowlist secret
	// Tenant A gets a token; tenant B's proxy must reject it.
	tokA, _ := jwtpkg.Issue([]byte(secret), "t-A", time.Hour)
	if _, err := jwtpkg.Verify([]byte(secret), "t-B", tokA); err == nil {
		t.Fatal("token from t-A must not verify for t-B")
	}
}
