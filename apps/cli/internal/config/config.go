// Package config loads and saves the exascale CLI config (~/.exascale/config.json): the session
// token and the platform service URLs. Env vars override the file; sensible dev defaults apply.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the CLI's persisted settings. Token is the platform JWT from `exascale login`.
type Config struct {
	Token       string `json:"token,omitempty"`
	PlatformURL string `json:"platform_url"`
	GatewayURL  string `json:"gateway_url"`
	LedgerURL   string `json:"ledger_url"`
	ComputeURL  string `json:"compute_url"`
}

// defaults are the local-dev service URLs (prod sets these to the api.exascale.io gateway).
func defaults() Config {
	return Config{
		PlatformURL: "http://localhost:8001",
		GatewayURL:  "http://localhost:8085",
		LedgerURL:   "http://localhost:8002",
		ComputeURL:  "http://localhost:8086",
	}
}

// Path returns the config file path (~/.exascale/config.json).
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".exascale", "config.json")
}

// Load reads the config file (if any), applies defaults, then env overrides.
func Load() Config {
	c := defaults()
	if b, err := os.ReadFile(Path()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if v := os.Getenv("EXASCALE_PLATFORM_URL"); v != "" {
		c.PlatformURL = v
	}
	if v := os.Getenv("EXASCALE_GATEWAY_URL"); v != "" {
		c.GatewayURL = v
	}
	if v := os.Getenv("EXASCALE_LEDGER_URL"); v != "" {
		c.LedgerURL = v
	}
	if v := os.Getenv("EXASCALE_COMPUTE_URL"); v != "" {
		c.ComputeURL = v
	}
	if v := os.Getenv("EXASCALE_TOKEN"); v != "" {
		c.Token = v
	}
	return c
}

// Save writes the config with 0600 perms (it holds the session token).
func Save(c Config) error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), b, 0o600)
}
