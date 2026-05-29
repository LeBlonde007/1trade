# Exascale — Build Plan (Master)

> This is the operational plan. It tells each agent **what to build, in what order, against
> which contracts, from project setup → local dev → deployment.** It is built against the
> **2026-05 GTM pivot** (`docs/re/exascale_gtm_focus_update.md`) — **platform-first, exchange
> paused** — and honors the system design in `docs/re/phase6_v2.md`.
>
> If anything below conflicts with `CLAUDE.md` immutable commitments, the immutable commitments
> win. If it conflicts with the GTM pivot, the GTM pivot wins (it explicitly supersedes the
> "trading layer first" sequencing in Phase 6 v2).

---

## 0. Read these in order

1. `CLAUDE.md` — global ground truth (every agent already loads it).
2. `docs/re/exascale_gtm_focus_update.md` — **the pivot. Read this first**: platform now, exchange paused.
3. `docs/re/phase6_v2.md` — product + system design source of truth.
4. `docs/plans/SEQUENCING.md` — the milestone-by-milestone order this plan delivers in.
5. `docs/plans/ENGINEERING_STANDARDS.md` — **how to write the code**: clean-code rules, the
   per-function comment mandate, security, per-language conventions, the review-skill workflow,
   and the full definition of done. Enforced by `.golangci.yml` / `.pre-commit-config.yaml` /
   `.editorconfig` / `.github/CODEOWNERS`.
6. Your agent file in `docs/plans/agents/<your-name>.md` — what you specifically own.
7. The feature files in `docs/plans/features/` you're listed as owner of.

---

## 1. What we are building (one paragraph)

A commodity-market-for-AI-compute, delivered in two phases:

- **Phase 1 — Platform (now → first dollar):** AI startups buy compute and inference from Exascale,
  paid for in prepaid, redeemable AI/GPU credits. Datacenters (owned + partner) supply capacity
  through a single scheduling fabric. Credits are explicitly **redeemable, not tradeable** — no
  securities/commodities license needed.
- **Phase 2 — Exchange (license-gated, designed-but-dormant):** order book, market maker, public
  tradeable AI index, trade surveillance, secondary-market settlement. Specced, mock-built, kept
  warm. Switched on once the license clears.

The platform produces the underlying (real consumption, real supply, real prices) that makes the
eventual exchange credible. Same vision; de-risked sequence.

---

## 2. Critical path under the GTM pivot

Build order — each step unblocks the next dollar:

```
infra-sre (foundation: K8s, data plane, CI/CD, observability)
     │
     ▼
platform-core (auth, orgs/sub-accounts, RBAC, CLI v0)
     │
     ▼
credit-ledger (prepaid + consumption debit, hash chain) ─┐
                                                          │
inference-ml (gateway, vLLM, first 3 models)  ◄───────────┤   debits go through credit-ledger
                                                          │
compute-platform (control plane, GPU lifecycle, supply pool) ◄┤
                                                              │
platform-core (billing: Stripe + ACH/wire, multi-currency)  ◄─┤   purchases land in credit-ledger
                                                              │
settlement-trust (DC onboarding, attestation, payout) ◄───────┘   payouts derive from consumption
     │
     ▼
trading-frontend (repurposed as platform console: catalog, wallet, compute, billing)
     │
     ▼
security-compliance (SOC 2 Type I + licensing track for Phase 2)
```

**Keep-warm track (runs in parallel, lower priority):**

```
index-service (private reference index from real platform transactions; methodology published)
matching-engine (spec + mock backend; switch-on later)
market-maker (designed only; activates with exchange)
surveillance (basic abuse monitoring only; full trade surveillance waits)
trading-frontend (the trading dashboard kept as investor-facing demo, clearly labelled)
```

---

## 3. The agent roster under the pivot

| Agent | Phase 1 status | What ships now |
|---|---|---|
| [`infra-sre`](agents/infra-sre.md) | ✅ active — foundation | Terraform, K8s base, CI/CD, observability, data plane |
| [`platform-core`](agents/platform-core.md) | ✅ active — revenue rails | Auth, orgs, RBAC, billing, gateway, CLI |
| [`credit-ledger`](agents/credit-ledger.md) | ✅ active (simplified) | Prepaid balance + consumption debit + hash chain. Trading settlement deferred. |
| [`inference-ml`](agents/inference-ml.md) | ✅ active — core | Gateway, vLLM, catalog, multi-model packing, usage debit |
| [`compute-platform`](agents/compute-platform.md) | ✅ active — core | Control plane, GPU lifecycle, supply-source abstraction |
| [`settlement-trust`](agents/settlement-trust.md) | ◐ partial | DC onboarding + attestation + payout. Tradeable-contract + trade-escrow deferred. |
| [`trading-frontend`](agents/trading-frontend.md) | ◐ repurposed | Becomes the **Platform Console** (catalog, wallet, compute, billing). Trading dashboard kept warm. |
| [`security-compliance`](agents/security-compliance.md) | ✅ active + owns licensing track | SOC 2 Type I now; license-clearance path for Phase 2 |
| [`tech-lead`](agents/tech-lead.md) | ✅ active — coordinator | Owns `docs/contracts/`, decomposes work, reviews drift |
| [`index-service`](agents/index-service.md) | ◐ keep-warm | Private internal reference index (not public, not tradeable) |
| [`matching-engine`](agents/matching-engine.md) | ⏸ paused / mock | Mock data adapter for the demo UI; engine specced |
| [`market-maker`](agents/market-maker.md) | ⏸ paused | Designed; turn on with exchange |
| [`surveillance`](agents/surveillance.md) | ⏸ paused (trade); minimal abuse only | Auth abuse / rate-limit floor; full trade surveillance waits |

