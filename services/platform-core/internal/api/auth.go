package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/domain"
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

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}
