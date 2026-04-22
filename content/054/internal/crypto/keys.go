// Package crypto wraps the small set of primitives trynet needs: Curve25519
// keypairs for WireGuard, and random token generation for pre-auth keys.
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"github.com/apolzek/trynet/internal/protocol"
)

// GenerateKeyPair returns a fresh Curve25519 private/public pair, WireGuard-style.
func GenerateKeyPair() (priv, pub protocol.Key, err error) {
	if _, err = rand.Read(priv[:]); err != nil {
		return priv, pub, fmt.Errorf("rand: %w", err)
	}
	// WireGuard clamps the private key before multiplying.
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pubB, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return priv, pub, fmt.Errorf("curve25519: %w", err)
	}
	copy(pub[:], pubB)
	return priv, pub, nil
}

// NewToken returns a URL-safe random token of n bytes, base64-encoded.
func NewToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // refusing to continue without randomness
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
