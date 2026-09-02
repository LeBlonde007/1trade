# 1Trade MVP Screens — Claude Design Prompts
**26 paste-ready prompts for Claude to produce visual mockups of every v1 screen.**

> Status: Drafted 2026-05-19. Derived from Phase 6 v2 §1.3 use cases and §4 feature specs.
> Purpose: Each prompt produces a visual HTML mockup when pasted into Claude with the frontend-design skill enabled. NOT code-architecture prompts — these are designer prompts.
> Output: Claude artifacts (HTML pages with mock data, charts, layouts) that Tai and the team can react to, share in pitches, and use as the foundation for production design work.

---

## How to use this catalog

### The right setup

1. **Open a fresh Claude conversation**
2. **Have the design brief open** (`1trade_design_system_brief.md`) — you'll paste the foundation tokens from it first
3. **Pick one screen** from this catalog at a time
4. **Paste the Section A foundation, then the screen-specific prompt** — both into the same Claude message
5. **Claude produces an HTML artifact** — a rendered visual mockup with mock data, charts, and styling
6. **Iterate** — ask Claude to adjust until the screen looks right
7. **Save the artifact** — screenshot it, share it, or evolve it into production design

### What Claude produces

For each prompt, Claude generates:
- A self-contained HTML page with inline CSS and JavaScript
- Mock data that looks realistic (Brownian motion price movement, plausible order book depth, etc.)
- Working interactivity where it makes sense (hover states, tab switching, simple animations)
- Charts using lightweight libraries (chart.js, lightweight-charts loaded from CDN, or hand-built SVG)
- Pixel-level visual design — not wireframes

### Why one screen per prompt

Bigger prompts produce worse output. Each prompt focuses Claude on a single artifact at the resolution needed for design review.

### Tier system

- **Tier 1 (deep prompts)** — the demo assets that prospects and investors will see. Spend the prompt budget here. Five screens.
- **Tier 2 (medium prompts)** — important supporting screens. Ten screens.
- **Tier 3 (lean prompts)** — utility screens and states. Eleven screens.

---

## Table of contents

