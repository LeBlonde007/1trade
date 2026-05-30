// Package api is the HTTP surface of the inference gateway — the OpenAI-compatible inference API
// (docs/contracts/openapi/inference.yaml). This scaffold wires health/readiness and the model
// catalog; auth, the credit pre-flight, the inference handlers, and usage metering land next
// (tasks #21–#25). Error bodies use the contract's ApiError shape; internals are logged, not leaked.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/exascale/inference-gateway/internal/auth"
	"github.com/exascale/inference-gateway/internal/catalog"
	"github.com/exascale/inference-gateway/internal/config"
)

// Server wires config + the auth resolver into an http.Handler.
type Server struct {
	cfg  config.Config
	auth *auth.Resolver
	mux  *http.ServeMux
}

// New builds the routed handler.
func New(cfg config.Config) *Server {
	s := &Server{cfg: cfg, auth: auth.NewResolver(cfg), mux: http.NewServeMux()}
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
