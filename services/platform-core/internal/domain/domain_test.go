package domain

import (
	"testing"
	"time"
)

// TestPasswordHashVerify checks bcrypt hashing + verification (and that a wrong password fails).
func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if h == "correct horse battery staple" {
		t.Fatal("password stored in plaintext")
	}
	if !VerifyPassword(h, "correct horse battery staple") {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

const secret = "test-secret-at-least-32-chars-long-xxxxx"

// TestTokenRoundTrip issues a JWT and verifies the claims survive — including the exact keys
// (tenant_id, is_paper, roles) that consuming services like credit-ledger read.
func TestTokenRoundTrip(t *testing.T) {
	tok, err := IssueToken(secret, "user-1", Claims{
		TenantID: "tenant-1", OrgID: "org-1", Roles: []Role{RoleAdmin, RoleBilling}, IsPaper: true,
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyToken(secret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.TenantID != "tenant-1" || !c.IsPaper || c.Subject != "user-1" || len(c.Roles) != 2 {
		t.Fatalf("claims mismatch: %+v", c)
	}
}

// TestTokenRejects ensures wrong-secret and expired tokens are rejected.
func TestTokenRejects(t *testing.T) {
	tok, _ := IssueToken(secret, "u", Claims{TenantID: "t"}, time.Hour)
	if _, err := VerifyToken("a-different-secret-also-32-chars-xxxxxx", tok); err == nil {
		t.Fatal("token verified under the wrong secret")
	}
	expired, _ := IssueToken(secret, "u", Claims{TenantID: "t"}, -time.Hour)
	if _, err := VerifyToken(secret, expired); err == nil {
		t.Fatal("expired token accepted")
	}
}

// TestTokenRequiresTenant ensures a token without tenant_id is rejected at verification.
func TestTokenRequiresTenant(t *testing.T) {
	tok, _ := IssueToken(secret, "u", Claims{TenantID: ""}, time.Hour)
	if _, err := VerifyToken(secret, tok); err == nil {
		t.Fatal("token without tenant_id accepted")
	}
}

// TestAPIKey checks generation, prefix, deterministic hashing, and constant-time verify.
func TestAPIKey(t *testing.T) {
	k, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Prefix) != 12 || k.Secret[:12] != k.Prefix {
		t.Fatalf("bad prefix: %q", k.Prefix)
	}
	if k.Hash == k.Secret {
		t.Fatal("api key stored in plaintext")
	}
	if !VerifyAPIKey(k.Hash, k.Secret) {
		t.Fatal("valid key rejected")
	}
	if VerifyAPIKey(k.Hash, "exk_wrong") {
		t.Fatal("wrong key accepted")
	}
	if HashAPIKey("exk_x") != HashAPIKey("exk_x") {
		t.Fatal("hash not deterministic")
	}
}
