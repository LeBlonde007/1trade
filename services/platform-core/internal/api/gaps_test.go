package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/exascale/platform-core/internal/api"
	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/store"
	"github.com/google/uuid"
)

// TestVerifyAndBudget covers the email-verification flow (resend → token → verify, single-use) and
// the monthly budget set/get.
func TestVerifyAndBudget(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping verify/budget integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	srv := httptest.NewServer(api.New(config.Config{Env: "dev", JWTSecret: jwtSecret, TokenTTL: 3600_000_000_000}, st))
	defer srv.Close()

	do := func(method, path, bearer string, body, out any) int {
		var rdr *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, rdr)
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

	var signup map[string]any
	do("POST", "/v1/auth/signup", "", map[string]string{"email": "verify+" + uuid.NewString() + "@acme.ai", "password": "pw-123456"}, &signup)
	tok, _ := signup["token"].(string)

	// resend → dev token; verify once → 200; reuse → 400; bad token → 400
	var rs map[string]any
	do("POST", "/v1/auth/verify/resend", tok, nil, &rs)
	dt, _ := rs["dev_token"].(string)
	if dt == "" {
		t.Fatal("resend returned no dev_token")
	}
	if code := do("POST", "/v1/auth/verify", "", map[string]string{"token": dt}, nil); code != 200 {
		t.Fatalf("verify = %d, want 200", code)
	}
	if code := do("POST", "/v1/auth/verify", "", map[string]string{"token": dt}, nil); code != 400 {
		t.Fatalf("verify reuse = %d, want 400", code)
	}
	if code := do("POST", "/v1/auth/verify", "", map[string]string{"token": "not-a-token"}, nil); code != 400 {
		t.Fatalf("verify bad token = %d, want 400", code)
	}

	// budget set (admin satisfies billing) → get
	if code := do("PUT", "/v1/billing/budget", tok, map[string]string{"credit_type": "text", "monthly_limit": "1000.000000"}, nil); code != 200 {
		t.Fatalf("set budget = %d, want 200", code)
	}
	var got struct {
		Budget map[string]any `json:"budget"`
	}
	do("GET", "/v1/billing/budget", tok, nil, &got)
	if got.Budget == nil || got.Budget["monthly_limit"] != "1000.000000" || got.Budget["credit_type"] != "text" {
		t.Fatalf("get budget mismatch: %v", got.Budget)
	}
}
