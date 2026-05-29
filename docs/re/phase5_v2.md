# Phase 5 (v2) — Product & Differentiation Strategy
**The credit-market thesis. Three-sided market. Who trades, who supplies, who consumes.**

> Status: Phase 5 v2 — 2026-05-19. Reflects Tai's latest direction on ICP (focus on AI frontier labs and Fortune 500 demand), supply-side via datacenter partnerships, UI/UX-first sequencing, and fundraising posture.
> Replaces: prior Phase 5 v1 which framed traders as sole primary ICP and treated demand-side as secondary.
> Owner: Ahmed.

---

## Table of contents

1. [What Exascale actually is](#1-what-exascale-actually-is)
2. [The credit architecture](#2-the-credit-architecture)
3. [The three-sided market structure](#3-the-three-sided-market-structure)
4. [Institutional validation and timing](#4-institutional-validation-and-timing)
5. [The ICP — demand, liquidity, supply](#5-the-icp--demand-liquidity-supply)
6. [Why not just back CoreWeave? — answered properly](#6-why-not-just-back-coreweave--answered-properly)
7. [The v1 product priorities — UI/UX first](#7-the-v1-product-priorities--uiux-first)
8. [The four things we never compromise](#8-the-four-things-we-never-compromise)
9. [Fees and revenue model](#9-fees-and-revenue-model)
10. [Demand engine and GTM](#10-demand-engine-and-gtm)
11. [Supply engine and datacenter partnerships](#11-supply-engine-and-datacenter-partnerships)
12. [Fundraising posture](#12-fundraising-posture)
13. [Brand voice and competitive narrative](#13-brand-voice-and-competitive-narrative)
14. [Strategic risks](#14-strategic-risks)
15. [Open design questions](#15-open-design-questions)

---

## 1. What Exascale actually is

After all the research and Tai's clarifications, the strategic answer is:

> **Exascale is the commodity market for AI compute — a venue connecting GPU datacenters with the AI companies that need their capacity, intermediated by traders who provide liquidity and price discovery. We own modest datacenter capacity to provide credible underlying delivery; we partner with larger operators to source supply; the business is the market itself, not the metal.**

The chain we are building:

```
   GPU DATACENTERS  →  TRADERS  →  AI COMPANIES
       (supply)      (liquidity)    (demand)
       
   Idle capacity     Price          Frontier labs
   New DC builds     discovery      Fortune 500
   Reserved unused   Hedging        Enterprise AI
```

The closest analogues:
- **CME** (futures exchange) — runs markets for commodities; doesn't own oil refineries
- **Coinbase** (crypto exchange) — runs markets for crypto; doesn't mine Bitcoin
- **ICE** (commodities + financial exchanges) — runs markets; doesn't pump natural gas
- **Bloomberg** (data + index + platform) — the reference price benchmark for the asset class

What Exascale is *not*:
- Not CoreWeave (vertically-integrated GPU cloud)
- Not Together (managed inference platform)
- Not AWS (services-everything-to-everyone)
- Not Vast.ai (consumer GPU aggregator)
- Not a marketplace in the StubHub/eBay sense (we provide the venue + liquidity, not just listings)

### Why this framing matters

The "three-sided market" framing changes strategic priorities:

| Decision | If we're a neocloud | If we're a credit market |
|---|---|---|
| Primary customer | AI engineering teams | **Three sides: supply (DCs) + liquidity (traders) + demand (AI companies)** |
| Revenue model | Per-GPU-hour markup | **Trading fees + spread + index licensing + consumption** |
| Capital efficiency | Capex-heavy | **Capital-light — we don't need to own most of the underlying** |
| Competitive moat | Hardware allocation + ops scale | **Liquidity + price discovery + market structure + warm enterprise relationships** |
| Build priority | Compute platform first | **Trading interface + credit ledger + demand engine + supply partnerships** |
| Exit dynamics | Acquired for capacity | **CME-style exchange acquisition, or independent venue at scale** |

### Positioning summaries

**Long form (one sentence)**: The commodity market for AI compute — connecting GPU datacenters with AI companies through tradeable credits, with traders providing liquidity and price discovery.

**Ten-word version**: A three-sided market for AI compute — supply, liquidity, demand.

**Five-word bumper sticker**: AI compute, but tradeable.

---

## 2. The credit architecture

Tai's design (from his 2026-05-19 messages), formalized.

### Two asset classes

**Asset Class 1: AI Credits** (the unified inference market)

A unified index credit, redeemable into sub-credits at conversion rates Exascale publishes:

```
                       UNIFIED AI INDEX
                       (the headline tradeable asset)
                              │
              ┌───────┬───────┼───────┬───────┐
              ▼       ▼       ▼       ▼       ▼
            TEXT   SPEECH   IMAGE  VIDEO   (niche)
           credits credits credits credits  credits
              │       │       │       │       │
              ▼       ▼       ▼       ▼       ▼
           LLM     TTS +   image   video  embeddings,
          tokens   STT    gen     gen    rerankers,
                 minutes  per-img sec    classifiers
```

**Key properties**:
- **Headline tradeable instrument**: the unified AI credit (the index)
- **Reference price**: $1 = N AI credits, where N is set by methodology
- **Redemption**: customer holds AI credits → redeems for any sub-credit at published conversion rate at redemption time → consumes inference
- **Sub-credit trading**: sub-credits can also trade independently (separate, thinner markets)
- **Spot index update**: probably daily; weighted basket of sub-credit utilization

**Asset Class 2: GPU Credits** (the raw compute market)

Separate instrument, separate market:

```
GPU CREDITS (per tier)
├── H100 80GB credits     (= 1 GPU-hour of H100 capacity)
├── H200 credits          (= 1 GPU-hour of H200 capacity)
├── [later] B200 credits  (Phase 2 — when capacity expands)
└── [later] B300 credits  (Phase 3)
```

**Key properties**:
- **Each tier trades separately** (different prices for different hardware)
- **Used for**: GPU rental, training, fine-tuning, customer-managed workloads
- **Settlement**: physical (delivered as GPU time) or cash (closed before delivery)
- **Different customer base than AI credits**: more sophisticated, more workload-specific

### Conversion mechanics

Three conversion directions matter:

| Conversion | Allowed? | Mechanism |
|---|---|---|
| AI credit → sub-credit | Yes | Published redemption ratio (e.g., 1 AI credit = 1.2 text credits at current ratio) |
| Sub-credit → consumption | Yes | Standard usage (text credit → tokens, image credit → generations, etc.) |
| Sub-credit → AI credit | **Needs Tai's decision** | Bidirectional makes liquid markets; one-way preserves index purity |
| Sub-credit → sub-credit | **Needs Tai's decision** | Convenient for users; arbitrage risk |
| AI credit ↔ GPU credit | **No** (default) | Two different asset classes; market sets relative prices |
| GPU credit → consumption | Yes | Spent down by GPU rental, per-second per-tier |

**Recommendation**: bidirectional AI ↔ sub-credit (with a small "house spread" — like FX bid-ask). No direct sub ↔ sub (force through the index). No AI ↔ GPU (two markets stay separate).

### Why the unified-index structure is genuinely smart

Three reasons:

**1. Liquidity concentration**. If we had only sub-credits, we'd have 5+ thin markets. Each would have wide spreads, poor price discovery. The unified index concentrates trading volume in one deep market.

**2. Index = headline product**. The "Exascale AI Index" becomes a quotable number — what is "the price of AI" today? Like the WTI Crude price for oil, or VIX for volatility. This is the route to becoming the reference benchmark for the entire AI economy.

**3. Customer optionality**. A buyer doesn't have to predict their text-vs-image-vs-video usage 6 months ahead. They buy AI credits and redeem into actual sub-credits at consumption time. This makes forward purchases viable for customers who can't forecast usage mix.

---

## 3. The three-sided market structure

Markets need supply, liquidity, and demand. Tai's chain is explicit:

```
                    EXASCALE TRADING VENUE
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
     SUPPLY              LIQUIDITY            DEMAND
   (GPU data           (Traders,           (AI frontier
    centers)            market              labs +
                        makers)              Fortune 500)
        │                   │                   │
   Idle capacity        Price            Compute spend
   New DC builds        discovery         that needs
   Reserved hours       Hedging           hedging /
                                          procurement
        │                   │                   │
        └─────────┬─────────┴─────────┬────────┘
                  │                   │
                  ▼                   ▼
            ORDER BOOK         INDEX REFERENCE
            (AI credits,           (the price
             sub-credits,         of compute)
             GPU credits)
```

### The four participant archetypes

**Type 1: The Demand-Side Buyer (the compute consumer)**
- AI frontier labs ($50M-$10B+ annual compute)
- Fortune 500 AI / IT teams ($1M-$50M annual)
- Mid-market AI companies as a tail (smaller checks, larger count)
- Buys credits to fulfill actual compute consumption
- May hold credits as inventory or hedge; primarily a consumer
- Example: a Fortune 500 retailer pre-buying $5M of AI credits for a Q4 model-launch project

**Type 2: The Liquidity Provider — Trader**
- Prop trading firms, commodity hedge funds, family offices
- Crypto-native market makers diversifying
- Quant funds with AI thesis
- Trades credits like any other commodity — active liquidity
- Most sensitive to fee schedule, execution quality, market data
- Example: a commodity-focused hedge fund treating AI compute as new asset class

**Type 3: The Market Maker (both sides)**
- Provides liquidity by quoting both bid and ask
- Earns the spread minus fees
- Critical for thin markets in v1
- Exascale itself plays this role in v1 (simulated then real)
- Eventually: external market-making firms (Jane Street, Wintermute, etc.)

**Type 4: The Supply-Side — Datacenter Partner**
- Datacenter operators selling capacity into the market
- v1: Exascale's own modest datacenter
- v1+: partner datacenters with idle capacity (significant near-term opportunity)
- v2+: new DC builds anchored by the market (price discovery enables financing)

### Why this matters for product priorities

Each archetype demands different product capabilities:

| Archetype | Critical product features | Defer-able features |
|---|---|---|
| Demand-side buyer | Volume purchase flow; SSO; procurement integration; predictable redemption; CLI tooling | Advanced order types |
| Trader | Low fees; deep order book; market data API; FIX protocol | Web UI polish; managed inference |
| Market Maker | API access; market data feeds; risk controls; rebate structure | Web UI |
| DC Partner | Capacity onboarding flow; settlement mechanics; payout flows; SLA monitoring | Trading interface |

**V1 priority order**:
1. Trading UI/UX (the demo asset — see Section 7)
2. Demand-side buyer onboarding (volume credit purchases, CLI)
3. Market-maker logic (Exascale-internal)
4. Supply-side partner onboarding (manual in v1; productized in v1.5)
5. External liquidity providers (post-v1.5, after liquidity proves)

---

## 4. Institutional validation and timing

The category is being validated by institutional voices outside Exascale. Worth capturing for positioning and pitch:

> **"There needs to be a market for compute. No solution yet."**
> — Larry Fink, CEO BlackRock (2026)

This is the type of macro signal that opens doors. When the largest asset manager in the world is publicly stating the thesis, capital allocators take meetings. Specifically:

- **Sovereign wealth funds and asset managers** are now actively looking for AI infrastructure exposure that isn't tied to a single hyperscaler. A tradeable market gives them that.
- **Commodity-trading firms** (Trafigura, Vitol, Glencore-equivalents) are exploring AI compute as their next adjacency after carbon and electricity.
- **Financial exchanges** (CME, ICE, Cboe) are evaluating whether to build or buy into this category.

### Why now (the timing argument)

Three convergent factors make 2026 the right window:

**1. AI compute is now demonstrably bursty and expensive enough to need hedging.** Through 2024, compute spend was relatively predictable. From 2025 onward, training runs cost more than $100M for a single model and inference traffic is now bursty enough that Fortune 500 CFOs feel the volatility.

**2. The OSS model catalog matured.** A market needs standardized underlying. The Llama 3/4 family, DeepSeek, Mixtral, Qwen, FLUX provide a stable enough catalog that "an AI credit redeemable for inference output" has a well-defined meaning.

**3. Public neocloud comparables exist.** CoreWeave's IPO disclosed the economics of the underlying. Nebius is public. Together AI has raised $1.5B+. The category is fundable.

The window does not stay open forever. CME has reportedly explored AI-compute futures. A hyperscaler could attempt to build something similar. We have a 12-24 month head start to establish category position.

---

## 5. The ICP — demand, liquidity, supply

Tai's instruction was clear: "Focus on AI frontier labs and Fortune 500 — that's where the big spend is." This represents a meaningful shift from the prior Phase 5 framing.

### Primary demand ICP — AI Frontier Labs

**Profile**:
- OpenAI, Anthropic, xAI, Mistral, Cohere, Inflection-class, AI-native scaleups raising late-stage rounds
- 200-5,000 employees
- $50M-$10B+ annual compute spend
- Currently locked into hyperscaler reserved contracts with limited flexibility

**Why they fit**:
- Compute is their dominant operating cost
- Volatility in compute pricing materially affects business planning
- Sophisticated finance teams that understand commodity hedging
- Already operate at the scale where a 5% efficiency gain on compute = $5M-$500M annual savings
- Hyperscaler exclusivity is increasingly painful (Microsoft-OpenAI dynamics, AWS-Anthropic dynamics)

**What they buy**:
- AI credits in bulk for inference workloads
- GPU credits for training workloads
- Forward contracts to lock in capacity (v1.5+)
- Reference index data to track market pricing for internal forecasting

### Primary demand ICP — Fortune 500

**Profile**:
- Walmart, JPMorgan, Saudi Aramco-tier organizations
- Internal AI teams (20-500 within larger enterprise)
- $1M-$50M annual AI compute spend
- Existing hyperscaler relationships but increasingly looking for cross-vendor options

**Why they fit**:
- Procurement-driven buyers comfortable with master agreements and commodity-style purchasing
- Financial sophistication for hedging and forward contracts
- Multi-vendor strategies are explicit policy at most large enterprises
- Compliance posture is mature; SOC 2 + audit log is enough for most use cases (FedRAMP not always required)

**What they buy**:
- Bulk AI credit purchases for budget predictability
- GPU credit reservations for known workloads
- Long-term forward contracts (v1.5+) for multi-year compute planning
- Multi-currency invoicing (USD, JPY, EUR per geography)

### Liquidity-side ICP — Traders

**Profile** (carries forward from prior Phase 5):
- Prop trading desks at banks
- Commodity-focused hedge funds (firms already trading carbon, weather, FX)
- Crypto-native market makers diversifying
- Quant funds with AI thesis
- Family offices and specialist desks

**Size of this universe**: ~30-100 firms globally have explicit mandates to explore AI compute as a tradeable asset class.

**Why they're essential, not optional**:
- Without active traders, the order book is empty and demand-side buyers can't execute at fair prices
- They make the market exist
- They're the reason the venue earns trading fees (the primary revenue line)

**The chicken-and-egg problem**: traders won't show up without volume; volume won't form without traders. Exascale-as-internal-market-maker is the bridge — we provide both sides until external market makers are confident enough to come on.

### Supply-side ICP — Datacenter Partners

**Profile**:
- Tier 2/3 datacenter operators with idle GPU capacity
- New DC build projects looking for anchor demand (e.g., 750MW-class new builds)
- Reserved-capacity holders (enterprises with unused reserved hours)

**Why this matters**:
- v1's owned capacity is not enough to back at-scale demand
- Partnerships unlock 10-100x more underlying without 10-100x more capex
- DC operators benefit from market access to monetize idle capacity (they currently struggle to sell short-tenor or burst capacity)

**The pitch to DC partners**:
- "We give you a venue to monetize idle compute at market clearing prices"
- "You don't need to build a sales motion; traders + AI companies come to us"
- "Settlement and payout are clean — credit movements map directly to your bank account"

### What we now deliberately deprioritize

Updated from prior Phase 5:

- **Indie developers** — no spend, no trading interest. Acquisition channel only.
- **Pure research customers** — no budget. Free credits as mindshare investment.
- **Sub-$200K-spend AI startups** — long tail; lower priority than F500/frontier-lab focus.

**REMOVED from "deprioritize"**: Fortune 500 enterprise IT teams. Per Tai's direction, these are now primary demand. The prior "too slow, won't migrate for years" framing assumed cold outreach; with warm relationships into procurement organizations, sales cycle is compressed significantly.

---

## 6. Why not just back CoreWeave? — answered properly

CoreWeave is publicly traded at $50B+. They have $25B+ backlog. They own 30+ datacenters. Why bet on Exascale?

### The three-part answer

**Part 1: We're not competing in CoreWeave's category.**

CoreWeave is a hyperscale-adjacent compute provider. They sell GPU-hours through reserved contracts. They compete with Lambda, Nebius, the hyperscalers, and Microsoft's internal capacity.

Exascale is a commodity market. There is no functioning tradeable AI compute market today. **The category we're creating doesn't have an incumbent.**

Backing CoreWeave is a bet on continued AI compute demand. Backing Exascale is a bet on AI compute becoming a tradeable asset class. Both can be right simultaneously.

**Part 2: The capital efficiency profile is fundamentally different.**

CoreWeave spent ~$10B+ on capex to reach their current scale. They have ~$7.5B+ in debt. They need utilization rates >70% to service debt.

Exascale's business is market structure + index + trading mechanic. Owned datacenter capacity exists for credibility (proving underlying is real), not as the revenue source. Supply scales primarily through partnerships, not capex. **Capital required to reach $100M revenue is dramatically lower than CoreWeave's capital path to the same milestone.**

CME doesn't own oil. Coinbase doesn't mine. We follow that pattern.

**Part 3: The exit dynamics are structurally different.**

CoreWeave's strategic exit options: scaling further as independent (uncertain economics with debt load), or selling capacity wholesale to hyperscalers.

Exascale's strategic exit options:
- Become the reference exchange for AI compute (CME / NYMEX path)
- Acquisition by an existing financial exchange wanting to enter AI commodities (CME, ICE, Cboe)
- Acquisition by a hyperscaler that wants a tradeable credit product they can't build themselves
- Acquisition by a sovereign wealth fund or asset manager looking for direct exposure to AI infrastructure economics

**Different markets. Different valuation multiples. Different acquirers.**

### The honest counter-argument

"Maybe AI compute doesn't actually become a real tradeable asset class. Maybe the whole credit-market thesis is overengineered."

This is a real risk. The mitigation:

- **Paper-trading v1 validates market interest before betting the company on it**
- **The owned datacenter + inference platform is real business even if the trading layer doesn't take off**
- **Demand-side revenue (F500 + frontier labs buying credits) anchors the business regardless of trading volume**

If the trading layer doesn't take off as hoped, Exascale is still a niche compute provider with enterprise relationships. Not a great outcome, but not catastrophic.

If the trading layer does take off, Exascale becomes the reference venue for an entirely new asset class. Asymmetric upside.

---

## 7. The v1 product priorities — UI/UX first

Per Tai's direction: ship UI/UX before backend. Specifically, ship a working trading interface with mock data and moving candlesticks first, then build backend underneath. The trading UI is the demo asset that opens conversations with demand-side prospects, traders, and investors.

### Sequencing (different from prior Phase 5)

**Months 1-2: UI/UX with light trading simulation**
- Trading interface: order book, depth chart, candlestick chart with moving data
- Credit balance dashboard
- Index publication page (mock data)
- Account management
- Mock data backend (no real matching engine yet — just plausible market behavior)
- **Goal**: demo-able to F500 procurement teams, frontier-lab CTOs, traders, and investors within 8 weeks

**Months 2-4: Real backend underneath**
- Matching engine v1 (real, persistent, recoverable)
- Credit ledger (atomic balance tracking)
- Index service (real methodology, daily publication)
- Market maker logic (Exascale-internal automated quoting)
- Inference platform (curated catalog, multi-tenant per GPU)
- Auth, billing, KYC

**Months 4-6: Production hardening + supply onboarding**
- Real-money internal testing
- DC partnership onboarding (manual flow)
- SOC 2 Type I audit
- Performance and reliability hardening
- Surveillance and risk controls

**Platform side: CLI as primary interface**
- The trading UI is the demo asset and customer-facing
- The compute/inference platform side stays basic UI; CLI is primary for Fortune 500 / frontier-lab engineers
- This matches how enterprise customers actually use cloud compute (kubectl, aws-cli, etc.)
- Saves significant frontend effort that doesn't drive differentiation

### Tier-A capabilities (must ship for v1)

**Trading interface and engine** (the new headline)
- Order book for AI credits (unified index spot market)
- Order book for sub-credits (text, speech, image, video — separate markets, visible)
- Order book for GPU credits (per tier — H100, H200)
- Candlestick charts, depth visualization, market data
- Maker-taker order types (limit + market)
- Exascale internal market-maker logic
- Real-money trading capability tested internally (deferred to customers in v1.5)
- Paper-trading mode for customer-facing v1
- Order management, position tracking, P&L visualization

**Credit ledger**
- Per-tenant balance tracking (AI credits, sub-credits, GPU credits)
- Atomic conversions
- Audit log of all credit movements
- Real-time balance API

**Index publication**
- Daily "$1 = N AI credits" reference price
- Methodology documented publicly
- Constituent weights disclosed
- API endpoint + web display

**Curated model catalog with multi-tenancy**
- Top 3-5 OSS models per category (text, speech, image, video)
- Multi-model-per-GPU deployment
- OpenAI-compatible API surface
- Quarterly catalog refresh

**Compute platform (the underlying)**
- Own modest datacenter capacity (H100/H200; not hyperscale yet)
- GPU rental via CLI (primary interface) and basic web UI
- Self-serve provisioning for sub-32-GPU workloads

**Demand-side onboarding**
- Volume credit purchase flow (cash → credits, ACH/wire/Stripe)
- Multi-currency (USD + JPY at launch)
- Procurement-friendly invoicing
- SSO for enterprise accounts

**Supply-side onboarding** (manual in v1)
- DC partner verification and capacity allocation
- Settlement and payout mechanics
- SLA monitoring for partner-supplied capacity

**Auth & compliance baseline**
- SAML 2.0 SSO for enterprise (F500 demand requirement)
- API keys with scoped permissions
- KYC tiered by activity (light for paper trading, full for real-money)
- SOC 2 Type I path

### Tier-B capabilities (v1.5, months 6-12)

- Real-money trading customer-facing
- Distributed training (after distributed inference proven)
- Forward contracts on credits (1mo, 3mo, 6mo cash-settled)
- Larger model catalog
- B200/B300 capacity (as market matures)
- FIX protocol for institutional traders
- Market data API as separate revenue stream

### Tier-C capabilities (v2+)

- Productized third-party capacity supply onboarding
- Cross-vendor inference routing
- Reference index licensing (Bloomberg-style)
- International expansion

### What we explicitly cut even under pressure

- Mobile app
- FedRAMP compliance (year 2-3)
- Multi-region in v1 (single region with multi-AZ)
- Marketplace functionality in the StubHub/eBay sense
- Custom GPU silicon partnerships (NVIDIA-only)

---

## 8. The four things we never compromise

**1. The trading layer is the headline product, not a feature.**

Even paper-trading v1 must feel like a real market — order book, quotes, P&L, candlesticks, history.

**2. Real-time credit visibility.**

Customers see balances live. Trades settle visibly. Credit movements are transparent.

**3. Sub-5-minute time-to-first-action.**

For traders: signup → first paper trade. For demand-side: signup → first credit purchase quote. For engineers: signup → first API call. All under 5 minutes.

**4. Index methodology integrity.**

Methodology published, constituent weights disclosed, audit firm engaged. The venue's reputation is the index's reputation.

---

## 9. Fees and revenue model

### Trading fees (v1 schedule)

| 30-day volume tier | Maker fee | Taker fee | Tier name |
|---|---|---|---|
| $0 – $10K | 0.50% | 1.00% | Retail |
| $10K – $100K | 0.40% | 0.80% | Active |
| $100K – $1M | 0.30% | 0.60% | Pro |
| $1M – $10M | 0.20% | 0.40% | Pro+ |
| $10M – $100M | 0.10% | 0.20% | Institutional |
| $100M – $1B | 0.05% | 0.15% | Institutional+ |
| $1B+ | 0.00% | 0.10% | VIP |

Starts higher than Coinbase because no competition exists. Drops to competitive levels at institutional volume.

### Revenue streams

**Primary**: Trading fees
- Maker-taker fees on every trade
- Volume-tiered as above

**Secondary**: Compute consumption
- GPU rental at competitive rates
- Inference API at OpenAI-compatible per-token pricing
- Customers pay with credits or cash

**Tertiary**: Index licensing (v1.5+)
- Subscription access to market data
- Historical prints, constituent transparency
- API for institutional traders

**Quaternary**: Spread (v2+)
- Exascale-as-market-maker spread
- Becomes meaningful at scale

### Unit economics

**Compute side**:
- H100 at $2.99/hr on-demand
- H100 at $1.99/hr reserved 1-year
- Llama 70B inference at $0.55/$0.79 per 1M tokens
- ~60-70% gross margin on compute, ~80% on inference

**Trading side**:
- Sandbox v1: zero marginal cost per trade
- Real-money v1.5+: ~0.05% settlement cost per trade
- Net margin on trading: 70-90% depending on volume mix

---

## 10. Demand engine and GTM

The biggest GTM asset Exascale has is warm enterprise relationships into Fortune 500 and frontier-lab procurement organizations. Most startups in this category cold-call into procurement; we don't have to.

### GTM motion

**Direct enterprise sales (primary)**:
- Warm-introduction outreach into named F500 and frontier-lab accounts
- Procurement-led purchasing (RFPs, master agreements)
- 3-9 month sales cycles for first deployment
- Multi-year master agreements once established
- Expansion-led account growth

**Trader recruitment (parallel)**:
- Direct outreach to the ~30-100 institutional trader firms
- Conference presence (commodity trading, AI infrastructure events)
- Letter of intent / paper trading commitments before real-money launch
- Liquidity-provider incentives (rebate tiers, market-data access)

**Supply-side partnerships (parallel)**:
- Direct outreach to datacenter operators with idle capacity
- Anchor agreements with new DC builds
- Capacity wholesale arrangements

**Content + community (mindshare)**:
- Technical content on AI compute economics
- Index publications (creates a news cycle around our reference prices)
- Conference talks and panels

### Why this matters for fundraising

A trading venue's defensibility is liquidity + relationships + market structure. Warm enterprise relationships shorten the path to material booked revenue, which is what converts capital into a real venue.

The roadmap should treat demand-side revenue and trader-pipeline conversion as parallel work streams, both contributing to the fundraising story.

---

## 11. Supply engine and datacenter partnerships

Exascale's owned datacenter is intentionally modest — not hyperscale, no B200/B300 yet. Per Tai: "we need to build up to the hyperscale data centers."

**The supply strategy has three components**:

### Component 1: Exascale-owned capacity (v1 anchor)

- Modest datacenter with H100/H200 capacity
- Provides credible underlying (every credit is backed by real GPUs)
- Sufficient for paper-trading v1 and limited real-money flow
- Expands organically as cash flow allows

### Component 2: Partner datacenters with idle capacity (v1+ scaling)

- Tier 2/3 datacenter operators have meaningful idle capacity
- They struggle to monetize short-tenor and burst capacity through traditional sales motions
- Exascale onboards them as supply partners; they earn from the market
- Initial partnerships negotiated bilaterally; productized onboarding flow in v1.5

**The pitch to DC partners**:
- We bring you market access without you needing a sales motion
- Settlement is automated; payouts map cleanly to your bank account
- You set minimum acceptable prices; we don't undercut you

### Component 3: New DC build anchors (v1.5+ strategic)

- New DC build projects (e.g., 750MW-class projects in development) need anchor demand to finance construction
- A live trading market provides forward price visibility that helps DC operators justify capex
- Exascale can sign forward contracts with new builds as anchor demand
- This creates a structural advantage: new capacity flows into the market by default

### Why this is capital-light

Exascale's capital requirement scales with market structure and team, not with hardware acquisition. Compare:

- **CoreWeave path to $1B revenue**: ~$10B+ in capex
- **Exascale path to $1B revenue**: <$500M in capex (owned capacity for credibility; rest is partner supply)

This is the genuine financial-engineering insight of the credit-market thesis: the market mechanic monetizes underlying we don't have to own.

---

## 12. Fundraising posture

Per Tai's direction: fundraising target is $100M minimum, with $1B+ aspirational once product, demand-side pipeline, and trader letters of intent are sufficient.

### Comparables (for investor context)

| Company | Cash raised | Position |
|---|---|---|
| Together AI | ~$1.5B | Managed inference + clusters |
| Nebius | (public) — substantial | Neocloud, European positioning |
| CoreWeave | ~$28B cash | Hyperscale-adjacent neocloud |
| **Exascale** | **TBD — $100M-$1B target** | **Credit market with owned + partner underlying** |

### Fundraising sequencing

**Pre-seed / seed extension (current)**:
- Demonstrate trading UI/UX with mock data
- Demand-side conversations with named F500 / frontier-lab accounts
- Trader letters of intent from initial outreach
- Datacenter partnership LOIs
- Team and partnership announcements as appropriate

**Series A ($100M-$300M target)**:
- Live trading platform (paper mode customer-facing)
- 3+ F500 / frontier-lab named demand commitments
- 3+ trader firms on paper-trading platform
- 1+ datacenter partnership operational
- Internal real-money trading proven

**Series B ($300M-$1B+ target)**:
- Live real-money trading
- Material booked revenue from F500 / frontier-lab consumption
- Multiple datacenter partnerships
- Daily index publication with external citations
- Initial international expansion (Japan via UBS anchor, plus 1-2 other geographies)

### Why this is fundable at the upper end

Investors funding $100M+ in this category will be evaluating:

1. **Market category**: validated by Larry Fink and others publicly
2. **Defensibility**: liquidity + relationships + market structure
3. **Capital efficiency**: lower than CoreWeave's path
4. **Demand evidence**: F500 / frontier-lab commitments before product GA
5. **Team**: senior engineers + financial-services anchor + warm enterprise relationships
6. **Comparables**: Together AI raised $1.5B without owning a market; we have stronger differentiation

Hitting all six is what unlocks $500M-$1B rounds.

---

## 13. Brand voice and competitive narrative

### Voice attributes

- **Market-aware, not sales-y**: traders respect substance over marketing fluff
- **Direct and specific**: real numbers, concrete claims, honest trade-offs
- **Confident but not arrogant**: we're creating a new category, not pretending to be CME yet
- **Honest about what we're not**: not a hyperscaler, not the cheapest, not yet at hyperscale-DC capacity

### The competitive narrative (per audience)

**For demand-side buyers (F500 + frontier labs)**:
> "Buy compute credits in bulk at locked-in prices. Hedge your AI cost exposure. Use credits across our curated model catalog or for raw GPU rental. Multi-currency, multi-vendor underlying, audit-grade trail."

**For traders**:
> "The first venue for trading AI compute as a commodity. Real underlying, real settlement, real liquidity. Paper-trading customer-facing v1 with real-money testing in parallel — we're proving the market mechanic before opening the firehose."

**For datacenter partners**:
> "Market access without a sales motion. We bring traders and Fortune 500 buyers; you provide capacity. Clean settlement, transparent pricing."

**For investors**:
> "We're creating the commodity market for AI compute. Modest owned underlying for credibility, datacenter partnerships for scale, warm Fortune 500 + frontier-lab relationships for demand, senior engineering team. The CME-equivalent path for the largest commodity in the world without a market today."

**For potential team members**:
> "Production-grade infrastructure + trading-system engineering. Build the first credit market for AI compute. Senior engineering team with frontier-lab pedigree."

### Competitive positioning per competitor

| Competitor | Don't compete on | Position differently |
|---|---|---|
| AWS / Azure / GCP | Compliance breadth, ecosystem | "Not your IT department's incumbent. The market for AI as a commodity." |
| CoreWeave | Capacity scale, NVIDIA-first allocation | "Different category. They sell GPU-hours. We trade credits backed by GPU-hours from many operators." |
| Lambda | H100 pricing, researcher brand | "Lambda for raw compute. Exascale for compute as a tradeable asset." |
| Together | OSS model catalog breadth | "Together inferences for you. Exascale lets you hedge your inference spend." |
| Groq | Tokens/sec speed | "Groq runs fast. Exascale lets you trade what Groq runs." |
| Nebius | EU positioning, pricing | "Different markets. They sell capacity; we run the venue." |

### Claims to never make

- ❌ "The fastest" anything
- ❌ "The cheapest"
- ❌ "The biggest" (we're explicitly not hyperscale-DC scale)
- ❌ "Decentralized" or Web3 framing
- ❌ Direct attacks on competitor brands

---

## 14. Strategic risks

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| 1 | Trading market doesn't form (no liquidity) | High | Critical | Exascale-as-market-maker provides v1 liquidity; recruit external market makers aggressively; demand-side revenue from F500 / frontier-lab credit purchases anchors business even if speculator trading is slow |
| 2 | Regulatory pushback on credit tradability | Medium | High | Conservative legal framing (credits as prepaid services); UBS Japan guidance on JFSA; consider Singapore or Cayman venue |
| 3 | Index gets manipulated or compromised | Medium | High | Public methodology; external audit firm; volume thresholds; outlier exclusion; cryptographic audit trail |
| 4 | Team doesn't materialize at v1 scale | Medium | High | Confirm employment vs. advisor status of all named team members |
| 5 | F500 / frontier-lab sales cycles longer than expected | Medium | Medium | Warm relationships compress cycles vs. cold outreach; demand-side conversion is multi-year game with first deal anchoring subsequent ones |
| 6 | Datacenter partnerships slow to materialize | Medium | Medium | Multiple operators in early conversations; Exascale-owned capacity sufficient for v1 paper trading regardless |
| 7 | Trading layer perceived as gimmick | Medium | High | Index integrity; institutional partnership endorsements; quality of execution proves it's real |
| 8 | AI compute costs collapse (eliminates hedging demand) | Low | Medium | Volatility creates hedging demand regardless of trend direction |
| 9 | NVIDIA allocation crisis for owned capacity | Medium | Medium | Pursue NVIDIA partnership; partner DCs may have NVIDIA relationships; multi-vendor backup |
| 10 | Customer-to-customer trading creates fraud/manipulation issues | Medium | High | Defer customer-to-customer to v2; v1 is Exascale-as-market-maker only; surveillance system before P2P launches |

The top-3 existential risks: liquidity formation, regulatory, manipulation. Everything else is manageable.

---

## 15. Open design questions

**Credit architecture**:
1. AI credit ↔ sub-credit conversion: bidirectional with spread, or one-way?
2. Sub-credit ↔ sub-credit conversion: allowed, or force through index?
3. Index publication frequency: daily, hourly, or both?
4. GPU credit tiers in v1: H100 only, or H100 + H200?

**Trading layer specifics**:
5. Maker rebates: pay makers (negative fee) at top tier, or just 0%?
6. Real-money internal testing scope: which trades, which counterparties?
7. Customer KYC: light for paper, full for real-money?
8. Cash settlement vs. physical: both options or one default?

**Regulatory**:
9. Trading entity jurisdiction: US, Japan, Singapore, Cayman?
10. Securities counsel engagement timing?

**Operational**:
11. Datacenter location and scale?
12. Team headcount confirmed?
13. UBS Japan partnership terms?
14. Existing platform access for cross-reference?

---

## Bottom-line summary

**What Exascale is**: A three-sided commodity market for AI compute. Tradeable credits (AI credits + GPU credits) connecting GPU datacenters, traders, and AI companies. Modest owned underlying for credibility; partner datacenters for scale.

**Why now**: AI compute is the largest commodity in the world without a tradeable market. Hyperscalers can't ship transparent pricing. The category window is open. BlackRock's CEO and other institutional voices are publicly validating the thesis.

**Primary demand ICP**: AI frontier labs ($50M-$10B+ compute spend) and Fortune 500 (Walmart, JPMorgan, Saudi Aramco-tier).

**Liquidity ICP**: ~30-100 institutional trader firms globally.

**Supply ICP**: Datacenter operators with idle capacity + new DC builds needing anchor demand.

**What we ship in v1 (6 months)**: UI/UX-first with light trading simulation in months 1-2 (the demo asset); backend underneath in months 2-4; production hardening in months 4-6. SOC 2 Type I.

**What we never compromise**: Trading layer as headline. Real-time visibility. Sub-5-min onboarding. Index methodology integrity.

**Fundraising**: $100M minimum target, $1B+ aspirational once product + demand + trader pipeline + supply partnerships demonstrate.

**The one-sentence pitch**: *The commodity market for AI compute. Real underlying, real liquidity, real settlement. The Bloomberg of AI compute.*

---

**End of Phase 5 (v2).**
