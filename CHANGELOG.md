# Changelog

All notable changes to Exascale. Format: [Keep a Changelog](https://keepachangelog.com); pre-GA
SemVer `0.<milestone>.<patch>` (milestones are dependency-ordered stages, not dates — see
`docs/plans/MANAGEMENT_PLAN.md`).

## [v0.1.1] — Milestone 1: auth & SSO (F02)

### Added
- **Platform-core auth (F02):** the fleet's identity service. Self-serve `signup` (creates an
  individual tenant + admin user, auto-login), `login` → HS256 JWT, `me`, `logout`, and scoped
  **API keys** (create → secret shown once, list, tenant-scoped revoke). OAuth endpoints scaffolded
  (501 until provider secrets are wired). Postgres schema (`tenants`/`orgs`/`users`/`api_keys`),
  self-migrating Kubernetes deploy (distroless, non-root), Tilt wiring.
- **Cross-service auth (F02 ↔ F05):** one signup at platform-core issues a JWT that the separately
  deployed credit-ledger verifies on its own — resolving `tenant_id`/`is_paper` straight from the
  token. Both services read the same `PLATFORM_JWT_SECRET` from the shared `platform-auth` Secret
  (generated locally, never committed; SOPS/Vault in prod). Verified end-to-end in k3d.
- Contract: `openapi/platform-core.yaml` → v1.1.0 (added `/v1/auth/signup`).

### Security
- F02 review gate: JWT verification rejects alg-confusion (HMAC-only) and tokens without
  `tenant_id`; passwords are bcrypt; API-key secrets are high-entropy random, stored only as
  sha256, compared in constant time. Identity is case-insensitive (canonicalised email) so case
  variants can't create shadow accounts. Login closes the user-enumeration **timing** oracle
  (equal bcrypt cost on the not-found path). Client errors are generic; audit hooks on
  signup / key-create / key-revoke. `gitleaks` clean — the signing secret is never committed.

### Fixed
- `openapi/platform-core.yaml`: `created_at:{` (missing space) that broke YAML parsing; all
  contracts now pass a parse check.

## [v0.1.0] — Milestone 1: foundation + credit ledger

### Added
- **Foundation infra (F01):** Kubernetes via k3s/k3d, core data plane (Postgres, TimescaleDB,
  Redis, NATS) — `make up`, reproducible from code; Docker templates; CI; repo scaffolding.
- **Credit ledger (F05):** the financial core — append-only, cryptographically hash-chained
  (`sha256(prev ‖ row)`), exact fixed-point money (no floats), atomic balance + transaction writes,
  **per-tenant** idempotency, `is_paper` isolation. HTTP API per `openapi/credit.yaml` (balances,
  transactions, purchase, debit, mint, burn, chain-verify; `/convert` → 501, F07). `credit.tx.v1`
  NATS events. Self-migrating Kubernetes deploy. Verified end-to-end in k3d.
- **Contracts:** `credit-types.md`, `schemas/types.sql`, `openapi/credit.yaml`,
  `openapi/platform-core.yaml`, events (`credit.tx.v1`, `inference.usage.v1`, `compute.usage.v1`).
- **Frontend:** 31-screen Nuxt app on mock data; server-enforced site password gate.

### Security
- F05 review gate: client error responses are generic (internal detail logged server-side, never
  returned) and the service-token comparison is constant-time. `gitleaks` clean; `govulncheck`
  reports 0 vulnerabilities affecting the code.

### Notes
- The exchange (order book, market maker, index) is designed; not part of this release.
- Follow-ups (non-blocking): bump `golangci-lint` to a Go-1.25-compatible version; add Prometheus
  `/metrics`; Grafana dashboard + RUNBOOK; scheduled reconciliation job; load/perf test.
