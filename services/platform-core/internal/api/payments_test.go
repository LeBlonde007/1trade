package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/platform-core/internal/api"
	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// payRig is platform-core with mock Stripe, a stub ledger and wire instructions configured.
type payRig struct {
	*teamRig
	booker *stubBooker
}

// newPayRig skips without DATABASE_URL.
func newPayRig(t *testing.T, wire bool) *payRig {
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
	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, ServiceToken: svcToken, StripeWebhookSecret: webhookSecret,
		BillingAutoSettle: true, TokenTTL: 3600_000_000_000}
	if wire {
		cfg.Wire = config.WireInstructions{BankName: "First Test Bank", AccountName: "1Trade Inc.", AccountNumber: "000123456789", RoutingNumber: "021000021", SWIFT: "FTBKUS33"}
	}
	b := &stubBooker{}
	rg := &payRig{teamRig: &teamRig{t: t, ledger: &fakeBudgetMover{}}, booker: b}
	rg.srv = httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, b))
	t.Cleanup(rg.srv.Close)
	return rg
}

// bookings returns the purchase bookings (trial grants at signup are filtered out).
func (rg *payRig) bookings() []billing.PurchaseBooking {
	var out []billing.PurchaseBooking
	for _, c := range rg.booker.purchases() {
		if strings.HasPrefix(c.IdempotencyKey, "purchase:") {
			out = append(out, c)
		}
	}
	return out
}

