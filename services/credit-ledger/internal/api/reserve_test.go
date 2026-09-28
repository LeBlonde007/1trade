package api_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// reserveBody builds a ReserveRequest.
func reserveBody(orderID, tenant, kind, asset, amount string) map[string]any {
	return map[string]any{"order_id": orderID, "tenant_id": tenant, "sub_account_id": nil,
		"asset_kind": kind, "asset": asset, "amount": amount, "is_paper": true}
}

// reserve calls /v1/credits/reserve as the engine and returns status + error code.
func (h *harness) reserve(orderID, tenant, kind, asset, amount string) (int, string) {
	h.t.Helper()
	var e map[string]any
	code := h.do("POST", "/v1/credits/reserve", engine(orderID), reserveBody(orderID, tenant, kind, asset, amount), &e)
	c, _ := e["code"].(string)
	return code, c
}

// release calls /v1/credits/release as the engine and returns status + the reservation.
func (h *harness) release(orderID string) (int, map[string]any) {
	h.t.Helper()
	var out map[string]any
	code := h.do("POST", "/v1/credits/release", map[string]string{"Authorization": "Bearer " + settleToken},
		map[string]any{"order_id": orderID}, &out)
	return code, out
}

// locked returns the locked_amount of a tenant's paper USD or gpu_h100 balance.
func (h *harness) locked(tenant, asset string) string {
	h.t.Helper()
	path := "/v1/credits/balances"
	if asset == "USD" {
		path = "/v1/credits/cash/balances"
	}
	var out struct {
		Balances []map[string]any `json:"balances"`
	}
	h.do("GET", path, map[string]string{"X-Dev-Tenant": tenant}, nil, &out)
	for _, b := range out.Balances {
		if b["credit_type"] == asset || b["currency"] == asset {
			return b["locked_amount"].(string)
		}
	}
	return "0.000000"
}

// tradeFor is tradeBody with the buyer's and seller's order ids fixed (so the legs find their
// reservations).
func tradeFor(tradeID, buyer, buyOrder, seller, sellOrder, price, qty, buyerFee, sellerFee string) map[string]any {
	b := tradeBody(tradeID, buyer, seller, price, qty, buyerFee, sellerFee)
	b["buyer"].(map[string]any)["order_id"] = buyOrder
	b["seller"].(map[string]any)["order_id"] = sellOrder
	return b
}

// TestReservedValueCannotBeSpentElsewhere checks available = balance − locked is enforced for
// reservations and for ordinary debits (inference consumption), not just for trading.
func TestReservedValueCannotBeSpentElsewhere(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "100", "10")

	if code, _ := h.reserve(uuid.NewString(), tenant, "credit", "gpu_h100", "8"); code != 200 {
		t.Fatalf("reserve 8 of 10 credits: %d", code)
	}
	if got := h.locked(tenant, "gpu_h100"); got != "8.000000" {
		t.Fatalf("locked = %s, want 8", got)
	}
	if code, c := h.reserve(uuid.NewString(), tenant, "credit", "gpu_h100", "3"); code != 402 || c != "INSUFFICIENT_CREDIT" {
		t.Errorf("reserve beyond available: %d %s, want 402 INSUFFICIENT_CREDIT", code, c)
	}
	// Balance is 10, but only 2 are available: a 3-credit consumption debit must be refused.
	svc := map[string]string{"Authorization": "Bearer " + serviceToken, "Idempotency-Key": "use-" + uuid.NewString()}
	var e map[string]string
	if code := h.do("POST", "/v1/credits/debit", svc, map[string]any{
		"tenant_id": tenant, "credit_type": "gpu_h100", "amount": "3", "usage_event_id": uuid.NewString(), "is_paper": true,
	}, &e); code != 402 {
		t.Errorf("debit of reserved credits: %d %v, want 402", code, e)
	}
	svc["Idempotency-Key"] = "use-" + uuid.NewString()
	if code := h.do("POST", "/v1/credits/debit", svc, map[string]any{
		"tenant_id": tenant, "credit_type": "gpu_h100", "amount": "2", "usage_event_id": uuid.NewString(), "is_paper": true,
	}, nil); code != 200 {
		t.Errorf("debit within available: %d, want 200", code)
	}
	if code, c := h.reserve(uuid.NewString(), tenant, "cash", "USD", "100.000001"); code != 402 || c != "INSUFFICIENT_CASH" {
		t.Errorf("cash reserve beyond balance: %d %s", code, c)
	}
}

