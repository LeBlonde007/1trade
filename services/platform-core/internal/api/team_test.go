package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/trade1/platform-core/internal/api"
	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// fakeBudgetMover records ledger transfers; short makes them fail for lack of credits.
type fakeBudgetMover struct {
	mu    sync.Mutex
	calls []billing.Transfer
	keys  []string
	short bool
}

// Transfer records the call.
func (f *fakeBudgetMover) Transfer(_ context.Context, t billing.Transfer, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.short {
		return billing.ErrInsufficientCredit
	}
	f.calls, f.keys = append(f.calls, t), append(f.keys, key)
	return nil
}

// teamRig is platform-core over the test database with a fake ledger.
type teamRig struct {
	t      *testing.T
	srv    *httptest.Server
	ledger *fakeBudgetMover
}

// newTeamRig skips without DATABASE_URL.
func newTeamRig(t *testing.T) *teamRig {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	s := api.New(config.Config{Env: "dev", JWTSecret: jwtSecret, ServiceToken: svcToken, TokenTTL: 3600_000_000_000}, st)
	rg := &teamRig{t: t, ledger: &fakeBudgetMover{}}
	s.SetBudgetMover(rg.ledger)
	rg.srv = httptest.NewServer(s)
	t.Cleanup(rg.srv.Close)
	return rg
}

// do sends a JSON request and decodes the answer.
func (rg *teamRig) do(method, path, bearer string, body any, hdr map[string]string) (int, map[string]any) {
	rg.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, rg.srv.URL+path, rdr)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		rg.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// signup creates a tenant and returns its admin's token.
func (rg *teamRig) signup() (string, *domain.Claims) {
	rg.t.Helper()
	_, out := rg.do("POST", "/v1/auth/signup", "", map[string]string{"email": "admin+" + uuid.NewString() + "@acme.ai", "password": "pw-123456", "tenant_name": "Acme"}, nil)
	tok, _ := out["token"].(string)
	c, err := domain.VerifyToken(jwtSecret, tok)
	if err != nil {
		rg.t.Fatalf("signup: %v %v", out, err)
	}
	return tok, c
}

// invite invites email with roles (and a sub-account) and returns the dev token.
func (rg *teamRig) invite(admin, email string, roles []string, sub any) string {
	rg.t.Helper()
	code, out := rg.do("POST", "/v1/account/invites", admin, map[string]any{"email": email, "roles": roles, "sub_account_id": sub}, nil)
	if code != 201 {
		rg.t.Fatalf("invite %s: %d %v", email, code, out)
	}
	return out["dev_token"].(string)
}

// accept accepts an invitation and returns the new member's token and claims.
func (rg *teamRig) accept(token string) (string, *domain.Claims) {
	rg.t.Helper()
	code, out := rg.do("POST", "/v1/account/invites/accept", "", map[string]string{"token": token, "password": "correct-horse"}, nil)
	if code != 201 {
		rg.t.Fatalf("accept: %d %v", code, out)
	}
	c, err := domain.VerifyToken(jwtSecret, out["token"].(string))
	if err != nil {
		rg.t.Fatal(err)
	}
	return out["token"].(string), c
}

