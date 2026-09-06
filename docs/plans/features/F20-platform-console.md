# F20 — Platform Console UI

> Ship in **Milestone 1** (shell) → **Milestone 6** (polish). Owner: `trading-frontend`.

## Spec

The Platform Console is the **customer-facing surface for Phase 1**. Repurposes the existing
Nuxt frontend (currently `1Trade Frontend/`, scheduled to migrate to `apps/web/` per
`REPO_LAYOUT.md` §3).

Screens:

```
Marketing (prerendered):
  /                — landing, value prop
  /signup, /login  — onboarding entry points
  /benchmark       — model latency benchmarks (lives at /benchmark already)

App (SPA, behind auth):
  /catalog         — inference catalog: browse + try
  /inference       — run inference from the UI (play with a model, see token usage)
  /compute         — list / create / stop GPU instances
  /compute/clusters — multi-node cluster management (M5+)
  /wallet          — credit balances + transactions + buy credits + convert
  /billing         — month-to-date consumption, projected, alerts, auto-stop, invoices
  /settings        — account, API keys, team, security
  /enterprise      — SAML config, SCIM, sub-accounts (M4+)
  /datacenter      — partner-DC portal (utilization, payouts, capacity health) (M4+)
  /docs            — documentation site (M6)

Kept warm (Phase 2 surface):
  /trade           — trading dashboard demo, "exchange paused" ribbon
  /markets/**, /portfolio, /history — kept warm
  /index           — methodology page (real); index value page is internal-only in Phase 1
```

Existing screens to confirm (cross-check against
`docs/1trade_mvp_screens_checklist.md`):
- Trading Dashboard, Market Detail, Wallet, Portfolio, Signup, Sign-in, 2FA, KYC, Buy Credits,
  Trader KYC, Methodology, Homepage, Brand Book.

## Owning agent

`trading-frontend`.

## Contracts consumed / produced

### Produces
- None (frontend doesn't own a contract).
- Visual design tokens (`tokens.css`), component library — local to `apps/web/`.

### Consumes
- All `openapi/*` contracts.
- `credit-types.md`.

## Dependencies

- F01 (CDN + deploy).
- F02 (auth — login screen wires to OAuth + SAML).
- F03 (orgs / sub-account switcher in nav).
- F04 (CLI install instructions on console pages).
- F05–F19 (every screen consumes a service).

## Sync points

- M1 end — shell renders against mock; design system audited.
- M2 — wallet + catalog wired to real backends; sub-5-min onboarding tested.
- M3 — compute + billing wired; reserved-capacity flow live.
- M4 — enterprise + DC-partner portal.
- M5 — full catalog UI + methodology page polished.
- M6 — docs site + SDK pages + onboarding tour.

## Acceptance criteria

- [ ] Every screen renders on mock data convincingly (M1).
- [ ] `TRADE1_API_MODE=local` switches all screens to live data with zero UI change (M2+).
- [ ] Sub-5-min signup → first inference flow (Playwright CI).
- [ ] WCAG AA: contrast, keyboard nav, screen-reader labels.
- [ ] Single design system; no drift; numbers in mono+`tabular-nums`.
- [ ] Trading dashboard ribbon: "Demo — exchange paused" until Phase 2.
- [ ] Documentation site live with OpenAPI auto-generated reference.

## Milestone

- M1 (Gate 1): shell.
- M2 (Gate 2): wallet + catalog wired.
- M3 (Gate 3): compute + billing wired.
- M4 (Gate 4): enterprise + DC portal.
- M6 (Gate 6): polish + docs.

---

## Status — rebrand, landing + IA rebuild, onboarding activation (2026-09-06)

**Brand** (`e3fc8c5`) — 1TRADE identity: mark rebuilt from the brand guide as clean vector assets,
tokens retoned to Midnight Black + Sovereign Gold, dark-first. Gold is 9.42:1 on midnight but
**1.98:1 on cream**, so the light theme's accent is Desert Bronze deepened to `#8A5D28` (5.38:1);
gold fills always take black text. Every ratio computed, not estimated.

**Landing page** — rebuilt, 1667 → 983 lines. Led with the paused exchange and CTA'd into `/markets`,
i.e. sold the one product that cannot be bought; and argued the three-sided market twice across three
sections. Now: exchange thesis → who it's for → what you can buy → how it works, with a one-line
definition under the headline. The simulated index series is labelled **Illustrative**.

**Navigation IA**
- Marketing nav: every entry now resolves where its label says. "Docs" pointed at `/inference`, an
  authenticated app screen; four of six links were same-page anchors.
- Sidebar: labels + groups + collapse, replacing a 64px icon-only rail whose `group` field was
  never rendered. Persona switcher restored — persona lives only in localStorage, so a wrong pick at
  signup had no way back.
- **12 app pages had no auth guard** and rendered a dead shell instead of redirecting. Fixed. (Data
  was never exposed — the BFF 401s.)
- **96 token references were used with no fallback**, so their declarations were invalid and the
  borders simply never drew. Defined; light/dark parity now 53/53.

**Onboarding — advances `[ ] Sub-5-min signup → first inference flow`.** That flow was previously
impossible: signup produced a tenant with no key and no credits, so first call required a Stripe
payment. `/onboarding/welcome` is now activation rather than a menu — starter credits (F06), API key
minted in place, first request shown ready to run. Step 1 reports the balance it actually reads back;
it never asserts a grant that may not have landed.

**Open**
- [ ] Playwright CI test for the sub-5-min flow — the flow works, the gate is unproven.
- [ ] Console/Wallet/Inference interiors are unchanged (~5,000 lines); only the shell and landing
      were rebuilt.
- [ ] 47 pre-existing `vue-tsc` errors remain in untouched files.
