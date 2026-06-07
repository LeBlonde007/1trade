# Exascale — Build Status (tree map)

**Living status.** ✅ done · 🟩 partly done / in progress · ⬜ not started · ⏸ paused (by design).
Updated as work lands. Pair with `SEQUENCING.md` (plan), `MANAGEMENT_PLAN.md` (tracker), `CHANGELOG.md` (releases).
Last updated: 2026-06-04.

> Legend: ✅ **green = done** · 🟩 in progress · ⬜ **white = not done** · ⏸ paused.
> Tags shipped: `v0.1.0 … v0.1.5` (M1) · `v0.2.0 … v0.2.12` (M2) · `v0.2.13 … v0.2.16` (F01 hardening:
> test-e2e, scheduling, CI build/test, golangci-lint v2) · `v0.3.0` (M3 — F13 GPU instance lifecycle) ·
> `v0.3.1 … v0.3.12` (M3 — live AI-company product: console command-center, compact UI, **rich audit
> screen**, **honest teams/SSO**, Settings = single admin home, inference+wallet+**billing** live (no
> mock), onboarding Walmart-mock removed, **F22 KYC/AML enforcement**). All on `main` (origin).

```
Exascale
│
├── ✅ Foundations — planning, standards, workflow, design system, ADRs, tooling
│
├── 🟩 Contracts — docs/contracts/
│   ├── ✅ credit-types.md · schemas/types.sql
│   ├── ✅ openapi/credit.yaml · openapi/platform-core.yaml (v1.6.0: auth+accounts+RBAC+keys+billing+verify+**F22 KYC**)
│   ├── ✅ openapi/inference.yaml (v1.0.0, OpenAI-compatible)
│   ├── ✅ openapi/compute.yaml (v1.1.0 — catalog/quota/jobs + instance lifecycle, F12+F13)
│   ├── ✅ events  credit.tx.v1 · inference.usage.v1 · compute.usage.v1 · admin.action.v1
│   └── ⬜ openapi/{supply,index}.yaml · events/partner.capacity.v1
│       └── ⏸ Phase 2: trading.yaml · events/{trades.executed,orders.state,surveillance.alert}.v1
│
├── 🟩 F01 Foundation infra  (~68%) — owner infra-sre
│   ├── ✅ Repo scaffold · ✅ Core data plane (Postgres/TimescaleDB/Redis/NATS+JetStream) on k3d
│   ├── ✅ make test-e2e — timed TTFA loop (signup→top-up→infer→GPU debit) <300s, ~6s live (v0.2.13)
│   ├── ✅ SCHED=1 — Kueue + Volcano + mock-GPU queues (cq-inference/-training-{small,large}); live:
│   │      Kueue admits a job + Volcano gang-schedules minAvailable:2 on the mock node (v0.2.14)
│   ├── ⬜ OBS=1 stack (Prometheus/Loki/Tempo)   ⬜ SOPS secrets
│   ├── 🟩 CI: go job builds + race-tests all 5 modules (Go 1.25, scripts/go-all.sh) + golangci-lint
│   │      v2.12.2 hard gate, all modules lint-clean (scripts/lint.sh) + contracts YAML multi-doc fix
│   │      (v0.2.15–v0.2.16). go + contracts jobs green; hygiene red only on non-Go pre-commit hooks.
│   ├── 🟩 Envs: ✅ env=local (mock-GPU) · 🟢 env=sandbox (RTX 5090 32GB over Tailscale, real-GPU
│   │      validation; needs Dockerfile.vllm CUDA-12.8 bump) · ⬜ env=prod (H100/H200)
│   └── ✅ Backup/restore drill (make backup-restore-drill) — dump→restore the ledger DB; asserts row
│          counts + hash-chain digest survive. Live: 14 hash-chained txns, digest unchanged.
│
├── ✅ M1 — Foundation (shipped; F04 CLI v0 done)
│   ├── ✅ F05 credit-ledger     financial heart — append-only hash chain, fixed-point, atomic,
│   │                            per-tenant idempotency; consumes inference.usage.v1 → debit. DEPLOYED.
│   ├── ✅ F02 auth & SSO        signup/login→JWT/me/keys, bcrypt, case-insensitive, anti-enumeration,
│   │                            key introspection; email verification — real SMTP send via Mailpit
│   │                            in local mode (v0.2.9). OAuth/SAML/2FA = M4.
│   ├── ✅ F03 accounts/orgs/RBAC  audit log (admin.action.v1 + queryable trail), requireRole→403,
│   │                            org CRUD + role assignment, tenant-scoped. Sub-accounts = M4.
│   ├── 🟩 F04 exascale CLI v0   login/whoami/credits balance+convert/catalog/infer/keys/config —
│   │                            Go client over the live APIs (v0.1.4). M3+: gpu/cluster/train; dist M6.
│   └── ✅ F20 console shell     Nuxt app + design system (live-only; no mock mode, v0.2.11).
│
├── ✅ M2 — First inference dollar / sandbox (shipped, complete)
│   ├── ✅ F06 billing           Stripe checkout → webhook (sig-verified) → idempotent ledger mint;
│   │                            monthly budgets (v0.1.3). ACH/wire/JPY = M3.
│   ├── ✅ F07 credit conversion  AI-index↔text, atomic two-leg (burn+mint, chained), rate table,
│   │                            1% spread, idempotent (v0.2.4). M3: other modalities + GPU + UI.
│   ├── ✅ F08 inference gateway  OpenAI-compatible /v1/chat/completions+/models, API-key auth,
│   │                            pre-flight 402, emits inference.usage.v1 → ledger debits (pull
│   │                            consumer, v0.2.7 fix; debit lands ~1s, proven live). DEPLOYED.
│   ├── 🟩 F09 vLLM runtime      gateway↔runtime contract + VLLMBackend (mock→vLLM config flip) +
│   │                            production server.py/Dockerfile.vllm/GPU manifest + CPU stub.
│   │                            Proven live on the stub; real GPU serving needs F12 + a GPU node.
│   ├── ✅ F12 compute control plane  v0 live in k3d (v0.2.12). compute-control service behind the
│   │                            scheduler.Scheduler seam: catalog + quota + internal scheduling
│   │                            (/v1/compute/jobs|types|quota|instances), mock-GPU backend (gang
│   │                            capacity, idempotent submit, supply attribution). Emits
│   │                            compute.usage.v1 → ledger debits gpu_* (live: gang→cancel→debit
│   │                            1000→999.976667). M3: real Kueue+Volcano backend (same interface),
│   │                            ✅ F13 instance lifecycle (v0.3.0; live e2e: create→stop→debit
│   │                            gpu_h100 100→99.997778), 32+ GPU correctness — GPU-node-gated.
│   └── ✅ F20 console wired      live-only BFF (proxies the platform; no mock mode, v0.2.11) + live
│                                /console (auth, catalog, inference, wallet, buy, keys, budget,
│                                purchases, audit). wallet convert (v0.2.5) + live movements (v0.2.6);
│                                inference real credit cost + session meter (v0.2.7); settings · API
│                                keys live (v0.2.8); persona-scoped nav + onboarding (v0.2.9–v0.2.10).
│
├── 🟩 M3+ — ✅ F13 GPU lifecycle (v0.3.0) · ⬜ F10 catalog · F11 packing · F14 reserved · F15 clusters
│        · F16 supply abstraction · F17 DC onboarding · F18 payouts · F19 attestation
│
├── 🟩 F23 Console v1.5 screens (~45%) — live, honest, no-mock screens shipped v0.3.1–v0.3.12:
│        console command-center, /enterprise/audit (rich — filters + before→after diffs),
│        /enterprise/teams & /sso (honest, M4-staged), /enterprise/billing (live budget + balances +
│        purchases), Settings = single admin home, /wallet/buy (live Stripe + KYC gate), compact
│        K/M/B numbers. Removed the /enterprise/onboarding Walmart mock. Exchange tiers ⏸ Phase 2.
│        See features/F23-console-v15-screens.md.
│
├── ⬜ F24 CLI developer experience (Claude-Code-grade) — DX layer over the live F04 CLI: P1
│        streaming inference (SSE) + spinners + semantic colour + actionable errors + --json; P2
│        `exascale chat` interactive REPL; P3 completion/browser-login/profiles. M3→M6, incremental.
│        See features/F24-cli-dx.md.
│
├── 🟩 Compliance & trust  — ⬜ F21 SOC 2 (M4→M6) · 🟩 F22 licensing track: ✅ **KYC/AML enforcement**
│        (v0.3.12 — real-money checkout gated server-side, sandbox exempt, audited) · ⬜ counsel
│        sign-off / ADR-0002 · ⬜ prod manual-review wiring (decision endpoint exists, service-token)
│
└── ⏸ Phase 2 — Exchange (paused, license-gated, kept warm)
    └── ⏸ KW01 index · KW02 trading demo UI · KW03 matching · KW04 market maker · KW05 surveillance
```

