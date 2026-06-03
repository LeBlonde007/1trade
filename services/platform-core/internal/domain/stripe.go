package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StripeSignatureTolerance is the default allowed clock skew between the webhook timestamp and now.
// Replays older than this are rejected even if the HMAC matches (Stripe's recommended default).
const StripeSignatureTolerance = 5 * time.Minute

// VerifyStripeSignature validates a Stripe webhook signature without the Stripe SDK. It parses the
// `Stripe-Signature` header (`t=<unix>,v1=<hex>[,v1=<hex>...]`), recomputes
// HMAC-SHA256(`<t>.<payload>`) with the endpoint's signing secret, constant-time compares it to each
// v1 value, and rejects timestamps outside the tolerance window (replay/skew defense). Returns nil
// only when a signature matches and the timestamp is fresh.
func VerifyStripeSignature(payload []byte, sigHeader, secret string, tolerance time.Duration, now time.Time) error {
	if secret == "" {
		return fmt.Errorf("no webhook secret configured")
	}
	var ts string
	var v1s []string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			v1s = append(v1s, kv[1])
		}
	}
	if ts == "" || len(v1s) == 0 {
		return fmt.Errorf("malformed Stripe-Signature header")
	}
	tsec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("bad signature timestamp")
	}
	if d := now.Sub(time.Unix(tsec, 0)); d > tolerance || d < -tolerance {
		return fmt.Errorf("signature timestamp outside tolerance")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ts + "." + string(payload))) // hash.Hash.Write never returns an error
	expected := mac.Sum(nil)
	for _, v := range v1s {
		sig, err := hex.DecodeString(v)
		if err != nil {
			continue
		}
		if hmac.Equal(sig, expected) { // constant-time
			return nil
		}
	}
	return fmt.Errorf("no matching v1 signature")
}

// SignStripePayload produces the hex v1 signature for a payload+timestamp (used by tests and by the
// mock Stripe to emit webhooks the verifier accepts).
func SignStripePayload(payload []byte, ts int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + string(payload))) // never errors
	return hex.EncodeToString(mac.Sum(nil))
}
