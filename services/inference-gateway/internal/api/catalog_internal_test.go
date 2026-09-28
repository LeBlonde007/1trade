package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/trade1/inference-gateway/internal/catalog"
	"github.com/trade1/inference-gateway/internal/config"
	"github.com/trade1/inference-gateway/internal/events"
	"github.com/trade1/inference-gateway/internal/model"
)

const secret = "inference-gateway-internal-test-secret-32c"

// recPub records usage events.
type recPub struct {
	mu  sync.Mutex
	evs []events.UsageEvent
}

// PublishUsage records e.
func (p *recPub) PublishUsage(_ context.Context, e events.UsageEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evs = append(p.evs, e)
	return nil
}

// last returns the most recent event (metering runs after the response, so poll briefly).
func (p *recPub) last(t *testing.T, n int) events.UsageEvent {
	t.Helper()
	for range 100 {
		p.mu.Lock()
		if len(p.evs) >= n {
			e := p.evs[n-1]
			p.mu.Unlock()
			return e
		}
		p.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("usage event %d never emitted", n)
	return events.UsageEvent{}
}

// rig is a gateway on the mock backend with a settable clock.
type rig struct {
	srv *httptest.Server
	s   *Server
	pub *recPub
	jwt string
}

// newRig builds the rig.
func newRig(t *testing.T) *rig {
	t.Helper()
	p := &recPub{}
	s := New(config.Config{Env: "dev", JWTSecret: secret, HTTPTimeout: time.Second}, model.MockBackend{}, p, nil)
	s.now = func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": "00000000-0000-4000-8000-000000000001", "is_paper": true, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	return &rig{srv: srv, s: s, pub: p, jwt: tok}
}

// do sends a JSON request and decodes the JSON answer into out (may be nil).
func (rg *rig) do(t *testing.T, method, path string, body any, out any) *http.Response {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, rg.srv.URL+path, rd)
	req.Header.Set("Authorization", "Bearer "+rg.jwt)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp
}

// TestEmbeddings: one string or a batch; one vector each; billed per 1M tokens in embeddings
// credits (not per 1K — that would overcharge 1000×).
func TestEmbeddings(t *testing.T) {
	rg := newRig(t)
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
		} `json:"usage"`
	}
	if resp := rg.do(t, "POST", "/v1/embeddings", map[string]any{"model": "bge-m3", "input": []string{"alpha beta gamma", "delta"}}, &out); resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if len(out.Data) != 2 || out.Data[1].Index != 1 || len(out.Data[0].Embedding) == 0 || out.Usage.PromptTokens == 0 {
		t.Fatalf("response = %+v", out)
	}
	e := rg.pub.last(t, 1)
	want, _ := json.Marshal(float64(out.Usage.PromptTokens) * 2 / 1_000_000) // price 2 per 1M
	if e.CreditType != "embeddings" || e.Modality != "embeddings" || e.Units != "0.000010" {
		t.Fatalf("event = %+v (want units for %s)", e, want)
	}
	if resp := rg.do(t, "POST", "/v1/embeddings", map[string]any{"model": "bge-m3", "input": "just one"}, &out); resp.StatusCode != 200 || len(out.Data) != 1 {
		t.Fatalf("single input: %d, %d vectors", resp.StatusCode, len(out.Data))
	}
	for name, body := range map[string]map[string]any{
		"chat model":  {"model": "llama-3.1-8b", "input": "x"},
		"no input":    {"model": "bge-m3", "input": []string{}},
		"bad input":   {"model": "bge-m3", "input": 42},
		"unknown":     {"model": "nope", "input": "x"},
		"over budget": {"model": "bge-m3", "input": make([]string, 257)},
	} {
		if resp := rg.do(t, "POST", "/v1/embeddings", body, nil); resp.StatusCode < 400 {
			t.Errorf("%s: %d, want an error", name, resp.StatusCode)
		}
	}
}

