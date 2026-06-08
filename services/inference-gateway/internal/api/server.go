// Package api is the HTTP surface of the inference gateway — the OpenAI-compatible inference API
// (docs/contracts/openapi/inference.yaml). This scaffold wires health/readiness and the model
// catalog; auth, the credit pre-flight, the inference handlers, and usage metering land next
// (tasks #21–#25). Error bodies use the contract's ApiError shape; internals are logged, not leaked.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/exascale/inference-gateway/internal/auth"
	"github.com/exascale/inference-gateway/internal/catalog"
	"github.com/exascale/inference-gateway/internal/config"
	"github.com/exascale/inference-gateway/internal/events"
	"github.com/exascale/inference-gateway/internal/metrics"
	"github.com/exascale/inference-gateway/internal/model"
	"github.com/exascale/inference-gateway/internal/pricing"
	"github.com/google/uuid"
)

// buyCreditsURL is returned in a 402 so the customer knows where to top up.
const buyCreditsURL = "https://app.exascale.io/billing/buy"

// CreditChecker is the pre-flight balance guard (implemented by internal/ledger). A nil checker
// disables the pre-flight (e.g. in unit tests, or when no JWT secret is configured).
type CreditChecker interface {
	Sufficient(ctx context.Context, tenantID string, isPaper bool, creditType string) (ok bool, balance string, err error)
}

// Server wires config + auth + the model backend + the usage publisher + the credit pre-flight.
type Server struct {
	cfg     config.Config
	auth    *auth.Resolver
	backend model.Backend
	usage   events.Publisher
	credit  CreditChecker
	mux     *http.ServeMux
}

// New builds the routed handler. credit may be nil to disable the pre-flight balance check.
func New(cfg config.Config, backend model.Backend, usage events.Publisher, credit CreditChecker) *Server {
	s := &Server{cfg: cfg, auth: auth.NewResolver(cfg), backend: backend, usage: usage, credit: credit, mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// routes registers every endpoint. Inference handlers are registered as they are implemented.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /v1/models", s.listModels)
	s.mux.HandleFunc("POST /v1/chat/completions", s.chatCompletions)
	s.mux.HandleFunc("POST /v1/images/generations", s.imageGenerations)
	s.mux.HandleFunc("POST /v1/videos", s.submitVideo)            // async text-to-video: submit
	s.mux.HandleFunc("GET /v1/videos/{id}", s.getVideo)           // poll job status
	s.mux.HandleFunc("GET /v1/videos/{id}/content", s.getVideoContent) // stream the finished mp4
}

// listModels serves the curated catalog in the OpenAI list shape (auth required).
func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAuth(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": catalog.List()})
}

// requireAuth resolves the caller from the bearer credential (API key or tenant JWT); on failure it
// writes a generic 401 and returns ok=false. Guard handlers with
// `p, ok := s.requireAuth(w, r); if !ok { return }`.
func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	p, err := s.auth.Resolve(r.Context(), bearer(r))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "a valid API key or token is required")
		return auth.Principal{}, false
	}
	return p, true
}

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// chatCompletionRequest mirrors the contract's ChatCompletionRequest (the fields we use).
type chatCompletionRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	MaxTokens int  `json:"max_tokens"`
	Stream    bool `json:"stream"`
}

// chatCompletions serves POST /v1/chat/completions (OpenAI-compatible): authenticate → resolve the
// model from the catalog → serve via the backend → return the completion (JSON, or SSE when
// stream=true) → emit exactly one inference.usage.v1 event for the ledger to debit.
func (s *Server) chatCompletions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" || len(req.Messages) == 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "model and messages are required")
		return
	}
	m, found := catalog.Lookup(req.Model)
	if !found {
		writeErr(w, http.StatusNotFound, "model_not_found", "unknown model: "+req.Model)
		return
	}
	if m.Exascale.Modality != "text" { // chat is text-only; refuse so we never bill the wrong sub-credit
		writeErr(w, http.StatusNotFound, "model_not_found", req.Model+" is not a chat model")
		return
	}
	// Pre-flight: reject before consuming a GPU when the tenant has no credit. Fail-open on a ledger
	// error (a balance-service blip shouldn't block inference; the event-driven debit still records it).
	if s.credit != nil {
		ok, bal, err := s.credit.Sufficient(r.Context(), p.TenantID, p.IsPaper, m.Exascale.CreditType)
		if err != nil {
			slog.Warn("pre-flight balance check failed; serving anyway", "tenant_id", p.TenantID, "err", err)
		} else if !ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"code":    "INSUFFICIENT_CREDIT",
				"message": "Not enough " + m.Exascale.CreditType + " credits to serve this request.",
				"details": map[string]any{
					"credit_type": m.Exascale.CreditType, "balance": bal,
					"required": m.Exascale.Price, "buy_credits_url": buyCreditsURL,
				},
			})
			return
		}
	}

	msgs := make([]model.Message, len(req.Messages))
	for i, mm := range req.Messages {
		msgs[i] = model.Message{Role: mm.Role, Content: mm.Content}
	}

	start := time.Now()
	res, err := s.backend.Chat(r.Context(), model.ChatRequest{Model: req.Model, Messages: msgs, MaxTokens: req.MaxTokens})
	if err != nil {
		serverError(w, err)
		return
	}
	latency := int(time.Since(start).Milliseconds())
	total := res.PromptTokens + res.CompletionTokens
	units, err := pricing.UnitsForTokens(m.Exascale.Price, total)
	if err != nil {
		serverError(w, err)
		return
	}
	requestID := "infreq_" + uuid.NewString()
	id := "chatcmpl_" + uuid.NewString()

	if req.Stream {
		s.streamChat(w, id, req.Model, res)
	} else {
		writeJSON(w, http.StatusOK, map[string]any{
			"id": id, "object": "chat.completion", "created": time.Now().Unix(), "model": req.Model,
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": res.Content},
				"finish_reason": res.FinishReason,
			}},
			"usage": map[string]any{
				"prompt_tokens": res.PromptTokens, "completion_tokens": res.CompletionTokens, "total_tokens": total,
			},
		})
	}
	// Emit AFTER the response completes (exactly once). request_id is the ledger's debit idempotency key.
	s.meter(p, m, req.Model, res, units, latency, requestID)
}

