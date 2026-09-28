package api

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/supply"
)

// gpuReport builds a GPU identity report of n H100s (UUIDs prefixed by tag) bound to nonce.
func gpuReport(sourceID, nonce, tag, model string, n int) []byte {
	type gpu struct {
		UUID  string `json:"uuid"`
		Model string `json:"model"`
	}
	gpus := make([]gpu, n)
	for i := range gpus {
		gpus[i] = gpu{UUID: fmt.Sprintf("GPU-%s-%04d", tag, i), Model: model}
	}
	b, _ := json.Marshal(map[string]any{"source_id": sourceID, "nonce": nonce, "gpus": gpus})
	return b
}

// challenge issues a challenge of kind for a source and returns it.
func (rg *supplyRig) challenge(t *testing.T, id, tok, kind string) map[string]any {
	t.Helper()
	code, c := rg.call("POST", "/v1/supply/sources/"+id+"/challenges", tok, `{"kind":"`+kind+`"}`, nil)
	if code != 201 {
		t.Fatalf("issue %s challenge: %d %v", kind, code, c)
	}
	return c
}

// answerHardware signs report with key and submits it for challenge c.
func (rg *supplyRig) answerHardware(id, tok string, c map[string]any, report []byte, key ed25519.PrivateKey) (int, map[string]any) {
	body, _ := json.Marshal(map[string]string{
		"report":    base64.StdEncoding.EncodeToString(report),
		"signature": base64.StdEncoding.EncodeToString(ed25519.Sign(key, report)),
	})
	return rg.call("POST", "/v1/supply/sources/"+id+"/challenges/"+c["id"].(string), tok, string(body), nil)
}

// answerChallenge computes the right (or a wrong) answer to challenge c and submits it.
func (rg *supplyRig) answerChallenge(id, tok string, c map[string]any, right bool) (int, map[string]any) {
	nonce, _ := hex.DecodeString(c["nonce"].(string))
	ans := supply.ChallengeAnswer(nonce, int(c["iterations"].(float64)))
	if !right {
		ans = supply.ChallengeAnswer(nonce, int(c["iterations"].(float64))+1)
	}
	return rg.call("POST", "/v1/supply/sources/"+id+"/challenges/"+c["id"].(string), tok, `{"result":"`+ans+`"}`, nil)
}

// layer returns one layer's state from an attestation status response.
func layer(st map[string]any, name string) string {
	l, _ := st["layers"].(map[string]any)[name].(map[string]any)
	s, _ := l["state"].(string)
	return s
}

// attest passes all five layers for a source of n H100s through the real endpoints, which
// activates it and leaves a fresh heartbeat.
func (rg *supplyRig) attest(t *testing.T, id, tok string, n int) {
	t.Helper()
	for _, l := range []string{"kyb", "bond"} {
		if code, st := rg.call("POST", "/v1/supply/sources/"+id+"/attestation/"+l, testSvc, `{"state":"pass","evidence":{"ref":"doc-1"}}`, nil); code != 200 || layer(st, l) != "pass" {
			t.Fatalf("%s: %d %v", l, code, st)
		}
	}
	hw := rg.challenge(t, id, tok, "hardware")
	if code, st := rg.answerHardware(id, tok, hw, gpuReport(id, hw["nonce"].(string), id[:8], "NVIDIA H100 80GB HBM3", n), rg.signer); code != 200 || layer(st, "hardware") != "pass" {
		t.Fatalf("hardware: %d %v", code, st)
	}
	ch := rg.challenge(t, id, tok, "challenge")
	if code, st := rg.answerChallenge(id, tok, ch, true); code != 200 || layer(st, "challenge") != "pass" {
		t.Fatalf("challenge: %d %v", code, st)
	}
	if code, hb := rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, fmt.Sprintf(`{"gpus_healthy":%d}`, n), nil); code != 200 || hb["state"] != "active" {
		t.Fatalf("the telemetry pass should have completed attestation and activated the source: %d %v", code, hb)
	}
}

// register registers an n-GPU H100 source for tok under key and returns its id.
func (rg *supplyRig) register(t *testing.T, tok, key string, n int) string {
	t.Helper()
	code, src := rg.call("POST", "/v1/supply/sources", tok, fmt.Sprintf(`{"name":"n","gpu_type":"gpu_h100","gpu_count":%d,"region":"r","sla_tier":"gold"}`, n), map[string]string{"Idempotency-Key": key})
	if code != 201 {
		t.Fatalf("register: %d %v", code, src)
	}
	return src["id"].(string)
}

