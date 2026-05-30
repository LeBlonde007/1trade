// Package api is the HTTP surface of platform-core — auth (signup, login → JWT, me, logout), API
// keys, and OAuth (scaffold). It maps docs/contracts/openapi/platform-core.yaml onto the store +
// domain. Error responses are generic; real detail is logged server-side (no info disclosure).
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/store"
	"github.com/google/uuid"
)

// Server wires config + store into an http.Handler.
type Server struct {
	cfg config.Config
	st  *store.Store
	mux *http.ServeMux
}

// New builds the routed handler.
func New(cfg config.Config, st *store.Store) *Server {
	s := &Server{cfg: cfg, st: st, mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// routes registers every endpoint.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /readyz", s.readyz)
	s.mux.HandleFunc("POST /v1/auth/signup", s.signup)
	s.mux.HandleFunc("POST /v1/auth/login", s.login)
	s.mux.HandleFunc("GET /v1/auth/me", s.me)
	s.mux.HandleFunc("POST /v1/auth/logout", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	s.mux.HandleFunc("GET /v1/auth/keys", s.listKeys)
	s.mux.HandleFunc("POST /v1/auth/keys", s.createKey)
	s.mux.HandleFunc("DELETE /v1/auth/keys/{id}", s.revokeKey)
	// OAuth is scaffolded; real provider wiring (client secrets via Vault) is a follow-up.
	s.mux.HandleFunc("GET /v1/auth/oauth/{provider}", notConfigured)
	s.mux.HandleFunc("GET /v1/auth/oauth/{provider}/callback", notConfigured)
}

// readyz checks the DB is reachable.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if err := s.st.Ping(r.Context()); err != nil {
		slog.Error("readyz: store unreachable", "err", err)
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
}

type credsBody struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantName string `json:"tenant_name"`
}

// signup creates an individual tenant + admin user and returns a token (auto-login).
func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var b credsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Email == "" || b.Password == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "email and password are required")
		return
	}
	hash, err := domain.HashPassword(b.Password)
	if err != nil {
		serverError(w, err)
		return
	}
	name := b.TenantName
	if name == "" {
		name = b.Email
	}
	u, err := s.st.Signup(r.Context(), b.Email, hash, name)
	if errors.Is(err, store.ErrEmailTaken) {
		writeErr(w, http.StatusConflict, "email_taken", "that email is already registered")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: signup", "user_id", u.ID, "tenant_id", u.TenantID) // admin.action audit (NATS event is a follow-up)
	s.issue(w, http.StatusCreated, u.ID, domain.Claims{TenantID: u.TenantID, Roles: u.Roles, IsPaper: u.IsPaper})
}

// login verifies the password and returns a JWT. Failures are generic (don't reveal whether the
// email exists).
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var b credsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	au, ok, err := s.st.GetUserByEmail(r.Context(), b.Email)
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		// Spend the same bcrypt time as a real check so latency doesn't reveal that the email is
		// unregistered (account enumeration). Result is discarded — this path always 401s.
		_ = domain.DummyPasswordCheck(b.Password)
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if !domain.VerifyPassword(au.PasswordHash, b.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	s.issue(w, http.StatusOK, au.UserID, domain.Claims{
		TenantID: au.TenantID, OrgID: au.OrgID, Roles: au.Roles, IsPaper: au.IsPaper,
	})
}

// issue signs a token for the given subject/claims and writes the token response.
func (s *Server) issue(w http.ResponseWriter, status int, sub string, c domain.Claims) {
	tok, err := domain.IssueToken(s.cfg.JWTSecret, sub, c, s.cfg.TokenTTL)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, status, map[string]any{
		"token":      tok,
		"expires_at": time.Now().UTC().Add(s.cfg.TokenTTL).Format(time.RFC3339),
	})
}

// me returns the caller's identity (decoded from the token + a store lookup for email).
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	idn, ok, err := s.st.GetUserByID(r.Context(), p.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": idn.UserID, "email": idn.Email, "tenant_id": idn.TenantID,
		"org_id": idn.OrgID, "roles": idn.Roles, "is_paper": idn.IsPaper,
	})
}

// listKeys returns the tenant's API keys (metadata only).
func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	keys, err := s.st.ListAPIKeys(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"id": k.ID, "name": k.Name, "prefix": k.Prefix, "scopes": k.Scopes,
			"created_at": k.CreatedAt.UTC().Format(time.RFC3339), "revoked": k.Revoked,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": out})
}

// createKey mints a scoped API key; the secret is returned ONCE.
func (s *Server) createKey(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Name == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	k, err := domain.GenerateAPIKey()
	if err != nil {
		serverError(w, err)
		return
	}
	id, err := s.st.CreateAPIKey(r.Context(), p.TenantID, b.Name, k.Prefix, k.Hash, b.Scopes)
	if err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: api key created", "tenant_id", p.TenantID, "key_id", id)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "name": b.Name, "prefix": k.Prefix, "scopes": b.Scopes, "secret": k.Secret,
	})
}

// revokeKey revokes one of the tenant's API keys.
func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil { // malformed id can't match any key — don't 500 on a cast error
		writeErr(w, http.StatusNotFound, "not_found", "key not found")
		return
	}
	if err := s.st.RevokeAPIKey(r.Context(), p.TenantID, id); err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: api key revoked", "tenant_id", p.TenantID, "key_id", id)
	w.WriteHeader(http.StatusNoContent)
}

// notConfigured is the OAuth scaffold response until provider secrets are wired.
func notConfigured(w http.ResponseWriter, _ *http.Request) {
	writeErr(w, http.StatusNotImplemented, "not_configured", "OAuth provider not configured")
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the contract's ApiError shape.
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}

// serverError logs the real error and returns a generic 500 (no internal detail leaks to clients).
func serverError(w http.ResponseWriter, err error) {
	slog.Error("request failed", "err", err)
	writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}
