# Persona journeys — what works end-to-end

**As of 2026-06-03 (tags v0.1.0 → v0.3.0).** The **full journey** for each persona, step by step, with
each step marked done / showcase / paused. Pair with [STATUS.md](STATUS.md) (tree map) and
[TESTING.md](TESTING.md) (how to verify each).

> **Legend:** ✅ **green = done** (live, backend-wired, verified) · ⬜ showcase (screen exists, no live
> backend yet) · ⏸ paused (by design — license-gated).

> **Personas are a frontend / onboarding filter, not an auth boundary.** The backend is *tenant*-scoped,
> so any logged-in tenant can call any API; "persona" only decides *which journey the UI shows*. Source:
> `usePersona.ts` + the `personas:` tags in `App/Sidebar.vue`. Switch via the persona pill at the bottom
> of the sidebar.
>
> | Persona id | Who | Journey status |
> |---|---|---|
> | **`enterprise`** | Maya Chen — AI Company (VP Eng) | ✅ **fully live end-to-end** |
> | **`partner`** | Tom Reyes — Datacenter (Capacity ops) | ⬜ account live; supply showcase |
> | **`trader`** | Jordan Park — Trader (Independent quant) | ⏸ account live; exchange paused |

---

## ✅ enterprise — AI Company (Maya Chen)

**The headline journey: signup → buy credits → run inference / rent a GPU → see the debit. Fully live.**

| # | Step | Screen / surface | Status | Tag |
|---|---|---|---|---|
| 1 | Sign up (pick **AI Company** account type) | `/signup` | ✅ | F02 (v0.1.1) |
| 2 | Verify email (real SMTP → Mailpit locally) | `/onboarding/verify` | ✅ | F02 (v0.2.9) |
| 3 | Onboarding — persona welcome + product tour | `/onboarding/welcome` · `/tour` | ✅ | F20 (v0.2.10) |
| 4 | KYC gate (real identity check before real money) | `/onboarding/kyc` | ⬜ shell | F22 (M3) |
| 5 | Browse the **live model catalog** | `/inference` | ✅ | F08 (v0.2.0) |
| 6 | **Buy credits** — Stripe checkout → webhook → ledger mint | `/wallet/buy` | ✅ (mock Stripe in dev) | F06 (v0.2.1) |
| 7 | **Run inference** (chat) → **real per-token debit** + session meter | `/inference` | ✅ | F08/F09 (v0.2.7) |
| 8 | Convert credits (AI-index ↔ text), 1% spread | `/wallet` convert | ✅ | F07 (v0.2.5) |
| 9 | **Rent a GPU instance** — provision → running + connect info | `/compute/new` → `/compute` | ✅ | **F13 (v0.3.0)** |
| 10 | Manage instances — list / stop / start / delete (GPU debit while running) | `/compute` | ✅ | **F13 (v0.3.0)** |
| 11 | Wallet — live balances + transaction movements | `/wallet` | ✅ | F05 (v0.2.6) |
| 12 | Mint an **API key** (shown once) | `/settings` | ✅ | F02 (v0.2.8) |
| 13 | Drive it all from the **CLI** — `login / infer / gpu / credits / keys` | `exascale …` | ✅ | F04 + F13 (v0.1.4, v0.3.0) |
| 14 | One-screen **console** ops | `/console` | ✅ | F20 (v0.2.3) |
| 15 | Budgets + purchase history | billing surfaces | ✅ | F06 (v0.1.3) |
| 16 | Audit log + RBAC (admin actions → queryable trail) | F03 backend | ✅ backend · ⬜ rich screen | F03 (v0.1.2) |
| 17 | Enterprise team management + SAML SSO + sub-accounts | `/enterprise/teams` · `/sso` | ⬜ | F02/F03 (M4) |

**Live today:** steps 1–3, 5–16 (backend). **Not yet:** 4 real KYC (M3), 16 rich audit screen (F23),
17 enterprise SSO/teams (M4).

---

## ⬜ partner — Datacenter (Tom Reyes)

**Account layer works; the whole supply side is showcase until M3/M4.**

| # | Step | Screen / surface | Status | Tag |
|---|---|---|---|---|
| 1 | Sign up (pick **Datacenter** account type) | `/signup` | ✅ | F02 |
| 2 | Verify email | `/onboarding/verify` | ✅ | F02 |
| 3 | DC dashboard — capacity / utilization overview | `/datacenter` | ⬜ showcase | F16 (M3/M4) |
| 4 | Register GPU capacity (tier · count · region) | `/datacenter/register` | ⬜ showcase | F17 (M4) |
| 5 | Capacity joins the **one supply pool** (owned + partner) | supply abstraction | ⬜ not built | F16 (M3) |
| 6 | GPU attestation / proof-of-authenticity | — | ⬜ not built | F19 (M5) |
| 7 | Get paid — escrow + streamed payout per usage | — | ⬜ not built | F18 (M4) |
| 8 | Wallet + settings (shared account layer) | `/wallet` · `/settings` | ✅ | F05 / F02 |

**Live today:** steps 1–2, 8. **Everything supply-specific (3–7) is showcase / not built.**

---

## ⏸ trader — Trader (Jordan Park)

**Account + wallet work; the exchange is paused (license-gated) by design — screens are mock showcase.**

| # | Step | Screen / surface | Status | Tag |
|---|---|---|---|---|
| 1 | Sign up (pick **Trader** account type) | `/signup` | ✅ | F02 |
| 2 | Verify email | `/onboarding/verify` | ✅ | F02 |
| 3 | Browse markets (AI-compute index products) | `/markets` | ⏸ showcase | KW (paused) |
| 4 | Order book + chart + tape | `/trade` | ⏸ showcase | KW03 (paused) |
| 5 | Place / match an order | `/trade` | ⏸ no engine | KW03 (paused) |
| 6 | Portfolio + P&L | `/portfolio` | ⏸ showcase | KW (paused) |
| 7 | Trade history | `/history` | ⏸ showcase | KW (paused) |
| 8 | The AI-compute **index** | `/benchmark` | ⏸ keep-warm | KW01 (paused) |
| 9 | Wallet — incl. the `ai_index` credit + convert | `/wallet` | ✅ | F05 / F07 |

**Live today:** steps 1–2, 9. **All trading (3–8) is paused mock showcase** — turns on with the license
(Phase 2, kept warm).

---

## One-line summary

**`enterprise` is the live product** — the full signup → credits → inference / GPU-rental → debit journey
works end-to-end. **`partner`** and **`trader`** have a live account + wallet layer only; their domain
journeys (supply onboarding / trading) are showcase or paused by design.
