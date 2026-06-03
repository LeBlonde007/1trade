# Working features by persona

**As of 2026-06-03 (tags v0.1.0 → v0.3.0).** What actually works today, grouped by the persona whose
UI journey exposes it. Pair with [STATUS.md](STATUS.md) (tree map) and [TESTING.md](TESTING.md) (how to
verify each).

> **Legend:** ✅ **green = working** (live, backend-wired, verified) · ⬜ showcase (screen exists, no
> live backend yet) · ⏸ paused (by design — license-gated).

> **Personas are a frontend / onboarding filter, not an auth boundary.** The backend is *tenant*-scoped,
> so any logged-in tenant can call any API; "depends on persona" means *which persona's UI exposes it*.
> Source of truth: `usePersona.ts` + the `personas:` tags in `App/Sidebar.vue`. Switch via the persona
> pill at the bottom of the sidebar.
>
> | Persona id | Who | Default |
> |---|---|---|
> | **`enterprise`** | Maya Chen — AI Company (VP Eng) | ✅ yes (platform-first audience) |
> | **`partner`** | Tom Reyes — Datacenter (Capacity ops) | |
> | **`trader`** | Jordan Park — Trader (Independent quant) | |

---

## ✅ Shared — all personas (account layer)

| Feature | Where | Tag |
|---|---|---|
| ✅ Signup / login / logout / me · email verify (real SMTP via Mailpit local) | `/signup` `/login` | F02 (v0.1.1, v0.2.9) |
| ✅ Wallet — live balances + transactions / movements | `/wallet` | F05 (v0.2.6) |
| ✅ Wallet — credit conversion (AI-index ↔ text), 1% spread, atomic two-leg | `/wallet` convert drawer | F07 (v0.2.5) |
| ✅ Settings — API keys create / list / revoke (secret shown once) | `/settings` | F02 (v0.2.8) |
| ✅ Console — one-screen ops (auth, balances, activity) | `/console` | F20 (v0.2.3+) |
| ✅ Onboarding — persona pick + KYC / verify shell | `/onboarding/*` | F20 (v0.2.10) |
| ✅ CLI `exascale` — login/whoami/credits/convert/catalog/infer/keys/config/**gpu** | terminal | F04 + F13 (v0.1.4, v0.3.0) |

---

## ✅ enterprise (AI Company — Maya) — the live end-to-end product

The **only persona with a fully working revenue loop**: signup → buy credits → infer (or rent a GPU) →
ledger debit → see it.

| Feature | Where | Tag |
|---|---|---|
| ✅ Inference — model catalog + chat completions + **real per-token debit** + session meter | `/inference` | F08 / F09 (v0.2.0, v0.2.7) |
| ✅ Buy credits — Stripe checkout → webhook → idempotent ledger mint (mock Stripe in dev) | `/wallet/buy` | F06 (v0.2.1) |
| ✅ Monthly budgets / purchases | billing surfaces | F06 (v0.1.3) |
| ✅ **Compute — GPU instances list + provision + stop / start / delete (live)** | `/compute`, `/compute/new` | **F13 (v0.3.0)** |
| ✅ Compute — GPU-type catalog + quota + internal job scheduling | compute-control | F12 (v0.2.12) |
| ✅ Audit log + RBAC (admin actions → queryable trail) | F03 backed | F03 (v0.1.2) |
| ⬜ Enterprise onboarding / richer billing screens (live path is `/wallet/buy`) | `/enterprise/*` | F23 backlog (~15%) |

---

## 🟡 partner (Datacenter — Tom) — almost all showcase

| Feature | State |
|---|---|
| ✅ Shared account layer (wallet, settings, convert) | working |
| ⬜ `/datacenter`, `/datacenter/register` — DC capacity dashboard + supply onboarding | **mock showcase** — supply-source abstraction + DC onboarding (F16 / F17) **not built** |

No supply-side feature works yet beyond the shared account screens. Tom can log in and hold a wallet;
onboarding GPU capacity is showcase-only until M3/M4.

---

## ⏸ trader (Trader — Jordan) — exchange paused

| Feature | State |
|---|---|
| ✅ Shared account layer (wallet, convert — incl. the `ai_index` credit) | working |
| ⏸ `/trade`, `/markets`, `/benchmark` (index), `/portfolio`, `/history` | **paused, mock showcase** — the whole exchange/trading layer is license-gated per the GTM pivot (KW01–KW05, kept warm) |

Trader's trading surfaces are headline-quality mocks (Brownian prices, live book/tape) but have **no
live backend** — by design.

---

## One-line summary

**`enterprise` is the live product** (inference + GPU rental + credits, all backend-wired);
**`partner`** and **`trader`** are showcase / paused, sharing only the live account + wallet layer.
