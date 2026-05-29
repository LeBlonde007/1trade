# Exascale — UI Walkthrough (the 31 screens we actually have)

> **NotebookLM source + screen-recording shot list.** Describes the *real* screens in the Nuxt app
> so the video references what exists. Grouped by the platform-first story: **what ships now**
> (platform console + enterprise + supply) vs **kept-warm** (the trading floor — the act-two demo).
> Run locally: `make web` → `http://localhost:3000` (mock-data mode). Recommended capture: 1440px.

---

## A. The hook — marketing + onboarding (light theme)

| Route | What's on screen | Why it matters in the demo |
|---|---|---|
| `/` (index) | Landing: "the commodity market for AI compute," three-audience split (AI companies · datacenters · traders), live index ticker | The thesis in one screen; opens the video |
| `/benchmark` | Model latency/price benchmarks | Proof of the "curated, transparent" positioning |
| `/signup` · `/login` | Email + OAuth entry | Sub-5-minute time-to-first-action story |
| `/onboarding/welcome` · `/verify` · `/kyc` | Guided onboarding, email verify, light KYC | "Signup to first action in minutes," compliance-aware |
| `/onboarding/tour` | **Persona-driven autoplay product tour** (Trader / AI Company / Datacenter) | The single best demo device — drives the whole walkthrough |

## B. The platform — what an AI startup uses today (dark app theme)

| Route | What's on screen | Demo beat |
|---|---|---|
| `/inference` | Inference playground — pick a model, run a prompt, see tokens + live credit debit | "One base-URL change; pay per use" — the fastest revenue path |
| `/compute` · `/compute/new` · `/compute/[id]` | GPU instances: list, create (type/count), instance detail | `exascale gpu create` → running box; on-demand + reserved |
| `/wallet` · `/wallet/buy` | Credit balances per type, transactions, buy-credits flow | Prepaid credits; real-time balance — ties to the live ledger |
| `/settings` | Account, API keys, security | DX + programmatic access |
| `/history` | Activity / transaction history | Every action is logged |

> **Tie-in to what's built:** the wallet/balances/transactions map directly onto the **live credit
> ledger** (append-only, hash-chained, deployed in k8s). When the video says "every credit movement
> is cryptographically audited," that is a real, running backend — not a mock.

## C. Enterprise — the buyer's compliance surface (light admin theme)

| Route | What's on screen | Demo beat |
|---|---|---|
| `/enterprise/onboarding` | Enterprise account setup | F500 / frontier-lab path |
| `/enterprise/sso` | SAML SSO configuration | "Your IdP, our platform" |
| `/enterprise/teams` | Sub-accounts, per-team budgets | Org-wide spend control |
| `/enterprise/billing` | MTD spend, projected month-end, cost alerts, invoices | The CFO view; burn meters |
| `/enterprise/audit` | Hash-chained audit log with cryptographic verification + chain-integrity badge | The compliance centerpiece — provable spend |

## D. Supply — the datacenter partner portal

| Route | What's on screen | Demo beat |
|---|---|---|
| `/datacenter` | Partner dashboard: capacity contributed, utilization/fill rate, settlements | "Monetize idle GPUs" |
| `/datacenter/register` | Capacity onboarding intake | One scheduling pool across owned + partner |

## E. Kept-warm — the exchange (act two; clearly labelled "demo / exchange paused")

| Route | What's on screen | Demo beat |
|---|---|---|
| `/trade` | Trading floor: live candle chart, order book, time-and-sales tape, order form (Bloomberg-tier density) | The vision made tangible — "this is what switches on with a license" |
| `/markets` · `/markets/[slug]` | Market list + per-market detail | Breadth of the eventual venue |
| `/portfolio` | P&L vs the index, position breakdown, attribution | Trader experience |
| `/status` | System/venue status | Operational maturity |

> **Framing rule for the video:** show the trading floor as the **funded act-two** ("designed,
> mock-built, switches on when licensed"), never as something live today. `security-compliance`
> requires the "exchange paused" framing — it keeps the credits-as-prepaid-service-units story clean.

---

## Design notes (so the capture looks institutional)

- Two themes on one token system: **light** for marketing + enterprise admin, **dark** for the
  product/trading app. Numbers are mono + tabular; sentence case; restrained color. The aesthetic
  is Bloomberg / Linear / Stripe — *not* crypto-flashy (this is deliberate and worth narrating).
- Live-updating mock data (candles tick, tape rolls, burn meters move) signals "real market."
- Best capture order mirrors the persona tour: **AI Company** (inference → compute → wallet →
  enterprise billing/audit) → **Datacenter** (partner dashboard → register) → **Trader** (trade →
  portfolio), then close on the audit chain.
