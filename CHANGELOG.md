# Changelog

All notable changes to Exascale. Format: [Keep a Changelog](https://keepachangelog.com); pre-GA
SemVer `0.<milestone>.<patch>` (milestones are dependency-ordered stages, not dates — see
`docs/plans/MANAGEMENT_PLAN.md`).

## [v0.3.3] — Sandbox buy actually settles (MockStripe inline-settle) (F06)

### Fixed
- **A sandbox purchase dead-ended.** MockStripe returned a placeholder `checkout.stripe.test`
  checkout URL that doesn't resolve (`DNS_PROBE_FINISHED_NXDOMAIN`), and no Stripe webhook ever fires
  in dev — so credits never minted. Now, when there's no real Stripe key (`STRIPE_SECRET_KEY`
  empty → `BillingAutoSettle`), `createCheckout` **books the credits inline** (mark-paid + ledger
  mint, exactly as the webhook would) and returns `settled: true`. Real Stripe (key set) is
  untouched — it returns `settled: false` + a hosted URL and the signed webhook books on settlement.
  The two paths are mutually exclusive (flag + MockStripe type), so there's no double-mint; the
  webhook test (which uses the same MockStripe but leaves the flag off) is unaffected, and a new
  test (`TestMockCheckoutAutoSettles`) covers the inline path.
- **`/wallet/buy` completes in place** in sandbox: on `settled`, it refreshes the wallet and shows
  "✓ Purchased — N credits added (test mode)" instead of opening the dead URL; real Stripe still
  redirects. `useBilling` + the `platform-core.yaml` checkout response carry the optional `settled`.

### Verified
- Live in k3d: signup → `/wallet/buy` → "Continue to secure checkout" → balance `0 → 100,000 text`,
  success banner shown (screenshotted). API proof: checkout returns `settled:true`, balance minted.
  `go build`/`vet`/`test` green; typecheck clean.

## [v0.3.2] — Live Stripe-only buy-credits screen (F06/F20)

### Changed
- **`/wallet/buy` is now live and Stripe-only** (was an 1,850-line standalone trader mock with
  card/wire/ACH + fake "Order placed" + mock data). Rewritten lean (~330 lines): pick a real credit
  type (`text` / `ai_index` / `speech` / `image` / `video` / `gpu_h100` / `gpu_h200`) + amount →
  `useBilling().checkout(amount, credit_type, 'usd')` → redirect to the **Stripe-hosted** checkout;
  the webhook books the credits on settlement. `amount` is credits (per the contract); the USD figure
  is **indicative** (published reference prices) with the exact charge shown at Stripe.
- Wire/ACH + JPY are removed from the screen (F06 ships Stripe only; ACH/wire/JPY are M3). All mock
  data dropped (wallet `$10,118.46`, order id, fake email, price ticker, inline card form). Now
  `middleware: 'auth'`; real wallet balance via `useWallet`; sandbox shows a test-mode note.
- The real-money **KYC gate** (v0.3.1) carries over; the `PAPER/LIVE` pill reflects `is_paper`.

### Verified
- Typecheck clean. The checkout call mirrors the proven console buy panel exactly.

## [v0.3.1] — AI-company onboarding Phase 1 → console (persona) + console KPIs + KYC gate (F20/F22)

### Added
- **AI-Company journey Phase 1 complete** (signup → verify → welcome → tour → `/console`),
  persona-aware throughout: signup sets the runtime persona from the account type + the BFF sets the
  session; `usePersona` gains `home`/`postOnboard` so each persona lands on its surface (AI company →
  `/console`, never the paused exchange); the welcome is an inference/compute quick-start (not trading).
- **Chrome de-trading-fied** for the platform persona: Topbar brand → persona home, AI-Index pill +
  USD trading balance are trader-only (AI company gets a plain Wallet link); Sidebar gains a Console
  home item. **Console** adopts the app layout + numbers-first KPI tiles (live today-delta, ▲/▼,
  activity pulse). **KYC gate** at `/wallet/buy` blocks real-money for unverified accounts (sandbox
  exempt; server enforcement = F22).

## [v0.3.0] — F13 GPU instance lifecycle (on-demand) — M3 begins

