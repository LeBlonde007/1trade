package domain

import (
	"fmt"
	"testing"
	"time"
)

// TestVerifyStripeSignature covers the happy path plus every rejection: wrong secret, tampered
// payload, stale timestamp, and a malformed header.
func TestVerifyStripeSignature(t *testing.T) {
	secret := "whsec_test_abc123"
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed"}`)
	now := time.Unix(1700000000, 0)
	hdr := func(ts int64, sig string) string { return fmt.Sprintf("t=%d,v1=%s", ts, sig) }

	good := hdr(now.Unix(), SignStripePayload(payload, now.Unix(), secret))
	if err := VerifyStripeSignature(payload, good, secret, StripeSignatureTolerance, now); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}

	// wrong secret
	if err := VerifyStripeSignature(payload, good, "whsec_other", StripeSignatureTolerance, now); err == nil {
		t.Fatal("accepted signature under the wrong secret")
	}
	// tampered payload
	if err := VerifyStripeSignature([]byte(`{"id":"evt_evil"}`), good, secret, StripeSignatureTolerance, now); err == nil {
		t.Fatal("accepted a tampered payload")
	}
	// stale timestamp (10 min old, signed correctly for that old t)
	oldTs := now.Add(-10 * time.Minute).Unix()
	stale := hdr(oldTs, SignStripePayload(payload, oldTs, secret))
	if err := VerifyStripeSignature(payload, stale, secret, StripeSignatureTolerance, now); err == nil {
		t.Fatal("accepted a stale (replayed) signature")
	}
	// malformed header
	if err := VerifyStripeSignature(payload, "not-a-real-header", secret, StripeSignatureTolerance, now); err == nil {
		t.Fatal("accepted a malformed header")
	}
	// empty secret fails closed
	if err := VerifyStripeSignature(payload, good, "", StripeSignatureTolerance, now); err == nil {
		t.Fatal("accepted with no configured secret")
	}
}
