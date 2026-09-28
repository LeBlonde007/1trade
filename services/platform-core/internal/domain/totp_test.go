package domain

import (
	"strings"
	"testing"
	"time"
)

// TestTOTPRFC6238Vectors checks the RFC 6238 SHA-1 test vectors (last 6 digits).
func TestTOTPRFC6238Vectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	for _, c := range []struct {
		unix int64
		want string
	}{{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"}, {1234567890, "005924"}, {2000000000, "279037"}, {20000000000, "353130"}} {
		if got := TOTPCode(secret, time.Unix(c.unix, 0)); got != c.want {
			t.Errorf("T=%d: %s, want %s", c.unix, got, c.want)
		}
	}
}

// TestVerifyTOTPWindowAndReplay: codes one step either side are accepted; a used step is not.
func TestVerifyTOTPWindowAndReplay(t *testing.T) {
	secret, _ := NewTOTPSecret()
	now := time.Unix(1_800_000_000, 0)
	prev := TOTPCode(secret, now.Add(-30*time.Second))
	step, ok := VerifyTOTP(secret, prev, now, 0)
	if !ok || step != now.Unix()/30-1 {
		t.Fatal("previous step should be accepted")
	}
	if _, ok := VerifyTOTP(secret, prev, now, step); ok {
		t.Fatal("a used code was accepted again")
	}
	if _, ok := VerifyTOTP(secret, TOTPCode(secret, now.Add(-90*time.Second)), now, 0); ok {
		t.Fatal("a code three steps old was accepted")
	}
	if _, ok := VerifyTOTP(secret, "12345", now, 0); ok {
		t.Fatal("a short code was accepted")
	}
}

// TestSecretBoxAndChallenge: secrets round-trip and tampering fails; a challenge token is not a session.
func TestSecretBoxAndChallenge(t *testing.T) {
	b, err := NewSecretBox(DeriveMFAKey("s"))
	if err != nil {
		t.Fatal(err)
	}
	sealed, _ := b.Seal([]byte("secret"))
	if got, err := b.Open(sealed); err != nil || string(got) != "secret" {
		t.Fatal("round trip")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := b.Open(sealed); err == nil {
		t.Fatal("tampered secret opened")
	}
	tok, _ := IssueMFAChallenge("jwt-secret", "user-1")
	if sub, err := VerifyMFAChallenge("jwt-secret", tok); err != nil || sub != "user-1" {
		t.Fatal("challenge")
	}
	if _, err := VerifyToken("jwt-secret", tok); err == nil {
		t.Fatal("a 2FA challenge was accepted as a session token")
	}
	session, _ := IssueToken("jwt-secret", "user-1", Claims{TenantID: "t"}, time.Hour)
	if _, err := VerifyMFAChallenge("jwt-secret", session); err == nil {
		t.Fatal("a session token was accepted as a 2FA challenge")
	}
	codes, _ := NewRecoveryCodes(3)
	if len(codes[0]) != 11 || NormalizeRecoveryCode(strings.ToUpper(strings.ReplaceAll(codes[0], "-", " "))) != codes[0] {
		t.Fatalf("recovery code %q", codes[0])
	}
	if !strings.HasPrefix(TOTPURI([]byte("x"), "a@b.c"), "otpauth://totp/1Trade:a@b.c?") {
		t.Fatal(TOTPURI([]byte("x"), "a@b.c"))
	}
}
