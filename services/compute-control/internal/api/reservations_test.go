package api

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/trade1/compute-control/internal/auth"
	"github.com/trade1/compute-control/internal/config"
	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/instance"
	"github.com/trade1/compute-control/internal/ledger"
	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/reserve"
	"github.com/trade1/compute-control/internal/scheduler"
)

// fakeLedger is an in-memory ledger: balances per (tenant, type, paper), idempotent on the usage id.
type fakeLedger struct {
	mu    sync.Mutex
	bal   map[string]*big.Rat
	seen  map[string]string
	calls int
	down  bool
}

// balKey keys a balance.
func balKey(tenant, ct string, paper bool) string { return fmt.Sprintf("%s|%s|%v", tenant, ct, paper) }

// Debit books a debit like the ledger: 402 when short, the original tx on a replay.
func (f *fakeLedger) Debit(_ context.Context, d ledger.Debit) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.down {
		return "", errors.New("connection refused")
	}
	if tx, ok := f.seen[d.UsageEventID]; ok {
		return tx, nil
	}
	amt, _ := new(big.Rat).SetString(d.Amount)
	b := f.bal[balKey(d.TenantID, d.CreditType, d.IsPaper)]
	if b == nil || b.Cmp(amt) < 0 {
		return "", ledger.ErrInsufficientCredit
	}
	b.Sub(b, amt)
	tx := fmt.Sprintf("tx-%d", len(f.seen)+1)
	f.seen[d.UsageEventID] = tx
	return tx, nil
}

// balance reads a balance to 6 places.
func (f *fakeLedger) balance(tenant, ct string, paper bool) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b := f.bal[balKey(tenant, ct, paper)]; b != nil {
		return b.FloatString(6)
	}
	return "0.000000"
}

// capture records published usage events.
type capture struct {
	mu  sync.Mutex
	evs []events.ComputeUsage
}

// PublishUsage records u.
func (c *capture) PublishUsage(u events.ComputeUsage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evs = append(c.evs, u)
}

// last returns the latest event for an instance.
func (c *capture) last(instanceID string) (events.ComputeUsage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.evs) - 1; i >= 0; i-- {
		if c.evs[i].InstanceID == instanceID {
			return c.evs[i], true
		}
	}
	return events.ComputeUsage{}, false
}

// reserveRig is compute-control with reservations on a real database, a fake ledger and 16 owned H100s.
type reserveRig struct {
	*supplyRig
	svc    *reserve.Service
	ledger *fakeLedger
	usage  *capture
	mgr    *instance.Manager
	dsn    string
}

