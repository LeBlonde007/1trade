# Exascale — Persona Demo Flows

Source of truth for shooting three persona walkthrough videos. Each flow is sequenced screen-by-screen with the exact route, the on-camera action, and a voice-over cue so a single take per persona is possible. Routes are clickable in a running `npm run dev` instance.

- **Trader / Quant** — the default user. Public marketing → signup → KYC → trading floor.
- **Enterprise Buyer** — procures compute in bulk. Custom onboarding → team setup → compute + inference.
- **Datacenter Partner** — sells capacity into the venue. Partner registration → partner dashboard.

A fourth optional sliver — **Internal / Admin** — reuses the partner dashboard and the design-reference page; it is included as an addendum.

> **Three screens are still missing for full coverage.** See the *Missing screens* section at the bottom before recording.

---

## Global navigation that every persona uses

| Surface | Mounted in | Entry |
|---|---|---|
| Marketing nav | `marketing` layout | `/`, `/benchmark` |
| App sidebar (8 icons + Settings) | `app` layout | every `/trade`, `/portfolio`, `/wallet`, etc. screen |
| App topbar (index ticker + balance + 🔔 + avatar) | `app` layout | same |
| Command palette (⌘K) | `app` layout, global | any in-app screen |
| Notifications drawer (🔔) | `app` layout, global | any in-app screen |

The command palette is the keyboard-driven shortcut for the whole demo — pre-show it on the *first* persona to introduce it, then re-use it freely afterwards.

---

# Persona 1 · Jordan Park — Independent quant / retail trader

> Trades AI compute the way they used to trade FX. Open laptop, scan the index, fire an order, manage risk, log out.

**Demo goal:** A new visitor lands on the homepage, opens an account, completes KYC, and is trading inside two minutes.

**Runtime target:** 3:30 – 4:00

### Flow

1. **Landing — `/`** *(light · marketing)*
   - Camera holds on the hero. Hover the live AI Index price in the footer status bar (auto-ticks every 5s) to establish *this is a real market*.
   - Scroll: hero → markets grid → AI Index card → products row → audiences split → final CTA.
   - VO: *"Exascale is the commodity market for AI compute. One credit equals one dollar of work — whether that's tokens, images, GPU-hours, or video frames."*
   - **Click:** `Open account` (top-right or hero CTA).