---

## Where we are vs. the sequencing (SEQUENCING.md)

- **M1 (Foundation):** ✅ essentially complete. Shipped F01(core)+F02+F03+F04(CLI v0)+F05+F20-shell.
  **Gaps vs plan: F01 remainder (observability/SOPS/real-envs/CI-green) pending.**
- **M2 (First inference dollar, sandbox):** ✅ **complete.** The loop works end-to-end live — signup →
  buy credits (Stripe) → run inference (gateway→runtime) → idempotent ledger debit → console shows it;
  **F07 conversion** (AI-index↔text) live (v0.2.4); **F12 compute control plane v0** live (v0.2.12) —
  mock-GPU gang scheduling → `compute.usage.v1` → ledger `gpu_*` debit, proven in k3d. **Deferred to
  M3 (GPU-node-gated): F12's real Kueue+Volcano backend + F09 real GPU serving.**
- **M3 (First real customer revenue):** 🟩 **started.** ✅ **F13 GPU instance lifecycle** (v0.3.0) —
  customer-facing on-demand instances (create/list/get/stop/start/delete) over `compute.yaml` v1.1.0,
  sharing one GPU pool with the scheduler; per-interval metering → `compute.usage.v1` → `gpu_*` debit;
  `exascale gpu …` CLI + live web `/compute` list & provision. Mock-GPU backend (real K8s provisioner +
  <90s-P95 timing GPU-node-gated). **Since v0.3.0 (v0.3.1→v0.3.12):** the AI-company product hardened
  to live / no-mock end-to-end (console command-center, inference, wallet, **billing**); the enterprise
  admin surface went live + honest under **Settings** (audit · teams · SSO · billing); and **F22
  KYC/AML enforcement** shipped (v0.3.12) — real-money purchases now gated server-side. Next: F11
  packing, F10 full catalog, F14 reserved, F16 supply.
