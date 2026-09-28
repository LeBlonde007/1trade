package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/scheduler"
)

// activePartner registers, activates and heartbeats a source for tenant; it returns the source id.
func (rg *supplyRig) activePartner(t *testing.T, tenant, tok string) string {
	t.Helper()
	code, src := rg.call("POST", "/v1/supply/sources", tok, `{"name":"row 1","gpu_type":"gpu_h100","gpu_count":16,"region":"eu","sla_tier":"gold"}`, map[string]string{"Idempotency-Key": "k-" + tenant})
	if code != 201 {
		t.Fatalf("register: %d %v", code, src)
	}
	id := src["id"].(string)
	rg.call("POST", "/v1/supply/sources/"+id+"/activate", testSvc, "", nil)
	rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":16}`, nil)
	return id
}

// TestPayoutCycleEndToEnd drives F18: usage on a partner source becomes statements (paper and real
// kept apart), reconciled exactly to usage, through wire, dispute, resolution and holdback release.
func TestPayoutCycleEndToEnd(t *testing.T) {
	rg := newSupplyRig(t)
	partner := "00000000-0000-4000-8000-0000000000aa"
	tok := partnerJWT(t, partner, "engineer")
	src := rg.activePartner(t, partner, tok)

	// Paper usage through the real scheduler path, plus one real-money record written as the metering
	// would (real trading/compute is licence-gated, so nothing produces it yet).
	job, err := rg.sched.Submit(scheduler.JobSpec{TenantID: "customer-1", IsPaper: true, WorkloadClass: domain.ClassInference,
		GPUType: domain.CreditH100, GPUs: 8, Pods: 1}, "pay-1")
	if err != nil || job.SupplySourceID != src {
		t.Fatalf("job on %q: %v", job.SupplySourceID, err)
	}
	time.Sleep(1100 * time.Millisecond)
	rg.sched.Cancel("customer-1", job.ID)
	st := rg.s.supply.Store
	if err := st.RecordUsage(context.Background(), "real-usage-1", src, "customer-2", "gpu_h100", "7200.000000", "4.000000", false); err != nil {
		t.Fatal(err)
	}

	// No agreement yet: closing pays nothing and leaves the usage unpaid.
	rg.s.now = func() time.Time { return time.Now().Add(time.Minute) }
	period := fmt.Sprintf(`{"period_start":%q,"period_end":%q}`, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), time.Now().Add(30*time.Second).UTC().Format(time.RFC3339))
	if code, out := rg.call("POST", "/v1/supply/payouts/cycles", testSvc, period, nil); code != 200 || len(out["data"].([]any)) != 0 {
		t.Fatalf("close without terms: %d %v", code, out)
	}
	if code, _ := rg.call("PUT", "/v1/supply/partners/"+partner+"/agreement", tok, `{"rates":{"gpu_h100":"1.800000"},"fee_percent":"10","holdback_percent":"15","dispute_days":14}`, nil); code != 403 {
		t.Fatalf("partner set its own terms: %d", code)
	}
	if code, a := rg.call("PUT", "/v1/supply/partners/"+partner+"/agreement", testSvc, `{"rates":{"gpu_h100":"1.800000"},"fee_percent":"10","holdback_percent":"15","dispute_days":14}`, nil); code != 200 || a["fee_percent"] != "10.000000" {
		t.Fatalf("set terms: %d %v", code, a)
	}
	if code, _ := rg.call("POST", "/v1/supply/payouts/cycles", tok, period, nil); code != 403 {
		t.Fatalf("partner closed a cycle: %d", code)
	}
	code, out := rg.call("POST", "/v1/supply/payouts/cycles", testSvc, period, nil)
	stmts, _ := out["data"].([]any)
	if code != 200 || len(stmts) != 2 {
		t.Fatalf("close: %d %v", code, out)
	}
	var paper, real map[string]any
	for _, s := range stmts {
		m := s.(map[string]any)
		if m["is_paper"] == true {
			paper = m
		} else {
			real = m
		}
	}
	// Real: 2 h × 1.8 = 3.6 gross; fee 0.36; payout 3.24; holdback 0.486; released 2.754.
	if real["gross"] != "3.600000" || real["fee"] != "0.360000" || real["payout"] != "3.240000" ||
		real["holdback"] != "0.486000" || real["released"] != "2.754000" || real["usage_records"] != float64(1) {
		t.Fatalf("real statement: %v", real)
	}
	if paper == nil || paper["usage_records"] != float64(1) {
		t.Fatalf("paper statement: %v", paper)
	}
	// Reconciliation: every usage record on the partner's source is in exactly one statement, and a
	// repeat close pays nothing again.
	var unpaid int
	if err := st.QueryRowForTest(context.Background(), `SELECT count(*) FROM supply_usage u LEFT JOIN payout_usage p USING (usage_id) WHERE u.source_id=$1 AND p.usage_id IS NULL`, src).Scan(&unpaid); err != nil || unpaid != 0 {
		t.Fatalf("unpaid usage after the cycle: %d (%v)", unpaid, err)
	}
	if code, again := rg.call("POST", "/v1/supply/payouts/cycles", testSvc, period, nil); code != 200 || len(again["data"].([]any)) != 0 {
		t.Fatalf("repeat close: %d %v", code, again)
	}

	pid, rid := paper["id"].(string), real["id"].(string)
	// A paper statement is a simulation: never wired.
	if code, e := rg.call("POST", "/v1/supply/payouts/"+pid+"/wire", testSvc, `{"reference":"SEPA-1"}`, nil); code != 409 || e["code"] != "PAPER_STATEMENT" {
		t.Fatalf("paper wire: %d %v", code, e)
	}
	if code, _ := rg.call("POST", "/v1/supply/payouts/"+rid+"/wire", tok, `{"reference":"SEPA-1"}`, nil); code != 403 {
		t.Fatalf("partner recorded a wire: %d", code)
	}
	if code, w := rg.call("POST", "/v1/supply/payouts/"+rid+"/wire", testSvc, `{"reference":"SEPA-2026-10-001"}`, nil); code != 200 || w["state"] != "wired" {
		t.Fatalf("wire: %d %v", code, w)
	}
	// Holdback cannot be released inside the window; the partner disputes; operations resolve.
	if code, _ := rg.call("POST", "/v1/supply/payouts/"+rid+"/release-holdback", testSvc, "", nil); code != 409 {
		t.Fatalf("early holdback release: %d", code)
	}
	if code, d := rg.call("POST", "/v1/supply/payouts/"+rid+"/dispute", tok, `{"reason":"We count 2.5 GPU-hours on 2026-09-28"}`, nil); code != 200 || d["state"] != "disputed" {
		t.Fatalf("dispute: %d %v", code, d)
	}
	if _, q := rg.call("GET", "/v1/supply/payouts?state=disputed", testSvc, "", nil); len(q["data"].([]any)) != 1 {
		t.Fatalf("review queue: %v", q)
	}
	if code, r := rg.call("POST", "/v1/supply/payouts/"+rid+"/resolve", testSvc, `{"outcome":"release","note":"meter logs confirm 2 h"}`, nil); code != 200 || r["state"] != "settled" || r["resolution"] != "release" {
		t.Fatalf("resolve: %d %v", code, r)
	}

	// A second real statement settles by the clock: after the window, the holdback is released.
	st.RecordUsage(context.Background(), "real-usage-2", src, "customer-2", "gpu_h100", "3600.000000", "2.000000", false)
	period2 := fmt.Sprintf(`{"period_start":%q,"period_end":%q}`, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), time.Now().Add(40*time.Second).UTC().Format(time.RFC3339))
	_, out2 := rg.call("POST", "/v1/supply/payouts/cycles", testSvc, period2, nil)
	if len(out2["data"].([]any)) != 1 {
		t.Fatalf("second cycle: %v", out2)
	}
	id2 := out2["data"].([]any)[0].(map[string]any)["id"].(string)
	rg.call("POST", "/v1/supply/payouts/"+id2+"/wire", testSvc, `{"reference":"SEPA-2"}`, nil)
	rg.s.now = func() time.Time { return time.Now().Add(15 * 24 * time.Hour) }
	if code, d := rg.call("POST", "/v1/supply/payouts/"+id2+"/dispute", tok, `{"reason":"late"}`, nil); code != 409 {
		t.Fatalf("dispute after the window: %d %v", code, d)
	}
	if code, s := rg.call("POST", "/v1/supply/payouts/"+id2+"/release-holdback", testSvc, "", nil); code != 200 || s["state"] != "settled" {
		t.Fatalf("holdback release: %d %v", code, s)
	}

	// Isolation: another partner sees nothing and cannot read these.
	other := partnerJWT(t, "00000000-0000-4000-8000-0000000000bb", "engineer")
	if _, l := rg.call("GET", "/v1/supply/payouts", other, "", nil); len(l["data"].([]any)) != 0 {
		t.Fatal("another partner listed these statements")
	}
	if code, _ := rg.call("GET", "/v1/supply/payouts/"+rid, other, "", nil); code != 404 {
		t.Fatalf("another partner read a statement: %d", code)
	}
	if code, _ := rg.call("GET", "/v1/supply/partners/"+partner+"/agreement", other, "", nil); code != 404 {
		t.Fatalf("another partner read the terms: %d", code)
	}
	if code, own := rg.call("GET", "/v1/supply/payouts/"+rid, tok, "", nil); code != 200 || len(own["lines"].([]any)) != 1 {
		t.Fatalf("own statement: %d %v", code, own)
	}
	b, _ := json.Marshal(real)
	if strings.Contains(string(b), "e+") {
		t.Fatal("an amount rendered as a float")
	}
}