### Added
- **Customer-facing on-demand GPU instances** (`compute-control`, `compute.yaml` **v1.1.0**):
  `POST/GET /v1/compute/instances`, `GET/DELETE /v1/compute/instances/{id}`, `POST {id}/stop|start`.
  Tenant-JWT scoped (cross-tenant → 404), `is_paper` derived from the principal, `Idempotency-Key` on
  create. Versioned **Exascale ML-Stack image** catalog (`stable`/`latest`/pinned); connect info
  (ssh/jupyter/http) on running instances.
- **Shared GPU pool** (`internal/pool`): customer instances and internal scheduler jobs draw from one
  per-tier pool, so capacity is never double-counted; stop/delete release GPUs back to it.
- **Per-interval GPU metering**: a ticker emits `compute.usage.v1` (units = GPU-hours × count) →
  credit-ledger's `Compute` consumer debits the `gpu_*` tier, idempotent on `usage_id`.
- **`exascale gpu`** CLI — `types | create | list | get | stop | start | delete` (+ `compute_url`
  config / `EXASCALE_COMPUTE_URL`).
- **Live web `/compute`** — BFF (`/api/compute/*`) + `useCompute`; the instances list + provision flow
  are wired to the real service (no mock mode).

### Changed
- Scheduler refactored onto the shared pool (`NewMockWithPool`); `compute.yaml` → v1.1.0
  (back-compatible with the M2 catalog/quota/jobs surface).
- `.golangci.yml`: `revive unhandled-error` now ignores infallible writers (`fmt.*Print*`,
  `strings.Builder`/`bytes.Buffer`); the 4 sha256/hmac hash writes in `chain.go`/`stripe.go` use
  explicit `_, _ =`. Closes a latent gap — all 5 modules are now lint-clean on a **cold** cache.

### Verified
- **Live e2e on k3d:** signup → mint `gpu_h100` → **create instance** (running + ssh connect info) →
  list → run 8s → **stop** → async debit: `gpu_h100 100.000000 → 99.997778` (8s × 1 GPU ÷ 3600 =
  0.002222 GPU-h, exact). Tenant-scoped, `is_paper` respected.
- Unit + API tests green (manager lifecycle/capacity/metering with a controllable clock; shared-pool
  instance↔job contention → 402). golangci-lint v2.12.2 clean; `/security-review` clean.
- **Mock-GPU backend** — the real Kueue+Volcano provisioner (same interface) and the **<90s-P95**
  real-hardware timing remain GPU-node-gated; idle auto-stop enforcement lands with the F06 policy.

## [v0.2.16] — golangci-lint v2 (Go 1.25) — lint is a hard CI gate again (F01 tooling)

### Changed
- **Bumped golangci-lint v1.59.1 → v2.12.2** (Go-1.25 compatible) and migrated `.golangci.yml` to the
  v2 schema. Lint runs across every module via a new `scripts/lint.sh` (golangci-lint at the repo
  root reaches nothing in a multi-module repo) and is now a **hard gate** in CI (was advisory in
  v0.2.15) — all 5 modules are lint-clean. Version bumped everywhere: CI, the pre-commit hook (now a
  local hook → `scripts/lint.sh`, multi-module aware), and `install-toolchain.sh`. `make lint` →
  `scripts/lint.sh` (`make lint FIX=1` to auto-fix).
- **Config tuning** for context-appropriate cases: the CLI is exempt from the `fmt.Print*` ban (it
  writes to stdout); tests are exempt from `noctx`/`errorlint`/`bodyclose` (httptest + sentinel `==`);
  `main.go` from `gocritic` exitAfterDefer (idiomatic startup); gosec `G101` excluded (false positives
  on credit-type code constants + public URLs — real secrets are caught by gitleaks + env/Secrets).
  **`wrapcheck` is deferred** (documented): a blanket enable is ~85 mechanical `%w` wraps; errcheck +
  errorlint already cover the correctness core. Re-enable after a dedicated wrapping sweep (tech-lead).

### Fixed
- Genuine lint findings: `compute-control` `writeSchedErr` now uses `errors.Is` (wrapped-sentinel
  safe) + preallocates the catalog slice; removed an unused `notImplemented` helper in `credit-ledger`;
  the CLI HTTP client uses `http.NewRequestWithContext`; a platform-core hash-determinism test no
  longer trips `SA4000` (identical-expression compare).

