# Exascale — Build Plan & Progress Tracker

**Audience:** management / stakeholders. **Owner:** Ahmed (eng lead). **Founder:** Tai.
**Last updated:** 2026-05-29. **Status of this document:** living — update the status columns at
each weekly review.

> **One-line summary:** We are building the **platform** (AI inference + GPU compute + prepaid
> credits) for AI startups **first**, because it earns revenue now without a license. The
> **exchange** (tradeable credits) is **paused** — fully designed and kept demo-ready — and
> switches on once a license is secured. Same long-term vision; lower-risk order.

---

## How to use this tracker

- The **Roadmap** (§4) and **Decision Gates** (§5) are the high-level view for status meetings.
- The **Feature Tracker** (§6–§7) is the detailed line-by-line view. Update the **Status**,
  **%**, and **Notes** columns weekly.
- **Status legend:** ⬜ Not started · 🔵 In progress · ✅ Done · ⚠️ At risk · 🔴 Blocked · ⏸ Paused (by design).
- **Milestones (M1–M6) are dependency-ordered stages, not calendar months.** They ship in
  sequence, each unblocking the next — delivered as fast as the work allows. The timeline is
  intentionally not date-boxed; track by **milestone completion**, not by date.

| Milestone | Headline |
|---|---|
| M1 | Foundation |
| M2 | First inference dollar (sandbox) |
| M3 | First real customer revenue |
| M4 | Supply + enterprise |
| M5 | Catalog scale + clusters |
| M6 | GA + license decision |

---

## 1. Executive summary

Exascale is a **commodity market for AI compute**. Three-sided: datacenters supply GPUs, AI
companies consume them, and (eventually) traders provide liquidity on tradeable credits.

In **May 2026** we made a deliberate sequencing decision: **build the platform first, pause the
exchange.** The exchange was always the hardest, riskiest layer — it needs a license, it needs
liquidity, and a credit is only credible once it's backed by real consumed compute. So we build
the platform that produces that real consumption, real supply, and real price data — and the
eventual exchange inherits all three.

**The crucial change:** revenue moves from *"blocked behind a license"* to *"reachable now."*
The platform earns the moment an AI startup runs inference or rents a GPU.

---

## 2. The bottom line for management

| Question | Answer |
|---|---|
| What earns money first? | **Inference consumption** (pay-per-use), then GPU rental, then prepaid credit purchases (cash upfront). |
| Do we need a license to launch? | **No** — credits are prepaid, redeemable service units (not tradeable). Counsel sign-off tracked as F22. |
| When is first revenue? | Target **M2** (sandbox), **M3** (first real paying AI-startup customer). |
| When is GA? | At **milestone M6**, alongside a SOC 2 Type I report and the license decision. |
| What's the revenue target by GA? | **$300K MRR** (consumption + prepaid purchases). |
| What happens to the exchange? | Paused, not cancelled. Designed + mock-built + kept demo-ready. Switch-on is a config flip once licensed. |
| Biggest risk? | "Just another neocloud" perception (mitigated by curated catalog + DX + the exchange as funded act-two), and liquidity/license timing for Phase 2. |

---

## 3. The two-phase plan

```
   PHASE 1 — PLATFORM (now → revenue)            PHASE 2 — EXCHANGE (when licensed)
   ─────────────────────────────────            ──────────────────────────────────
   AI startups consume compute + inference        Tradeable credits, order book,
   Datacenters supply capacity                     market maker, public index, settlement
   Prepaid/redeemable credits (no license)         Secondary market (license required)
   → REVENUE NOW                                   → fee revenue + price-discovery moat
            │                                                  ▲
            └──── builds the underlying, the demand, ──────────┘
                  and the price data the exchange needs
```

---

## 4. Roadmap at a glance