// meter emits exactly one inference.usage.v1 event for a served request. Best-effort: a publish
// failure is logged, never fails the customer (the response was already produced).
func (s *Server) meter(p auth.Principal, m catalog.Model, modelID string, res model.ChatResult, units string, latencyMS int, requestID string) {
	in, out, lat := res.PromptTokens, res.CompletionTokens, latencyMS
	metrics.RecordInference(modelID, m.Exascale.Modality, in, out)
	e := events.UsageEvent{
		RequestID: requestID, TenantID: p.TenantID, Model: modelID,
		Modality: m.Exascale.Modality, CreditType: m.Exascale.CreditType,
		InputTokens: &in, OutputTokens: &out, Units: units, LatencyMS: &lat,
		IsPaper: p.IsPaper, TS: time.Now().UTC().Format(time.RFC3339),
	}
	if p.SubAccountID != "" {
		e.SubAccountID = &p.SubAccountID
	}
	if err := s.usage.PublishUsage(e); err != nil {
		slog.Error("publish inference.usage.v1 failed", "request_id", requestID, "err", err)
	}
}

// imageRequest is the OpenAI-compatible text-to-image request body.
type imageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

// imageGenerations serves POST /v1/images/generations (OpenAI-compatible): authenticate → resolve the
// image model from the catalog → generate via the backend → return base64 images → emit one
// inference.usage.v1 event (units = image count) for the ledger to debit in `image` credits.
func (s *Server) imageGenerations(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req imageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" || strings.TrimSpace(req.Prompt) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "model and prompt are required")
		return
	}
	m, found := catalog.Lookup(req.Model)
	if !found {
		writeErr(w, http.StatusNotFound, "model_not_found", "unknown model: "+req.Model)
		return
	}
	if m.Exascale.Modality != "image" { // refuse non-image models so we never bill the wrong sub-credit
		writeErr(w, http.StatusNotFound, "model_not_found", req.Model+" is not an image model")
		return
	}
	n := req.N
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4 // cap fan-out per request
	}
	// Pre-flight credit check (image credits). Fail-open on a ledger blip, like chat.
	if s.credit != nil {
		okBal, bal, err := s.credit.Sufficient(r.Context(), p.TenantID, p.IsPaper, m.Exascale.CreditType)
		if err != nil {
			slog.Warn("pre-flight balance check failed; serving anyway", "tenant_id", p.TenantID, "err", err)
		} else if !okBal {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"code":    "INSUFFICIENT_CREDIT",
				"message": "Not enough " + m.Exascale.CreditType + " credits to generate this image.",
				"details": map[string]any{
					"credit_type": m.Exascale.CreditType, "balance": bal,
					"required": m.Exascale.Price, "buy_credits_url": buyCreditsURL,
				},
			})
			return
		}
	}
	ib, ok := s.backend.(model.ImageBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "image generation is not available on this deployment (no image provider configured)")
		return
	}
	start := time.Now()
	res, err := ib.Image(r.Context(), model.ImageRequest{Model: req.Model, Prompt: req.Prompt, N: n, Size: req.Size})
	if err != nil {
		serverError(w, err)
		return
	}
	latency := int(time.Since(start).Milliseconds())
	count := len(res.B64)
	units, err := pricing.UnitsForCount(m.Exascale.Price, count)
	if err != nil {
		serverError(w, err)
		return
	}
	requestID := "imgreq_" + uuid.NewString()
	data := make([]map[string]any, 0, count)
	for _, b64 := range res.B64 {
		data = append(data, map[string]any{"b64_json": b64})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"created": time.Now().Unix(), "model": req.Model, "data": data,
	})
	// Emit AFTER the response (exactly once). request_id is the ledger's debit idempotency key.
	s.meterImage(p, m, req.Model, count, units, latency, requestID)
}