2. **Signup — `/signup`** *(light · standalone)*
   - Fill email + password. Strength meter live-updates.
   - VO: *"Standard account opening — email, strong password, agree to terms."*
   - **Click:** `Continue`. Navigates to `/onboarding/kyc`. *(Note: a `D2 email-verify + welcome` interstitial belongs here but doesn't exist yet — see Missing screens.)*

3. **KYC — `/onboarding/kyc`** *(light · standalone, 4 steps)*
   - Step 1: Name + DOB + country (combobox is real, type "Un" → United Kingdom / United States).
   - Step 2: Address.
   - Step 3: Tax residency + investor-type radio.
   - Step 4: Review → `Submit`. Success state shows green check + *"You're cleared to trade."*
   - VO: *"Light-touch KYC because trading credits — not securities — keeps the venue under commodity, not securities, rules."*
   - **Click:** `Continue to trading →` → `/trade`.

4. **Trading dashboard — `/trade`** *(dark · app layout)*
   - First-time, focus on the chrome:
     - Topbar: `EAI-IDX` selector, live price, USD balance pill, 🔔 bell with **2** badge.
     - Sidebar icons: Trade · Markets · Index · Portfolio · History · Wallet · Compute · Inference.
   - Camera pans across: candle chart → order book → tape → order form → positions footer.
   - VO: *"Bloomberg-tier density. Real-time book, real-time tape, and order entry against the same instrument an institutional desk sees."*
   - Place a $1,000 market buy on EAI-IDX. Fill toast appears top-right (and increments the bell badge to 3).
   - **Press `⌘K`** — introduce the command palette here. Type "port" → highlight `Portfolio · G P`. Press **Enter**.

5. **Portfolio — `/portfolio`** *(dark · app layout)*
   - Hero card with total P&L + benchmark-relative bar (vs AI Index).
   - Performance chart with **1D · 1W · 1M · 3M · YTD · ALL** range tabs and a dashed *AI-INDEX* overlay (toggle on once on camera).
   - Sortable positions table (5 rows), allocation donut, asset-class bar, recent activity feed.
   - VO: *"Performance attribution against the index is built-in — you always know whether you're beating beta or just riding it."*
   - **Click** the **🔔 bell** in the topbar.

6. **Notifications drawer** *(global · 400px right drawer)*
   - Six mock notifications appear: Order filled, Price alert, Budget alert, API key created, Maintenance scheduled, Welcome.
   - Click `Mark all as read` — badge clears.
   - VO: *"Trade fills, price alerts, account events — everything lands here, nothing in email."*
   - **Close** drawer (Esc).

7. **Wallet — `/wallet`** *(dark · app layout)*
   - Three credit balances (cash, AI-IDX, sub-credits). Recent ledger rows.
   - VO: *"Cash on one side, credits on the other. Conversion is one click at the venue rate."*
   - **Click:** `Buy credits` primary action.

8. **Buy credits — `/wallet/buy`** *(dark · standalone funnel)*
   - 3-step: amount → payment method (card / wire / ACH) → confirm.
   - Live receipt panel updates as you type the amount.
   - VO: *"Settles into your trading account in under a minute on card, T+1 on wire."*
   - Hit `Save & exit` → returns to `/wallet`.

9. **(Optional · 20s) Methodology — `/benchmark`** *(light · marketing)*
   - Open in a fresh tab to show the AI-INDEX whitepaper.
   - Side-nav TOC + formula block + historical chart. Scroll once.
   - VO: *"Every constituent, weight, and audit pointer is published. The index is a public artifact."*

10. **End** — return to `/trade`, hold for one second on the live ticker, fade out.

### Optional detours
- `/markets/eai-idx` — full market detail page (depth · related markets · methodology blurb).
- `/history` — full trade history with summary tiles and CSV/JSON export.
- `/settings` (hash `#api`) — show the API Keys section, create-key modal, one-time-reveal flow.

---

# Persona 2 · Maya Chen — VP Engineering at an AI startup (Enterprise Buyer)

> Procures compute at scale. Cares about budget control, team seats, audit, and uptime — not chart patterns.

**Demo goal:** Sales-assisted onboarding → activate enterprise account → invite team → provision compute → run inference. Shows that *the same platform serves a trader and a CFO*.

**Runtime target:** 4:00 – 4:30

### Flow

1. **Landing — `/`** *(light · marketing)*
   - Scroll to the **For traders / For AI companies / For datacenters** audience split. Click `Talk to enterprise sales →` (visual — link is currently a `#`).
   - VO: *"Enterprise procurement isn't a self-serve flow — sales runs a guided activation."*
   - **Type URL** or `⌘K` to: `/enterprise/onboarding`.

2. **Enterprise onboarding — `/enterprise/onboarding`** *(light · custom)*
   - Header shows the assigned CSM with email / Slack / phone CTAs. *(Personalised: company name, MSA reference, order form, contract value.)*
   - Six-step activation checklist:
     1. ✅ Master service agreement signed
     2. ✅ Initial credit purchase
     3. ✅ Billing entity verified
     4. ◐ Identity provider connected (SSO/SAML)
     5. ☐ Invite team
     6. ☐ Provision first compute instance
   - VO: *"This is the contract dashboard. Every box that gets checked is a chunk of usage unlocked."*
   - Right rail: legal documents (MSA, pricing schedule, DPA, order form).
   - **Click:** `Invite team` (or sidebar item *Team & sub-accounts*).

3. **Team & sub-accounts — `/enterprise/teams`** *(light · custom)*
   - Roster of 8 members with role chips (Owner · Admin · Trader · Viewer).
   - Per-member monthly budget bars; org-level spend tile.
   - VO: *"Per-seat budget caps and role-based scopes — finance keeps the lid on, engineering still ships."*
   - Click `Invite member` → modal with role + budget + scopes.
   - **Click:** breadcrumb back to `/enterprise/onboarding`, tick *Invite team* as done.

4. **Compute instances list — `/compute`** *(dark · app layout)*
   - 8 mock instances across regions (US-EAST, US-WEST, EU-CENTRAL, AP-SOUTH). Status column with `RUNNING`, `PROVISIONING`, `STOPPED`.
   - **Per-GPU utilization** mini-bars per row (8 GPUs × % bar each).
   - Total org spend tile and headroom tile in the chrome.
   - VO: *"Every instance in one view — provisioning state, real-time utilization, and what it's costing."*
   - **Click:** `+ New instance`.

5. **Provision new instance — `/compute/new`** *(dark · standalone funnel)*
   - Picker: GPU type (H100 · H200) → count → region → image → SSH key → budget cap.
   - Live price calculator (top-right) updates on every change.
   - Sidebar tips: *"Reserve capacity through the index for ~12% lower rate."*
   - VO: *"You can hit the spot market or pre-purchase via the index. The platform is the same either way."*
   - **Click:** `Launch instance` → confirmation toast, returns to `/compute`.

6. **Inference playground — `/inference`** *(dark · app layout)*
   - Catalog of 10 models, tagged by category (text / speech / image / video / embed).
   - Center: playground with system prompt + user prompt + streaming output (mock typewriter cursor).
   - Right: metadata sidebar — tokens, cost, latency, model card.
   - Top tabs: `Playground · Code` — switch to **Code** to show Python / cURL / JS snippets that hit the same endpoint with a real API key.
   - VO: *"Same credit, two doors. Buy on the trading floor or burn it through the API — the meter is identical."*
   - The bottom hint links to `/settings#api`.

7. **Settings → API Keys — `/settings#api`** *(dark · app layout)*
   - 5 mock keys table with color-coded scopes.
   - Click `+ Create key` → modal: name + scopes + expiry → generated state with one-time `sk_live_…` reveal and copy button.
   - VO: *"Keys are scoped, expiry-bound, and revealed once. Audit log records every issuance."*
   - **Click:** `Audit log →` link. *(Note: this currently leads nowhere — the audit log page does not exist yet. See Missing screens.)*

8. **(Optional · 30s) Notifications** — open the bell to show the **Budget alert** card: *"AI-Research-Team has used 78% of June budget."* Demonstrates the loop closes.

9. **End** — fade out on `/inference` mid-stream.

### Optional detours
- `/wallet` — same wallet UI, just with a much larger balance.
- `/portfolio` — for the firm-trading-account version of the enterprise.

---

# Persona 3 · Tom Reyes — Capacity Operations, Datacenter Partner

> Sells GPU-hours into the venue. Cares about fill rate, dispatch, and getting paid.

**Demo goal:** Register a new datacenter as a venue supplier, then drop into the partner dashboard to see the same capacity earning revenue.

**Runtime target:** 2:30 – 3:00

### Flow

1. **Landing — `/`** *(light · marketing)*
   - Scroll to the audience split. Hover the **For datacenters** card — *"Sell capacity into the index."*
   - VO: *"On the other side of every trade is a datacenter shipping electrons. Here is what they see."*
   - **Type URL** directly: `/datacenter` (no nav link exposes this — it's a partner-portal URL).

2. **Partner dashboard — `/datacenter`** *(light · custom partner portal)*
   - Header: partner identity (e.g. "Northstar DC · Reno NV · partner ID `pdc_7c2a…`"), CSM contact buttons, contract-terms link.
   - **Headline KPIs:** capacity sold this month · fill-rate % · payout pending · next settlement date.
   - **Capacity grid:** racks × clusters with status (`LIVE · MAINT · OFFLINE`) and utilization.
   - **Recent orders table:** which credit families consumed their capacity (text, image, GPU-hours).
   - **Statements:** monthly settlement statements, downloadable.
   - VO: *"The supplier view: what's online, what's earning, what just settled."*
   - **Click:** `+ Register partner` (top-right CTA).

3. **Register partner capacity — `/datacenter/register`** *(light · standalone form)*
   - Multi-step form:
     1. Org details (legal name, jurisdiction, primary contact).
     2. Site details (region, PUE, certifications: SOC 2 / ISO 27001 / Tier III).
     3. Capacity declaration (GPU type, count, available windows).
     4. Pricing floor + auto-throttle rules.
     5. Banking / payout (mock fields).
     6. Review → `Submit for review`.
   - VO: *"Onboarding a datacenter is not a credit card and a checkbox — it's a documentation pack and a venue review."*
   - **Click:** `← Cancel` → back to `/datacenter`.

4. **(Optional · 30s) Statement detail** — click a row in the Statements table — *"Download May 2026 →"* — show that a settlement CSV is the artifact of every fill.

5. **End** — fade out on the partner dashboard KPIs.

> The datacenter portal is **not exposed from the marketing nav or the trader sidebar by design.** It is a separate audience. Record the URL on screen as part of the VO if needed.

---

# Persona 4 (addendum) · Internal — Design / Engineering reference

Not a customer-facing video. Useful for a *behind-the-scenes* / engineering reel.

1. **States — `/states`** — the six canonical empty / loading / error patterns. VO: *"Every state in the app is a designed surface, not a default browser fallback."*
2. **Command palette demo** — show every chord (`G T`, `G P`, `G W`, `G M`, `G H`, `G B`, `G S`) and the live filter.
3. **Notifications drawer** — All / Unread / Trades / Account / Alerts filter chips.

---

# Missing screens — record these *before* shooting, or you'll have to cut

| ID | Screen | Why it's needed | Currently |
|----|--------|-----------------|-----------|
| **D2** | Email verification + welcome | Persona 1 flow jumps `/signup` → `/onboarding/kyc` with no email-verify step. Demo VO can hand-wave this, but a real customer needs the gate. | **No page exists.** `/signup` `await navigateTo('/onboarding/kyc')` ships them straight in. |
| **D3** | Audit log | Persona 2 (Enterprise) ends a beat on `Settings → API Keys → Audit log →` and the link goes to `#`. Compliance is *the* enterprise pitch. | **No page exists.** Linked from `/settings` API Keys footer as `<a href="#">Audit log →</a>`. |
| **D4** | Billing dashboard | Persona 2 has no billing screen. Enterprise prospects ask *"where do I see invoices and credit consumption?"* on first call. Currently the answer is *"it's in /wallet"* — which conflates trading cash with subscription billing. | **No page exists.** No `/billing`, `/wallet/billing`, or `/settings/billing` route. |

**Also worth fixing for clean demo takes:**

- **Marketing nav `Docs` and `About` links** point to `#` (dead). If the camera lingers on the nav, this is visible. Either point them at `/benchmark` and a placeholder `/about`, or remove them for the demo build.
- **Marketing footer** — every link is `href="#"`. Five columns × ~5 links = ~25 dead links. Fine if camera doesn't dwell on it.
- **`/login` page** — `Forgot password?`, `Sign in with SSO`, and `Lost your authenticator?` all dead. The 2FA step is mocked (any 6-digit code passes).
- **`/enterprise/teams` brand link** goes to `/` (marketing home). Should go back to `/enterprise/onboarding` to keep the enterprise persona in their own context. One-liner fix.
- **`/datacenter` is unreachable from any nav** — only direct URL or command palette. Command palette currently does **not** list it. Either add `Datacenter partner portal` to the palette `Pages` group, or accept it as URL-only (it's a different audience anyway).
- **No "logout" affordance** — the avatar button in the topbar is a no-op. If the demo VO says *"and out"* you have nowhere to click. Either wire avatar → menu with `Sign out`, or skip that beat.

---

# Per-persona screen index (cheat sheet)

| # | Screen | Route | Persona uses it |
|---|--------|-------|-----------------|
| B1 | Landing | `/` | 1 · 2 · 3 |
| B2 | Trade | `/trade` | 1 |
| B3 | Market detail | `/markets/[slug]` | 1 (optional) |
| B4 | Wallet | `/wallet` | 1 · 2 (optional) |
| B5 | Methodology | `/benchmark` | 1 (optional) · 2 (optional) |
| C1 | Signup | `/signup` | 1 |
| C2 | KYC | `/onboarding/kyc` | 1 |
| C3 | Portfolio | `/portfolio` | 1 |
| C4 | Trade history | `/history` | 1 (optional) |
| C5 | Buy credits | `/wallet/buy` | 1 |
| C6 | Enterprise onboarding | `/enterprise/onboarding` | 2 |
| C7 | Teams & sub-accounts | `/enterprise/teams` | 2 |
| C8 | Compute instances | `/compute` | 2 |
| C9 | New instance | `/compute/new` | 2 |
| C10 | Inference playground | `/inference` | 2 |
| D1 | Login | `/login` | (return-user variant of 1 or 2) |
| D2 | Email verify | *(missing)* | 1 |
| D3 | Audit log | *(missing)* | 2 |
| D4 | Billing dashboard | *(missing)* | 2 |
| D5 | DC partner dashboard | `/datacenter` | 3 |
| D6 | Register capacity | `/datacenter/register` | 3 |
| D7 | Settings shell | `/settings` | 1 · 2 |
| D8 | API keys | `/settings#api` | 2 |
| D9 | Notifications drawer | (global) | 1 |
| D10 | Command palette | `⌘K` (global) | 1 (introduces it) |
| D11 | States reference | `/states` | 4 (internal) |

---

# Suggested recording order

1. **Persona 3 first** (shortest, fewest dependencies, sets supply-side context).
2. **Persona 1 second** (longest, introduces the most chrome — palette, notifications, ticker).
3. **Persona 2 last** (re-uses palette + notifications, so the audience is already trained on them).
4. (Optional) **Persona 4** — engineering reel.

If recording one continuous demo instead of three, run **1 → 2 → 3** in that order. The narrative is *demand-side first, supply-side last*.