// TestSettlementConsumesAndReleaseFrees walks the full life of a price-improved trade: the buyer
// reserves at its 3.00 limit plus the taker fee, fills at 2.99, and the unspent remainder is freed
// only by release.
func TestSettlementConsumesAndReleaseFrees(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "1000", "")
	h.fund(seller, "", "10")
	buyOrder, sellOrder := uuid.NewString(), uuid.NewString()

	// 10 @ limit 3.00 → 30.00 notional + 1% taker fee 0.30 = 30.30 reserved.
	if code, _ := h.reserve(buyOrder, buyer, "cash", "USD", "30.30"); code != 200 {
		t.Fatalf("buyer reserve: %d", code)
	}
	if code, _ := h.reserve(sellOrder, seller, "credit", "gpu_h100", "10"); code != 200 {
		t.Fatalf("seller reserve: %d", code)
	}
	id := uuid.NewString()
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), tradeFor(id, buyer, buyOrder, seller, sellOrder, "2.99", "10", "0.299", "0.1495"), nil); code != 200 {
		t.Fatalf("settle: %d", code)
	}
	// Buyer spent 29.90 + 0.299 = 30.199 out of the 30.30 reserved; 0.101 stays locked until release.
	checks := map[string][2]string{
		"buyer cash":    {h.cash(buyer), "969.801000"},
		"buyer locked":  {h.locked(buyer, "USD"), "0.101000"},
		"seller gpu":    {h.gpu(seller), "0.000000"},
		"seller locked": {h.locked(seller, "gpu_h100"), "0.000000"},
	}
	for k, v := range checks {
		if v[0] != v[1] {
			t.Errorf("%s = %s, want %s", k, v[0], v[1])
		}
	}
	code, r := h.release(buyOrder)
	if code != 200 || r["state"] != "released" || r["remaining"] != "0.000000" {
		t.Fatalf("release: %d %v", code, r)
	}
	if got := h.locked(buyer, "USD"); got != "0.000000" {
		t.Errorf("buyer locked after release = %s, want 0", got)
	}
	if code, r2 := h.release(buyOrder); code != 200 || r2["state"] != "released" {
		t.Errorf("second release not idempotent: %d %v", code, r2)
	}
	if code, _ := h.release(uuid.NewString()); code != 404 {
		t.Errorf("release unknown order: %d, want 404", code)
	}
	if got := h.cash(buyer); got != "969.801000" {
		t.Errorf("release moved cash: %s", got)
	}
}

// TestSettlementBeyondReservationIsRefused checks a leg larger than its order's reservation is 402 and
// writes nothing — the engine reserved wrongly, and the ledger must not let it spend unreserved value.
func TestSettlementBeyondReservationIsRefused(t *testing.T) {
	h := newHarness(t)
	buyer, seller := uuid.NewString(), uuid.NewString()
	h.fund(buyer, "1000", "")
	h.fund(seller, "", "50")
	buyOrder, sellOrder := uuid.NewString(), uuid.NewString()
	h.reserve(buyOrder, buyer, "cash", "USD", "100")
	h.reserve(sellOrder, seller, "credit", "gpu_h100", "5") // reserved 5, trade is 10
	id := uuid.NewString()
	var e map[string]string
	if code := h.do("POST", "/v1/credits/settle-trade", engine(id), tradeFor(id, buyer, buyOrder, seller, sellOrder, "1", "10", "0", "0"), &e); code != 402 || e["code"] != "INSUFFICIENT_CREDIT" {
		t.Fatalf("over-reservation settle: %d %v", code, e)
	}
	if h.gpu(seller) != "50.000000" || h.locked(seller, "gpu_h100") != "5.000000" || h.cash(buyer) != "1000.000000" || h.locked(buyer, "USD") != "100.000000" {
		t.Error("a refused settlement changed balances or reservations")
	}
}

