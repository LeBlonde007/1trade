package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/exascale/compute-control/internal/auth"
	"github.com/exascale/compute-control/internal/config"
	"github.com/exascale/compute-control/internal/events"
	"github.com/exascale/compute-control/internal/scheduler"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret = "test-secret"
	testSvc    = "test-service-token"
)

// newServer builds a Server backed by the mock scheduler with a known secret + service token.
func newServer() *Server {
	cfg := config.Config{Env: "dev", Paper: true, SupplySource: "dc-owned-1"}
	sched := scheduler.NewMock(8, 0, "dc-owned-1", events.NoopPublisher{})
	return New(cfg, auth.NewResolver(testSecret, testSvc), sched)
}

// tenantJWT mints a valid first-party tenant token.
func tenantJWT(t *testing.T, tenantID string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenantID, "is_paper": true, "exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

// do issues a request and returns the recorder.
func do(s *Server, method, path, bearer string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

// TestTypes_Public checks the GPU-type catalog is served without auth.
func TestTypes_Public(t *testing.T) {
	w := do(newServer(), "GET", "/v1/compute/types", "", nil, nil)
	if w.Code != 200 {
		t.Fatalf("code = %d", w.Code)
	}
	var resp struct {
		Types []map[string]any `json:"types"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Types) != 2 {
		t.Fatalf("types = %d, want 2", len(resp.Types))
	}
}

// TestQuota_RequiresAuth checks an unauthenticated quota read is rejected.
func TestQuota_RequiresAuth(t *testing.T) {
	w := do(newServer(), "GET", "/v1/compute/quota", "", nil, nil)
	if w.Code != 401 {
		t.Fatalf("code = %d, want 401", w.Code)
	}
}

// TestJobLifecycle_ServiceToken submits via the service token, reads + cancels via the tenant JWT.
func TestJobLifecycle_ServiceToken(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(map[string]any{
		"workload_class": "inference", "gpu_type": "gpu_h100", "gpus": 2, "pods": 1,
	})
	w := do(s, "POST", "/v1/compute/jobs", testSvc, body, map[string]string{
		"Idempotency-Key": "k1", "X-Tenant-Id": "t1",
	})
	if w.Code != 202 {
		t.Fatalf("submit code = %d body=%s", w.Code, w.Body.String())
	}
	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &job)
	if job.ID == "" || job.Status != "running" {
		t.Fatalf("job = %+v", job)
	}

	// Tenant can read its own job.
	jwtTok := tenantJWT(t, "t1")
	g := do(s, "GET", "/v1/compute/jobs/"+job.ID, jwtTok, nil, nil)
	if g.Code != 200 {
		t.Fatalf("get code = %d", g.Code)
	}

	// Another tenant cannot.
	other := do(s, "GET", "/v1/compute/jobs/"+job.ID, tenantJWT(t, "t2"), nil, nil)
	if other.Code != 404 {
		t.Fatalf("cross-tenant get = %d, want 404", other.Code)
	}

	// Cancel.
	c := do(s, "DELETE", "/v1/compute/jobs/"+job.ID, jwtTok, nil, nil)
	if c.Code != 200 {
		t.Fatalf("cancel code = %d", c.Code)
	}
}

// TestSubmit_RequiresIdempotencyKey checks the submit endpoint demands the header.
func TestSubmit_RequiresIdempotencyKey(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(map[string]any{"workload_class": "inference", "gpu_type": "gpu_h100", "gpus": 1})
	w := do(s, "POST", "/v1/compute/jobs", testSvc, body, map[string]string{"X-Tenant-Id": "t1"})
	if w.Code != 400 {
		t.Fatalf("code = %d, want 400", w.Code)
	}
}

// TestSubmit_CapacityExhausted maps an over-capacity gang to 402 quota_exhausted.
func TestSubmit_CapacityExhausted(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(map[string]any{"workload_class": "inference", "gpu_type": "gpu_h100", "gpus": 4, "pods": 4})
	w := do(s, "POST", "/v1/compute/jobs", testSvc, body, map[string]string{"Idempotency-Key": "k1", "X-Tenant-Id": "t1"})
	if w.Code != 402 {
		t.Fatalf("code = %d, want 402", w.Code)
	}
}
