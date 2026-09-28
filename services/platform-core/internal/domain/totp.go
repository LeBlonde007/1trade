package domain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 TOTP is defined over HMAC-SHA1; authenticator apps expect it
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TOTP parameters (RFC 6238 defaults every authenticator app supports).
const (
	TOTPPeriod = 30
	TOTPDigits = 6
	totpSkew   = 1 // accept one step either side for clock drift
)

// b32 is unpadded base32, as authenticator apps expect.
var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a fresh 160-bit secret.
func NewTOTPSecret() ([]byte, error) {
	s := make([]byte, 20)
	if _, err := rand.Read(s); err != nil {
		return nil, err
	}
	return s, nil
}

// TOTPSecretText is a secret as the user types it into an authenticator app.
func TOTPSecretText(secret []byte) string { return b32.EncodeToString(secret) }

// TOTPURI is the otpauth:// URI an authenticator app scans.
func TOTPURI(secret []byte, account string) string {
	v := url.Values{}
	v.Set("secret", TOTPSecretText(secret))
	v.Set("issuer", "1Trade")
	v.Set("algorithm", "SHA1")
	v.Set("digits", fmt.Sprint(TOTPDigits))
	v.Set("period", fmt.Sprint(TOTPPeriod))
	return "otpauth://totp/" + url.PathEscape("1Trade:"+account) + "?" + v.Encode()
}

// totpAt is the code for one time step (RFC 4226 HOTP over the step counter).
func totpAt(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step)) //nolint:gosec // steps are positive
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", TOTPDigits, bin%1_000_000)
}

// TOTPCode is the current code (for tests and enrolment checks).
func TOTPCode(secret []byte, t time.Time) string { return totpAt(secret, t.Unix()/TOTPPeriod) }

// VerifyTOTP checks a code at t and returns the time step it matched. Only steps after lastStep are
// accepted, so a code can never be used twice (replay). Comparison is constant-time.
func VerifyTOTP(secret []byte, code string, t time.Time, lastStep int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != TOTPDigits {
		return 0, false
	}
	now := t.Unix() / TOTPPeriod
	for d := int64(-totpSkew); d <= totpSkew; d++ {
		step := now + d
		if step <= lastStep {
			continue
		}
		if hmac.Equal([]byte(totpAt(secret, step)), []byte(code)) {
			return step, true
		}
	}
	return 0, false
}

// SecretBox encrypts TOTP secrets at rest with AES-256-GCM.
type SecretBox struct{ aead cipher.AEAD }

// NewSecretBox builds a box from a 32-byte key.
func NewSecretBox(key []byte) (SecretBox, error) {
	if len(key) != 32 {
		return SecretBox{}, errors.New("mfa key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return SecretBox{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return SecretBox{}, err
	}
	return SecretBox{aead: aead}, nil
}

// DeriveMFAKey derives the at-rest key from the platform secret when no dedicated key is configured
// (domain-separated, so the JWT secret itself is never used as the cipher key).
func DeriveMFAKey(platformSecret string) []byte {
	sum := sha256.Sum256([]byte("1trade/mfa-secret-box/v1\x00" + platformSecret))
	return sum[:]
}

// Seal encrypts plaintext (nonce || ciphertext).
func (b SecretBox) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decrypts what Seal produced.
func (b SecretBox) Open(sealed []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("sealed secret too short")
	}
	return b.aead.Open(nil, sealed[:n], sealed[n:], nil)
}

// NewRecoveryCodes returns n single-use recovery codes (xxxxx-xxxxx, base32).
func NewRecoveryCodes(n int) ([]string, error) {
	out := make([]string, n)
	for i := range out {
		raw := make([]byte, 7)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		s := strings.ToLower(b32.EncodeToString(raw))[:10]
		out[i] = s[:5] + "-" + s[5:]
	}
	return out, nil
}

// NormalizeRecoveryCode canonicalises what a user typed (case, spaces, dash).
func NormalizeRecoveryCode(code string) string {
	c := strings.ToLower(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(code)))
	if len(c) != 10 {
		return ""
	}
	return c[:5] + "-" + c[5:]
}

// mfaPurpose marks a challenge token: proof of the password step, good only for the 2FA step.
const mfaPurpose = "mfa"

// mfaClaims is a challenge token. It has no tenant_id, so no service accepts it as a session.
type mfaClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// IssueMFAChallenge signs a 5-minute challenge for user sub.
func IssueMFAChallenge(secret, sub string) (string, error) {
	now := time.Now().UTC()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, mfaClaims{Purpose: mfaPurpose, RegisteredClaims: jwt.RegisteredClaims{
		Subject: sub, Issuer: "platform-core", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}}).SignedString([]byte(secret))
}

// VerifyMFAChallenge returns the user a valid, unexpired challenge token is for.
func VerifyMFAChallenge(secret, tok string) (string, error) {
	c := &mfaClaims{}
	t, err := jwt.ParseWithClaims(tok, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !t.Valid || c.Purpose != mfaPurpose || c.Subject == "" {
		return "", errors.New("invalid or expired 2FA challenge")
	}
	return c.Subject, nil
}