// TestReserveIdempotencyAndAuth checks replay, conflict, the engine-only caller, and real money.
func TestReserveIdempotencyAndAuth(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "100", "")
	order := uuid.NewString()
	if code, _ := h.reserve(order, tenant, "cash", "USD", "10"); code != 200 {
		t.Fatal(code)
	}
	if code, _ := h.reserve(order, tenant, "cash", "USD", "10"); code != 200 || h.locked(tenant, "USD") != "10.000000" {
		t.Errorf("replay: %d locked %s, want 200 and 10 (not 20)", code, h.locked(tenant, "USD"))
	}
	if code, c := h.reserve(order, tenant, "cash", "USD", "11"); code != 409 || c != "IDEMPOTENCY_CONFLICT" {
		t.Errorf("conflicting replay: %d %s", code, c)
	}
	if code := h.do("POST", "/v1/credits/reserve", map[string]string{"Authorization": "Bearer " + serviceToken, "Idempotency-Key": order},
		reserveBody(order, tenant, "cash", "USD", "10"), nil); code != 403 {
		t.Errorf("shared token reserve: %d, want 403", code)
	}
	if code := h.do("POST", "/v1/credits/release", map[string]string{"Authorization": "Bearer " + serviceToken},
		map[string]any{"order_id": order}, nil); code != 403 {
		t.Errorf("shared token release: %d, want 403", code)
	}
	real := reserveBody(uuid.NewString(), tenant, "cash", "USD", "1")
	real["is_paper"] = false
	var e map[string]string
	if code := h.do("POST", "/v1/credits/reserve", engine(real["order_id"].(string)), real, &e); code != 422 || e["code"] != "REAL_MONEY_DISABLED" {
		t.Errorf("real-money reserve: %d %v", code, e)
	}
	if code, _ := h.reserve(uuid.NewString(), tenant, "cash", "JPY", "1"); code != 422 {
		t.Errorf("unknown currency reserve: %d, want 422", code)
	}
	if code, _ := h.reserve(uuid.NewString(), tenant, "credit", "gold", "1"); code != 422 {
		t.Errorf("unknown credit type reserve: %d, want 422", code)
	}
}

// TestConcurrentReservesNeverOvercommit fires 20 reserves of 1000 against 10000 available at once:
// exactly 10 may succeed, and locked must equal exactly what was granted.
func TestConcurrentReservesNeverOvercommit(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "10000", "")
	var ok atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code, _ := h.reserve(uuid.NewString(), tenant, "cash", "USD", "1000"); code == 200 {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 10 || h.locked(tenant, "USD") != "10000.000000" {
		t.Errorf("granted %d reserves, locked %s; want 10 and 10000", ok.Load(), h.locked(tenant, "USD"))
	}
}

// TestLockedBackstop checks the database itself refuses a balance below its locked amount.
func TestLockedBackstop(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "5", "")
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(context.Background(), `UPDATE cash_balances SET locked_amount = 6 WHERE tenant_id = $1`, tenant); err == nil {
		t.Error("locked_amount above balance was accepted")
	}
}

// TestReplayOfReleasedReservationIsClosed: v1.3 — a released reservation locks nothing, so a replay
// is 409 RESERVATION_CLOSED (not a 200 the engine could mistake for funds), and nothing is re-locked.
func TestReplayOfReleasedReservationIsClosed(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "100", "")
	order := uuid.NewString()
	var res map[string]any
	if code := h.do("POST", "/v1/credits/reserve", engine(order), reserveBody(order, tenant, "cash", "USD", "10"), &res); code != 200 || res["created_at"] == "" || res["created_at"] == nil {
		t.Fatalf("reserve: %d %v", code, res)
	}
	if code, _ := h.release(order); code != 200 {
		t.Fatal(code)
	}
	if code, c := h.reserve(order, tenant, "cash", "USD", "10"); code != 409 || c != "RESERVATION_CLOSED" {
		t.Fatalf("replay of a released reservation: %d %s, want 409 RESERVATION_CLOSED", code, c)
	}
	if got := h.locked(tenant, "USD"); got != "0.000000" {
		t.Fatalf("locked = %s after the refused replay", got)
	}
}

