# Exascale — Build Status (tree map)

**Living status.** ✅ done · 🟩 partly done / in progress · ⬜ not started · ⏸ paused (by design).
Updated as work lands. Pair with `SEQUENCING.md` (plan), `MANAGEMENT_PLAN.md` (tracker), `CHANGELOG.md` (releases).
Last updated: 2026-06-02.

> Legend: ✅ **green = done** · 🟩 in progress · ⬜ **white = not done** · ⏸ paused.
> Tags shipped: `v0.1.0 … v0.1.5` (M1) · `v0.2.0 … v0.2.12` (M2) · `v0.2.13 … v0.2.16` (F01 hardening:
> test-e2e, scheduling, CI build/test, golangci-lint v2). All on `main` (origin).

```
Exascale
│
├── ✅ Foundations — planning, standards, workflow, design system, ADRs, tooling
│
├── 🟩 Contracts — docs/contracts/
│   ├── ✅ credit-types.md · schemas/types.sql
│   ├── ✅ openapi/credit.yaml · openapi/platform-core.yaml (v1.5.0: auth+accounts+RBAC+keys+billing+verify)
│   ├── ✅ openapi/inference.yaml (v1.0.0, OpenAI-compatible)
│   ├── ✅ events  credit.tx.v1 · inference.usage.v1 · compute.usage.v1 · admin.action.v1
│   └── ⬜ openapi/{compute,supply,index}.yaml · events/partner.capacity.v1
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
│   └── ⬜ Real envs (staging/prod) · backup/restore drill
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
│   │                            F13 instance lifecycle, 32+ GPU correctness — GPU-node-gated.
│   └── ✅ F20 console wired      live-only BFF (proxies the platform; no mock mode, v0.2.11) + live
│                                /console (auth, catalog, inference, wallet, buy, keys, budget,
│                                purchases, audit). wallet convert (v0.2.5) + live movements (v0.2.6);
│                                inference real credit cost + session meter (v0.2.7); settings · API
│                                keys live (v0.2.8); persona-scoped nav + onboarding (v0.2.9–v0.2.10).
│
├── ⬜ M3+ — F10 catalog · F11 packing · F13 GPU lifecycle · F14 reserved · F15 clusters
│        · F16 supply abstraction · F17 DC onboarding · F18 payouts · F19 attestation
│
├── 🟩 F23 Console v1.5 screens (~15%) — the v1.5 screen catalog itemized screen-by-screen
│        (Tiers E–N) vs the backend each needs. Built: detail drawers, toasts, status, markets
│        index, tour, footer. Active: money/compute/enterprise/datacenter/marketing/help/polish;
│        exchange tiers ⏸ Phase 2. See features/F23-console-v15-screens.md.
│
├── 🟩 Compliance & trust  — ⬜ F21 SOC 2 (M4→M6) · 🟩 F22 licensing track (parallel)
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
- **Pulled forward:** F03 audit-log + RBAC (sequenced M4) built in M1; email-verify + budgets
  (M3-ish) shipped as gap-closers (v0.1.3).
- **M3–M6:** ⬜ not started.

Repo: trunk = `main`, pushed to `origin` (ex-main). Tags v0.1.0→v0.1.5 (M1), v0.2.0→v0.2.12 (M2).

---

## TO DO — next up (ordered, to converge on the sequencing)

1. **F01 remainder → Decision Gate 2** — ✅ `make test-e2e` (sub-5-min TTFA, ~6s live, v0.2.13);
   ✅ `SCHED=1` Kueue+Volcano+mock-GPU queues (F12's real-backend prereq, gang scheduling verified,
   v0.2.14). Remaining: observability (`OBS=1` Prometheus/Loki/Tempo + per-service `/metrics`; tasks
   #3/#13), SOPS secrets, real-env clusters, CI green (wire `make test-e2e` into the pre-staging gate).
   Prod-hardening before paying customers (M3).
2. **M3 begins — F13 GPU instance lifecycle** (<90s start, CLI `gpu create/list/stop`), the direct
   continuation of F12 on the `scheduler.Scheduler` seam; then F11 packing, F10 full catalog, F14
   reserved, F16 supply abstraction.
3. **M3 provisioning (your side)** — Stripe + domain/Cloudflare + registry + GPU node + HF token +
   email provider. See `PROVISIONING.md`.

_Done since last update:_ **F12 compute control plane v0 — M2 now complete** (v0.2.12): the
`compute-control` service (mock-GPU gang scheduling, idempotent submit, supply attribution) emits
`compute.usage.v1` → credit-ledger debits `gpu_*`, proven live in k3d (gang → cancel → debit
`1000→999.976667`). Earlier: wallet screen fully live — convert drawer (v0.2.5) + live movements
(v0.2.6); inference real credit cost + session meter, and the push→pull consumer fix so debits land
~1s after a run (v0.2.7); settings API keys live (v0.2.8); Mailpit email + persona-scoped nav
(v0.2.9); persona-aware onboarding (v0.2.10); live-only frontend, mock mode removed (v0.2.11).

### Decisions still open
- **ADR-0002** (AI↔sub-credit conversion direction) — provisional (bidirectional, 1% spread); counsel
  sign-off via F22. Needed to finalize F07.
