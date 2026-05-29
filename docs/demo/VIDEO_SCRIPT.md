# Exascale — Investor Demo Video Script (current state)

> Platform-first cuts, grounded in the real UI (`UI_WALKTHROUGH.md`) and the narrative
> (`INVESTOR_NARRATIVE.md`). VO in *italics*; [camera/action] in brackets. Mock numbers — replace.
> Two cuts: **90s** (cold outreach) and **3-min** (warm intro / follow-up). Use as the screen-record
> storyboard, and as a NotebookLM source so the generated overview follows this arc.

---

## Cut 1 — 90 seconds

[Open on `/` landing, hold on the headline + the live index ticker.]

**VO (0:00):** *"AI compute is the new oil — but it trades like real estate: no public price, no
order book, no way to hedge. Every commodity that became liquid made a fortune for whoever built
the venue. Compute is the last one without it."*

[Cut to `/inference` — pick a model, run a prompt, tokens stream, a credit debit ticks down.]

**VO (0:12):** *"So we're building the venue — but we're earning revenue on the way there. Today,
an AI startup points the OpenAI SDK at one base URL and runs inference on a curated catalog of
state-of-the-art models. Pay per use."*

[Cut to `/compute/new` → `/compute` list showing an H100 box spinning up.]

**VO (0:24):** *"Same account, on-demand GPUs — one command to a running box. Inference and
compute, paid for in prepaid credits. No license required, because a credit is a service unit,
not a security."*

[Cut to `/wallet` balances, then `/enterprise/audit` — hover the green "chain integrity verified" badge.]

**VO (0:36):** *"Every credit movement runs through a ledger that's append-only and
cryptographically hash-chained — and this is live, deployed, and verified, not a mock. It's the
compliance backbone a CFO and, later, a regulator will sign off on."*

[Cut to `/datacenter` partner dashboard — capacity, fill rate, settlements animating.]

**VO (0:50):** *"On the supply side, datacenters monetize idle GPUs through one scheduling pool —
that's the real supply that powers the market."*

[Cut to `/trade` — candles tick, order book flickers, the tape rolls.]

**VO (1:02):** *"And this is where it leads — a full exchange for AI compute: a live order book, a
market maker, a published price index. Built on the real supply, demand, and prices the platform
underneath it generates."*

[Pull back to the brand mark / end card.]

**VO (1:18):** *"Exascale. The commodity market for AI compute."*

[End card: E X A S C A L E · the commodity market for AI compute · raising [seed] · exascale.com]

---

## Cut 2 — 3 minutes (adds depth to the 90s arc)

### + The problem, stated (after the opening, ~30s)
[Slowly scroll the three-audience split on `/`.]

**VO:** *"If you're an AI startup buying compute today, you get one quarterly price from a
hyperscaler, take it or leave it — or a six-week procurement deal with a tier-3 datacenter. Most
overpay by 25–40% because there's no comparison shop. Datacenters lose money on capacity that sits
dark. Both sides need a market — but a market needs a credible underlying first."*

### + Why platform-first (the investable insight, ~25s)
[Static number/sentence card on dark background.]

**VO:** *"Every compute-trading attempt before us died on the same three things: a license,
liquidity, and a credible underlying. So we inverted the order. Build the platform first — it earns
revenue immediately and produces the supply, demand, and price data the exchange needs to be real.
The platform is a standalone business. The exchange is the venture-scale upside on top of it."*

### + Deeper platform walkthrough (~45s)
[`/inference`: run a model; show the per-call token + credit debit. Then `/compute/[id]`: instance
detail. Then `/wallet/buy`: prepaid purchase. Then `/enterprise/billing`: MTD burn + projected
month-end + cost alerts. Then `/enterprise/teams`: sub-account budgets.]

**VO:** *"This is the product an AI startup uses on day one. Inference metered to the token,
compute metered to the second, both drawn from prepaid credits. Enterprises get sub-account
budgets, live burn meters, cost alerts — and a billing line every dollar of which traces back to a
hash in the audit chain."*

### + The financial core is real (~30s) — the credibility beat
[`/enterprise/audit`: click a row → cryptographic detail (block, prev-hash, this-hash, algorithm).]

**VO:** *"Most companies at this stage have this on a slide. Ours is running. The credit ledger is
append-only, hash-chained, exact fixed-point money math, atomic, idempotent, per-tenant isolated —
deployed to a Kubernetes cluster and verified end-to-end. A purchase debits, the balance updates,
the chain re-verifies. That's the part that has to be bulletproof before anyone touches real
money, and it already is."*

### + Datacenter supply (~20s)
[`/datacenter` dashboard → `/datacenter/register`.]

**VO:** *"Supply comes from our own datacenter as the credible anchor, plus partner DCs onboarded
into one scheduling pool — so we scale capacity without owning every GPU, and every served request
is attributed for partner payout."*

### + The exchange, and the ask (~25s)
[`/trade` + `/portfolio`. Then brand-mark close.]

**VO:** *"This is the exchange: the order book, the market maker, and the price index that becomes
the reference rate every compute contract prices against. The round funds the platform to GA, the
first datacenter partners and AI-startup customers, and the path to the open market. Exascale —
the commodity market for AI compute."*

---

## On-screen / production notes
- Capture at 1440px, dark app + light admin as the screens dictate; let live data animate.
- The video presents the exchange as a real, working part of the product — no "paused" or
  build-status caveats on screen. (Don't add VO claiming it's processing live real-money trades or
  is licensed; keep operational/regulatory status for live Q&A.)
- Replace every mock number (overpay %, market sizes, the ask) before sending to real investors.
- If recording live: `make web` → walk the persona tour at `/onboarding/tour` in the order above.
