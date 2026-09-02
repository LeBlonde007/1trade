# 1Trade — v1.5 Screens & UIs (post-MVP prompts catalog)

> Companion to `1trade_mvp_screens_prompts.md` (the original 26-prompt MVP set)
> and `1trade_mvp_screens_checklist.md` (status tracker — currently 24 / 26 done).
>
> This file lists the **next batch of screens the product still needs** to feel
> complete after the MVP. Each entry is a copy-pasteable Claude prompt in the
> same format as the original B/C/D tier briefs.
>
> ### How to use
> 1. Open a fresh chat in the 1Trade claude.ai project (it has A1–A4
>    foundation files attached already — design tokens, mock data shape,
>    aesthetic guardrails, artifact instructions).
> 2. Pick one prompt from below, paste it as-is. Do **not** re-paste A1–A4 —
>    the project files cover them.
> 3. Iterate on the artifact until it's right.
> 4. Port to Vue under `app/pages/...` matching the project conventions:
>    sharp 2px radius, Inter Tight / Inter / JetBrains Mono, tabular-nums
>    on every number, semantic green/red paired with ▲ / ▼, `layout: 'app'`
>    for dark in-app pages, `layout: false` for standalone funnels.
>
> ### Status convention
> `[ ]` not started · `[~]` partial / stub exists · `[x]` shipped.
> Match the parent checklist style.

---

## Tier E — Trading depth (highest demo / sales value)

- [ ] **E1. Order detail view**
  MODE: Dark (in-app).
  VIEWPORT: 1440px, slide-in panel 480px wide from the right over /trade.
  LAYOUT
  1. Header: order id `ord_8c2a48f1` mono · status chip (FILLED / WORKING / CANCELLED / REJECTED) · close X.
  2. Timeline (vertical): placed → routed → partial fill → fully filled, each step timestamped to ms.
  3. Fills table — dense rows: qty, px, ts (HH:MM:SS.mmm), fee, counterparty hash (truncated mono).
  4. Order params block: side, type, limit / stop / TIF, sub-account, originating IP, originating client (web / API / palette).
  5. Matching telemetry: 5 routed venues, slippage vs mid, queue position, time-on-book.
  6. Actions: Cancel / Replace (only when status allows). Reduce-only flag if margin trading.
  Bottom: "View in /history →" link with the order id pre-filtered.
  ACCEPTANCE: every order from /history or the /trade positions footer is one click from this view.

- [ ] **E2. Watchlist / favorites**
  MODE: Dark, in-app at `/watchlist`.
  CONTENT
  - Top-of-page tabs for multiple watchlists: "Core", "GPUs", "Speculative", "+ New".
  - Dense live table: pin · symbol · last · 24h Δ · spread bps · top-of-book depth · 60-min sparkline · vol · alert bell.
  - Drag-to-reorder rows. Right-click row → "Move to list", "Set price alert", "Open in /trade".
  - Empty state: "Pin markets from /trade or /markets/* to build your watchlist." [Browse markets →]
  - Add to sidebar between Markets and Index.
  ACCEPTANCE: every cell updates live without jitter; pinning from /markets/[slug] immediately appears here.

- [ ] **E3. Heat map**
  MODE: Dark, full-bleed at `/markets/heat`.
  CONTENT
  - Single screen, minimal chrome (just top breadcrumb + filter chips).
  - Grid of every credit market, **sized by 24h volume**, **colored by % change** (ramp from `--pos` saturated → neutral → `--neg` saturated).
  - Hover → tooltip: symbol, last px, Δ%, vol, spread.
  - Click → /markets/[slug].
  - Filter chips along the top: all · text · speech · image · video · GPU · index.
  - Footer micro-legend: -5% ━━━━ 0 ━━━━ +5% with the color ramp.
  ACCEPTANCE: at a glance, you see which families are hot/cold today. No labels inside tiles smaller than 60px.

- [ ] **E4. Depth chart (cumulative volume)**
  MODE: Dark, lives inside /markets/[slug] as a tab.
  CONTENT
  - Segmented control on the market detail page: `[ Book | Depth | Trades ]`.
  - Mirrored area chart: bids on the left in `--pos-soft`, asks on the right in `--neg-soft`, mid line at center.
  - Hover crosshair: price · cumulative size · estimated cost-to-take.
  - Realistic power-law depth per A2 (best bid/ask 5K-50K, deeper levels growing).
  - "Walk the book" widget below chart: enter a buy/sell quantity → highlights consumed levels + computes effective price.