| Milestone | Goal (what "done" looks like) | Gate | Status | % |
|---|---|---|---|---|
| **M1** | Foundation: `make up` works; auth + CLI login + empty wallet + console shell on mock data. | Gate 1 | ⬜ | 0% |
| **M2** | First inference dollar (sandbox): buy credits → run Llama through real vLLM → wallet shows debit. | Gate 2 | ⬜ | 0% |
| **M3** | First real customer revenue: ≥1 paying AI startup; on-demand GPU rental; reserved capacity. | Gate 3 | ⬜ | 0% |
| **M4** | Supply + enterprise: ≥1 partner DC serving load; SAML SSO; sub-accounts; first partner payout. | Gate 4 | ⬜ | 0% |
| **M5** | Catalog scale + clusters: full model catalog; multi-node InfiniBand clusters; proof-of-reserves; 2nd partner DC. | Gate 5 | ⬜ | 0% |
| **M6** | GA + license decision: $300K MRR; SOC 2 Type I; jurisdiction decision; Phase 2 switch-on scheduled. | Gate 6 | ⬜ | 0% |

**Pre-build (Phase 0) — complete:** product design, screen mockups, the full engineering plan
(per-agent + per-feature), repo + contract scaffolding, and engineering standards/tooling. ✅

---

## 5. Decision gates (go / no-go checkpoints)

Each gate is a go/no-go at the **end of its milestone** (not a calendar date). Pass criteria
below; mark ✅/⚠️/🔴 at review.

### Gate 1 — M1 — Foundation
- [ ] Cluster up; CI/CD green.
- [ ] Auth + first CLI command work end-to-end (`login`, `credits balance` → $0).
- [ ] Platform console shell renders on mock data.
- **If fail:** reduce scope; inference + compute do not start until met.
- **Status:** ⬜

### Gate 2 — M2 — First inference dollar (sandbox)
- [ ] vLLM serves Llama-70B + Llama-8B through the gateway.
- [ ] Usage debits the credit ledger correctly.
- [ ] Stripe prepaid purchase books to the ledger.
- [ ] Sub-5-minute signup → first inference (measured in CI).
- **Status:** ⬜

### Gate 3 — M3 — First real customer revenue
- [ ] ≥1 AI-startup customer on real-money inference.
- [ ] On-demand GPU self-serve working (<90s to running).
- [ ] Reserved-capacity purchase live; ACH/wire + JPY live.
- [ ] SOC 2 audit firm engaged.
- **Status:** ⬜

### Gate 4 — M4 — Supply + enterprise
- [ ] ≥1 partner DC actively serving real customer load.
- [ ] SAML SSO live; sub-account RBAC working.
- [ ] First partner payout cycle wired; backing ratio published.
- **Status:** ⬜

### Gate 5 — M5 — Catalog scale + clusters
- [ ] Full top-3–5-per-category catalog serving.
- [ ] Multi-node clusters available; gang scheduling proven at scale.
- [ ] Proof-of-reserves published; 2nd partner DC.
- **Status:** ⬜

### Gate 6 — M6 — GA + license decision
- [ ] $300K MRR (consumption + prepaid purchases).
- [ ] SOC 2 Type I report achieved.
- [ ] Licensing jurisdiction decision made.
- [ ] Phase 2 switch-on date scheduled (or explicit, reasoned deferral).
- **Status:** ⬜

---

## 6. Feature tracker — Phase 1 (Platform, active)

Status legend: ⬜ Not started · 🔵 In progress · ✅ Done · ⚠️ At risk · 🔴 Blocked.

