# Phase 6 (v2) — 1Trade Integrated PRD + System Design
**The engineering kickoff document. Built around the credit-market thesis. UI/UX first.**

> Status: Phase 6 v2 — 2026-05-19. Reflects Tai's latest direction: UI/UX with mock data first, then backend underneath. CLI as primary platform-side interface. Demand-side onboarding for Fortune 500 and frontier labs. Supply-side onboarding for datacenter partners.
> Replaces: Phase 6 v1 which sequenced backend first.
> Owner: Ahmed.

---

## Table of contents

**Part A — Product Requirements**
1. [Product overview](#1-product-overview)
2. [The four immutable commitments](#2-the-four-immutable-commitments)
3. [Personas and user stories](#3-personas-and-user-stories)
4. [Feature specifications](#4-feature-specifications)
5. [Success metrics](#5-success-metrics)
6. [Out of scope for v1](#6-out-of-scope-for-v1)

**Part B — System Design**
7. [Architecture overview](#7-architecture-overview)
8. [Component design](#8-component-design)
9. [Data models](#9-data-models)
10. [API specifications](#10-api-specifications)
11. [Critical flows](#11-critical-flows)
12. [Security, compliance, and surveillance](#12-security-compliance-and-surveillance)

**Part C — Roadmap (UI/UX-first)**
13. [Six-month delivery plan](#13-six-month-delivery-plan)
14. [Team allocation](#14-team-allocation)
15. [Decision gates](#15-decision-gates)

**Part D — Risk Register**
16. [Risk register](#16-risk-register)
17. [Open questions](#17-open-questions)

---

# Part A — Product Requirements

## 1. Product overview

### 1.1 What we're building

1Trade v1 is a **commodity market for AI compute**, with three integrated layers:

1. **Trading layer** (headline product): Order book, market maker, credit ledger, index publication. Shipped UI/UX first as the demo asset.
2. **Inference layer**: Curated catalog of SoTA OSS models with multi-tenant per-GPU deployment.
3. **Compute layer**: GPU rental — primarily CLI-driven, basic web UI sufficient.

All three layers share one credit system (AI credits + GPU credits), one billing, one auth, one dashboard.

### 1.2 Positioning

From Phase 5:
> The commodity market for AI compute. Three-sided market connecting datacenters, traders, and AI companies. Real underlying, real settlement, real liquidity.

### 1.3 Primary use cases

**Use case 1: Trader executes a position on AI credits**
- Trader logs into trading dashboard, sees live index price, candlestick chart, order book
- Submits limit or market order
- Matches against 1Trade market-maker quote
- Position + P&L track in real-time

**Use case 2: Fortune 500 procurement buys AI credits in bulk**
- F500 procurement team logs in via SSO (SAML)
- Reviews available credit products and current pricing
- Initiates volume credit purchase (ACH/wire, multi-currency)
- Credits sit in account; engineering teams within the F500 redeem against actual usage
- CFO sees mark-to-market value and consumption rate

**Use case 3: Frontier lab hedges quarterly compute spend**
- Frontier lab CFO buys forward contracts (v1.5) or holds AI credit balance
- Locks in compute cost for the quarter
- Engineering teams redeem credits across the model catalog or for GPU rental
- Treasury reports mark-to-market value at quarter-end

**Use case 4: AI engineer uses CLI to provision and consume**
- Engineer installs `1trade` CLI
- Authenticates with corporate SSO
- Provisions GPU via `1trade gpu create`
- Runs training/inference workload
- Credits debited per-second; CLI shows real-time spend

**Use case 5: Datacenter partner onboards capacity**
- DC operator engages with 1Trade supply team
- Capacity validated (GPU type, network, SLA capability)
- Onboarded into supply pool — manual process in v1
- DC sees demand flow; receives payouts from credit consumption against their capacity

### 1.4 What this isn't

Explicitly NOT:
- A neutral marketplace aggregating others' supply only
- A traditional neocloud (Lambda / CoreWeave model)
- A traditional managed-inference provider (Together / Groq model)
- A speculation venue without underlying
- A consumer-facing fintech app

We are: **a commodity exchange for AI compute with credible owned underlying plus partner-supplied scale.**

---

## 2. The four immutable commitments

These ship in v1 no matter what:

**1. Trading layer is the headline product.**
Order book, market maker, credit ledger, paper trading customer-facing, real-money internal testing. Even sandbox mode must feel like a real market — depth charts, P&L, settlement history, moving candlesticks.

**2. Index methodology integrity.**
Public methodology document. External audit firm engaged. Constituent weights disclosed. Manipulation surveillance from day one.

**3. Sub-5-minute time-to-first-action.**
Trader signup → first trade in under 5 minutes (paper mode). F500 procurement signup → first credit quote in under 5 minutes. AI engineer signup → first CLI command in under 5 minutes.

**4. Real-time credit visibility.**
Live balances. Visible trade settlement. Real-time inference debit. Audit trail accessible.

---

## 3. Personas and user stories

### 3.1 Persona A — The Institutional Trader (liquidity provider)

**Profile**: Senior trader at prop firm, commodity hedge fund, bank's prop desk, or family office. Mandate to explore AI compute as new asset class.

**User stories**:
```
As a trader, I can:
- Sign up with corporate email and complete light KYC within 24 hours
- Access trading dashboard with live order book, candle charts, market depth
- Submit market and limit orders via web UI or REST API (FIX in v1.5)
- See real-time trade prints, time-and-sales
- Track positions and P&L in real-time
- Pull historical trade data and index history
- Get notifications on price/volume thresholds
- View published index methodology with constituent weights
```

### 3.2 Persona B — The Fortune 500 Procurement Buyer (demand-side, primary)

**Profile**: Procurement director or VP IT at a Fortune 500 enterprise. Buying AI compute on behalf of internal AI teams. Comfortable with master agreements, RFPs, multi-year contracts.

**User stories**:
```
As an F500 procurement buyer, I can:
- Engage through enterprise sales channel for initial onboarding
- Set up corporate account with SAML SSO
- Receive multi-currency invoicing (USD, JPY, EUR)
- Purchase AI credits in bulk via wire / ACH (not just credit card)
- Receive purchase orders, support PO-based purchasing flows
- Assign credit budgets to internal teams (sub-accounts)
- See organization-wide consumption dashboard
- Access audit log for compliance reporting
- Generate quarterly statements for finance team
```

### 3.3 Persona C — The Frontier Lab Engineer (demand-side, primary)

**Profile**: Senior engineer or ML platform lead at a frontier AI lab. $50M-$10B+ annual compute spend. Heavy CLI user. Will not tolerate UX friction.

**User stories**:
```
As a frontier-lab engineer, I can:
- Install `1trade` CLI in <60 seconds
- Authenticate via SSO with corporate identity provider
- Provision GPUs with one command (`1trade gpu create --type h100 --count 32`)
- Submit distributed training jobs (`1trade train --image ... --gpus 256`)
- Use OpenAI-compatible SDK with one base-URL change
- Pull metrics via Prometheus-compatible endpoint
- See real-time spend via `1trade billing today`
- Set budget alerts and auto-stop policies
- Migrate workloads from hyperscaler with minimal code changes
```

### 3.4 Persona D — The AI Company Hedger (demand-side, secondary)

**Profile**: VP Eng or CFO at Series B/C AI startup. $200K-$5M annual compute spend. Wants predictable budget for board reporting.

**User stories**:
```
As a hedger, I can:
- Buy AI credits in bulk with USD or JPY
- Redeem credits for any sub-credit at published rate
- Set up auto-redeem rules
- See projected runway based on credit balance and consumption rate
- Sell unused credits back at market if usage drops
- Generate reports for CFO showing mark-to-market value
```

### 3.5 Persona E — The Datacenter Partner (supply-side)

**Profile**: Operations lead at a Tier 2/3 datacenter operator with idle or underutilized GPU capacity.

**User stories**:
```
As a DC partner, I can:
- Engage with 1Trade supply team for capacity onboarding
- Have my capacity verified (GPU type, network specs, SLA)
- Set minimum acceptable prices per GPU tier
- See real-time utilization of my contributed capacity
- Receive payouts via wire transfer mapped to credit consumption
- Access reporting dashboard for capacity sold and revenue earned
- Adjust capacity allocation up/down as my own demand changes
```

### 3.6 Persona F — Internal Market Maker (system, not user)

**Profile**: 1Trade's automated quoting system providing v1 liquidity.

**System requirements**:
```
The market-maker system must:
- Continuously post bid and ask quotes for all tradeable products
- Adjust spread based on inventory imbalance
- Adjust quotes based on real-time consumption signals
- Respect risk limits (max position, max daily loss)
- Withdraw quotes during volatility events
- Log all quote updates for audit
- Maintain target spread (1% in v1)
```

---

## 4. Feature specifications

### 4.1 Trading interface (the headline UI)

**Trading dashboard** (the demo asset, shipped first):
- Real-time index price ticker
- Candlestick chart with moving data (multiple intervals: 1m, 5m, 15m, 1h, 1d)
- Order book depth visualization
- Time-and-sales feed
- Order entry panel (market, limit, IOC, FOK)
- Position panel with mark-to-market P&L
- Trade history
- Account balance widget

**Mock data mode (months 1-2)**:
- Plausible market behavior (candles, depth, trade prints) without real matching engine
- Used as demo for F500 / frontier-lab prospects, traders, and investors
- Becomes paper-trading mode once real matching engine ships

**Order types (v1)**:
- Market order (immediate fill at best available)
- Limit order (sits on book)
- IOC (immediate or cancel)
- FOK (fill or kill)

**Order types (v1.5+)**:
- Stop orders, stop-limit
- Forward contracts (cash-settled, 1mo/3mo/6mo)

### 4.2 Credit ledger

**AI credits** (unified):
- Customers purchase with cash (Stripe, ACH, wire)
- Tradeable on AI index spot market
- Redeemable into any sub-credit at published rate

**Sub-credits**: text, speech, image, video, niche
- Each tradeable on its own spot market
- Consumed by API usage
- Convertible back to AI credits (with house spread)

**GPU credits**: per-tier
- Tradeable on per-tier spot markets
- Consumed by GPU rental
- Forward contracts for tenor-based purchasing (v1.5)

**Balance API**:
- Real-time balance per credit type
- Audit log of all movements
- Per-resource attribution

### 4.3 Index publication

**Daily print** (16:00 UTC):
- AI index spot price ($ per credit)
- Constituent breakdown
- Methodology version
- Audit hash chain

**Methodology** (publicly documented):
- Volume-weighted across trades + capacity utilization
- 95% trimmed mean (outlier exclusion)
- Volume floor for valid print
- Updated quarterly

**Web display**:
- Index ticker on landing page (live, even pre-launch)
- Historical chart
- Constituent transparency

### 4.4 Curated inference catalog

**Categories at launch**:

| Category | Models (top 3-5) |
|---|---|
| Text — general | Llama 3.3 70B, DeepSeek-R1 distill 70B, Qwen 2.5 72B, Mixtral 8x22B |
| Text — small/fast | Llama 3.1 8B, Phi-4, Qwen 2.5 7B |
| Text — code | Qwen 2.5 Coder 32B, DeepSeek Coder V2.5 |
| Speech — STT | Whisper Large v3, Distil-Whisper |
| Speech — TTS | XTTS v2, F5-TTS |
| Image — generation | FLUX.1-dev, FLUX.1-schnell, SDXL |
| Image — understanding | Llama 3.2 Vision 90B, Qwen2-VL |
| Video — generation | Hunyuan Video, CogVideoX (gated to v1.5 if capacity tight) |
| Embeddings | bge-m3, jina-embeddings-v3 |
| Niche — reranker | bge-reranker-v2 |

**Multi-tenant per-GPU deployment**:
- Static co-location for top-3 most-popular models per GPU
- Hot-swap for less-popular models (LRU eviction)
- MIG partitioning for video gen where needed

**Catalog refresh**:
- Quarterly review
- Promote new SoTA models within 2-4 weeks of release
- Deprecate underperforming models with 30-day notice

### 4.5 Compute platform (CLI-primary)

**CLI tool** (`1trade`):
- Installable via `brew`, `apt`, `pip`, or direct binary download
- SSO-friendly authentication (browser-based OAuth flow)
- Resource provisioning (`1trade gpu create`, `gpu list`, `gpu stop`)
- Training submission (`1trade train`)
- Inference (`1trade infer`)
- Billing (`1trade billing today`, `billing alerts set`)
- Credits (`1trade credits balance`, `credits purchase`)
- Trading (`1trade trade quote`, `trade buy`, `trade orders`)

**Basic web UI for compute**:
- View instances, jobs, billing
- Provisioning available via web for non-CLI users
- Not the primary interface — secondary to CLI

**On-demand GPU instances**:
- H100 80GB SXM5, H200
- Self-serve up to 32 GPUs
- Pre-built 1Trade ML Stack images
- Time-to-running: <90 seconds typical

**Reserved capacity**:
- 1-month, 6-month, 1-year terms
- Discounts: 17%, 27%, 33%
- Reservations represented as GPU credits

**Clusters**:
- Multi-node, InfiniBand fabric
- Gang-scheduled (Kueue + Volcano)
- Slurm or K8s-native submission
- Sales-engaged for 32+ GPU clusters (the one exception to self-serve)

### 4.6 Demand-side onboarding

**Self-serve path** (smaller customers):
- Email signup
- Light KYC for trader paper trading; full KYC for real-money
- Credit card or ACH for credit purchase
- Multi-currency: USD + JPY at launch

**Enterprise path** (F500, frontier labs):
- Engagement through enterprise sales channel
- SAML SSO setup with customer's IdP (Okta, Entra ID, Google Workspace)
- Master Services Agreement
- Purchase Order-based purchasing (wire / ACH)
- Multi-currency invoicing
- Sub-account structure for internal teams
- Custom contract terms (volume commitments, dedicated capacity)

### 4.7 Supply-side onboarding

**v1: Manual process**
- DC partner engagement through 1Trade supply team
- Capacity verification (GPU type, NIC topology, SLA capability)
- Bilateral commercial agreement (pricing floor, payout terms)
- Manual integration into supply pool (1Trade ops team adds capacity to scheduling)
- Payout via wire transfer, monthly

**v1.5: Productized onboarding**
- Self-serve capacity onboarding portal
- Automated verification
- Real-time utilization dashboard for DC partner
- Automated payout flow

### 4.8 Auth, KYC, RBAC

**Standard signup**:
- Email + password or OAuth (Google, GitHub, Microsoft)
- API keys, scoped permissions
- 2FA optional

**Enterprise SSO**:
- SAML 2.0
- SCIM provisioning
- IP allowlisting
- Audit log for compliance

**Trader KYC**:
- Light KYC for paper trading (email, name, jurisdiction)
- Full KYC for real-money (gov ID, accreditation where required, AML)
- Institutional onboarding (corporate documents, beneficial ownership)

**RBAC**:
- Roles: admin, billing, trader, engineer, viewer
- Audit log of admin actions
- API key scopes

### 4.9 Billing and credit purchase

**Cash → credits**:
- Stripe for credit cards
- ACH / wire for $10K+ purchases
- Multi-currency: USD + JPY at launch; EUR, GBP in v1.5

**Real-time visibility**:
- Current balance per credit type
- Month-to-date consumption
- Projected month-end
- Cost alerts at 50/80/100% of budget

**Auto-stop policies**:
- Idle resource auto-stop
- Budget-hit auto-stop
- Customer-configurable

### 4.10 Surveillance and risk controls

**Trade surveillance**:
- Wash trade detection
- Spoofing detection
- Layering detection
- Marking the close
- Cross-product manipulation
- Excessive cancellation rate

**Position limits**:
- Per-customer max position per product
- Per-customer max gross exposure
- Configurable by tier

**Market-maker risk system**:
- Max inventory per product
- Daily P&L stop-loss
- Volatility-triggered spread widening
- Manual override

---

## 5. Success metrics

### 5.1 Trading metrics

| Metric | Month 3 | Month 6 |
|---|---|---|
| Active trader accounts | 15 | 75 |
| Trades per day (paper) | 50 | 500 |
| Notional volume per day (paper) | $50K | $1M |
| Index price accuracy vs. settlement | ±5% | ±2% |
| Order book depth (best bid+ask within 1%) | $5K | $50K |
| Internal real-money test volume | $1K/day | $100K/day |

### 5.2 Demand-side metrics

| Metric | Month 3 | Month 6 |
|---|---|---|
| F500 / frontier-lab engaged accounts | 3-5 | 10-15 |
| F500 / frontier-lab signed deals | 0-1 | 2-5 |
| Total paying organizations | 30 | 150 |
| MRR (consumption + bulk credit purchases) | $50K | $300K |
| Bulk credit purchase volume (cash inflow) | $50K | $1M+ |
| Inference requests per day | 100K | 2M |
| Compute hours per day | 200 | 2,000 |

### 5.3 Supply-side metrics

| Metric | Month 3 | Month 6 |
|---|---|---|
| DC partner conversations | 5-10 | 15-25 |
| DC partners onboarded | 0 | 1-2 |
| Partner capacity contributed (GPU-hours/day) | 0 | 200-500 |

### 5.4 Reliability

| Metric | Target |
|---|---|
| Trading engine uptime | 99.95% |
| Order matching latency P99 | <10ms |
| Inference API uptime | 99.95% |
| Compute uptime | 99.9% |
| Index publication reliability | 100% (never miss daily print) |

### 5.5 Strategic

| Metric | Month 6 |
|---|---|
| F500 / frontier-lab signed accounts | 2-5 |
| Trader firms on paper trading | 5-10 |
| DC partnerships operational | 1-2 |
| SOC 2 Type I report | Achieved |

---

## 6. Out of scope for v1

**Trading**:
- Customer-to-customer matching (1Trade-internal market maker only)
- Margin trading
- Options or volatility products
- Forward contracts (v1.5)
- FIX protocol (v1.5; REST API only for v1)

**Compute**:
- B200/B300 (v2)
- Spot/preemptible (v1.5)
- Multi-region (v1 single region with multi-AZ)
- Custom silicon

**Inference**:
- Video generation (v1.5 unless capacity allows)
- Custom model deployment / BYO weights (v2)
- Fine-tuning service (v2)

**Platform**:
- Mobile app
- Office/home GPU supply
- Productized DC partner self-serve onboarding (manual in v1)

**Compliance**:
- HIPAA-eligible deployment
- FedRAMP
- SOC 2 Type II (v1.5)
- Sector-specific (JFSA, etc.)

---

# Part B — System Design

## 7. Architecture overview

```
┌───────────────────────────────────────────────────────────────┐
│                         CUSTOMERS                              │
│  Traders · F500 Procurement · Frontier Labs · DC Partners      │
│  Internal Market Maker · AI Engineers                          │
└───────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌───────────────────────────────────────────────────────────────┐
│  EDGE (Cloudflare) → API Gateway (Kong)                        │
└───────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Trading API  │ │ Inference    │ │ Compute API  │ │ Supply API   │
│ (priority)   │ │ API          │ │ (CLI-first)  │ │ (partner     │
│              │ │              │ │              │ │  onboarding) │
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │                │
       ▼                ▼                ▼                ▼
┌──────────────────────────────────────────────────────────────────┐
│                   CORE SERVICES                                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
│  │ Matching │ │  Credit  │ │  Index   │ │  Market  │ │ Sur-   │ │
│  │  Engine  │ │  Ledger  │ │ Service  │ │  Maker   │ │ veil-  │ │
│  │   (Go)   │ │   (Go)   │ │   (Go)   │ │   (Go)   │ │ lance  │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬───┘ │
│       │            │            │            │            │      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Auth · Account · Billing · Quota · Audit · Notification    │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
       │            │            │            │            │
       ▼            ▼            ▼            ▼            ▼
┌──────────────────────────────────────────────────────────────────┐
│                  DATA PLANE                                       │
│  PostgreSQL · TimescaleDB · Redis · NATS · Object Storage         │
└──────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────────┐
                    │  KUBERNETES CLUSTER       │
                    │  (Kueue + Volcano +       │
                    │   NVIDIA GPU Operator)    │
                    │                            │
                    │  ┌─────────────────────┐  │
                    │  │ Inference Pods      │  │
                    │  │ (vLLM, multi-model) │  │
                    │  └─────────────────────┘  │
                    │                            │
                    │  ┌─────────────────────┐  │
                    │  │ Customer Workloads  │  │
                    │  │ (GPU rental, etc.)  │  │
                    │  └─────────────────────┘  │
                    └──────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
            ┌─────────────┐     ┌─────────────┐
            │ 1Trade-   │     │ Partner DC  │
            │ owned DC    │     │ capacity    │
            │ (v1 anchor) │     │ (v1+ scale) │
            └─────────────┘     └─────────────┘

         ┌───────────────────────────────────────────┐
         │ CROSS-CUTTING                              │
         │ - Observability (Grafana stack)            │
         │ - Compliance (Vanta)                       │
         │ - Secrets (Vault)                          │
         └───────────────────────────────────────────┘
```

### Design principles

**Trading layer is the spine.** Every other service integrates with the credit ledger.

**Index integrity is sacred.** Index Service is single source of truth for AI index price.

**Real-money testing isolated from paper trading.** Strict account separation.

**Surveillance from day one.** Even paper trading is monitored.

**Mock data mode for trading UI.** First 2 months use plausible mock data; real backend swaps in seamlessly.

**Supply abstraction.** Compute control plane doesn't care whether GPU is 1Trade-owned or partner-supplied. Single scheduling fabric across both.

---

## 8. Component design

### 8.1 Matching Engine

**Responsibility**: Match buy and sell orders for all tradeable products.

**Architecture**:
```
Order submission → Order validation → Order book (Redis sorted sets)
                                              ↓
                                       Matching algorithm
                                       (price-time priority)
                                              ↓
                                       Trade execution
                                              ↓
                                  Credit ledger atomic update
                                              ↓
                                       Audit log entry
                                              ↓
                                  Surveillance evaluation
                                              ↓
                                  Notification dispatch
```

**Key design decisions**:
- Single-threaded matching per product (v1)
- Order book in Redis sorted sets
- Persistent snapshot to PostgreSQL every 5 minutes
- Event-sourced; replayable from event log

**Performance targets**:
- Order acceptance P99 < 10ms
- Match execution P99 < 5ms
- End-to-end trade confirmation P99 < 100ms

**Mock data mode (months 1-2)**:
- The matching engine isn't running yet
- Trading API returns simulated quotes, fills, and order book state
- Mock data follows realistic statistical patterns (Brownian motion with mean reversion, plausible volume distributions)
- Customer-visible difference: zero — they see a working trading UI from day one
- Backend swap: when real matching engine is ready, the API switches over without UI changes

### 8.2 Credit Ledger

**Responsibility**: Maintain per-tenant credit balances with cryptographic auditability.

**Schema** (PostgreSQL):

```sql
CREATE TABLE credit_balances (
    balance_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    credit_type TEXT NOT NULL, -- 'ai_index' | 'text' | 'speech' | 'image' | 'video' | 'gpu_h100' | 'gpu_h200'
    balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    locked_amount NUMERIC(20, 6) NOT NULL DEFAULT 0,
    UNIQUE (tenant_id, credit_type)
);

CREATE TABLE credit_transactions (
    tx_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    credit_type TEXT NOT NULL,
    operation TEXT NOT NULL, -- 'purchase' | 'consumption' | 'trade' | 'conversion' | 'refund'
    amount NUMERIC(20, 6) NOT NULL,
    reference_id TEXT,
    balance_before NUMERIC(20, 6) NOT NULL,
    balance_after NUMERIC(20, 6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    chain_hash TEXT NOT NULL
);

CREATE INDEX idx_credit_tx_tenant_time ON credit_transactions (tenant_id, created_at DESC);
CREATE INDEX idx_credit_tx_ref ON credit_transactions (reference_id);
```

**Critical invariants**:
- Append-only on `credit_transactions`
- Atomicity between balance update and transaction insert
- Hash chain: `chain_hash = hash(prev_chain_hash || canonical_json(row))`
- Reconciliation: replaying transactions reproduces current balance

### 8.3 Index Service

**Responsibility**: Compute and publish the AI credit index. Maintain audit-grade integrity.

**Methodology (v1)**:
- Daily print at 16:00 UTC
- 95% trimmed mean across constituent observations
- Volume floor for valid print
- Manipulation resistance via cross-validation

**Publication flow**:
1. 15:30 UTC: gather inputs from last 24h
2. 15:45 UTC: filter outliers, apply methodology
3. 15:55 UTC: compute final value
4. 16:00 UTC: extend audit hash chain, publish
5. 16:00:01 UTC: notify subscribers
6. 16:00:05 UTC: publish constituent transparency

**Mock data mode (months 1-2)**:
- Synthetic index price following plausible market dynamics
- Real methodology spec published from day one (so investors can review)
- Real index switch-over happens with backend cutover

### 8.4 Market Maker Service

**Responsibility**: Provide liquidity by quoting both sides. v1: 1Trade-internal automated quoting.

**Pricing logic**:
- Base spread: 1% bid-ask (matches taker fee tier)
- Skew based on inventory imbalance
- Risk-off: widen or withdraw quotes during volatility
- Inventory limits

**Risk controls**:
- Max position per product
- Daily P&L stop-loss
- Automatic quote withdrawal during anomalous order book
- Manual override

### 8.5 Surveillance Service

**Detection patterns**:

| Pattern | Description | Action |
|---|---|---|
| Wash trade | Same beneficial owner on both sides | Flag; review |
| Spoofing | Large orders frequently canceled | Alert; rate-limit |
| Layering | Multiple staggered orders creating false depth | Alert; investigate |
| Marking the close | Concentrated activity at index calc window | Exclude from index |
| Cross-product manipulation | Spot manipulation affecting forwards | Cross-product surveillance |
| Excessive cancellation | Order-to-trade ratio above threshold | Rate-limit |

### 8.6 Inference Gateway

**Responsibility**: Route inference requests to vLLM backends. Authenticate, debit credits, log usage.

```
[Customer request] → [Gateway]
                        ↓
               [Auth + quota check]
                        ↓
             [Credit balance check]
                        ↓
               [Route to vLLM pool]
                        ↓
             [Stream response back]
                        ↓
        [Emit usage event → debit credits]
                        ↓
                [Update real-time bill]
```

**Multi-model-per-GPU**:
- Always-resident for top models
- Hot-swappable for less-popular models
- Model routing accounts for cold-start probability

### 8.7 Compute Control Plane

**Responsibility**: Manage GPU lifecycle across 1Trade-owned and partner DC capacity.

**Key feature**: supply-source abstraction. The control plane treats 1Trade-owned GPUs and partner-supplied GPUs as a single pool, scheduled by Kueue + Volcano with topology and locality awareness.

**Partner capacity integration**:
- Partner DCs run a small 1Trade agent that registers capacity with control plane
- Capacity reports include: GPU type, count, NIC topology, current utilization, SLA status
- Scheduler treats partner capacity equivalently for placement decisions
- Billing pipeline tracks which capacity served which request (for partner payout)

### 8.8 Supply Service

**Responsibility**: Onboard and manage datacenter partners.

**v1 (manual flow)**:
- DC partner record: ops contact, location, capacity, SLA, pricing floor
- Capacity validation workflow (ops team verifies before activation)
- Payout calculation: aggregates consumption against partner capacity, computes payout monthly
- Wire transfer to partner per agreed terms

**v1.5 (productized)**:
- Self-serve partner portal
- Automated capacity verification
- Real-time partner dashboard

---

## 9. Data models

### 9.1 Trading data models

```sql
CREATE TABLE products (
    product_id TEXT PRIMARY KEY,
    product_type TEXT NOT NULL, -- 'spot' | 'forward'
    underlying_asset TEXT,
    delivery_date DATE,
    contract_size NUMERIC(20, 6),
    tick_size NUMERIC(20, 6) NOT NULL,
    tradeable BOOLEAN NOT NULL DEFAULT TRUE,
    is_real_money_enabled BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE orders (
    order_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    product_id TEXT REFERENCES products(product_id),
    side TEXT NOT NULL,
    order_type TEXT NOT NULL,
    quantity NUMERIC(20, 6) NOT NULL,
    limit_price NUMERIC(20, 6),
    time_in_force TEXT,
    state TEXT NOT NULL,
    filled_quantity NUMERIC(20, 6) NOT NULL DEFAULT 0,
    avg_fill_price NUMERIC(20, 6),
    is_paper BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trades (
    trade_id UUID PRIMARY KEY,
    product_id TEXT REFERENCES products(product_id),
    buy_order_id UUID REFERENCES orders(order_id),
    sell_order_id UUID REFERENCES orders(order_id),
    quantity NUMERIC(20, 6) NOT NULL,
    price NUMERIC(20, 6) NOT NULL,
    buyer_tenant_id UUID NOT NULL,
    seller_tenant_id UUID NOT NULL,
    buyer_fee NUMERIC(20, 6) NOT NULL,
    seller_fee NUMERIC(20, 6) NOT NULL,
    is_paper BOOLEAN NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    chain_hash TEXT NOT NULL
);

CREATE INDEX idx_trades_product_time ON trades (product_id, executed_at DESC);

CREATE TABLE positions (
    position_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    product_id TEXT REFERENCES products(product_id),
    net_quantity NUMERIC(20, 6) NOT NULL DEFAULT 0,
    avg_entry_price NUMERIC(20, 6),
    unrealized_pnl NUMERIC(20, 6) DEFAULT 0,
    realized_pnl_total NUMERIC(20, 6) DEFAULT 0,
    UNIQUE (tenant_id, product_id)
);

CREATE TABLE index_prints (
    print_id UUID PRIMARY KEY,
    print_date DATE NOT NULL,
    print_time TIMESTAMPTZ NOT NULL,
    index_value NUMERIC(20, 6) NOT NULL,
    methodology_version TEXT NOT NULL,
    constituent_data JSONB NOT NULL,
    observation_count INT NOT NULL,
    prev_chain_hash TEXT NOT NULL,
    chain_hash TEXT NOT NULL,
    UNIQUE (print_date)
);
```

### 9.2 Supply data models

```sql
CREATE TABLE dc_partners (
    partner_id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    primary_contact_email TEXT NOT NULL,
    region TEXT NOT NULL,
    state TEXT NOT NULL, -- 'pending' | 'verified' | 'active' | 'suspended'
    sla_tier TEXT NOT NULL,
    payout_terms JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE partner_capacity (
    capacity_id UUID PRIMARY KEY,
    partner_id UUID REFERENCES dc_partners(partner_id),
    gpu_type TEXT NOT NULL,
    gpu_count INT NOT NULL,
    pricing_floor NUMERIC(10, 4) NOT NULL,
    state TEXT NOT NULL, -- 'available' | 'allocated' | 'maintenance'
    last_seen_at TIMESTAMPTZ
);

CREATE TABLE partner_payouts (
    payout_id UUID PRIMARY KEY,
    partner_id UUID REFERENCES dc_partners(partner_id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    gpu_hours_consumed NUMERIC(20, 4) NOT NULL,
    gross_revenue NUMERIC(20, 4) NOT NULL,
    1trade_fee NUMERIC(20, 4) NOT NULL,
    partner_payout NUMERIC(20, 4) NOT NULL,
    state TEXT NOT NULL, -- 'pending' | 'wired' | 'settled'
    wired_at TIMESTAMPTZ
);
```

### 9.3 Surveillance

```sql
CREATE TABLE surveillance_alerts (
    alert_id UUID PRIMARY KEY,
    tenant_id UUID,
    pattern_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    evidence JSONB NOT NULL,
    state TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    notes TEXT
);
```

---

## 10. API specifications

### 10.1 Trading API

```
GET  /v1/trading/products
GET  /v1/trading/products/{id}/quote
GET  /v1/trading/products/{id}/orderbook
GET  /v1/trading/products/{id}/trades
GET  /v1/trading/products/{id}/candles?interval=1m&from=&to=

POST /v1/trading/orders
GET  /v1/trading/orders?state=open
GET  /v1/trading/orders/{id}
POST /v1/trading/orders/{id}/cancel
DELETE /v1/trading/orders  # cancel all

GET  /v1/trading/positions
GET  /v1/trading/trades?from=&to=
GET  /v1/trading/pnl?from=&to=

GET  /v1/trading/account/fees
GET  /v1/trading/account/limits
```

### 10.2 Index API

```
GET  /v1/index/current
GET  /v1/index/history?from=&to=&interval=daily
GET  /v1/index/constituents/{date}
GET  /v1/index/methodology/{version}

WS   /v1/ws/index/stream
WS   /v1/ws/products/{id}/trades
WS   /v1/ws/products/{id}/orderbook
```

### 10.3 Credit API

```
GET  /v1/credits/balances
GET  /v1/credits/transactions?from=&to=&type=
POST /v1/credits/purchase
POST /v1/credits/convert
GET  /v1/credits/conversion-rates
```

### 10.4 Compute API (CLI-driven, web available)

```
POST /v1/compute/instances
GET  /v1/compute/instances
GET  /v1/compute/instances/{id}
POST /v1/compute/instances/{id}/stop
POST /v1/compute/instances/{id}/start
DELETE /v1/compute/instances/{id}

GET  /v1/compute/types
GET  /v1/compute/quota
```

### 10.5 Supply API (internal + partner-facing)

```
GET  /v1/supply/partners
POST /v1/supply/partners/{id}/capacity
GET  /v1/supply/partners/{id}/utilization
GET  /v1/supply/partners/{id}/payouts
```

---

## 11. Critical flows

### 11.1 Flow: Trader signup to first paper trade

```
1. Trader visits 1trade.com
2. Clicks "Sign Up" → corporate email
3. Email verified
4. Light KYC: name, jurisdiction, accreditation self-attestation
5. Account created with paper-trading enabled
6. $10,000 paper credits allocated automatically
7. Quick-start: copy this trade order
8. Trader submits first paper buy order
9. Engine matches against market maker
10. Trade executes; position recorded; P&L visible
```

Target: <5 minutes start to finish.

### 11.2 Flow: F500 procurement onboarding

```
1. Engagement initiated through enterprise sales
2. NDA + initial scoping call
3. Technical evaluation (4-8 weeks typical)
4. Master Services Agreement negotiation
5. SAML SSO configured with customer IdP
6. Sub-account structure set up for customer's internal teams
7. First credit purchase via wire (e.g., $500K)
8. Credits available; internal teams begin consumption
9. Monthly invoicing; quarterly business reviews
```

Target: 3-9 month sales cycle for first deal; faster for subsequent.

### 11.3 Flow: AI credit redemption to text credit

```
1. Customer calls POST /v1/credits/convert
   { "from": "ai_index", "to": "text", "amount": 100 }
2. Credit ledger reads current conversion rate
3. Atomic transaction: deduct AI credits, add text credits
4. Audit chain extended
5. Updated balances returned
6. Customer consumes text credits via inference API
```

### 11.4 Flow: Inference call with credit debit

```
1. Customer hits POST /v1/chat/completions with Llama 70B
2. Gateway authenticates, checks quota
3. Gateway checks text credit balance
4. Route to vLLM pool
5. Response streams back
6. Usage event emitted
7. Text credit balance debited
8. Audit chain extended
9. Real-time dashboard reflects update
```

### 11.5 Flow: Daily index publication

```
1. 15:30 UTC — collection begins
2. 15:45 UTC — filter outliers, apply methodology
3. 15:55 UTC — final value computed
4. 16:00 UTC — published; audit chain extended
5. 16:00:01 UTC — notify subscribers via WS
6. 16:00:05 UTC — publish constituent transparency
```

### 11.6 Flow: Partner DC capacity onboarding (v1 manual)

```
1. Sales conversation with partner DC operator
2. Mutual NDA + technical scoping
3. Capacity assessment (GPU type, count, NIC topology, SLA)
4. Commercial agreement (pricing floor, payout %, term)
5. 1Trade agent deployed at partner DC
6. Capacity registered with compute control plane
7. Soft launch — partner capacity receives small allocation
8. Monitor for 2-4 weeks
9. Full activation — partner capacity in regular scheduling pool
10. Monthly payouts begin
```

Target: 4-12 week onboarding for first capacity-online.

---

## 12. Security, compliance, and surveillance

### 12.1 Trading-specific security

**Anti-manipulation**:
- Surveillance service (Section 8.5)
- Position limits per customer
- Order-to-trade ratio limits
- Suspension authority

**Customer fund protection**:
- Segregated accounts for real-money trading (v1.5)
- Audit trail for every credit movement
- Daily reconciliation

**Insider risk**:
- Internal accounts can't trade with customer paper-trading accounts
- All internal trading logged separately
- Quarterly internal audit

### 12.2 Regulatory positioning

**Conservative framing**:
- Credits are "prepaid service units"
- Trading is "secondary market for unused prepaid services"
- Avoid: "futures," "speculation," "investment return"

**Jurisdiction strategy**:
- US: state-by-state evaluation; possible MSB registration
- Japan: consult with UBS Japan COO
- Consider Singapore (MAS-friendly) or Cayman
- Securities counsel before customer-facing real-money

**KYC requirements**:
- Paper trading: light KYC
- Real-money: full KYC, AML
- Institutional: corporate documents, beneficial ownership

### 12.3 SOC 2 path

- Type I in v1 (months 0-6)
- Type II in v1.5
- Vanta or Drata for automation

---

# Part C — Roadmap (UI/UX-first)

## 13. Six-month delivery plan

The biggest change from Phase 6 v1: UI/UX with mock data ships in months 1-2 as the demo asset; real backend ships underneath in months 2-4.

### Month 1: Trading UI/UX + Foundations

**Engineering — frontend priority**:
- Brand and design system (consistent with old-designer brand book once received)
- Trading dashboard UI (order book, depth chart, candle chart with moving mock data)
- Credit balance dashboard UI
- Index publication page UI (with mock current/historical values)
- Account management UI
- Marketing site + landing page
- Mock data service (plausible market behavior)

**Engineering — backend foundation**:
- Auth, account, org services (real, not mock)
- API gateway (Kong)
- PostgreSQL + Redis + NATS infrastructure
- CI/CD
- Observability baseline (Grafana stack)

**Business**:
- Lock team composition
- Engage securities counsel
- Vanta engagement for SOC 2
- Demand-side warm outreach to F500 / frontier-lab contacts
- DC partner conversations begin

**End-of-month-1 demo asset**: A working trading dashboard with realistic mock data. Can be shared with F500 / frontier-lab prospects, traders, and investors to provoke conversation.

### Month 2: Real Backend Begins + Inference Foundations

**Engineering — frontend**:
- Compute web UI (basic — CLI is primary)
- DC partner ops dashboard (internal use)
- Sub-account structure for enterprise accounts

**Engineering — backend**:
- **Matching engine v1** (real, persistent, recoverable — replaces mock data for trading)
- Credit ledger v1 (atomic balance updates)
- Index service v0 (basic methodology, daily print starts)
- Compute control plane v1 (K8s + Kueue + Volcano + NVIDIA Operator)
- vLLM deployment with first 3 models (Llama 70B, Llama 8B, Whisper)
- CLI tool v0 (`1trade` — basic commands)
- Trading API endpoints

**Business**:
- Customer discovery interviews continuing
- First UBS Japan workload identified
- Trading entity jurisdiction decision
- First DC partner agreement signed (target)
- F500 / frontier-lab early evaluations underway

### Month 3: Trading + Inference Integration

**Engineering**:
- Market maker service v1 (1Trade-internal automated quoting)
- Multi-model-per-GPU deployment (top 5 models)
- Credit conversion service (AI → sub-credit)
- Inference API with credit debit logic
- CLI v1 (full feature set for compute, inference, billing, trading)
- Web UI: trading interface backed by real engine (transition from mock)
- Stripe Connect integration for credit purchases (smaller customers)
- ACH / wire flow for large purchases (enterprise)

**Business**:
- Public alpha — invite-only
- First 10 institutional trader applicants on paper trading
- First F500 / frontier-lab signed deal (target)
- Documentation site

### Month 4: Real-Money Internal Testing + Enterprise

**Engineering**:
- Real-money trading capability (internal accounts only)
- Surveillance service v1 (6 rule-based patterns)
- Multi-currency support (USD + JPY)
- Cost alerts, auto-stop policies
- SAML SSO for enterprise accounts
- Sub-account RBAC for enterprise org structures
- Audit log export

**Business**:
- SOC 2 Type I audit kickoff
- First paying enterprise customers (deals signed in M2-3)
- Internal real-money testing begins
- Index methodology audit firm engaged
- Additional DC partners signed

### Month 5: Catalog Expansion + Market Liquidity

**Engineering**:
- Full model catalog (top 3-5 per category)
- Training orchestrator service
- GPU credit trading (separate from AI credits)
- Forward contract groundwork (data models)
- Performance optimization
- Partner capacity scheduling fully integrated

**Business**:
- Public beta launch
- First 25 institutional traders on paper trading
- Marketing campaign (HN, AI Twitter, trader-focused)
- F500 / frontier-lab pipeline maturing
- Series A conversations begin

### Month 6: GA Launch

**Engineering**:
- Index methodology audit complete
- Production-grade reliability hardening
- API documentation finalized
- SDK releases (Python, JS, Go)
- Customer audit log feature

**Business**:
- GA launch
- SOC 2 Type I report achieved
- Internal real-money testing concludes
- First $300K MRR (consumption + bulk credit purchases)
- 5+ reference customers (mix of F500, frontier labs, traders)
- 1-2 DC partnerships operational
- Series A close (target)

---

## 14. Team allocation

| Role | Month 1 | Month 2 | Month 3 | Month 4 | Month 5 | Month 6 |
|---|---|---|---|---|---|---|
| **Frontend (priority)** | Trading UI + mock data | Backend integration | Trading polish + enterprise UI | Surveillance UI | Catalog UI | Polish |
| K8s/Infra | Foundation | Compute core | Compute scaling | Reliability | Networking | Hardening |
| ML Platform | Stack prep | vLLM + first models | Multi-model deployment | Catalog expansion | All categories | Polish |
| **Trading Systems** (critical hire) | Spec + architecture | Engine v1 | Market maker | Real-money capable | Surveillance | Production-ready |
| Backend (×2) | Auth/Accounts | Credit ledger | Stripe + enterprise | Multi-currency + audit | Partner integrations | RBAC + final |
| SRE | Observability | K8s ops | Trading observability | Customer monitoring | Incident response | Hardening |
| **Enterprise sales** (warm intros leveraged) | Outreach | Discovery | First deals | Procurement cycles | Pipeline | Close |
| DC partnerships | Outreach | First conversations | First agreement | Onboarding | Additional partners | Operational |
| Founder (Tai) | Hiring + partnerships | Customer dev | Investor conversations | Investor + customers | Series A | Launch |
| Ahmed (lead) | UI/UX architecture | Backend integration | Critical path | Critical path | Critical path | Final |

**Critical hires (priority order)**:
1. Senior frontend engineer (trading UI is the headline product; ship it first)
2. Trading systems engineer (the matching engine + market maker)
3. K8s / GPU infrastructure engineer
4. ML platform engineer
5. Enterprise sales lead (warm intro engine)
6. DC partnership lead

---

## 15. Decision gates

### Gate 1: End of Month 1 — Demo Asset Check

**Pass criteria**:
- Trading dashboard with mock data is demo-able to enterprise prospects
- 6+ senior engineers committed
- Foundation infrastructure stood up
- Datacenter capacity confirmed (modest is fine)

**If fail**: Reduce v1 scope substantially. Consider extending mock-data phase.

### Gate 2: End of Month 3 — Real Trading Check

**Pass criteria**:
- Matching engine v1 live (paper trades execute against real backend)
- Market maker quoting both sides continuously
- Credit conversion atomic
- First F500 / frontier-lab signed deal in progress

**If fail**: Halt feature work; fix matching engine fundamentals.

### Gate 3: End of Month 5 — Liquidity + Demand Check

**Pass criteria**:
- Order book has measurable depth
- Daily trade volume >$100K paper
- Internal real-money testing cleanly executing
- 3+ F500 / frontier-lab signed accounts
- 1+ DC partnership operational

**If fail**: Reassess. Maybe scope adjustments. Maybe pricing changes.

### Gate 4: End of Month 6 — GA Readiness

**Pass criteria**:
- $300K MRR
- 5+ reference customers
- SOC 2 Type I complete
- Trading uptime >99.95%
- Series A conversations advancing

**If fail**: Delay GA, address blockers.

---

# Part D — Risk Register

## 16. Risk register

### 16.1 Trading-layer-specific risks

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| T1 | Matching engine race conditions / data corruption | Medium | Critical | Single-threaded per-product; event sourcing; replay capability |
| T2 | Index manipulation in low-liquidity paper trading | High | Medium | Trimmed mean; volume floors; surveillance |
| T3 | Real-money testing leaks into customer paper trading | Low | Critical | Strict account separation; `is_paper` flag enforced |
| T4 | Market maker logic causes spread blowout | Medium | High | Risk limits; auto-quote-withdrawal; manual override |
| T5 | Surveillance misses critical manipulation | Medium | High | Rule-based v1 has gaps; ML-based v2 planned |
| T6 | Liquidity never forms | High | Critical | 1Trade-as-market-maker v1; recruit external; demand-side anchors |

### 16.2 Demand-side risks

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| D1 | F500 / frontier-lab sales cycles longer than projected | Medium | High | Multi-deal pipeline; first deal anchors subsequent; demand-side outreach starts month 1 |
| D2 | Enterprise compliance requirements exceed v1 capabilities | Medium | Medium | SOC 2 Type I in v1; flexibility on customer-specific controls; case-by-case approval |
| D3 | Customer procurement teams unfamiliar with credit-based purchasing | Medium | Medium | Education materials; reference customer testimonials once available |

### 16.3 Supply-side risks

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| S1 | DC partners reluctant to commit capacity without volume proof | Medium | Medium | Bilateral commercial terms; small initial allocation; volume-based scaling |
| S2 | Partner DC reliability issues affect 1Trade brand | Medium | High | SLA terms with partners; capacity verification before activation; alerting |
| S3 | Partner payout disputes | Low | Medium | Clear contractual terms; transparent reporting; reconciliation processes |

### 16.4 Regulatory risks

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R1 | CFTC/SEC classifies credits as securities | Medium | High | Conservative legal framing; counsel before launch; non-US jurisdiction option |
| R2 | JFSA pushback on Japanese operations | Medium | High | UBS Japan COO guidance |
| R3 | Customer KYC failures | Medium | Medium | Persona / Parallel Markets integration; AML screening |

### 16.5 Other risks

| ID | Risk | Mitigation |
|---|---|---|
| O1 | Team doesn't materialize | Confirm employment status; recruit aggressively |
| O2 | NVIDIA allocation crisis | NVIDIA partnership; partner DCs may have allocation; AMD backup |
| O3 | Hyperscaler launches competing market | Speed of execution; cross-vendor neutrality |

---

## 17. Open questions

**Credit architecture**:
1. AI credit ↔ sub-credit conversion: bidirectional with spread or one-way?
2. Sub-credit ↔ sub-credit conversion: allowed or force through index?
3. Index publication frequency: daily, hourly, both?
4. GPU credit tiers in v1: H100 only or H100 + H200?

**Trading layer**:
5. Maker rebates at top tier: 0% or negative (pay makers)?
6. Real-money internal testing scope?
7. Customer KYC: light for paper, full for real-money — or full from day one?
8. Cash vs. physical settlement?

**Regulatory**:
9. Trading entity jurisdiction?
10. Securities counsel engagement timing?

**Operational**:
11. Datacenter location and scale?
12. Team headcount?
13. UBS Japan partnership terms?
14. Existing platform access for cross-reference?

**Brand**:
15. Existing brand book — when can Ahmed see it for design system alignment?

---

**End of Phase 6 (v2).**
