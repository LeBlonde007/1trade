package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/trade1/inference-gateway/internal/auth"
	"github.com/trade1/inference-gateway/internal/catalog"
	"github.com/trade1/inference-gateway/internal/model"
	"github.com/trade1/inference-gateway/internal/pricing"
)

// Input bounds: one request is at most maxEmbedInputs strings and maxEmbedChars characters in all,
// and one audio file is at most maxAudioBytes (OpenAI's own limit).
const (
	maxEmbedInputs = 256
	maxEmbedChars  = 1 << 20
	maxAudioBytes  = 25 << 20
)

// admitModel runs the checks every inference handler shares after auth: the model resolves (with
// its lifecycle), this deployment can serve it, it has the endpoint's modality, and the tenant has
// credits of its type. ok=false means a response has been written.
func (s *Server) admitModel(w http.ResponseWriter, r *http.Request, p auth.Principal, id, modality, noun string) (catalog.Model, bool) {
	m, found := s.resolveModel(w, id)
	if !found {
		return catalog.Model{}, false
	}
	if !catalog.IsServable(id, s.cfg.InferenceModelMap) {
		writeErr(w, http.StatusNotFound, "model_not_available", id+" is not available on this deployment")
		return catalog.Model{}, false
	}
	if m.Trade1.Modality != modality { // never bill the wrong sub-credit
		writeErr(w, http.StatusNotFound, "model_not_found", id+" is not "+noun)
		return catalog.Model{}, false
	}
	if s.credit != nil {
		ok, bal, err := s.credit.Sufficient(r.Context(), p.TenantID, p.IsPaper, m.Trade1.CreditType)
		if err != nil {
			slog.Warn("pre-flight balance check failed; serving anyway", "tenant_id", p.TenantID, "err", err)
		} else if !ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"code":    "INSUFFICIENT_CREDIT",
				"message": "Not enough " + m.Trade1.CreditType + " credits for this request.",
				"details": map[string]any{
					"credit_type": m.Trade1.CreditType, "balance": bal,
					"required": m.Trade1.Price, "buy_credits_url": buyCreditsURL,
				},
			})
			return catalog.Model{}, false
		}
	}
	return m, true
}

// embeddings serves POST /v1/embeddings (OpenAI-compatible): input is one string or a batch; billed
// per 1M input tokens in `embeddings` credits.
func (s *Server) embeddings(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var req struct {
		Model string          `json:"model"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 2*maxEmbedChars)).Decode(&req); err != nil || req.Model == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "model and input are required")
		return
	}
	var inputs []string
	var one string
	if json.Unmarshal(req.Input, &one) == nil {
		inputs = []string{one}
	} else if json.Unmarshal(req.Input, &inputs) != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "input must be a string or an array of strings")
		return
	}
	chars := 0
	for _, in := range inputs {
		chars += len(in)
	}
	if len(inputs) == 0 || len(inputs) > maxEmbedInputs || chars > maxEmbedChars {
		writeErr(w, http.StatusBadRequest, "bad_request", "input must be 1-256 strings, 1 MiB in total")
		return
	}
	m, ok := s.admitModel(w, r, p, req.Model, "embeddings", "an embeddings model")
	if !ok {
		return
	}
	eb, ok := s.backend.(model.EmbeddingsBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "embeddings are not available on this deployment")
		return
	}
	start := time.Now()
	res, err := eb.Embed(r.Context(), model.EmbedRequest{Model: req.Model, Input: inputs})
	if errors.Is(err, model.ErrCapabilityUnavailable) {
		writeErr(w, http.StatusNotImplemented, "unsupported", "embeddings are not available on this deployment")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	units, err := pricing.UnitsPerBlock(m.Trade1.Price, res.PromptTokens, 1_000_000)
	if err != nil {
		serverError(w, err)
		return
	}
	data := make([]map[string]any, len(res.Vectors))
	for i, v := range res.Vectors {
		data[i] = map[string]any{"object": "embedding", "index": i, "embedding": v}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list", "model": req.Model, "data": data,
		"usage": map[string]any{"prompt_tokens": res.PromptTokens, "total_tokens": res.PromptTokens},
	})
	s.meter(r.Context(), p, m, req.Model, model.ChatResult{PromptTokens: res.PromptTokens}, units,
		elapsedMS(start), "embreq_"+uuid.NewString())
}

// transcriptions serves POST /v1/audio/transcriptions (OpenAI-compatible multipart: model, file,
// optional language), billed per minute of audio in `speech` credits, pro rata to the second.
func (s *Server) transcriptions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "a multipart body with model and file (at most 25 MiB) is required")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	id := r.FormValue("model")
	f, hdr, err := r.FormFile("file")
	if id == "" || err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "model and file are required")
		return
	}
	defer f.Close()
	audio, err := io.ReadAll(io.LimitReader(f, maxAudioBytes+1))
	if err != nil || len(audio) == 0 || len(audio) > maxAudioBytes {
		writeErr(w, http.StatusBadRequest, "bad_request", "file must be non-empty and at most 25 MiB")
		return
	}
	m, ok := s.admitModel(w, r, p, id, "transcription", "a speech-to-text model")
	if !ok {
		return
	}
	tb, ok := s.backend.(model.TranscriptionBackend)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "unsupported", "speech-to-text is not available on this deployment")
		return
	}
	start := time.Now()
	res, err := tb.Transcribe(r.Context(), model.TranscribeRequest{Model: id, Filename: hdr.Filename, Audio: audio,
		Language: strings.TrimSpace(r.FormValue("language"))})
	if errors.Is(err, model.ErrCapabilityUnavailable) {
		writeErr(w, http.StatusNotImplemented, "unsupported", "speech-to-text is not available on this deployment")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	seconds := int(math.Ceil(res.DurationSec)) // bill whole seconds, rounded up
	units, err := pricing.UnitsPerBlock(m.Trade1.Price, seconds, 60)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": res.Text, "duration": res.DurationSec})
	s.meter(r.Context(), p, m, id, model.ChatResult{}, units, elapsedMS(start), "sttreq_"+uuid.NewString())
}