### Verified
- `scripts/lint.sh` green across all 5 modules on v2.12.2; `go-all.sh build` + `test -race` still
  green; `.golangci.yml`, `.pre-commit-config.yaml`, and `ci.yml` parse. With this, the CI `go` job
  (build + race-test + lint) is fully green; the remaining red on the `hygiene` job is its *other*
  pre-commit hooks (eslint/ruff/formatting), outside the golangci-lint bump.

## [v0.2.15] — CI actually builds + tests the Go services (F01)

### Fixed
- **The CI `go` job was a silent no-op.** Each `services/*` and `apps/*` is its own Go module, so the
  old `go build ./...` / `go test ./...` from the repo root reached **zero** packages (no root
  module) — and `setup-go` pinned **1.23**, which can't even build modules that declare `go 1.25.0`.
  CI compiled and tested nothing. Now: `setup-go` → **1.25**, and a new `scripts/go-all.sh` loops
  every module (`go build`/`go test -race ./...`, auto-discovered, `GOWORK=off` for deterministic
  per-module builds), so CI genuinely builds + race-tests all five modules. `make build` / `make test`
  use the same script (one source of truth).
- **The contracts job's YAML check crashed on multi-document manifests.** It used `yaml.safe_load`
  (single-doc) on `deploy/k8s/local/*.yaml`, but `data-plane.yaml` has `---` separators → the step
  always failed. Switched to `safe_load_all` and extended it to cover `deploy/k8s/scheduling/*.yaml`.

### Changed
- The opinionated lint steps are **advisory** (`continue-on-error`) until their cleanups land: the go
  job's **golangci-lint** (the v1.59.1 pin can't parse Go 1.25 and `.golangci.yml` is still v1 format
  — tracked: golangci-lint v2 bump) and the contracts job's **redocly** spec lint (11 findings —
  `security-defined` + 3.0/3.1 nullable — live in shared specs only tech-lead may revise). The hard
  gates (build, race-test, structural YAML parse) now pass for real; these surface findings without
  blocking. The `hygiene` (pre-commit) job's full green is gated on the same golangci-lint bump.

### Verified
- Locally reproduced: `scripts/go-all.sh build` + `... test -race` green across all 5 modules
  (compute-control, credit-ledger, inference-gateway, platform-core, cli) with no DB; the fixed
  YAML-parse step passes; `make build`/`make test` drive the loop. `.github/workflows/ci.yml` parses.

## [v0.2.14] — Scheduling stack: Kueue + Volcano + mock-GPU queues (F01 SCHED, unblocks F12 M3)

### Added
- **`make sched` / `SCHED=1` — the cluster-side scheduling layer** the F12 control plane targets,
  running on k3d with no real GPU. Installs **Kueue** (per-workload-class quota admission) + **Volcano**
  (gang scheduling), advertises a **mock GPU** extended resource (`exascale.io/gpu=8`) on the
  `exascale.io/gpu=mock` node, and applies the project Kueue config under `deploy/k8s/scheduling/`:
  `ResourceFlavor`s (`default-flavor`, `mock-gpu`) + a `ClusterQueue` per workload class
  (`cq-inference` 4 GPU / `cq-training-small` 2 / `cq-training-large` 2, shared cohort `exascale`) +
  a `LocalQueue` per class (`kueue.x-k8s.io/v1beta2`).
- `deploy/k8s/scheduling/` — `kueue-config.yaml`, `mock-gpu-resource.sh` (node-status PATCH; re-run
  after a cluster restart), `examples/{kueue-inference-job,volcano-gang-job}.yaml`, and a README.
- The data-plane installer is now tiered: `SCHED=1` (scheduling) and `OBS=1` (observability) compose
  independently; `FULL=1` = both. New `make sched` target; `make data-plane` passes the tier flags.

### Fixed
- **Install ordering:** Kueue registers a `Fail`-policy mutating webhook on *all* Deployments/Jobs
  cluster-wide, so applying Volcano before Kueue's webhook had endpoints made Volcano's own
  Deployments fail to create. The installer now waits for the Kueue controller + webhook endpoints
  before applying Volcano. The Kueue config is server-side-applied (avoids the v1beta1→v1beta2
  last-applied annotation clash; `cohort` was renamed to `cohortName` in v1beta2).

