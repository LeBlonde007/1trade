# Exascale — Build Status (tree map)

**Living status.** ✅ done · 🟩 partly done / in progress · ⬜ not started · ⏸ paused (by design).
Updated as work lands. Pair with `SEQUENCING.md` (plan), `MANAGEMENT_PLAN.md` (tracker), `CHANGELOG.md` (releases).
Last updated: 2026-05-31.

> Legend: ✅ **green = done** · 🟩 in progress · ⬜ **white = not done** · ⏸ paused.
> Tags shipped: `v0.1.0 … v0.1.5` (M1) · `v0.2.0 … v0.2.11` (M2). All on `main` (origin).

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
├── 🟩 F01 Foundation infra  (~55%) — owner infra-sre
│   ├── ✅ Repo scaffold · ✅ Core data plane (Postgres/TimescaleDB/Redis/NATS+JetStream) on k3d
│   ├── ⬜ FULL=1 stack (Kueue + Volcano + Prometheus/Loki/Tempo)   ⬜ SOPS secrets
│   └── ⬜ Real envs (staging/prod) · ⬜ CI green on GitHub · backup/restore drill · make test-e2e
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
│   └── ✅ F20 console shell     Nuxt app + design system + mock-data mode.
│
├── ✅ M2 — First inference dollar / sandbox (shipped, except F07 + F12)
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
│   ├── ⬜ F12 compute control plane ← M2 GAP (Kueue+Volcano+GPU Operator; GPU-gated)
│   └── ✅ F20 console wired      BFF (EXASCALE_API_MODE mock|local) + live /console (auth, catalog,
│                                inference, wallet, buy, keys, budget alerts, purchases, audit).
│                                wallet drawer executes live F07 conversions (v0.2.5) +
│                                recent-movements renders live ledger txs (v0.2.6); inference
│                                playground shows real credit cost + session meter (v0.2.7);
│                                settings · API keys manage live platform-core keys (v0.2.8);
│                                persona-scoped nav — each persona shows only its screens (v0.2.9).
│
├── ⬜ M3+ — F10 catalog · F11 packing · F13 GPU lifecycle · F14 reserved · F15 clusters
│        · F16 supply abstraction · F17 DC onboarding · F18 payouts · F19 attestation
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
- **M2 (First inference dollar, sandbox):** ✅ the loop works end-to-end live — signup → buy credits
  (Stripe) → run inference (gateway→runtime) → idempotent ledger debit → console shows it; **F07
  conversion** (AI-index↔text) live (v0.2.4). **Gaps vs plan: F12 (compute control plane) not started
  (GPU-gated); F09 real GPU serving deferred to a GPU node.**
- **Pulled forward:** F03 audit-log + RBAC (sequenced M4) built in M1; email-verify + budgets
  (M3-ish) shipped as gap-closers (v0.1.3).
- **M3–M6:** ⬜ not started.

Repo: trunk = `main` (10 features merged), pushed to `origin` (ex-main). Tags v0.1.0→v0.1.4, v0.2.0→v0.2.4.

---

## TO DO — next up (ordered, to converge on the sequencing)

1. **F12 — compute control plane** (M2 gap). Needs a GPU node + GPU Operator to be meaningful; pairs
   with provisioning. Unblocks real F09 serving + F13 GPU lifecycle.
2. **F01 remainder** — observability (Prometheus /metrics + Grafana/Loki/Tempo; tasks #3/#13), SOPS
   secrets, real-env clusters, CI green. Prod-hardening before paying customers (M3).
3. **M3 provisioning (your side)** — Stripe + domain/Cloudflare + registry + GPU node + HF token +
   email provider. See `PROVISIONING.md`.

_Done since last update:_ wallet screen fully live — convert drawer executes F07 conversions
(v0.2.5) + recent-movements renders live ledger txs (v0.2.6); inference playground shows real
credit cost + a live session meter (v0.2.7); **fixed a silent bug where inference debits never
flowed** — the usage consumer was a push durable that failed to rebind after restarts; now a pull
consumer (v0.2.7), debit proven to land ~1s after a run. Settings API keys live (v0.2.8). Local email
via Mailpit + persona-scoped nav (v0.2.9).

### Decisions still open
- **ADR-0002** (AI↔sub-credit conversion direction) — provisional (bidirectional, 1% spread); counsel
  sign-off via F22. Needed to finalize F07.
