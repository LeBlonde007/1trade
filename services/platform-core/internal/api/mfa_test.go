package api_test

import (
	"encoding/base32"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/trade1/platform-core/internal/domain"
)

// enrol signs up, turns 2FA on, and returns the email, secret and recovery codes.
func (rg *teamRig) enrol(t *testing.T) (email string, secret []byte, recovery []any, tok string) {
	t.Helper()
	email = "mfa+" + uuid.NewString() + "@acme.ai"
	_, out := rg.do("POST", "/v1/auth/signup", "", map[string]string{"email": email, "password": "pw-123456"}, nil)
	tok = out["token"].(string)
	code, setup := rg.do("POST", "/v1/auth/2fa/setup", tok, nil, nil)
	if code != 200 {
		t.Fatalf("setup: %d %v", code, setup)
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(setup["secret"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := rg.do("POST", "/v1/auth/2fa/enable", tok, map[string]string{"code": "000000"}, nil); code != 422 {
		t.Fatalf("wrong confirmation code: %d", code)
	}
	code, en := rg.do("POST", "/v1/auth/2fa/enable", tok, map[string]string{"code": domain.TOTPCode(secret, time.Now())}, nil)
	if code != 200 {
		t.Fatalf("enable: %d %v", code, en)
	}
	return email, secret, en["recovery_codes"].([]any), tok
}

// TestTwoFactorLogin: with 2FA on, the password earns only a challenge; a code redeems it once; a
// recovery code works once; a challenge is not a session.
func TestTwoFactorLogin(t *testing.T) {
	rg := newTeamRig(t)
	email, secret, recovery, tok := rg.enrol(t)
	if len(recovery) != 10 {
		t.Fatalf("recovery codes: %v", recovery)
	}
	if _, me := rg.do("GET", "/v1/auth/me", tok, nil, nil); me["mfa_enabled"] != true {
		t.Fatalf("me: %v", me)
	}
	if code, _ := rg.do("POST", "/v1/auth/2fa/setup", tok, nil, nil); code != 409 {
		t.Fatalf("setup while on: %d", code)
	}
	login := func() string {
		code, out := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "pw-123456"}, nil)
		if code != 200 || out["mfa_required"] != true || out["token"] != nil {
			t.Fatalf("login: %d %v", code, out)
		}
		return out["mfa_token"].(string)
	}
	ch := login()
	if code, _ := rg.do("GET", "/v1/auth/me", ch, nil, nil); code != 401 {
		t.Fatalf("a challenge used as a session: %d", code)
	}
	// The code that enabled 2FA is spent; the next step's code works once.
	if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": ch, "code": domain.TOTPCode(secret, time.Now())}, nil); code != 401 {
		t.Fatalf("the enrolment code reused: %d", code)
	}
	next := domain.TOTPCode(secret, time.Now().Add(30*time.Second))
	code, out := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": login(), "code": next}, nil)
	if code != 200 || out["token"] == nil {
		t.Fatalf("second step: %d %v", code, out)
	}
	if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": login(), "code": next}, nil); code != 401 {
		t.Fatalf("replayed code: %d", code)
	}
	rc := recovery[0].(string)
	if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": login(), "recovery_code": rc}, nil); code != 200 {
		t.Fatalf("recovery code: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": login(), "recovery_code": rc}, nil); code != 401 {
		t.Fatalf("recovery code reused: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": tok, "code": next}, nil); code != 401 {
		t.Fatalf("a session token used as a challenge: %d", code)
	}
	_, st := rg.do("GET", "/v1/auth/2fa", tok, nil, nil)
	if st["enabled"] != true || st["recovery_codes_left"] != float64(9) {
		t.Fatalf("status: %v", st)
	}
	// Turning 2FA off needs a second factor, not just the session.
	if code, _ := rg.do("POST", "/v1/auth/2fa/disable", tok, map[string]string{"code": "123456"}, nil); code != 422 {
		t.Fatalf("disable with a wrong code: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/auth/2fa/disable", tok, map[string]string{"recovery_code": recovery[1].(string)}, nil); code != 204 {
		t.Fatalf("disable: %d", code)
	}
	if code, out := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "pw-123456"}, nil); code != 200 || out["token"] == nil {
		t.Fatalf("login after disabling: %d %v", code, out)
	}
}

// TestTwoFactorLockout: five wrong codes lock the second step, even for a right code.
func TestTwoFactorLockout(t *testing.T) {
	rg := newTeamRig(t)
	email, secret, _, _ := rg.enrol(t)
	_, out := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "pw-123456"}, nil)
	ch := out["mfa_token"].(string)
	for i := range 5 {
		if code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": ch, "code": "000000"}, nil); code != 401 {
			t.Fatalf("wrong code %d: %d", i, code)
		}
	}
	good := domain.TOTPCode(secret, time.Now().Add(30*time.Second))
	if code, out := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": ch, "code": good}, nil); code != 429 || out["code"] != "mfa_locked" {
		t.Fatalf("after five wrong codes: %d %v", code, out)
	}
}

// TestTwoFactorCodeRaceWinsOnce: the same code submitted concurrently signs in exactly once.
func TestTwoFactorCodeRaceWinsOnce(t *testing.T) {
	rg := newTeamRig(t)
	email, secret, _, _ := rg.enrol(t)
	next := domain.TOTPCode(secret, time.Now().Add(30*time.Second))
	chs := make([]string, 8)
	for i := range chs {
		_, out := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "pw-123456"}, nil)
		chs[i] = out["mfa_token"].(string)
	}
	results := make(chan int, len(chs))
	start := make(chan struct{})
	for _, ch := range chs {
		go func(ch string) {
			<-start
			code, _ := rg.do("POST", "/v1/auth/login/2fa", "", map[string]string{"mfa_token": ch, "code": next}, nil)
			results <- code
		}(ch)
	}
	close(start)
	wins := 0
	for range chs {
		if <-results == 200 {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("%d concurrent sign-ins with one code, want exactly 1", wins)
	}
}
