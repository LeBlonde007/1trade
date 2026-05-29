# Agent plan — `trading-frontend` (repurposed: **Platform Console**)

> Repurposed under the GTM pivot. Becomes the **Platform Console** in Phase 1 — catalog, wallet,
> compute, billing, account. The trading dashboard is **kept warm** as the investor demo and
> Phase 2 surface; clearly labelled as such today.

## 1. Scope under the GTM pivot

The frontend agent's original headline (trading dashboard) shifts: it stays the eventual headline,
but Phase 1's customer-facing screens are platform-console screens. The agent owns BOTH:

- **Platform Console** (active, customer-facing): the screens an AI-startup engineer or enterprise
  procurement buyer uses to onboard, buy credits, run inference, manage GPUs, see billing.
- **Trading dashboard** (keep-warm): a working demo on mock data, used in investor conversations
  and ready for Phase 2 switch-on. The existing screens stay.

Both run on **one design system** (already established in the existing repo's `tokens.css`).

## 2. Features owned

| Feature | Status |
|---|---|
| [F20 — Platform Console](../features/F20-platform-console.md) | active, M1 (shell) → M6 (polish) |
| [KW02 — Trading dashboard kept warm](../features/KW02-trading-demo-ui.md) | keep-warm, continuous |
| Cross-cutting: design system, mock-data layer, HTML→Nuxt migration | active, continuous |

## 3. Milestone-by-milestone

### Milestone 1 — Shell + design system + mock layer
- Confirm/lock the design system (`tokens.css`, components, mono/`tabular-nums` for numbers,
  sentence case, institutional aesthetic — no crypto-flashy).
- Migration: the existing Nuxt repo (`Exascale Frontend/`) stays in place this milestone; the move to
  `apps/web/` is scheduled for end of M1 (after CI is green; see `REPO_LAYOUT.md` §3).
- Migrate the remaining HTML mockups (`docs/htmls/*`) into Nuxt routes if not already done. The
  existing app already has many — confirm coverage against [`exascale_mvp_screens_checklist.md`](../../other/exascale_mvp_screens_checklist.md).
- Platform Console shell: nav, layouts (marketing light, app dark), "wallet empty" + "no models
  yet" empty states.
- Trading dashboard at `/trade`: kept on mock data with a "Demo — exchange paused" ribbon.
- Mock-data layer: a single `useApi()` composable that flips between `mock | local | staging`
  per `EXASCALE_API_MODE`. Same shapes as the OpenAPI contracts.

### Milestone 2 — Wallet + catalog (wired)
- Wallet page wired to `credit-ledger` `GET /v1/credits/balances` + `GET /v1/credits/transactions`.
- Catalog page wired to `inference-ml`'s `GET /v1/inference/models`.
- Onboarding flow: signup → email verify → first inference (5-minute target measured by Playwright).
- Buy-credits flow wired to Stripe (cards) for M2.

### Milestone 3 — Compute + billing
- Compute page: list/create/stop GPU instances against `compute-platform`'s API.
- Reserved capacity purchase UI: tier picker (1/6/12 mo), discount preview, prepay through wallet.
- Billing page: month-to-date consumption, projected month-end, cost alerts at 50/80/100%, auto-stop.
- ACH/wire + JPY purchase paths added.

### Milestone 4 — Enterprise
- SAML SSO login (consume `platform-core` SAML config).
- Sub-account switcher (org → sub-account hierarchy in the top nav).
- Per-team budgets + org-wide consumption dashboard.
- Audit-log export view.
- DC-partner portal (separate sub-app for partner operators): utilization, payouts, capacity health.

### Milestone 5 — Catalog scale + index page
- Full catalog UI with category filters + latency badges per model.
- (Keep-warm) Index page wired to `index-service`'s **private** reference index — visible only
  to internal accounts; the public-facing version goes live in Phase 2.
- Methodology page published (the methodology spec is real even though the index is private).

### Milestone 6 — Polish + docs site
- Documentation site at `/docs` (MDX-based; OpenAPI auto-generated reference).
- SDK pages (Python, JS, Go quickstarts).
- Onboarding tour (`driver.js` already in deps).
- Polish: empty states, error states, loading states across every screen.

### Phase 2 — Trading goes from "demo" to "live"
- Flip the `/trade` route from mock-data to the real backend (matching engine, market maker, index).
- Remove the "Demo — exchange paused" ribbon.
- Customer-facing onboarding flow for traders (light KYC, paper-trading allocation).
- Order types: market, limit, IOC, FOK (UI already exists; backend cutover is the change).

## 4. Contracts owned / consumed

### Owned
- None (frontend doesn't own a contract; it consumes them).

### Consumed (read-only)
- `openapi/platform-core.yaml`, `openapi/credit.yaml`, `openapi/inference.yaml`,
  `openapi/compute.yaml`, `openapi/supply.yaml`, `openapi/index.yaml`, `openapi/trading.yaml` (keep-warm).
- `credit-types.md` (drives the wallet UI labels).

## 5. Local dev

- Current path: `Exascale Frontend/` (own repo) — `npm install && npm run dev` on `:3000`.
- Target path post-migration: `apps/web/` under the monorepo.
- `EXASCALE_API_MODE=mock` default; switch to `local` once `make up` services are running.
- `apps/web/server/api/*` holds the mocks (already partially built — see existing repo).
- Storybook (or equivalent) not in scope for v1; we lean on visual smoke tests via Playwright.

## 6. Dockerfile

`deploy/docker/Dockerfile.nuxt` (see `DEPLOYMENT.md` §3).

For marketing/static routes, build via `nuxt generate` and serve on Cloudflare Pages.
For the SSR app routes, deploy the Dockerfile to K8s.

## 7. Deploy

- Marketing (`/`, `/signup`, `/login`, `/benchmark`) → Cloudflare Pages (prerendered).
- App (`/trade`, `/markets/**`, `/wallet`, `/portfolio`, `/compute`, `/inference`, `/settings/**`, `/datacenter/**`) → K8s Deployment (Node SSR).
- CDN: Cloudflare in front of everything.
- The current `nuxt.config.ts` already has the right `routeRules`; preserve those during the
  monorepo migration.

## 8. Conventions

- **`docs/plans/DESIGN_SYSTEM.md` is the authority** — tokens, type scale, spacing, number
  formatting, mock-data realism, and the aesthetic guardrails (institutional, never crypto-flashy).
  Every screen/component conforms; a design change is a one-line edit in `tokens.css`, never in a page.
- **No hardcoded colors/sizes/fonts** — everything via `var(--token)` or a component.
- **Numbers**: mono + `tabular-nums`. Sentence case everywhere.
- **Charts in `.client.vue`** (chart libs are SSR-unsafe).
- **One data layer.** `useApi()` composable hides mock-vs-real. Switching backends requires zero
  UI changes — this is the contract that lets the M3 mock→real cutover be clean.
- **Trading dashboard ribbon**: until Phase 2, every trading route shows "Demo — exchange paused.
  See methodology." in a prominent ribbon. `security-compliance` reviews the regulatory framing.

## 9. Hard boundaries

- Don't implement backend logic. Consume APIs.
- Don't invent endpoint shapes — if you need one, ask `tech-lead`.
- Don't reframe credits or trading in marketing copy without `security-compliance` review.

## 10. Definition of done

- Platform Console screens match the spec, run convincingly on mock data, and switch to real
  data with zero UI change.
- One token system + one component library; no drift. **Conforms to `DESIGN_SYSTEM.md`** —
  `design-tokens-guard` passes (no raw hex/rgb in components); numbers mono + `tabular-nums`;
  passes the institutional-aesthetic gut check (no anti-references).
- Sub-5-min onboarding flow tested by Playwright in CI.
- Trading dashboard kept demo-able to investors (mock data, methodology page real).
- Documentation site live.
- Accessibility: WCAG AA baseline (color contrast, keyboard nav, screen-reader labels).
