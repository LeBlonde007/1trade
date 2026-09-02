package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/trade1/platform-core/internal/domain"
	"github.com/google/uuid"
)

// TestStoreIntegration exercises the real Postgres store: signup (tenant+user), login lookup,
// duplicate-email rejection, and the API-key create→lookup→revoke lifecycle. Skips unless
// DATABASE_URL is set (schema must be applied).
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping store integration test")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	email := "founder+" + uuid.NewString() + "@acme.ai" // unique per run
	hash, _ := domain.HashPassword("s3cret-pw")

	// signup → tenant + admin user
	u, err := s.Signup(ctx, email, hash, "Acme AI")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if u.TenantID == "" || !u.IsPaper || len(u.Roles) != 1 || u.Roles[0] != domain.RoleAdmin {
		t.Fatalf("unexpected signup result: %+v", u)
	}

	// duplicate email rejected
	if _, err := s.Signup(ctx, email, hash, "Dupe"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	// login lookup returns the record with the right tenant + password hash + is_paper
	au, ok, err := s.GetUserByEmail(ctx, email)
	if err != nil || !ok {
		t.Fatalf("GetUserByEmail ok=%v err=%v", ok, err)
	}
	if au.TenantID != u.TenantID || !domain.VerifyPassword(au.PasswordHash, "s3cret-pw") || !au.IsPaper {
		t.Fatalf("login record mismatch: %+v", au)
	}
	if _, ok, _ := s.GetUserByEmail(ctx, "nobody-"+uuid.NewString()+"@x.com"); ok {
		t.Fatal("unknown email reported as found")
	}
	// Login is case-insensitive: the same address in a different case resolves the same account.
	if up, ok, err := s.GetUserByEmail(ctx, strings.ToUpper(email)); err != nil || !ok || up.UserID != u.ID {
		t.Fatalf("case-insensitive lookup failed: ok=%v err=%v", ok, err)
	}

	// api key: create → lookup by hash → revoke → lookup fails
	k, _ := domain.GenerateAPIKey()
	id, err := s.CreateAPIKey(ctx, u.TenantID, "ci", k.Prefix, k.Hash, []string{"inference:read"})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	tid, scopes, isPaper, ok, err := s.LookupAPIKey(ctx, k.Hash)
	if err != nil || !ok || tid != u.TenantID || len(scopes) != 1 || !isPaper {
		t.Fatalf("lookup key: ok=%v tid=%v scopes=%v isPaper=%v err=%v", ok, tid, scopes, isPaper, err)
	}
	if err := s.RevokeAPIKey(ctx, u.TenantID, id); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, _, _, ok, _ := s.LookupAPIKey(ctx, k.Hash); ok {
		t.Fatal("revoked key still resolves")
	}
}
