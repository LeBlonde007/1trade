package pool

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trade1/inference-gateway/internal/model"
	"github.com/trade1/inference-gateway/internal/routing"
)

// pod is a fake single-model runtime that counts the requests it serves and can hold them open.
type pod struct {
	*httptest.Server
	hits atomic.Int64
	hold chan struct{} // when non-nil, each request blocks until it is closed
}

// newPod starts a runtime that answers with its own name.
func newPod(t *testing.T, name string, hold chan struct{}) *pod {
	p := &pod{hold: hold}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.hits.Add(1)
		if p.hold != nil {
			<-p.hold
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": name}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	t.Cleanup(p.Close)
	return p
}

// client builds the real HTTP client the gateway uses for a replica.
func client(url string) model.Backend { return model.NewVLLMBackend(url, "", nil, 5*time.Second) }

// measured is a fixed measurement time.
var measured = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// spec is a two-GPU pool: llama-3.1-8b (static) on both GPUs, qwen pooled but not running.
func spec(u0, u1 string) Spec {
	return Spec{
		GPUs: []GPUSpec{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}},
		Profiles: []ProfSpec{
			{Model: "llama-3.1-8b", PeakVRAMMiB: 22000, MeasuredAt: measured, Static: true},
			{Model: "qwen2.5-1.5b", PeakVRAMMiB: 9000, MeasuredAt: measured},
		},
		Replicas: []ReplSpec{{Model: "llama-3.1-8b", GPU: "g0", URL: u0}, {Model: "llama-3.1-8b", GPU: "g1", URL: u1}},
	}
}

// chat is a one-message request.
func chat(m string) model.ChatRequest {
	return model.ChatRequest{Model: m, Messages: []model.Message{{Role: "user", Content: "hi"}}}
}

// TestPoolSpreadsAcrossReplicas: concurrent requests land on both pods, and in-flight counts drain.
func TestPoolSpreadsAcrossReplicas(t *testing.T) {
	hold := make(chan struct{})
	p0, p1 := newPod(t, "p0", hold), newPod(t, "p1", hold)
	b, err := New(spec(p0.URL, p1.URL), model.MockBackend{}, client)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := b.Chat(context.Background(), chat("llama-3.1-8b")); err != nil {
				t.Error(err)
			}
		}()
	}
	deadline := time.Now().Add(5 * time.Second)
	for p0.hits.Load()+p1.hits.Load() < 8 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if p0.hits.Load() != 4 || p1.hits.Load() != 4 {
		t.Fatalf("fewest-in-flight should split 8 held requests 4/4, got %d/%d", p0.hits.Load(), p1.hits.Load())
	}
	close(hold)
	wg.Wait()
	for _, r := range b.Router().Replicas() {
		if r.Inflight != 0 {
			t.Fatalf("in-flight leaked on %s: %d", r.GPU, r.Inflight)
		}
	}
}

// TestPoolColdAndFallback: a pooled model with no replica is ErrModelCold (not served elsewhere);
// a model outside the pool goes to the fallback backend.
func TestPoolColdAndFallback(t *testing.T) {
	p0, p1 := newPod(t, "p0", nil), newPod(t, "p1", nil)
	b, err := New(spec(p0.URL, p1.URL), model.MockBackend{}, client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Chat(context.Background(), chat("qwen2.5-1.5b")); !errors.Is(err, model.ErrModelCold) {
		t.Fatalf("want ErrModelCold, got %v", err)
	}
	res, err := b.Chat(context.Background(), chat("llama-3.2-1b"))
	if err != nil || res.Content == "p0" || res.Content == "p1" {
		t.Fatalf("non-pooled model must use the fallback: %+v %v", res, err)
	}
	if p0.hits.Load()+p1.hits.Load() != 0 {
		t.Fatal("pods served a request they do not host")
	}
	res, _ = b.Chat(context.Background(), chat("llama-3.1-8b"))
	if res.Content != "p0" && res.Content != "p1" {
		t.Fatalf("pooled model must come from a pod, got %q", res.Content)
	}
}

// TestNewRefusesBadSpecs: every VRAM rule is enforced at boot.
func TestNewRefusesBadSpecs(t *testing.T) {
	cases := map[string]func(*Spec){
		"unmeasured":          func(s *Spec) { s.Profiles[1].MeasuredAt = time.Time{} },
		"static not running":  func(s *Spec) { s.Replicas = s.Replicas[:1]; s.Replicas[0].Model = "qwen2.5-1.5b" },
		"overcommit":          func(s *Spec) { s.Profiles[0].PeakVRAMMiB = 80000 },
		"unknown gpu":         func(s *Spec) { s.Replicas[1].GPU = "g9" },
		"bad residency":       func(s *Spec) { s.MinResidency = "soon" },
		"static cannot place": func(s *Spec) { s.GPUs = s.GPUs[:1]; s.Profiles[0].PeakVRAMMiB = 78000 },
	}
	for name, mut := range cases {
		s := spec("http://a", "http://b")
		mut(&s)
		if _, err := New(s, model.MockBackend{}, client); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := New(spec("http://a", "http://b"), model.MockBackend{}, client); err != nil {
		t.Fatalf("valid spec refused: %v", err)
	}
	s := spec("http://a", "http://b")
	s.Profiles[1].MeasuredAt = time.Time{}
	if _, err := New(s, model.MockBackend{}, client); !errors.Is(err, routing.ErrUnmeasured) {
		t.Fatalf("want ErrUnmeasured, got %v", err)
	}
}

// TestParseSpec round-trips the JSON and rejects unknown fields (a typo must not drop a limit).
func TestParseSpec(t *testing.T) {
	raw := `{"headroom_ppm":80000,"min_residency":"5m","gpus":[{"id":"g0","class":"h100-80gb","vram_mib":81920}],
	  "profiles":[{"model":"llama-3.1-8b","peak_vram_mib":22000,"measured_at":"2026-09-01T00:00:00Z","static":true}],
	  "replicas":[{"model":"llama-3.1-8b","gpu":"g0","url":"http://rt-0:8000"}]}`
	s, err := ParseSpec(raw)
	if err != nil || s.HeadroomPPM != 80000 || s.GPUs[0].VRAMMiB != 81920 || !s.Profiles[0].Static || s.Replicas[0].URL != "http://rt-0:8000" {
		t.Fatalf("%+v %v", s, err)
	}
	if _, err := ParseSpec(strings.Replace(raw, "headroom_ppm", "headroom_pct", 1)); err == nil {
		t.Fatal("unknown field accepted")
	}
}