// TestAttestationGatesActivation: nothing activates a source until all five layers pass; the pass
// that completes attestation activates it by itself, audited as attestation.
func TestAttestationGatesActivation(t *testing.T) {
	rg := newSupplyRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000f1", "engineer")
	id := rg.register(t, tok, "k", 4)

	code, st := rg.call("GET", "/v1/supply/sources/"+id+"/attestation", tok, "", nil)
	if code != 200 || st["complete"] != false || len(st["missing"].([]any)) != 5 {
		t.Fatalf("fresh status: %d %v", code, st)
	}
	rg.call("POST", "/v1/supply/sources/"+id+"/attestation/kyb", testSvc, `{"state":"pass","evidence":{"ref":"kyb-1"}}`, nil)
	rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":4}`, nil)
	if code, a := rg.call("POST", "/v1/supply/sources/"+id+"/activate", testSvc, "", nil); code != 409 || a["code"] != "ATTESTATION_INCOMPLETE" {
		t.Fatalf("activate with kyb+telemetry only: %d %v", code, a)
	}
	rg.sync.Sync(context.Background())
	if rg.pool.Capacity(domain.CreditH100) != 4 {
		t.Fatal("an unattested source is in the pool's capacity")
	}

	rg.attest(t, id, tok, 4)
	code, st = rg.call("GET", "/v1/supply/sources/"+id+"/attestation", tok, "", nil)
	if code != 200 || st["complete"] != true || len(st["missing"].([]any)) != 0 {
		t.Fatalf("attested status: %d %v", code, st)
	}
	if rg.pool.Capacity(domain.CreditH100) != 8 {
		t.Fatalf("attested source not in the pool: %d", rg.pool.Capacity(domain.CreditH100))
	}
	var actor string
	if err := rg.s.supply.Store.QueryRowForTest(context.Background(),
		`SELECT actor FROM supply_events WHERE source_id=$1 AND kind='activated'`, id).Scan(&actor); err != nil || actor != "attestation" {
		t.Fatalf("activation audit: %q %v", actor, err)
	}
	var records int
	if err := rg.s.supply.Store.QueryRowForTest(context.Background(),
		`SELECT count(*) FROM attestation_records WHERE source_id=$1`, id).Scan(&records); err != nil || records != 6 {
		t.Fatalf("records %d (%v), want kyb twice + bond, hardware, challenge, telemetry once", records, err)
	}
}

