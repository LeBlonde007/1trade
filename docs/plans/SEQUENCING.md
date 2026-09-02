# Sequencing — Milestone-by-milestone delivery

This is the build order: which feature ships in which milestone, which agent owns it, and what
unblocks what. Built against the GTM pivot (platform-first), six milestones to GA and a license
decision.

Read in tandem with `README.md` §2 (the critical path diagram) and `features/` (one doc per
feature).

---

## Milestone 1 — Foundation

**Goal**: a developer can `make up` and get auth, CLI login, an empty wallet, and the platform
console shell on mock data. Nothing customer-facing yet.

| Feature | Owner | Notes |
|---|---|---|
| F01 — Foundation infra | `infra-sre` | K8s via k3s (`k3d` local, k3s staging/prod-paper), Postgres, TimescaleDB, Redis, NATS, Traefik ingress, K8s Secrets+SOPS, Grafana. Repo skeleton + Makefile + CI green. (Kong/Vault/Terraform are later-milestone hardening — ADR-0001.) |
| F02 — Auth & SSO (email/password + OAuth providers) | `platform-core` | SAML SSO scheduled for Milestone 4 (enterprise). |
| F03 — Accounts, orgs, sub-accounts, RBAC | `platform-core` | Sub-accounts and SCIM in Milestone 4. |
| F04 — CLI v0 | `platform-core` | `login`, `whoami`, `credits balance` (zero), `--help`. |
| F05 — Prepaid credit ledger (skeleton) | `credit-ledger` | Schema + hash chain + balance/tx tables. No conversion yet. |
| F20 — Platform Console (shell) | `trading-frontend` | Nav, layouts, design tokens consolidated, "wallet empty" state, all on mock API mode. |
| KW02 — Trading dashboard kept warm | `trading-frontend` | Existing trading dashboard kept as `/trade` route, on mock data, labelled "demo". |

**Contracts authored this milestone** (`tech-lead`):
- `openapi/platform-core.yaml` v1.0.0
- `openapi/credit.yaml` v1.0.0 (purchase + balance + convert; conversion implementation lands later)
- `schemas/types.sql`
- `credit-types.md`
- `events/credit.tx.v1.yaml`

**Decision Gate 1** (end of milestone): cluster up, CI green, CLI login works end-to-end, console
shell renders against mock data. If not, scope reduces — `inference-ml` and `compute-platform`
do **not** start until this is met.

---

## Milestone 2 — First inference dollar (sandbox)

**Goal**: a developer signs up, buys $100 of credits via Stripe (paper), runs `1trade infer chat`
through a real vLLM (Llama-8B), wallet shows the debit. End-to-end on a sandbox tier.

| Feature | Owner | Notes |
|---|---|---|
| F06 — Credit purchase (Stripe) | `platform-core` + `credit-ledger` | ACH/wire later. |
| F07 — Credit conversion (AI ↔ text) | `credit-ledger` | Only text in M2; other modalities as their inference paths come online. |
| F08 — Inference gateway | `inference-ml` | OpenAI-compatible. Auth, quota, balance check, debit. |
| F09 — vLLM for first 3 models (Llama-70B, Llama-8B, Whisper) | `inference-ml` | Single-tenant per GPU first; packing comes M3. |
| F12 — Compute control plane v0 | `compute-platform` | Enough K8s + Kueue + Volcano + GPU Operator to schedule inference pods. Customer-facing GPU lifecycle (F13) lands M3. |
| F20 — Platform Console (wired wallet + catalog) | `trading-frontend` | Wallet shows real balance from `credit-ledger`. Catalog renders real models. |

**Contracts authored this milestone**:
- `openapi/inference.yaml` v1.0.0
- `openapi/compute.yaml` v1.0.0 (provisional — final shape settles M3)
- `events/inference.usage.v1.yaml`
- `events/compute.usage.v1.yaml`

