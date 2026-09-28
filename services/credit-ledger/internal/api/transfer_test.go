package api_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// subBalance reads a paper gpu_h100 balance of a tenant's sub-account ("" = main) straight from the
// ledger tables.
func subBalance(t *testing.T, tenant, sub string) string {
	t.Helper()
	p, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	var subArg any
	if sub != "" {
		subArg = sub
	}
	var bal string
	err = p.QueryRow(context.Background(), `SELECT coalesce((SELECT balance::text FROM credit_balances WHERE tenant_id=$1
		AND sub_account_id IS NOT DISTINCT FROM $2::uuid AND credit_type='gpu_h100' AND is_paper), '0.000000')`, tenant, subArg).Scan(&bal)
	if err != nil {
		t.Fatal(err)
	}
	return bal
}

// TestTransferBetweenMainAndSubAccount: credits move main → sub-account and back as two chained
// legs; a replay moves nothing more; an overdraft moves nothing at all; only the platform may call it.
func TestTransferBetweenMainAndSubAccount(t *testing.T) {
	h := newHarness(t)
	tenant, sub := uuid.NewString(), uuid.NewString()
	h.fund(tenant, "", "10")
	svc := func(key string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + serviceToken, "Idempotency-Key": key}
	}
	body := func(from, to any, amount string) map[string]any {
		return map[string]any{"tenant_id": tenant, "from_sub_account_id": from, "to_sub_account_id": to,
			"credit_type": "gpu_h100", "amount": amount, "is_paper": true}
	}
	var first map[string]any
	if code := h.do("POST", "/v1/credits/transfer", svc("fund-1"), body(nil, sub, "4"), &first); code != 200 {
		t.Fatalf("fund: %d %v", code, first)
	}
	if subBalance(t, tenant, "") != "6.000000" || subBalance(t, tenant, sub) != "4.000000" {
		t.Fatalf("after funding: main %s sub %s", subBalance(t, tenant, ""), subBalance(t, tenant, sub))
	}
	var again map[string]any
	if code := h.do("POST", "/v1/credits/transfer", svc("fund-1"), body(nil, sub, "4"), &again); code != 200 ||
		again["debit"].(map[string]any)["tx_id"] != first["debit"].(map[string]any)["tx_id"] {
		t.Fatalf("replay: %d %v", code, again)
	}
	if subBalance(t, tenant, "") != "6.000000" {
		t.Fatal("a replay moved credits again")
	}
	if code := h.do("POST", "/v1/credits/transfer", svc("fund-2"), body(nil, sub, "6.000001"), nil); code != 402 {
		t.Fatalf("overdraft: %d", code)
	}
	if subBalance(t, tenant, "") != "6.000000" || subBalance(t, tenant, sub) != "4.000000" {
		t.Fatal("a refused transfer moved credits")
	}
	if code := h.do("POST", "/v1/credits/transfer", svc("return-1"), body(sub, nil, "1.5"), nil); code != 200 {
		t.Fatalf("return: %d", code)
	}
	if subBalance(t, tenant, "") != "7.500000" || subBalance(t, tenant, sub) != "2.500000" {
		t.Fatalf("after return: main %s sub %s", subBalance(t, tenant, ""), subBalance(t, tenant, sub))
	}
	for name, c := range map[string]struct {
		hdr  map[string]string
		body map[string]any
		want int
	}{
		"same balance":  {svc("x1"), body(sub, sub, "1"), 422},
		"zero":          {svc("x2"), body(nil, sub, "0"), 422},
		"bad sub id":    {svc("x3"), body(nil, "team-a", "1"), 422},
		"no key":        {map[string]string{"Authorization": "Bearer " + serviceToken}, body(nil, sub, "1"), 400},
		"not a service": {map[string]string{"X-Dev-Tenant": tenant, "Idempotency-Key": "x4"}, body(nil, sub, "1"), 401},
	} {
		if code := h.do("POST", "/v1/credits/transfer", c.hdr, c.body, nil); code != c.want {
			t.Errorf("%s: %d, want %d", name, code, c.want)
		}
	}
	var v map[string]any
	if code := h.do("GET", "/v1/credits/audit/chain-verify?tenant_id="+tenant, map[string]string{"Authorization": "Bearer " + serviceToken}, nil, &v); code != 200 || v["ok"] != true || v["checked"] != float64(5) {
		t.Fatalf("chain verify: %d %v", code, v)
	}
}