// newReserveRig builds it.
func newReserveRig(t *testing.T) *reserveRig {
	t.Helper()
	dsn := supplyDSN(t)
	rg := &reserveRig{supplyRig: &supplyRig{now: time.Now()}, ledger: &fakeLedger{bal: map[string]*big.Rat{}, seen: map[string]string{}}, usage: &capture{}, dsn: dsn}
	rg.pool = pool.New("dc-owned-1", map[string]int{domain.CreditH100: 16})
	rg.sched = scheduler.NewMockWithPool(rg.pool, rg.usage)
	rg.mgr = instance.NewManager(rg.pool, rg.usage)
	rg.s = New(config.Config{Env: "dev", Paper: true}, auth.NewResolver(testSecret, testSvc), rg.sched, rg.mgr)
	svc, err := reserve.Open(context.Background(), dsn, rg.pool, rg.ledger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	rg.svc = svc
	rg.s.EnableReservations(svc)
	return rg
}

// fund gives a tenant paper GPU credits.
func (rg *reserveRig) fund(tenant, ct, amount string) {
	v, _ := new(big.Rat).SetString(amount)
	rg.ledger.bal[balKey(tenant, ct, true)] = v
}

// TestReservationQuote prices exactly: gpus × term hours, less the term discount.
func TestReservationQuote(t *testing.T) {
	rg := newReserveRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000d1", "viewer")
	for _, c := range []struct {
		q                    string
		hours, price, saving string
	}{
		{"gpu_type=gpu_h100&gpus=8&term=1mo", "5840.000000", "4847.200000", "992.800000"},
		{"gpu_type=gpu_h100&gpus=1&term=6mo", "4380.000000", "3197.400000", "1182.600000"},
		{"gpu_type=gpu_h200&gpus=3&term=12mo", "26280.000000", "17607.600000", "8672.400000"},
	} {
		code, out := rg.call("GET", "/v1/compute/reservations/quote?"+c.q, tok, "", nil)
		q, _ := out["quote"].(map[string]any)
		if code != 200 || q["gpu_hours"] != c.hours || q["price_credits"] != c.price || q["saving_credits"] != c.saving {
			t.Errorf("%s: %d %v", c.q, code, out)
		}
	}
	for _, q := range []string{"gpu_type=gpu_a100&gpus=1&term=1mo", "gpu_type=gpu_h100&gpus=0&term=1mo",
		"gpu_type=gpu_h100&gpus=257&term=1mo", "gpu_type=gpu_h100&gpus=1&term=2mo", "gpu_type=gpu_h100&gpus=x&term=1mo"} {
		if code, _ := rg.call("GET", "/v1/compute/reservations/quote?"+q, tok, "", nil); code != 422 {
			t.Errorf("%s: %d, want 422", q, code)
		}
	}
}

// TestReservationPrepaysAndHoldsCapacity: a reservation is paid once, sets GPUs aside that nobody
// else can take, and the owner's instance runs on it with usage billed at zero.
func TestReservationPrepaysAndHoldsCapacity(t *testing.T) {
	rg := newReserveRig(t)
	owner := "00000000-0000-4000-8000-0000000000d2"
	tok := partnerJWT(t, owner, "billing")
	rg.fund(owner, "gpu_h100", "5000")
	body := `{"gpu_type":"gpu_h100","gpus":8,"term":"1mo"}`
	key := map[string]string{"Idempotency-Key": "res-1"}

	code, res := rg.call("POST", "/v1/compute/reservations", tok, body, key)
	if code != 201 || res["state"] != "active" || res["price_credits"] != "4847.200000" || res["ends_at"] == nil {
		t.Fatalf("buy: %d %v", code, res)
	}
	if got := rg.ledger.balance(owner, "gpu_h100", true); got != "152.800000" {
		t.Fatalf("balance %s, want 5000 - 4847.2", got)
	}
	starts, _ := time.Parse(time.RFC3339Nano, res["starts_at"].(string))
	ends, _ := time.Parse(time.RFC3339Nano, res["ends_at"].(string))
	if ends.Sub(starts) != 730*time.Hour {
		t.Fatalf("term %v", ends.Sub(starts))
	}
	if code, again := rg.call("POST", "/v1/compute/reservations", tok, body, key); code != 201 || again["id"] != res["id"] {
		t.Fatalf("replay: %d %v", code, again)
	}
	if got := rg.ledger.balance(owner, "gpu_h100", true); got != "152.800000" {
		t.Fatal("a replay charged again")
	}
	if code, _ := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":4,"term":"1mo"}`, key); code != 409 {
		t.Fatalf("same key, different body: %d", code)
	}

	// Another tenant's on-demand work cannot take the 8 set aside.
	other := tenantJWT(t, "00000000-0000-4000-8000-0000000000d3")
	if code, _ := rg.call("POST", "/v1/compute/instances", other, `{"type":"gpu_h100","count":9}`, map[string]string{"Idempotency-Key": "o-1"}); code != 402 {
		t.Fatalf("on-demand took reserved GPUs: %d", code)
	}
	if _, err := rg.sched.Submit(scheduler.JobSpec{TenantID: "x", WorkloadClass: domain.ClassInference, GPUType: domain.CreditH100, GPUs: 9, Pods: 1}, "j"); err == nil {
		t.Fatal("an internal job took reserved GPUs")
	}

	// The owner's instance runs on its reservation; its usage bills zero but still reports the time.
	code, inst := rg.call("POST", "/v1/compute/instances", tok, `{"type":"gpu_h100","count":8}`, map[string]string{"Idempotency-Key": "i-1"})
	if code != 201 || inst["reserved"] != true {
		t.Fatalf("owner instance: %d %v", code, inst)
	}
	_, list := rg.call("GET", "/v1/compute/reservations", tok, "", nil)
	capy, _ := list["capacity"].([]any)
	if len(capy) != 1 || capy[0].(map[string]any)["reserved_gpus"] != float64(8) || capy[0].(map[string]any)["in_use_gpus"] != float64(8) {
		t.Fatalf("capacity: %v", list)
	}
	time.Sleep(1100 * time.Millisecond)
	rg.call("POST", "/v1/compute/instances/"+inst["id"].(string)+"/stop", tok, "", nil)
	ev, ok := rg.usage.last(inst["id"].(string))
	if !ok || !ev.Reserved || ev.Units != "0.000000" || ev.GPUSeconds == "0.000000" {
		t.Fatalf("reserved usage: %+v", ev)
	}
	// A second instance beyond the reservation is on-demand and billed.
	code, od := rg.call("POST", "/v1/compute/instances", tok, `{"type":"gpu_h100","count":8}`, map[string]string{"Idempotency-Key": "i-2"})
	if code != 201 || od["reserved"] != true {
		t.Fatalf("the stopped instance freed the reservation for the next: %d %v", code, od)
	}
	code, od2 := rg.call("POST", "/v1/compute/instances", tok, `{"type":"gpu_h100","count":8}`, map[string]string{"Idempotency-Key": "i-3"})
	if code != 201 || od2["reserved"] != false {
		t.Fatalf("beyond the reservation: %d %v", code, od2)
	}

	// Isolation and roles.
	if code, _ := rg.call("GET", "/v1/compute/reservations/"+res["id"].(string), other, "", nil); code != 404 {
		t.Fatalf("another tenant read it: %d", code)
	}
	if code, _ := rg.call("POST", "/v1/compute/reservations", partnerJWT(t, owner, "engineer"), body, map[string]string{"Idempotency-Key": "res-2"}); code != 403 {
		t.Fatalf("engineer bought capacity: %d", code)
	}
}

