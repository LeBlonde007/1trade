package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/compute-control/internal/auth"
	"github.com/trade1/compute-control/internal/config"
	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/instance"
	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/scheduler"
	"github.com/trade1/compute-control/internal/supply"
)

// supplyDSN creates an isolated schema with the migrations applied; skips without DATABASE_URL.
func supplyDSN(t *testing.T) string {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("supply_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if p, err := pgxpool.New(context.Background(), base); err == nil {
			_, _ = p.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			p.Close()
		}
	})
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	dsn := base + sep + "search_path=" + schema
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	files, _ := filepath.Glob("../../migrations/*.sql")
	sort.Strings(files)
	for _, f := range files {
		sql, _ := os.ReadFile(f)
		if _, err := p.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migrate %s: %v", f, err)
		}
	}
	return dsn
}

// supplyRig is a compute-control with partner supply on a real database and a settable clock.
type supplyRig struct {
	s     *Server
	pool  *pool.Pool
	sched *scheduler.MockScheduler
	sync  *supply.Syncer
	now   time.Time
	// signer stands in for the GPU attestation root the rig's verifier trusts.
	signer ed25519.PrivateKey
}

// newSupplyRig builds it: 4 owned H100s.
func newSupplyRig(t *testing.T) *supplyRig {
	t.Helper()
	st, err := supply.Open(context.Background(), supplyDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	rg := &supplyRig{now: time.Now()}
	rg.pool = pool.New("dc-owned-1", map[string]int{domain.CreditH100: 4})
	pub := supply.Recorder{Next: events.NoopPublisher{}, Store: st}
	rg.sched = scheduler.NewMockWithPool(rg.pool, pub)
	rg.sync = &supply.Syncer{Store: st, Pool: rg.pool, Now: func() time.Time { return rg.now }}
	rg.s = New(config.Config{Env: "dev", Paper: true}, auth.NewResolver(testSecret, testSvc), rg.sched, instance.NewManager(rg.pool, pub))
	pub2, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	rg.signer = priv
	rg.s.EnableSupply(&SupplyDeps{Store: st, Pool: rg.pool, Sync: rg.sync, Verifier: supply.Ed25519Verifier{Keys: []ed25519.PublicKey{pub2}}})
	return rg
}

// partnerJWT mints a token for a tenant with the given roles.
func partnerJWT(t *testing.T, tenant string, roles ...string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenant, "is_paper": true, "roles": roles, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// call sends a request and decodes the JSON answer.
func (rg *supplyRig) call(method, path, bearer, body string, headers map[string]string) (int, map[string]any) {
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	w := do(rg.s, method, path, bearer, b, headers)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestPartnerSupplyEndToEnd: a partner's GPUs join the pool only once active and heartbeating; the
// scheduler then uses them like owned capacity and attributes the usage to them; suspension drains;
// retirement removes the source once its work is gone.
func TestPartnerSupplyEndToEnd(t *testing.T) {
	rg := newSupplyRig(t)
	partner := "00000000-0000-4000-8000-00000000000a"
	tok := partnerJWT(t, partner, "engineer")
	key := map[string]string{"Idempotency-Key": "reg-1"}
	reg := `{"name":"Frankfurt row 7","gpu_type":"gpu_h100","gpu_count":16,"region":"eu-central-1a","sla_tier":"gold"}`

	code, src := rg.call("POST", "/v1/supply/sources", tok, reg, key)
	if code != 201 || src["state"] != "pending" || src["schedulable"] != false {
		t.Fatalf("register: %d %v", code, src)
	}
	id := src["id"].(string)
	if code, again := rg.call("POST", "/v1/supply/sources", tok, reg, key); code != 201 || again["id"] != id {
		t.Fatalf("replay: %d %v", code, again)
	}
	if code, _ := rg.call("POST", "/v1/supply/sources", tok, strings.Replace(reg, "16", "17", 1), key); code != 409 {
		t.Fatalf("different body under the same key: %d", code)
	}

	// Pending: not in the pool's accepting capacity, and a partner cannot activate itself.
	rg.sync.Sync(context.Background())
	if rg.pool.Capacity(domain.CreditH100) != 4 {
		t.Fatal("a pending source is taking work")
	}
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/activate", tok, "", nil); code != 403 {
		t.Fatalf("partner self-activation: %d", code)
	}
	if code, a := rg.call("POST", "/v1/supply/sources/"+id+"/activate", testSvc, "", nil); code != 409 || a["code"] != "ATTESTATION_INCOMPLETE" {
		t.Fatalf("activate before attestation: %d %v", code, a)
	}
	rg.attest(t, id, tok, 16)
	if code, a := rg.call("GET", "/v1/supply/sources/"+id, tok, "", nil); code != 200 || a["state"] != "active" {
		t.Fatalf("attestation did not activate the source: %d %v", code, a)
	}

	// Heartbeat with 12 of 16 healthy: 12 join the pool.
	if code, hb := rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":12,"utilization_pct":40}`, nil); code != 200 || hb["schedulable"] != true {
		t.Fatalf("heartbeat: %d %v", code, hb)
	}
	if rg.pool.Capacity(domain.CreditH100) != 16 {
		t.Fatalf("pool capacity %d, want 4 owned + 12 partner", rg.pool.Capacity(domain.CreditH100))
	}

	// Placement: the partner has the most free GPUs, so it takes the job — and the usage is its.
	job, err := rg.sched.Submit(scheduler.JobSpec{TenantID: "customer-1", IsPaper: true, WorkloadClass: domain.ClassInference,
		GPUType: domain.CreditH100, GPUs: 8, Pods: 1}, "job-1")
	if err != nil || job.SupplySourceID != id {
		t.Fatalf("job on %q (%v), want the partner %s", job.SupplySourceID, err, id)
	}
	if code, got := rg.call("GET", "/v1/supply/sources/"+id, tok, "", nil); code != 200 || got["gpus_in_use"] != float64(8) {
		t.Fatalf("in use: %d %v", code, got)
	}

	// Suspend: new work goes to owned capacity; the running job keeps its GPUs.
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/suspend", tok, "", nil); code != 200 {
		t.Fatalf("suspend: %d", code)
	}
	job2, err := rg.sched.Submit(scheduler.JobSpec{TenantID: "customer-1", IsPaper: true, WorkloadClass: domain.ClassInference,
		GPUType: domain.CreditH100, GPUs: 2, Pods: 1}, "job-2")
	if err != nil || job2.SupplySourceID != "dc-owned-1" {
		t.Fatalf("job while suspended on %q (%v)", job2.SupplySourceID, err)
	}
	if rg.pool.InUse(domain.CreditH100) != 10 {
		t.Fatal("suspension evicted running work")
	}

	// The partner's usage is recorded against the partner (the payout basis).
	time.Sleep(1100 * time.Millisecond) // the mock bills whole elapsed seconds
	if _, err := rg.sched.Cancel("customer-1", job.ID); err != nil {
		t.Fatal(err)
	}
	code, use := rg.call("GET", "/v1/supply/sources/"+id+"/usage", tok, "", nil)
	tiers, _ := use["by_tier"].([]any)
	if code != 200 || len(tiers) != 1 || tiers[0].(map[string]any)["gpu_type"] != domain.CreditH100 ||
		tiers[0].(map[string]any)["gpu_seconds"] == "0.000000" || tiers[0].(map[string]any)["sessions"] != float64(1) {
		t.Fatalf("usage: %d %v", code, use)
	}

	// Retire: drained (nothing running on it now), so it leaves the pool.
	if code, r := rg.call("DELETE", "/v1/supply/sources/"+id, tok, "", nil); code != 200 || r["state"] != "retired" {
		t.Fatalf("retire: %d %v", code, r)
	}
	for _, s := range rg.pool.Sources() {
		if s.ID == id {
			t.Fatal("a retired, drained source is still in the pool")
		}
	}
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":16}`, nil); code != 409 {
		t.Fatalf("heartbeat on a retired source: %d", code)
	}
}

