package domain

import "regexp"

// creditTypes is the canonical enum (docs/contracts/credit-types.md §1).
var creditTypes = map[string]bool{
	"ai_index": true, "text": true, "speech": true, "image": true, "video": true, "embeddings": true,
	"gpu_h100": true, "gpu_h200": true,
}

// ValidCreditType reports whether t is a credit type in the canonical enum.
func ValidCreditType(t string) bool { return creditTypes[t] }

// amountRe is a positive fixed-point decimal with at most 14 integer digits and 6 decimals
// (NUMERIC(20,6)).
var amountRe = regexp.MustCompile(`^(0|[1-9][0-9]{0,13})(\.[0-9]{1,6})?$`)

// ValidAmount reports whether s is a positive NUMERIC(20,6) amount.
func ValidAmount(s string) bool {
	if !amountRe.MatchString(s) {
		return false
	}
	for _, c := range s {
		if c >= '1' && c <= '9' {
			return true
		}
	}
	return false // all zeros
}
