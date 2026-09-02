package config

import (
	"os"
	"testing"
)

// withCleanEnv points HOME at a temp dir (so Load doesn't read a real ~/.1trade) and clears every
// TRADE1_* override, so each test starts from the built-in defaults.
func withCleanEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"TRADE1_API_URL", "TRADE1_PLATFORM_URL", "TRADE1_GATEWAY_URL", "TRADE1_LEDGER_URL", "TRADE1_COMPUTE_URL", "TRADE1_TOKEN"} {
		_ = os.Unsetenv(k)
	}
}

// TestAPIURLSetsAllServices verifies TRADE1_API_URL points every service URL at one host — the
// sandbox/prod path where a single api gateway routes by path prefix.
func TestAPIURLSetsAllServices(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TRADE1_API_URL", "https://api.1trade.ai")
	c := Load()
	for name, got := range map[string]string{"platform": c.PlatformURL, "gateway": c.GatewayURL, "ledger": c.LedgerURL, "compute": c.ComputeURL} {
		if got != "https://api.1trade.ai" {
			t.Fatalf("%s URL = %q, want the api host", name, got)
		}
	}
}

// TestPerServiceOverridesAPIURL verifies a specific TRADE1_*_URL still wins over TRADE1_API_URL,
// so a split deployment (e.g. inference on its own host) can override just that service.
func TestPerServiceOverridesAPIURL(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("TRADE1_API_URL", "https://api.1trade.ai")
	t.Setenv("TRADE1_GATEWAY_URL", "https://infer.1trade.ai")
	c := Load()
	if c.GatewayURL != "https://infer.1trade.ai" {
		t.Fatalf("gateway URL = %q, want the per-service override", c.GatewayURL)
	}
	if c.PlatformURL != "https://api.1trade.ai" {
		t.Fatalf("platform URL = %q, want the api host", c.PlatformURL)
	}
}

// TestDefaultsWhenUnset verifies the local-dev defaults apply when nothing is set.
func TestDefaultsWhenUnset(t *testing.T) {
	withCleanEnv(t)
	c := Load()
	if c.PlatformURL != "http://localhost:8001" {
		t.Fatalf("platform URL = %q, want local default", c.PlatformURL)
	}
}
