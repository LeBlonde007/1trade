package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/exascale/credit-ledger/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// principal is the authenticated caller resolved from the request.
type principal struct {
	TenantID string
	IsPaper  bool
	Service  bool // true for internal service-to-service calls
}

// tenantPrincipal resolves a customer principal from the platform JWT (HS256, shared secret).
// In dev, an X-Dev-Tenant header is accepted as a shortcut so the ledger is testable without
// platform-core running (X-Dev-Paper defaults to true). Never enabled outside EXASCALE_ENV=dev.
func tenantPrincipal(cfg config.Config, r *http.Request) (principal, error) {
	if cfg.IsDev() {
		if t := r.Header.Get("X-Dev-Tenant"); t != "" {
			return principal{TenantID: t, IsPaper: r.Header.Get("X-Dev-Paper") != "false"}, nil
		}
	}
	tok := bearer(r)
	if tok == "" || cfg.JWTSecret == "" {
		return principal{}, fmt.Errorf("missing or unverifiable bearer token")
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return principal{}, fmt.Errorf("invalid token: %w", err)
	}
	tenantID, _ := claims["tenant_id"].(string)
	if tenantID == "" {
		return principal{}, fmt.Errorf("token missing tenant_id")
	}
	isPaper, _ := claims["is_paper"].(bool)
	return principal{TenantID: tenantID, IsPaper: isPaper}, nil
}

// servicePrincipal authorises an internal endpoint: a matching service bearer token, or (dev only)
// the X-Dev-Tenant shortcut. Internal callers pass the acting tenant in the request body.
func servicePrincipal(cfg config.Config, r *http.Request) (principal, error) {
	if cfg.IsDev() && r.Header.Get("X-Dev-Tenant") != "" {
		return principal{Service: true}, nil
	}
	// Constant-time compare so a wrong token can't be discovered byte-by-byte via response timing.
	if tok := bearer(r); cfg.ServiceToken != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(cfg.ServiceToken)) == 1 {
		return principal{Service: true}, nil
	}
	return principal{}, fmt.Errorf("service authorization required")
}

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}
