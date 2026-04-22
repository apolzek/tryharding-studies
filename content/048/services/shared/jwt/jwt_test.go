package jwt_test

import (
	"strings"
	"testing"
	"time"

	jwtpkg "github.com/obs-saas/shared/jwt"
)

func TestIssueAndVerifyRoundTrip(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef") // pragma: allowlist secret
	tok, err := jwtpkg.Issue(secret, "t-abc", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := jwtpkg.Verify(secret, "t-abc", tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TenantID != "t-abc" {
		t.Fatalf("tid = %q", claims.TenantID)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef") // pragma: allowlist secret
	other := []byte("ffffffffffffffffffffffffffffffff") // pragma: allowlist secret
	tok, _ := jwtpkg.Issue(secret, "t-abc", time.Hour)
	if _, err := jwtpkg.Verify(other, "t-abc", tok); err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestVerifyRejectsTidMismatch(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef") // pragma: allowlist secret
	tok, _ := jwtpkg.Issue(secret, "t-abc", time.Hour)
	_, err := jwtpkg.Verify(secret, "t-xyz", tok)
	if err == nil || !strings.Contains(err.Error(), "tid") {
		t.Fatalf("expected tid mismatch, got %v", err)
	}
}

func TestIssueRequiresStrongSecret(t *testing.T) {
	if _, err := jwtpkg.Issue([]byte("short"), "t-abc", 0); err == nil {
		t.Fatal("expected short-secret error")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef") // pragma: allowlist secret
	tok, _ := jwtpkg.Issue(secret, "t-abc", 1*time.Nanosecond)
	time.Sleep(5 * time.Millisecond)
	if _, err := jwtpkg.Verify(secret, "t-abc", tok); err == nil {
		t.Fatal("expected expired token")
	}
}

func TestIssueNonExpiring(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef") // pragma: allowlist secret
	tok, err := jwtpkg.Issue(secret, "t-abc", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jwtpkg.Verify(secret, "t-abc", tok); err != nil {
		t.Fatal(err)
	}
}