| ID | Feature | Owner team | Target | Status | % | Notes |
|---|---|---|---|---|---|---|
| F01 | Foundation infra (K8s, data plane, CI/CD, observability, secrets) | Infra/SRE | M1 | 🔵 | ~50% | Scaffold + **core data plane (Postgres/TimescaleDB/Redis/NATS) verified on k3d** (all 1/1 Ready). Pending: FULL=1 scheduling+observability, real envs (staging/prod), SOPS secrets, CI-green, backup/restore + e2e skeleton. |
| F02 | Auth — email/OAuth (M1); SAML/SCIM/2FA (M4) | Platform | M1 / M4 | ✅ | ~90% | **M1 core complete & deployed in k3d (v0.1.1)** — signup/login→JWT/me/logout/API-keys, bcrypt + HS256, case-insensitive identity, anti-enumeration. **Cross-service auth proven live: one platform-core login → credit-ledger verifies the JWT** (shared `platform-auth` secret). Review gate green. Later (M4): OAuth providers, SAML/SCIM, 2FA, full accounts/org surface (F03). |
| F03 | Accounts, orgs, sub-accounts, RBAC, audit log | Platform | M1 / M4 | ⬜ | 0% | Sub-accounts M4. |
| F04 | `exascale` CLI (login → infer → gpu → billing) | Platform | M1→M6 | ⬜ | 0% | Primary engineer interface. |
| F05 | Prepaid credit ledger (balances, append-only tx, hash chain) | Ledger | M1→M4 | ✅ | ~95% | **Phase-1 complete & deployed in k3d** — domain + store + API + NATS events + self-migrating k8s deploy, all verified (pod 1/1 Ready, live purchase→chain ok). Now verifies real platform-core JWTs (F02 integration live). Later: prod overlays/HPA, scheduled reconciliation. See STATUS.md. |
| F06 | Credit purchase / billing (Stripe → ACH/wire → multi-currency) | Platform + Ledger | M2 / M3 | ⬜ | 0% | The revenue rails. |
| F07 | Credit conversion (AI ↔ sub-credits, GPU tiers) | Ledger | M2 / M3 | ⬜ | 0% | |
| F08 | Inference gateway (OpenAI-compatible, auth, debit) | Inference | M2 | ✅ | ~85% | **Core complete & deployed in k3d (v0.2.0).** OpenAI-compatible `/v1/chat/completions` (mock backend + SSE) + `/v1/models`; API-key auth via platform-core introspection; **pre-flight 402**; usage → `inference.usage.v1` → ledger **idempotent debit**. **Proven live: API key → infer → wallet debit `100→99.865`.** Remaining: real vLLM (F09) behind the backend interface, the other modality endpoints. |
| F09 | vLLM deployment — first 3 models | Inference | M2 | ⬜ | 0% | Llama-70B/8B, Whisper. |
| F10 | Curated SoTA catalog (full) | Inference | M3 / M5 | ⬜ | 0% | The differentiation. |
| F11 | Multi-model-per-GPU packing | Inference | M3 | ⬜ | 0% | Unit economics. |
| F12 | Compute control plane (Kueue + Volcano + GPU Operator) | Compute | M2 | ⬜ | 0% | "Most important part." |
| F13 | GPU instance lifecycle (on-demand, <90s) | Compute | M3 | ⬜ | 0% | |
| F14 | Reserved capacity (1/6/12-mo discounts) | Compute + Ledger | M3 | ⬜ | 0% | Cash upfront. |
| F15 | Multi-node clusters (InfiniBand, gang-scheduled) | Compute | M5 | ⬜ | 0% | Sales-engaged for 32+ GPU. |
| F16 | Supply-source abstraction (owned + partner = one pool) | Compute + Settlement | M3 / M4 | ⬜ | 0% | |
| F17 | DC partner onboarding (manual v1) | Settlement | M4 / M5 | ⬜ | 0% | |
| F18 | Partner payouts (escrow + streamed; proof-of-reserves) | Settlement + Ledger | M4 / M5 | ⬜ | 0% | |
| F19 | GPU attestation (KYB + NVIDIA + challenge-response + DCGM + bond) | Settlement | M4 | ⬜ | 0% | Makes credits credible. |
| F20 | Platform console UI (catalog, wallet, compute, billing) | Frontend | M1→M6 | 🔵 | ~20% | Existing Nuxt app + screen mockups already built. |
| F21 | SOC 2 Type I (Vanta, controls, evidence) | Security + Infra | M4 / M6 | ⬜ | 0% | |
| F22 | Licensing track (jurisdiction, counsel, prepaid framing) | Security | parallel | 🔵 | ~10% | Counsel engagement starting; keeps Phase 2 unblockable. |

---

