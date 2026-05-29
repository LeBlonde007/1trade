# Exascale — Exchange Demo Gaps

> Audited 2026-05-24 against `/mnt/f/ex/Exascale Frontend/app/pages/**` and the
> three personas in `exascale_persona_demo_flows.md`. Lists every step of the
> end-to-end exchange demo and the screens that are **still missing** to make
> that full loop runnable on the frontend.
>
> Saved here instead of overwriting `exascale_mvp_screens_prompts.md` (the
> original 26-MVP catalog). Same paste-ready prompt format as A1–A4 (the
> claude.ai project already has those foundation files attached — don't
> re-paste them).
>
> ### Status convention
> `[x]` shipped · `[~]` partial / present but the demo loop isn't closed ·
> `[ ]` not started.

---

## Audit at a glance

### Foundations

| | Screen | Path | Status |
|---|---|---|---|
| | Website / landing | `/`, `/benchmark` | `[x]` |
| | Signup | `/signup` | `[x]` |
| | Login + 2FA | `/login` | `[x]` |
| | Email verify | `/onboarding/verify` | `[x]` |
| | Welcome / paper-trading account | `/onboarding/welcome` | `[x]` |
| | KYC light flow (4-step) | `/onboarding/kyc` | `[x]` |
| **F1** | **KYC — document upload + selfie + OCR** | *(missing)* | `[ ]` |

### 1. Trader (Jordan persona)

| | Action | Path | Status |
|---|---|---|---|
| | Test: paper-trade | `/trade` | `[x]` |
| **T1** | **Link bank account** (Plaid + manual) | *(missing)* | `[ ]` |
| **T2** | **Deposit cash via wire / ACH** | *(missing — `/wallet/buy` is credit-purchase, not USD deposit)* | `[ ]` |
| | Trade: market & limit buy / sell | `/trade` | `[x]` |
| **T3** | **Withdrawal: bank wire / ACH** | *(missing)* | `[ ]` |

### 2. AI Company (Maya persona)

| | Action | Path | Status |
|---|---|---|---|
| | Enterprise onboarding shell | `/enterprise/onboarding` | `[x]` |
| | Team / sub-accounts | `/enterprise/teams` | `[x]` |
| | Billing dashboard | `/enterprise/billing` | `[x]` |
| **A1** | **Bulk credit procurement** (AI-Co buyer, NOT individual trader funnel) | *(missing)* | `[ ]` |
| **A2** | **Use AI credits — burn meter** on `/inference` (cost-per-token live) | partial (`/inference` exists; no meter) | `[~]` |
| **A3** | **Use GPU credits — burn meter** on `/compute/[id]` (cost-per-second live + credits-left) | partial (`/compute/[id]` exists; no meter) | `[~]` |
| **A4** | **Sell back unused credits** to the venue (AI-Co as supplier) | *(missing)* | `[ ]` |

### 3. GPU Datacenter (Tom persona)

| | Action | Path | Status |
|---|---|---|---|
| | Partner dashboard | `/datacenter` | `[x]` |
| | Capacity registration (intake form) | `/datacenter/register` | `[x]` |
| **D1** | **Connect GPUs** — agent install / SSH / verify | *(missing)* | `[ ]` |
| **D2** | **Download AI models / containers** to fulfill demand | *(missing)* | `[ ]` |
| **D3** | **Mint GPU-hours → GPU credits** | *(missing — settlements visible but not the mint step)* | `[ ]` |
| **D4** | **List credits for sale** on the venue (supplier listing) | *(missing — partner sees orders consumed but not the supply side they listed)* | `[ ]` |

---

## The 12 prompts (paste into the claude.ai project chat one at a time)

---

### F1. KYC — document upload + selfie + OCR

**MODE:** Light, standalone funnel (extends `/onboarding/kyc`).
**VIEWPORT:** 1440px, centered 720px column.