// TestReleaseReasonIsAudited: the optional reason lands on the release audit event; unknown reasons
// are refused.
func TestReleaseReasonIsAudited(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "100", "")
	order := uuid.NewString()
	if code, _ := h.reserve(order, tenant, "cash", "USD", "5"); code != 200 {
		t.Fatal(code)
	}
	settle := map[string]string{"Authorization": "Bearer " + settleToken}
	if code := h.do("POST", "/v1/credits/release", settle, map[string]any{"order_id": order, "reason": "because"}, nil); code != 422 {
		t.Fatalf("unknown reason: %d, want 422", code)
	}
	if code := h.do("POST", "/v1/credits/release", settle, map[string]any{"order_id": order, "reason": "orphaned"}, nil); code != 200 {
		t.Fatalf("release: %d", code)
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var ref string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(reference,'') FROM reservation_events WHERE order_id=$1 AND kind='release'`, order).Scan(&ref); err != nil || ref != "orphaned" {
		t.Fatalf("release audit reference = %q (%v), want orphaned", ref, err)
	}
}

// TestListOpenReservations: engine-only, open only, age on the ledger clock, keyset pages that
// neither skip nor repeat.
func TestListOpenReservations(t *testing.T) {
	h := newHarness(t)
	tenant := uuid.NewString()
	h.fund(tenant, "100", "")
	mine := make([]string, 0, 3)
	for range 3 {
		id := uuid.NewString()
		if code, _ := h.reserve(id, tenant, "cash", "USD", "1"); code != 200 {
			t.Fatal(code)
		}
		mine = append(mine, id)
	}
	gone := uuid.NewString()
	h.reserve(gone, tenant, "cash", "USD", "1")
	h.release(gone)

	settle := map[string]string{"Authorization": "Bearer " + settleToken}
	type page struct {
		Data []struct {
			OrderID string `json:"order_id"`
			State   string `json:"state"`
		} `json:"data"`
		NextCursor *string `json:"next_cursor"`
	}
	seen := map[string]int{}
	last := ""
	for after, n := "", 0; ; n++ {
		var p page
		path := "/v1/credits/reservations?state=open&limit=2"
		if after != "" {
			path += "&after=" + after
		}
		if code := h.do("GET", path, settle, nil, &p); code != 200 {
			t.Fatalf("list: %d", code)
		}
		for _, r := range p.Data {
			if r.State != "open" || r.OrderID <= last {
				t.Fatalf("page %d: %+v after %s (closed, or out of order)", n, r, last)
			}
			last = r.OrderID
			seen[r.OrderID]++
		}
		if p.NextCursor == nil {
			break
		}
		after = *p.NextCursor
	}
	for _, id := range mine {
		if seen[id] != 1 {
			t.Fatalf("open reservation %s listed %d times", id, seen[id])
		}
	}
	if seen[gone] != 0 {
		t.Fatal("a released reservation was listed")
	}
	var young page
	h.do("GET", "/v1/credits/reservations?state=open&limit=500&min_age_seconds=3600", settle, nil, &young)
	for _, r := range young.Data {
		for _, id := range mine {
			if r.OrderID == id {
				t.Fatal("a reservation younger than min_age_seconds was listed")
			}
		}
	}
	for _, c := range []struct {
		path    string
		headers map[string]string
		want    int
	}{
		{"/v1/credits/reservations?state=open", map[string]string{"Authorization": "Bearer " + serviceToken}, 403},
		{"/v1/credits/reservations?state=open", map[string]string{"X-Dev-Tenant": tenant}, 403},
		{"/v1/credits/reservations", settle, 422},
		{"/v1/credits/reservations?state=released", settle, 422},
		{"/v1/credits/reservations?state=open&limit=0", settle, 422},
		{"/v1/credits/reservations?state=open&limit=501", settle, 422},
		{"/v1/credits/reservations?state=open&min_age_seconds=-1", settle, 422},
		{"/v1/credits/reservations?state=open&after=x'--", settle, 422},
	} {
		if code := h.do("GET", c.path, c.headers, nil, nil); code != c.want {
			t.Errorf("%s: %d, want %d", c.path, code, c.want)
		}
	}
}
