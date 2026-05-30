# Platform — Done / Not‑Done (UI/UX + capabilities)

**Status as of 2026‑05‑30.** What exists today vs what's missing, for the customer‑facing platform
and the broader capability list. Pair with [PROVISIONING.md](PROVISIONING.md) (external accounts)
and [MANAGEMENT_PLAN.md](MANAGEMENT_PLAN.md) (the feature roadmap).

**Legend:** ✅ done (backend live, proven in k3d) · ⚠️ backend done, **UI not wired** · ❌ not done ·
📋 planned (feature doc exists) · 🆕 net‑new (not in the roadmap at all)

> **The one‑line takeaway:** for the core self‑serve flow, the **backends are live and proven; the
> Nuxt screens exist only as mock UI and aren't wired to them.** The gap is the **console wiring
> (F20)** plus four small backend gaps — not the backends themselves.

---

## A. Core product flow (the sellable self‑serve loop)

| # | Item | Backend | UI | Verdict |
|---|------|---------|----|---------|
| 1 | Auth — Login / Signup | ✅ F02 (signup/login→JWT/me) | 🎨 `login`,`signup` mock | ⚠️ wire to live API |
| 1 | Auth — Verify (email) | ❌ none | 🎨 `onboarding/verify` mock | ❌ no send/verify endpoint |
| 1 | Auth — 2FA / OTP, OAuth | ❌ OAuth stubbed (501), no 2FA | — | ❌ F02‑M4 |
| 2 | AI Models — selection | ✅ F08 `GET /v1/models` | 🎨 `inference` mock | ⚠️ wire model picker |
| 3 | API key — generate | ✅ F02 keys create/list/revoke | 🎨 no dedicated screen | ⚠️ add keys screen |
| 4 | Test — **text** | ✅ F08 `/v1/chat/completions` | 🎨 `inference` playground | ⚠️ wire playground |
| 4 | Test — speech / image / video | ❌ contract only, no handlers/models | 🎨 mock | ❌ needs handlers + F10 models |
| 5 | AutoPay — credit card | ✅ F06 Stripe checkout→mint | 🎨 `wallet/buy` mock | ⚠️ wire + Stripe Elements |
| 5 | AutoPay — auto‑recharge / saved card | ❌ none | — | ❌ not built |
| 6 | Cost — balance / bill ($) | ✅ ledger balances + tx | 🎨 `wallet`,`enterprise/billing` mock | ⚠️ wire dashboard |
| 6 | Cost — budgets / alerts / auto‑stop | ❌ none | — | ❌ F06‑M3 (50/80/100% alerts) |

---

## B. Platform capabilities

### On the roadmap (planned, not built)
- ❌ 📋 **Kubernetes / Slurm** — F12 compute control plane (K8s + Kueue + Volcano). *Slurm out of scope (K8s‑native).* UI `compute/*` mock.
- ❌ 📋 **Security — full RBAC / orgs / sub‑accounts** — F03. (Role enum exists in F02.)
- ❌ 📋 **Security — OTP / 2FA** — F02‑M4.
- ❌ 📋 **Compliance — SLA / SOC 2** — F21 (Vanta), not started.
- ❌ 📋 **Privacy — HIPAA / GDPR / CCPA** — part of F21/F22.
- ⚠️ **Speed / Latency test** — `latency_ms` emitted on every request; test tool/UI not wired. UI `benchmark` mock.
- ❌ **Realtime — speech / vision** — text SSE streaming ✅; realtime (websocket) speech/vision not built.
- ❌ **Webhook (outbound) / analytics** — inbound Stripe webhook ✅; customer‑facing webhooks + analytics dashboard not built.

### Net‑new — NOT in the roadmap (a bigger product; needs a scope decision)
- 🆕 **Data — store / clean / tokenize / pipeline**
- 🆕 **Training — MLops / evals** (raw GPU rental is F12–F15; orchestration/evals are not)
- 🆕 **Fine‑tune — data / mem / speed**
- 🆕 **Prompt‑engineering sandbox** (overlaps the playground)
- 🆕 **AI Safety & guardrails** (moderation)
- 🆕 **RAG / vector retrieval**
- 🆕 **Storage — file / block / object**

---

## C. Summary buckets

**✅ Fully done (backend + proven live):** auth core, model catalog, API keys, text inference,
Stripe purchase, credit balance/ledger, PCI‑by‑Stripe.

**⚠️ Backend done — UI not wired (= the F20 console job):** login/signup, model picker, API‑key
screen, text playground, buy‑credits, cost/balance view, latency view.

**✅ Closed (v0.1.3):** email verification (token + verify/resend, single-use, audited); monthly
budgets (set/get, audited); the previously-unlinked endpoints (verify, budget, purchases, audit) now
have BFF routes.

**❌ Not built — close the loop (small, near‑term):** budget alert UI + auto-stop (M3),
speech/image/video inference, AutoPay auto‑recharge.

**❌ Not built — net‑new scope (large, needs a decision):** Data, Training/MLops, Fine‑tune, RAG,
AI Safety, Storage, realtime speech/vision, outbound webhooks/analytics, OTP/2FA, full RBAC (F03),
SOC 2 (F21), HIPAA/GDPR.

---

## D. What this implies for sequencing
1. **F20 — wire the console to the live APIs** is the highest‑leverage next move: it turns 5 proven
   backends into a product a customer can click through (auth → key → model → playground → buy →
   usage). The screens already exist on mock data.
2. **Close the four small backend gaps:** email verify · usage/cost aggregation + budgets/alerts ·
   speech/image/video handlers · auto‑recharge.
3. **Decide on net‑new scope** (Data / Training / Fine‑tune / RAG / Safety / Storage) deliberately,
   before building — these expand from "inference + GPU marketplace" to a full ML platform.