// TestInviteAcceptAndMembers: an admin invites a member into a sub-account; the invitation works once;
// the member signs in with the invited roles and sub-account; only admins manage the team.
func TestInviteAcceptAndMembers(t *testing.T) {
	rg := newTeamRig(t)
	admin, ac := rg.signup()
	code, sa := rg.do("POST", "/v1/account/sub-accounts", admin, map[string]string{"name": "Research"}, nil)
	if code != 201 {
		t.Fatalf("sub-account: %d %v", code, sa)
	}
	subID := sa["id"].(string)
	if code, _ := rg.do("POST", "/v1/account/sub-accounts", admin, map[string]string{"name": "Research"}, nil); code != 409 {
		t.Fatalf("duplicate sub-account name: %d", code)
	}

	bob := "bob+" + uuid.NewString() + "@acme.ai"
	tok := rg.invite(admin, strings.ToUpper(bob[:1])+bob[1:], []string{"engineer"}, subID)
	if code, _ := rg.do("POST", "/v1/account/invites", admin, map[string]any{"email": bob, "roles": []string{"viewer"}}, nil); code != 409 {
		t.Fatalf("second pending invite: %d", code)
	}
	code, look := rg.do("GET", "/v1/account/invites/lookup?token="+tok, "", nil, nil)
	if code != 200 || look["email"] != bob || look["tenant_name"] != "Acme" {
		t.Fatalf("lookup: %d %v", code, look)
	}
	if code, _ := rg.do("POST", "/v1/account/invites/accept", "", map[string]string{"token": tok, "password": "short"}, nil); code != 422 {
		t.Fatalf("weak password: %d", code)
	}
	bobTok, bc := rg.accept(tok)
	if bc.TenantID != ac.TenantID || bc.SubAccountID != subID || len(bc.Roles) != 1 || bc.Roles[0] != domain.RoleEngineer {
		t.Fatalf("bob's claims: %+v", bc)
	}
	if code, _ := rg.do("POST", "/v1/account/invites/accept", "", map[string]string{"token": tok, "password": "correct-horse"}, nil); code != 404 {
		t.Fatalf("second use of an invitation: %d", code)
	}
	// Bob signs in normally and keeps his sub-account.
	code, login := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": bob, "password": "correct-horse"}, nil)
	if lc, err := domain.VerifyToken(jwtSecret, login["token"].(string)); code != 200 || err != nil || lc.SubAccountID != subID {
		t.Fatalf("bob login: %d %v", code, login)
	}
	code, ms := rg.do("GET", "/v1/account/members", bobTok, nil, nil)
	if code != 200 || len(ms["members"].([]any)) != 2 {
		t.Fatalf("members: %d %v", code, ms)
	}
	// Engineers do not manage the team.
	if code, _ := rg.do("POST", "/v1/account/invites", bobTok, map[string]any{"email": "x@acme.ai"}, nil); code != 403 {
		t.Fatalf("engineer invites: %d", code)
	}
	if code, _ := rg.do("DELETE", "/v1/account/members/"+ac.Subject, bobTok, nil, nil); code != 403 {
		t.Fatalf("engineer removes: %d", code)
	}
	// Invite an email that already has an account → refused.
	if code, out := rg.do("POST", "/v1/account/invites", admin, map[string]any{"email": bob}, nil); code != 409 || out["code"] != "email_taken" {
		t.Fatalf("invite an existing account: %d %v", code, out)
	}
	// Revoke: the link stops working.
	carol := rg.invite(admin, "carol+"+uuid.NewString()+"@acme.ai", nil, nil)
	_, list := rg.do("GET", "/v1/account/invites", admin, nil, nil)
	inv := list["invites"].([]any)[0].(map[string]any)
	if inv["roles"].([]any)[0] != "viewer" {
		t.Fatalf("default role: %v", inv)
	}
	if code, _ := rg.do("DELETE", "/v1/account/invites/"+inv["id"].(string), admin, nil, nil); code != 204 {
		t.Fatalf("revoke: %d", code)
	}
	if code, _ := rg.do("GET", "/v1/account/invites/lookup?token="+carol, "", nil, nil); code != 404 {
		t.Fatalf("revoked link: %d", code)
	}
	// Move Bob to the main balance, then remove him: he can no longer sign in.
	if code, _ := rg.do("PUT", "/v1/account/members/"+bc.Subject+"/sub-account", admin, map[string]any{"sub_account_id": nil}, nil); code != 200 {
		t.Fatalf("move to main: %d", code)
	}
	if code, out := rg.do("DELETE", "/v1/account/members/"+ac.Subject, admin, nil, nil); code != 409 || out["code"] != "self" {
		t.Fatalf("remove yourself: %d %v", code, out)
	}
	if code, _ := rg.do("DELETE", "/v1/account/members/"+bc.Subject, admin, nil, nil); code != 204 {
		t.Fatalf("remove bob: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": bob, "password": "correct-horse"}, nil); code != 401 {
		t.Fatalf("removed member signs in: %d", code)
	}
	_, audit := rg.do("GET", "/v1/account/audit", admin, nil, nil)
	seen := map[string]bool{}
	for _, e := range audit["entries"].([]any) {
		seen[e.(map[string]any)["action"].(string)] = true
	}
	for _, a := range []string{"invite.create", "invite.accept", "invite.revoke", "user.remove", "user.sub_account.set", "sub_account.create"} {
		if !seen[a] {
			t.Errorf("audit is missing %s", a)
		}
	}
}

