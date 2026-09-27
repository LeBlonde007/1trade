package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/credit-ledger/internal/api"
	"github.com/trade1/credit-ledger/internal/config"
	"github.com/trade1/credit-ledger/internal/events"
	"github.com/trade1/credit-ledger/internal/store"
)

const (
	settleToken  = "engine-only-token"
	serviceToken = "shared-service-token"
)

// harness is a ledger HTTP server over real Postgres with a settlement token configured.
type harness struct {
	t   *testing.T
	srv *httptest.Server
	st  *store.Store
}

// newHarness skips unless DATABASE_URL points at a migrated ledger database.
func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping settlement integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)
	srv := httptest.NewServer(api.New(config.Config{Env: "dev", ServiceToken: serviceToken, SettleToken: settleToken}, st, events.LogPublisher{}))
	t.Cleanup(srv.Close)
	return &harness{t: t, srv: srv, st: st}
}

// do sends a JSON request with the given headers and decodes the response into out.
func (h *harness) do(method, path string, headers map[string]string, body, out any) int {
	h.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, h.srv.URL+path, rdr)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

// fund gives a tenant paper cash and paper gpu_h100 credits.
func (h *harness) fund(tenant, cash, gpu string) {
	h.t.Helper()
	svc := map[string]string{"Authorization": "Bearer " + serviceToken}
	if cash != "" {
		svc["Idempotency-Key"] = "paper-grant:" + tenant
		if code := h.do("POST", "/v1/credits/paper-cash/grant", svc,
			map[string]any{"tenant_id": tenant, "currency": "USD", "amount": cash, "is_paper": true}, nil); code != 200 {
			h.t.Fatalf("grant status %d", code)
		}
	}
	if gpu != "" {
		svc["Idempotency-Key"] = "buy-gpu:" + tenant
		if code := h.do("POST", "/v1/credits/purchase", svc,
			map[string]any{"tenant_id": tenant, "credit_type": "gpu_h100", "amount": gpu, "reference_id": "buy-gpu:" + tenant, "is_paper": true}, nil); code != 200 {
			h.t.Fatalf("purchase status %d", code)
		}
	}
}

// cash returns a tenant's paper USD balance ("0.000000" when none).
func (h *harness) cash(tenant string) string {
	h.t.Helper()
	var out struct {
		Balances []struct{ Balance string } `json:"balances"`
	}
	h.do("GET", "/v1/credits/cash/balances", map[string]string{"X-Dev-Tenant": tenant}, nil, &out)
	if len(out.Balances) == 0 {
		return "0.000000"
	}
	return out.Balances[0].Balance
}

// gpu returns a tenant's paper gpu_h100 balance ("0.000000" when none).
func (h *harness) gpu(tenant string) string {
	h.t.Helper()
	var out struct {
		Balances []struct {
			CreditType string `json:"credit_type"`
			Balance    string `json:"balance"`
		} `json:"balances"`
	}
	h.do("GET", "/v1/credits/balances", map[string]string{"X-Dev-Tenant": tenant}, nil, &out)
	for _, b := range out.Balances {
		if b.CreditType == "gpu_h100" {
			return b.Balance
		}
	}
	return "0.000000"
}

// tradeBody builds a settle-trade body: buyer buys qty gpu_h100 from seller at price.
func tradeBody(tradeID, buyer, seller, price, qty, buyerFee, sellerFee string) map[string]any {
	return map[string]any{
		"trade_id": tradeID, "product_id": "H100-SPOT", "credit_type": "gpu_h100",
		"price": price, "quantity": qty, "currency": "USD", "is_paper": true, "chain_hash": "engine-hash-" + tradeID,
		"buyer":  map[string]any{"tenant_id": buyer, "sub_account_id": nil, "order_id": uuid.NewString(), "fee": buyerFee, "is_internal": false},
		"seller": map[string]any{"tenant_id": seller, "sub_account_id": nil, "order_id": uuid.NewString(), "fee": sellerFee, "is_internal": false},
	}
}

// engine returns the headers the matching engine sends for a trade.
func engine(tradeID string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + settleToken, "Idempotency-Key": tradeID}
}