// TestHardwareReportRejections: every way a GPU report can be wrong records a fail, and a challenge
// takes exactly one answer before its deadline.
func TestHardwareReportRejections(t *testing.T) {
	rg := newSupplyRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000f2", "engineer")
	id := rg.register(t, tok, "k", 4)
	_, stranger, _ := ed25519.GenerateKey(nil)
	other := "00000000-0000-4000-8000-000000000999"

	for name, c := range map[string]struct {
		report func(nonce string) []byte
		key    ed25519.PrivateKey
	}{
		"untrusted signer": {func(n string) []byte { return gpuReport(id, n, "a", "H100", 4) }, stranger},
		"another source":   {func(n string) []byte { return gpuReport(other, n, "a", "H100", 4) }, rg.signer},
		"stale nonce":      {func(string) []byte { return gpuReport(id, hex.EncodeToString(make([]byte, 32)), "a", "H100", 4) }, rg.signer},
		"too few gpus":     {func(n string) []byte { return gpuReport(id, n, "a", "H100", 3) }, rg.signer},
		"wrong model":      {func(n string) []byte { return gpuReport(id, n, "a", "NVIDIA A100", 4) }, rg.signer},
		"not json":         {func(string) []byte { return []byte("GPU-a GPU-b") }, rg.signer},
	} {
		hw := rg.challenge(t, id, tok, "hardware")
		code, st := rg.answerHardware(id, tok, hw, c.report(hw["nonce"].(string)), c.key)
		if code != 200 || layer(st, "hardware") != "fail" {
			t.Errorf("%s: %d %v, want a recorded fail", name, code, st)
		}
	}

	// Duplicated UUIDs count once: four entries naming two GPUs is two GPUs.
	hw := rg.challenge(t, id, tok, "hardware")
	dup, _ := json.Marshal(map[string]any{"source_id": id, "nonce": hw["nonce"], "gpus": []map[string]string{
		{"uuid": "GPU-x", "model": "H100"}, {"uuid": "GPU-x", "model": "H100"}, {"uuid": "GPU-y", "model": "H100"}, {"uuid": "GPU-y", "model": "H100"}}})
	if code, st := rg.answerHardware(id, tok, hw, dup, rg.signer); code != 200 || layer(st, "hardware") != "fail" {
		t.Errorf("duplicated UUIDs: %d %v", code, st)
	}

	// A good report passes once; the same challenge answered again is refused and records nothing.
	hw = rg.challenge(t, id, tok, "hardware")
	good := gpuReport(id, hw["nonce"].(string), "a", "NVIDIA H100 80GB HBM3", 4)
	if code, st := rg.answerHardware(id, tok, hw, good, rg.signer); code != 200 || layer(st, "hardware") != "pass" {
		t.Fatalf("good report: %d %v", code, st)
	}
	if code, _ := rg.answerHardware(id, tok, hw, good, rg.signer); code != 409 {
		t.Errorf("replayed answer: %d, want 409", code)
	}

	// An answer after the deadline is refused.
	late := rg.challenge(t, id, tok, "hardware")
	rg.s.now = func() time.Time { return time.Now().Add(supply.HardwareTTL + time.Second) }
	if code, _ := rg.answerHardware(id, tok, late, gpuReport(id, late["nonce"].(string), "a", "H100", 4), rg.signer); code != 409 {
		t.Errorf("late answer: %d, want 409", code)
	}
	rg.s.now = nil

	// The same GPUs cannot back a second live source — until the first is retired.
	tok2 := partnerJWT(t, "00000000-0000-4000-8000-0000000000f3", "engineer")
	id2 := rg.register(t, tok2, "k2", 4)
	hw2 := rg.challenge(t, id2, tok2, "hardware")
	if code, st := rg.answerHardware(id2, tok2, hw2, gpuReport(id2, hw2["nonce"].(string), "a", "H100", 4), rg.signer); code != 200 || layer(st, "hardware") != "fail" {
		t.Fatalf("GPUs already attested elsewhere: %d %v", code, st)
	}
	rg.call("DELETE", "/v1/supply/sources/"+id, tok, "", nil)
	hw2 = rg.challenge(t, id2, tok2, "hardware")
	if code, st := rg.answerHardware(id2, tok2, hw2, gpuReport(id2, hw2["nonce"].(string), "a", "H100", 4), rg.signer); code != 200 || layer(st, "hardware") != "pass" {
		t.Fatalf("GPUs of a retired source: %d %v", code, st)
	}
	// A retired source takes no attestation.
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/challenges", tok, `{"kind":"hardware"}`, nil); code != 409 {
		t.Errorf("challenge on a retired source: %d", code)
	}
}

// TestNoTrustRootFailsClosed: without a configured attestation root, no report passes.
func TestNoTrustRootFailsClosed(t *testing.T) {
	rg := newSupplyRig(t)
	rg.s.supply.Verifier = nil
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000f4", "engineer")
	id := rg.register(t, tok, "k", 2)
	hw := rg.challenge(t, id, tok, "hardware")
	if code, st := rg.answerHardware(id, tok, hw, gpuReport(id, hw["nonce"].(string), "a", "H100", 2), rg.signer); code != 200 || layer(st, "hardware") != "fail" {
		t.Fatalf("%d %v", code, st)
	}
}

// TestLivenessChallenge: a wrong answer fails, a late one is refused, and open challenges are capped.
func TestLivenessChallenge(t *testing.T) {
	rg := newSupplyRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000f5", "engineer")
	id := rg.register(t, tok, "k", 2)
	c := rg.challenge(t, id, tok, "challenge")
	if c["iterations"] != float64(supply.ChallengeIterations) || len(c["nonce"].(string)) != 64 {
		t.Fatalf("challenge shape: %v", c)
	}
	if code, st := rg.answerChallenge(id, tok, c, false); code != 200 || layer(st, "challenge") != "fail" {
		t.Fatalf("wrong answer: %d %v", code, st)
	}
	late := rg.challenge(t, id, tok, "challenge")
	rg.s.now = func() time.Time { return time.Now().Add(supply.ChallengeTTL + time.Second) }
	if code, _ := rg.answerChallenge(id, tok, late, true); code != 409 {
		t.Fatalf("late answer: %d", code)
	}
	rg.s.now = nil
	c = rg.challenge(t, id, tok, "challenge")
	if code, st := rg.answerChallenge(id, tok, c, true); code != 200 || layer(st, "challenge") != "pass" {
		t.Fatalf("right answer: %d %v", code, st)
	}
	// Open challenges are capped. The late one is refused but, by the real clock, still open: it counts.
	for i := range supply.MaxOpenChallenges - 1 {
		if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/challenges", tok, `{"kind":"hardware"}`, nil); code != 201 {
			t.Fatalf("challenge %d: %d", i, code)
		}
	}
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/challenges", tok, `{"kind":"hardware"}`, nil); code != 429 {
		t.Fatalf("over the cap: %d, want 429", code)
	}
}