// TestSupplyAccessControl: roles, tenant isolation, operations' view, and validation.
func TestSupplyAccessControl(t *testing.T) {
	rg := newSupplyRig(t)
	a, b := "00000000-0000-4000-8000-00000000000a", "00000000-0000-4000-8000-00000000000b"
	tokA, tokB := partnerJWT(t, a, "admin"), partnerJWT(t, b, "engineer")
	viewer := partnerJWT(t, a, "viewer")
	reg := `{"name":"n","gpu_type":"gpu_h200","gpu_count":8,"region":"r","sla_tier":"silver"}`
	if code, _ := rg.call("POST", "/v1/supply/sources", viewer, reg, map[string]string{"Idempotency-Key": "k"}); code != 403 {
		t.Fatalf("viewer registered: %d", code)
	}
	code, src := rg.call("POST", "/v1/supply/sources", tokA, reg, map[string]string{"Idempotency-Key": "k"})
	if code != 201 {
		t.Fatal(code)
	}
	id := src["id"].(string)
	for _, c := range []struct{ method, path string }{
		{"GET", "/v1/supply/sources/" + id}, {"POST", "/v1/supply/sources/" + id + "/suspend"},
		{"DELETE", "/v1/supply/sources/" + id}, {"GET", "/v1/supply/sources/" + id + "/usage"},
	} {
		if code, _ := rg.call(c.method, c.path, tokB, `{"gpus_healthy":1}`, nil); code != 404 {
			t.Errorf("another tenant: %s %s = %d, want 404", c.method, c.path, code)
		}
	}
	if _, list := rg.call("GET", "/v1/supply/sources", tokB, "", nil); len(list["data"].([]any)) != 0 {
		t.Fatal("another tenant's source was listed")
	}
	if _, all := rg.call("GET", "/v1/supply/sources", testSvc, "", nil); len(all["data"].([]any)) != 2 {
		t.Fatalf("operations should see the partner source and the owned capacity: %v", all)
	}
	if code, _ := rg.call("POST", "/v1/supply/sources", testSvc, reg, map[string]string{"Idempotency-Key": "k2"}); code != 403 {
		t.Fatalf("operations registering: %d", code)
	}
	for name, body := range map[string]string{
		"tier":    `{"name":"n","gpu_type":"gpu_a100","gpu_count":8,"region":"r","sla_tier":"gold"}`,
		"count":   `{"name":"n","gpu_type":"gpu_h100","gpu_count":0,"region":"r","sla_tier":"gold"}`,
		"sla":     `{"name":"n","gpu_type":"gpu_h100","gpu_count":8,"region":"r","sla_tier":"platinum"}`,
		"unknown": `{"name":"n","gpu_type":"gpu_h100","gpu_count":8,"region":"r","sla_tier":"gold","is_owned":true}`,
		"no name": `{"name":" ","gpu_type":"gpu_h100","gpu_count":8,"region":"r","sla_tier":"gold"}`,
	} {
		if code, _ := rg.call("POST", "/v1/supply/sources", tokA, body, map[string]string{"Idempotency-Key": "v-" + name}); code != 422 {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	if code, _ := rg.call("POST", "/v1/supply/sources", tokA, reg, nil); code != 422 {
		t.Errorf("no idempotency key: %d", code)
	}
	if code, _ := rg.call("GET", "/v1/supply/sources/not-a-uuid", tokA, "", nil); code != 404 {
		t.Errorf("bad id: %d", code)
	}
}

// TestStaleHeartbeatStopsNewWork: two minutes without a heartbeat and a source takes no new work;
// its running work stays; a fresh heartbeat brings it back.
func TestStaleHeartbeatStopsNewWork(t *testing.T) {
	rg := newSupplyRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-00000000000c", "engineer")
	_, src := rg.call("POST", "/v1/supply/sources", tok, `{"name":"n","gpu_type":"gpu_h100","gpu_count":8,"region":"r","sla_tier":"gold"}`, map[string]string{"Idempotency-Key": "k"})
	id := src["id"].(string)
	rg.attest(t, id, tok, 8)
	if rg.pool.Capacity(domain.CreditH100) != 12 {
		t.Fatal(rg.pool.Capacity(domain.CreditH100))
	}
	rg.now = rg.now.Add(supply.StaleAfter + time.Second)
	rg.sync.Sync(context.Background())
	if rg.pool.Capacity(domain.CreditH100) != 4 {
		t.Fatal("a silent source is still taking work")
	}
	rg.now = time.Now()
	rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":8}`, nil)
	if rg.pool.Capacity(domain.CreditH100) != 12 {
		t.Fatal("a fresh heartbeat did not bring the source back")
	}
}
