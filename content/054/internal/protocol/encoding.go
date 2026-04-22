package protocol

import (
	"encoding/base64"
	"fmt"
)

func encodeKey(k Key) string {
	return base64.RawURLEncoding.EncodeToString(k[:])
}

func decodeKey(s string) (Key, error) {
	var k Key
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		// allow standard padded too
		b2, err2 := base64.StdEncoding.DecodeString(s)
		if err2 != nil {
			return k, fmt.Errorf("decode key: %w", err)
		}
		b = b2
	}
	if len(b) != 32 {
		return k, fmt.Errorf("key length %d, want 32", len(b))
	}
	copy(k[:], b)
	return k, nil
}

// ParseKey decodes a base64 key from a string.
func ParseKey(s string) (Key, error) { return decodeKey(s) }