// TestLastAdminAndIsolation: a stale admin token cannot remove the last real admin; nothing crosses
// tenants.
func TestLastAdminAndIsolation(t *testing.T) {
	rg := newTeamRig(t)
	admin, ac := rg.signup()
	dave, dc := rg.accept(rg.invite(admin, "dave+"+uuid.NewString()+"@acme.ai", []string{"admin"}, nil))
	// Dave demotes the founder; the founder's token still says admin.
	if code, _ := rg.do("PUT", "/v1/account/users/"+ac.Subject+"/roles", dave, map[string]any{"roles": []string{"viewer"}}, nil); code != 200 {
		t.Fatalf("demote: %d", code)
	}
	if code, out := rg.do("DELETE", "/v1/account/members/"+dc.Subject, admin, nil, nil); code != 409 || out["code"] != "last_admin" {
		t.Fatalf("remove the last admin with a stale token: %d %v", code, out)
	}
	other, _ := rg.signup()
	_, sa := rg.do("POST", "/v1/account/sub-accounts", other, map[string]string{"name": "Theirs"}, nil)
	theirs := sa["id"].(string)
	if code, _ := rg.do("PUT", "/v1/account/members/"+dc.Subject+"/sub-account", dave, map[string]any{"sub_account_id": theirs}, nil); code != 404 {
		t.Fatalf("assign another tenant's sub-account: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/account/invites", dave, map[string]any{"email": "e@acme.ai", "sub_account_id": theirs}, nil); code != 404 {
		t.Fatalf("invite into another tenant's sub-account: %d", code)
	}
	if code, _ := rg.do("DELETE", "/v1/account/members/"+ac.Subject, other, nil, nil); code != 404 {
		t.Fatalf("remove another tenant's member: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/account/sub-accounts/"+theirs+"/transfer", dave, map[string]string{"credit_type": "text", "amount": "1", "direction": "fund"}, map[string]string{"Idempotency-Key": "k1"}); code != 404 {
		t.Fatalf("fund another tenant's sub-account: %d", code)
	}
}

// TestSubAccountTransfers: billing funds and drains a sub-account through the ledger; the key is
// scoped to the sub-account and direction; bad input and short balances are refused.
func TestSubAccountTransfers(t *testing.T) {
	rg := newTeamRig(t)
	admin, ac := rg.signup()
	_, sa := rg.do("POST", "/v1/account/sub-accounts", admin, map[string]string{"name": "Infra"}, nil)
	id := sa["id"].(string)
	path := "/v1/account/sub-accounts/" + id + "/transfer"
	key := map[string]string{"Idempotency-Key": "fund-1"}
	if code, out := rg.do("POST", path, admin, map[string]string{"credit_type": "gpu_h100", "amount": "10.5", "direction": "fund"}, key); code != 200 {
		t.Fatalf("fund: %d %v", code, out)
	}
	c := rg.ledger.calls[0]
	if c.TenantID != ac.TenantID || c.ToSub == nil || *c.ToSub != id || c.FromSub != nil || c.Amount != "10.5" || !c.IsPaper || rg.ledger.keys[0] != "sub:"+id+":fund:fund-1" {
		t.Fatalf("ledger call: %+v key %s", c, rg.ledger.keys[0])
	}
	if code, _ := rg.do("POST", path, admin, map[string]string{"credit_type": "gpu_h100", "amount": "1", "direction": "return"}, map[string]string{"Idempotency-Key": "ret-1"}); code != 200 ||
		rg.ledger.calls[1].FromSub == nil || rg.ledger.calls[1].ToSub != nil {
		t.Fatalf("return: %d %+v", code, rg.ledger.calls[1])
	}
	for name, b := range map[string]map[string]string{
		"direction": {"credit_type": "gpu_h100", "amount": "1", "direction": "steal"},
		"type":      {"credit_type": "gold", "amount": "1", "direction": "fund"},
		"zero":      {"credit_type": "text", "amount": "0.000", "direction": "fund"},
		"precision": {"credit_type": "text", "amount": "1.0000001", "direction": "fund"},
		"negative":  {"credit_type": "text", "amount": "-1", "direction": "fund"},
	} {
		if code, _ := rg.do("POST", path, admin, b, map[string]string{"Idempotency-Key": "v-" + name}); code != 422 {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	if code, _ := rg.do("POST", path, admin, map[string]string{"credit_type": "text", "amount": "1", "direction": "fund"}, nil); code != 400 {
		t.Fatalf("no key: %d", code)
	}
	rg.ledger.short = true
	if code, _ := rg.do("POST", path, admin, map[string]string{"credit_type": "text", "amount": "1", "direction": "fund"}, map[string]string{"Idempotency-Key": "s"}); code != 402 {
		t.Fatalf("short: %d", code)
	}
	eng, _ := rg.accept(rg.invite(admin, "eng+"+uuid.NewString()+"@acme.ai", []string{"engineer"}, nil))
	if code, _ := rg.do("POST", path, eng, map[string]string{"credit_type": "text", "amount": "1", "direction": "fund"}, map[string]string{"Idempotency-Key": "e"}); code != 403 {
		t.Fatalf("engineer moves budget: %d", code)
	}
}