// event posts a signed Stripe event for a checkout session.
func (rg *payRig) event(typ, session, paymentStatus string) int {
	rg.t.Helper()
	obj := map[string]any{"id": session}
	if paymentStatus != "" {
		obj["payment_status"] = paymentStatus
	}
	payload, _ := json.Marshal(map[string]any{"id": "evt_" + uuid.NewString(), "type": typ, "data": map[string]any{"object": obj}})
	ts := time.Now().Unix()
	req, _ := http.NewRequest("POST", rg.srv.URL+"/v1/billing/webhook/stripe", strings.NewReader(string(payload)))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s", ts, domain.SignStripePayload(payload, ts, webhookSecret)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		rg.t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// status reads one purchase's status from the history.
func (rg *payRig) status(tok, id string) map[string]any {
	rg.t.Helper()
	_, out := rg.do("GET", "/v1/billing/purchases", tok, nil, nil)
	for _, p := range out["purchases"].([]any) {
		if p.(map[string]any)["id"] == id {
			return p.(map[string]any)
		}
	}
	rg.t.Fatalf("purchase %s not listed", id)
	return nil
}

// TestACHSettlesOnlyWhenTheBankDoes: an ACH checkout books nothing at checkout or when the debit
// starts; it books once when the debit clears; a returned debit books nothing, ever.
func TestACHSettlesOnlyWhenTheBankDoes(t *testing.T) {
	rg := newPayRig(t, false)
	tok, _ := rg.signup()
	code, co := rg.do("POST", "/v1/billing/checkout", tok, map[string]string{"amount": "100000", "credit_type": "text", "method": "ach"}, nil)
	if code != 200 || co["settled"] != false || co["method"] != "ach" {
		t.Fatalf("ach checkout: %d %v", code, co)
	}
	id := co["purchase_id"].(string)
	session := "cs_mock_" + id
	if rg.event("checkout.session.completed", session, "unpaid") != 200 || len(rg.bookings()) != 0 {
		t.Fatal("credits booked when the debit only started")
	}
	if s := rg.status(tok, id); s["status"] != "processing" || s["method"] != "ach" || s["amount_usd"] != "121.00" {
		t.Fatalf("after checkout: %v", s)
	}
	rg.event("checkout.session.async_payment_succeeded", session, "paid")
	rg.event("checkout.session.async_payment_succeeded", session, "paid") // Stripe retries
	b := rg.bookings()
	if len(b) != 2 || b[0].IdempotencyKey != "purchase:"+id || b[1].IdempotencyKey != b[0].IdempotencyKey || b[0].Amount != "100000.000000" {
		t.Fatalf("bookings: %+v", b)
	}
	if rg.status(tok, id)["status"] != "paid" {
		t.Fatal("not paid")
	}

	// A returned debit.
	_, co = rg.do("POST", "/v1/billing/checkout", tok, map[string]string{"amount": "5000", "credit_type": "text", "method": "ach"}, nil)
	id2 := co["purchase_id"].(string)
	rg.event("checkout.session.completed", "cs_mock_"+id2, "unpaid")
	rg.event("checkout.session.async_payment_failed", "cs_mock_"+id2, "unpaid")
	rg.event("checkout.session.async_payment_succeeded", "cs_mock_"+id2, "paid") // must not revive it
	if s := rg.status(tok, id2); s["status"] != "failed" || s["failure_reason"] == nil {
		t.Fatalf("returned debit: %v", s)
	}
	if len(rg.bookings()) != 2 {
		t.Fatalf("a failed debit booked credits: %+v", rg.bookings())
	}
	// A completed event without proof of payment books nothing.
	_, co = rg.do("POST", "/v1/billing/checkout", tok, map[string]string{"amount": "5000", "credit_type": "text", "method": "ach"}, nil)
	rg.event("checkout.session.completed", "cs_mock_"+co["purchase_id"].(string), "")
	if len(rg.bookings()) != 2 {
		t.Fatal("booked without payment_status=paid")
	}
}

// TestUSDOnlyAndMethods: US dollars only; card or ACH at checkout; card still auto-settles in dev.
func TestUSDOnlyAndMethods(t *testing.T) {
	rg := newPayRig(t, false)
	tok, _ := rg.signup()
	for name, body := range map[string]map[string]string{
		"jpy":    {"amount": "100", "credit_type": "text", "currency": "jpy"},
		"eur":    {"amount": "100", "credit_type": "text", "currency": "eur"},
		"method": {"amount": "100", "credit_type": "text", "method": "crypto"},
		"wire":   {"amount": "100", "credit_type": "text", "method": "wire"},
	} {
		if code, _ := rg.do("POST", "/v1/billing/checkout", tok, body, nil); code != 422 {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	code, co := rg.do("POST", "/v1/billing/checkout", tok, map[string]string{"amount": "100", "credit_type": "text"}, nil)
	if code != 200 || co["settled"] != true || co["method"] != "card" {
		t.Fatalf("default card checkout: %d %v", code, co)
	}
	// A late failure event never turns a paid purchase into a failed one.
	rg.event("checkout.session.async_payment_failed", "cs_mock_"+co["purchase_id"].(string), "unpaid")
	if s := rg.status(tok, co["purchase_id"].(string)); s["status"] != "paid" {
		t.Fatalf("a paid purchase was marked failed: %v", s)
	}
	if code, _ := rg.do("POST", "/v1/billing/wires", tok, map[string]string{"amount": "1000", "credit_type": "gpu_h100"}, nil); code != 503 {
		t.Fatalf("wire without bank details: %d", code)
	}
}

// TestWireTransfer: an invoice with our bank details and a unique reference; treasury's receipt must
// match to the cent; credits are booked once; roles and KYC gate it.
func TestWireTransfer(t *testing.T) {
	rg := newPayRig(t, true)
	admin, ac := rg.signup()
	if code, out := rg.do("POST", "/v1/billing/wires", admin, map[string]string{"amount": "300", "credit_type": "gpu_h100"}, nil); code != 422 || out["code"] != "below_wire_minimum" {
		t.Fatalf("below the minimum: %d %v", code, out)
	}
	code, inv := rg.do("POST", "/v1/billing/wires", admin, map[string]string{"amount": "1000", "credit_type": "gpu_h100"}, nil)
	if code != 201 || inv["amount_usd"] != "2990.00" || inv["currency"] != "usd" || !strings.HasPrefix(inv["reference"].(string), "1T-") ||
		inv["instructions"].(map[string]any)["routing_number"] != "021000021" {
		t.Fatalf("invoice: %d %v", code, inv)
	}
	id := inv["purchase_id"].(string)
	if len(rg.bookings()) != 0 {
		t.Fatal("booked before the money arrived")
	}
	svc := svcToken
	recv := func(amount, ref string, tok string) (int, map[string]any) {
		return rg.do("POST", "/v1/billing/wires/"+id+"/received", tok, map[string]string{"amount_usd": amount, "bank_reference": ref}, nil)
	}
	if code, _ := recv("2990.00", "FED123", admin); code != 401 {
		t.Fatalf("a tenant recording its own wire: %d", code)
	}
	for _, amt := range []string{"2989.99", "2990.01", "3000", "29.90"} {
		if code, out := recv(amt, "FED123", svc); code != 409 || out["code"] != "amount_mismatch" {
			t.Fatalf("amount %s: %d %v", amt, code, out)
		}
	}
	if code, _ := recv("2990.001", "FED123", svc); code != 422 {
		t.Fatalf("sub-cent amount: %d", code)
	}
	if code, out := recv("2990", "FED123", svc); code != 200 || out["status"] != "paid" {
		t.Fatalf("receipt: %d %v", code, out)
	}
	if code, _ := recv("2990.00", "FED123", svc); code != 200 { // replay: re-books idempotently
		t.Fatalf("replayed receipt: %d", code)
	}
	if code, _ := recv("2990.00", "FED999", svc); code != 409 {
		t.Fatalf("a second, different receipt: %d", code)
	}
	b := rg.bookings()
	if len(b) != 2 || b[0].IdempotencyKey != "purchase:"+id || b[1].IdempotencyKey != b[0].IdempotencyKey || b[0].Amount != "1000.000000" || b[0].TenantID != ac.TenantID {
		t.Fatalf("bookings: %+v", b)
	}
	if s := rg.status(admin, id); s["method"] != "wire" || s["status"] != "paid" || s["wire_reference"] != inv["reference"] {
		t.Fatalf("history: %v", s)
	}
	// Engineers do not buy by wire.
	eng, _ := rg.accept(rg.invite(admin, "eng+"+uuid.NewString()[:6]+"@acme.ai", []string{"engineer"}, nil))
	if code, _ := rg.do("POST", "/v1/billing/wires", eng, map[string]string{"amount": "1000", "credit_type": "gpu_h100"}, nil); code != 403 {
		t.Fatalf("engineer wire: %d", code)
	}
	// A real-money tenant without KYC is refused.
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	real, rc := rg.signup()
	_, err = db.Exec(context.Background(), `UPDATE tenants SET is_paper=false WHERE id=$1`, rc.TenantID)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	_ = real
	c := domain.Claims{TenantID: rc.TenantID, Roles: []domain.Role{domain.RoleAdmin}, IsPaper: false}
	realTok, _ := domain.IssueToken(jwtSecret, rc.Subject, c, time.Hour)
	if code, out := rg.do("POST", "/v1/billing/wires", realTok, map[string]string{"amount": "1000", "credit_type": "gpu_h100"}, nil); code != 403 || out["code"] != "kyc_required" {
		t.Fatalf("real money without KYC: %d %v", code, out)
	}
}
