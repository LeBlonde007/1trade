# Exascale — GTM Focus & Sequencing Update

> **Decision (2026-05):** Focus on **AI startups (demand) + datacenters (supply)** — the compute &
> inference platform. **Pause the exchange / trading layer until the license is secured.** Credits
> stay (as prepaid, redeemable units); only the *trading* of them is paused.
> Supersedes the "trading layer is the headline product, ship first" sequencing in Phase 6 v2.

---

## 1. GTM progress

| Stage | Status | Meaning |
|---|---|---|
| **Vision** | ✅ done | The commodity-market-for-AI-compute thesis is set. |
| **Product** (design, code, launch) | ☑️ in progress | Design done; code + launch underway. |
| **Platform** | ▶ current focus | Compute + inference for AI startups, backed by owned + partner DCs. |
| **Exchange** | ⏸ paused | License-gated. Designed and mock-built; switched on once licensed. |
| **Revenue** | ❌ not yet | The immediate target — and now reachable *without* the license. |

The single most important line: **Revenue moves from "blocked behind a license" to "reachable now,"**
because the platform earns money the moment AI startups consume compute and inference.

---

## 2. Why pause the exchange

The exchange was always the hardest and riskiest layer. Pausing it removes the three things that
would otherwise block revenue and raise risk:

- **License dependency.** A tradeable secondary market needs regulatory clearance (jurisdiction TBD —
  US/Japan/Singapore). That's months of counsel and process. Revenue shouldn't wait on it.
- **Liquidity formation.** Every prior compute-trading attempt died on liquidity, not technology.
  Forcing a market before there's real supply and demand is the classic failure mode.
- **No underlying yet.** A credit is only credible if it's backed by real, consumed compute. Build
  the platform first and the eventual exchange inherits real supply, real demand, and real price data.

Pausing isn't retreat — it's correct ordering. The platform *is* the foundation the exchange needs.

---

## 3. The two-phase plan

```
   PHASE 1 — PLATFORM (now)                    PHASE 2 — EXCHANGE (when licensed)
   ───────────────────────────                ──────────────────────────────────
   AI startups consume compute + inference     Tradeable credits, order book, market
   Datacenters supply capacity                 maker, public index, settlement
   Prepaid/redeemable credits (no license)     Secondary market (license required)
   → REVENUE NOW                               → fee revenue + price-discovery moat
            │                                              ▲
            └──────── builds the underlying, the ──────────┘
                      demand, and the price data
                      the exchange needs to be credible
```

The platform generates the data, supply, and demand that make the exchange real later. Same
long-term vision; de-risked sequence.

---

## 4. Phase 1 — Platform (active)

### 4.1 ICP (sharpened)
**Primary: AI startups** — Series A–C, $200K–$5M annual compute spend, heavy inference + training,
DX-sensitive, fast to adopt, frustrated by hyperscaler pricing/complexity. (This is Persona C/D from
Phase 6, now promoted to primary.)
**Supply side: datacenters** — own DC as the v1 anchor + partner DCs for scale.
Frontier labs and Fortune 500 remain warm pipeline, not the v1 wedge.

### 4.2 Product surface that ships now
- **Inference**: curated SoTA OSS catalog (text/code/speech/image/video/embeddings), OpenAI-compatible
  API, multi-model-per-GPU. The fastest path to first revenue.
- **Compute**: GPU rental, CLI-first, on-demand + reserved (reserved = prepaid credits).
- **Credits as prepaid units**: buy in advance, redeem against inference or GPU usage. Redeemable,
  **not tradeable** — so no license needed. Improves cash flow (cash in before consumption).
- **Platform console**: the frontend (formerly the trading dashboard) refocuses on a compute/inference/
  credits/billing console. Sub-5-minute time-to-first-action stays.
- **Datacenter supply onboarding**: bring owned + partner capacity online as one scheduling pool.

### 4.3 Differentiation in the interim
With trading paused, Exascale competes more directly as a neocloud — so the platform-phase edge has
to come from elsewhere: the **curated, always-current SoTA catalog**, **AI-startup-first DX**,
**transparent pricing**, and **prepaid-credit flexibility**. Keep the exchange vision alive in the
investor narrative as the second act — it's what makes Exascale more than "another neocloud."

### 4.4 How ❌ Revenue becomes ✅
First-dollar paths, in order of speed:
1. **Inference consumption** — pay-per-use, fastest to land with AI startups.
2. **GPU rental** — on-demand compute hours.
3. **Prepaid credit purchases** — cash upfront, redeemable; strengthens runway.
4. **Reserved capacity** — startups locking in 1–12 month terms (discounted).
5. *(Phase 2, post-license)* **Trading fees** — the maker-taker exchange revenue.