## 7. Feature tracker — Phase 2 (Exchange, paused / kept-warm)

These are **not** being shipped to customers now. They are designed, partly mock-built, and kept
demo-ready so the eventual switch-on is a config flip, not a rebuild.

| ID | Feature | Owner team | Status | Notes |
|---|---|---|---|---|
| KW01 | Private internal reference index (from real platform tx) | Index | ⏸ → 🔵 M5 | Methodology published from day one; index runs internal-only. |
| KW02 | Trading dashboard kept warm (investor demo) | Frontend | 🔵 | Exists; shows "Demo — exchange paused" ribbon. |
| KW03 | Matching engine — mock adapter + real-engine spec | Trading Systems | ⏸ | Mock data drives the demo; engine specced for switch-on. |
| KW04 | Market maker — design + risk-control spec | Trading Systems | ⏸ | Spec only until license. |
| KW05 | Surveillance — design + basic platform abuse monitoring | Surveillance | ⏸ (partial) | Basic abuse runs now; full trade surveillance waits. |

**Phase 2 switch-on (post-license)** is a sequence of flag flips + deploys: matching engine on,
ledger settlement on, market maker deployed, surveillance full, index public + tradeable, trading
UI live. No new architecture.

---

## 8. Success metrics (targets)

> Trading metrics from the original plan are **paused** (no live exchange). The metrics below are
> the platform-phase targets that matter now.

### Revenue & demand
| Metric | M3 target | M6 target |
|---|---|---|
| Paying organizations | ~10 | ~50–150 |
| MRR (consumption + prepaid purchases) | $50K | **$300K** |
| Prepaid credit purchase volume (cash inflow) | $50K | $1M+ |
| Inference requests / day | 100K | 2M |
| Compute hours / day | 200 | 2,000 |
| F500 / frontier-lab engaged (warm pipeline) | 3–5 | 10–15 |

### Supply
| Metric | M3 | M6 |
|---|---|---|
| DC partner conversations | 5–10 | 15–25 |
| DC partners onboarded | 0 | 1–2 |
| Partner capacity (GPU-hours/day) | 0 | 200–500 |

### Reliability (SLA targets)
| Metric | Target |
|---|---|
| Inference API uptime | 99.95% |
| Compute uptime | 99.9% |
| Credit ledger integrity (hash chain) | 100% (zero tolerance) |
| Private index publication | 100% (never miss a daily print) |

### Strategic (M6)
- SOC 2 Type I report achieved.
- 5+ reference customers.
- 1–2 DC partnerships operational.
- Licensing jurisdiction decision made; Series A conversations advancing.

---

## 9. Team & ownership

| Domain (agent) | Phase 1 role | Owner / hire |
|---|---|---|
| Infra / SRE | Foundation, CI/CD, observability, deploy | _assign_ |
| Platform-core | Auth, billing, RBAC, CLI (the revenue rails) | _assign_ |
| Credit ledger | Prepaid balances, debits, hash chain | _assign_ |
| Inference / ML | Gateway, vLLM, catalog, packing | _assign_ |
| Compute platform | K8s control plane, GPU lifecycle, supply pool | _assign (critical hire)_ |
| Settlement / trust | DC onboarding, attestation, payouts | _assign_ |
| Frontend | Platform console + kept-warm trading demo | _assign (priority hire)_ |
| Security / compliance | SOC 2 + licensing track + review gate | _assign_ |
| Tech lead | Contracts, sequencing, cross-service review | Ahmed |
| Trading systems | Matching engine + market maker (mostly spec in Phase 1) | _assign (Phase 2 critical hire)_ |
| Index / surveillance | Private index + basic abuse monitoring | _assign_ |

**Critical hires, priority order:** (1) compute/GPU infra, (2) inference/ML platform,
(3) senior frontend, (4) platform/backend, (5) enterprise sales lead, (6) DC partnership lead.
(Trading-systems hire moves to Phase 2 priority under the pivot.)

---

## 10. Risk register

