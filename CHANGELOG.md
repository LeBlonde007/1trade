# Changelog

All notable changes to Exascale. Format: [Keep a Changelog](https://keepachangelog.com); pre-GA
SemVer `0.<milestone>.<patch>` (milestones are dependency-ordered stages, not dates — see
`docs/plans/MANAGEMENT_PLAN.md`).

## [v0.2.5] — Milestone 2: wallet convert UI — live (F20 × F07)

### Added
- **The wallet convert drawer now executes real conversions** (`Exascale Frontend`). In `local` mode
  the showcase drawer is driven by live data: published rate + house spread for the selected pair,
  and the submit button calls the ledger to atomically burn `from` / mint `to`, then refreshes
  balances and toasts the result. `mock` mode is untouched — same drawer, canned cross-rate.
- **BFF routes:** `GET /api/wallet/conversion-rates` and `POST /api/wallet/convert` (Nitro), proxying
  credit-ledger `/v1/credits/{conversion-rates,convert}`. `is_paper` is derived from the session JWT
  (never the client); the client-supplied `rate` is advisory/mock-only and **cannot** influence a
  real conversion (the ledger looks the rate up server-side). The idempotency key is minted in
  `useWallet.convert` so a retried submit de-dupes instead of double-converting.
- **`useWallet`** gains `loadConversionRates()`, `rateFor(from,to)`, `convert(from,to,amount)`, and a
  `converting` flag.

### Proven live (through the running frontend BFF)
- signup → mint `ai_index` → `GET /api/wallet/conversion-rates` → `POST /api/wallet/convert` 100
  ai_index → **82.227321 text** (−1% spread), balances 900 / 82.227321.
- **Idempotent replay** through the BFF leaves balances unchanged (one conversion only).
- **Error propagation:** an unseeded pair returns **HTTP 422 `no_rate`** with `data.message`, which
  the drawer surfaces as a toast; `402 INSUFFICIENT_CREDIT` propagates the same way.

### Notes
- Live conversion is gated to seeded pairs (`ai_index ↔ text` today); other pairs show an indicative
  rate and a "no live rate" note until M3 adds GPU-tier / other-modality rates. Copy stays
  "convert" (prepaid framing), never "trade"/"exchange".

## [v0.2.4] — Milestone 2: credit conversion (F07)

### Added
- **Credit conversion (F07):** `POST /v1/credits/convert` + `GET /v1/credits/conversion-rates` (the
  AI-index ↔ sub/GPU mechanism — the thesis's "index converts into any compute"). M2 ships
  `ai_index ↔ text`.
- **Atomic + correct:** one DB transaction burns `from` and mints `to` as two chained legs (each
  extends its own balance's hash chain), with **distinct per-leg idempotency keys** so they don't
  shadow each other. Rate read from a `conversion_rates` table (no hardcoding); amount =
  `floor(amount × rate × (1−spread))` in exact fixed-point (1% house spread, ADR-0002); the floor
  means the house never over-credits.
- **Proven live in k3d:** convert 100 ai_index → 82.227321 text (value − 1% spread); idempotent
  replay leaves balances unchanged; insufficient → 402; unknown pair → 422; **hash chain verifies**
  after (checked 3, ok). Tests: domain math + store integration.

### Notes
- M3: the other modalities + GPU-tier rates, and the wallet "convert" UI (copy stays
  "convert"/"redeem", never "trade"/"exchange" — prepaid framing).

## [v0.2.3] — Milestone 2: platform console — live data layer + console (F20)

### Added
- **Mock→live seam (F20):** a Nitro BFF (`server/api/**`) so every screen calls same-origin `/api/**`
  with the JWT in an httpOnly cookie; `EXASCALE_API_MODE=mock|local` flips canned↔live with zero UI
  change. BFF routes for auth, catalog, wallet (balances/transactions), inference, billing checkout,
  and API keys — each mock|live. Composables: `useAuth/useCatalog/useWallet/useInference/useBilling/
  useKeys`. `login` + `signup` wired to real auth; route guard.
- **Live `/console`:** a dense, dark, institutional dashboard (design-system tokens only, mono +
  tabular numbers, semantic ▲/▼, sharp radius) wired to all six composables — identity, balances,
  model picker, inference playground (run → token usage + est. cost; 402 → buy-credits), buy-credits
  (Stripe checkout), API keys (generate → secret-shown-once → revoke), recent transactions.
- **Proven live** against the running platform: signup→identity, httpOnly session, catalog→real 3
  models, wallet→ledger, key create (secret once), checkout→Stripe URL, inference 402 surfaced; and
  the console run path: seed 200 text → Run → 200 (tokens 7/13/20) → wallet 199.900000.

### Notes
- The large existing mock screens (`inference.vue`, `wallet/index.vue`, …) remain as the polished
  showcase; the live data layer + `/console` deliver the M2 "catalog + wallet wired" goal. Wiring
  those individual screens to the composables is incremental UI work (browser-iterated).

## [v0.2.2] — Milestone 2: vLLM runtime integration (F09)

### Added
- **inference-runtime (F09):** `services/inference-runtime/server.py` — a production vLLM worker
  (FastAPI + `AsyncLLMEngine`, OpenAI-compatible `/v1/chat/completions` + `/healthz` `/readyz`
  `/metrics`; weights mount from a PVC, not baked into the image; exact token usage). `Dockerfile.vllm`
  (CUDA, pinned vLLM) + a GPU `Deployment` (`nvidia.com/gpu`, nodeSelector/tolerations, readiness-gated
  weight load, accurate requests/limits). The internal gateway↔runtime contract is documented in the
  runtime README.
- **The mock→real swap:** the gateway's `model.VLLMBackend` calls the runtime over HTTP and bills
  from the runtime's real token usage; `INFERENCE_BACKEND=vllm` selects it with **no customer-API
  change**. Unit-tested (success, usage-omitted fallback, runtime error).
- **CPU stub runtime** (`stub/`, zero deps) so the whole path runs in k3d without a GPU. **Proven
  live:** gateway `backend: vllm` → `/v1/chat/completions` served by the runtime → tokens from the
  runtime usage → `text` balance `100 → 99.900000` (`5 × 20/1000`).

### Notes
- GPU-bound acceptance (3 models load/serve, P50/P95 latency, OOM-safe under load, crash re-route)
  needs **F12 (GPU Operator) + a 40 GB+ GPU node** — the artifacts are ready; verification is deferred
  to a staging GPU node. Per-model routing (model→pod registration/heartbeat) lands with F11.

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

## [v0.1.4] — Milestone 1: exascale CLI v0 (F04)

### Added
- **`exascale` CLI (F04):** the primary engineer interface — a thin Go client over the live platform
  (`apps/cli`). Commands: `login`/`signup` (hidden password)/`whoami`/`logout`; `credits
  balance`/`transactions`/`convert`; `catalog`; `infer chat -m MODEL "prompt"`; `keys
  create`/`list`/`revoke`; `config get`/`set`. Token in `~/.exascale/config.json` (0600); per-service
  URLs with env overrides + dev defaults. Client unit-tested.
- **Proven live:** `login → whoami → catalog → credits balance → convert (ai_index→text) → infer
  chat`. M1 v0 acceptance (login/whoami/credits balance) met, plus the live M2/M3 commands whose
  backends already exist.

### Notes
- `gpu`/`cluster`/`train`/`billing` commands wire in as F12/F13 land; distribution
  (brew/apt/pip/install.sh) is M6.

## [v0.1.3] — Milestone 1: platform gap-closers (email verify, budgets, linked endpoints)

### Added
- **Email verification:** signup issues a one-time token (only its hash is stored; the raw token is
  emailed in prod / logged in dev). `POST /v1/auth/verify` (the token is the credential — no bearer,
  single-use, audited), `POST /v1/auth/verify/resend`. The existing `onboarding/verify` page now
  consumes the magic-link `?token`.
- **Monthly budgets:** `GET`/`PUT /v1/billing/budget` (billing/admin, audited) — the basis for
  50/80/100% consumption alerts (auto-stop is M3).
- **Linked the orphan endpoints:** BFF routes for verify, budget, purchase history, and the audit
  log, so they're reachable from the UI (no more backend-only endpoints).
- Contract `platform-core.yaml` v1.5.0. Proven live: resend → verify (single-use) → reuse 400;
  budget set/get; audit captures `user.email.verify` + `budget.set`.

## [v0.1.2] — Milestone 1: accounts, orgs & RBAC (F03)

### Added
- **Audit log (F03):** `audit_log` table + `admin.action.v1` contract; every sensitive action
  (signup, API-key create/revoke, org create, role assignment) records a tenant-scoped row with
  actor + before/after. `GET /v1/account/audit` — queryable trail, admin-only (a SOC 2 control).
- **RBAC enforcement:** `requireRole` (admin satisfies any) → 403; applied to key + org + role
  management.
- **Org management:** create/list orgs (admin), assign user roles (admin, role-validated + audited),
  read own tenant (cross-tenant → 404, no info leak), list org users — all tenant-scoped. Contract
  `platform-core.yaml` v1.4.0.
- **Proven live in k3d:** signup → create org → audit `[org.create, tenant.signup]`; viewer-only →
  403 on key/org create + audit read.

### Notes
- Sub-accounts + per-team budgets + the `admin.action.v1` NATS fan-out are M4 (the DB audit log is
  authoritative today). SAML/SCIM are M4 (F02).

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
