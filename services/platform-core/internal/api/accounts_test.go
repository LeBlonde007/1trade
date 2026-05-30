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

	"github.com/exascale/platform-core/internal/api"
	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/store"
	"github.com/google/uuid"
)

// TestAuditAndRBAC verifies that sensitive actions are recorded in the audit log and queryable
// (admin), and that RBAC blocks a viewer-only principal from minting keys or reading the audit log.
func TestAuditAndRBAC(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping audit/RBAC integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, ServiceToken: svcToken, TokenTTL: 3600_000_000_000}
	srv := httptest.NewServer(api.New(cfg, st))
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

	// signup → admin principal
	var signup map[string]any
	do("POST", "/v1/auth/signup", "", map[string]string{"email": "admin+" + uuid.NewString() + "@acme.ai", "password": "pw-123456"}, &signup)
	adminTok, _ := signup["token"].(string)
	claims, err := domain.VerifyToken(jwtSecret, adminTok)
	if err != nil {
		t.Fatalf("verify admin token: %v", err)
	}

	// admin can create a key (engineer satisfied by admin)
	var created map[string]any
	if code := do("POST", "/v1/auth/keys", adminTok, map[string]any{"name": "ci", "scopes": []string{"inference:read"}}, &created); code != 201 {
		t.Fatalf("admin create key = %d, want 201", code)
	}

	// audit log (admin) contains tenant.signup + apikey.create
	var audit struct {
		Entries []map[string]any `json:"entries"`
	}
	if code := do("GET", "/v1/account/audit", adminTok, nil, &audit); code != 200 {
		t.Fatalf("admin audit = %d, want 200", code)
	}
	actions := map[string]bool{}
	for _, e := range audit.Entries {
		if a, ok := e["action"].(string); ok {
			actions[a] = true
		}
	}
	if !actions["tenant.signup"] || !actions["apikey.create"] {
		t.Fatalf("audit missing expected actions: %v", actions)
	}

	// viewer-only principal (same tenant) is blocked from minting keys + reading audit
	viewerTok, _ := domain.IssueToken(jwtSecret, uuid.NewString(), domain.Claims{
		TenantID: claims.TenantID, Roles: []domain.Role{domain.RoleViewer}, IsPaper: true,
	}, time.Hour)
	if code := do("POST", "/v1/auth/keys", viewerTok, map[string]any{"name": "x", "scopes": []string{}}, nil); code != 403 {
		t.Fatalf("viewer create key = %d, want 403", code)
	}
	if code := do("GET", "/v1/account/audit", viewerTok, nil, nil); code != 403 {
		t.Fatalf("viewer audit = %d, want 403", code)
	}
	if code := do("POST", "/v1/account/orgs", viewerTok, map[string]any{"name": "x"}, nil); code != 403 {
		t.Fatalf("viewer create org = %d, want 403", code)
	}
}

// TestOrgsAndRoles covers org create/list, role assignment (audited), and tenant-scoped reads.
func TestOrgsAndRoles(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping orgs/roles integration test")
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
	do("POST", "/v1/auth/signup", "", map[string]string{"email": "owner+" + uuid.NewString() + "@acme.ai", "password": "pw-123456"}, &signup)
	tok, _ := signup["token"].(string)
	claims, _ := domain.VerifyToken(jwtSecret, tok)

	// create + list org
	var org map[string]any
	if code := do("POST", "/v1/account/orgs", tok, map[string]any{"name": "Platform Team"}, &org); code != 201 || org["id"] == nil {
		t.Fatalf("create org status %d (%v)", code, org)
	}
	var orgs struct {
		Orgs []map[string]any `json:"orgs"`
	}
	do("GET", "/v1/account/orgs", tok, nil, &orgs)
	if len(orgs.Orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs.Orgs))
	}

	// getTenant: own tenant ok; a foreign id → 404 (no cross-tenant read)
	if code := do("GET", "/v1/account/tenants/"+claims.TenantID, tok, nil, nil); code != 200 {
		t.Fatalf("get own tenant = %d, want 200", code)
	}
	if code := do("GET", "/v1/account/tenants/"+uuid.NewString(), tok, nil, nil); code != 404 {
		t.Fatalf("get foreign tenant = %d, want 404", code)
	}

	// assign roles to the admin user; an unknown role → 422
	if code := do("PUT", "/v1/account/users/"+claims.Subject+"/roles", tok, map[string]any{"roles": []string{"admin", "billing"}}, nil); code != 200 {
		t.Fatalf("assign roles = %d, want 200", code)
	}
	if code := do("PUT", "/v1/account/users/"+claims.Subject+"/roles", tok, map[string]any{"roles": []string{"wizard"}}, nil); code != 422 {
		t.Fatalf("assign unknown role = %d, want 422", code)
	}

	// audit captured org.create + user.roles.assign
	var audit struct {
		Entries []map[string]any `json:"entries"`
	}
	do("GET", "/v1/account/audit", tok, nil, &audit)
	got := map[string]bool{}
	for _, e := range audit.Entries {
		if a, ok := e["action"].(string); ok {
			got[a] = true
		}
	}
	if !got["org.create"] || !got["user.roles.assign"] {
		t.Fatalf("audit missing org/role actions: %v", got)
	}
}
