# Changelog

All notable changes to Exascale. Format: [Keep a Changelog](https://keepachangelog.com); pre-GA
SemVer `0.<milestone>.<patch>` (milestones are dependency-ordered stages, not dates — see
`docs/plans/MANAGEMENT_PLAN.md`).

## [v0.2.1] — Milestone 2: credit purchase / billing (F06)

### Added
- **Credit purchase (F06), money-in side:** `POST /v1/billing/checkout` (auth → pending purchase +
  Stripe checkout URL), `POST /v1/billing/webhook/stripe` (Stripe-signature authenticated, not
  bearer), `GET /v1/billing/purchases`. On `checkout.session.completed` the webhook marks the
  purchase paid and books credits to the ledger via the service-token `/v1/credits/purchase`,
  **idempotent on the Stripe event id**. Stripe sits behind a `StripeClient` interface — a mock runs
  the full loop locally with no keys; real Stripe is the M3 swap.
- **Proven live in k3d:** checkout 500 text credits → signed webhook → `text` balance `0 → 500`;
  **webhook replay → still 500** (deduped); forged signature → 401; purchase shows `paid`. Confirms
  the platform-core → credit-ledger service-token path end to end.
- Contract `openapi/platform-core.yaml` v1.3.0 (billing surface + `Purchase`); migration
  `0002_billing.sql` (purchases, `stripe_event_id` UNIQUE = mint idempotency anchor).

### Security
- F06 review gate: webhook signatures verified by HMAC-SHA256 (Stripe's `t=,v1=` scheme),
  constant-time, with a replay/skew window; **fail-closed** when no secret is configured; the mint
  is idempotent (no double-credit on replay); `is_paper` threads tenant → purchase → ledger; checkout
  + booking are audited; webhook body is size-limited; no Stripe secrets committed. Known follow-ups
  (M3 / F22, non-blocking for the paper sandbox): KYC gate before real-money purchases, FX/multi-
  currency pricing, and the real Stripe checkout-session client.

## [v0.2.0] — Milestone 2: first inference dollar (sandbox) — inference gateway (F08)

### Added
- **Inference gateway (F08):** the Exascale OpenAI-compatible inference API. `GET /v1/models`
  (curated catalog: Llama-70B/8B + Whisper, with fixed-point pricing); `POST /v1/chat/completions`
  on a swappable `model.Backend` (mock now, vLLM in F09) with JSON + **SSE** streaming. Customer
  auth by **API key** (resolved via platform-core introspection, 30s cache) or first-party JWT.
  Distroless/non-root deploy + Tilt.
- **The billing loop (F08 ↔ F05), proven live in k3d:** a request emits exactly one
  `inference.usage.v1` (fixed-point `units`, `request_id` = idempotency key); credit-ledger consumes
  it over a **durable JetStream** consumer and debits via `ApplyMovement` (atomic, hash-chained,
  idempotent — redelivery never double-bills). End-to-end: signup → API key →
  `curl /v1/chat/completions` → `text` balance `100.000000 → 99.865000`.
- **Pre-flight 402:** a zero-credit tenant is rejected with `INSUFFICIENT_CREDIT`
  (+ balance/required/buy-credits link) before any GPU is touched.
- Contract: `openapi/inference.yaml` v1.0.0 (new); `openapi/platform-core.yaml` v1.2.0 (internal
  API-key introspection endpoint + `serviceToken` scheme).

### Security
- F08 review gate: JWT verification is HMAC-only (alg-confusion rejected); API keys resolve via a
  service-token-guarded introspection endpoint (404 on unknown/revoked — no enumeration); the
  `SERVICE_TOKEN` is generated, never committed; `is_paper` threads tenant → usage event → debit;
  units are exact fixed-point (`math/big.Rat`), never float. Known follow-ups (non-blocking for the
  sandbox): asymmetric JWT so services can't mint, and a balance hold to close the pre-flight
  fail-open window.

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
