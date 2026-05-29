# Exascale — Pitch Deck + Demo Video Script

> Two artefacts, one document, same narrative.
>
> - **Part A** is the deck — one section per slide, ~12 slides total.
>   Investor-grade: numbers up front, no fluff, no AI buzzwords used as
>   adjectives.
> - **Part B** is the demo video script — three cuts (90-second, 3-minute,
>   5-minute). Voice-over written out, with on-screen action notes.
>
> Both lean on the personas in `exascale_persona_demo_flows.md` and the
> screens we've already built. Mock numbers throughout — swap before
> shipping to real investors.

---

# Part A — Pitch Deck

## Slide 1 — Title

```
        E X A S C A L E
─────────────────────────────────────
The commodity market for AI compute.

Seed round · $5M · 2026
Tai · CEO/founder
contact@exascale.com
```

Speaker note: don't elaborate — the title is the thesis. Hold for a beat,
move on.

---

## Slide 2 — The problem (one sentence, three data points)

> **AI compute is the new oil, but it trades like real estate —
> opaque, bilateral, illiquid.**

- **No price discovery.** What does an H100-hour cost today? *Three
  different answers depending on whether you ask AWS, Lambda, or a
  Discord broker.* No public quote, no index, no historical curve.
- **Massive misallocation.** Microsoft committed **$50B** to capex in
  2024 alone. AI labs sign **multi-year reservations**. Meanwhile DC
  partners run at **62% average utilisation** because no liquid spot
  market exists to absorb the rest.
- **Bilateral procurement.** Every deal is a custom Slack thread →
  legal pack → 6-week purchase order. Procurement at a Fortune-500
  AI buyer spends **40 days** to close one capacity contract.

The mismatch is structural. Today there's no AON, no NASDAQ for the
single commodity that powers the entire AI economy.

---

## Slide 3 — Why now (three concurrent forcing functions)

1. **Demand inflection.** Global LLM token volume **10× in 18 months**
   (Jul-2024 → Jan-2026). Demand growth outpaces fab capacity. Hoarding
   replaces planning.
2. **Supply fragmentation.** ~340 medium-tier datacenters now own
   meaningful AI GPU capacity. Five years ago this was 10. None of
   them have channel access to AI buyers without going through a
   hyperscaler.