// meterImage emits one inference.usage.v1 event for an image-generation request (units = image count),
// billed in `image` credits. Best-effort, like meter — a publish failure never fails the customer.
func (s *Server) meterImage(p auth.Principal, m catalog.Model, modelID string, count int, units string, latencyMS int, requestID string) {
	lat := latencyMS
	metrics.RecordInference(modelID, m.Exascale.Modality, 0, 0)
	e := events.UsageEvent{
		RequestID: requestID, TenantID: p.TenantID, Model: modelID,
		Modality: m.Exascale.Modality, CreditType: m.Exascale.CreditType,
		Units: units, LatencyMS: &lat,
		IsPaper: p.IsPaper, TS: time.Now().UTC().Format(time.RFC3339),
	}
	if p.SubAccountID != "" {
		e.SubAccountID = &p.SubAccountID
	}
	if err := s.usage.PublishUsage(e); err != nil {
		slog.Error("publish inference.usage.v1 (image) failed", "request_id", requestID, "err", err)
	}
	_ = count // count is reflected in `units`; kept for symmetry with meter()'s signature
}

// videoRequest is the text-to-video submit body.
type videoRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
}

// submitVideo serves POST /v1/videos — authenticate → resolve the video model → video-credit pre-flight
// → submit the async job → meter one clip → return {id, status}. The client polls GET /v1/videos/{id}
// until status=="completed", then GET /v1/videos/{id}/content.
func (s *Server) submitVideo(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req videoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" || strings.TrimSpace(req.Prompt) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "model and prompt are required")
		return
	}
	m, found := catalog.Lookup(req.Model)
	if !found {
		writeErr(w, http.StatusNotFound, "model_not_found", "unknown model: "+req.Model)
		return
	}
	if m.Exascale.Modality != "video" {
		writeErr(w, http.StatusNotFound, "model_not_found", req.Model+" is not a video model")
		return
	}
	if s.credit != nil {
		okBal, bal, err := s.credit.Sufficient(r.Context(), p.TenantID, p.IsPaper, m.Exascale.CreditType)
		if err != nil {
			slog.Warn("pre-flight balance check failed; serving anyway", "tenant_id", p.TenantID, "err", err)
		} else if !okBal {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"code": "INSUFFICIENT_CREDIT", "message": "Not enough " + m.Exascale.CreditType + " credits to generate this video.",
				"details": map[string]any{"credit_type": m.Exascale.CreditType, "balance": bal, "required": m.Exascale.Price, "buy_credits_url": buyCreditsURL},
			})
			return
		}
	}
	vb, ok := s.backend.(model.VideoBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "video generation is not available on this deployment (no video provider configured)")
		return
	}
	job, err := vb.SubmitVideo(r.Context(), model.VideoRequest{Model: req.Model, Prompt: req.Prompt, Size: req.Size})
	if err != nil {
		serverError(w, err)
		return
	}
	// Bill one clip on a successful submit (units = price × 1).
	units, _ := pricing.UnitsForCount(m.Exascale.Price, 1)
	s.meterImage(p, m, req.Model, 1, units, 0, "vidreq_"+uuid.NewString())
	writeJSON(w, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "model": req.Model})
}

// getVideo serves GET /v1/videos/{id} — poll the async job (auth required).
func (s *Server) getVideo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAuth(w, r); !ok {
		return
	}
	vb, ok := s.backend.(model.VideoBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "video generation is not available")
		return
	}
	job, err := vb.GetVideo(r.Context(), r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	out := job.Raw
	if out == nil {
		out = map[string]any{"id": job.ID, "status": job.Status}
	}
	writeJSON(w, http.StatusOK, out)
}

// getVideoContent serves GET /v1/videos/{id}/content — stream the finished mp4 through (auth required).
func (s *Server) getVideoContent(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAuth(w, r); !ok {
		return
	}
	vb, ok := s.backend.(model.VideoBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "video generation is not available")
		return
	}
	body, ctype, status, err := vb.GetVideoContent(r.Context(), r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	defer body.Close()
	if ctype == "" {
		ctype = "video/mp4"
	}
	w.Header().Set("Content-Type", ctype)
	w.WriteHeader(status)
	_, _ = io.Copy(w, body)
}

// streamChat sends the completion as an SSE stream of OpenAI chat.completion.chunk objects: a role
// delta, then content deltas, then a finish chunk, then `data: [DONE]`.
func (s *Server) streamChat(w http.ResponseWriter, id, modelID string, res model.ChatResult) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		serverError(w, fmt.Errorf("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	send := func(delta map[string]any, finish any) {
		chunk := map[string]any{
			"id": id, "object": "chat.completion.chunk", "created": time.Now().Unix(), "model": modelID,
			"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finish}},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
	send(map[string]any{"role": "assistant"}, nil)
	for _, word := range strings.Fields(res.Content) {
		send(map[string]any{"content": word + " "}, nil)
	}
	send(map[string]any{}, res.FinishReason)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the contract's ApiError shape ({code, message[, details]}).
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}

// serverError logs the real error and returns a generic 500 (no internal detail leaks to clients).
func serverError(w http.ResponseWriter, err error) {
	slog.Error("request failed", "err", err)
	writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}
