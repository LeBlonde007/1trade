# 1Trade — Project Context (read by every agent)

> This file is auto-loaded into every Claude Code session and subagent. It is the shared
> ground truth. If something here conflicts with an agent's instructions, this file wins,
> except for the four immutable commitments which win over everything.

> **⚠️ CURRENT DIRECTION (2026-09 — trading restored) — read before sequencing any work.**
> Both tracks are now active:
> - **Platform** (inference + compute + prepaid credits) — revenue now, and the foundation the
>   exchange settles on. Keeps shipping.
> - **Exchange / trading layer** — **no longer keep-warm.** Build it for real: matching engine,
>   market maker, index, surveillance, trading UI. This restores commitment #1's ordering; the
>   2026-05 pivot's "exchange paused" sequencing no longer applies.
>
> **BUILDING IT IS NOT THE SAME AS OPERATING IT.** The licence gate (F22) is unchanged and still
> governs going live. Until counsel signs off:
> - no real-money trading, no live venue, **no taking trading accounts**;
> - every exchange surface stays explicitly staged (paper mode, "in design", order books labelled
>   illustrative) — do not quietly drop those labels;
> - the framing guide holds on **all** customer-facing copy. Banned words: "futures",
>   "speculation", "investment return", "guaranteed yield". F22 audits this.
>
> So: engineering unblocked, go-live still gated. If those two get conflated the licence track is
> the thing that breaks, and it is the long-lead item.
>
> - The 2026-05 pivot (historical context, superseded on sequencing): `docs/re/1trade_gtm_focus_update.md`
> - The build plan (per-agent + per-feature, setup → local → deploy): `docs/plans/README.md`
> - The engineering rulebook every agent follows: `docs/plans/ENGINEERING_STANDARDS.md`
>
> Product spec lives at `docs/re/phase6_v2.md` (the `docs/phase6-prd-ssd.md` path below is its
> intended home).

## What 1Trade is

A **commodity market for AI compute**. Three integrated layers over one credit system:

1. **Trading layer** (headline) — order book, market maker, credit ledger, index publication.
2. **Inference layer** — curated SoTA OSS model catalog, multi-tenant per-GPU.
3. **Compute layer** — GPU rental, CLI-first.

Three-sided market: datacenters (supply) → traders (liquidity) → AI companies (demand).
Real owned datacenter as credible underlying; partner DCs for scale.

## The four immutable commitments (never trade these away)

1. **Trading layer is the headline product** — even sandbox/paper mode must feel like a real market.
2. **Index methodology integrity** — public methodology, audit hash chain, surveillance from day one.
3. **Sub-5-minute time-to-first-action** — signup → first trade / first quote / first CLI command.
4. **Real-time credit visibility** — live balances, visible settlement, full audit trail.

## Tech stack (locked)

| Layer | Choice |
|---|---|
| Core services | **Go** (matching engine, ledger, index, market maker, surveillance, supply, platform-core) |
| Inference | **Python + vLLM** behind a Go gateway |
| Frontend | **Nuxt 4 + Vue 3 + TypeScript**, CSS-variable design tokens (see design system) |
| CLI | `1trade` — Go binary |
| Data | **PostgreSQL** (state), **TimescaleDB** (time-series: candles, prints, metrics), **Redis** (order books, cache), **NATS** (events) |
| Orchestration | **Kubernetes + Kueue + Volcano + NVIDIA GPU Operator** |
| Gateway / edge | Kong behind Cloudflare |
| Obs / secrets / compliance | Grafana stack · Vault · Vanta |

## Monorepo layout

```
1trade/
├── CLAUDE.md                  ← this file
├── docs/
│   ├── phase6-prd-ssd.md      ← the spec (source of product truth)
│   └── contracts/             ← SHARED CONTRACTS — the only way services couple
│       ├── openapi/           ← per-service REST specs (one yaml per service)
│       ├── schemas/           ← SQL migrations + shared DB schema
│       ├── events/            ← NATS subjects + payload schemas
│       └── credit-types.md    ← the canonical credit-type enum (shared everywhere)
├── services/                  ← Go services (one dir each)
├── apps/web/                  ← Nuxt 4 frontend
├── apps/cli/                  ← 1trade CLI
├── deploy/                    ← k8s · terraform · ci
└── .claude/agents/            ← the agent roster
```

## Contract-first rule (this is how agents avoid colliding)

Agents have **isolated context and cannot see each other's work directly.** They coordinate ONLY
through written contracts in `docs/contracts/`:

- An agent **consumes** contracts as read-only truth.
- An agent **never edits a shared contract** (OpenAPI, SQL schema, event schema, credit-types)
  on its own. Contract changes are proposed to the orchestrator, authored/approved by `tech-lead`,
  then other agents adapt. This prevents two agents silently diverging on an interface.
- If a contract you need doesn't exist yet, STOP and ask the orchestrator to have `tech-lead`
  author it before you build against assumptions.

## Global conventions (all agents enforce)

- **`is_paper` flag is sacred.** Every order, trade, and balance carries it. Real-money and
  paper state never mix. Internal/market-maker accounts never trade against customer paper accounts.
- **Credit ledger is append-only** with a hash chain: `chain_hash = hash(prev_chain_hash || canonical_json(row))`. Balance update and transaction insert are atomic.
- **Credit types** come from `docs/contracts/credit-types.md` — never hardcode a new credit type.
- **All money/credit math uses fixed-point** (`NUMERIC(20,6)` in SQL, decimal libs in Go) — never floats.
- **Frontend numbers** use mono font + `tabular-nums`. Sentence case everywhere. Institutional aesthetic (Bloomberg/Polymarket), never crypto-flashy.
- **Live-only frontend (no mock mode).** The web app is always wired to the real backend: every BFF route (`server/api/**`) proxies to a platform service, and screens render real data with loading/empty states — no `isMock` fallbacks, no in-component mock arrays. When wiring a new feature, ship it live (this supersedes the earlier "mock-data mode" convention). Screens with **no backend yet** (e.g. the paused exchange) may keep placeholder data until their service exists.
- **Tests + migrations ship with the code**, not after. No PR without tests for new logic.
- **Audit everything** that touches credits, orders, or admin actions.
- **Every function gets a doc comment** on the lines directly above it (what + why). Clean code,
  security baked in, fixed-point money math, idempotent mutations. Full rules + per-language
  formats in `docs/plans/ENGINEERING_STANDARDS.md` (CI-enforced via `.golangci.yml`,
  `.pre-commit-config.yaml`, `.editorconfig`).
- **Run the review skills before merge:** `/code-review` then `/simplify`, and `/security-review`
  on any diff touching credits / orders / auth / billing / customer data. See
  `ENGINEERING_STANDARDS.md` §9.

## Definition of done (every agent)

Code compiles · tests pass · contracts honored (no undocumented interface drift) ·
**every function documented** · linters clean · audit/log hooks present where credits/orders/admin
involved · `is_paper` respected · fixed-point money math · idempotent mutating calls · no secrets
in code · `/security-review` clean on sensitive surfaces · matches the spec in `docs/re/phase6_v2.md`
and the GTM pivot. Full checklist: `docs/plans/ENGINEERING_STANDARDS.md` §10.