// TestReservationRefusals: no capacity records nothing and charges nothing; too few credits records
// a failed reservation and gives the GPUs back.
func TestReservationRefusals(t *testing.T) {
	rg := newReserveRig(t)
	owner := "00000000-0000-4000-8000-0000000000d4"
	tok := partnerJWT(t, owner, "admin")
	rg.fund(owner, "gpu_h100", "100")
	if code, out := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":17,"term":"1mo"}`, map[string]string{"Idempotency-Key": "a"}); code != 409 || out["code"] != "capacity_unavailable" {
		t.Fatalf("over capacity: %d %v", code, out)
	}
	if rg.ledger.calls != 0 {
		t.Fatal("charged for capacity that does not exist")
	}
	code, out := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":8,"term":"1mo"}`, map[string]string{"Idempotency-Key": "b"})
	if code != 402 || out["code"] != "insufficient_credit" {
		t.Fatalf("short of credits: %d %v", code, out)
	}
	if rg.pool.Available(domain.CreditH100) != 16 {
		t.Fatal("an unpaid reservation kept its GPUs")
	}
	if code, _ := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":8,"term":"1mo"}`, map[string]string{"Idempotency-Key": "b"}); code != 402 {
		t.Fatalf("replay of a failed purchase: %d", code)
	}
	_, list := rg.call("GET", "/v1/compute/reservations", tok, "", nil)
	if d := list["data"].([]any); len(d) != 1 || d[0].(map[string]any)["state"] != "failed" {
		t.Fatalf("list: %v", list)
	}
}