---

## 4. Feature index

See [`features/`](features/) for one doc per feature.

### Phase 1 — Platform (active)

| ID | Feature | Owner | Status |
|---|---|---|---|
| [F01](features/F01-foundation-infra.md) | Foundation infra (K8s, data plane, CI/CD, obs) | `infra-sre` | active |
| [F02](features/F02-auth-and-sso.md) | Auth (email/OAuth/SAML), SCIM, 2FA | `platform-core` | active |
| [F03](features/F03-accounts-orgs-rbac.md) | Accounts, orgs, sub-accounts, RBAC | `platform-core` | active |
| [F04](features/F04-cli.md) | `exascale` CLI (browser-OAuth login, all platform commands) | `platform-core` | active |
| [F05](features/F05-prepaid-credit-ledger.md) | Prepaid credit ledger (balances, append-only tx, hash chain) | `credit-ledger` | active |
| [F06](features/F06-credit-purchase-billing.md) | Credit purchase: Stripe (cards) + ACH/wire + POs + multi-currency | `platform-core` + `credit-ledger` | active |
| [F07](features/F07-credit-conversion.md) | AI ↔ sub-credit conversion (text/speech/image/video/embeddings/GPU tiers) | `credit-ledger` | active |
| [F08](features/F08-inference-gateway.md) | Inference gateway (OpenAI-compatible, auth, quota, debit) | `inference-ml` | active |
| [F09](features/F09-vllm-deployment.md) | vLLM deployment for first 3 models (Llama 70B, Llama 8B, Whisper) | `inference-ml` | active |
| [F10](features/F10-model-catalog.md) | Curated SoTA catalog (full top-3–5 per category) | `inference-ml` | active |
| [F11](features/F11-multi-model-per-gpu.md) | Multi-model-per-GPU packing (static co-loc / hot-swap / MIG) | `inference-ml` | active |
| [F12](features/F12-compute-control-plane.md) | Compute control plane (Kueue + Volcano + GPU Operator) | `compute-platform` | active |
| [F13](features/F13-gpu-instance-lifecycle.md) | GPU instance lifecycle (on-demand H100/H200, <90s to running) | `compute-platform` | active |
| [F14](features/F14-reserved-capacity.md) | Reserved capacity (1mo/6mo/1yr discounts, prepaid GPU credits) | `compute-platform` + `credit-ledger` | active |
| [F15](features/F15-clusters-infiniband.md) | Multi-node clusters (InfiniBand, gang-scheduled) | `compute-platform` | active |
| [F16](features/F16-supply-source-abstraction.md) | Supply-source abstraction (owned + partner DCs as one pool) | `compute-platform` + `settlement-trust` | active |
| [F17](features/F17-dc-partner-onboarding.md) | DC partner onboarding (manual v1 → productized v1.5) | `settlement-trust` | active |
| [F18](features/F18-partner-payouts.md) | Partner payouts (escrow + streamed; backing/proof-of-reserves) | `settlement-trust` | active |
| [F19](features/F19-gpu-attestation.md) | GPU attestation (KYB + NVIDIA + challenge-response + DCGM + bond) | `settlement-trust` | active |
| [F20](features/F20-platform-console.md) | Platform console UI (catalog, wallet, compute, billing, account) | `trading-frontend` | active |
| [F21](features/F21-soc2-type1.md) | SOC 2 Type I (Vanta, controls, evidence) | `security-compliance` + `infra-sre` | active |
| [F22](features/F22-licensing-track.md) | Licensing track (jurisdiction, counsel, prepaid framing sign-off) | `security-compliance` | parallel |

### Phase 2 — Exchange (keep-warm)

| ID | Feature | Owner | Status |
|---|---|---|---|
| [KW01](features/KW01-private-reference-index.md) | Private internal reference index (from real platform tx) | `index-service` | keep-warm |
| [KW02](features/KW02-trading-demo-ui.md) | Trading dashboard kept warm as investor demo | `trading-frontend` | keep-warm |
| [KW03](features/KW03-matching-engine-mock-and-spec.md) | Matching engine mock data adapter + real-engine spec | `matching-engine` | keep-warm |
| [KW04](features/KW04-market-maker-spec.md) | Market maker design + risk-control spec | `market-maker` | spec only |
| [KW05](features/KW05-surveillance-v0.md) | Surveillance design (6 patterns) + basic abuse monitoring only | `surveillance` | spec + abuse-only |

