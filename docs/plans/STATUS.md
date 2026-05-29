# Exascale — Build Status (tree map)

**Living status.** ✅ done · 🟩 partly done / in progress · ⬜ not started · ⏸ paused (by design).
Updated as work lands (the `/ex-*` commands keep this + `MANAGEMENT_PLAN.md` honest).
Last updated: 2026-05-29.

> Legend: ✅ **green = done** · 🟩 in progress · ⬜ **white = not done** · ⏸ paused.

```
Exascale
│
├── ✅ Foundations — planning, standards, workflow                     (the groundwork is complete)
│   ├── ✅ Plan docs           docs/plans/* — master, sequencing, 13 agent + 27 feature plans
│   ├── ✅ Engineering standards  per-function doc comments, security, tests, review-skill flow
│   ├── ✅ Design system        tokens, type scale, mock-data realism, aesthetic guardrails
│   ├── ✅ Decisions (ADRs)     0001 k3s/k3d · 0002 conversion (provisional) · 0003 design system
│   ├── ✅ Tooling              .golangci · pre-commit · editorconfig · gitignore · CODEOWNERS · install-toolchain.sh
│   └── ✅ Workflow             /ex-start /ex-update-feature /ex-fix /ex-review /ex-contract /ex-status + agent roster
│
├── 🟩 Contracts — docs/contracts/ (the only way services couple)
│   ├── ✅ credit-types.md       canonical credit enum + conversion policy
│   ├── ✅ schemas/types.sql     shared SQL types (credit_types table, conventions)
│   ├── ✅ openapi/credit.yaml          ledger API (balances/tx/purchase/convert/debit/mint/burn)
│   ├── ✅ openapi/platform-core.yaml   auth + accounts + JWT claims  ← NEW
│   ├── ✅ events  credit.tx.v1 · inference.usage.v1 · compute.usage.v1
│   └── ⬜ openapi/{inference,compute,supply,index}.yaml · events/partner.capacity.v1
│       └── ⏸ Phase 2: trading.yaml · events/{trades.executed,orders.state,surveillance.alert}.v1
│
├── 🟩 F01 Foundation infra  (~50%) — owner infra-sre, the bottom of the stack
│   ├── ✅ Repo scaffold        Makefile · Dockerfile.{go-service,nuxt,vllm} · k3d.yaml · Tiltfile · CI · README · .env.example
│   ├── ✅ Core data plane      Postgres 16 · TimescaleDB · Redis · NATS+JetStream — VERIFIED on k3d (1/1 Ready)
│   ├── ⬜ FULL=1 stack         Kueue + Volcano scheduling · Prometheus/Loki/Tempo observability
│   ├── ⬜ Secrets              SOPS-sealed manifests (Vault is M4)
│   ├── ⬜ Real envs            staging · prod-paper · prod-real (gated) clusters
│   └── ⬜ Hardening            CI green on GitHub · ledger backup/restore drill · make test-e2e skeleton
│
├── 🟩 Revenue path — Phase 1 critical path (inference dollar → first customer)
│   ├── 🟩 F05 credit-ledger   (~80%) — the financial heart — VERIFIED vs real Postgres
│   │   ├── ✅ Domain core       fixed-point Money (exact) · hash chain · Apply (balance + insufficient-credit + chain) — 6/6 tests
│   │   ├── ✅ Schema            migrations/0001_init.sql (append-only trigger · per-tenant idempotency · NULLS NOT DISTINCT · is_paper)
│   │   ├── ✅ Store             Postgres atomic balance+tx (one DB tx, per-balance lock) · idempotency · chain-verify — integration test green
│   │   ├── ✅ API               credit.yaml endpoints + auth (JWT/service) — httptest integration green; /convert→501 (F07)
│   │   └── ⬜ Deploy            credit.tx.v1 NATS publish · Dockerfile · k8s manifest · Tilt wiring
│   ├── ⬜ F02 auth             email/OAuth (contract ✅, impl not started) · SAML/SCIM/2FA (M4)
│   ├── ⬜ F03 accounts/orgs/RBAC + audit log
│   ├── ⬜ F04 exascale CLI
│   ├── ⬜ F06 billing          Stripe → ACH/wire → multi-currency
│   ├── ⬜ F07 credit conversion (AI↔sub↔gpu, 1% spread — ADR-0002 provisional)
│   ├── ⬜ F08 inference gateway (OpenAI-compatible, debit on usage)
│   ├── ⬜ F09 vLLM (first 3 models) · F10 catalog · F11 multi-model-per-GPU
│   ├── ⬜ F12 compute control plane · F13 GPU lifecycle · F14 reserved · F15 clusters
│   └── ⬜ F16 supply abstraction · F17 DC onboarding · F18 payouts · F19 GPU attestation
│
├── 🟩 Frontend
│   ├── 🟩 F20 platform console  Nuxt app exists on mock data (wallet/catalog/compute/billing screens)
│   └── ✅ Password gate         server-enforced (Nitro middleware + sealed cookie) — verified
│       └── on branch feat/site-password-gate · mirrored to github.com/saadallahdev/Exascale-Frontend
│
├── 🟩 Compliance & trust
│   ├── ⬜ F21 SOC 2 Type I      Vanta + control evidence (M4→M6)
│   └── 🟩 F22 licensing track   prepaid-framing sign-off + jurisdiction (parallel)
│
└── ⏸ Phase 2 — Exchange (paused, license-gated, kept warm)
    ├── ⏸ KW01 private reference index   ⏸ KW02 trading demo UI (exists, "exchange paused" ribbon)
    └── ⏸ KW03 matching engine (mock+spec) · KW04 market maker (spec) · KW05 surveillance (spec + basic abuse)
```

