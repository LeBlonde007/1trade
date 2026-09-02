package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trade1/inference-gateway/internal/api"
	"github.com/trade1/inference-gateway/internal/config"
	"github.com/trade1/inference-gateway/internal/events"
	"github.com/trade1/inference-gateway/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "inference-gateway-api-test-secret-32char"

// capturePub records emitted usage events so the test can assert on metering.
type capturePub struct {
	mu     sync.Mutex
	events []events.UsageEvent
}

// PublishUsage records the event.
func (c *capturePub) PublishUsage(e events.UsageEvent) error {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
	return nil
}

// waitForEvent polls briefly for exactly one captured event (metering runs after the response).
func (c *capturePub) waitForEvent(t *testing.T) events.UsageEvent {
	t.Helper()
	for i := 0; i < 50; i++ {
		c.mu.Lock()
		n := len(c.events)
		c.mu.Unlock()
		if n > 0 {
			c.mu.Lock()
			defer c.mu.Unlock()
			if len(c.events) != 1 {
				t.Fatalf("expected exactly 1 usage event, got %d", len(c.events))
			}
			return c.events[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no usage event emitted")
	return events.UsageEvent{}
}

// signJWT mints a tenant JWT the gateway will verify locally.
func signJWT(t *testing.T, tenant string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenant, "is_paper": true, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// newServer builds a gateway with the mock backend + a capturing publisher (pre-flight disabled).
func newServer() (*httptest.Server, *capturePub) {
	pub := &capturePub{}
	cfg := config.Config{Env: "dev", JWTSecret: testSecret, HTTPTimeout: time.Second}
	return httptest.NewServer(api.New(cfg, model.MockBackend{}, pub, nil)), pub
}

// stubChecker is a CreditChecker that always reports the configured sufficiency.
type stubChecker struct {
	ok  bool
	bal string
}

// Sufficient returns the stubbed result.
func (s stubChecker) Sufficient(_ context.Context, _ string, _ bool, _ string) (bool, string, error) {
	return s.ok, s.bal, nil
}

// TestChatCompletion drives a full chat request and asserts the OpenAI-shaped response plus the
// metering event (credit_type, fixed-point units, is_paper, tenant) that the ledger debits on.
func TestChatCompletion(t *testing.T) {
	srv, pub := newServer()
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"model":    "llama-3.1-8b",
		"messages": []map[string]string{{"role": "user", "content": "hello there"}},
	})
	req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+signJWT(t, "tenant-abc"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out struct {
		Object  string `json:"object"`
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Object != "chat.completion" || len(out.Choices) != 1 || out.Choices[0].Message.Content == "" || out.Usage.TotalTokens <= 0 {
		t.Fatalf("unexpected completion: %+v", out)
	}

	e := pub.waitForEvent(t)
	if e.TenantID != "tenant-abc" || e.CreditType != "text" || !e.IsPaper || e.RequestID == "" {
		t.Fatalf("bad usage event: %+v", e)
	}
	if e.Units == "" || !strings.Contains(e.Units, ".") {
		t.Fatalf("units must be a fixed-point string, got %q", e.Units)
	}
}

// TestChatAuthAndModel covers the 401 (no credential) and 404 (unknown model) paths.
func TestChatAuthAndModel(t *testing.T) {
	srv, _ := newServer()
	defer srv.Close()

	// no auth → 401
	resp, _ := http.Post(srv.URL+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"llama-3.1-8b","messages":[{"role":"user","content":"x"}]}`))
	if resp.StatusCode != 401 {
		t.Fatalf("no-auth = %d, want 401", resp.StatusCode)
	}

	// unknown model → 404
	req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-nope","messages":[{"role":"user","content":"x"}]}`))
	req.Header.Set("Authorization", "Bearer "+signJWT(t, "t1"))
	resp2, _ := http.DefaultClient.Do(req)
	if resp2.StatusCode != 404 {
		t.Fatalf("unknown-model = %d, want 404", resp2.StatusCode)
	}
}

// TestChatInsufficientCredit asserts the pre-flight returns 402 INSUFFICIENT_CREDIT (and does not
// serve / meter) when the tenant has no credit.
func TestChatInsufficientCredit(t *testing.T) {
	pub := &capturePub{}
	cfg := config.Config{Env: "dev", JWTSecret: testSecret, HTTPTimeout: time.Second}
	srv := httptest.NewServer(api.New(cfg, model.MockBackend{}, pub, stubChecker{ok: false, bal: "0.000000"}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions",
		strings.NewReader(`{"model":"llama-3.1-8b","messages":[{"role":"user","content":"x"}]}`))
	req.Header.Set("Authorization", "Bearer "+signJWT(t, "t1"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("status %d, want 402", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["code"] != "INSUFFICIENT_CREDIT" {
		t.Fatalf("code = %v, want INSUFFICIENT_CREDIT", body["code"])
	}
	pub.mu.Lock()
	n := len(pub.events)
	pub.mu.Unlock()
	if n != 0 {
		t.Fatalf("metered a rejected request: %d events", n)
	}
}

// TestChatStreaming asserts stream=true yields an SSE body terminated by [DONE].
func TestChatStreaming(t *testing.T) {
	srv, _ := newServer()
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions",
		strings.NewReader(`{"model":"llama-3.1-8b","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+signJWT(t, "t1"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "data: [DONE]") || !strings.Contains(string(b), "chat.completion.chunk") {
		t.Fatalf("SSE body missing chunks/DONE: %s", b)
	}
}