---

## 5. Sync model (how isolated agents stay coherent)

Agents cannot see each other's work. Coordination happens through **written contracts** in
`docs/contracts/`, owned by `tech-lead`. See [`CONTRACTS.md`](CONTRACTS.md) for the rules. The
short version:

- Need an interface from another service? Look in `docs/contracts/` first; if it's not there,
  STOP and ask `tech-lead` to author it before building against assumptions.
- Never edit a shared contract on your own. Propose changes via `tech-lead`.
- Every feature doc in `features/` lists `Contracts consumed:` and `Contracts produced:`, so
  every cross-service touchpoint is explicit.

---

## 6. Setup → local dev → deployment

Four documents:

- [`REPO_LAYOUT.md`](REPO_LAYOUT.md) — the monorepo structure agents build into.
- [`LOCAL_DEV.md`](LOCAL_DEV.md) — clone-to-running in <15 min on a dev laptop (`make up`).
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — Dockerfiles, K8s manifests, CI/CD, environments.
- [`ENGINEERING_STANDARDS.md`](ENGINEERING_STANDARDS.md) — how the code is written and kept clean,
  secure, and consistent across all 13 agents (the per-function comment mandate, security, tests,
  the review-skill workflow, the definition of done). Enforced by the root tooling configs
  (`.golangci.yml`, `.pre-commit-config.yaml`, `.editorconfig`, `.github/CODEOWNERS`).
- [`DECISIONS.md`](DECISIONS.md) — the ADR log: significant architecture decisions and *why*
  (e.g. ADR-0001: Kubernetes via k3s/k3d, not Dokploy). Check it before re-opening a settled call.
- [`DESIGN_SYSTEM.md`](DESIGN_SYSTEM.md) — the canonical frontend visual spec (tokens, type scale,
  number formatting, mock-data realism, aesthetic guardrails). **Every frontend change conforms.**

---

## 7. Decision gates (under the pivot)

Re-cast from `phase6_v2.md` §15, adjusted for platform-first ordering.

### Gate 1 — End of Milestone 1: Platform foundation
Pass: cluster up; CI/CD green; auth + first CLI command works end-to-end (`exascale login`,
`exascale credits balance` shows $0); `trading-frontend` running platform-console shell with mock data.

### Gate 2 — End of Milestone 2: First inference dollar (sandbox)
Pass: vLLM serves Llama-70B and Llama-8B through the gateway; usage events debit the ledger
correctly; CLI `exascale infer` works; prepaid credit purchase via Stripe books to ledger.

### Gate 3 — End of Milestone 3: First real customer revenue
Pass: ≥1 AI-startup customer using inference for real workloads on real money; on-demand GPU
self-serve working; reserved-capacity purchase flow live; SOC 2 audit firm engaged.

### Gate 4 — End of Milestone 4: Supply + enterprise
Pass: ≥1 partner DC capacity actively serving real customer load through the supply abstraction;
SAML SSO live for enterprise accounts; sub-account RBAC working; partner payouts wired (first cycle).

### Gate 5 — End of Milestone 5: Catalog + scale
Pass: full top-3–5-per-category catalog serving; multi-model-per-GPU packing in production;
attestation + proof-of-reserves published; second partner DC.

### Gate 6 — End of Milestone 6: GA + license decision
Pass: $300K MRR (revised — consumption + prepaid purchases); SOC 2 Type I report; licensing
jurisdiction decision; Phase 2 switch-on date scheduled (or explicit defer with reason).

---

## 8. What this plan deliberately is NOT

- **A re-justification of the GTM pivot.** That argument is closed in
  `docs/re/exascale_gtm_focus_update.md`. This plan executes it.
- **A frozen design.** Contracts in `docs/contracts/` are versioned and change through `tech-lead`.
  Plan docs update when contracts do — every PR that changes a contract touches the relevant
  feature doc.
- **A spec for the exchange to ship now.** The exchange is **paused** (designed, mock-built,
  switched on later). Anyone shipping exchange feature code today is off-plan.

---

## 9. Glossary

- **AI credit** — prepaid, redeemable unit; convertible into sub-credits; not tradeable until Phase 2.
- **Sub-credit** — modality-scoped credit (text / speech / image / video / embeddings).
- **GPU credit** — class-standardized GPU-hour (e.g., H100-80GB-hour); represents reserved capacity.
- **Backing ratio** — outstanding GPU credits / attested reserved capacity; ≥100% in v1.
- **Owned DC** — Exascale's own datacenter (v1 anchor, trivially attestable).
- **Partner DC** — external operator's capacity registered through the supply-source abstraction.
- **Keep-warm** — designed, sometimes mock-built, not shipped to customers; switch-on later.
- **`is_paper`** — sacred flag on every order/trade/balance — even though the exchange is paused,
  this flag stays in the ledger schema so the Phase 2 switch-on is a config flip, not a migration.
