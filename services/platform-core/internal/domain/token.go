package domain

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the platform JWT payload — the contract every service reads (openapi/platform-core.yaml
// JwtClaims). The custom claims use the exact JSON keys consumers expect (`tenant_id`, `is_paper`,
// `roles`), so a token issued here is accepted unchanged by credit-ledger and the rest of the fleet.
type Claims struct {
	TenantID     string `json:"tenant_id"`
	OrgID        string `json:"org_id,omitempty"`
	SubAccountID string `json:"sub_account_id,omitempty"`
	Roles        []Role `json:"roles"`
	IsPaper      bool   `json:"is_paper"`
	jwt.RegisteredClaims
}

// IssueToken signs an HS256 JWT for a user. `sub` is the user id; `ttl` bounds validity. The shared
// secret is the same one consuming services verify with (PLATFORM_JWT_SECRET).
func IssueToken(secret, sub string, c Claims, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("empty signing secret")
	}
	now := time.Now().UTC()
	c.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   sub,
		Issuer:    "platform-core",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// VerifyToken validates an HS256 token (signature, expiry, and that the signing method really is
// HMAC — rejecting alg-confusion attacks) and returns its claims. Requires a tenant_id.
func VerifyToken(secret, tokenStr string) (*Claims, error) {
	c := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if c.TenantID == "" {
		return nil, fmt.Errorf("token missing tenant_id")
	}
	return c, nil
}
