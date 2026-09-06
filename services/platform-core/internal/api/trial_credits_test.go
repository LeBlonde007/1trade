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

	"github.com/google/uuid"
	"github.com/trade1/platform-core/internal/api"
	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// TestVerifyGrantsTrialCreditsOnce proves the starter grant that removes the payment wall from
// time-to-first-action: verifying an email books a one-time paper balance, so a fresh tenant can
// make a real metered call without buying anything. It also pins the two properties that keep the
// grant safe — it is ALWAYS paper (a free grant must never reach a real-money balance), and it is
// idempotent, so replaying the verification link cannot mint twice.
// Runs against real Postgres (skips without DATABASE_URL).
func TestVerifyGrantsTrialCreditsOnce(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping trial-credit integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	booker := &stubBooker{}
	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, RequireEmailVerification: true, TokenTTL: time.Hour}
	srv := httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, booker))
	defer srv.Close()

	postJSON := func(path string, body any) int {
		b, _ := json.Marshal(body)
		resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	email := "trial+" + uuid.NewString() + "@acme.ai"
	if code := postJSON("/v1/auth/signup", map[string]string{
		"email": email, "password": "pw-123456", "tenant_name": "Trial Co",
	}); code != http.StatusCreated {
		t.Fatalf("signup = %d, want 201", code)
	}

	// With the gate ON, signup alone must NOT grant — the account is unusable until verified.
	if got := booker.snapshot(); len(got) != 0 {
		t.Fatalf("bookings after signup = %d, want 0", len(got))
	}

	au, ok, err := st.GetUserByEmail(context.Background(), email)
	if err != nil || !ok {
		t.Fatalf("lookup user: ok=%v err=%v", ok, err)
	}
	raw := "trial-token-" + uuid.NewString()
	if err := st.SetVerifyToken(context.Background(), au.UserID, domain.HashAPIKey(raw)); err != nil {
		t.Fatalf("set verify token: %v", err)
	}

	if code := postJSON("/v1/auth/verify", map[string]string{"token": raw}); code != http.StatusOK {
		t.Fatalf("verify = %d, want 200", code)
	}

	got := booker.snapshot()
	if len(got) != 1 {
		t.Fatalf("bookings after verify = %d, want 1", len(got))
	}
	g := got[0]
	if !g.IsPaper {
		t.Error("trial grant must be paper — a free grant must never book to a real-money balance")
	}
	if g.CreditType != api.TrialCreditType {
		t.Errorf("credit_type = %q, want %q", g.CreditType, api.TrialCreditType)
	}
	if g.Amount != api.TrialCreditAmount {
		t.Errorf("amount = %q, want %q", g.Amount, api.TrialCreditAmount)
	}
	if g.IdempotencyKey == "" {
		t.Error("grant needs an Idempotency-Key so the ledger can dedupe a replayed verification")
	}

	// Replaying a consumed token must not grant again. The token is single-use, so this 400s —
	// the point is that no second booking is attempted.
	if code := postJSON("/v1/auth/verify", map[string]string{"token": raw}); code != http.StatusBadRequest {
		t.Fatalf("replayed verify = %d, want 400", code)
	}
	if after := booker.snapshot(); len(after) != 1 {
		t.Fatalf("bookings after replay = %d, want 1 (no double grant)", len(after))
	}
}

// TestSignupGrantsTrialCreditsWhenNoVerificationGate covers the environment shape that local dev
// uses: RequireEmailVerification=false, so signup issues a session immediately and verifyEmail is
// never called. Granting only on verify would leave every account here with a zero balance while
// onboarding claimed credits were waiting — the screen would lie.
// Runs against real Postgres (skips without DATABASE_URL).
func TestSignupGrantsTrialCreditsWhenNoVerificationGate(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping trial-credit integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	booker := &stubBooker{}
	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, RequireEmailVerification: false, TokenTTL: time.Hour}
	srv := httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, booker))
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"email": "nogate+" + uuid.NewString() + "@acme.ai", "password": "pw-123456", "tenant_name": "NoGate Co",
	})
	resp, err := http.Post(srv.URL+"/v1/auth/signup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup = %d, want 201", resp.StatusCode)
	}

	got := booker.snapshot()
	if len(got) != 1 {
		t.Fatalf("bookings after gate-less signup = %d, want 1", len(got))
	}
	if !got[0].IsPaper {
		t.Error("trial grant must be paper")
	}
	if got[0].CreditType != api.TrialCreditType || got[0].Amount != api.TrialCreditAmount {
		t.Errorf("grant = %s %s, want %s %s",
			got[0].Amount, got[0].CreditType, api.TrialCreditAmount, api.TrialCreditType)
	}
}