- [ ] **E5. Position detail + close flow**
  MODE: Dark drawer launched from /portfolio positions table.
  CONTENT
  - Header: instrument + side chip + size.
  - Body: avg cost, current value, unrealized P&L, realized P&L (lifetime), open lots breakdown (FIFO), 30-day strip of related fills.
  - Footer actions: **Close position** (market) · **Reduce** (size slider 0-100%) · **Convert to limit** (modal pre-filled) · **Set stop**.
  - Confirmation modal before each destructive action — show estimated P&L crystallization in plain English.
  ACCEPTANCE: closing a position from here updates /portfolio in real time and writes a row to /history + the audit log.

- [ ] **E6. Advanced order types modal**
  MODE: Dark modal over /trade.
  CONTENT
  - Launched from `/trade` order form via `[Advanced…]` button.
  - Tabs: Limit · Stop · Stop-limit · Trailing · TWAP · Iceberg.
  - Each tab: parameter inputs (left) + live preview chart with projected trigger / working zones overlaid on the candle chart (right).
  - Right rail: "Why use this order type?" plain-English explainer ("A TWAP slices a 500K-credit order into 60 minute-by-minute slices to minimize market impact…") — Bloomberg voice, not tooltip voice.
  ACCEPTANCE: a buy-side trader new to AI-credit markets can pick the right order type without a Google search.

- [ ] **E7. Risk / margin dashboard**
  MODE: Dark at `/portfolio/risk`.
  CONTENT
  - KPI strip: portfolio VaR (95 / 99 · 1-day), beta to AI-INDEX, max single-position weight, margin used %, maintenance margin, distance-to-liquidation.
  - Per-position contribution table (sortable by VaR contribution, beta, weight, P&L).
  - Stress-test scenarios: "AI-INDEX -10%", "H100 -20%", "Image credits -30%", "All long correlation = 1" — projected portfolio P&L per scenario as colored bars.
  - For paper trading: banner explaining these are paper risk numbers; capital trading unlocks live margin lines.

---

## Tier F — Money operations

- [ ] **F1. Withdrawal flow** (`/wallet/withdraw`)
  MODE: Dark standalone funnel, mirrors `/wallet/buy` shape.
  STEPS: amount → destination (saved bank · new wire · ACH) → confirm.
  - Live receipt panel (USD requested, fee, net to bank, ETA by destination).
  - Real-money trading gate banner if user is paper-trading-only.
  - Pending-withdrawal status pill if user has one in flight (links to F4 transaction detail).

- [ ] **F2. Bank linking (Plaid-style modal)**
  MODE: Dark modal stack on /wallet.
  STEPS: search bank (logo grid + search) → credentials screen (Plaid partner badge visible) → success ("Connected · Chase · 2 accounts · ready for ACH in 1–2 business days").
  - Security ribbon at top: "Read-only · routing + account number · masked at rest".
  - Manual entry fallback: routing + account + 2× micro-deposit verification.

- [ ] **F3. Credit conversion modal** (cash ↔ credit, credit ↔ credit)
  MODE: Dark modal on /wallet.
  CONTENT
  - From / To pickers (cash USD, AI-INDEX, TEXT, SPEECH, IMAGE, VIDEO, H100, H200).
  - Live rate + slippage estimate + 5-second rate-lock countdown ring.
  - "Convert" button locks the rate; settles instantly at the venue rate.
  - Mini history strip below: last 10 conversions this session.

- [ ] **F4. Transaction detail** (any ledger row → drawer)
  MODE: Dark drawer.
  CONTENT
  - Header: tx id + type icon (buy / sell / conversion / deposit / withdrawal / fee).
  - Body: amount in / amount out, counterparty (or "venue" for trading-side), settlement state, fees breakdown, associated order id (if any, with link to E1), audit hash + verify link to /enterprise/audit pre-filtered.
  - Footer: "Download receipt PDF".

- [ ] **F5. Recurring purchase setup** (`/wallet/recurring`)
  MODE: Dark form + list page.
  FORM: "Buy $X of credit Y every {day, week, month} starting Z, paying with method P, until {end-date · total cap · indefinitely}."
  - Live calendar preview of the next 6 occurrences.
  - List of existing recurring purchases as cards with pause / edit / cancel.

- [ ] **F6. Payment methods page** (`/wallet/methods`)
  MODE: Dark, in-app.
  CONTENT
  - List of saved cards / wires / ACH with masked details, expiry, last-used date.
  - Set-default toggle per method.
  - Add new: card via embedded provider iframe; wire via instructions reveal; ACH via F2 bank-link.
  - Compliance notice: card buys settle instant, wire T+1, ACH T+3.