// TestReservationLedgerOutage: when the ledger does not answer, the reservation waits with its GPUs
// set aside; Sync (or a retry) pays it exactly once.
func TestReservationLedgerOutage(t *testing.T) {
	rg := newReserveRig(t)
	owner := "00000000-0000-4000-8000-0000000000d5"
	tok := partnerJWT(t, owner, "billing")
	rg.fund(owner, "gpu_h100", "10000")
	rg.ledger.down = true
	code, out := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":4,"term":"1mo"}`, map[string]string{"Idempotency-Key": "k"})
	if code != 503 || out["code"] != "payment_pending" {
		t.Fatalf("outage: %d %v", code, out)
	}
	if rg.pool.Available(domain.CreditH100) != 12 {
		t.Fatal("a pending reservation lost its GPUs")
	}
	rg.ledger.down = false
	// The caller retries with the same key: the payment goes through now.
	if code, again := rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":4,"term":"1mo"}`, map[string]string{"Idempotency-Key": "k"}); code != 201 || again["state"] != "active" {
		t.Fatalf("retry: %d %v", code, again)
	}
	// A second one left pending is settled by Sync instead.
	rg.ledger.down = true
	rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":2,"term":"1mo"}`, map[string]string{"Idempotency-Key": "k2"})
	rg.ledger.down = false
	if err := rg.svc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, list := rg.call("GET", "/v1/compute/reservations", tok, "", nil)
	for _, d := range list["data"].([]any) {
		if d.(map[string]any)["state"] != "active" {
			t.Fatalf("after sync: %v", list)
		}
	}
	if err := rg.svc.Sync(context.Background()); err != nil { // nothing left to pay
		t.Fatal(err)
	}
	if got := rg.ledger.balance(owner, "gpu_h100", true); got != "6364.600000" {
		t.Fatalf("balance %s: each reservation should be paid exactly once (10000 - 2423.6 - 1211.8)", got)
	}
}

// TestReservationExpiresAndRebuilds: holds are rebuilt from the records after a restart; when the
// term ends, the GPUs go back and running work moves to on-demand billing.
func TestReservationExpiresAndRebuilds(t *testing.T) {
	rg := newReserveRig(t)
	owner := "00000000-0000-4000-8000-0000000000d6"
	tok := partnerJWT(t, owner, "billing")
	rg.fund(owner, "gpu_h100", "10000")
	rg.call("POST", "/v1/compute/reservations", tok, `{"gpu_type":"gpu_h100","gpus":8,"term":"1mo"}`, map[string]string{"Idempotency-Key": "k"})

	// A restart: a fresh pool gets the hold back from the database.
	fresh := pool.New("dc-owned-1", map[string]int{domain.CreditH100: 16})
	svc2, err := reserve.Open(context.Background(), rg.dsn, fresh, rg.ledger)
	if err != nil {
		t.Fatal(err)
	}
	defer svc2.Close()
	if err := svc2.Sync(context.Background()); err != nil || fresh.Available(domain.CreditH100) != 8 {
		t.Fatalf("rebuilt holds: %v, available %d", err, fresh.Available(domain.CreditH100))
	}

	_, inst := rg.call("POST", "/v1/compute/instances", tok, `{"type":"gpu_h100","count":8}`, map[string]string{"Idempotency-Key": "i"})
	if inst["reserved"] != true {
		t.Fatalf("instance: %v", inst)
	}
	rg.svc.Now = func() time.Time { return time.Now().Add(731 * time.Hour) }
	if err := rg.svc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, list := rg.call("GET", "/v1/compute/reservations", tok, "", nil)
	if list["data"].([]any)[0].(map[string]any)["state"] != "expired" {
		t.Fatalf("not expired: %v", list)
	}
	time.Sleep(1100 * time.Millisecond)
	rg.mgr.MeterTick() // meters the last prepaid interval, then moves the instance to on-demand
	if ev, _ := rg.usage.last(inst["id"].(string)); ev.Units != "0.000000" {
		t.Fatalf("the interval inside the term should be prepaid: %+v", ev)
	}
	time.Sleep(1100 * time.Millisecond)
	rg.mgr.MeterTick()
	if ev, _ := rg.usage.last(inst["id"].(string)); ev.Reserved || ev.Units == "0.000000" {
		t.Fatalf("after the term, usage must be billed: %+v", ev)
	}
	if rg.pool.Available(domain.CreditH100) != 8 {
		t.Fatalf("available %d: 16 less the running 8, nothing held", rg.pool.Available(domain.CreditH100))
	}
}