3. **Regulatory window.** Commodity / derivative trading rules
   (CFTC's commodity-pool framework) cleanly cover credits where
   each unit is a defined unit of service — same rules that birthed
   the electricity exchanges in the 90s.

> Bloomberg made FX trading possible in the 80s by aggregating quotes
> across banks. Coinbase did it for crypto in 2012. *Compute is next.*

---

## Slide 4 — The solution (one diagram)

```
                              EXASCALE VENUE
                  ┌────────────────────────────────────┐
                  │  Matching engine + audit ledger    │
                  └────────────────────────────────────┘
                       ▲                          ▲
                       │                          │
       ┌───────────────┴────────┐   ┌─────────────┴────────────┐
       │  SUPPLY                │   │  DEMAND                  │
       │                        │   │                          │
       │  Datacenters mint      │   │  AI cos buy credits      │
       │  GPU-hours → credits   │   │  to power tokens, image, │
       │  H100, H200, fwd       │   │  video, GPU-hours        │
       │                        │   │                          │
       │  Traders speculate     │   │  Sub-account budgets,    │
       │  on price direction    │   │  burn meters, audit log  │
       └────────────────────────┘   └──────────────────────────┘
```

Three primitives:

1. **Standardised credits** — AI-INDEX (basket), TEXT, SPEECH, IMAGE,
   VIDEO, H100-SPOT, H200-SPOT, H100-FWD-30D.
2. **A real order book** — bid/ask, depth, last-print tape, candle
   history, settlement.
3. **A hash-chained audit log** — every fill, every action, every
   role change, with SHA-256 + ed25519 signatures + periodically
   published Merkle root.

The only product. Nothing else.

---

## Slide 5 — Three audiences, one venue

| Persona | Pain today | What we give them |
|---|---|---|
| **AI company** *(Maya — VP Eng)* | Bilateral contracts, no comparison-shop | Spot procurement at index price · forward contracts to hedge buildout · burn meters to track $/token live |
| **Datacenter partner** *(Tom — Capacity Ops)* | 38% spare capacity earns $0 | Mint unused GPU-hours → tokenised credits → list on venue · partner dashboard with fill-rate + settlements |
| **Trader** *(Jordan — Quant)* | Can't bet on compute prices at all | Direct exposure to AI-INDEX, sub-credit pairs, GPU forwards — same instruments equities desks already trade |

Two-sided market → liquidity. Trader speculation tightens spreads and
makes prices honest. Same playbook as every commodity exchange before.

---

## Slide 6 — Product (live, not vapor)

A real, running frontend on **Nuxt 4 + Vue 3**, **24/26 MVP screens
shipped**, hash-chained audit log, persona-driven guided tours, dark
trading floor + light enterprise admin.

**What you'll see in the demo (Part B):**
- `/trade` — live candle chart, order book, tape, order form (Bloomberg-tier density).
- `/portfolio` — P&L vs the index, position breakdown, attribution.
- `/enterprise/audit` — 20-row chain-of-custody log with cryptographic verification.
- `/enterprise/billing` — MTD spend, projected month-end, cost alerts, invoices.
- `/datacenter` — partner dashboard with capacity sold, fill rate, settlements.
- `/onboarding/tour` — persona-driven autoplay tour with end-of-tour rating.

This deck links a working app, not a Figma file. Investors who say
*"show me the product"* — we open `localhost:3000` and walk it live.

---

## Slide 7 — Market size (TAM-SAM-SOM, conservative)

| Layer | 2026 | 2028 (est.) |
|---|---|---|
| **TAM** Global AI compute spend | $260B | **$1T** |
| **SAM** Compute that *could* trade on a venue (spot + forwards + standardised SKUs) | $40B | **$160B** |
| **SOM** Volume routed through Exascale in Y5 | — | **$5B** (0.5% of TAM) |

**At 10% venue take rate, SOM = $500M ARR by Y5.**

Sources: Bain "Beyond Generative AI" 2025 · Citi/Bernstein AI infra
notes · McKinsey "Compute economics" 2025. Numbers conservative on
purpose — we're not pitching the 0.5% case to win, we're pitching it
to show the floor.

---

## Slide 8 — Business model

**One revenue line. Two ways to charge.**

1. **Venue fee.** 10% of every fill (5% from maker, 5% from taker —
   maker-taker rebate possible). Net 1% effective for very large
   participants via volume tiers.
2. **Forward-roll fees.** Each forward contract that rolls into the
   next maturity accrues a fee. Forwards trade ~3× more frequently
   than spot — disproportionate revenue.

**Unit economics (mock, Y3 mid-case):**

| | Per $1M traded |
|---|---|
| Gross volume | $1,000,000 |
| Venue fee at blended 0.8% | $8,000 |
| Cost-to-clear (matching, settlement, audit) | $400 |
| **Contribution margin** | **$7,600 (95%)** |

This is software margin on top of software margin. The marginal cost
of clearing the millionth trade is approximately zero.

---

## Slide 9 — Moat (it's liquidity, plus three other things)

1. **Liquidity is winner-take-most.** Traders go where the tightest
   spreads are; tightest spreads happen where the most volume is.
   First credible venue captures it.
2. **Datacenter integrations are sticky.** Once a DC has installed
   the agent, integrated into their billing, and routed three months
   of fills through us, switching costs are operational, not technical.
3. **Compliance is a moat for B2B.** SOC 2 Type II, ISO 27001,
   regulator engagement, audit-chain transparency. Anthropic, Meta,
   and Amazon will not buy from a venue without these. They take
   18 months to build. The clock is running.
4. **Index ownership.** Once AI-INDEX becomes a published reference
   rate (think LIBOR for AI compute), every downstream contract
   prices off it. The index *is* the venue.

---

## Slide 10 — Traction & state of the build

**As of 2026-05-25:**

- **24 / 26 MVP screens shipped** (per `exascale_mvp_screens_checklist.md`).
- **3 personas fully covered** — Trader, Enterprise Buyer, Datacenter
  Partner — with end-to-end demo flows in `exascale_persona_demo_flows.md`.
- **Production-style frontend** in Nuxt 4 / Vue 3, dark mode trading +
  light mode admin, hash-chained audit screens, persona-driven
  guided tours (click-through + autoplay video modes).
- **30+ post-MVP screens designed and catalogued** in
  `exascale_v15_screens_prompts.md` — investors can see the v1.5 roadmap.
- **Backend / integration spec** documented in
  `exascale_integration_primer.md` — agent install via mTLS WSS,
  per-job ed25519 signatures, Merkle-root publishing for audit
  verification.

Not yet built: the matching engine, the actual partner agent binary,
real bank rails. That's what the round funds.

---

## Slide 11 — Team & ask

**The ask:** **$5M seed.** 18-month runway.

**Use of funds:**

| | % | $ |
|---|---|---|
| Engineering — matching engine + partner agent + integrations | 55% | $2.75M |
| Compliance + legal — CFTC engagement, SOC 2, MTLs in 50 states | 20% | $1.0M |
| BD — first 5 DC partnerships + 10 AI-co LOIs | 15% | $750K |
| Reserve | 10% | $500K |

**Team to hire in the round:**
- 2 senior infra engineers (matching engine, agent)
- 1 compliance lead (ex-CFTC or ex-CME ideal)
- 1 BD lead (ex-hyperscaler GTM)
- 1 founder pays themselves last

**12-month milestones:**
- Q3-26: alpha matching engine + first DC partner live
- Q4-26: 5 DC partners + 3 AI-co paper-trade customers
- Q1-27: paper → real money, regulatory green light
- Q2-27: $100M in committed annualised volume

---

## Slide 12 — Closing

> Every commodity that became liquid in the last century made
> someone a fortune by being the venue.
>
> Compute is the last unliquid commodity at scale.
>
> We're building the venue.

`exascale.com · contact@exascale.com · 2026`

---

# Part B — Demo video script

Three cuts. Use the longest version for warm intros, the 90-second
version for cold emails, the 3-minute version for follow-up calls.

Voice-over (VO) is in italics. Camera direction is in brackets.
Action notes describe what the recorded screen shows.

---

## Cut 1 — 90 seconds (cold-email teaser)

*Open on the marketing landing page. Camera holds on the headline.*

**VO (0:00):** *"Every commodity that becomes liquid makes someone a
fortune by being the venue. We're building the venue for AI compute."*

*Cut to /trade. Live ticker, candle chart updating, order book
flickering.*

**VO (0:08):** *"This is the trading floor. An order book for tokens,
image generations, GPU-hours — priced in real time, settled in real
time, fully audited."*

*Click the green avatar — the user menu drops down.*

**VO (0:18):** *"Same venue, three audiences."*

*Open /onboarding/tour, pick Trader. Trader walkthrough flashes through
4 screens (chart → portfolio → wallet → close).*

**VO (0:25):** *"Traders bet on compute prices like they bet on oil."*

*Back to picker, pick AI Company. Cuts to /enterprise/billing burn
meter spinning, then /enterprise/audit.*

**VO (0:38):** *"AI companies buy in bulk, watch every dollar burn in
real time, and get a compliance log their CFO will sign off on."*

*Back to picker, pick Datacenter. Cuts to /datacenter dashboard with
KPI strip animating, then /datacenter/register intake.*

**VO (0:52):** *"Datacenters monetise the 38% of GPU capacity that
sits dark today. They install an agent, mint capacity into credits,
and list them on the order book."*

*Cut to the audit page closeup of a hash chain.*

**VO (1:08):** *"Every action — every fill, every role change, every
key rotation — lands in an immutable hash chain. Regulators can verify
the venue ex-post. Enterprises can prove their compute spend was
clean."*

*Pull back to the floating CTA on the homepage with the persona pulse
dot.*

**VO (1:20):** *"Three commodity markets — oil, FX, electricity —
became multi-trillion-dollar venues in the last forty years. Compute
is the last one that hasn't. We're building it."*

*End card:*

```
        E X A S C A L E
─────────────────────────────────
The commodity market for AI compute.

exascale.com · raising $5M seed
```

**VO (1:30):** *"Exascale. The commodity market for AI compute."*

*Cut.*

---

## Cut 2 — 3 minutes (follow-up call)

Add to the 90-second cut, in this order:

### + 30s: The problem in the opening

After the title card, before the trading floor:

**VO:** *"Today, if you're an AI company and you want to buy a million
H100-hours, you have a choice. You can talk to AWS — they'll quote you
whatever this quarter's price is, take it or leave it. You can call a
broker on Discord. Or you can wait six weeks for a custom procurement
deal with a tier-3 datacenter. There is no public price. No order
book. No way to hedge. Most AI companies overpay by 25 to 40 percent
because there's no comparison shop. Datacenters lose money on capacity
that sits idle. Both sides need a market."*

*Camera shows the audience-split section on the landing page — for
traders, for AI companies, for datacenters — slowly scrolling.*

### + 40s: The market sizing slide

After the persona walkthroughs, before the audit page:

*Cut to a clean number card on dark background.*

**VO:** *"Global AI compute spend is on track to cross one trillion
dollars by 2028. Bain, Citi, McKinsey all converge on that number.
About 40 percent of that — 400 billion dollars annually — is
standardised enough to trade. Even 0.5 percent flowing through us is
five billion in annual volume. At a ten percent venue fee, that's
five hundred million in revenue per year by year five."*

*Show a small bar chart of the TAM-SAM-SOM funnel.*

### + 20s: Why now

After the datacenter cut, before the audit screen:

**VO:** *"This wasn't possible five years ago. Three things converged.
Token volume exploded ten-x in eighteen months — demand far outstripped
supply. The CFTC's commodity-pool framework now cleanly covers
standardised compute credits. And the trading-infrastructure stack —
matching engines, mTLS, hash-chained audit — is commodity software now.
The clearing tech that took NASDAQ a billion dollars to build, we run
on rented servers."*

*Show a montage: token-volume chart climbing, then a CFTC seal, then a
schematic of the venue stack.*

---

## Cut 3 — 5 minutes (warm intro / pitch day)

Build on Cut 2. Add:

### + 90s: Deep walkthrough on the trader path

Right after the trader picker selection in the persona section, expand
to a real two-minute walkthrough:

*Open `/trade` cleanly. Camera holds on the chrome.*

**VO:** *"This is the trading floor. Let me show you what a Bloomberg-
trained desk sees. Top-left, the live AI Index ticker — that's $1 in
US dollars buys 1,002.4 credits at this moment. It's ticking — every
two and a half seconds, the matching engine prints a new mid-price."*

*Drop the cursor over the candle chart.*

**VO:** *"One-minute candles, last 240 of them visible. Hover the
crosshair — it shows OHLC, volume at that bar, and cost-to-take at
the current depth."*

*Click into the order book.*

**VO:** *"Real-time depth. Click any level to pre-fill the order form
on the right. Place a $1,000 market buy on AI-INDEX. Fill comes back
in under fifty milliseconds — fee, slippage versus mid, queue position,
all visible. Same telemetry an equities prop desk has."*

*Cut to /portfolio.*

**VO:** *"Portfolio attributes P&L against the AI-INDEX automatically.
You always know whether you beat beta or just rode it. Position
breakdown by credit family, allocation donut, 30-day P&L strip per
position. Click any row…"*

*Click a position row → position drawer slides in.*

**VO:** *"…and you get the close/reduce/limit/stop flow. Confirm any
destructive action and the realised P&L is crystallised, the fill
lands in /history, and the action writes a row to the audit chain.
End to end, under a second."*

### + 60s: Deep walkthrough on the audit chain

Replace the existing audit cut with a longer one:

*Open /enterprise/audit. Show the 20-row dense table.*

**VO:** *"Compliance officers spend their lives in two places:
spreadsheets and audit logs. Here is the audit log a compliance
officer wants. Twenty rows visible, fourteen thousand two hundred and
eighty seven on the chain. Each row is the type of action — a login,
an API key created, a budget update, a sub-account spend — actor,
sub-account, resource, IP, and the hash."*

*Click any row → detail panel expands.*

**VO:** *"Click a row and you get cryptographic proof. Block number,
previous hash, this hash, algorithm — SHA-256 with ed25519
signatures, signed by the venue's audit key. Merkle root for the
epoch is published to a public-write-once medium every fifty blocks,
so external auditors can verify the chain ex-post."*

*Hover the chain-integrity badge in the header.*

**VO:** *"This pill at the top — chain integrity verified, last
checked nine minutes ago — is a live re-hash of the entire chain. If
anyone tampers with a single byte anywhere in fourteen thousand blocks,
this turns red within three hundred seconds."*

*Cut to /enterprise/billing.*

**VO:** *"And it ties into billing. Every dollar spent through the
venue traces back to a chain hash. The CFO sees not just the number
on the invoice — she sees the cryptographic chain proving that number
is the correct sum of every fill her organisation made this month."*

### + 30s: The team and ask

Same as Slides 11 / 12 in the deck. Close on the brand mark holding
for two beats over a soft ambient track.

---

# Part C — Investor objection prep (for live Q&A, not script)

Investors will ask. Answers prepared.

**"Why won't AWS / Azure / GCP just build this?"**
They could, but won't. Building a venue means inviting their own
hyperscaler customers to comparison-shop them. AWS already lost
margin to Cloudflare on egress and Snowflake on warehousing — they
will not voluntarily commoditise their highest-margin SKU.

**"Why won't Anthropic / OpenAI build this?"**
Same logic. Frontier labs want to *be* the buyer, not the venue. The
venue is a neutral utility. Frontier labs are participants.

**"How do you bootstrap liquidity in a two-sided market?"**
The same way every exchange has, ever: pay maker rebates, partner
exclusively with two anchor DCs and two anchor AI buyers, write the
first 90 days of trades manually if we have to. Two-sided markets are
hard; "hard" doesn't mean "impossible," it means defensible.

**"What if NVIDIA changes the SKU and the standard breaks?"**
SKU lifecycles are 18-24 months. By the time H200 → B200 → next, we
already have liquid forward markets on the new generation. The venue
indexes the *current* spot SKU; the SKU is a parameter, not the
architecture.

**"What's stopping a DC from cheating you on capacity reports?"**
- Hardware validation at onboarding (real NCCL allreduce — can't fake).
- Spot-check jobs the venue knows the expected hash for.
- Buyer reputation feedback.
- NVIDIA's GPU attestation feature on H100+ is rolling out and
  cryptographically proves what code a GPU ran. We'll adopt as it
  matures.

**"What's the regulatory risk?"**
Lower than crypto. AI credits are unit-of-service, not currency. CFTC
commodity-pool framework cleanly applies. We've engaged a former CFTC
commissioner as an adviser (placeholder for the real deck). We do
NOT need MTLs in every state because we don't hold customer fiat —
banking partners do that for us.

**"What's the realistic worst case?"**
We get to $50M ARR in five years instead of $500M. We're still a
$300M-$500M exit at the strategic acquirer level (CME, ICE, even
Stripe-as-rails). The bull case is the listed-exchange comparable —
that's a $10B-$20B outcome.

---

# Production checklist

- [ ] Replace all placeholder names ("Tai · CEO/founder") with real
      team page.
- [ ] Swap mock numbers in Slide 8 (unit economics) for real cost-per-
      clear once the matching engine spec is final.
- [ ] Re-record the demo cuts after `/onboarding/tour` autoplay video
      is fully polished (the bottom video-player bar is the new visual
      anchor — make sure it reads cleanly at 1080p).
- [ ] Cut the 90s teaser version with no VO — captions only — for
      LinkedIn / Twitter. People watch with sound off there.
- [ ] Have a compliance lawyer review every claim in Slide 8 (revenue
      projections) and Slide 9 (regulatory framing) before sending to
      anyone you haven't met.
