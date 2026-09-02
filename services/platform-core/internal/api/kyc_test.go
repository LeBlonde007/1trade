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
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
	"github.com/google/uuid"
)

// TestKYCGatesRealMoneyCheckout proves the F22 server-side gate: a sandbox (is_paper) checkout never
// consults KYC; a real-money (is_paper=false) checkout is blocked with 403 kyc_required until the
// tenant verifies, then succeeds. Runs against real Postgres (skips without DATABASE_URL).
func TestKYCGatesRealMoneyCheckout(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping KYC integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, BillingAutoSettle: true, KYCAutoApprove: true, TokenTTL: time.Hour}
	srv := httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, &stubBooker{}))
	defer srv.Close()

	doJSON := func(method, path, bearer string, body, out any) int {
		var r *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		} else {
			r = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, r)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
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

	// Signup → a sandbox (is_paper=true) tenant + token.
	var signup map[string]any
	doJSON("POST", "/v1/auth/signup", "", map[string]string{
		"email": "kyc+" + uuid.NewString() + "@acme.ai", "password": "pw-123456", "tenant_name": "Acme",
	}, &signup)
	sandboxToken, _ := signup["token"].(string)
	if sandboxToken == "" {
		t.Fatal("no token from signup")
	}
	claims, err := domain.VerifyToken(jwtSecret, sandboxToken)
	if err != nil {
		t.Fatalf("verify signup token: %v", err)
	}

	// 1) Sandbox checkout is exempt — unverified, but is_paper=true → 200.
	var co map[string]any
	if code := doJSON("POST", "/v1/billing/checkout", sandboxToken,
		map[string]string{"amount": "100.000000", "credit_type": "text", "currency": "usd"}, &co); code != 200 {
		t.Fatalf("sandbox checkout = %d (want 200); KYC must not gate paper flows: %v", code, co)
	}

	// Mint a real-money (is_paper=false) token for the SAME tenant/user (simulates a real-money account).
	realToken, err := domain.IssueToken(jwtSecret, claims.Subject,
		domain.Claims{TenantID: claims.TenantID, Roles: []domain.Role{domain.RoleAdmin}, IsPaper: false}, time.Hour)
	if err != nil {
		t.Fatalf("issue real-money token: %v", err)
	}

	// 2) Real-money checkout while unverified → 403 kyc_required.
	var blocked map[string]any
	if code := doJSON("POST", "/v1/billing/checkout", realToken,
		map[string]string{"amount": "100.000000", "credit_type": "text", "currency": "usd"}, &blocked); code != 403 {
		t.Fatalf("real-money checkout (unverified) = %d (want 403): %v", code, blocked)
	}
	if blocked["code"] != "kyc_required" {
		t.Fatalf("expected code=kyc_required, got %v", blocked)
	}

	// 3) Submit KYC → auto-verified in dev.
	var kyc map[string]any
	if code := doJSON("POST", "/v1/account/kyc", realToken,
		map[string]string{"legal_name": "Acme Inc.", "country": "us", "entity_type": "business"}, &kyc); code != 200 {
		t.Fatalf("kyc submit = %d: %v", code, kyc)
	}
	if kyc["status"] != "verified" || kyc["can_purchase"] != true {
		t.Fatalf("expected verified + can_purchase after dev submit, got %v", kyc)
	}

	// 4) Real-money checkout now passes → 200, settled inline.
	var co2 map[string]any
	if code := doJSON("POST", "/v1/billing/checkout", realToken,
		map[string]string{"amount": "100.000000", "credit_type": "text", "currency": "usd"}, &co2); code != 200 {
		t.Fatalf("real-money checkout (verified) = %d (want 200): %v", code, co2)
	}
	if co2["settled"] != true {
		t.Fatalf("verified real-money checkout should settle: %v", co2)
	}

	// 5) Resubmitting once verified is a conflict (no churn on a decided record).
	if code := doJSON("POST", "/v1/account/kyc", realToken,
		map[string]string{"legal_name": "Acme Inc.", "country": "us", "entity_type": "business"}, nil); code != 409 {
		t.Fatalf("resubmit while verified = %d (want 409)", code)
	}
}
