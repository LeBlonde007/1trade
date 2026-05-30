package domain

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// NormalizeEmail canonicalises an email for storage and lookup: trimmed + lowercased. Logins are
// case-insensitive and there is exactly one account per address regardless of the case it was typed
// in — so `CEO@Acme.ai` and `ceo@acme.ai` are the same identity, never two accounts.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// NewVerifyToken returns a high-entropy email-verification token (raw). The raw token goes in the
// email link; only its hash (HashAPIKey) is stored, so a leaked DB row can't be used to verify.
func NewVerifyToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