---

## Where we are vs. milestones

- **M1 (Foundation):** 🟩 in progress — F01 core data plane up; auth contract authored; ledger core built.
- **M2 (First inference dollar):** ⬜ blocked on F05 (finish) → F08/F09 (inference) + F06 (billing).
- **M3–M6:** ⬜ not started.

Repo: trunk = `main` (F01 merged in). Branches: `feat/site-password-gate` (frontend gate, mirrored).
Remotes: `origin`=ex-main · `frontend`=Exascale-Frontend. Nothing pushed from here yet (needs your SSH).

---

## TO DO — next up (ordered)

1. **Finish F05 credit-ledger** (`/ex-update-feature F05`)
   - `internal/store/` Postgres: atomic balance-update + tx-insert in one DB transaction; idempotency.
   - `cmd/credit-ledger/` HTTP handlers for the `credit.yaml` endpoints; emit `credit.tx.v1`.
   - Integration tests against the live k3d Postgres; Dockerfile + k8s manifest; chain-verify endpoint.
2. **F02 auth (impl)** — build against the now-authored `platform-core.yaml` (login/OAuth/JWT/keys).
   Unblocks every service's tenant context. (`/ex-start F02`.)
3. **F06 billing + F07 conversion** — once ledger + auth exist, prepaid purchase → balance is the
   first revenue mechanic. (`/ex-start F06`.)
4. **Inference path: contracts then build** — author `openapi/inference.yaml` (`/ex-contract inference`),
   then F08 gateway + F09 vLLM + F12 compute control plane → **first inference dollar (M2)**.
5. **F01 remainder, just-in-time** — `FULL=1` scheduling+observability (needed before F12), then SOPS,
   real-env clusters, CI-green (needs the GitHub push), backup/restore + e2e skeleton.
6. **Push to GitHub** — `git push -u origin main` (ex-main) + `git push frontend frontend-export:main --force`
   (run with your SSH key). Then CI/CODEOWNERS activate; replace the `@placeholder` handles in CODEOWNERS.

### Decisions still open
- **ADR-0002** (AI↔sub-credit conversion direction) — provisional A (bidirectional, 1% spread); needs counsel sign-off (F22).
- **CODEOWNERS** GitHub team handles (currently `@placeholder`).
