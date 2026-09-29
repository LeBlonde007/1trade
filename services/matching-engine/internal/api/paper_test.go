package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/trade1/matching-engine/internal/api"
	"github.com/trade1/matching-engine/internal/auth"
	"github.com/trade1/matching-engine/internal/config"
	"github.com/trade1/matching-engine/internal/domain"
	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/liquidity"
	"github.com/trade1/matching-engine/internal/venue"
)

const (
	secret = "paper-test-secret-paper-test-secret"
	liqID  = "00000000-0000-4000-8000-00000000b07e"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

// rig is the API over a paper venue whose books the liquidity bot has quoted (H100 around 3.00).
type rig struct {
	t   *testing.T
	srv *httptest.Server
	v   *venue.Venue
}

// newRig builds the rig.
func newRig(t *testing.T) *rig {
	t.Helper()
	v := venue.New(engine.New(engine.Config{}), nil, nil, liqID)
	bot := &liquidity.Bot{Venue: v, Tenant: liqID, Now: func() time.Time { return now },
		Mid: func(p domain.Product, _ time.Time) float64 {
			if p.ID == "H100-SPOT" {
				return 3.00
			}
			return p.Reference
		}}
	bot.Step(context.Background())
	s := api.NewPaper(config.Config{MethodologyURL: "/methodology"}, auth.NewResolver(secret), v, liqID,
		func() time.Time { return now })
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	return &rig{t: t, srv: srv, v: v}
}

// token signs a tenant JWT.
func token(t *testing.T, tenant string, paper bool) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenant, "is_paper": paper, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// call sends a request and decodes the JSON answer.
func (rg *rig) call(method, path, tok, key string, body any) (int, map[string]any) {
	rg.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, rg.srv.URL+path, rdr)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
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

// buy is a marketable H100 buy of qty.
func buy(qty string) map[string]any {
	return map[string]any{"product_id": "H100-SPOT", "side": "buy", "order_type": "limit", "quantity": qty,
		"limit_price": "3.020000", "time_in_force": "gtc"}
}

// TestRealMoneyStillPaused checks a real-money principal cannot place or cancel: 503 EXCHANGE_PAUSED,
// and nothing reaches the engine.
func TestRealMoneyStillPaused(t *testing.T) {
	rg := newRig(t)
	real := token(t, "11111111-1111-4111-8111-111111111111", false)
	if code, body := rg.call("POST", "/v1/trading/orders", real, "k1", buy("1")); code != 503 || body["code"] != "EXCHANGE_PAUSED" {
		t.Fatalf("real-money order = %d %v, want 503 EXCHANGE_PAUSED", code, body)
	}
	if code, _ := rg.call("DELETE", "/v1/trading/orders/00000000-0000-4000-8000-000000000001", real, "", nil); code != 503 {
		t.Fatalf("real-money cancel = %d, want 503", code)
	}
	if n := len(rg.v.OpenOrders("11111111-1111-4111-8111-111111111111")); n != 0 {
		t.Fatalf("a real-money order reached the engine: %d open", n)
	}
	if code, _ := rg.call("POST", "/v1/trading/orders", "", "k1", buy("1")); code != 401 {
		t.Fatalf("anonymous order = %d, want 401", code)
	}
}

// TestPaperOrderFillsAndIsIdempotent places a marketable paper buy: it fills against the liquidity
// ladder, shows up in fills and positions, a retry returns the same order, and a reused key with a
// different order is refused.
func TestPaperOrderFillsAndIsIdempotent(t *testing.T) {
	rg := newRig(t)
	alice := token(t, "22222222-2222-4222-8222-222222222222", true)
	code, o := rg.call("POST", "/v1/trading/orders", alice, "k-alice-1", buy("2"))
	if code != 201 || o["state"] != "filled" || o["avg_fill_price"] != "3.020000" || o["is_paper"] != true {
		t.Fatalf("paper buy = %d %v, want 201 filled at 3.02", code, o)
	}
	if code, again := rg.call("POST", "/v1/trading/orders", alice, "k-alice-1", buy("2")); code != 200 || again["order_id"] != o["order_id"] {
		t.Fatalf("retry = %d %v, want 200 with the same order", code, again)
	}
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-alice-1", buy("3")); code != 409 {
		t.Fatalf("reused key with a different order = %d, want 409", code)
	}
	dayTIF := buy("2")
	dayTIF["time_in_force"] = "day"
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-alice-1", dayTIF); code != 409 {
		t.Fatalf("reused key with a different time in force = %d, want 409", code)
	}
	noTIF := buy("2")
	delete(noTIF, "time_in_force")
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-alice-1", noTIF); code != 200 {
		t.Fatalf("retry omitting the default time in force = %d, want 200", code)
	}
	_, fills := rg.call("GET", "/v1/trading/fills", alice, "", nil)
	if f := fills["fills"].([]any); len(f) != 1 || f[0].(map[string]any)["liquidity"] != "taker" {
		t.Fatalf("fills = %v", fills)
	}
	_, pos := rg.call("GET", "/v1/trading/positions", alice, "", nil)
	p := pos["positions"].([]any)
	if len(p) != 1 || p[0].(map[string]any)["net_quantity"] != "2.000000" || p[0].(map[string]any)["side"] != "long" {
		t.Fatalf("positions = %v", pos)
	}
	_, tape := rg.call("GET", "/v1/trading/products/H100-SPOT/trades", "", "", nil)
	if len(tape["trades"].([]any)) != 1 {
		t.Fatalf("the trade is not on the tape: %v", tape)
	}
}

