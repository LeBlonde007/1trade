// Package auth resolves an inbound bearer credential to a Principal. A customer presents either an
// API key (`exk_...`, resolved via platform-core's introspection endpoint, then cached briefly) or a
// first-party tenant JWT (verified locally with the shared HS256 secret — no network call). Either
// way the gateway learns tenant_id / sub_account_id / scopes / is_paper, which it propagates into
// the credit pre-check and the usage event.
package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/trade1/inference-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// ErrUnauthenticated is returned when a credential is missing, malformed, or rejected. The API
// layer maps every auth error to a generic 401 (no detail about why, to avoid an oracle).
var ErrUnauthenticated = errors.New("unauthenticated")

// Principal is the authenticated customer resolved from the credential.
type Principal struct {
	TenantID     string
	SubAccountID string
	Scopes       []string
	IsPaper      bool
}

// cacheEntry is a cached introspection result with its expiry.
type cacheEntry struct {
	p   Principal
	exp time.Time
}

// Resolver authenticates credentials. API-key results are cached for cacheTTL to avoid a
// platform-core round trip per inference request; JWTs are verified locally every time (cheap).
type Resolver struct {
	platformURL  string
	serviceToken string
	jwtSecret    string
	http         *http.Client
	cacheTTL     time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry // key: hex(sha256(api_key))
}

// NewResolver builds a Resolver from config.
func NewResolver(cfg config.Config) *Resolver {
	return &Resolver{
		platformURL:  strings.TrimRight(cfg.PlatformCoreURL, "/"),
		serviceToken: cfg.ServiceToken,
		jwtSecret:    cfg.JWTSecret,
		http:         &http.Client{Timeout: cfg.HTTPTimeout},
		cacheTTL:     30 * time.Second,
		cache:        make(map[string]cacheEntry),
	}
}

// Resolve authenticates a raw bearer credential (the token after "Bearer "). An `exk_` prefix is an
// API key (introspected + cached); anything else is treated as a tenant JWT (verified locally).
func (rv *Resolver) Resolve(ctx context.Context, bearer string) (Principal, error) {
	if bearer == "" {
		return Principal{}, ErrUnauthenticated
	}
	if strings.HasPrefix(bearer, "exk_") {
		return rv.resolveAPIKey(ctx, bearer)
	}
	return rv.resolveJWT(bearer)
}

// resolveAPIKey returns the cached principal for a key, or introspects it via platform-core and
// caches the result. The cache is keyed by the key's sha256 (never the raw secret).
func (rv *Resolver) resolveAPIKey(ctx context.Context, key string) (Principal, error) {
	sum := sha256.Sum256([]byte(key))
	ck := hex.EncodeToString(sum[:])

	rv.mu.Lock()
	if e, ok := rv.cache[ck]; ok && time.Now().Before(e.exp) {
		rv.mu.Unlock()
		return e.p, nil
	}
	rv.mu.Unlock()

	p, err := rv.introspect(ctx, key)
	if err != nil {
		return Principal{}, err
	}
	rv.mu.Lock()
	rv.cache[ck] = cacheEntry{p: p, exp: time.Now().Add(rv.cacheTTL)}
	rv.mu.Unlock()
	return p, nil
}

// introspect calls platform-core's internal key-introspection endpoint (service-token auth) to
// resolve a raw API key. Any non-200 is treated as unauthenticated.
func (rv *Resolver) introspect(ctx context.Context, key string) (Principal, error) {
	body, _ := json.Marshal(map[string]string{"api_key": key})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rv.platformURL+"/v1/auth/keys/introspect", bytes.NewReader(body))
	if err != nil {
		return Principal{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rv.serviceToken)

	resp, err := rv.http.Do(req)
	if err != nil {
		return Principal{}, fmt.Errorf("introspect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Principal{}, ErrUnauthenticated
	}
	var out struct {
		TenantID     string   `json:"tenant_id"`
		SubAccountID *string  `json:"sub_account_id"`
		Scopes       []string `json:"scopes"`
		IsPaper      bool     `json:"is_paper"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Principal{}, err
	}
	if out.TenantID == "" {
		return Principal{}, ErrUnauthenticated
	}
	p := Principal{TenantID: out.TenantID, Scopes: out.Scopes, IsPaper: out.IsPaper}
	if out.SubAccountID != nil {
		p.SubAccountID = *out.SubAccountID
	}
	return p, nil
}

// resolveJWT verifies a first-party tenant JWT locally (HS256, shared secret; HMAC-only to reject
// alg-confusion) and reads tenant_id / is_paper / sub_account_id from its claims.
func (rv *Resolver) resolveJWT(tokenStr string) (Principal, error) {
	if rv.jwtSecret == "" {
		return Principal{}, ErrUnauthenticated
	}
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(rv.jwtSecret), nil
	})
	if err != nil || !tok.Valid {
		return Principal{}, ErrUnauthenticated
	}
	tenantID, _ := claims["tenant_id"].(string)
	if tenantID == "" {
		return Principal{}, ErrUnauthenticated
	}
	isPaper, _ := claims["is_paper"].(bool)
	sub, _ := claims["sub_account_id"].(string)
	return Principal{TenantID: tenantID, SubAccountID: sub, IsPaper: isPaper}, nil
}
