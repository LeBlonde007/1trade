package domain

import "strings"

// NormalizeEmail canonicalises an email for storage and lookup: trimmed + lowercased. Logins are
// case-insensitive and there is exactly one account per address regardless of the case it was typed
// in — so `CEO@Acme.ai` and `ceo@acme.ai` are the same identity, never two accounts.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
