package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// APIKey is a freshly minted key. Secret is returned to the caller ONCE and never stored; only
// Prefix (for identification in listings) and Hash (for verification) are persisted.
type APIKey struct {
	Secret string // full token "exk_<hex>" — shown once
	Prefix string // first chars, safe to display
	Hash   string // sha256(Secret), stored
}

// GenerateAPIKey mints a scoped API key from 32 bytes of crypto-random entropy.
func GenerateAPIKey() (APIKey, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return APIKey{}, fmt.Errorf("read random: %w", err)
	}
	secret := "exk_" + hex.EncodeToString(raw)
	return APIKey{Secret: secret, Prefix: secret[:12], Hash: HashAPIKey(secret)}, nil
}

// HashAPIKey returns the storage hash of a key secret. API keys are high-entropy random tokens, so
// a fast hash (sha256) is appropriate — unlike user passwords, which need bcrypt.
func HashAPIKey(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// VerifyAPIKey reports whether a presented key matches the stored hash, in constant time.
func VerifyAPIKey(storedHash, presented string) bool {
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(HashAPIKey(presented))) == 1
}