---

## Tier G — Compute & Inference v1.5

- [ ] **G1. Compute instance detail** (`/compute/[id]`)
  MODE: Dark, full page.
  CONTENT
  - Header: instance name · region · GPU type × count · status chip · spend-to-date.
  - Tabs:
    1. **Overview** — live per-GPU utilization (8 mini bars), RAM gauge, disk IOPS sparkline, network in/out
    2. **Logs** — tail of stderr / stdout · follow toggle · search · download
    3. **Snapshots** — volume backups with size + date + restore action
    4. **Events** — provisioning, restarts, resizes (timeline)
    5. **Cost** — per-hour breakdown chart, monthly projection
  - Actions row: Stop · Restart · Resize · Terminate.

- [ ] **G2. SSH keys management** (`/settings#ssh` — fill out the stub section)
  MODE: Dark.
  CONTENT
  - Table of registered public keys: label, fingerprint, added date, last used.
  - Add-key modal: paste public key + label (validates ssh-ed25519 / ssh-rsa).
  - Color-code by environment label (prod / dev / personal).
  - Revoke = immediate; audit-log entry.

- [ ] **G3. Job history** (`/compute/jobs` or `/inference/jobs`)
  MODE: Dark, dense log-style table.
  COLUMNS: time · job type · target (model / instance) · duration · status · cost · output size · re-run.
  - Filter chips: type / status / date.
  - Per-row expand: input config snapshot + output preview (first 200 chars or thumbnail).

- [ ] **G4. Model detail page** (`/inference/models/[slug]`)
  MODE: Dark, full page.
  CONTENT
  - Hero card: model name · publisher · version · param count · context window · throughput · price-per-1M-tok.
  - Tabs:
    1. Card (markdown model card)
    2. Benchmarks (small charts vs peer models on MMLU / GSM8K / HumanEval / venue-internal)
    3. Examples (3 representative prompts + responses)
    4. API (endpoint URL + auth + curl / python / js snippets)
    5. Versions (changelog, deprecation date if applicable)
  - "Open in playground →" CTA back to /inference pre-filled with this model.

- [ ] **G5. Fine-tuning wizard** (`/inference/finetune/new`)
  MODE: Dark, 4-step funnel.
  STEPS:
    1. Base model picker (from G4 catalog, filtered to "supports fine-tune")
    2. Dataset upload (jsonl drag-drop with schema validator + row preview)
    3. Hyperparams (LR, epochs, batch — sensible defaults with "advanced" reveal)
    4. Review (cost estimate, ETA, output destination, billing sub-account)
  - Submit → job appears in G3.

- [ ] **G6. Batch processing console** (`/inference/batch`)
  MODE: Dark.
  CONTENT
  - Upload csv/jsonl of prompts → pick model → run.
  - Progress bar per file, throughput counter (req/s), live failed-row count.
  - Failed-row retry batch action.
  - Download merged results (csv / jsonl).

- [ ] **G7. API usage analytics** (`/settings#analytics` or `/inference/analytics`)
  MODE: Dark.
  CONTENT
  - Time-series of req/min, tokens/min, $/min (stacked by model / sub-account / API key).
  - P50 / P95 / P99 latency strips per endpoint.
  - Top endpoints / models by spend.
  - Rate-limit headroom gauge per sub-account.

---

## Tier H — Enterprise depth

- [ ] **H1. Sub-account detail** (`/enterprise/teams/[id]`)
  MODE: Light enterprise admin.
  CONTENT
  - Members list (sub-set of org team).
  - Budget cap + spend trend chart (30 days).
  - Attached resources: compute instances, API keys, fine-tunes.
  - Audit slice: last 50 events scoped to this sub-account (link to /enterprise/audit pre-filtered).

- [ ] **H2. Permission matrix** (`/enterprise/permissions`)
  MODE: Light.
  CONTENT
  - Big table: Roles (Owner, Admin, Trader, Viewer, custom roles) × Permissions (40+ scopes grouped: Trading, Compute, Inference, Account, Billing, Security, Admin).
  - Cells: checkmark / dash / "inherited from parent role" badge.
  - Edit-role modal: clone an existing role and toggle individual scopes.
  - Show effective permissions for a selected member (intersection of all assigned roles).

