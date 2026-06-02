// Package auth resolves an inbound bearer credential to a Principal. Customer reads present a
// first-party tenant JWT (verified locally with the shared HS256 secret — no network call); internal
// scheduling calls (gateway/runtime → compute-control) present the service token. Either way we learn
// tenant_id / sub_account_id / is_paper, which flow into quota + the compute.usage.v1 event.
package auth

import (
	"crypto/subtle"
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
	Service      bool // true when authenticated with the service token (internal calls)
}

// Resolver verifies credentials against the shared HS256 secret + the service token.
type Resolver struct {
	jwtSecret    string
	serviceToken string
}

// NewResolver builds a Resolver.
func NewResolver(jwtSecret, serviceToken string) *Resolver {
	return &Resolver{jwtSecret: jwtSecret, serviceToken: serviceToken}
}

// IsService reports whether the bearer is the configured service token (constant-time compare).
func (r *Resolver) IsService(bearer string) bool {
	if r.serviceToken == "" || bearer == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(bearer), []byte(r.serviceToken)) == 1
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