### Verified
- Live in k3d: install completes clean; all 3 `ClusterQueue`s report `Active=True`. A Kueue-admitted
  Job (`inference` queue, `exascale.io/gpu` request) → `Admitted=True`, both pods Running on the
  mock-GPU node. A Volcano `minAvailable:2` gang → `PodGroup` `Running` (`MINMEMBER=2 RUNNINGS=2`),
  both pods bound together by the `volcano` scheduler. App stack unaffected by the new operators.
- This is the F12 mock-GPU gang-scheduling acceptance at cluster level; it unblocks F12's M3 `k8s`
  scheduler backend (same `scheduler.Scheduler` interface, targets these queues). The 32+ GPU
  correctness run remains real-GPU-gated.

## [v0.2.13] — Milestone 2: `make test-e2e` — sub-5-min time-to-first-action gate (F01)

### Added
- **`make test-e2e` (`scripts/e2e.sh`) — the time-to-first-action acceptance test** (Decision Gate 2
  + immutable commitment #3). An API-level harness that drives the **real** services over HTTP and
  asserts the whole first-value loop completes within a 300s budget:
  **signup → top up credits → first inference (text debit) → first GPU job (gpu_\* debit)** — the
  inference *and* compute (F12) billing loops, end to end. It opens its own `kubectl` port-forwards
  for anything not already reachable (so it runs against a `make up` stack with no extra setup), uses
  a fresh tenant per run, and is idempotent (per-tenant idempotency keys). Prefers `jq`, falls back to
  `python3`. Exits non-zero on any failed step or a blown budget.

### Verified
- Green twice against the live k3d stack: full loop in **~6s** (budget 300s) — signup → JWT →
  top up (`text 100000`, `gpu_h100 1000`) → inference (`llama-3.1-8b`) 200 → text debit
  `100000 → 99999.9` → GPU gang submit→cancel → `gpu_h100 1000 → 999.999444`. The sub-5-minute
  contract holds with ~50× margin locally. (Next: wire it into the CI pre-staging gate; a
  browser-level Playwright variant over the Nuxt console can layer on later.)

## [v0.2.12] — Milestone 2: compute control plane v0 (F12) — closes M2

### Added
- **`compute-control` service (F12) — the GPU control plane.** New Go service (`:8086`) behind the
  `scheduler.Scheduler` interface: a GPU-type catalog + per-tenant quota for customers, and the
  internal scheduling surface (`POST/GET/DELETE /v1/compute/jobs`, `/types`, `/quota`, `/instances`)
  the inference gateway+runtime use to place gang-scheduled pods. Honors
  `docs/contracts/openapi/compute.yaml`. M2 ships the in-memory **mock-GPU** backend
  (`COMPUTE_SCHEDULER=mock`) — all-or-nothing gang capacity, idempotent submit on `Idempotency-Key`,
  supply-source attribution, tenant scoping; the real Kueue+Volcano+GPU-Operator backend implements
  the same interface in M3.
- **GPU usage → credit debit.** The service emits `compute.usage.v1` (exact fixed-point
  GPU-seconds → GPU-hours, no floats); **credit-ledger** now drains **both** `inference.usage.v1`
  and `compute.usage.v1` through one generalized consumer → idempotent `gpu_*` debit (idempotent on
  `usage_id`, hash-chained).
- **Auth model:** tenant JWT for customer reads (local HS256 verify, alg-pinned); service token for
  internal scheduling; `is_paper` is taken from the principal, never the request body.
- **Deploy:** distroless non-root Dockerfile, `deploy/k8s/compute-control/base` (kustomize), Tilt
  wiring, and the k3d local registry + real `seed.sh` (tenants + paper credits incl. `gpu_*`).

### Verified
- Unit tests green (scheduler: gang capacity, idempotency, tenant scoping, cancel→meter, validation,
  quota; API: public catalog, JWT-gated quota, service-token submit→get→cancel lifecycle, 402 on
  capacity). `go build`/`go vet` clean on both services.
- **Live in k3d:** `GET /v1/compute/types` (8×H100) → submit a 2-pod×2-GPU H100 gang → 202 running,
  `placement dc-owned-1/mock`, availability 8→4, quota `used 4 / remaining 4`, idempotent re-submit
  returns the same job → cancel frees capacity (→8) and emits `compute.usage.v1` → credit-ledger
  debits **gpu_h100 1000 → 999.976667** (hash-chained `consumption` txn keyed on `usage_id`,
  `is_paper=true`).
- `/security-review` clean (alg-pinned JWT, constant-time service-token compare, tenant-scoped
  404-not-403, negative-units guard, idempotent debit). Real Kueue+Volcano gang scheduling at GPU
  scale is GPU-node-gated → M3.

## [v0.2.11] — Milestone 2: live-only frontend — mock mode removed (F20)

### Changed
- **Removed frontend "mock mode" entirely — the web app is always live.** Every BFF route
  (`server/api/**`, 19 routes) now just proxies to the real platform service — no `isMock` branch;
  `isMock`/`mockIdentity` dropped from `server/utils/api.ts`; `EXASCALE_API_MODE`/`apiMode` removed
  from `nuxt.config`. `npm run dev` now needs the platform stack up.
- **Wired screens render only real data** with loading/empty states — the seeded showcase/mock that
  used to render on linked screens is gone: the inference playground starts with an **empty
  conversation** (no fake turns) and shows credits-only economics; the wallet **recent-movements**
  shows real ledger transactions (empty state when none, no canned rows); **settings → API keys** and
  **console** are live-only. Dead mock helpers/arrays removed (`mockReplyFor`, `mockCost`, mock
  `apiKeys`, `MOVEMENTS`, USD cost fallbacks).
- **Convention updated:** CLAUDE.md's "mock-data mode" is retired in favour of live-only; new features
  ship wired. Screens with **no backend yet** (the paused exchange, datacenter/compute supply) keep
  placeholder data until their service exists. Docs (RUN_LOCAL, TESTING) updated.

### Verified
- Typecheck clean on the changed files (only the pre-existing `console.vue` router-typing warning
  remains). Dev server boots and serves (`/`, `/login` → 200); BFF routes compile and proxy (a 500 on
  `/api/catalog` with the cluster down is the correct upstream-unreachable path, not a build error).
  Full live click-through pending a cluster restart.

## [v0.2.10] — Milestone 2: persona-aware onboarding (F20)

### Fixed
- **`/onboarding/kyc` was the Trader Light-KYC flow for every persona.** KYC (identity verification)
  is a *trading* requirement, so it now shows only for the **trader** persona. **AI Company**
  (`enterprise`) — verified, no trading — sees a short "you're ready, go to your console" panel;
  **Datacenter** (`partner`) is pointed at capacity registration (`/datacenter/register`). No personal
  KYC is forced on non-traders.
- **The signup account-type now actually drives the experience.** Picking Trader / AI Company /
  Enterprise on `/signup` sets the runtime persona on submit (Trader → `trader`; AI Company /
  Enterprise → `enterprise`) — so the sidebar surface *and* the onboarding path match the choice
  (previously the cards were cosmetic and everyone defaulted to one flow).

## [v0.2.9] — Milestone 2: local email (Mailpit) + persona-scoped nav (F02, F20)

### Added
- **Transactional email — works in local mode (F02).** platform-core now actually **sends** the
  email-verification link over SMTP (`internal/email`), wired to **Mailpit** in the local data plane
  (`axllent/mailpit` — SMTP `:1025`, web UI `:8025`). Sign up → the *"Verify your Exascale email"*
  message appears in Mailpit → click the link → `/onboarding/verify` consumes the token → you land on
  `/console`. The signup email-verify dead-end is gone. `SMTP_ADDR`/`EMAIL_FROM`/`APP_BASE_URL` config;
  a blank `SMTP_ADDR` makes sending a safe no-op (CI), which still surfaces the dev token.
- **Persona-scoped navigation (F20).** Each app persona now shows **only its screens**: the AI company
  (`enterprise`) gets Inference / Compute / Wallet / Onboarding / Audit / Billing — **not** the
  exchange screens (Trade/Markets/Index/Portfolio/History are trader-only; the stale "AI co also
  trades" tagging predated the GTM pivot). Datacenter (`partner`) = DC dashboard + Wallet; Wallet is
  shared. Default persona flipped to `enterprise` (the platform-first audience).

### Proven live (k3d)
- signup → **Mailpit inbox shows the verification email** → token extracted from the link →
  `POST /v1/auth/verify` → `{verified:true}`; replay of the used token → `400` (single-use). Mailpit
  reachable at `http://localhost:8025`. Unit tests: `VerifyURL` + disabled-sender no-op.

## [v0.2.8] — Milestone 2: settings · API keys — live (F20 × F02)

### Added
- **The settings → API Keys screen now manages real platform-core keys** in `local` mode
  (`Exascale Frontend`). Create mints a key through platform-core and reveals the **real one-time
  secret**; the list and revoke run against the live API; the count, created date, and status
  (active/revoked) reflect real data. `mock` mode keeps the showcase keys + client-side generation.
  Scope rendering tolerates arbitrary live scope strings (falls back gracefully for scopes not in the
  showcase taxonomy); an empty state and inline create/revoke errors were added. Design-token clean
  (mono prefixes, semantic status tags, sentence case).
- Wired via the existing `useKeys` composable + `/api/keys` BFF (no contract change).

### Verification
- Typecheck clean. **Live e2e passed** (through the running BFF): fresh tenant → `GET /api/keys` empty
  → `POST /api/keys` returns the **one-time secret** + prefix + scopes → `GET /api/keys` lists
  metadata only → `DELETE /api/keys/{id}` → 200 → list shows `active:0, total:1` (revoked). Confirms
  the screen's create-reveal-list-revoke + active/revoked mapping against live platform-core.
  _(Verification was briefly deferred when the local k3d cluster needed recreating; it has since been
  rebuilt and the e2e run.)_

## [v0.2.7] — Milestone 2: inference playground credit meter + debit-flow fix (F20 × F08, F08↔F05)

### Added
- **Real-time credit visibility in the inference playground** (`Exascale Frontend`). In `local` mode
  each run shows its **real credit cost** — `total_tokens / 1000 × catalog price`, in the model's
  credit type — on the turn, in the response sidebar, and as a pre-send estimate. A live **session
  meter** sums credits spent and shows the tenant's live `text` balance (pulled after each run).
  `mock` mode keeps the USD showcase. Cost is derived from the live catalog (`useCatalog`) + ledger
  balances (`useWallet`); the gateway debits the same `tokens × price` server-side.

### Fixed
- **Inference debits weren't flowing** (`credit-ledger` consumer). The `inference.usage.v1` consumer
  used a **push-based durable** subscription, which is exclusive — after a pod restart the server
  still held the old deliver subject "bound", so the new subscription failed with *"consumer is
  already bound to a subscription"* and the consumer **silently never started**. Usage events piled
  up unbilled, breaking the M2 inference-dollar loop (and commitment #4, visible settlement).
  Switched to a **durable pull consumer** (fetch loop) which rebinds cleanly across restarts; a
  legacy push consumer is auto-migrated (deleted → recreated as pull). Idempotency on `request_id`
  is unchanged, so the catch-up replay can't double-bill.

### Proven live (k3d)
- catalog `llama-3.1-8b` = 5.000000 / 1K tokens · text. Run via the BFF → 18 total tokens → the UI
  shows **0.090000 text credits**, and the ledger balance debits **50 → 49.91 within ~1s** through
  the fixed pull consumer. Consumer boots `…ledger debit (pull)` with no bind error.

## [v0.2.6] — Milestone 2: wallet "recent movements" — live (F20 × F05)

### Added
- **The wallet "Recent movements" table now renders real ledger transactions** in `local` mode
  (`useWallet.loadTransactions` → `/api/wallet/transactions`). Each ledger row maps to a movement:
  operation → coloured pill (`conversion`/`purchase`/`mint`/`consumption`/`burn`), credit type →
  asset name, signed amount (▲/▼ via `+`/`−`), and balance-after — mono + `tabular-nums`. A just-made
  conversion appears immediately (balances + movements both refresh on submit). `mock` mode keeps the
  canned showcase rows.

### Proven live (through the running frontend BFF)
- signup → purchase 300 ai_index → convert 120 → `GET /api/wallet/transactions` returns, newest-first:
  `Conversion text +98.672785` (bal 98.672785) · `Conversion ai_index −120` (bal 180) · `Purchase
  ai_index +300` (bal 300) — exactly the rows the table maps.

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

## [v0.1.5] — Milestone 1: CLI signup `--password` flag (F04 fix)

### Fixed
- **`exascale signup` now accepts `--password`** (and `EXASCALE_PASSWORD`), matching `login` — it
  previously only took `--email`/`--name` and always prompted, so the documented non-interactive
  signup (`signup --email … --password …`) failed with "flag provided but not defined: -password".
  The hidden prompt remains the default when neither is given. Proven live: `signup --email …
  --password … --name …` → account created + identity printed.

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
