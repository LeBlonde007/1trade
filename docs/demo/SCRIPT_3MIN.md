# Exascale — 3-minute investor video — production script

> **How to use:** record the voice-over **straight through from Part 1** (≈450 words ≈ 3:00 at a
> natural ~150 wpm). Then record the screens and match them to the VO using the **synced shot list
> in Part 2**. Part 3 has pacing/production notes. The exchange is presented as a real, working part
> of the product — no "paused/coming-soon" anywhere. Avoid ad-libbing claims that it's processing
> live real-money trades or is licensed (keep that for live Q&A). Any hard figure you add later is
> illustrative until confirmed.

---

## Part 1 — Voice-over (record this, top to bottom)

**(0:00 — Hook)**
AI compute is the most valuable commodity on earth — and the hardest to buy. There's no public
price for an H100-hour, no order book, no way to hedge a budget. Every commodity that became liquid
— oil, electricity, currencies — made a fortune for whoever built the marketplace. Compute is the
last one without one. Exascale is the world's first exchange and platform for AI and GPU compute —
and the platform that powers it is live today.

**(0:24 — Inference)**
It starts with inference. An AI startup connects to our platform — one API endpoint — and runs a
curated catalog of state-of-the-art models across text, code, speech, image, and video. Pay per
use, metered to the token. A single integration, and they're live. It's the fastest path to the
first dollar.

**(0:46 — Compute + credits)**
The same account rents GPUs on demand — one command to a running H100, or reserved capacity at a
discount. Inference and compute both draw down from prepaid credits the customer buys up front.
Cash in before consumption — and a real-time balance they can actually trust.

**(1:08 — The ledger: the credibility beat)**
And this is what others put on a roadmap — ours is already live. Every credit movement runs through
a ledger that's append-only and cryptographically hash-chained — tamper-evident, exact to the
micro-credit, fully audited. Built, deployed, and verified today. It's the compliance backbone a
CFO signs off on.

**(1:32 — Enterprise)**
For enterprises, that becomes control: single sign-on, sub-accounts with per-team budgets, live
burn meters, cost alerts — and a billing line where every dollar traces back to a hash in the chain.

**(1:52 — Supply)**
Supply comes from our own datacenter as the anchor, plus partner datacenters onboarded into one
pool — monetizing the GPUs that sit idle today. Every served request is attributed, so partners get
paid for exactly what they deliver. Real supply, real demand, real prices.

**(2:12 — The exchange)**
Which is what powers the market. This is the trading floor — a live order book for compute, a market
maker, and a published price index. Standardized credits for tokens, images, and GPU-hours. Traders
get exposure to compute prices the way they trade oil or currencies — and that liquidity makes the
price honest for everyone.

**(2:36 — Why now + trajectory)**
Why now? Demand has outrun supply, hundreds of datacenters have capacity and no buyers, and the
clearing stack is finally commodity software. And this isn't a deck — it's running software: a full
product, a live platform, a verified financial core. The platform turns revenue on the moment
customers consume — and it's the on-ramp to a market we're driving toward a hundred million in
annual revenue and a hundred billion in compute traded.

**(2:54 — Ask + close)**
We're raising to take the platform to general availability, sign our first datacenters and AI
customers, and scale toward that hundred-million-ARR, hundred-billion-volume market. Exascale —
the commodity market for AI compute.

---

## Part 2 — Synced shot list (match screens to the VO)

| Timecode | VO opens with… | On screen (route + action) | Capture note |
|---|---|---|---|
| 0:00 | "AI compute is the most valuable…" | `/` landing — hold on headline + the live index ticker | Let the ticker tick; slow scroll |
| 0:24 | "It starts with inference." | `/inference` — pick a model, type a prompt, tokens stream, credit balance ticks down | Show the live token + credit debit |
| 0:46 | "The same account rents GPUs…" | `/compute/new` (type=H100, count) → `/compute` list (instance spinning up) → `/wallet` balances | Show "running" state + the credit balance |
| 1:08 | "And this is what others put on a roadmap…" | `/enterprise/audit` — click a row → cryptographic detail (block, prev-hash, this-hash) → hover the green chain-integrity badge | This is the hero shot — hold on the hash detail |
| 1:32 | "For enterprises, that becomes control…" | `/enterprise/billing` (burn meter / MTD / projected) → `/enterprise/teams` (budgets) → `/enterprise/sso` | Let the burn meter animate |
| 1:52 | "Supply comes from our own datacenter…" | `/datacenter` dashboard (capacity, fill rate, settlements) → `/datacenter/register` | KPI strip animating |
| 2:12 | "Which is what powers the market." | `/trade` (candles tick, order book flickers, tape rolls) → `/markets` → `/portfolio` (P&L vs index) | The Bloomberg-density money shot; let data move |
| 2:36 | "Why now? Demand has outrun supply…" | Montage: a rising token-volume chart → the running app → a terminal showing the deployed service + green tests → a trajectory slide ("$100M ARR · $100B GMV — target") | "Running software, not a deck" → then the trajectory; label the numbers **target** |
| 2:54 | "We're raising to take the platform…" | Brand-mark end card | Hold ~3s over soft ambient track |

**End card:**
```
        E X A S C A L E
─────────────────────────────────────
The commodity market for AI compute.

exascale.com · raising [round]
```

---

## Part 3 — Production notes

- **Pace:** ~150 wpm. The 9 VO blocks are timed to land at ~3:00 with natural pauses between
  sections. If a block runs long, trim the *adjectives*, not the *nouns*.
- **Tone:** confident, concrete, institutional. No "revolutionary / cutting-edge / AI-powered."
- **Capture:** run `make web` → `http://localhost:3000` (mock-data mode — candles tick, burn meters
  move). Record at **1440px**. Dark theme for the product/trade screens, light for enterprise admin.
- **Easiest capture route:** drive the persona tour at `/onboarding/tour` (AI Company → Datacenter
  → Trader) — it sequences most of these screens for you.
- **Numbers / revenue framing:** the only figures in the VO are the **$100M ARR** and **$100B GMV
  (trading volume)** — say them as **forward-looking targets**, not current results. We're pre-revenue
  today; the framing is "the platform is live and turns on revenue as customers consume, and it's
  the on-ramp to" those targets. On screen, label them **"target" / "projection"** (never a current
  KPI). Everything else in the VO avoids hard figures.
- **Two assets from one script:** this VO also makes a clean ~3-min audio-only overview for
  warm-intro emails.
- **Music:** one restrained ambient/tech bed, low under VO, slight lift at 2:12 (the exchange) and
  the close.