- [ ] **H3. SSO / SAML configuration wizard** (`/enterprise/sso`)
  MODE: Light, 3 steps.
  STEPS:
    1. Choose IdP — Okta, Azure AD, Google Workspace, custom SAML 2.0
    2. Exchange metadata — download 1Trade SP metadata, paste IdP metadata XML or URL
    3. Test sign-in — real handshake against a sandbox user, reports back with attribute mapping preview
  - Success state: active connection card + SCIM provisioning toggle + nightly re-sync indicator.

- [ ] **H4. Vendor risk packet generator** (`/enterprise/risk-packet`)
  MODE: Light, one page.
  CONTENT (sections, all populated by the platform):
  - Company overview (legal name, jurisdiction, HQ, founding year, headcount)
  - Certifications (SOC 2 Type II, ISO 27001, GDPR DPA on file, PCI-DSS scope)
  - Sub-processor list (AWS, Cloudflare, Plaid, etc. with locations)
  - Pen-test summaries (last 2 annual tests, severity counts, remediation status)
  - Incident history (last 12 months, none-or-list)
  - Data flow diagram
  - Download as PDF (signed) for the customer's procurement team.

- [ ] **H5. MSA / contract renewal flow**
  MODE: Light, inline banner inside /enterprise/onboarding when T-60 to renewal.
  CONTENT
  - Diff view of pricing schedule v2.4 → v2.5 (line-by-line).
  - Actions: Approve · Negotiate (opens email to CSM) · Schedule call.
  - Tracked as a 7th checklist step in /enterprise/onboarding for the renewal window.

---

## Tier I — Datacenter partner v1.5

- [ ] **I1. Settlement detail** (`/datacenter/statements/[id]`)
  MODE: Light.
  CONTENT
  - Header: period · gross · venue fee · net · payout state · wire date.
  - Body: line-by-line capacity sold (instrument, GPU-hours, $/hr, total).
  - Reconciliation diff vs partner's own meter (if uploaded).
  - Wire instructions echoed back for the partner's banking records.

- [ ] **I2. Live capacity dashboard** (`/datacenter/live`)
  MODE: Light, dense grid.
  CONTENT
  - Rack × cluster grid with live utilization bars (tick every 2s).
  - Color: green normal, amber >85%, red full.
  - Filter by region, GPU type, customer.
  - Click cell → instance-level breakdown drawer.

- [ ] **I3. SLA dashboard** (`/datacenter/sla`)
  MODE: Light.
  CONTENT
  - Uptime % rolling 30 / 90 / 365.
  - MTTR by incident category.
  - Incident log table.
  - SLA credits owed/paid this period, with red highlight on any breach of the partner agreement targets.

- [ ] **I4. Hardware lifecycle / asset register** (`/datacenter/assets`)
  MODE: Light table.
  COLUMNS: GPU asset id · model · serial · install date · firmware · health · age · projected EOL.
  - Maintenance window scheduler (modal).
  - Decommission flow (multi-step approval).

- [ ] **I5. Tax forms / 1099** (`/datacenter/tax`)
  MODE: Light.
  CONTENT
  - Year-by-year listing of issued forms (1099-NEC for US LLCs, equivalents per jurisdiction).
  - Download buttons per form.
  - W-9 / W-8BEN on file with expiry, "Re-submit" prompt at T-30 to expiry.

---

## Tier J — Marketing / public / developer

- [ ] **J1. About page** (`/about`)
  MODE: Light marketing.
  - Team section (founder photos + bios).
  - Backers / investors (logo wall).
  - Thesis: 1-page brief on "commodity market for AI compute" written for serious readers.
  - Milestones timeline.

- [ ] **J2. Careers** (`/careers`)
  MODE: Light.
  - Roles list with filters: team · location · remote-ok.
  - Per-role: markdown JD + Greenhouse / Lever application embed.
  - Culture section + benefits.

- [ ] **J3. Status page** (`/status`)
  MODE: Light, public, scoped to 1Trade's own systems.
  CONTENT
  - Component grid (Matching engine · Order entry · Wallet · API · Datacenter regions us-east-1 / us-west-2 / eu-central-1 / ap-south-1) with green / amber / red dots.
  - Active incident strip (if any) with timestamps and updates.
  - Historical uptime per component (last 90 days).
  - Subscribe-to-updates form (email, SMS, RSS).

- [ ] **J4. Changelog** (`/changelog`)
  MODE: Light.
  - Reverse-chronological list of dated entries.
  - Per entry: title, summary, tags (api / ui / pricing / market-rules / compliance).
  - RSS link in header.

