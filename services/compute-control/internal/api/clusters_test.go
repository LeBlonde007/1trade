package api

import (
	"encoding/json"
	"testing"

	"github.com/trade1/compute-control/internal/auth"
	"github.com/trade1/compute-control/internal/config"
	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/instance"
	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/scheduler"
)

// clusterServer is compute-control over two datacenters of 32 H100s each (64 in all).
func clusterServer() (*Server, *pool.Pool) {
	p := pool.New("dc-owned-1", map[string]int{domain.CreditH100: 32})
	p.SetSource("dc-partner-1", map[string]int{domain.CreditH100: 32}, true)
	s := New(config.Config{Env: "dev", Paper: true}, auth.NewResolver(testSecret, testSvc),
		scheduler.NewMockWithPool(p, events.NoopPublisher{}), instance.NewManager(p, events.NoopPublisher{}))
	return s, p
}

// callJSON sends a request and decodes the JSON answer.
func callJSON(s *Server, method, path, bearer, body string, headers map[string]string) (int, map[string]any) {
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	w := do(s, method, path, bearer, b, headers)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestClusterLifecycle: a self-serve cluster is whole nodes on one fabric, placed all at once, kept
// apart from instances, and released together.
func TestClusterLifecycle(t *testing.T) {
	s, p := clusterServer()
	tok := tenantJWT(t, "00000000-0000-4000-8000-0000000000e1")
	key := map[string]string{"Idempotency-Key": "c-1"}
	body := `{"gpu_type":"gpu_h100","gpus":24,"network":"infiniband","topology":"fat-tree"}`
	code, c := callJSON(s, "POST", "/v1/compute/clusters", tok, body, key)
	if code != 201 || c["state"] != "running" || c["gpus"] != float64(24) || len(c["nodes"].([]any)) != 3 || c["network"] != "infiniband" {
		t.Fatalf("create: %d %v", code, c)
	}
	id := c["id"].(string)
	if code, again := callJSON(s, "POST", "/v1/compute/clusters", tok, body, key); code != 201 || again["id"] != id {
		t.Fatalf("replay: %d %v", code, again)
	}
	if p.InUse(domain.CreditH100) != 24 {
		t.Fatalf("in use %d", p.InUse(domain.CreditH100))
	}
	// Not an instance: the instance endpoints do not see it, and it is not stopped piecemeal.
	if code, _ := callJSON(s, "GET", "/v1/compute/instances/"+id, tok, "", nil); code != 404 {
		t.Fatalf("cluster via instances: %d", code)
	}
	if code, _ := callJSON(s, "POST", "/v1/compute/instances/"+id+"/stop", tok, "", nil); code != 404 {
		t.Fatalf("stop a cluster as an instance: %d", code)
	}
	if _, list := callJSON(s, "GET", "/v1/compute/instances", tok, "", nil); len(list["instances"].([]any)) != 0 {
		t.Fatal("cluster listed as an instance")
	}
	if _, list := callJSON(s, "GET", "/v1/compute/clusters", tok, "", nil); len(list["clusters"].([]any)) != 1 {
		t.Fatal("cluster not listed")
	}
	if code, _ := callJSON(s, "GET", "/v1/compute/clusters/"+id, tenantJWT(t, "00000000-0000-4000-8000-0000000000e2"), "", nil); code != 404 {
		t.Fatalf("another tenant read it: %d", code)
	}
	code, d := callJSON(s, "DELETE", "/v1/compute/clusters/"+id, tok, "", nil)
	if code != 200 || d["state"] != "terminated" || len(d["nodes"].([]any)) != 0 || p.InUse(domain.CreditH100) != 0 {
		t.Fatalf("delete: %d %v, in use %d", code, d, p.InUse(domain.CreditH100))
	}
}

// TestClusterAllOrNothing: a cluster never spans datacenters and is never partly placed — 40 GPUs
// free across two sites is not a 40-GPU cluster.
func TestClusterAllOrNothing(t *testing.T) {
	s, p := clusterServer()
	svcHdr := map[string]string{"Idempotency-Key": "big", "X-Tenant-Id": "00000000-0000-4000-8000-0000000000e3"}
	code, _ := callJSON(s, "POST", "/v1/compute/clusters", testSvc, `{"gpu_type":"gpu_h100","gpus":40,"network":"infiniband","topology":"rail-optimized"}`, svcHdr)
	if code != 402 || p.InUse(domain.CreditH100) != 0 {
		t.Fatalf("40 GPUs over two 32-GPU sites: %d, in use %d (want refused, nothing held)", code, p.InUse(domain.CreditH100))
	}
	svcHdr["Idempotency-Key"] = "fits"
	code, c := callJSON(s, "POST", "/v1/compute/clusters", testSvc, `{"gpu_type":"gpu_h100","gpus":32,"network":"infiniband","topology":"rail-optimized"}`, svcHdr)
	if code != 201 || len(c["nodes"].([]any)) != 4 {
		t.Fatalf("32 on one site: %d %v", code, c)
	}
	for _, st := range p.Sources() {
		if n := st.InUse[domain.CreditH100]; n != 0 && n != 32 {
			t.Fatalf("cluster split across sites: %v", p.Sources())
		}
	}
}

// TestClusterShapeAndSales: whole 8-GPU nodes, 16-256 GPUs, an InfiniBand fabric; over 32 GPUs is
// arranged with sales (operations create it).
func TestClusterShapeAndSales(t *testing.T) {
	s, _ := clusterServer()
	tok := tenantJWT(t, "00000000-0000-4000-8000-0000000000e4")
	for name, body := range map[string]string{
		"not whole nodes": `{"gpu_type":"gpu_h100","gpus":20,"network":"infiniband","topology":"fat-tree"}`,
		"one node":        `{"gpu_type":"gpu_h100","gpus":8,"network":"infiniband","topology":"fat-tree"}`,
		"ethernet":        `{"gpu_type":"gpu_h100","gpus":16,"network":"ethernet","topology":"fat-tree"}`,
		"topology":        `{"gpu_type":"gpu_h100","gpus":16,"network":"infiniband","topology":"torus"}`,
		"tier":            `{"gpu_type":"gpu_a100","gpus":16,"network":"infiniband","topology":"fat-tree"}`,
		"unknown field":   `{"gpu_type":"gpu_h100","gpus":16,"network":"infiniband","topology":"fat-tree","nodes":2}`,
	} {
		if code, _ := callJSON(s, "POST", "/v1/compute/clusters", tok, body, map[string]string{"Idempotency-Key": name}); code != 422 {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	if code, out := callJSON(s, "POST", "/v1/compute/clusters", tok, `{"gpu_type":"gpu_h100","gpus":40,"network":"infiniband","topology":"fat-tree"}`, map[string]string{"Idempotency-Key": "big"}); code != 403 || out["code"] != "contact_sales" {
		t.Fatalf("self-serve over 32: %d %v", code, out)
	}
	if code, _ := callJSON(s, "POST", "/v1/compute/instances", tok, `{"type":"gpu_h100","count":33}`, map[string]string{"Idempotency-Key": "i33"}); code != 422 {
		t.Fatalf("a 33-GPU instance: %d (instances are at most 32; larger is a cluster)", code)
	}
	if code, _ := callJSON(s, "POST", "/v1/compute/clusters", "", `{}`, map[string]string{"Idempotency-Key": "x"}); code != 401 {
		t.Fatalf("no auth: %d", code)
	}
}
