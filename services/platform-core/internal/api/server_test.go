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
	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/store"
	"github.com/google/uuid"
)

const jwtSecret = "platform-core-test-secret-32+chars-xxxxx"

// TestAPIIntegration drives the real HTTP handlers against real Postgres via httptest: signup →
// token, /me, login, API-key create/list/revoke, and the auth failure paths. It also asserts the
// issued token carries the exact claims (tenant_id, is_paper) that other services (credit-ledger)
// verify — the cross-service auth contract.
func TestAPIIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping API integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, TokenTTL: 3600_000_000_000} // 1h
	srv := httptest.NewServer(api.New(cfg, st))
	defer srv.Close()

	email := "ceo+" + uuid.NewString() + "@acme.ai"

	do := func(method, path, bearer string, body any, out any) int {
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

	// signup → token
	var signup map[string]any
	if code := do("POST", "/v1/auth/signup", "", map[string]string{"email": email, "password": "pw-12345", "tenant_name": "Acme"}, &signup); code != 201 {
		t.Fatalf("signup status %d (%v)", code, signup)
	}
	token, _ := signup["token"].(string)
	if token == "" {
		t.Fatal("signup returned no token")
	}

	// CROSS-SERVICE CONTRACT: the token must verify with the shared secret and carry tenant_id +
	// is_paper — exactly what credit-ledger reads. This is what makes one login work fleet-wide.
	claims, err := domain.VerifyToken(jwtSecret, token)
	if err != nil || claims.TenantID == "" || !claims.IsPaper {
		t.Fatalf("issued token is not fleet-verifiable: err=%v claims=%+v", err, claims)
	}

	// /me with the token → email matches
	var me map[string]any
	if code := do("GET", "/v1/auth/me", token, nil, &me); code != 200 || me["email"] != email {
		t.Fatalf("me status %d email=%v", code, me["email"])
	}
	// /me without a token → 401
	if code := do("GET", "/v1/auth/me", "", nil, nil); code != 401 {
		t.Fatalf("me without token = %d, want 401", code)
	}

	// login (correct + wrong password)
	var login map[string]any
	if code := do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "pw-12345"}, &login); code != 200 || login["token"] == nil {
		t.Fatalf("login status %d (%v)", code, login)
	}
	if code := do("POST", "/v1/auth/login", "", map[string]string{"email": email, "password": "wrong"}, nil); code != 401 {
		t.Fatalf("login wrong password = %d, want 401", code)
	}

	// API key: create (secret once) → list → revoke
	var created map[string]any
	if code := do("POST", "/v1/auth/keys", token, map[string]any{"name": "ci", "scopes": []string{"inference:read"}}, &created); code != 201 {
		t.Fatalf("create key status %d (%v)", code, created)
	}
	if created["secret"] == nil || created["id"] == nil {
		t.Fatalf("create key missing secret/id: %v", created)
	}
	var list struct {
		Keys []map[string]any `json:"keys"`
	}
	do("GET", "/v1/auth/keys", token, nil, &list)
	if len(list.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(list.Keys))
	}
	if code := do("DELETE", "/v1/auth/keys/"+created["id"].(string), token, nil, nil); code != 204 {
		t.Fatalf("revoke key = %d, want 204", code)
	}
}
