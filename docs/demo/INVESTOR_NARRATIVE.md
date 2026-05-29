# Exascale — Investor Narrative (current state, 2026-05)

> **Primary NotebookLM source.** This is the spine of the investor video: the story, grounded in
> what is actually built today. It reflects the **2026-05 platform-first pivot** (not the older
> trading-first pitch). All financial figures are **illustrative/mock** — replace before sending to
> real investors. Companion sources: `UI_WALKTHROUGH.md` (the screens), `VIDEO_SCRIPT.md` (the cuts).

---

## 1. One sentence

**Exascale is the world's first exchange and platform for AI and GPU compute — already earning
revenue by selling inference and GPU compute to AI startups, paid for in prepaid credits.**

## 2. The thesis (the big vision)

AI compute is the new oil — but it trades like real estate: opaque, bilateral, illiquid. There's
no public price for an H100-hour, no order book, no way to hedge a compute budget. Every commodity
that became *liquid* in the last century — oil, FX, electricity — minted a fortune for whoever
built the venue. Compute is the last large commodity with no venue. **That venue is the long-term
prize.**

## 3. The insight that changed our plan (why this is investable now, not in 3 years)

The exchange is the hardest, riskiest layer: it needs deep liquidity, and a credit is only credible
once it's backed by **real, consumed compute**. Most compute-trading attempts die right there —
they force a market before there's anything real underneath it.

So we inverted the order. **Build the platform first.** Sell inference and GPU compute to AI
startups now, paid for in **prepaid, redeemable credits** (a unit of service). The platform throws
off revenue immediately *and* produces the supply, demand, and price data a credible market needs —
so the exchange is built on a live underlying, not an empty book.

**Same vision, de-risked order: we earn from the platform today, and the market is built on top of
real activity.**

## 4. What we sell today (Phase 1 — the platform)

- **Inference** — our own platform API serving a curated catalog of state-of-the-art models across
  **text, code, speech, image, and video**. A single integration and an AI startup is live;
  pay-per-use, metered to the token. The fastest path to the first dollar.
- **GPU compute** — on-demand H100/H200, reserved capacity (discounted, prepaid as GPU credits),
  CLI-first. From `exascale gpu create` to a running box in under 90 seconds (target).
- **Prepaid credits** — buy in advance, redeem against inference or compute. Improves our cash flow
  (cash in before consumption) and is the same credit primitive the exchange will later trade.
- **Datacenter supply** — our own datacenter as the credible v1 anchor, plus partner DCs onboarded
  into one scheduling pool, so we can scale supply without owning every GPU.

## 5. The moat — the exchange and the index

On top of the platform sits the exchange: an order book, an automated market maker, trade
surveillance, secondary-market settlement, and a published **AI compute price index** — the layer
that turns a compute *platform* into a compute *market*. The index is the real long-term moat: once
"AI-INDEX" becomes a published reference rate, every downstream compute contract prices off it.
*The index is the venue.*

The price data is already accumulating — a reference index computes from real platform transactions
from day one, so the market launches with history, not from zero.

## 6. Traction — this is built, not a Figma file

As of 2026-05, the platform is **real software, much of it running**:

- **A 31-screen production frontend** (Nuxt 4 / Vue 3) on one design system: the platform console
  (catalog, inference playground, compute, wallet, billing), enterprise admin (SSO, teams, audit,
  billing), the datacenter partner portal, full onboarding/KYC, and the trading floor.
- **A running platform foundation** — Kubernetes (k3s) with the full data plane (Postgres,
  TimescaleDB, Redis, NATS) provisioned by one command, reproducible from code.
- **The financial core is live and verified.** The **credit ledger** — append-only,
  cryptographically **hash-chained** (every movement is `sha256(prev ‖ row)`, tamper-evident),
  exact **fixed-point** money math (no floating-point error), **atomic** balance updates,
  **idempotent** and **per-tenant-isolated** — is built, deployed to the cluster, and verified
  end-to-end (a purchase debits, the balance updates, the audit chain verifies). This is what other
  platforms put on a roadmap; ours is already running.
- **Engineering discipline that institutions require** — contract-first service architecture,
  a per-function documentation mandate, automated secrets scanning, security review gates on
  anything touching money, and an audit trail on every credit movement. The kind of foundation a
  Fortune-500 or frontier-lab buyer (and later, a regulator) will sign off on.

What is **not** yet built — and what the round funds: the inference gateway + GPU control plane at
scale, billing rails (Stripe/ACH/wire), the partner-DC agent, SOC 2, and the licensing track that
unlocks the exchange.

## 7. Who buys (sharpened ICP)

**Primary: AI startups** (Series A–C, $200K–$5M annual compute spend) — inference-heavy,
DX-sensitive, frustrated by hyperscaler pricing and complexity, fast to adopt. **Supply side:
datacenters** with idle GPU capacity and no channel to AI buyers. Frontier labs and Fortune-500
procurement are a warm pipeline, not the v1 wedge.

## 8. Why us / why now

- **Demand inflection** — LLM token volume has exploded; demand outstrips supply; AI startups
  overpay and under-optimize.
- **Supply fragmentation** — hundreds of mid-tier datacenters now hold real GPU capacity with no
  route to AI buyers except a hyperscaler.
- **The clearing stack is commodity software now** — Kubernetes, hash-chained audit, standardized
  model APIs. What took an exchange a billion dollars to build, we run on rented servers.
- **A neutral venue is structurally un-buildable by the incumbents** — AWS/Azure/GCP won't invite
  their own customers to comparison-shop them; frontier labs want to *be* the buyer, not the venue.

## 9. Business model (illustrative)

- **Now:** margin on inference + compute consumption, plus prepaid-credit float (cash before
  consumption). Standard neocloud-plus economics, but with a curated catalog and AI-startup-first DX.
- **At scale:** maker-taker **venue fees** on every trade + forward-roll fees — software margin on
  software margin; the marginal cost of clearing the millionth trade is ~zero. The index becomes a
  licensable reference rate.

## 10. The ask (illustrative — replace)

Seed round to fund: the inference + compute platform to GA, billing rails, the first datacenter
partners and AI-startup customers, SOC 2, and the path to the open market. The platform is a
standalone, profitable business; the exchange is the venture-scale upside on top of it.

## 11. The closing line

> Every commodity that became liquid made someone a fortune by being the venue. Compute is the last
> one that hasn't. We're building the venue — and unlike everyone who tried before, we're earning
> revenue and proving the underlying from day one.
