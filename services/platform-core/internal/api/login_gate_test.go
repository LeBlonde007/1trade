package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/trade1/platform-core/internal/api"
	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/store"
	"github.com/google/uuid"
)

// TestLoginRequiresVerifiedEmail proves the sandbox/prod gate (RequireEmailVerification): signup issues
// NO session (verification_required), login is refused with 403 email_unverified until the email is
// confirmed, then login succeeds. Runs against real Postgres (skips without DATABASE_URL).
func TestLoginRequiresVerifiedEmail(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping login-gate integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, RequireEmailVerification: true, TokenTTL: time.Hour}
	srv := httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, &stubBooker{}))
	defer srv.Close()

	doJSON := func(method, path string, body, out any) int {
		var r *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		} else {
			r = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, r)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		if out != nil {
			_ = json.NewDecoder(resp.Body).Decode(out)
		}
		return resp.StatusCode
	}

	email := "gate+" + uuid.NewString() + "@acme.ai"
	creds := map[string]string{"email": email, "password": "pw-123456", "tenant_name": "Gate Co"}

	// 1) Signup → 201 but NO session token (verification required).
	var signup map[string]any
	if code := doJSON("POST", "/v1/auth/signup", creds, &signup); code != http.StatusCreated {
		t.Fatalf("signup = %d, want 201", code)
	}
	if signup["token"] != nil {
		t.Fatal("signup must NOT return a token when email verification is required")
	}
	if signup["status"] != "verification_required" {
		t.Fatalf("signup status = %v, want verification_required", signup["status"])
	}

	// 2) Login before verifying → 403 email_unverified (not 401 — password is correct).
	var blocked map[string]any
	if code := doJSON("POST", "/v1/auth/login", creds, &blocked); code != http.StatusForbidden {
		t.Fatalf("unverified login = %d, want 403", code)
	}
	if blocked["code"] != "email_unverified" {
		t.Fatalf("unverified login code = %v, want email_unverified", blocked["code"])
	}

	// 3) Confirm the email (simulate the clicked link via the store), then login → 200 + token.
	au, ok, err := st.GetUserByEmail(context.Background(), email)
	if err != nil || !ok {
		t.Fatalf("lookup user: ok=%v err=%v", ok, err)
	}
	if err := st.SetVerifyToken(context.Background(), au.UserID, "gate-token-hash"); err != nil {
		t.Fatalf("set verify token: %v", err)
	}
	if _, _, _, verified, err := st.VerifyEmail(context.Background(), "gate-token-hash"); err != nil || !verified {
		t.Fatalf("verify email: ok=%v err=%v", verified, err)
	}
	var success map[string]any
	if code := doJSON("POST", "/v1/auth/login", creds, &success); code != http.StatusOK {
		t.Fatalf("verified login = %d, want 200", code)
	}
	if _, hasToken := success["token"].(string); !hasToken {
		t.Fatal("verified login must return a session token")
	}
}