- [ ] **J5. API documentation** (`/docs/api`)
  MODE: Light developer.
  - Sidebar TOC, sticky.
  - Per-endpoint section: method + path · auth + required scopes · request schema · response schema · curl / python / js examples (tabs) · rate limits · errors.
  - ⌘K palette to search across endpoints.
  - Copy-on-click code blocks.

- [ ] **J6. Trust center** (`/trust`)
  MODE: Light.
  - Compliance certs (SOC 2, ISO 27001, PCI DSS scope).
  - Sub-processor list.
  - Security whitepaper download.
  - Bug-bounty program link.
  - Privacy policy, DPA, trading rules links.

- [ ] **J7. Blog / press** (`/blog`, `/press`)
  MODE: Light, editorial.
  - Post template: hero, byline, body markdown, related posts.
  - Filter by tag.
  - Press releases section.

---

## Tier K — Admin (internal 1Trade ops, dark)

- [ ] **K1. Market-making controls**
  MODE: Dark, internal-only.
  CONTENT
  - Per-market quoter parameters: spread, depth, max inventory.
  - On-call MM dashboard with live P&L per market.
  - Kill-switch per market and global.
  - 2-person approval required for any param change.

- [ ] **K2. Circuit breaker dashboard**
  MODE: Dark, internal.
  - Triggered breakers history.
  - Active halt states across markets.
  - Manual halt / resume controls (2-person approval).
  - Every action writes to the audit chain.

- [ ] **K3. KYC review queue**
  MODE: Dark, internal compliance.
  - Pending applications list.
  - Per-applicant: identity docs preview, sanctions screen result, PEP screen, computed risk score.
  - Actions: Approve · Request-info · Deny · Escalate.
  - Bulk actions DISABLED BY DESIGN to force individual review.

- [ ] **K4. Trade surveillance**
  MODE: Dark, internal.
  - Alerts feed for wash trading, spoofing, layering patterns.
  - Per-alert evidence trail (orders + fills + timing).
  - Mark false-positive or escalate to compliance officer.

- [ ] **K5. AML / sanctions alerts**
  MODE: Dark, internal compliance.
  - Daily screen-hit list.
  - Manual case file per hit.
  - Closure reason recorded immutably to the audit chain (links to /enterprise/audit).

---

## Tier L — Help, errors, mobile

- [ ] **L1. Help center / KB** (`/help`)
  MODE: Light.
  - ⌘K-style search.
  - Category tiles (Trading · Wallet · Compute · Inference · Enterprise · API · Datacenter partners).
  - Article template (markdown body + "Was this helpful?" feedback + "Contact support" fallback).

- [ ] **L2. Glossary** (`/help/glossary`)
  MODE: Light.
  - Alphabetical terms: AI credit · spread · tape · slippage · maker/taker · liquidation · halt · circuit breaker · etc.
  - Plain-English definitions + links to deeper KB articles.

- [ ] **L3. Maintenance mode page**
  MODE: Dark, full-page takeover during venue maintenance.
  - Banner: "Read-only window · 04:00–04:01 UTC · est. 30s."
  - Live countdown.
  - /status link.
  - Browsing OK, trading paused — call this out in copy.

- [ ] **L4. Rate-limited / permission denied / suspended states**
  Three small state pages, mode matches origin (light from marketing, dark from app).
  COPY: honest, specific, gives the user a path forward. Mirror /states D11 voice. No "Oops!".
  - 429 rate limited — "You've hit 60 req/min. Backoff in 24s. Increase your quota →"
  - 403 permission denied — name the missing scope, link to who to ask.
  - 423 suspended — name the reason category, point at support, give compliance contact.

- [ ] **L5. Mobile trading dashboard**
  MODE: Dark, 390px viewport.
  CONTENT
  - Compressed /trade: ticker · candle chart · book (collapsed by default) · order form (bottom sheet) · positions (bottom sheet).
  - Banner: "Information mode — for full pro venue use desktop."
  - Web-only. No native app for v1.

- [ ] **L6. Tour mode** (referenced from `/onboarding/welcome` CTA)
  MODE: Dark overlay over /trade.
  CONTENT
  - Sequential spotlight + tooltip on: ticker → sidebar → chart → book → tape → order form → positions → command palette.
  - Each step: "Next" / "Skip tour" / progress dots.
  - Total ~60 seconds.
  - Replayable from /settings → Display.

---

## Tier M — Existing screens that need rounding out

