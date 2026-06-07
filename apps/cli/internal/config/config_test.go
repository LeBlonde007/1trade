package config

import (
	"os"
	"testing"
)

// withCleanEnv points HOME at a temp dir (so Load doesn't read a real ~/.exascale) and clears every
// EXASCALE_* override, so each test starts from the built-in defaults.
func withCleanEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"EXASCALE_API_URL", "EXASCALE_PLATFORM_URL", "EXASCALE_GATEWAY_URL", "EXASCALE_LEDGER_URL", "EXASCALE_COMPUTE_URL", "EXASCALE_TOKEN"} {
		_ = os.Unsetenv(k)
	}
}

// TestAPIURLSetsAllServices verifies EXASCALE_API_URL points every service URL at one host — the
// sandbox/prod path where a single api gateway routes by path prefix.
func TestAPIURLSetsAllServices(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("EXASCALE_API_URL", "https://api.exascale.ai")
	c := Load()
	for name, got := range map[string]string{"platform": c.PlatformURL, "gateway": c.GatewayURL, "ledger": c.LedgerURL, "compute": c.ComputeURL} {
		if got != "https://api.exascale.ai" {
			t.Fatalf("%s URL = %q, want the api host", name, got)
		}
	}
}

// TestPerServiceOverridesAPIURL verifies a specific EXASCALE_*_URL still wins over EXASCALE_API_URL,
// so a split deployment (e.g. inference on its own host) can override just that service.
func TestPerServiceOverridesAPIURL(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("EXASCALE_API_URL", "https://api.exascale.ai")
	t.Setenv("EXASCALE_GATEWAY_URL", "https://infer.exascale.ai")
	c := Load()
	if c.GatewayURL != "https://infer.exascale.ai" {
		t.Fatalf("gateway URL = %q, want the per-service override", c.GatewayURL)
	}
	if c.PlatformURL != "https://api.exascale.ai" {
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