| ID | Risk | Likelihood | Impact | Mitigation | Status |
|---|---|---|---|---|---|
| R1 | Looks like "just another neocloud" | Med | High | Curated SoTA catalog + AI-startup DX + transparent pricing; exchange as funded act-two | Open |
| R2 | License never comes / takes too long | Med | Med | Platform is profitable standalone; exchange is upside, not survival | Open |
| R3 | "Prepaid credits" drifts toward looking tradeable | Low | High | Counsel sign-off that redeemable-only stays clear; no customer-to-customer transfer in Phase 1 (F22) | Open |
| R4 | GPU supply / NVIDIA allocation crunch | Med | High | Own DC anchor + partner DCs; AMD backup path | Open |
| R5 | Enterprise sales cycles longer than projected | Med | Med | Self-serve AI-startup revenue doesn't depend on enterprise; multi-deal pipeline | Open |
| R6 | Gang-scheduling correctness (wasted GPU on partial placement) | Med | High | Heavy testing at 32–256 GPU scale; inference favored over training for reliability | Open |
| R7 | Ledger bug → financial/audit incident | Low | Critical | Append-only + hash chain + atomic writes + fault-injection tests + zero-tolerance alerting | Open |
| R8 | Partner DC reliability hurts brand | Med | High | Attestation stack + SLA terms + staked bond + capacity verification before activation | Open |
| R9 | Team doesn't materialize on time | Med | High | Aggressive recruiting; critical-path hires first; plan scoped to reduce if needed | Open |

---

## 11. Critical path (what unblocks what)

```
Foundation infra (F01)
   └─> Auth + CLI (F02/F04) ──> Credit ledger (F05)
            └─> Inference gateway + vLLM (F08/F09) ─┐
            └─> Compute control plane (F12) ────────┤──> debits flow through the ledger
                  └─> Billing / purchase (F06) ─────┘
                        └─> Supply onboarding + payouts (F16/F17/F18) [needs real consumption first]
                              └─> Platform console ties it together for customers (F20)
                                    └─> SOC 2 + licensing run in parallel (F21/F22)
```

If F01 slips, everything slips. If the ledger (F05) slips, inference revenue (F08) slips.

---

## 12. Decisions needed from management

These are open and gate parts of the plan. Tracked so they don't silently stall the build.

| # | Decision | Needed by | Owner |
|---|---|---|---|
| 1 | Prepaid credit terms (expiry, refundability, transfer restrictions — the line that keeps it license-free) | M1 | Founder + counsel |
| 2 | Licensing jurisdiction direction (US / Japan-UBS / Singapore / offshore) | M5 | Founder + counsel |
| 3 | First-revenue target that flips ❌→✅ (MRR + customer count + by when) | M1 | Founder |
| 4 | Owned-DC scale online now + first partner-DC target date | M1 | Founder + DC lead |
| 5 | Multi-currency at launch — confirm USD + JPY | M2 | Founder |
| 6 | AI ↔ sub-credit conversion: bidirectional w/ spread vs. one-way | M2 | Tech lead + counsel |
| 7 | GitHub team handles for code-ownership routing (CODEOWNERS) | M1 | Eng lead |

---

## 13. Where the detail lives (for the engineering team)

This document is the management view. The full engineering plan it summarizes:

- `docs/plans/README.md` — master index.
- `docs/plans/SEQUENCING.md` — milestone-by-milestone (M1–M6) build order + sync points.
- `docs/plans/agents/` — per-team detailed plans (13 files).
- `docs/plans/features/` — per-feature specs (F01–F22, KW01–KW05).
- `docs/plans/REPO_LAYOUT.md` · `LOCAL_DEV.md` · `DEPLOYMENT.md` · `CONTRACTS.md` —
  setup, local dev, deployment, and how teams stay in sync.
- `docs/plans/ENGINEERING_STANDARDS.md` — code quality, security, and review discipline.

---

## 14. Change log

| Date | Change | By |
|---|---|---|
| 2026-05-29 | Initial plan & tracker created from the GTM pivot + Phase 6 v2 design. | Eng lead |
| _add rows at each weekly review_ | | |
