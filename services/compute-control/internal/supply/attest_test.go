package supply

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

// TestChallengeAnswer: the answer is SHA-256 applied iterations times to the nonce.
func TestChallengeAnswer(t *testing.T) {
	nonce := []byte("0123456789abcdef0123456789abcdef")
	one := sha256.Sum256(nonce)
	two := sha256.Sum256(one[:])
	if ChallengeAnswer(nonce, 1) != hex.EncodeToString(one[:]) || ChallengeAnswer(nonce, 2) != hex.EncodeToString(two[:]) {
		t.Fatal("wrong iteration")
	}
}

// TestParseTrustKeysAndVerify: keys parse from base64; only a trusted key's signature verifies.
func TestParseTrustKeysAndVerify(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	_, other, _ := ed25519.GenerateKey(nil)
	keys, err := ParseTrustKeys(" " + base64.StdEncoding.EncodeToString(pub) + ", ")
	if err != nil || len(keys) != 1 {
		t.Fatalf("parse: %v %d", err, len(keys))
	}
	v := Ed25519Verifier{Keys: keys}
	msg := []byte(`{"source_id":"x"}`)
	if v.Verify(msg, ed25519.Sign(priv, msg)) != nil {
		t.Fatal("trusted signature refused")
	}
	if v.Verify(msg, ed25519.Sign(other, msg)) == nil || v.Verify([]byte(`{"source_id":"y"}`), ed25519.Sign(priv, msg)) == nil {
		t.Fatal("untrusted or altered report accepted")
	}
	for _, bad := range []string{"not base64!", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := ParseTrustKeys(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
	if keys, err := ParseTrustKeys(""); err != nil || len(keys) != 0 {
		t.Fatal("empty should be no keys")
	}
}

// TestRecordLayerOnlyManualLayers: only kyb and bond, pass or fail, are recorded by hand — checked
// before any database access.
func TestRecordLayerOnlyManualLayers(t *testing.T) {
	s := &Store{}
	for _, c := range [][2]string{{"hardware", Pass}, {"telemetry", Pass}, {"challenge", Pass}, {"kyb", Drift}, {"bond", "maybe"}} {
		if _, err := s.RecordLayer(context.Background(), "00000000-0000-4000-8000-000000000001", c[0], c[1], nil, "service"); err == nil {
			t.Errorf("%s/%s recorded by hand", c[0], c[1])
		}
	}
}