// TestSettleTradeEndToEnd settles one trade and checks every balance, the response, replay, conflict,
// and that both tenants' credit and cash chains still verify.
func TestSettleTradeEndToEnd(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "10000", "")
	h.fund(seller, "", "100")

	id := uuid.NewString()
	body := tradeBody(id, buyer, seller, "2.99", "10", "0.299", "0.1495")
	var res struct {
		Notional string           `json:"notional"`
		Credit   []map[string]any `json:"credit_transactions"`
		Cash     []map[string]any `json:"cash_transactions"`
	}
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), body, &res); code != 200 {
		t.Fatalf("settle status %d", code)
	}
	if res.Notional != "29.900000" || len(res.Credit) != 2 || len(res.Cash) != 4 {
		t.Fatalf("result = %+v", res)
	}
	check := func(label, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %s, want %s", label, got, want)
		}
	}
	check("buyer gpu", h.gpu(buyer), "10.000000")
	check("seller gpu", h.gpu(seller), "90.000000")
	check("buyer cash", h.cash(buyer), "9969.801000") // 10000 − 29.9 − 0.299
	check("seller cash", h.cash(seller), "29.750500") // 29.9 − 0.1495

	// Replay: same body → same result, nothing moves.
	var again struct {
		Notional string `json:"notional"`
	}
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), body, &again); code != 200 || again.Notional != "29.900000" {
		t.Fatalf("replay status %d notional %s", code, again.Notional)
	}
	check("buyer cash after replay", h.cash(buyer), "9969.801000")
	check("seller gpu after replay", h.gpu(seller), "90.000000")

	// Same trade_id, different body → 409, nothing moves.
	changed := tradeBody(id, buyer, seller, "2.99", "11", "0.299", "0.1495")
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), changed, nil); code != 409 {
		t.Errorf("conflicting replay status %d, want 409", code)
	}
	check("buyer gpu after conflict", h.gpu(buyer), "10.000000")

	for _, tenant := range []string{buyer, seller} {
		var v struct {
			OK      bool `json:"ok"`
			Checked int  `json:"checked"`
		}
		h.do("GET", "/v1/credits/audit/chain-verify?tenant_id="+tenant, map[string]string{"Authorization": "Bearer " + serviceToken}, nil, &v)
		if !v.OK || v.Checked < 3 {
			t.Errorf("chain-verify %s = %+v", tenant, v)
		}
	}
}

// TestSettleAllOrNothing checks a short seller or a short buyer writes nothing at all.
func TestSettleAllOrNothing(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "20", "")
	h.fund(seller, "", "5")

	id := uuid.NewString()
	var e map[string]string
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), tradeBody(id, buyer, seller, "2.99", "6", "0", "0"), &e); code != 402 || e["code"] != "INSUFFICIENT_CREDIT" {
		t.Fatalf("seller short: %d %v", code, e)
	}
	id = uuid.NewString()
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), tradeBody(id, buyer, seller, "2.99", "5", "6", "0"), &e); code != 402 || e["code"] != "INSUFFICIENT_CASH" {
		t.Fatalf("buyer short (fee pushes over): %d %v", code, e)
	}
	if h.cash(buyer) != "20.000000" || h.gpu(seller) != "5.000000" || h.gpu(buyer) != "0.000000" || h.cash(seller) != "0.000000" {
		t.Errorf("a refused settlement moved value: buyer cash %s gpu %s, seller cash %s gpu %s",
			h.cash(buyer), h.gpu(buyer), h.cash(seller), h.gpu(seller))
	}
}