**Decision Gate 2**: end-to-end signup → buy → infer → debit. Sub-5-minute time-to-first-action
contract met (timed in CI's `make test-e2e`).

---

## Milestone 3 — First real customer revenue

**Goal**: ≥1 real AI-startup customer (warm intro list) paying for inference on real money, on
the platform — through Stripe (cards) and ACH/wire for larger purchases.

| Feature | Owner | Notes |
|---|---|---|
| F06 — Credit purchase: ACH/wire + multi-currency (USD + JPY) | `platform-core` | Stripe Connect for cards. |
| F10 — Catalog (full top-3–5 per category) | `inference-ml` | Embeddings, image, code categories ship. |
| F11 — Multi-model-per-GPU packing | `inference-ml` | Static co-loc for top models; hot-swap LRU for the tail. |
| F13 — GPU instance lifecycle (on-demand) | `compute-platform` | <90s instance start; CLI `gpu create/list/stop`. |
| F14 — Reserved capacity (1/6/12 mo) | `compute-platform` + `credit-ledger` | Discount tiers; represented as GPU credits. |
| F16 — Supply-source abstraction (1Trade-owned DC anchor) | `compute-platform` + `settlement-trust` | Owned DC as the only supply source this milestone; partner DCs land M4. |

**Contracts authored this milestone**:
- `openapi/inference.yaml` v1.1.0 (modality additions)
- `openapi/compute.yaml` v1.1.0 (final)

**Decision Gate 3**: first paying real-money inference customer; on-demand GPU rental working.

---

## Milestone 4 — Supply + enterprise

**Goal**: first partner DC delivers real customer load through the supply abstraction. Enterprise
customers can sign up with SAML SSO and run multi-team sub-accounts.

| Feature | Owner | Notes |
|---|---|---|
| F02 — Auth: SAML SSO + SCIM + 2FA | `platform-core` | Enterprise IdP-friendly (Okta, Entra, Google Workspace). |
| F03 — Sub-accounts + RBAC + audit log | `platform-core` | Per-team budgets, org-wide consumption views. |
| F17 — DC partner onboarding (manual flow) | `settlement-trust` | First partner DC contracted, agent deployed, capacity verified. |
| F18 — Partner payouts (first monthly cycle) | `settlement-trust` + `credit-ledger` | Escrow + streamed payout; backing ratio published. |
| F19 — GPU attestation (KYB + NVIDIA + challenge-response + DCGM + bond) | `settlement-trust` | Owned DC attests trivially; partner runs the full stack. |
| F21 — SOC 2 Type I (audit kickoff) | `security-compliance` + `infra-sre` | Vanta engaged; control evidence pipeline starts feeding. |

**Contracts authored this milestone**:
- `openapi/supply.yaml` v1.0.0
- `events/partner.capacity.v1.yaml`

**Decision Gate 4**: ≥1 partner DC actively serving load; SAML enterprise login works; first
partner payout wired; SOC 2 audit kicked off.

---

## Milestone 5 — Catalog scale + clusters

**Goal**: full catalog at scale, multi-node clusters available for training-heavy customers,
second partner DC online.

| Feature | Owner | Notes |
|---|---|---|
| F10 — Catalog (full) — at scale | `inference-ml` | All categories serving with documented latency. |
| F15 — Multi-node clusters (InfiniBand, gang-scheduled) | `compute-platform` | Sales-engaged for 32+ GPU clusters. |
| F17 — DC partner onboarding (second partner) | `settlement-trust` | Second partner adds redundancy. |
| F18 — Proof-of-reserves publication | `settlement-trust` | Public PoR endpoint; backing ratio dashboard. |
| KW01 — Private reference index (live, internal) | `index-service` | Methodology published; index computed daily from real platform tx. Not customer-facing/tradeable. |

**Contracts**:
- Index methodology v1.0 documented at `docs/index-methodology.md` (in `index-service`).

**Decision Gate 5**: second partner DC operational; private index computing daily; performance
targets met.

---

## Milestone 6 — GA + license decision

**Goal**: ship GA; ship SOC 2 Type I report; decide Phase 2 timing.

| Feature | Owner | Notes |
|---|---|---|
| F21 — SOC 2 Type I (report) | `security-compliance` + `infra-sre` | Vanta audit completes. |
| F22 — Licensing track (jurisdiction decision + counsel sign-off on prepaid framing) | `security-compliance` | Inputs: prepaid stays redeemable-only; jurisdiction (US/Japan/Singapore) decided. |
| F20 — Platform Console polish + docs site | `trading-frontend` | Documentation site, SDK pages, integration guides. |
| Hardening across all services | every agent | Performance, reliability, runbooks, dashboards. |
| KW03 — Matching engine spec finalized | `matching-engine` | Ready for Phase-2 switch-on when license arrives. |
| KW04 — Market maker spec finalized | `market-maker` | Same. |

**Decision Gate 6**: $300K MRR; SOC 2 Type I report; license decision (proceed with Phase 2
switch-on or document why deferred and for how long).

---

## Phase 2 — Exchange switch-on (post-license)

Once Decision Gate 6 yields a license decision and platform liquidity precursors are present
(real supply, real demand, real price data):

```
1. matching-engine: turn the mock data adapter off, the real engine on.
2. credit-ledger: enable settlement on fill (already designed; flag flip).
3. market-maker: deploy from spec; quote both sides under risk limits.
4. surveillance: deploy from spec; full 6-pattern detection on.
5. index-service: flip from private internal to public + tradeable.
6. trading-frontend: trading dashboard goes from "demo" to live.
7. settlement-trust: real-money escrow path on (already designed; flag flip).
```

The switch-on is **flags + deploys, not new architecture**. That is the whole point of keeping
the contracts and the `is_paper` boundary intact today.

---

## Cross-milestone sync points

These are the moments where multiple agents must coordinate. Each is a `tech-lead`-facilitated
sync.

| Sync point | When | Who must align |
|---|---|---|
| `credit-types.md` v1 | M1 | tech-lead authors; all agents review |
| Auth contract finalized | M1 | platform-core (producer); all services (consumer) |
| Inference debit contract | M2 | inference-ml + credit-ledger + tech-lead |
| Multi-currency adds JPY to credit purchase | M3 | platform-core + credit-ledger + trading-frontend |
| Supply-source registration | M3-M4 | compute-platform + settlement-trust + tech-lead |
| Partner payout calculation | M4 | settlement-trust + credit-ledger + infra-sre (data pipeline) |
| SAML + sub-account adds tenant hierarchy | M4 | platform-core + every service that authorizes by tenant |
| SOC 2 evidence pipeline | M4-M6 | security-compliance + infra-sre + platform-core |
| Index methodology audit (private → public-ready) | M5-M6 | index-service + security-compliance + tech-lead |

---

## What's NOT in this sequencing

- Customer-to-customer trade matching (Phase 2 only).
- Real-money trading (license-gated).
- Forward contracts (Phase 2; spec lands during keep-warm).
- Mobile app, office GPU supply, fine-tuning service, video gen at scale, B200/B300 (out of v1).
- HIPAA, FedRAMP, SOC 2 Type II (out of v1).

These show up in feature docs only if/when they're in scope.