// TestDriftSuspends: ECC errors on an active source record drift and suspend it at once; it cannot
// be resumed until telemetry is clean again.
func TestDriftSuspends(t *testing.T) {
	rg := newSupplyRig(t)
	tok := partnerJWT(t, "00000000-0000-4000-8000-0000000000f6", "engineer")
	id := rg.register(t, tok, "k", 4)
	rg.attest(t, id, tok, 4)
	if rg.pool.Capacity(domain.CreditH100) != 8 {
		t.Fatal("not in the pool")
	}
	code, hb := rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":4,"ecc_errors":3}`, nil)
	if code != 200 || hb["state"] != "suspended" || hb["schedulable"] != false {
		t.Fatalf("ECC errors: %d %v", code, hb)
	}
	if rg.pool.Capacity(domain.CreditH100) != 4 {
		t.Fatal("a drifted source is still taking work")
	}
	_, st := rg.call("GET", "/v1/supply/sources/"+id+"/attestation", tok, "", nil)
	if layer(st, "telemetry") != "drift" || st["complete"] != false {
		t.Fatalf("status after drift: %v", st)
	}
	if code, r := rg.call("POST", "/v1/supply/sources/"+id+"/resume", tok, "", nil); code != 409 || r["code"] != "ATTESTATION_INCOMPLETE" {
		t.Fatalf("resume while drifted: %d %v", code, r)
	}
	rg.call("POST", "/v1/supply/sources/"+id+"/heartbeat", tok, `{"gpus_healthy":4}`, nil)
	if code, r := rg.call("POST", "/v1/supply/sources/"+id+"/resume", tok, "", nil); code != 200 || r["state"] != "active" {
		t.Fatalf("resume after clean telemetry: %d %v", code, r)
	}
	// A failed manual layer suspends too.
	if code, st := rg.call("POST", "/v1/supply/sources/"+id+"/attestation/bond", testSvc, `{"state":"fail","evidence":{"reason":"bond withdrawn"}}`, nil); code != 200 || layer(st, "bond") != "fail" {
		t.Fatalf("bond fail: %d %v", code, st)
	}
	if _, g := rg.call("GET", "/v1/supply/sources/"+id, tok, "", nil); g["state"] != "suspended" {
		t.Fatalf("a failed bond did not suspend: %v", g)
	}
}

// TestAttestationAccess: partners see and answer only their own; only operations record kyb/bond.
func TestAttestationAccess(t *testing.T) {
	rg := newSupplyRig(t)
	tokA := partnerJWT(t, "00000000-0000-4000-8000-0000000000f7", "engineer")
	tokB := partnerJWT(t, "00000000-0000-4000-8000-0000000000f8", "engineer")
	id := rg.register(t, tokA, "k", 2)
	c := rg.challenge(t, id, tokA, "challenge")
	for _, r := range []struct{ method, path, body string }{
		{"GET", "/v1/supply/sources/" + id + "/attestation", ""},
		{"POST", "/v1/supply/sources/" + id + "/challenges", `{"kind":"challenge"}`},
		{"POST", "/v1/supply/sources/" + id + "/challenges/" + c["id"].(string), `{"result":"00"}`},
	} {
		if code, _ := rg.call(r.method, r.path, tokB, r.body, nil); code != 404 {
			t.Errorf("another partner: %s %s = %d, want 404", r.method, r.path, code)
		}
	}
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/attestation/kyb", tokA, `{"state":"pass","evidence":{"r":1}}`, nil); code != 403 {
		t.Errorf("partner self-KYB: %d", code)
	}
	for name, r := range map[string]struct {
		path, body string
		want       int
	}{
		"hardware by hand": {"/attestation/hardware", `{"state":"pass","evidence":{"r":1}}`, 404},
		"no evidence":      {"/attestation/kyb", `{"state":"pass","evidence":{}}`, 422},
		"drift by hand":    {"/attestation/bond", `{"state":"drift","evidence":{"r":1}}`, 422},
		"bad kind":         {"/challenges", `{"kind":"telemetry"}`, 422},
	} {
		if code, _ := rg.call("POST", "/v1/supply/sources/"+id+r.path, testSvc, r.body, nil); code != r.want {
			t.Errorf("%s: %d, want %d", name, code, r.want)
		}
	}
	if code, _ := rg.call("POST", "/v1/supply/sources/"+id+"/challenges/"+c["id"].(string), tokA, `{"report":"%%%","signature":"x"}`, nil); code != 422 {
		t.Errorf("non-base64 report: %d", code)
	}
}