// TestBodyCannotChooseRealMoney checks is_paper cannot be smuggled in the body: unknown fields are
// refused, so the only source of is_paper is the verified token.
func TestBodyCannotChooseRealMoney(t *testing.T) {
	rg := newRig(t)
	alice := token(t, "22222222-2222-4222-8222-222222222222", true)
	body := buy("1")
	body["is_paper"] = false
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-smuggle", body); code != 422 {
		t.Fatalf("body with is_paper = %d, want 422", code)
	}
}

// TestCancelIsTenantScoped rests an order, checks another tenant cannot see or cancel it, then
// cancels it and checks a second cancel is a 409.
func TestCancelIsTenantScoped(t *testing.T) {
	rg := newRig(t)
	alice := token(t, "22222222-2222-4222-8222-222222222222", true)
	eve := token(t, "33333333-3333-4333-8333-333333333333", true)
	rest := map[string]any{"product_id": "H100-SPOT", "side": "buy", "order_type": "limit", "quantity": "1", "limit_price": "2.500000"}
	_, o := rg.call("POST", "/v1/trading/orders", alice, "k-rest", rest)
	if o["state"] != "open" {
		t.Fatalf("resting order state = %v", o["state"])
	}
	id := o["order_id"].(string)
	if code, _ := rg.call("DELETE", "/v1/trading/orders/"+id, eve, "", nil); code != 404 {
		t.Fatalf("another tenant's cancel = %d, want 404", code)
	}
	if _, list := rg.call("GET", "/v1/trading/orders?state=open", eve, "", nil); len(list["orders"].([]any)) != 0 {
		t.Fatal("another tenant sees the order")
	}
	if code, c := rg.call("DELETE", "/v1/trading/orders/"+id, alice, "", nil); code != 200 || c["state"] != "cancelled" {
		t.Fatalf("cancel = %d %v", code, c)
	}
	if code, _ := rg.call("DELETE", "/v1/trading/orders/"+id, alice, "", nil); code != 409 {
		t.Fatalf("second cancel = %d, want 409", code)
	}
}

// TestLiquidityAccountCannotUseTheAPI checks the liquidity account's id cannot trade through HTTP.
func TestLiquidityAccountCannotUseTheAPI(t *testing.T) {
	rg := newRig(t)
	if code, _ := rg.call("POST", "/v1/trading/orders", token(t, liqID, true), "k", buy("1")); code != 403 {
		t.Fatalf("liquidity account order = %d, want 403", code)
	}
}

// TestValidation checks a missing key, a bad quantity and an off-tick price are refused.
func TestValidation(t *testing.T) {
	rg := newRig(t)
	alice := token(t, "22222222-2222-4222-8222-222222222222", true)
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "", buy("1")); code != 400 {
		t.Fatalf("no Idempotency-Key = %d, want 400", code)
	}
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-q", buy("-1")); code != 422 {
		t.Fatalf("negative quantity = %d, want 422", code)
	}
	off := buy("1")
	off["limit_price"] = "3.001000" // H100 ticks in 0.01
	if code, _ := rg.call("POST", "/v1/trading/orders", alice, "k-tick", off); code != 422 {
		t.Fatalf("off-tick price = %d, want 422", code)
	}
}

// TestMarketDataIsTheRealBook checks the book, quote, candles and status come from the paper venue.
func TestMarketDataIsTheRealBook(t *testing.T) {
	rg := newRig(t)
	_, book := rg.call("GET", "/v1/trading/products/H100-SPOT/orderbook", "", "", nil)
	bids, asks := book["bids"].([]any), book["asks"].([]any)
	if len(bids) != 3 || len(asks) != 3 || bids[0].(map[string]any)["price"] != "2.980000" {
		t.Fatalf("book = %v", book)
	}
	if _, q := rg.call("GET", "/v1/trading/products/H100-SPOT/quote", "", "", nil); q["source"] != "book" || q["ask"] != "3.020000" {
		t.Fatalf("quote = %v", q)
	}
	_, c := rg.call("GET", "/v1/trading/products/H100-SPOT/candles?interval=1m&limit=5", "", "", nil)
	for _, bar := range c["candles"].([]any) {
		if bar.(map[string]any)["source"] != "reference" {
			t.Fatalf("a bar with no trades is not labelled reference: %v", bar)
		}
	}
	if _, prods := rg.call("GET", "/v1/trading/products", "", "", nil); prods["exchange_status"].(map[string]any)["state"] != "paper" {
		t.Fatalf("status = %v", prods["exchange_status"])
	}
}