**CONTENT**
1. Sticky stepper: `① Identity → ② Address → ③ Tax → ④ Documents → ⑤ Review`.
2. **Documents** is the new step (between Tax and Review):
   - Picker: **Government photo ID** (Passport · Driver's license · National ID).
   - Drop zone × 2 (front + back), with live thumbnail preview, file-type / size validation, image-quality hint ("Glare on top edge — try again?").
   - **Selfie capture** via getUserMedia: oval guide, 3-2-1 countdown, retake button, capture as JPEG blob.
   - **OCR preview**: parsed name / DOB / doc number / expiry shown beside the thumbnail (mock values for demo). Flag when OCR mismatches the Identity step.
   - **Face-match score** chip ("Match: 0.96 · ✓ Passed") with explanatory tooltip.
3. Live status rail right side: `Uploading 2/3 · 14% · SHA-256: 7a3f…e021`. Hashes shown so the audit chain story is implicit.
4. Submit → success state: "Submitted to compliance — typical review window 4 business hours" with reference number `kyc_8c2a48f1`.

**ACCEPTANCE:** A compliance officer would believe this collects what real KYC providers (Persona, Onfido) collect. Plugs into `K3 KYC review queue` from the v1.5 catalog.

---

### T1. Link bank account (Plaid-style + manual fallback)

**MODE:** Dark modal on `/wallet`.
**CONTENT**
- 3 steps: **Search bank** (logo grid + search input → Chase, Bank of America, Wells Fargo, etc.) → **Credentials** (Plaid-style partner-badged screen) → **Success** (`Connected · Chase · 2 accounts · ready for ACH in 1–2 business days`).
- **Manual fallback tab:** routing # + account # + 2× micro-deposit verification step ("Two small deposits will arrive within 1–2 business days. Return here to verify.")
- Security ribbon: `Read-only · routing + account number · masked at rest`.
- After success, the linked account appears in `/wallet/methods` and `/wallet/withdraw` destination picker.

**ACCEPTANCE:** No PII in any visible field after success; account ends `••4421`.

---

### T2. Deposit USD via wire / ACH

**MODE:** Dark standalone funnel at `/wallet/deposit` (sibling of `/wallet/buy`).
**CONTENT**
- **Step 1 — Method:** Wire (instant if same-day cutoff, T+0 same-day, $15 fee) · ACH push (T+2, free) · Wire international (T+1, $40 fee).
- **Step 2 — Amount** with quick chips ($1,000 · $10,000 · $100,000 · custom).
- **Step 3 — Instructions:**
  - Wire: reveal bank coordinates with copy buttons (Bank name · Routing · Account · Beneficiary `Exascale FBO Jordan Park` · **Reference code** `EXA-DEP-9F2A` mono — call out *"include the reference or your wire will sit"*).
  - ACH: pick a linked bank (from T1), confirm push, instant pending row appears in `/wallet`.
- **Live state rail:** pending deposit ticker — *"Listening for wire reference EXA-DEP-9F2A…"* with a pulsing dot.
- **Confirmation page** with timeline (Initiated → Bank acked → Settled).

**ACCEPTANCE:** Distinct from `/wallet/buy` — that converts USD → credit; this lands USD in the cash leg of the wallet.

---

### T3. Withdrawal: bank wire / ACH

**MODE:** Dark standalone funnel at `/wallet/withdraw`.
**CONTENT**
- Mirrors `/wallet/buy` shape: amount → destination (saved bank from T1) → confirm.
- Live receipt: USD requested, fee, **net to bank**, ETA badge per method (Wire today before 14:00 UTC / Wire next day / ACH T+2).
- Real-money trading gate banner if user is paper-trading-only: *"Capital trading requires further verification — complete F1 docs upload."*
- Pending-withdrawal status pill if user has one in flight, linking to F4 transaction detail (v1.5 catalog).
- Confirm step: full SHA-256 hash of the wire instruction set is shown — audit-chain feel.

**ACCEPTANCE:** Bank balance updates in `/wallet` ledger immediately as PENDING; settled state arrives via toast at the demo's pace.

---

### A1. Bulk credit procurement (AI-Co buyer)

**MODE:** Light, enterprise at `/enterprise/procurement` (sibling of /onboarding, /teams, /audit, /billing).
**CONTENT**
- Buyer's-side dashboard. Not a re-skin of `/wallet/buy` — this is **procurement**:
  - Quote builder: AI Index credits + sub-credit breakdown (text · speech · image · video) + GPU credits (H100 · H200) + GPU forwards (30 / 60 / 90-day).
  - **Volume tier table** (1M – 10M – 100M – 1B credits) with effective $/credit rates that improve with size; rebate kick-in marked.
  - **Reserve via index** vs **buy spot** toggle — preview shows projected cost over 30 days at each strategy.
  - **Allocation by sub-account** matrix (AI-Research, Customer-Service-AI, Marketing-Image-Gen) — split the buy across teams with budget caps.
  - PO + invoice flow: generate signed PDF quote → submit to procurement → upon approval, credit is delivered to the org wallet.
- Right rail CSM card (same as `/enterprise/onboarding`) for "Talk through the strategy".

**ACCEPTANCE:** A CFO would forward this to AP and not need a sales call.

---

### A2. Inference burn meter (use AI credits)

**MODE:** Dark, integrated into the existing `/inference` page.
**CONTENT**
- New right-sidebar panel (replaces or augments the model-card metadata sidebar):
  - **Credits remaining** large mono number — `12,847,492 AI credits ($12,839.66 USD)` — ticks down live as tokens stream.
  - **This response**: tokens in / out · $/1M tok · cumulative cost — updating with the streamed output.
  - **Sub-account dropdown** at top — switches which budget the call burns against.
  - **Rate-limit headroom** mini bar (rps + tpm).
  - **"$0.34 spent in this session"** running tally; "Top up →" link routes to `/wallet/buy`.
  - Each streamed character flashes the cost number in `--pos-soft`; ticker tone subtle, not jarring.
- When credits drop below 100K → amber banner "Auto-top-up not configured · [Configure →]" inline.

**ACCEPTANCE:** A buyer can watch the credit burn in real time. Cost feels physical, not abstract.

---

### A3. Compute burn meter (use GPU credits)

**MODE:** Dark, integrated into the **Cost tab** of `/compute/[id]`.
**CONTENT**
- Header KPI strip on top of the Cost tab:
  - `Running for 04:12:38` (uptime mono)
  - `Cost so far $58.30` mono
  - `Burn rate $13.96/hr` mono — recomputed live every 5s
  - `Credits remaining 1,420 GPU-hours · ~101h headroom`
- Per-second line chart: spend vs time, last 60 minutes (Chart.js).
- Cost split table: instance ($) + storage ($) + egress ($) per hour, per day, projected month.
- Auto-stop config: pause at `$X total` or `Y hours` — live countdown if armed.

**ACCEPTANCE:** From this view you can tell finance exactly what this single instance will cost by month-end.

---

### A4. Sell unused credits back to the venue

**MODE:** Dark drawer launched from `/wallet` or `/enterprise/billing`.
**CONTENT**
- **Two paths:**
  1. **Market sell** — accept best bid right now. Live spread shown (`bid $0.000995 / ask $0.001005 · 1.0% spread`). Slippage estimate.
  2. **Limit list** — list at your price. Shows current depth at neighbouring levels. Time-in-force chips: GTC / DAY / GTD.
- **What to sell:** amount + which credit (AI-INDEX / TEXT / IMAGE / H100 / H200) + which sub-account it draws from.
- **Use of proceeds:** convert to USD (lands in cash leg of `/wallet`) · roll into another credit family · hold.
- **Tax estimate** (US LLC mock): "Realized gain $1,240 · withholding 24% if you're a US person → $297.60 estimated".
- Confirm fires an order; row appears in `/history`.

**ACCEPTANCE:** Demonstrates the venue is bidirectional. AI co with surplus credits exits without a phone call.

---

### D1. Connect GPUs — agent install + verify

**MODE:** Light, partner-portal at `/datacenter/connect/[siteId]`.
**CONTENT**
- **Step 1 — Pick site** (Reno NV · Phoenix AZ · Dallas TX).
- **Step 2 — Install the Exascale agent:**
  - Copyable one-liner: `curl -sL https://exa.sc/agent | sudo SITE=pdc_7c2a INSTALL_TOKEN=ix_8a91…0c sh` (mono, copy button).
  - OS tabs: Ubuntu 22.04 · Rocky 9 · Talos · "BYOD / custom" with package URLs.
  - Air-gapped option: download a signed tarball.
- **Step 3 — Heartbeat detection** (live):
  - Loading state: "Listening for first heartbeat from `pdc_7c2a` …"
  - When agent dials in, the list populates: hostname · GPU model · count · CUDA version · driver · fingerprint hash · temperature.
  - Each row: **Run validation** button — 60-second test suite (CUDA sanity · NCCL allreduce · disk IO · network egress). PASS/FAIL chip and detail.
- **Step 4 — Confirm onboarding** — selected hosts move into the active capacity pool. Auto-redirect to `/datacenter/assets`.

**ACCEPTANCE:** A real datacenter ops engineer would believe the install path. Hashes, signed packages, validation suite all visible.

---

### D2. Download AI models / containers

**MODE:** Light at `/datacenter/library`.
**CONTENT**
- Library of model artefacts the venue ships to partner hardware so fills happen locally (no model-cold-start latency):
  - Columns: model · publisher · size · format (gguf · safetensors · ONNX · NIM container) · checksum · last updated.
  - Per row: pre-fetch status per site (Reno: 100% · Phoenix: 64% · Dallas: 0%).
  - **Bulk policy:** *Auto-prefetch top-10 by 30-day demand* toggle.
  - Bandwidth meter showing current pull rate per site.
- **Per-model detail** drawer: license, hardware compatibility (H100/H200/MI300), supported quantization, minimum VRAM.
- Disk pressure gauge per site — red when > 85%.

**ACCEPTANCE:** Storage cost story is visible. Partner can see exactly which models occupy how many TB.

---

### D3. Mint GPU-hours → GPU credits

**MODE:** Light at `/datacenter/mint`.
**CONTENT**
- Header: **Available capacity to tokenise** — `Reno: 12,840 H100-hours · 4,920 H200-hours` (rolling 30-day uncommitted supply).
- **Mint form:**
  - Asset type (H100 / H200 / A100 / B200 / custom).
  - Quantity (GPU-hours).
  - Maturity window: spot · 7-day · 30-day · 90-day forwards (different markets).
  - Floor price (the lowest $/hr the partner will accept).
  - Auto-throttle rules (pause minting if utilization > 90%).
- **Preview card:** "You are minting **8,000 H100-hours** as **`H100-FWD-30D`** at floor **$2.85/hr**. At current order book, ~62% likely to fill within 24h, generating ~**$22,800** gross."
- **Mint** button → on confirm: 2-person approval (partner ops + finance signoff) → credits appear in the venue as supply.
- Recent mints table below with cancel option (only for unsold portion).

**ACCEPTANCE:** This is the supply-creation primitive that makes the whole venue work. Treat it like a securities issuance UI.

---

### D4. List credits for sale (supplier listing)

**MODE:** Light at `/datacenter/listings`.
**CONTENT**
- Open listings table: instrument · qty · listed at · floor · best bid · % filled · accrued $.
- Row actions: **Re-price** · **Add to listing** · **Cancel remaining**.
- New listing form (subset of D3 — used when credits already minted but unsold):
  - Pick from minted-but-unlisted bucket.
  - Time-in-force (GTC · GTD · close-of-day).
  - Auto-roll into next maturity on expiry checkbox.
- Settlements feed (right rail): every fill shows up with buyer category masked ("AI-Co · Tier 2 · US-East") and net to partner.

**ACCEPTANCE:** Pair with D3 to complete the supplier loop: mint → list → fill → settle.

---

## Suggested shipping order

For the canonical exchange demo (a single take-from-zero recording):

1. **F1** — KYC docs (unblocks "capital trading" gate everywhere).
2. **T1 → T2 → T3** — bank rails (deposit / withdrawal) close the trader loop.
3. **A2 + A3** — burn meters (visible cost is the AI-Co value prop in 10 seconds).
4. **A1** — bulk procurement (sells the enterprise pitch in 1 minute).
5. **D1 → D3 → D4** — supply side (demonstrates Exascale is two-sided).
6. **A4** — sell-back (closes the bidirectional story).
7. **D2** — model library (last; storytelling polish, not core flow).

With those 12 screens added, every step of the user's checklist
(Foundations → Trader → AI Company → Datacenter) is demoable end-to-end
without hand-waving "trust me, this would exist".