- **Pulled forward:** F03 audit-log + RBAC (sequenced M4) built in M1; email-verify + budgets
  (M3-ish) shipped as gap-closers (v0.1.3).
- **M4–M6:** ⬜ not started.

Repo: trunk = `main`, pushed to `origin` (ex-main). Tags v0.1.0→v0.1.5 (M1), v0.2.0→v0.2.16 (M2 +
F01 hardening), v0.3.0 (M3 — F13), v0.3.1→v0.3.12 (M3 — live AI-company product + F22 KYC enforcement).

---

## TO DO — next up (ordered, to converge on the sequencing)

1. **F01 remainder → Decision Gate 2** — ✅ `make test-e2e` (sub-5-min TTFA, ~6s live, v0.2.13);
   ✅ `SCHED=1` Kueue+Volcano+mock-GPU queues (F12's real-backend prereq, gang scheduling verified,
   v0.2.14). Remaining: observability (`OBS=1` Prometheus/Loki/Tempo + per-service `/metrics`; tasks
   #3/#13), SOPS secrets, real-env clusters, CI green (wire `make test-e2e` into the pre-staging gate).
   Prod-hardening before paying customers (M3).
2. **M3 continues — F11 multi-model-per-GPU packing, F10 full catalog, F14 reserved capacity, F16
   supply abstraction.** ✅ **F13 GPU instance lifecycle done** (v0.3.0). The real Kueue+Volcano
   provisioner behind F13's instance manager (same interface) + <90s-P95 timing remain GPU-node-gated.
3. **M3 provisioning (your side)** — Stripe + domain/Cloudflare + registry + GPU node + HF token +
   email provider. See `PROVISIONING.md`.

_Done since last update (v0.3.1 → v0.3.12):_ **live AI-company product + F22 KYC.** The product is now
live / no-mock end-to-end — console command-center, inference, wallet, and **billing** wired to the real
backend. The enterprise admin surface is consolidated under **Settings** as the single admin home, each
linking to a live page: **/enterprise/audit** (rich — search/filter + before→after diffs, admin-only),
**/enterprise/billing** (live budget get/set + real balances + purchase history), and honest
**/enterprise/teams** + **/sso** (real account/roles + M4 roadmap, no mock). Removed the
`/enterprise/onboarding` Walmart mock; added compact K/M/B number formatting. **F22 KYC/AML enforcement**
(v0.3.12): real-money checkout 403s `kyc_required` unless the tenant is verified (sandbox exempt) —
migration 0005 + domain status-machine + store + API (incl. service-token-only review endpoint) + tests,
deployed to k3d and proven live; `/security-review` clean. Also added `WAY_OF_WORKING.md` (portable build
playbook). _Earlier:_ **F13 GPU instance lifecycle — M3 begins** (v0.3.0): customer-facing
on-demand instances (create/list/get/stop/start/delete) over `compute.yaml` v1.1.0, drawing from one
shared `pool.Pool` with the scheduler (capacity never double-counted); a per-interval metering ticker
emits `compute.usage.v1` → credit-ledger `Compute` consumer debits the `gpu_*` tier (idempotent on
`usage_id`); versioned ML-Stack image catalog (stable/latest/pinned); `exascale gpu …` CLI + live web
`/compute` list & provision (BFF + `useCompute`). Mock-GPU backend; security-review clean. Also closed
a latent golangci-lint cold-cache gap (all 5 modules lint-clean cold). _Earlier:_ **F12 compute control
plane v0 — M2 complete** (v0.2.12): mock-GPU gang scheduling → `gpu_*` debit (gang→cancel→debit
`1000→999.976667`); wallet live (v0.2.5–v0.2.6); inference meter + pull-consumer fix (v0.2.7); settings
keys (v0.2.8); Mailpit + persona nav (v0.2.9–v0.2.10); live-only frontend (v0.2.11).

### Decisions still open
- **ADR-0002** (AI↔sub-credit conversion direction) — provisional (bidirectional, 1% spread); counsel
  sign-off via F22. Needed to finalize F07.
