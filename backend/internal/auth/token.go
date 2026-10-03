package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateToken creates a cryptographically random, hex-encoded opaque
// token with 256 bits of entropy. Used for session tokens, password reset
// tokens, and invitation tokens alike — the raw value is shown to the
// client exactly once (as a cookie, or as a link) and is never itself
// stored; see HashToken.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generating token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 hash of a raw token. Unlike
// passwords, tokens are already high-entropy random values rather than
// something a human chose, so a fast cryptographic hash is appropriate
// here — Argon2id's deliberate slowness is unnecessary and would make
// every authenticated request noticeably slower for no security benefit.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
