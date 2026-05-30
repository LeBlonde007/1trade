package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/exascale/inference-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testSvcToken = "test-service-token"
	testSecret   = "inference-gateway-test-secret-32-chars-x"
)

// stubPlatform stands in for platform-core's introspection endpoint and counts how many times it is
// hit (to prove the resolver caches). "exk_known" resolves; anything else 404s; a wrong service
// token 401s.
func stubPlatform(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/keys/introspect", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("Authorization") != "Bearer "+testSvcToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var b struct {
			APIKey string `json:"api_key"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.APIKey != "exk_known" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tenant_id": "tenant-123", "sub_account_id": nil,
			"scopes": []string{"inference:read"}, "is_paper": true,
		})
	})
	return httptest.NewServer(mux)
}

// TestResolveAPIKey verifies an API key resolves via introspection, the result is cached (no second
// network hit), and an unknown key is rejected.
func TestResolveAPIKey(t *testing.T) {
	var hits int32
	srv := stubPlatform(t, &hits)
	defer srv.Close()

	rv := NewResolver(config.Config{PlatformCoreURL: srv.URL, ServiceToken: testSvcToken, HTTPTimeout: 2 * time.Second})
	ctx := context.Background()

	p, err := rv.Resolve(ctx, "exk_known")
	if err != nil || p.TenantID != "tenant-123" || !p.IsPaper || len(p.Scopes) != 1 {
		t.Fatalf("resolve known key: p=%+v err=%v", p, err)
	}
	// second call is served from cache — the stub is not hit again.
	if _, err := rv.Resolve(ctx, "exk_known"); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 introspection hit (cache), got %d", hits)
	}
	// unknown key → unauthenticated.
	if _, err := rv.Resolve(ctx, "exk_unknown"); err == nil {
		t.Fatal("unknown key resolved")
	}
}

// TestResolveAPIKeyWrongServiceToken ensures a misconfigured service token can't authenticate.
func TestResolveAPIKeyWrongServiceToken(t *testing.T) {
	var hits int32
	srv := stubPlatform(t, &hits)
	defer srv.Close()
	rv := NewResolver(config.Config{PlatformCoreURL: srv.URL, ServiceToken: "wrong", HTTPTimeout: 2 * time.Second})
	if _, err := rv.Resolve(context.Background(), "exk_known"); err == nil {
		t.Fatal("resolved with a wrong service token")
	}
}

// TestResolveJWT verifies a first-party tenant JWT is accepted locally (no network) and that an
// empty / bad credential is rejected.
func TestResolveJWT(t *testing.T) {
	rv := NewResolver(config.Config{JWTSecret: testSecret, HTTPTimeout: time.Second})

	claims := jwt.MapClaims{
		"tenant_id": "tenant-jwt", "is_paper": true,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	p, err := rv.Resolve(context.Background(), tok)
	if err != nil || p.TenantID != "tenant-jwt" || !p.IsPaper {
		t.Fatalf("resolve jwt: p=%+v err=%v", p, err)
	}
	if _, err := rv.Resolve(context.Background(), ""); err == nil {
		t.Fatal("empty credential resolved")
	}
	if _, err := rv.Resolve(context.Background(), "not-a-jwt"); err == nil {
		t.Fatal("garbage token resolved")
	}
}
