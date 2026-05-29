# Agent plan — `platform-core`

> Active, critical-path. Owns the revenue rails (auth + billing) and the unified client (CLI).

## 1. Scope under the GTM pivot

Unchanged. Auth, accounts/orgs, RBAC, billing, gateway config, CLI. Under the pivot, billing
matters **immediately** — the platform earns the moment a customer buys credits. Trading-specific
CLI commands (`trade quote/buy/orders`) are kept in the CLI spec but their backend wiring is
deferred to Phase 2; they return a "trading paused — see /trade demo" message in Phase 1.

## 2. Features owned

| Feature | Status |
|---|---|
| [F02 — Auth & SSO](../features/F02-auth-and-sso.md) | active, M1 (email/OAuth) + M4 (SAML/SCIM/2FA) |
| [F03 — Accounts, orgs, sub-accounts, RBAC](../features/F03-accounts-orgs-rbac.md) | active, M1 (base) + M4 (sub-accounts) |
| [F04 — CLI v0](../features/F04-cli.md) | active, M1 (login/whoami/balance) + M2/3 (infer/gpu/billing) |
| [F06 — Credit purchase / billing](../features/F06-credit-purchase-billing.md) | active, M2 (Stripe) + M3 (ACH/wire + multi-currency) |
| Gateway (Kong) config | active, continuous |

## 3. Milestone-by-milestone

### Milestone 1
- `services/platform-core/` scaffolded.
- Email/password + OAuth (Google, GitHub, Microsoft).
- Tenants, orgs, users tables (no sub-accounts yet).
- RBAC: admin, billing, engineer, viewer (trader role pre-added but unused under the pivot).
- API keys (scoped, hashed at rest).
- Kong: routes for `/v1/auth/*`, `/v1/account/*`, `/v1/credits/*`.
- CLI v0: `login` (browser OAuth flow), `logout`, `whoami`, `credits balance`.

### Milestone 2
- Stripe Connect for card-based credit purchases.
- CLI: `infer chat`, `credits purchase` (Stripe link out), `--help` polish.
- Sub-5-min onboarding: signup → `exascale login` → `exascale credits purchase $50` → `exascale infer chat -m llama-3.1-8b "hi"`. Measured in `make test-e2e`.

### Milestone 3
- ACH / wire flow for $10K+ purchases (manual reconciliation initially; automated later).
- Multi-currency: USD + JPY.
- PO-based purchasing groundwork (enterprise terms).
- CLI: `gpu create/list/stop` (uses `compute.yaml` contract), `billing today`, `credits convert`.

### Milestone 4
- SAML SSO (Okta, Entra ID, Google Workspace).
- SCIM provisioning.
- 2FA (TOTP).
- IP allowlisting.
- Sub-accounts: per-team budgets, org-wide consumption views.
- Admin-action audit log (already producing events; this milestone adds the queryable surface).

### Milestone 5
- Partner-portal scaffolding (the supply-side login; partner DC operators log in to see
  utilization + payouts). The view itself lives in `apps/web/`, owned by `trading-frontend`.
- CLI: `audit log` (read), `keys list/create/revoke` polish.

### Milestone 6
- SDKs: Python, JS, Go — thin wrappers over the public APIs.
- Documentation site collaboration with `trading-frontend` (docs route in `apps/web/`).
- CLI 1.0 release across brew/apt/pip/binary.

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/platform-core.yaml` — auth, accounts, orgs, users, RBAC, audit, billing.
- The auth JWT shape (`tenant_id`, `org_id`, `sub`, `roles`, `is_paper`, `exp`).
- API-key shape and scopes.

### Consumed
- `openapi/credit.yaml` (purchase calls go to `credit-ledger`).
- `openapi/inference.yaml`, `openapi/compute.yaml`, `openapi/index.yaml` (CLI calls them).

## 5. Local dev

- `services/platform-core/` runs on `:8001`.
- `apps/cli/` builds with `make cli`; `exascale config set api-url http://localhost:8080`,
  `exascale login --dev` uses a dev token issued by platform-core's dev endpoint.
- `make seed` creates 3 tenants (`acme-ai`, `f500-co`, `internal-mm`) and an admin user per.

## 6. Dockerfile

Uses `deploy/docker/Dockerfile.go-service` with `--build-arg SERVICE=platform-core`. CLI is
released as a static binary (cross-compiled) via `make cli-release`, not as a container.

## 7. Deploy

- K8s Deployment, 3 replicas minimum, HPA on CPU.
- Vault-injected secrets (DB creds, OAuth client secrets, Stripe API keys, SAML signing keys).
- Network policy: only `inference-gateway`, `compute-control`, and Kong may call platform-core.
- Stripe webhooks: route through Kong, signature-verified at the service.

## 8. Hard boundaries

- No matching engine logic, no ledger internals, no GPU scheduling.
- The CLI is a thin client over the public APIs — no privileged backdoor endpoints.

## 9. Definition of done

- SSO works for all four IdP types (M4).
- Billing supports Stripe (cards) + ACH/wire + POs + USD+JPY (M3).
- CLI is the documented surface for everything a customer can do.
- Admin-action audit log queryable; all CLI/web actions logged.
- Sub-5-min time-to-first-action contract tested in CI.
- All gateway routes for the platform are registered; no service is reachable without going
  through Kong.
