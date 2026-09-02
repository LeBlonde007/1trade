// Package config loads compute-control configuration from environment variables only.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration for the compute control plane.
type Config struct {
	Env           string        // dev | staging | prod
	Addr          string        // listen address, e.g. ":8086"
	NATSURL       string        // JetStream URL for emitting compute.usage.v1
	JWTSecret     string        // shared HS256 secret to verify first-party tenant JWTs
	ServiceToken  string        // service-to-service token (internal scheduling calls: gateway → compute-control)
	Scheduler     string        // "mock" (default) | "k8s" — which scheduler backend to place pods with
	SupplySource  string        // supply_source_id attributed to placements this milestone (owned DC)
	Paper         bool          // is_paper at the runtime level; dev defaults to true
	H100Count     int           // schedulable H100s (mock-GPU count locally; real count via GPU Operator)
	H200Count     int           // schedulable H200s
	HTTPTimeout   time.Duration // upstream HTTP client timeout
	MeterInterval time.Duration // how often running instances are metered → compute.usage.v1
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	return Config{
		Env:           envOr("TRADE1_ENV", "dev"),
		Addr:          envOr("COMPUTE_ADDR", ":8086"),
		NATSURL:       envOr("NATS_URL", "nats://nats.data.svc.cluster.local:4222"),
		JWTSecret:     os.Getenv("PLATFORM_JWT_SECRET"),
		ServiceToken:  os.Getenv("SERVICE_TOKEN"),
		Scheduler:     envOr("COMPUTE_SCHEDULER", "mock"),
		SupplySource:  envOr("SUPPLY_SOURCE_ID", "dc-owned-1"),
		Paper:         envOr("TRADE1_PAPER", "true") != "false",
		H100Count:     intEnv("COMPUTE_H100_COUNT", 8),
		H200Count:     intEnv("COMPUTE_H200_COUNT", 0),
		HTTPTimeout:   5 * time.Second,
		MeterInterval: time.Duration(intEnv("COMPUTE_METER_INTERVAL_SEC", 60)) * time.Second,
	}
}

// IsDev reports whether the dev environment is active.
func (c Config) IsDev() bool { return c.Env == "dev" }

// envOr returns the env var or a fallback when unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// intEnv parses an integer env var, returning the fallback when unset or invalid.
func intEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