---

## 5. Phase 2 — Exchange (paused, license-gated)

### 5.1 What's paused
Order matching, market maker, the public tradeable index, trade surveillance, secondary-market
settlement — everything that constitutes trading credits.

### 5.2 Parallel workstream: the licensing track
Pausing the *build* doesn't mean pausing the *path*. Run a low-cost parallel track so the license is
ready when the platform is:
- Decide jurisdiction (US / Japan via UBS / Singapore / offshore).
- Engage securities/commodities counsel; confirm whether credits-as-prepaid stays clear and what
  trading triggers licensing.
- Map the specific license/registration required and its timeline.

### 5.3 Keep-warm (cheap, high-option-value)
- **Internal reference price index**: keep computing a price benchmark from *real platform
  transactions* — private, non-tradeable. This builds the data backbone and methodology credibility
  so the public index launches strong on day one.
- **Mock trading UI**: keep it demo-able for investors as the "act two" story — clearly labeled future.
- **Designed-but-dormant services**: matching engine, market maker, surveillance stay specced (and
  mock-built) so the exchange is a switch-on, not a rebuild, once licensed.

### 5.4 Trigger to resume
License secured (or a clearly-clear jurisdiction confirmed) **and** platform liquidity precursors
present (real supply + demand + price data). Then flip the exchange on.

---

## 6. Build implications

How this maps onto the agent roster / Phase 6 components:

| Component / agent | Phase 1 status | Note |
|---|---|---|
| `inference-ml` | ✅ active — core | Fastest revenue path |
| `compute-platform` | ✅ active — core | Owned + partner DC scheduling |
| `platform-core` (auth, billing, RBAC, CLI) | ✅ active — core | Billing = the revenue rails |
| `credit-ledger` | ✅ active (simplified) | Prepaid + consumption debit; trading settlement paused |
| `settlement-trust` | ◐ partial | DC onboarding + backing/payout active; tradeable-contract + trade-escrow deferred |
| `infra-sre` | ✅ active — foundation | |
| `security-compliance` | ✅ active + owns licensing track | SOC 2 + KYC now; license path for Phase 2 |
| `trading-frontend` | ◐ repurposed | Becomes the platform console; trading UI kept warm as demo |
| `index-service` | ◐ keep-warm | Private reference index only; public/tradeable paused |
| `matching-engine` | ⏸ paused | Specced + mock; switch-on later |
| `market-maker` | ⏸ paused | |
| `surveillance` | ⏸ paused (trade) | Keep basic abuse monitoring; trade surveillance waits |

Critical-path now: `inference-ml` → `compute-platform` → `platform-core` (billing) → `credit-ledger`
(prepaid) → `settlement-trust` (DC supply). That's the revenue machine.

---

## 7. Milestones — closing out each GTM stage

| Stage | "Done" looks like |
|---|---|
| Product | Platform console + inference API + CLI launched to first AI-startup users |
| Platform | Owned + ≥1 partner DC live; catalog serving; reserved + on-demand available |
| Revenue | First paying AI startups; consumption + prepaid credit cash flowing; target first MRR |
| Exchange | License secured → matching/MM/index switched on → first trading fees |

---

## 8. Risks of the pivot + mitigations

| Risk | Mitigation |
|---|---|
| Looks like "just another neocloud" without the exchange | Lead DX + curated catalog + transparent pricing; keep exchange as the funded act-two narrative |
| Investors bought the exchange thesis | Reframe: platform = revenue + the credible underlying; exchange = the moat, de-risked and license-pending |
| "Prepaid credits" drifts toward looking tradeable | Counsel sign-off that redeemable-only stays clear of licensing; no secondary transfer until licensed |
| License never comes / takes too long | Platform is a standalone profitable business regardless; exchange is upside, not survival |
| Team built for trading-first | Re-point critical path to inference/compute/billing; trading talent works keep-warm + Phase 2 prep |

---

## 9. Open questions

1. **AI-startup ICP specifics** — which segments first (inference-heavy app builders vs. training-heavy labs)?
2. **Prepaid credit terms** — expiry, refundability, transfer restrictions (the line that keeps it license-free).
3. **Licensing jurisdiction + timeline** — what exactly triggers the requirement, and how long?
4. **Datacenter status** — own DC scale online now; first partner DC target date.
5. **Investor framing** — how explicitly to position the exchange as "paused, pending license" vs. "phase two."
6. **First-revenue target** — what MRR / customer count flips ❌ → ✅, and by when.