// TestTranscriptions: multipart upload, billed per minute pro rata to the (rounded-up) second, in
// speech credits, with the event's modality mapped to the contract's enum.
func TestTranscriptions(t *testing.T) {
	rg := newRig(t)
	send := func(modelID string, audio []byte) (*http.Response, map[string]any) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("model", modelID)
		fw, _ := mw.CreateFormFile("file", "call.wav")
		_, _ = fw.Write(audio)
		_ = mw.Close()
		req, _ := http.NewRequest("POST", rg.srv.URL+"/v1/audio/transcriptions", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+rg.jwt)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp, out
	}
	// 90.5 s of mock audio (32 kB/s) bills 91 s: 0.6 × 91/60 = 0.910000.
	resp, out := send("whisper-large-v3", make([]byte, 32000*90+16000))
	if resp.StatusCode != 200 || out["text"] == "" {
		t.Fatalf("status %d, %v", resp.StatusCode, out)
	}
	if e := rg.pub.last(t, 1); e.CreditType != "speech" || e.Modality != "speech" || e.Units != "0.910000" {
		t.Fatalf("event = %+v", e)
	}
	if resp, _ := send("elevenlabs-tts", []byte("x")); resp.StatusCode != 404 {
		t.Fatalf("a TTS model on the STT endpoint: %d, want 404", resp.StatusCode)
	}
	if resp, _ := send("whisper-large-v3", nil); resp.StatusCode != 400 {
		t.Fatalf("empty file: %d, want 400", resp.StatusCode)
	}
}

// TestDeprecationLifecycle: a deprecated model is served with Deprecation/Sunset/Link headers and
// stays listed with its schedule; after the sunset it is 410 with the replacement and unlisted.
func TestDeprecationLifecycle(t *testing.T) {
	rg := newRig(t)
	now := rg.s.now()
	restore, err := catalog.SetDeprecations(map[string]catalog.Deprecation{
		"llama-3.1-70b": {AnnouncedAt: now.Add(-time.Hour), SunsetAt: now.Add(catalog.MinNotice - time.Hour), Replacement: "llama-3.1-8b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer restore()
	chat := map[string]any{"model": "llama-3.1-70b", "messages": []map[string]string{{"role": "user", "content": "hi"}}}
	resp := rg.do(t, "POST", "/v1/chat/completions", chat, nil)
	if resp.StatusCode != 200 || resp.Header.Get("Deprecation") == "" || resp.Header.Get("Sunset") == "" ||
		resp.Header.Get("Link") != `</v1/models/llama-3.1-8b>; rel="successor-version"` {
		t.Fatalf("deprecated: %d, headers %v", resp.StatusCode, resp.Header)
	}
	var list struct {
		Data []catalog.Model `json:"data"`
	}
	rg.do(t, "GET", "/v1/models", nil, &list)
	found := false
	for _, m := range list.Data {
		if m.ID == "llama-3.1-70b" {
			found = true
			if m.Trade1.Status != "deprecated" || m.Trade1.Deprecation == nil || m.Trade1.Deprecation.Replacement != "llama-3.1-8b" {
				t.Fatalf("listed as %+v", m.Trade1)
			}
			if m.Trade1.LatencyP50MS == nil {
				t.Fatal("no measured latency after serving a request")
			}
		}
		if m.Trade1.Category == "" {
			t.Fatalf("%s listed without a category", m.ID)
		}
	}
	if !found {
		t.Fatal("a deprecated model must stay listed")
	}

	rg.s.now = func() time.Time { return now.Add(catalog.MinNotice) }
	var gone map[string]any
	if resp := rg.do(t, "POST", "/v1/chat/completions", chat, &gone); resp.StatusCode != http.StatusGone || gone["code"] != "model_retired" {
		t.Fatalf("after sunset: %d %v", resp.StatusCode, gone)
	}
	rg.do(t, "GET", "/v1/models", nil, &list)
	for _, m := range list.Data {
		if m.ID == "llama-3.1-70b" {
			t.Fatal("a retired model is still listed")
		}
	}
}

// TestVisionBillsAsTextInTheEvent: finer catalog modalities map onto the event contract's enum.
func TestEventModalityMapping(t *testing.T) {
	for id, want := range map[string]string{"nemotron-vision": "text", "code-writer": "text", "whisper-large-v3": "speech", "bge-m3": "embeddings", "flux-schnell": "image"} {
		m, _ := catalog.Lookup(id)
		if got := eventModality(m); got != want {
			t.Errorf("%s → %s, want %s", id, got, want)
		}
	}
}
