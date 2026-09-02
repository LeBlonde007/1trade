package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/domain"
)

// principal is the authenticated caller resolved from a platform JWT.
type principal struct {
	UserID   string
	TenantID string
	OrgID    string
	Roles    []domain.Role
	IsPaper  bool
}

// authPrincipal verifies the bearer JWT (the one platform-core itself issued) and returns the caller.
func authPrincipal(cfg config.Config, r *http.Request) (principal, error) {
	tok := bearer(r)
	if tok == "" {
		return principal{}, fmt.Errorf("missing bearer token")
	}
	c, err := domain.VerifyToken(cfg.JWTSecret, tok)
	if err != nil {
		return principal{}, err
	}
	return principal{
		UserID: c.Subject, TenantID: c.TenantID, OrgID: c.OrgID, Roles: c.Roles, IsPaper: c.IsPaper,
	}, nil
}

// authed resolves the caller from the bearer JWT; on failure it writes a generic 401 and returns
// ok=false, so handlers can guard with `p, ok := s.authed(w, r); if !ok { return }`.
func (s *Server) authed(w http.ResponseWriter, r *http.Request) (principal, bool) {
	p, err := authPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return principal{}, false
	}
	return p, true
}

// requireRole enforces RBAC: if the principal lacks `role` (admin satisfies any), it writes a 403
// and returns false. Guard sensitive handlers with `if !requireRole(w, p, domain.RoleEngineer) { return }`.
func requireRole(w http.ResponseWriter, p principal, role domain.Role) bool {
	if domain.HasRole(p.Roles, role) {
		return true
	}
	writeErr(w, http.StatusForbidden, "forbidden", "insufficient role")
	return false
}

// serviceAuthorized reports whether the request carries the valid service-to-service token. The
// comparison is constant-time so a wrong token can't be discovered byte-by-byte via timing. A
// service token that is unset always denies (fail closed).
func serviceAuthorized(cfg config.Config, r *http.Request) bool {
	tok := bearer(r)
	return cfg.ServiceToken != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(cfg.ServiceToken)) == 1
}

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}