// TestSettleCallerAndPolicy checks only the engine's token is accepted and every policy refusal.
func TestSettleCallerAndPolicy(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "100", "")
	h.fund(seller, "", "10")
	id := uuid.NewString()
	body := tradeBody(id, buyer, seller, "1", "1", "0", "0")

	for name, hdr := range map[string]map[string]string{
		"no token":             {"Idempotency-Key": id},
		"shared service token": {"Authorization": "Bearer " + serviceToken, "Idempotency-Key": id},
		"dev tenant shortcut":  {"X-Dev-Tenant": buyer, "Idempotency-Key": id},
		"wrong token":          {"Authorization": "Bearer engine-only-tokeX", "Idempotency-Key": id},
	} {
		if code := h.do("POST", "/v1/credits/settle-trade", hdr, body, nil); code != 403 {
			t.Errorf("%s: status %d, want 403", name, code)
		}
	}

	policy := []struct {
		name string
		mod  func(b map[string]any)
		hdr  map[string]string
		code string
	}{
		{"real money", func(b map[string]any) { b["is_paper"] = false }, nil, "REAL_MONEY_DISABLED"},
		{"self trade", func(b map[string]any) { b["seller"].(map[string]any)["tenant_id"] = buyer }, nil, "SELF_TRADE"},
		{"internal on paper", func(b map[string]any) { b["seller"].(map[string]any)["is_internal"] = true }, nil, "INTERNAL_ON_PAPER"},
		{"key differs from trade_id", func(map[string]any) {}, map[string]string{"Authorization": "Bearer " + settleToken, "Idempotency-Key": "other"}, "bad_request"},
		{"non-uuid tenant", func(b map[string]any) { b["buyer"].(map[string]any)["tenant_id"] = "x" }, nil, "bad_request"},
		{"too many decimals", func(b map[string]any) { b["price"] = "1.0000001" }, nil, "bad_request"},
	}
	for _, p := range policy {
		b := tradeBody(id, buyer, seller, "1", "1", "0", "0")
		p.mod(b)
		hdr := p.hdr
		if hdr == nil {
			hdr = engine(id)
		}
		var e map[string]string
		if code := h.do("POST", "/v1/credits/settle-trade", hdr, b, &e); code != 422 || e["code"] != p.code {
			t.Errorf("%s: %d %v, want 422 %s", p.name, code, e, p.code)
		}
	}
	if h.gpu(buyer) != "0.000000" || h.cash(buyer) != "100.000000" {
		t.Error("a refused request moved value")
	}
}

// TestSettleConcurrentSameTrade fires the same trade from many goroutines: exactly one settlement.
func TestSettleConcurrentSameTrade(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "1000", "")
	h.fund(seller, "", "100")
	id := uuid.NewString()
	body := tradeBody(id, buyer, seller, "2", "10", "0.2", "0.1")
	var wg sync.WaitGroup
	codes := make([]int, 20)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = h.do("POST", "/v1/credits/settle-trade", engine(id), body, nil)
		}(i)
	}
	wg.Wait()
	for i, c := range codes {
		if c != 200 {
			t.Errorf("call %d status %d", i, c)
		}
	}
	if h.gpu(buyer) != "10.000000" || h.cash(buyer) != "979.800000" || h.cash(seller) != "19.900000" {
		t.Errorf("settled more than once: buyer gpu %s cash %s, seller cash %s", h.gpu(buyer), h.cash(buyer), h.cash(seller))
	}
}

// TestSettleConcurrentOppositeDirections settles many trades between two tenants in both directions at
// once. The global leg ordering must prevent deadlocks: every call succeeds, and the net position is
// exactly the sum of the trades.
func TestSettleConcurrentOppositeDirections(t *testing.T) {
	h := newHarness(t)
	a, b := uuid.NewString(), uuid.NewString()
	h.fund(a, "1000", "100")
	h.fund(b, "1000", "100")
	var wg sync.WaitGroup
	errs := make(chan string, 40)
	for i := range 40 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			buyer, seller := a, b
			if i%2 == 1 {
				buyer, seller = b, a
			}
			id := uuid.NewString()
			if code := h.do("POST", "/v1/credits/settle-trade", engine(id), tradeBody(id, buyer, seller, "1", "1", "0.01", "0.005"), nil); code != 200 {
				errs <- fmt.Sprintf("trade %d: status %d", i, code)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	// 20 trades each way: credits net to zero per tenant; each paid 20×0.01 + 20×0.005 in fees.
	for _, tenant := range []string{a, b} {
		if h.gpu(tenant) != "100.000000" || h.cash(tenant) != "999.700000" {
			t.Errorf("tenant %s: gpu %s cash %s, want 100 / 999.7", tenant[:8], h.gpu(tenant), h.cash(tenant))
		}
	}
}

// TestCashAppendOnly checks the database itself refuses edits to cash rows and settlement records.
func TestCashAppendOnly(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "5", "")
	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, q := range []string{
		`UPDATE cash_transactions SET amount = 1000000 WHERE tenant_id = '` + tenant + `'`,
		`DELETE FROM cash_transactions WHERE tenant_id = '` + tenant + `'`,
		`UPDATE trade_settlements SET notional = 0`,
		`INSERT INTO cash_balances (balance_id, tenant_id, currency, is_paper) VALUES ('` + uuid.NewString() + `', '` + tenant + `', 'USD', false)`,
	} {
		if _, err := pool.Exec(context.Background(), q); err == nil || (!strings.Contains(err.Error(), "append-only") && !strings.Contains(err.Error(), "check")) {
			t.Errorf("%q was not refused: %v", q, err)
		}
	}
}
