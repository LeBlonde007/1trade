// Package auth resolves an inbound bearer credential to a Principal. The matching engine only accepts
// first-party tenant JWTs, verified locally with the shared HS256 secret (no network call): market data
// is public, and the tenant-scoped reads (orders, positions, fills) need a verified tenant_id.
//
// There is no service-token path here, unlike compute-control. Nothing internal calls this service in
// Phase 1 — it emits no events and settles nothing — so granting a service credential would widen the
// surface for no reason.
package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// ErrUnauthenticated is returned when a credential is missing, malformed, or rejected. The API layer
// maps every auth error to a generic 401 (no detail about why, to avoid an oracle).
var ErrUnauthenticated = errors.New("unauthenticated")

// Principal is the authenticated caller resolved from the credential.
type Principal struct {
	TenantID     string
	SubAccountID string
	IsPaper      bool
}

// Resolver verifies credentials against the shared HS256 secret.
type Resolver struct {
	jwtSecret string
}

// NewResolver builds a Resolver.
func NewResolver(jwtSecret string) *Resolver {
	return &Resolver{jwtSecret: jwtSecret}
}

// ResolveJWT verifies a first-party tenant JWT locally (HS256) and extracts the principal.
func (r *Resolver) ResolveJWT(bearer string) (Principal, error) {
	if bearer == "" || r.jwtSecret == "" {
		return Principal{}, ErrUnauthenticated
	}
	tok, err := jwt.Parse(bearer, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnauthenticated
		}
		return []byte(r.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return Principal{}, ErrUnauthenticated
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	tenantID, _ := claims["tenant_id"].(string)
	if tenantID == "" {
		return Principal{}, ErrUnauthenticated
	}
	sub, _ := claims["sub_account_id"].(string)
	isPaper, _ := claims["is_paper"].(bool)
	return Principal{TenantID: tenantID, SubAccountID: sub, IsPaper: isPaper}, nil
}