**Section A — Foundation (paste with every prompt)**
- [A1. Design tokens](#a1-design-tokens)
- [A2. Mock data shape](#a2-mock-data-shape)
- [A3. Aesthetic guardrails](#a3-aesthetic-guardrails)
- [A4. Claude artifact instructions](#a4-claude-artifact-instructions)

**Section B — Tier 1 (demo-asset screens)**
- [B1. Landing page](#b1-landing-page)
- [B2. Trading dashboard (the headline)](#b2-trading-dashboard-the-headline)
- [B3. Market detail](#b3-market-detail)
- [B4. Credit wallet](#b4-credit-wallet)
- [B5. Index methodology page](#b5-index-methodology-page)

**Section C — Tier 2 (supporting screens)**
- [C1. Signup](#c1-signup)
- [C2. KYC light flow](#c2-kyc-light-flow)
- [C3. Portfolio](#c3-portfolio)
- [C4. Trade history](#c4-trade-history)
- [C5. Buy credits flow](#c5-buy-credits-flow)
- [C6. Enterprise onboarding](#c6-enterprise-onboarding)
- [C7. Team and sub-accounts](#c7-team-and-sub-accounts)
- [C8. Compute instances list](#c8-compute-instances-list)
- [C9. Create instance flow](#c9-create-instance-flow)
- [C10. Inference playground + model catalog](#c10-inference-playground--model-catalog)

**Section D — Tier 3 (utility screens)**
- [D1. Login](#d1-login)
- [D2. Email verification + welcome](#d2-email-verification--welcome)
- [D3. Audit log](#d3-audit-log)
- [D4. Billing dashboard](#d4-billing-dashboard)
- [D5. DC partner dashboard (internal)](#d5-dc-partner-dashboard-internal)
- [D6. Partner capacity registration](#d6-partner-capacity-registration)
- [D7. Settings shell](#d7-settings-shell)
- [D8. API keys settings](#d8-api-keys-settings)
- [D9. Notifications drawer](#d9-notifications-drawer)
- [D10. Command palette (cmd+K)](#d10-command-palette-cmdk)
- [D11. Empty / loading / error states](#d11-empty--loading--error-states)

---

# Section A — Foundation (paste with every prompt)

## A1. Design tokens

```
DESIGN TOKENS — 1Trade v1 (placeholder, swap when brand book lands)

DARK MODE (primary for trading product)
  --surface-canvas:       #0A0B0E
  --surface-elevated:     #14161B
  --surface-overlay:      #1C1F26
  --text-primary:         #E8E6E0
  --text-secondary:       #9A9A95
  --text-tertiary:        #5F5F5C

LIGHT MODE (primary for marketing + compute dashboard)
  --surface-canvas:       #F8F7F4
  --surface-elevated:     #FFFFFF
  --surface-overlay:      #FFFFFF
  --text-primary:         #0A0B0E
  --text-secondary:       #5F5F5C
  --text-tertiary:        #9A9A95

BORDERS
  Dark: rgba(255,255,255,0.08) default, 0.16 strong
  Light: rgba(0,0,0,0.08) default, 0.16 strong
  Focus ring: #4A90E2

BRAND ACCENT (placeholder)
  --brand-primary:        #C8F25C (lime)
  --brand-primary-hover:  #B8E548
  --brand-accent:         #4A90E2

SEMANTIC (CRITICAL for trading UI)
  --positive:             #19C37D (bullish, gain, buy)
  --positive-subtle:      rgba(25,195,125,0.12)
  --negative:             #EF4444 (bearish, loss, sell)
  --negative-subtle:      rgba(239,68,68,0.12)
  --warning:              #F59E0B
  --info:                 #4A90E2

  Always pair with shape: positive → ▲ / negative → ▼

TYPOGRAPHY
  Body / UI: Inter, system-ui fallback. Use 'font-feature-settings: "tnum"' for numbers.
  Display: Inter (heavier weight) or Inter Display
  Monospace: JetBrains Mono (prices, code, trading data)

TYPE SCALE
  display-xxl:  96px / 700 / -2% tracking
  display-xl:   64px / 700 / -1.5% tracking
  display-l:    48px / 600 / -1% tracking
  h1:           32px / 600
  h2:           24px / 600
  h3:           20px / 600
  h4:           18px / 600
  body-l:       18px / 400 / 1.55 line-height
  body:         16px / 400 / 1.55 line-height
  body-s:       14px / 400
  caption:      12px / 500
  tiny:         10px / 600 / 0.08em tracking / UPPERCASE

SPACING (base 4): 0, 4, 8, 12, 16, 24, 32, 48, 64, 96 px

RADIUS — SHARP (institutional)
  none: 0
  sm:   2px
  md:   4px
  lg:   6px

NUMBER FORMATTING — always
  font-variant-numeric: tabular-nums
  Right-align in tables
  Decimal alignment for prices
  Monospace font for prices in trading UI
  Positive: green + ▲; Negative: red + ▼; Neutral: text-primary
```

## A2. Mock data shape

```
USE REALISTIC MOCK DATA, NOT RANDOM NUMBERS.

PRICES — Brownian motion with mean reversion
  AI Index current: $0.001005 (1 AI credit ≈ $0.001)
  Sub-credit relative prices:
    TEXT  ≈ $0.001210
    SPEECH ≈ $0.001200
    IMAGE ≈ $0.008000
    VIDEO ≈ $0.250000
    NICHE ≈ $0.000400
  GPU credits:
    H100 = $2.99 (1 GPU-hour)
    H200 = $3.49

ORDER BOOK — power law depth
  Spread: 1% (e.g., bid $0.000990, ask $0.001010)
  10 levels each side
  Best bid/ask: 5K-50K credit quantity
  Deeper levels grow larger (power law)

TRADE TAPE — log-normal sizes
  30-300 trades/minute
  Median size: 500 credits
  Occasional large prints (50K-100K credits)
  Time format: HH:MM:SS.mmm

CANDLES — plausible OHLC
  1-minute candles for chart
  ~0.3% range per candle typical
  Volume bars at bottom: log-normal

INDEX HISTORY
  Normalized to 1.0000 at launch
  Currently 1.0024
  24h change: +0.18%, 7d -0.42%, 30d +2.31%, YTD +4.18%
  Realistic curve: drift up with periodic 5-10% drawdowns

DEMO ACCOUNT — paper trading
  Cash: $10,247.83
  Positions:
    Long 50,000 AI credits @ avg 0.000980 → P&L +$1.25 (+2.55%)
    Long 12,000 text credits @ avg 0.00118 → P&L +$0.24 (+1.69%)
    Short 5,000 image credits @ avg 0.00810 → P&L +$0.60 (+1.48%)
    Long 8 H100 credits @ avg 2.95 → P&L +$0.32 (+1.36%)
  Total value: $10,247.83
  Today P&L: +$23.41 (+0.45%)
```

## A3. Aesthetic guardrails

```
1TRADE IS INSTITUTIONAL / TECHNICAL / FINANCIAL / MODERN.

REFERENCE POINTS (study these)
  ✓ Bloomberg Terminal — information density, professional gravitas
  ✓ Polymarket — clean trading UI, serious modern look
  ✓ Stripe — calm confidence, restrained color
  ✓ Linear — performant, dark-mode-native, technical
  ✓ Vercel — minimal, designer-credible, monochrome

ANTI-REFERENCES (NEVER look like these)
  ✗ Robinhood — too consumer-cheerful
  ✗ Coinbase consumer app — too playful
  ✗ Web3 / crypto aesthetic — neon, hex shapes, gradients
  ✗ Generic AI startup — purple gradients, neural-network imagery
  ✗ Stodgy traditional bank — navy + gold
  ✗ Agency portfolio — asymmetric grids, oversized type for type's sake

DESIGN PRINCIPLES

  1. NUMBERS-FIRST DESIGN
     The trading product is a numbers-first product. Numbers get the
     largest size in their context, mono/tabular font, and the
     semantic color. Everything else is supporting chrome.

  2. DENSITY > WHITESPACE (for product UI)
     Marketing site can be generous. Product UI (especially trading)
     should be DENSE — Bloomberg-tier per-screen information.
     Compress padding, tighten line-heights, use smaller type sizes
     than you would for marketing.

  3. BOLDNESS COMES FROM PRECISION, NOT MAXIMALISM
     The frontend-design skill says be bold. For 1Trade, boldness
     means: information density, perfect alignment, intentional
     spacing, restrained color. NOT unusual fonts, asymmetric grids,
     or decorative animation.

  4. RESTRAINT
     When in doubt, favor restraint. A trading venue earns trust
     through visual sobriety.

  5. LIVE DATA WHERE POSSIBLE
     For mockups: animate price changes, candle updates, tape rolls.
     Movement signals "real market" to viewers. But subtle — never
     jarring or distracting.
```

## A4. Claude artifact instructions

```
OUTPUT FORMAT — every prompt in this catalog wants:

1. A SINGLE HTML ARTIFACT
   - Self-contained: inline CSS, inline JS, CDN libraries OK
   - Renders inline in the Claude chat
   - Full-page mockup, not snippets

2. INTERACTIVITY WHERE IT MATTERS
   - Live-updating mock data (use setInterval to simulate price ticks)
   - Hover states, tab switching, dropdown opens
   - But: simple. No build tools, no frameworks. Vanilla JS or alpine.js if needed.

3. ALLOWED LIBRARIES (CDN only)
   - chart.js (for line charts and some candle approximations)
   - lightweight-charts (TradingView's library — best for candlesticks)
   - alpine.js (for declarative interactivity)
   - lucide (icons via CDN)
   - tailwindcss (via play CDN)
   - Tone.js (only if generating audio cues — usually NOT needed)

4. RESPONSIVE
   - Design for 1440px viewport native (the trading dashboard especially)
   - Should gracefully degrade to 1024px
   - Mobile: lower priority for v1 (trading UI is desktop-first)

5. DENSITY APPROPRIATE
   - Marketing pages: generous whitespace, large headlines
   - Trading UI: dense, information-rich
   - Match the surface

6. PERFECT TYPOGRAPHY
   - Tabular figures for numbers (always)
   - Monospace for prices in trading UI
   - Correct vertical rhythm
   - Proper letter-spacing on small caps / uppercase labels

7. NO PLACEHOLDER TEXT
   - Use real-looking data, not "Lorem ipsum"
   - Names, addresses, market names should look credible
   - Numbers should be realistic per A2

OUTPUT THIS, NOT THAT:
  ✓ Full visual mockup with realistic data, rendered HTML
  ✗ Wireframe with boxes and "TITLE HERE"
  ✓ Working chart with mock OHLC data
  ✗ Static image of a chart
  ✓ Interactive order book updating every second
  ✗ Snapshot of an order book

When designing, ASK YOURSELF: "Would Larry Fink mistake this for a real Bloomberg-tier product, or does it look like a startup demo?" Aim for the former.
```

---

# Section B — Tier 1 (demo-asset screens)

These five screens are the demo asset. Spend the time. These will be screenshotted, shared in pitches, evaluated by investors and prospects.

---

## B1. Landing page

```
Design the 1Trade public homepage as a single HTML artifact.

CONTEXT
1Trade is the commodity market for AI compute — a three-sided market 
connecting GPU datacenters, traders, and AI companies through tradeable 
credits. Owned modest datacenter capacity provides credible underlying; 
partner datacenters provide scale. Think CME for AI compute. The closest 
analogues are Bloomberg Terminal and Polymarket, NOT CoreWeave or Lambda.

This landing page is the first thing prospects see. It must read 
institutional, technical, and credible in under 5 seconds — and drive 
the visitor to either open an account or contact enterprise sales.

MODE: Light mode (cream background #F8F7F4)
VIEWPORT: 1440px native, gracefully responsive

REQUIRED SECTIONS IN ORDER

1. TOP NAV — sticky on scroll, subtle backdrop blur when scrolled past hero
   - Left: "1Trade" wordmark (use a tight modern sans, plain text — no logo yet)
   - Center: links — Markets, Index, Compute, Docs, About
   - Right: "Sign In" text link, "Open Account" primary button (sharp 
     corners, brand-primary lime fill, dark text)

2. HERO — the focal point
   - Eyebrow tiny label: "— THE COMMODITY MARKET FOR AI COMPUTE"
   - Headline: "AI compute, but tradeable." — display-xxl size, 
     weight 700, tight tracking
   - Subhead body-l: "Tradeable credits connecting GPU datacenters, 
     traders, and AI companies. Owned underlying for credibility. 
     Partner-supplied scale."
   - LIVE INDEX TICKER — the killer feature
     A horizontal pill or card showing real-time mock index value:
     "1Trade AI Index · $1 = 1,002.4 AI credits · ▲ 0.18% (24h)"
     The number updates every 5 seconds with a subtle 200ms color flash 
     on change. Below: "Last print 16:00 UTC · Next in 3h 24m"
   - CTA row: "Open Account" primary button, "View markets →" text link
   - Below CTAs: small 30-day index sparkline (200px wide, mock data 
     showing slight upward drift with occasional dips)

3. THE PROBLEM — institutional validation
   - Section eyebrow: "— THE PROBLEM"
   - Centered pull-quote, italicized, large display-l:
     "There needs to be a market for compute. No solution yet."
     — Larry Fink, CEO BlackRock, 2026
   - Three stat blocks below (3-column grid):
     | $200B+ | Annual global GPU compute spend, untraded
     | 5-10× | Spread between reserved and spot pricing
     | 0 | Functioning venues for AI compute as asset class
     Stats are huge (display-xl), lime-tinted, tabular figures, with 
     small caption labels below.

4. THE THREE-SIDED MARKET — diagram + explanation
   - Eyebrow: "— HOW IT WORKS"
   - Headline: "Three sides. One venue."
   - Custom SVG diagram showing three nodes connected by arrows to a 
     central "1Trade Trading Venue" hub:
       [ GPU DATACENTERS ] → [ TRADERS ] → [ AI COMPANIES ]
            Supply             Liquidity        Demand
     All converging into a central rectangle labeled "1Trade Venue."
     Three nodes color-coded: supply green, liquidity orange, demand blue.
     Below the diagram: three columns of body text, one per role,
     each with 2-3 bullet points on what that side does.

5. THE PRODUCTS — three stacked cards
   - Eyebrow: "— THE PRODUCTS"
   - Cards arranged vertically (full-width on desktop), one per product:
     a) Trading layer — "Order book, market maker, daily index. 
        AI credits and GPU credits as tradeable instruments."
        Right side of card: tiny mockup of an order book widget
     b) Inference layer — "Curated SoTA models. OpenAI-compatible API. 
        Multi-tenant per GPU."
        Right side: tiny mockup of API request/response
     c) Compute layer — "H100 / H200 capacity. CLI-first. Free egress."
        Right side: tiny mockup of terminal output

6. THE INDEX — featured
   - Eyebrow: "— THE 1TRADE AI INDEX"
   - Headline: "The price of AI compute, published daily."
   - Large interactive chart (use chart.js or lightweight-charts) 
     showing 365 days of mock index data. Hover to see exact daily value.
   - Below chart row: current value | yesterday | 7d change | 30d | YTD
   - "View methodology →" link

7. FOR DIFFERENT AUDIENCES — 3-column
   - "For Traders" → fee tiers, FIX (coming), market data
   - "For AI Companies" → bulk credit purchase, free egress, multi-currency
   - "For Datacenter Partners" → market access, idle capacity monetization
   - Each column gets a short paragraph, 3 bullets, and a "Learn more →" link

8. TRUST INDICATORS — single row
   - "SOC 2 Type I path · Index audited externally · 
      Surveillance from day one · Anchor partner: UBS Japan"
   - No specific names, just credibility markers
   - Subtle dividers between items

9. FOOTER CTA
   - Display-xl headline: "Open an account in 5 minutes."
   - Subhead: "Paper trading available immediately. Real-money 
     trading in v1.5 with full KYC."
   - Two buttons side by side: "Open Account" (primary), 
     "Talk to enterprise sales →" (text)

10. FOOTER
    - Multi-column: Product | Markets | Compute | Company | Legal
    - Each column with 4-6 links
    - Bottom row: 1Trade wordmark, copyright, status indicator 
      "● All systems operational" (green dot), live index price 
      always visible

AESTHETIC NOTES
- Institutional, not consumer-cheerful. Bloomberg / Stripe energy.
- Restraint in color: cream/black/grey/brand-primary lime as accent only
- Heavy typography, restrained imagery, no stock photography
- Generous vertical rhythm — 96-128px between sections
- Subtle grid pattern at very low opacity (1-2%) in background to 
  allude to market depth aesthetic without being literal
- All numbers use tabular figures throughout

LIVE DATA — make it feel alive
- Hero ticker updates every 5 seconds with mock price drift
- Sparkline in hero animates draw on page load
- Index chart in section 6 has live cursor on hover
- Status indicator dot pulses subtly

DELIVERABLE
Single HTML file, complete, self-contained. Tailwind CDN OK.
chart.js or lightweight-charts via CDN OK.
Inline JS for the live ticker simulation.
Render it as a Claude artifact.

ACCEPTANCE
- Looks like Stripe / Bloomberg / Linear, NOT like Robinhood / Coinbase consumer
- Live ticker visible in hero with realistic price movement
- All numbers tabular, decimal-aligned
- Scrolls cleanly through all 10 sections
- No emoji decoration, no generic "rocket ship" iconography
```

---

## B2. Trading dashboard (the headline)

```
Design the 1Trade trading dashboard — the HEADLINE product screen.

CONTEXT
This is the single most important visual asset 1Trade will produce 
for v1. Tai's direction: "moving candlesticks, mock data, get a look 
and feel of things." This screen is what investors, F500 procurement 
teams, frontier-lab CTOs, and traders see first when evaluating 
whether 1Trade is a credible market venue.

It MUST look Bloomberg / Polymarket / TradingView tier. If this 
looks like a startup MVP, the pitch collapses. Spend the time.

MODE: DARK MODE PRIMARY (#0A0B0E surface)
VIEWPORT: 1440px native — this screen is desktop-first

LAYOUT — multi-pane

  TOP BAR (60px height, full width)
  ─────────────────────────────────
  [SIDEBAR  ][   CHART   ][ ORDER BOOK ]
  [   80px  ][  ~50% W   ][   ~25% W   ]
  [         ][           ]
  [         ][ ORDER     ][ TRADE TAPE ]
  [         ][ ENTRY     ][   ~25% W   ]
  [         ][  ~50% W   ]
  ─────────────────────────────────
  [        POSITIONS PANEL (full width below)        ]

  All panels conceptually resizable (just show them at sensible defaults).

TOP BAR — 60px tall
  Left:
    - 1Trade wordmark (small, monochrome)
    - Market selector: "EAI-IDX · AI Index" with current price "$0.001005" 
      and change "▲ 0.18%" inline. Dropdown caret. Click expands market list.
  Center: empty (breathing room)
  Right:
    - Account pill: "$10,247.83" — clickable opens wallet
    - Search icon (cmd+K hint)
    - Notifications bell with badge "2"
    - User avatar (initial in circle)

SIDEBAR — 80px wide collapsed, icon-only
  Vertical stack of Lucide icons with text label below each on hover:
    - chart-line (Trade — CURRENTLY ACTIVE, brand-primary highlighted)
    - list (Markets)
    - trending-up (Index)
    - briefcase (Portfolio)
    - clock (History)
    - wallet (Wallet)
    [separator]
    - server (Compute)
    - message-square (Inference)
    [bottom]
    - settings (Settings)
  Active item: brand-primary left border (3px), tinted background
  Bottom: status indicator "● Market open" (small green dot)

CHART PANEL (top-center, ~50% width × ~60% height of available)
  Header row (compact, 40px):
    "EAI-IDX" big bold next to current price "$0.001005" mono tabular 
    next to "▲ 0.18%" semantic-green badge next to 
    "Vol $1.2M · H $0.001012 · L $0.000996" small muted
    Right side: timeframe selector — segmented control:
    [1m] [5m] [15m] [1h] [4h] [1d] (5m active by default)
    Then: "+ Indicators" dropdown button
  
  Chart itself:
    Use lightweight-charts (CDN: 
    https://unpkg.com/lightweight-charts/dist/lightweight-charts.standalone.production.js)
    - Candlestick series with volume histogram below
    - Bullish candles: var(--positive) #19C37D
    - Bearish candles: var(--negative) #EF4444
    - Background: var(--surface-canvas)
    - Grid: very subtle, near-invisible
    - Crosshair on hover with floating price+time tooltip
    - Y-axis right (price, mono tabular)
    - X-axis bottom (time)
    - Generate 200 1-minute mock candles with Brownian motion
    - Update with a new candle/tick every 3 seconds
    - Volume bars at bottom, semi-transparent

ORDER BOOK PANEL (top-right, ~25% width × ~60% height)
  Header: "Order Book" tiny label + "10 levels" badge
  
  Two stacked sections: ASKS (top, red-tinted), BIDS (bottom, green-tinted)
  Each row 24px tall, 3 columns:
    Price (mono tabular, right-aligned, in semantic color)
    Quantity (mono tabular, right-aligned, muted color)
    Total (mono tabular, right-aligned, muted+smaller)
  
  Behind each row: semi-transparent horizontal bar showing relative 
  size, left-anchored. Bar color matches side tint.
  
  10 levels of asks (ordered ascending — best ask at bottom of asks 
  section), 10 levels of bids (descending — best bid at top of bids 
  section).
  
  MID ROW (between asks and bids, larger):
    "$0.001005 ▲" centered, bigger font
    "Spread $0.000020 / 1.99%" smaller below
  
  Updates: every 1-2 seconds, simulate level adjustments with 
  brief opacity flash on changed levels (200ms fade).

TRADE TAPE PANEL (bottom-right, ~25% width × ~40% height)
  Header: "Trades" with "auto-scroll ⏸" toggle
  
  Scrolling list (newest on top):
    HH:MM:SS.mmm  Price       Quantity     Side
    14:23:47.123  0.001005    1,500        ▲ (semantic-green tint)
    14:23:46.892  0.001004      250        ▼ (semantic-red tint)
    14:23:46.451  0.001004    8,200        ▼
    ...
  
  All mono tabular. New trades slide in from top (150ms animation).
  Show last 40 trades. Auto-scroll. Pauses on hover.

ORDER ENTRY PANEL (bottom-center, ~50% width × ~40% height)
  Tabs at top: [Buy] [Sell] — full-width segmented, the active tab 
  is in semantic color (green for Buy, red for Sell)
  
  Below active tab:
    Order type segmented: [Market] [Limit] [Stop disabled]
    
    Price input (only when Limit):
      Big number field, mono tabular, with ▲/▼ tick buttons.
      Pre-filled with current best bid (for sell) or best ask (for buy).
      Right side: "Last 0.001005" inline button to set to last price
    
    Quantity input:
      Big number field, mono tabular
      Quick-fill row: [25%] [50%] [75%] [100%] — of available balance
    
    Summary preview (live, mono tabular, right-aligned):
      Price × Quantity      $0.001005 × 1,000   = $1.005
      Taker fee 1.00%                           + $0.010
      Total                                       $1.015
    
    Available balance: "$10,247.83 USD · 10,196,841 AI credits buying power"
    
    Submit button — LARGE, full-width
      Background: semantic-green (Buy) or semantic-red (Sell)
      Text in white: "Buy 1,000 AI credits — $1.015"
      Click → confirmation modal (don't implement, just stub)

POSITIONS PANEL (bottom, full width spanning all columns)
  Header: "Positions" + tabs [Open · 4] [Closed] [All]
  Right: "P&L today +$23.41 ▲ 0.45%" then "Total value $10,247.83"
  
  Table (data-dense):
    Market    | Side  | Size    | Avg Entry | Mark     | P&L $    | P&L %   | Actions
    EAI-IDX   | Long  | 50,000  | 0.000980  | 0.001005 | +$1.25   | ▲ 2.55% | [Close]
    TEXT-SPOT | Long  | 12,000  | 0.00118   | 0.00120  | +$0.24   | ▲ 1.69% | [Close]
    IMAGE-SPOT| Short | 5,000   | 0.00810   | 0.00798  | +$0.60   | ▲ 1.48% | [Close]
    H100-SPOT | Long  | 8       | 2.95      | 2.99     | +$0.32   | ▲ 1.36% | [Close]
  
  All numbers mono tabular, decimal-aligned.
  Side: "Long" green-tint badge, "Short" red-tint badge.
  P&L: semantic color + arrow.
  Row hover: subtle background tint.
  [Close] action: small destructive-style button.

AESTHETIC NOTES — DON'T MISS THESE
- DENSE. This is Bloomberg-tier, not consumer-app. Compress padding 
  everywhere. Use 12-13px body text in trading UI, not 16px.
- All numbers mono tabular. All. Numbers.
- Subtle dividers (1px, 8% opacity) between panels — don't use heavy borders
- No decorative graphics. No illustration. Just data + chrome.
- Hover states: subtle background tint, never glow or shadow
- Active state on tabs: brand-primary underline or fill
- Color discipline: dark grey/black surfaces, white-ish text, semantic 
  green/red only where it carries meaning, brand-primary lime VERY 
  sparingly (just active states and CTAs)

LIVE BEHAVIOR (make it feel like a real market)
- Chart updates every 3 seconds with new tick
- Order book updates every 1.5 seconds with subtle flash
- Trade tape adds 1-3 trades every 5 seconds, sliding in from top
- Mid-price in order book updates with each new trade
- Position P&L recalculates on every price change

DELIVERABLE
Single HTML file, dark mode. Tailwind CDN. lightweight-charts CDN.
Lucide icons via CDN.
Inline JavaScript for all the live data simulation:
  - Mock OHLC candle generator (Brownian motion)
  - Order book generator (power law depth)
  - Trade tape generator (log-normal sizes)
  - Position mark-to-market

ACCEPTANCE CRITERIA — be ruthless
- Would a real trader trust this to place a $100K order? If not, redesign.
- Information density similar to Polymarket or close-to-Bloomberg
- 60fps on chart updates (use requestAnimationFrame, not setInterval at <100ms)
- No purple gradients. No glow effects. No emoji. No friendly rounded corners.
- All four panels visible without scrolling at 1440×900 viewport
```

---

## B3. Market detail

```
Design a single-market detail page for the 1Trade platform.

CONTEXT
This is the view a user lands on when they click into a specific 
market — either from the public markets listing or from the markets 
sidebar in-app. It's a STUDY / RESEARCH view: less dense than the 
trading dashboard (B2), more focused on a single market's data and 
context.

MODE: Dark mode (consistent with rest of app)
VIEWPORT: 1440px native, responsive down to 1024px
URL conceptually: /markets/EAI-IDX

CONTENT — using AI Index (EAI-IDX) as the example market

1. APP CHROME — top bar and sidebar same as B2

2. MARKET HEADER (full width, large, ~200px tall)
   Left side (~70% width):
     - Symbol "EAI-IDX" tiny eyebrow
     - Market name: "AI Index" big display-l
     - Current price: "$0.001005" — HUGE, display-xl, mono tabular
     - 24h change: "▲ 0.18% · +$0.000002" semantic-green badge
     - Below: small line "Last updated 14:23:47.123 UTC · next index print 16:00 UTC"
   Right side (~30% width):
     - Big primary CTA button "Trade this market →" (routes to B2 with this market)
     - Below: "Add to watchlist" text link
     - Below: 24h sparkline (subtle, 200px wide × 60px tall)

3. STATS GRID (4 columns, single row)
   Each cell: caption label on top, big mono number below, small change indicator
   - 24h volume: "$1.24M"  with "+12.3% vs avg"
   - 24h high: "$0.001012"
   - 24h low: "$0.000996"
   - All-time high: "$0.001047" with date
   - 7d change: "-0.42%" red
   - 30d change: "+2.31%" green
   - YTD change: "+4.18%" green
   - Open interest: "—" (placeholder for v1.5)

4. PRIMARY CHART (full width, large — ~500px tall)
   Larger version of B2's candlestick chart.
   Timeframe selector with MORE options: 1m / 5m / 15m / 1h / 4h / 1d / 1w / 1M / 1y / all
   Default to 1d.
   Volume bars below.
   Indicators dropdown — show 2-3 placeholder indicators (MA, RSI, BB).
   Use lightweight-charts library.

5. MARKET DETAILS — two columns
   LEFT COLUMN — Description and metadata
     - "About this market" heading
     - 2-3 paragraphs explaining what EAI-IDX is:
       "The 1Trade AI Index represents the unified price of AI 
       inference, computed from observed market data and consumption 
       metrics across text, speech, image, video, and niche credit 
       markets. The index is published daily at 16:00 UTC..."
     - Methodology link "View full methodology →" (routes to B5)
   
   RIGHT COLUMN — Specifications table
     - Settlement type: "Cash-settled in USD"
     - Tick size: "$0.000001"
     - Minimum order: "100 credits ($0.10)"
     - Trading hours: "24/7"
     - Currency: "USD (JPY pricing available)"
     - Fees: "0.50% maker / 1.00% taker (retail tier)" with link to fee schedule

6. RELATED MARKETS (horizontal row of cards)
   - Show 4 related markets: TEXT-SPOT, SPEECH-SPOT, IMAGE-SPOT, VIDEO-SPOT
   - Each card: symbol, current price, 24h change, mini sparkline
   - Click to navigate to that market detail

7. RECENT TRADES & ORDER BOOK PREVIEW (collapsed by default)
   - Expandable section showing simplified order book and trade tape
   - "View full trading view →" button at bottom of section

AESTHETIC NOTES
- Slower and calmer than the trading dashboard
- More generous spacing — 32-48px between sections
- Charts dominate
- Numbers still mono tabular
- Same dark-mode palette as B2

LIVE BEHAVIOR
- Header price ticks every 3-5 seconds
- Main chart updates with new candles every 3 seconds
- Related markets row also updates live

DELIVERABLE
Single HTML file. Same library choices as B2.
Mock data: 1 year of daily candles plus 1 day of 5-min candles for the
default timeframe view.

ACCEPTANCE
- Calmer than B2 (this is research mode, not active trading)
- Still institutional feel
- Numbers prominent
- Chart is the visual hero
- "Trade this market" is the clear primary action
```

---

## B4. Credit wallet

```
Design the 1Trade credit wallet — where users see all their credit 
balances and convert between them.

CONTEXT
Users hold three classes of asset on 1Trade:
  - Cash (USD or JPY)
  - AI Credits (unified) and sub-credits (text, speech, image, video, niche)
  - GPU Credits per tier (H100, H200)

The wallet is the surface where they see all balances, convert between 
them (AI → sub-credit at published rate), and initiate buy/sell flows.

MODE: Dark mode
VIEWPORT: 1440px native

LAYOUT

1. APP CHROME (top bar + sidebar — same as B2)

2. WALLET HEADER (full width)
   Left:
     - "Wallet" page title (h1)
     - Subhead: "Across all credit types and cash balances"
   Right (single big stat):
     - Tiny label: "TOTAL VALUE USD"
     - Huge number: "$10,247.83" mono tabular, display-xl
     - Below: "▲ +$23.41 (0.23%) today"

3. PORTFOLIO ALLOCATION (single row)
   Horizontal stacked bar chart showing % of total value by credit type:
   [Cash ████████ 38%][AI Credits ██████ 24%][Text ██ 8%][Image ███ 12%][H100 ████ 18%]
   With labels above each segment. Color-coded.

4. BALANCE CARDS — grid (3 columns desktop)
   One card per credit type. Featured prominently: AI Credits and Cash.
   Cards in this order:
     a) Cash USD — "$3,891.42" + JPY balance "¥0" + "Add cash →" button
     b) AI Credits (the unified index) — featured card, larger, with mini chart
        "2,425,000 credits" mono large
        "$2,437.13 USD equivalent" smaller
        "+1.05% today" badge
        Mini 24h sparkline
        Actions: [Convert] [Trade] [Send]
     c) Text Credits — "1,200,000" + USD equiv + change + actions
     d) Speech Credits — same pattern
     e) Image Credits — same
     f) Video Credits — same
     g) Niche Credits — same
     h) H100 GPU Credits — "245" + USD equiv + actions
     i) H200 GPU Credits — "0" + "Acquire" prompt
   
   Each card:
     - Sharp corners (radius-sm 2px)
     - Subtle border 1px at 8% opacity
     - Card padding 24px
     - Title in caption-small uppercase
     - Big number mono tabular
     - USD equivalent muted secondary
     - Lock indicator if any credits are in open orders: 
       "🔒 50,000 locked in open orders"
     - Three icon-action buttons at bottom: Convert / Trade / Send

5. CONVERSION DRAWER (slide-out from right)
   Open by clicking [Convert] on any card.
   Inside:
     - "Convert credits" h2
     - From dropdown: "AI Credits — 2,425,000 available"
     - Down arrow
     - To dropdown: "Text Credits"
     - Amount input (big, mono): "1,000"
     - "Available to convert: 2,425,000"
     - Conversion preview card:
       "1,000 AI Credits → 1,196 Text Credits"
       "Rate: 1 AI = 1.196 Text (live)"
       "House spread: 0.5%"
       "Effective rate after spread: 1.190"
     - Submit button: "Convert →"
     - Show this drawer as part of the mockup (semi-open state visible)

6. RECENT MOVEMENTS — last 8 credit transactions
   Compact table:
     Time           Type          Asset          Amount        Balance after
     14:23:47       Trade buy     AI Credits     +5,000        2,425,000
     14:18:23       Trade sell    Text Credits   -250          1,200,000
     13:55:12       Conversion    AI → Speech    -1,500        2,420,000
     ...
   
   "View full history →" link below

AESTHETIC NOTES
- Card-heavy layout but sharp corners (no rounded cards)
- AI Credits card visually emphasized (slightly larger, featured)
- All balances mono tabular
- Color-coding: lime for AI Credits, blue for sub-credits, orange for GPU credits, neutral for cash
- Subtle hierarchy — featured card has slight brand-primary accent border

LIVE BEHAVIOR
- Balances tick in mock real-time as if simulated activity is happening
- Mini sparklines on each card animate on page load (staggered)
- Total value at top updates whenever balances change

DELIVERABLE
Single HTML file. Tailwind CDN.
Chart.js or hand-rolled SVG for sparklines.
Include the open conversion drawer in the mockup.

ACCEPTANCE
- All credit types visible at a glance
- Conversion flow is obvious
- USD equivalent always shown alongside credit count
- Locked-amount indicator works
- Doesn't feel like a crypto wallet (no Web3 vibes); feels like a 
  financial-services account dashboard
```

---

## B5. Index methodology page

```
Design the 1Trade AI Index methodology page — the credibility document.

CONTEXT
The 1Trade AI Index is the headline tradeable instrument and the 
reference price for AI compute. Its methodology is what makes the 
entire venue credible vs. manipulable. This page must read like a 
financial-grade white paper — closer to S&P or MSCI methodology pages 
than a SaaS docs site.

It will be evaluated by: compliance officers, sophisticated traders, 
external auditors, financial journalists, prospective enterprise 
customers' procurement teams.

MODE: Light mode (more readable for long-form)
VIEWPORT: 1440px, with a 720px reading column for body text

LAYOUT — docs-style three-pane

  [   Marketing site nav (top)   ]
  [Left nav  ][   Content    ][TOC]
  [  ~240px ][   ~720px      ][~200px]

LEFT NAV — sticky, scrolls within page
  Section list (current section highlighted):
    OVERVIEW
      - Executive summary
      - Current index value
    METHODOLOGY
      - Constituent inputs
      - Calculation formula
      - Outlier filtering
      - Volume floor
      - Manipulation resistance
    OPERATIONS
      - Publication schedule
      - Methodology versioning
      - Audit and oversight
    DATA
      - Historical prints
      - Constituent transparency
    GOVERNANCE
      - Methodology committee
      - Complaint process
      - Contact

CONTENT AREA — body 720px reading column

  HEADER
    Eyebrow: "— METHODOLOGY"
    Title: "1Trade AI Index Methodology" h1
    Version line: "Version 1.2 · Effective 2026-04-01 · Next review 2026-Q3"
    Download row: 
      "PDF (full whitepaper)" link with download icon
      "JSON (machine-readable)" link with download icon
  
  LIVE INDEX BOX (prominent card, just below header)
    Three columns inside:
      Left: "Current value" tiny label + "$1 = 1,002.4 AI credits" big mono
      Middle: "Last print" + "16:00 UTC, 2026-05-19"
      Right: "Next print in" + "3h 24m" countdown (animated)
    Border 1px brand-accent, subtle.
  
  SECTION 1: EXECUTIVE SUMMARY (~150 words)
    "The 1Trade AI Index is a daily reference price for AI inference,
    expressed as the number of AI credits redeemable per $1 USD. The 
    index is computed from..."
    
  SECTION 2: CONSTITUENT INPUTS
    Subtle heading style, body text below.
    Bullet list of constituent categories (text credit spot, speech, 
    image, video, niche, weighted-average inference cost) with brief 
    description of each. 
    
    Followed by a TABLE — current constituent weights:
      Constituent          Weight    Source                    Last update
      Text credit          0.42      Internal market           14:23 UTC
      Speech credit        0.08      Internal market           14:18 UTC
      Image credit         0.15      Internal market           14:21 UTC
      Video credit         0.05      Internal market           13:55 UTC
      Niche credit         0.10      Internal market           14:10 UTC
      Inference cost avg   0.20      Capacity utilization      14:00 UTC
      TOTAL                1.00
  
  SECTION 3: CALCULATION FORMULA
    "The index value V at print time t is computed as:"
    
    [Render LaTeX math using KaTeX CDN]
    V(t) = trimmedMean₀.₀₂₅,₀.₉₇₅( Σᵢ Pᵢ(t) × wᵢ(t) )
    
    Explanation below: "where P_i is the observed price of constituent 
    i over the calculation window, and w_i is its weight..."
  
  SECTION 4: OUTLIER FILTERING
    Description of 95% trimmed mean approach.
    A small histogram/visualization showing what the trimmed mean does 
    visually (a curve with the tails shaded as "excluded").
  
  SECTION 5: VOLUME FLOOR
    "A valid print requires minimum N observations across the 
    calculation window. If volume falls below this threshold, the 
    methodology committee reviews..."
    
  SECTION 6: MANIPULATION RESISTANCE
    Numbered list of resistance mechanisms:
      1. Cross-validation against multiple input sources
      2. Outlier exclusion via trimmed mean
      3. Volume threshold for valid prints
      4. Surveillance for marking-the-close patterns
      5. Cryptographic audit chain on all prints
  
  SECTION 7: HISTORICAL DATA
    Heading + body intro.
    Full-width chart (chart.js) showing all 365 days of mock index history.
    Hover for daily values.
    "Download full historical data (CSV) →" link
    
  SECTION 8: AUDIT AND OVERSIGHT
    Body: "The methodology is reviewed quarterly by..."
    Audit firm name: [PLACEHOLDER — italicized, in brackets]
    Link to audit reports archive.
    Description of methodology committee.

  FOOTER OF PAGE
    Contact: "methodology@1trade.com · audit@1trade.com"
    "Last updated 2026-04-01 · Methodology v1.2"

RIGHT TOC — sticky scrollspy
  Mini list of sub-sections currently visible
  Active subsection highlighted

AESTHETIC NOTES
- White paper / academic feel
- Numbered sections (1, 2, 3 — not iconographic)
- Body text 16-17px, line-height 1.6+ for readability
- Tables sized for the reading column, no horizontal scroll
- KaTeX or MathJax rendering for the formula (load from CDN)
- Citations footnoted if any references
- Print-friendly: subtle styling that survives PDF export
- NO marketing language. NO emoji. NO icons except for download links.

DELIVERABLE
Single HTML file. KaTeX CDN for math. Chart.js for historical chart.
Tailwind CDN for layout.

ACCEPTANCE
- Reads like an MSCI / S&P methodology document, not a SaaS docs site
- Live index value card is the only "product" element on the page
- Math formula renders correctly
- TOC nav works (clicking takes you to section)
- Tables are readable, properly aligned
- Page could be sent to BlackRock's research team without embarrassment
```

---

# Section C — Tier 2 (supporting screens)

These screens matter but get medium-depth prompts. They share the design system established by Tier 1 screens.

---

## C1. Signup

```
Design the 1Trade signup page.

CONTEXT
Single page where new users open an account. Three account types: 
Trader, AI Company / Engineer, Enterprise.

MODE: Light mode primary (matches marketing site)
VIEWPORT: 1440px

LAYOUT — split-screen
  LEFT (60%): the form
  RIGHT (40%): reassurance panel with live index ticker

LEFT — FORM
  Heading "Open an 1Trade account"
  Subhead "5 minutes. No credit card required for paper trading."
  Step indicator: "Step 1 of 2"
  
  ACCOUNT TYPE — 3 cards side by side, click to select:
    [ Trader ]            [ AI Company ]        [ Enterprise ]
    Chart icon            Code icon              Building icon
    "Trade credits"       "Use credits"          "Multi-user SSO"
    "Paper or real"       "Inference & compute"  "Procurement-friendly"
    Selected card: brand-primary border + tinted background
  
  EMAIL field — full-width, label above
  PASSWORD field — show/hide toggle, strength meter below
  
  "Continue with..." divider
  
  THREE OAUTH BUTTONS — full-width:
    Google logo + "Continue with Google"
    GitHub logo + "Continue with GitHub"
    Microsoft logo + "Continue with Microsoft"
  
  Bottom:
    Checkbox + "I agree to the Terms of Service and Privacy Policy"
    Primary button "Open Account" (full-width)
    Link "Already have an account? Sign in →"

RIGHT — REASSURANCE PANEL (cream background, distinct)
  Top: rotating value prop (one of three, transitions every 6 seconds):
    "Start trading in paper mode immediately — no funding needed"
    "Real-time AI Index, published daily"
    "Free egress on all compute. Always."
  
  Middle: live index ticker (same component from B1)
  
  Below ticker: small chart of 30-day index history
  
  Bottom row: 3 small trust marks
    "SOC 2 Type I path"
    "Audited externally"
    "UBS Japan anchor"

AESTHETIC
- Form is clean, minimal — no decorative chrome
- Right panel is the "selling" half, lively but not distracting
- Buttons sharp-cornered, primary button uses brand-primary

DELIVERABLE
Single HTML file. Tailwind CDN. Make the live ticker on the right work.

ACCEPTANCE
- Three account types clearly differentiated
- OAuth feels first-class (not buried)
- Trust marks visible
- Mobile responsive (panel hides on mobile)
```

---

## C2. KYC light flow

```
Design the trader KYC light flow — multi-step form for paper trading.

CONTEXT
After signup with Trader account type, users complete KYC light (4 steps).
Light KYC is sufficient for paper trading. Real-money requires upgrade 
to full KYC (v1.5).

MODE: Light mode
VIEWPORT: 1440px (form centered, max-width 720px)

LAYOUT
Centered card. Progress dots at top (●●○○ for step 2 of 4).
Two-column body where applicable: form on left, contextual help on right.

SHOW ALL 4 STEPS in your mockup as separate cards stacked vertically
(or use tabs at top of one card to switch between them).

STEP 1 — Personal info
  - Legal first name
  - Legal last name
  - Date of birth (date picker)
  - Country of residence (searchable dropdown)
  - State/province (conditional)
  Right panel: "Why we ask — Required by financial regulations even for paper trading"

STEP 2 — Trading background
  - "Have you traded financial markets before?" (radio: Yes / Some / No)
  - "Years of experience" (dropdown: <1, 1-3, 3-10, 10+)
  - "Are you a professional trader at a firm?" (radio: Yes / No)
  - Conditional firm name field
  Right panel: "This helps us calibrate the right onboarding"

STEP 3 — Trading intentions
  - "Why are you opening this account?" (multi-select checkboxes):
    □ Explore AI compute as new asset class
    □ Hedge my company's AI compute costs
    □ Test the venue before institutional deployment
    □ Personal investing
    □ Other (text field)
  - "Expected monthly trading volume" (dropdown with ranges)
  Right panel: "Optional — helps us serve you better"

STEP 4 — Confirmation
  - Summary card with all data entered
  - Checkbox: "I confirm the above is accurate"
  - Big button: "Open paper trading account"
  Right panel: "Next: we'll fund your account with $10,000 paper credit"

NAVIGATION
- Each step has [Back] and [Continue] buttons
- Continue disabled until required fields filled
- Browser back button works as expected
- Resumable — saves progress

AESTHETIC
- Calm and professional — financial onboarding, not consumer
- No celebratory animations
- After final submit: brief loading state, then route to welcome
- Tabular figures on date inputs

DELIVERABLE
Single HTML showing all 4 step states (stacked or tabbed).

ACCEPTANCE
- Looks like a real broker's onboarding (Interactive Brokers, IBKR, etc.)
- Not like a consumer fintech onboarding (Robinhood, Cash App)
- Required vs optional clearly marked
- Right panel context is helpful, not salesy
```

---

## C3. Portfolio

```
Design the 1Trade portfolio overview page.

CONTEXT
Trader's at-a-glance view: all positions, total P&L, performance over 
time. Different from B2 dashboard — this is reflection/analysis, not 
active trading.

MODE: Dark mode (in-app)
VIEWPORT: 1440px

LAYOUT (single column, max-width 1280px)

1. APP CHROME (top bar + sidebar as in B2)

2. HEADER
   Total account value HUGE (display-xl mono): "$10,247.83 USD"
   Below: 
     Today: "+$23.41 (▲ 0.23%)" semantic-green
     All-time: "+$247.83 (▲ 2.48%)" semantic-green
   Right side: [Deposit] [Withdraw] buttons (greyed in paper mode 
   with tooltip "Available after real-money upgrade")

3. PERFORMANCE CHART (full-width, ~400px tall)
   Account value over time. Default 30d. Toggle: 24h / 7d / 30d / 90d / 1y / all
   Compare-to-AI-Index toggle (overlays index performance for comparison)
   Y-axis: USD value, mono tabular
   X-axis: dates
   Soft line chart (chart.js or hand-rolled SVG), brand-primary line, 
   semi-transparent fill below.

4. POSITIONS TABLE (same as in B2 but expanded)
   Columns: Market, Side, Size, Avg Entry, Current Price, P&L $, P&L %, Mark, Action
   Sortable by every column. Default sort: P&L $ desc.
   Click row to expand: shows entry timestamps, partial fills, fees paid.
   Show ~6 rows worth of mock data (4 open positions + 2 closed for variety)

5. ALLOCATION DONUT CHART (right side, or below table)
   Donut chart: % of portfolio by market
   Hover: highlight slice + show details
   Legend with values

6. ASSET BREAKDOWN BAR
   Horizontal bar split by asset type: Cash / AI Credits / Sub-Credits / GPU Credits

7. RECENT ACTIVITY (small section at bottom)
   Last 10 trades / fills / conversions
   Same row format as D5 trade history
   "View full history →" link

AESTHETIC NOTES
- Calmer than B2 (this is reflection mode)
- Tables are clean and breathable
- Performance chart is the visual hero
- All numbers tabular
- Donut chart uses muted variations of brand colors, not all rainbow

DELIVERABLE
Single HTML file, dark mode, with all sections.
chart.js for performance chart and donut.

ACCEPTANCE
- Account value is the first thing you see
- Performance chart tells story over time
- Positions table is the meat
- Doesn't feel "consumer app cluttered" — feels "broker statement clean"
```

---

## C4. Trade history

```
Design the 1Trade trade history page.

CONTEXT
Complete record of all trades, fills, conversions, deposits, withdrawals.
Doubles as compliance/audit trail.

MODE: Dark mode
VIEWPORT: 1440px

LAYOUT (single column, max-width 1440px — wide table needs space)

1. APP CHROME

2. PAGE HEADER
   "Trade History" h1
   Right: [Export CSV] [Export JSON] [Export PDF compliance report] button group

3. FILTER BAR (sticky on scroll)
   - Date range picker (default: last 30 days)
   - Market dropdown (multi-select, default: All)
   - Type dropdown (Trade / Conversion / Deposit / Withdrawal / All)
   - Side dropdown (Buy / Sell / All)
   - Search by order ID (input field)
   - "Reset filters" link

4. SUMMARY ROW (4 stat cells)
   Total trades: 247
   Total volume: $24,891.34
   Total fees paid: $124.46
   Realized P&L: +$247.83

5. MAIN TABLE — wide, dense
   Columns:
     Date / Time         (default sort desc) - mono
     Type                Trade / Conv / Deposit
     Market              EAI-IDX
     Side                Buy ▲ / Sell ▼ (color)
     Quantity            mono tabular
     Price               mono tabular
     Total               mono tabular
     Fee                 mono tabular, muted
     P&L                 mono tabular, color (if closed)
     Order ID            truncated "ord_abc...123" with copy icon
     [expand row icon]
   
   Show 20 rows of mock data. Mix of trades, one conversion, one deposit.
   
   Row hover: subtle bg tint
   Row expansion: clicking shows full transaction detail inline (fills, audit hash, etc.)
   
   Pagination at bottom: "Showing 1-20 of 247 · [prev] page 1 of 13 [next]"

AESTHETIC
- Dense, data-table aesthetic — Bloomberg-tier
- All numbers tabular
- Row hover affordance, but no decoration
- Color coding subtle (just for side and P&L)
- Export buttons are not loud — they're chrome

DELIVERABLE
Single HTML file. Tailwind CDN. Mock data for 20 rows.

ACCEPTANCE
- Looks like a brokerage statement
- All filters work (or appear to)
- Row expansion shows realistic audit detail
- Export buttons feel "compliance-grade," not "marketing-CTA"
```

---

## C5. Buy credits flow

```
Design the Buy Credits flow — how users add cash to 1Trade and convert it to credits.

CONTEXT
Used by all customer types — trader funding paper account upgrade, AI 
company doing first bulk purchase, etc. Multi-step flow.

MODE: Light mode
VIEWPORT: 1440px (centered, max-width 720px)

LAYOUT
Centered card with progress dots. Show all 3 steps in mockup (stacked or tabs).

STEP 1 — Choose amount and credit type
  Heading: "Buy credits"
  
  Two large currency segments:
    [USD] [JPY] active currency highlighted
  
  Amount input (huge, mono):
    "$ ____" with placeholder
    Quick amounts row: [$100] [$500] [$1,000] [$5,000] [$10,000] [$50,000]
  
  Credit type selector:
    "Receive credits as:"
    Cards in a row:
      [AI Index Credits — featured, default]
      [Cash Balance — hold as cash]
      [Specific Sub-Credit] — opens dropdown for which sub-credit
  
  Live preview card (updates as you change amount):
    "You pay: $1,000.00 USD"
    "You receive: 990,099 AI Index Credits"
    "Effective rate: $0.00101 per credit (live)"
    "Conversion fee: $0 (no fee on bulk purchase)"
  
  [Continue →] button

STEP 2 — Payment method
  Heading: "How will you pay?"
  
  Payment method cards (radio-style selection):
    [ ] Credit / Debit Card (instant, $25 max for first-time)
        Stripe Elements form fields
    [ ] Bank Transfer / Wire (1-3 business days, no limit)
        Wire instructions shown
    [ ] ACH (US only, 3-5 business days)
        Bank account routing fields
  
  Below: 
    Order summary card (small)
    Terms checkbox
    [Submit Payment →] button

STEP 3 — Confirmation
  Big confirmation card
  Success indicator (subtle, not celebratory)
  Order summary
  "Credits will appear in your wallet within 5 minutes (card) or 1-3 days (wire)."
  Order ID and email confirmation note
  Action buttons: [Go to wallet →] [Place first trade →] [Done]

AESTHETIC
- Calm, financial-services flow
- Numbers prominent throughout (you're spending money)
- Quick-amounts row helps friction
- Multi-currency feels first-class (not US-only)
- No high-pressure CTAs ("Buy now!" or "Limited time")

DELIVERABLE
Single HTML showing all 3 steps stacked or as tabs.

ACCEPTANCE
- Feels like a credible financial transaction, not a casual purchase
- JPY option is visible from the start (UBS Japan context)
- Bank/wire option is first-class (enterprise won't use cards for big purchases)
```

---

## C6. Enterprise onboarding

```
Design the Enterprise onboarding flow — for Fortune 500 and frontier labs.

CONTEXT
Enterprise customers don't sign up self-serve. Their flow:
  1. Sales engagement
  2. NDA + scoping call
  3. Master Services Agreement
  4. SAML SSO setup
  5. Sub-account structure
  6. First bulk purchase via wire/ACH

This screen is for AFTER they're under contract — they're being 
onboarded by 1Trade's enterprise sales team or onboarding the 
account themselves with sales support.

MODE: Light mode
VIEWPORT: 1440px

LAYOUT

1. ENTERPRISE TOP BAR (different from standard)
   - "Enterprise" badge next to 1Trade logo
   - Customer org name: "Walmart Inc." (placeholder)
   - Dedicated CSM contact widget: "Your CSM: [name] · [chat icon]"

2. WELCOME HEADER
   "Welcome, Walmart team."
   "Let's get your 1Trade enterprise account set up."

3. ONBOARDING CHECKLIST (vertical, with progress)
   Each item: status icon (○ pending / ◐ in progress / ● done), title, brief description, action
   
   1. ● Master Services Agreement signed — completed 2026-05-15 [View document →]
   2. ◐ SAML SSO configuration — in progress [Configure now →]
        Expandable item shows SSO setup form (next section)
   3. ○ Add team members — pending [Add team →]
   4. ○ Set sub-account structure — pending [Configure →]
        Description: "Organize your account by team, project, or business unit"
   5. ○ Make first credit purchase — pending [Initiate wire →]
        Description: "Minimum first purchase: $50,000 USD (or JPY equivalent)"
   6. ○ Set spending limits and alerts — pending [Configure →]
   7. ○ Review compliance settings — pending [Review →]
        Description: "Audit log retention, data residency, etc."
   
   Right side of each checklist item: [Action button] that opens 
   detail panel inline

4. EXPANDED: SAML SSO CONFIGURATION (sample expansion of item 2)
   Form fields:
     - SAML metadata XML upload
     - OR enter manually: IdP entity ID, SSO URL, certificate
     - Email domain to auto-route: "walmart.com"
     - Attribute mapping (department, role, etc.)
     - "Test connection" button
     - "Activate SSO" button (disabled until tested)

5. ACCOUNT SUMMARY (right sidebar, sticky)
   Sticky card showing:
   - Organization: Walmart Inc.
   - Contract type: 3-year enterprise agreement
   - Total credit purchase commitment: $5M / year
   - CSM: [name] + contact info
   - Support tier: Enterprise Premium
   - SLA: 99.95%

AESTHETIC NOTES
- Less "shiny consumer onboarding," more "B2B procurement portal"
- Checklist-driven (procurement teams expect this)
- Right sidebar reinforces the commercial agreement
- Less playful color, more institutional grey-and-white

DELIVERABLE
Single HTML file. Show the SSO config item expanded for context.

ACCEPTANCE
- Doesn't look like consumer signup (this is post-contract)
- CSM contact prominent
- Checklist drives clear path
- Could plausibly be sent to a Walmart procurement team without embarrassment
```

---

## C7. Team and sub-accounts

```
Design the Team / Sub-accounts management page (for enterprise accounts).

CONTEXT
Enterprise accounts have many users and often need sub-account structure:
  - Org: Walmart Inc.
    - Team: AI Research (50 users, $2M annual budget)
    - Team: Customer Service AI (12 users, $500K budget)
    - Team: Supply Chain ML (8 users, $300K budget)
    - etc.

Each team has its own credit budget, separate consumption tracking, 
own RBAC.

MODE: Light mode (admin dashboard surface)
VIEWPORT: 1440px

LAYOUT

1. APP CHROME with enterprise badge

2. PAGE HEADER
   "Team Management" h1
   Org context: "Walmart Inc. · 4 sub-accounts · 73 total users"
   Right: [+ Add sub-account] primary button

3. SUB-ACCOUNTS TABLE (cards or table view, toggle)
   Default card view: grid of sub-account cards
   
   Each sub-account card:
     - Team name (h3): "AI Research"
     - User count badge: "50 users"
     - Allocated budget: "$2,000,000 / year"
     - Used so far: "$847,210 (42%)"
     - Progress bar showing budget consumption
     - Mini stats: today's spend, last 7d spend
     - Status indicator: ● Active
     - Action menu: View | Edit | Delete
   
   Show 4 sub-account cards in mockup.

4. ORG-LEVEL SUMMARY (top stats row, 4 cells)
   - Total budget allocated: $5,000,000
   - Total used YTD: $1,847,210 (37%)
   - Total users: 73
   - Active in last 7d: 51

5. RECENT USER ACTIVITY (collapsible)
   List of recent user actions across all sub-accounts:
     - Time, user email, action, sub-account
     - "jane.doe@walmart.com placed a bulk credit purchase in AI Research"
     - etc.
   "View full audit log →" link

6. EXPANDED SUB-ACCOUNT (sample, shown below the card grid)
   If user clicks "View" on AI Research sub-account, show detail:
     - Members table (with avatars, name, email, role, last active)
     - Sub-budget breakdown
     - Permissions / RBAC settings
     - Activity feed

AESTHETIC
- B2B admin dashboard energy
- Restrained color palette
- Cards have subtle structure (border, slight shadow on hover)
- Budget progress bars use semantic colors (green when healthy, yellow at 80%, red at 95%)

DELIVERABLE
Single HTML file. Both card grid and one expanded card visible.

ACCEPTANCE
- Procurement / IT leader can understand budgeting at a glance
- User management feels admin-grade, not consumer
- RBAC permissions visible and editable
- Compliance audit log accessible
```

---

## C8. Compute instances list

```
Design the Compute Instances list page.

CONTEXT
Engineer-facing — shows all GPU instances the user has provisioned 
(or has access to). CLI is primary; this web UI is for monitoring 
and quick actions.

MODE: Dark mode (consistent with in-app)
VIEWPORT: 1440px

LAYOUT

1. APP CHROME

2. PAGE HEADER
   "Compute Instances" h1
   Subhead: "GPU rentals for training, fine-tuning, and custom workloads"
   Right: 
     [Provision New →] primary button
     [CLI command] code-styled inline showing "1trade gpu create"

3. ACTIVE INSTANCES TABLE
   Columns:
     Name (e.g., "training-run-2026-05-19")
     Status (● Running / ● Provisioning / ● Stopping / ○ Stopped)
     Type (H100 80GB SXM5)
     GPU Count (mono: 8)
     Region (us-east-1)
     Uptime (mono: 4h 23m)
     Cost so far (mono: $95.68)
     Spend rate (mono: $23.92/hr)
     Actions: [SSH] [Stop] [Logs]
   
   Mock data — show 5 instances:
     - 1 large running cluster (8x H100, "training-run")
     - 2 single-GPU instances ("inference-test", "fine-tune-run")
     - 1 provisioning (just started)
     - 1 stopped (recent)

4. RIGHT SIDEBAR — Resource summary
   Sticky card:
     - Current usage: 11 GPUs (10 running, 1 provisioning)
     - This month cost: $1,847.32
     - Available budget: $13,152.68 of $15,000
     - Budget bar (green)
   
   Below:
     - "Reserved capacity" section
     - Current reservations: 8 H100 (1-year, ends 2027-02-15)
     - "Manage reservations →" link

5. INSTANCE EXPANSION (sample expansion shown)
   Click row → expands to show:
     - Specs (GPU type, RAM, storage, network)
     - SSH access info
     - Live GPU utilization chart
     - Logs preview (last 20 lines)
     - Cost breakdown for this instance

6. EMPTY STATE (also visible)
   "No instances running. Spin one up:"
   Two paths:
     [Via web] button
     [Via CLI] code block: "1trade gpu create --type h100 --count 8"

AESTHETIC
- Engineer-facing — denser, more terminal-like
- Status indicators with semantic dots (small, clean)
- Code-styled inline elements for CLI commands
- Tables sortable, filterable
- Mono font for technical fields (instance names, costs)

DELIVERABLE
Single HTML file, dark mode, with running instances + expanded view + 
budget sidebar.

ACCEPTANCE
- Engineer recognizes this as a developer tool
- CLI command parity is shown
- Cost tracking visible without being annoying
- Action buttons feel terminal-quick, not buried
```

---

## C9. Create instance flow

```
Design the Provision New Instance flow.

CONTEXT
Used from C8 — provisioning a new GPU instance via web UI.

MODE: Dark mode (in-app)
VIEWPORT: 1440px (form centered, max-width 960px to allow side-by-side configuration)

LAYOUT
Modal or page (treat as page).

  HEADER
    "Provision new instance"
    Subhead: "Or use the CLI: 1trade gpu create"
  
  THREE-COLUMN CONFIGURATION (visible all at once, not stepwise)
  
  COLUMN 1 — Hardware
    Section: GPU Type
      Radio cards (vertical):
        [● H100 80GB SXM5] — $2.99/hr — "Recommended for training"
        [  H200 ]          — $3.49/hr — "Larger memory, newer"
        [  Coming: B200 ]   — disabled, "v2 — Sign up for waitlist"
    
    Section: Count
      Slider 1 to 32 with numeric input + presets [1] [2] [4] [8] [16] [32]
      Above 32: "Contact sales for cluster size 32+"
    
    Section: Region
      Dropdown: us-east-1 | eu-west-1 | ap-northeast-1 (Tokyo)
  
  COLUMN 2 — Configuration
    Section: Base image
      Dropdown:
        1Trade ML Stack (default) — PyTorch + CUDA pre-installed
        Custom image — paste image URI
        Bring your own — Docker image URL
    
    Section: SSH access
      Text area: paste SSH public key
      "Or use the CLI to manage keys" link
    
    Section: Networking
      Public IP (toggle, on by default)
      VPC selector (advanced)
      InfiniBand (auto-enabled for >1 GPU)
    
    Section: Persistent storage
      Slider 0 GB to 10 TB
      Cost: "$0.10/GB-month"
  
  COLUMN 3 — Cost preview
    Sticky card (this is the cost calculator)
    
    "Instance summary"
    Configuration recap (live updates):
      8 × H100 80GB SXM5
      us-east-1
      1Trade ML Stack
      1 TB persistent storage
    
    Cost breakdown:
      GPU rate: 8 × $2.99/hr = $23.92/hr
      Storage: 1 TB × $0.10/GB-month = $100/month
      Network: free egress
      Total: $23.92/hr + $100/month storage
    
    Estimated 24h: $574.08
    Estimated 30d: $17,322.40
    
    Available balance: "$13,152.68 budget remaining"
    
    [Provision →] primary button big at bottom

AESTHETIC
- Engineer-friendly, technical
- Cost preview always visible (no surprise billing)
- CLI parity message at top
- Sharp inputs and selectors
- Dark mode

DELIVERABLE
Single HTML file. Show all three columns populated.

ACCEPTANCE
- Engineer can configure an instance in <30 seconds
- Cost preview is honest and detailed
- No fees hidden
- CLI command equivalent is visible
```

---

## C10. Inference playground + model catalog

```
Design the Inference Playground with adjacent Model Catalog.

CONTEXT
Engineers test inference calls against the catalog of curated OSS 
models. Two surfaces in one screen:
  - Model catalog (browse / pick a model)
  - Playground (chat-like UI for testing)

MODE: Dark mode
VIEWPORT: 1440px

LAYOUT — split screen
  LEFT (~30%): Model catalog
  RIGHT (~70%): Playground

LEFT — MODEL CATALOG
  Search bar at top: "Search 24 models..."
  
  Filter chips: [Text] [Speech] [Image] [Video] [Embeddings] [All]
  
  Model cards (vertical scrollable list):
    For each model:
      Name (h4): "Llama 3.3 70B Instruct"
      Provider: "Meta"
      Category tag: "TEXT"
      Per-token price: "$0.55 / $0.79 per 1M (in/out)"
      Context length: "128K context"
      [Try in playground] button
    
    Currently selected model has brand-primary border + tint.
  
  Show 8-10 models in catalog:
    - Llama 3.3 70B Instruct (selected by default)
    - Llama 3.1 8B Instruct
    - DeepSeek-R1 Distill 70B
    - Mixtral 8x22B Instruct
    - Qwen 2.5 72B Instruct
    - Whisper Large v3 (speech)
    - FLUX.1-dev (image)
    - bge-m3 (embeddings)
    - etc.

RIGHT — PLAYGROUND
  Top bar of playground area:
    Selected model: "Llama 3.3 70B Instruct" with version
    Right side: settings cog → opens parameters panel
  
  Parameters bar (collapsed, expandable):
    Temperature slider (0.7 default)
    Max tokens (1000 default)
    Top P, frequency penalty (advanced)
    System prompt text area (expandable)
  
  Chat-style conversation area:
    Mock conversation showing:
      User: "Explain the difference between paper trading and real-money 
      trading in 1Trade's v1 architecture."
      Assistant: [Long response — markdown rendered]
      User: "What KYC level is required for each?"
      Assistant: [Response]
    
    Bottom: chat input
      Big text area with placeholder "Send a message..."
      Submit button or Enter to send
      Below input: cost preview "Estimated cost: $0.005 (~$0.55/M input × 8K tokens)"
  
  RIGHT SIDEBAR (within playground, narrow):
    Response metadata for last message:
      Tokens used (in / out): 8,234 / 1,206
      Latency: 423ms (P50)
      Backend used: vllm-pool-llama-70b-us-east
      Cost: $0.00547
      Request ID: req_abc123 [copy]
    
    "View as cURL" button — opens modal showing the equivalent API call

  CODE EXAMPLE TAB (alternative view, toggleable)
    Show the conversation as code:
      Python SDK example
      cURL example
      JavaScript example

AESTHETIC
- Developer-tool feel (Postman / Insomnia / OpenAI Playground)
- Clear streaming response capability (typewriter effect optional)
- Cost transparency is a feature, not buried
- Code samples first-class

DELIVERABLE
Single HTML file. Live-feeling chat conversation with mock response.

ACCEPTANCE
- Engineer can pick a model and send a message in seconds
- Cost shown before and after each request
- CURL/Python equivalent always available
- Catalog browsing is fluid
- Doesn't feel like consumer ChatGPT — feels like developer tool
```

---

# Section D — Tier 3 (utility screens)

Lean prompts for utility and system screens. They share design system with above; minimal extra context needed.

---

## D1. Login

```
Design the 1Trade login page.

MODE: Light mode (matches signup)
VIEWPORT: 1440px (split screen)

LEFT — minimal form
  "Sign in" h1
  Email field
  Password field with show/hide
  [Sign in] primary button (full-width)
  OR divider
  OAuth buttons (Google, GitHub, Microsoft)
  Below:
    "Forgot password?" link
    "New to 1Trade? Open an account →" link

RIGHT — same reassurance panel as signup with live index ticker

ALSO MOCKUP: 2FA challenge state
  Same layout, but with TOTP input field after password validates:
    "Enter the 6-digit code from your authenticator app"
    6-digit code input
    "Use a backup code instead →" link

AESTHETIC: same as signup — institutional, calm.

DELIVERABLE: Single HTML with login + 2FA state shown.

ACCEPTANCE: Looks like a financial-services login, not consumer.
```

---

## D2. Email verification + welcome

```
Design two related screens:
  A) Email verification waiting screen
  B) First-login welcome (after verification)

MODE: Light mode
VIEWPORT: 1440px (centered cards)

SCREEN A — Verification waiting
  Centered card:
    Email envelope icon (Lucide)
    "Verify your email"
    "We sent a verification link to jane.doe@walmart.com"
    [Resend email] button (cooldown timer if recently sent)
    [Change email] link
    Below: live index ticker (always visible, calming presence)

SCREEN B — First-login welcome
  Big welcome with subtle entry animation:
    "Welcome to 1Trade, Jane."
    Subhead: "Your paper trading account is ready."
  
  Three orientation cards in a row:
    1. "Paper balance" — preview shows "$10,000 USD ready"
    2. "AI Index live" — preview shows "$1 = 1,002.4 credits"  
    3. "Markets open" — preview shows top 3 markets with prices
  
  Below cards: 
    Primary CTA: "Start trading →" big button
    Secondary: "Or take the 60-second tour" text link

DELIVERABLE: Single HTML showing both screens (A and B) stacked.

ACCEPTANCE: Brief warmth in welcome (not consumer-cheerful), trust-building.
```

---

## D3. Audit log

```
Design the Audit Log page (enterprise feature for compliance).

CONTEXT
Every action on the account is logged immutably with cryptographic 
chain. Compliance officers and auditors use this.

MODE: Light mode (admin surface)
VIEWPORT: 1440px

LAYOUT
1. Header: "Audit Log · Walmart Inc."
   Right: [Export CSV] [Export JSON] [Schedule SIEM export]

2. Filter bar (similar to C4):
   Date range, actor (user), action type, resource, sub-account

3. Main table — DENSE
   Columns:
     Timestamp (mono, ISO 8601)
     Actor (user email)
     Sub-account
     Action (verb)
     Resource (object)
     Result (success/fail)
     IP address (mono)
     Audit hash (truncated mono, copy icon, links to verification)

4. Each row clickable for full event detail with cryptographic proof

5. Top-right: "Audit chain integrity: ● Verified · Last check: 14:23 UTC"
   Reassurance that the chain has not been tampered with.

Show 20 rows of realistic audit data (logins, trades, credit purchases, 
SSO config changes, role assignments, etc.)

DELIVERABLE: Single HTML, light mode, dense table.

ACCEPTANCE: Compliance-officer-acceptable. Cryptographic chain visible.
```

---

## D4. Billing dashboard

```
Design the Billing Dashboard.

MODE: Light mode (admin/finance surface)
VIEWPORT: 1440px

CONTENT

1. PAGE HEADER: "Billing · Walmart Inc."
   Current period: "May 2026 · 19 days in"

2. CURRENT MONTH HEADER STATS (4 cards)
   - This month spend: $42,847.23 (mono, large)
   - Projected month-end: $67,800 (extrapolated)
   - Budget: $80,000 (progress bar)
   - Status: ● On track (semantic-green)

3. SPEND BREAKDOWN — pie or horizontal stacked bar
   By service: Compute (60%) / Inference (25%) / Storage (10%) / Trading fees (5%)
   By sub-account: AI Research (52%) / Customer Service AI (28%) / etc.

4. SPEND OVER TIME chart
   Daily spend last 30d (bar chart)
   Compared to last month (line overlay)

5. COST ALERTS section
   Current alerts:
     ● Active: "AI Research approaching 80% of monthly budget"
     ○ Inactive: "Customer Service AI auto-stop at $30K (not yet hit)"
   [Configure alerts →] button

6. INVOICES table (bottom)
   Past invoices: May 2026 (current) / April 2026 / March 2026 / ...
   Columns: Period, Amount, Status (Paid/Pending), Payment method, [Download PDF]

DELIVERABLE: Single HTML, dense but readable, light mode.

ACCEPTANCE: CFO/finance lead can understand spend at a glance. Forecast 
visible. Alerts actionable.
```

---

## D5. DC partner dashboard (internal)

```
Design the Datacenter Partner Dashboard.

CONTEXT
Internal-only in v1 — used by 1Trade ops + DC partners. Shows 
partner's contributed capacity, real-time utilization, payout 
history.

MODE: Dark mode (admin surface, partner-facing in v1.5)
VIEWPORT: 1440px

CONTENT

1. PARTNER HEADER
   "DC Partner: Czech Data Center 1" (placeholder)
   Status: ● Active since 2026-04-12
   Right: contact info, contract terms link

2. CAPACITY OVERVIEW (top stats row)
   - Contributed capacity: 1,024 H100 GPUs
   - Currently utilized: 847 GPUs (83%)
   - Available: 177 GPUs
   - Pricing floor: $2.50/GPU-hour (your minimum)
   - Average sale price: $2.92/GPU-hour (last 7d)

3. UTILIZATION CHART (real-time)
   Time series — last 24h hour-by-hour utilization %
   Highlight peaks and dips
   Live updating

4. REVENUE THIS MONTH
   Big number: $1,234,500 (USD)
   Below: 
     Your share (70%): $864,150
     1Trade fee (30%): $370,350
     Next payout: 2026-06-01

5. PAYOUT HISTORY table
   Recent payouts:
     Period | Capacity sold | Gross | 1Trade fee | Your payout | Status
     Apr 2026 | 562,400 GPU-hr | $1,642,400 | $492,720 | $1,149,680 | ● Paid
     Mar 2026 | ... | ... | ... | ... | ● Paid

6. CAPACITY STATUS (real-time grid)
   Visual grid of 1024 cells, color-coded by status:
     Green: in use (paid)
     Blue: available
     Yellow: maintenance
     Red: offline
   Cluster/rack groupings visible

AESTHETIC
- Operational dashboard feel
- Dark mode
- Heavy on real-time data
- Clear revenue visualization

DELIVERABLE: Single HTML with live-feeling data.

ACCEPTANCE: DC operator can understand utilization and revenue 
without explanation. Trust-building for partners.
```

---

## D6. Partner capacity registration

```
Design the Partner Capacity Registration form.

CONTEXT
Used by 1Trade ops team in v1 to onboard new DC partners. 
Self-serve in v1.5+.

MODE: Light mode (admin tool)
VIEWPORT: 1440px (max-width 960px)

LAYOUT — multi-section single page

1. Partner identification section
   - Partner name (legal entity)
   - Primary technical contact (name, email, phone)
   - Primary business contact
   - Datacenter location (country, city)
   - Operator since (date)

2. Capacity details
   - GPU type (dropdown: H100 / H200 / etc.)
   - GPU count
   - Networking spec (InfiniBand HDR / NDR, bandwidth)
   - SLA commitment (uptime %)
   - Maintenance windows

3. Commercial terms
   - Pricing floor (USD per GPU-hour)
   - Revenue share % to partner
   - Payout currency
   - Payout cadence (weekly / monthly)
   - Bank wire details

4. Capacity verification
   - Test workload status (run automated capacity validation)
   - Status indicators for each verification step:
     ● Network reachability verified
     ● GPU detection (NVIDIA SMI) confirmed
     ◐ NCCL collective benchmark — running
     ○ Final SLA test — pending

5. Activation
   - "Activate this capacity" button
   - Soft-launch option: start with 20% of capacity for 2 weeks
   - Date selector for activation

DELIVERABLE: Single HTML showing form populated with one partner.

ACCEPTANCE: Ops can register a new partner in ~15 minutes. Verification 
flow clear. Commercial terms unambiguous.
```

---

## D7. Settings shell

```
Design the Settings page shell.

MODE: Dark mode (in-app)
VIEWPORT: 1440px

LAYOUT — left nav + content
  LEFT: Settings sections list
    - Profile
    - Account & Security (PASSWORD, 2FA, SESSIONS)
    - API Keys
    - Notifications
    - Display preferences (theme, density)
    - KYC status
    - Billing & Payment
    - Team & Permissions (if enterprise)
    - Compliance & Audit
    - Danger zone (close account)
  
  RIGHT: Active section content (show Profile as default)

PROFILE SECTION content:
  - Avatar upload
  - Name fields
  - Email (verified badge)
  - Phone (with verify)
  - Timezone
  - Default currency
  - Language

DELIVERABLE: Single HTML showing left nav + Profile content.

ACCEPTANCE: Standard SaaS settings, sharp and clean. Easy to find sections.
```

---

## D8. API keys settings

```
Design the API Keys section of settings.

MODE: Dark mode
VIEWPORT: 1440px (within settings shell)

CONTENT

1. Section header: "API Keys"
   Subhead: "Programmatic access to 1Trade APIs"
   Right: [+ Create new key] primary button

2. ACTIVE KEYS TABLE
   Columns:
     Name              "Production trading bot"
     Key prefix        "esx_prod_a47b9c..." (mono)
     Scopes            Badge group: trade · read · market-data
     Created           2026-04-10
     Last used         "12 minutes ago"
     Created by        jane.doe@walmart.com
     Status            ● Active
     Actions           [Edit] [Revoke]
   
   Show 4-5 mock keys with different scopes, ages, last-used times.

3. CREATE KEY MODAL (also visible in mockup)
   - Key name input
   - Scope multi-select: trade, read, market-data, billing, admin
   - Expiration: never / 30 days / 90 days / 1 year / custom
   - "Generate key" button
   - Generated state: shows key once (with copy button), warning 
     "Save this key now — it won't be shown again"

DELIVERABLE: Single HTML with active keys table + create modal visible.

ACCEPTANCE: Developer-friendly. Scopes clear. Security best practices 
visible (one-time display, scope-limiting).
```

---

## D9. Notifications drawer

```
Design the Notifications drawer (slides in from right of any screen).

MODE: Dark mode (in-app)
DIMENSIONS: ~400px wide, full-height drawer

CONTENT

  Header: "Notifications" + filter tabs [All] [Unread] [Trades] [Account] [Alerts]
  Right: "Mark all as read" link + close icon

  NOTIFICATION LIST:
    Each item is a row with:
      - Icon (semantic — order check, alert triangle, etc.)
      - Timestamp (e.g., "2m ago")
      - Title and short description
      - Action link (if applicable)
      - Unread indicator dot
    
    Mock examples:
      ● Just now — "Order filled: Bought 1,000 AI Credits at $0.001005"
        Action: "View trade →"
      ● 5m ago — "Price alert: AI Index up 0.5% in last hour"
      ○ 1h ago — "Budget alert: AI Research sub-account at 80% of monthly budget"
        Action: "Adjust budget →"
      ○ 3h ago — "New API key created: 'Production bot'"
      ○ 1d ago — "Maintenance scheduled: 2026-05-20 02:00 UTC, ~5 min"
      ○ 2d ago — "Welcome to 1Trade! Take the 60-second tour →"

  FOOTER:
    "Notification settings →" link

DELIVERABLE: Single HTML showing the drawer in open state, overlaid on a 
darkened backdrop of the trading dashboard.

ACCEPTANCE: Information-dense but scannable. Action-oriented. Doesn't 
feel like a social media inbox.
```

---

## D10. Command palette (cmd+K)

```
Design the Command Palette (cmd+K) — universal search and action.

MODE: Dark or light depending on surface (show dark)
DIMENSIONS: Centered modal, ~640px wide

CONTEXT
Hit cmd+K from anywhere in the app to:
  - Switch markets
  - Jump to any page
  - Trigger actions (place order, buy credits, etc.)
  - Search documentation
  - Access shortcuts

CONTENT
  Top: search input with placeholder "Search markets, actions, docs..." 
  Below input, results organized by section:

  RECENT (just-used items):
    EAI-IDX AI Index
    Buy credits
    Trade history

  MARKETS (matching query):
    EAI-IDX · AI Index · $0.001005 ▲
    TEXT-SPOT · Text Credit · $0.00120 ▲
    H100-SPOT · H100 GPU Credit · $2.99 ▲

  ACTIONS:
    Place buy order...
    Place sell order...
    Convert credits...
    Buy credits with cash...
    Stop all running instances

  PAGES:
    Trading dashboard
    Portfolio
    Wallet
    Settings

  DOCS:
    API Reference
    Index methodology
    Trading rules

  Each item shows: icon, label, optional shortcut hint (e.g., "G then P" 
  for "Go to Portfolio")

  Bottom hint bar:
    "↑↓ navigate · ⏎ select · esc close"

AESTHETIC
- Familiar pattern (Linear, Notion, GitHub all use this)
- Dense list of results
- Keyboard-driven
- Subtle but clear category dividers

DELIVERABLE: Single HTML with palette open, backdrop showing trading 
dashboard behind it (subtle).

ACCEPTANCE: Power-user feel. Fast and dense. Clear categories.
```

---

## D11. Empty / loading / error states

```
Design the empty, loading, and error state patterns for the app.

DELIVERABLE: Single HTML showing 6 example states stacked or in grid:

1. EMPTY TRADE HISTORY
   Centered:
     Subtle icon (clock with dashes)
     "No trades yet"
     "Your trade history will appear here once you place your first order."
     [Place your first trade →] button

2. LOADING TRADING DASHBOARD (skeleton)
   Show the trading dashboard layout with skeleton placeholders:
     - Skeleton bars for chart (animated shimmer subtle)
     - Skeleton rows for order book
     - Skeleton cells for stats
   Skeleton uses subtle pulse animation, not jarring.

3. EMPTY ORDER BOOK
   Inside the order book panel:
     "No orders at this depth"
     Subtle message, no decoration

4. ERROR: ORDER FAILED
   Toast notification (top-right, slides in):
     Red-tinted left border (semantic-negative)
     Title: "Order failed"
     Description: "Insufficient balance. You need $1,005 USD but have $847."
     Actions: [Buy credits →] [Dismiss]

5. ERROR: PAGE NOT FOUND (404)
   Full page:
     Subtle "404"
     "This page doesn't exist."
     [Go to trading dashboard →] [Open command palette ⌘K]

6. ERROR: NETWORK CONNECTION LOST
   Sticky banner at top of page:
     Yellow-tinted (warning):
     "Connection to 1Trade lost. Reconnecting..."
     Animated spinner inline
     [Retry now] button

AESTHETIC NOTES
- All empty states have a clear next action (the most important part)
- Loading is subtle, never aggressive
- Errors are specific (not "Oops!" or "Something went wrong")
- All states match the surrounding UI's mode (dark/light)

ACCEPTANCE
- Empty states feel like an invitation, not a dead end
- Loading states feel calm
- Errors are blame-free and actionable
- Specific error messages (with actual numbers when relevant)
```

---

# Section E — How to iterate with Claude

A few notes on getting the best output:

### First-pass workflow

1. Paste Section A foundation + one Tier 1 screen prompt into Claude
2. Wait for the artifact to render
3. Read it, screenshot it, share it for review
4. Ask for specific adjustments: "Make the chart 30% taller. Add more 
   density to the order book. Use less brand-primary lime."

### When the output feels wrong

Common feedback patterns that work:

- "This looks too consumer-app. Make it more Bloomberg / institutional."
- "Density is too low. Compress spacing 30%, reduce font sizes by 1-2px."
- "The brand-primary lime is overused. Use it only for active states and CTAs."
- "Numbers aren't tabular. Apply font-variant-numeric: tabular-nums everywhere."
- "Chart looks fake. Generate more realistic mock data with Brownian motion."
- "Hero is too flashy. More restrained — Stripe-tier, not agency-tier."

### When asking for variants

Useful prompts after the first pass:

- "Now generate the same screen in light mode."
- "Show the same screen with a different aesthetic — more Bloomberg, less Linear."
- "Generate the mobile responsive version."
- "Add accessibility annotations (ARIA labels, screen reader notes)."

### When integrating into the real product

When ready to move from mockup to production:

- "Now convert this to a Vue 3 + Nuxt 3 + Tailwind component, with the 
  mock data behind a composable. Match the visual exactly."

That's the bridge from design mockup to code.

---

# Section F — Open questions before final brand book lands

Some design decisions wait on Tai's existing brand book:

1. Final logo and wordmark direction
2. Final primary typeface (Inter is placeholder)
3. Final accent color (lime is placeholder)
4. Final card aesthetic confirmation (sharp recommended)
5. Whether existing brand has photography style requirements

When the brand book arrives, ONE update propagates across all screens: 
swap design tokens in Section A. The screen prompts themselves don't 
need to change — the foundation is what binds them.

---

**End of catalog. 26 screens covered. Ready for iterative design work with Claude.**