- [~] **M1. Settings sections beyond Profile + API Keys**
  /settings has 10 left-nav sections but only Profile, Account & Security, and API Keys have real content. Build the remaining 7: **Notifications · Display · Trading defaults · Tax · Sessions · Compliance docs · Data export.** Use the same dense pattern as the existing API Keys section.

- [~] **M2. /history expanded** (closes C4)
  Current /history is a 20-row mock table. Add: filter bar (date / market / side / result), quarterly / yearly aggregates, CSV / JSON export with cryptographic signing, per-row link to E1 order detail, date dividers between days.

- [~] **M3. Marketing footer links**
  ~25 links in `MarketingFooter.vue` point to `#`. Either wire each one to a real page as it ships (J1, J3, J6, etc.) or remove the dead columns until they exist. No `href="#"` in a shipped product.

- [ ] **M4. /markets list page** (`/markets` index, no slug)
  Sortable list of every credit market with mini sparklines, last px, Δ, vol, spread. Group by family (Index · Text · Speech · Image · Video · GPU). Per-row pin → adds to E2 watchlist. Currently `/markets` 404s — sidebar "Markets" goes straight to `/markets/eai-idx`.

- [~] **M5. Logout flow + avatar menu**
  Topbar avatar button is a no-op. Wire it to a dropdown: Profile · Settings · Sign out · Switch organization (for enterprise users). Sign-out clears local state, returns to /login.

- [ ] **M6. SSO sign-in on /login**
  "Sign in with SSO (firm accounts)" link is a `#`. Wire to: email input → IdP discovery (look up domain → IdP) → SAML redirect mock that returns to /trade.

- [ ] **M7. KYC: actual document upload**
  Current /onboarding/kyc skips identity-document capture. Add: passport / driver's license front+back, selfie capture (webcam permission flow), auto-OCR preview. Plug into K3 review queue.

- [ ] **M8. /benchmark methodology — historical revisions**
  The methodology page has the spec but no revision history. Add a "Methodology revisions" appendix: every quarterly change to index constituents, with diff view (added / removed / re-weighted) and the audit-chain hash of the revision.

---

## Cross-cutting work (not screens, but blocks the polished demo)

- [ ] **N1. Keyboard chord coverage** — extend the command palette chords:
  `G C` → /compute · `G I` → /inference · `G E` → /enterprise/onboarding · `G A` → /enterprise/audit · `G L` → /markets/heat · `G ?` → help.

- [ ] **N2. Skeletons on every list / table page** — every page that fetches data needs a skeleton that mirrors final layout (no layout shift on first frame). /states D11 is the reference.

- [ ] **N3. Global toast region** — used by every action (success / info / warn / error). Match D9 notifications visual language but ephemeral (auto-dismiss 4s, hover to pause, click to expand to full notification card).

- [ ] **N4. Uniform `<ConfirmDialog>`** — for every destructive action: close position, cancel order, revoke key, terminate instance, delete sub-account, revoke role. 2-step (type the resource name) for irreversible ones.

- [ ] **N5. Empty / loading / error per route** — every list page should ship all three D11 patterns. Currently several pages render nothing on first load.

- [ ] **N6. Price-tick flash animations** — when a price updates in any cell, brief 200ms background flash (`--pos-soft` on up, `--neg-soft` on down). Already done in some places; needs to be consistent everywhere prices live.

- [ ] **N7. Locale + currency** — every $ today is en-US USD. Add locale negotiation (Accept-Language) + USD/EUR/GBP/JPY display preference in M1 Settings → Display.

- [ ] **N8. A11y pass** — focus rings, ARIA labels on icon-only buttons (notifications bell, avatar, sidebar items), keyboard nav of every table, screen-reader announcements for live ticker.

---

## Suggested shipping order

Highest demo / sales value first (so the next pitch is materially better):

1. **M5** (logout + avatar menu) — 1 day, fixes the most visible "demo break".
2. **E1** (order detail) — required to make /history and /trade feel finished.
3. **M4** (markets index) — fills a 404, completes navigation.
4. **L6** (tour mode) — completes the /onboarding/welcome promise.
5. **E5** (position close) — the "close my position" question always comes up in demos.
6. **F4** (transaction detail) — round-trips /wallet ledger to evidence.
7. **J3** (status page) — trust signal for enterprise.
8. **N3** (toasts) — quality-of-life across the whole app.
9. **G1** (instance detail) — completes /compute story.
10. **H3** (SSO wizard) — checks the biggest enterprise sales box.

Everything else fills in as backlog. Tier K is internal-only; ship after public surfaces are settled.
