# Exascale — HTML → Nuxt 4 Migration Plan & Build Prompt

> **What this is:** a paste-ready execution plan for converting the existing Exascale HTML mockups into one clean, consistent Nuxt 4 application.
> **Hand it to:** Claude Code (recommended), or follow it yourself.
> **Non-negotiable goal:** every page shares ONE design system. Consistency comes from architecture, not from hand-matching.

---

## 0. The one principle that governs everything

The eight HTML files were generated independently. That means each one carries its own copy of colors, spacing, fonts, button styles, and card treatments — and they have almost certainly drifted apart. If you port each file 1:1 into a Vue page, **you import the drift permanently** and "same design everywhere" becomes a manual chore forever.

The migration fixes this by extracting shared concerns exactly once:

```
ONE token file        → every color, font, space, radius, shadow lives here
TWO layouts           → marketing (light) + app (dark). No page repeats chrome.
ONE component library  → one Button, one Card, one Table, one PriceDelta. Reused everywhere.
ZERO inline styles     → no hardcoded hex, px, or font in any page. Everything via var() or a component.
```

After this, changing the brand lime or the corner radius is a **one-line edit** that propagates to all 26 screens. That is what "same design for all pages" actually means in practice.

> **Clarification on "same design":** marketing pages are light-theme, app pages are dark-theme. That is intentional and stays. "Same design" means _same design system_ — identical tokens, components, spacing rhythm, typography, and motion — expressed as two coherent themes derived from the **same** token file. It does not mean making the homepage dark.

---

## 1. Aesthetic lock (do not reinvent)

The design direction is already decided and is correct for the product. The migration preserves it exactly:

- **Aesthetic:** institutional / financial-terminal. Reference points: Bloomberg Terminal, TradingView, Polymarket. **Anti-references:** Robinhood, generic crypto, generic AI startup, agency-portfolio flashiness.
- **Type:** `Inter Tight` (display), `Inter` (body), `JetBrains Mono` (all numbers). Yes, Inter — the institutional context makes the "use a distinctive font" advice wrong here. Keep it.
- **Numbers:** always `font-variant-numeric: tabular-nums`. Non-negotiable in a trading UI.
- **Cards:** sharp. 2–4px radius. No big rounded "friendly" corners.
- **Density:** information-dense, calm, precise. Generous where marketing; tight where trading.
- **Color:** dark canvas + lime brand accent, green/red for pos/neg only. No decorative gradients in the app surface.

The executor's job is **fidelity**, not creativity. If the generated HTML drifts from this, pull it back toward institutional.

---

## 2. Prerequisites

- Node.js **22.x+**
- The 8 completed HTML files in a known folder (e.g. `/source-html/`):
  - `Exascale Homepage.html`
  - `Exascale Signup.html`
  - `Login.html`
  - `Trader KYC.html`
  - `Exascale Trading Dashboard.html`
  - `Exascale Market Detail.html`
  - `Exascale Wallet.html`
  - `Portfolio.html`
- The Exascale design-system brief (for tokens that aren't yet in the HTML).

---

## 3. Phase plan (execute in order)

Each phase has a clear deliverable. Do **not** start migrating pages (Phase 5) until Phases 0–4 are done, or you will reintroduce drift.

| Phase | Name                                | Output                                     | Gate before next phase            |
| ----- | ----------------------------------- | ------------------------------------------ | --------------------------------- |
| 0     | Design audit & token reconciliation | `tokens.css`, `global.css`, decisions note | Tokens agreed; conflicts resolved |
| 1     | Scaffold                            | Empty Nuxt 4 project, correct structure    | `npm run dev` shows blank app     |
| 2     | Assets                              | Fonts, logo, favicon, icons, chart lib     | Fonts + logo render               |
| 3     | Layouts                             | `marketing.vue`, `app.vue`                 | Both shells render with nav       |
| 4     | Component library                   | Base + chrome + trading components         | Storybook page renders all        |
| 5     | Page migration                      | One `.vue` per HTML file                   | Each route matches its mockup     |
| 6     | Mock data                           | Live-simulation composables                | Candles/ticker/orderbook move     |
| 7     | Consistency QA                      | Passing checklist                          | Contract in §10 holds             |

---

## Phase 0 — Design audit & token reconciliation

**This is the most important phase. Do it carefully.**

1. Open all 8 HTML files. Extract every distinct value into a spreadsheet/table:
   - colors (background, surface, text, border, brand, pos/neg, accent)
   - font families, sizes, weights, line-heights
   - spacing scale (margins/paddings actually used)
   - border-radius values
   - shadow values
   - transition/animation durations
2. **Reconcile conflicts.** Where files disagree (e.g. one card is `#14161B`, another `#15171C`), pick ONE canonical value and record the decision. The drift dies here.
3. Build the canonical token file. Use the values below as the **starting source of truth** (from the design brief) and only adjust if the audited HTML proves a value should differ:

```css
/* app/assets/css/tokens.css */

/* ─── Dark theme — trading app surfaces ─── */
:root[data-theme="dark"],
.dark {
  --canvas: #0a0b0e;
  --elevated: #14161b;
  --overlay: #1c1f26;
  --hover: #1f2229;
  --text: #e8e6e0;
  --text-2: #9a9a95;
  --text-3: #5f5f5c;
  --border: rgba(255, 255, 255, 0.08);
  --border-2: rgba(255, 255, 255, 0.14);
  --brand: #c8f25c;
  --brand-hov: #b8e548;
  --accent: #4a90e2;
  --pos: #19c37d;
  --pos-sub: rgba(25, 195, 125, 0.12);
  --neg: #ef4444;
  --neg-sub: rgba(239, 68, 68, 0.12);
}

/* ─── Light theme — marketing / onboarding surfaces ─── */
:root,
:root[data-theme="light"] {
  --surface-canvas: #f8f7f4;
  --surface-elevated: #ffffff;
  --text-primary: #0a0b0e;
  --text-secondary: #5f5f5c;
  --border: rgba(0, 0, 0, 0.08);
  --brand-primary: #c8f25c;
  --positive: #19c37d;
  --negative: #ef4444;
}

/* ─── Cross-theme primitives (one source of truth) ─── */
:root {
  --font-display: "Inter Tight", "Inter", sans-serif;
  --font-sans: "Inter", system-ui, sans-serif;
  --font-mono: "JetBrains Mono", "Fira Code", monospace;

  /* spacing scale — use ONLY these values, no arbitrary px */
  --sp-1: 4px;
  --sp-2: 8px;
  --sp-3: 12px;
  --sp-4: 16px;
  --sp-5: 24px;
  --sp-6: 32px;
  --sp-7: 48px;
  --sp-8: 64px;

  /* radii — sharp, institutional */
  --r-sm: 2px;
  --r-md: 4px;
  --r-lg: 8px;
  --r-full: 999px;

  /* type scale */
  --fs-xs: 11px;
  --fs-sm: 13px;
  --fs-base: 14px;
  --fs-md: 16px;
  --fs-lg: 20px;
  --fs-xl: 28px;
  --fs-2xl: 40px;
  --fs-3xl: 56px;

  /* motion */
  --dur-fast: 120ms;
  --dur: 180ms;
  --dur-slow: 320ms;
  --ease: cubic-bezier(0.4, 0, 0.2, 1);

  --shadow-1: 0 1px 2px rgba(0, 0, 0, 0.2);
  --shadow-2: 0 4px 16px rgba(0, 0, 0, 0.3);
}
```

4. Build `global.css`: CSS reset, base typography, `*{box-sizing}`, body defaults, link reset, scrollbar styling, and a global rule that **all numeric displays use tabular figures**.

**Deliverable:** `tokens.css`, `global.css`, and a one-page "design decisions" note logging every conflict you resolved.

---

## Phase 1 — Scaffold the project

```bash
npm create nuxt@latest exascale-app
cd exascale-app
npm run dev -- -o     # confirm blank app boots
```

Enable Nuxt 4 mode and strict TS in `nuxt.config.ts`:

```ts
export default defineNuxtConfig({
  future: { compatibilityVersion: 4 },
  typescript: { typeCheck: true, strict: true },
  css: ["~/assets/css/tokens.css", "~/assets/css/global.css"],
  devtools: { enabled: true },
});
```

Set `app.vue` to the canonical shell — nothing else lives here:

```vue
<template>
  <NuxtLoadingIndicator />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

> **Rule:** never put a header/footer/sidebar in `app.vue`. Shared chrome lives in layouts only.

---

## Phase 2 — Assets

| Asset                  | Where                                                                                    | Notes                                                 |
| ---------------------- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| Fonts                  | Google Fonts `<link>` in `nuxt.config` `app.head`, OR self-host woff2 in `public/fonts/` | Self-host for production speed; link is fine for demo |
| Logo (mark + wordmark) | `app/assets/images/logo.svg` → `BrandLogo.vue`                                           | One component, takes `variant` + `theme` props        |
| Favicon / OG image     | `public/favicon.ico`, `public/og-image.png`                                              | Referenced as absolute `/...`                         |
| Icons                  | `lucide-vue-next` package                                                                | One icon system everywhere; never mix icon sets       |
| Charts                 | `lightweight-charts` (TradingView OSS)                                                   | Institutional candlestick look out of the box         |

Install:

```bash
npm i lucide-vue-next lightweight-charts @vueuse/core date-fns
```

Fonts in `nuxt.config.ts` `app.head.link`:

```ts
link: [
  { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
  { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Inter+Tight:wght@500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500;600&display=swap' },
],
```

**Styling decision (recommended):** use **CSS custom properties + scoped component styles**, and **skip Tailwind** for v1. Reason: your HTML is built on a heavy custom-CSS token system. Adding Tailwind creates two competing styling languages and makes "same design everywhere" _harder_, not easier. One token file + scoped styles keeps a single source of truth. (If the audited HTML turns out to be Tailwind-utility-heavy, revisit — but default to CSS vars.)

---

## Phase 3 — Layouts (the two-mode system)

Build exactly two layouts. Every page picks one via `definePageMeta`.

```vue
<!-- app/layouts/marketing.vue — LIGHT, public pages -->
<template>
  <div class="marketing-shell" data-theme="light">
    <MarketingNav />
    <main><slot /></main>
    <MarketingFooter />
  </div>
</template>
```

```vue
<!-- app/layouts/app.vue — DARK, authenticated trading app -->
<template>
  <div class="app-shell" data-theme="dark">
    <AppTopbar />
    <AppSidebar />
    <main class="app-main"><slot /></main>
    <AppStatusBar />
  </div>
</template>
```

> Single root element per layout (transitions break otherwise). The `data-theme` attribute is what flips the token set — that is the entire light/dark mechanism.

---

## Phase 4 — Component library

Build these BEFORE pages. A page should be mostly composition of existing components, not new CSS.

**Primitives (used everywhere):**
| Component | Purpose |
|---|---|
| `BaseButton` | one button, variants: `primary / ghost / danger`, sizes `sm/md` |
| `BaseCard` | sharp card surface, optional header slot |
| `BaseInput` / `BaseSelect` | form fields, consistent focus ring |
| `BaseBadge` | status pills (Active / Pending / KYC) |
| `BaseTable` | dense data table, right-aligned numeric cols |
| `StatTile` | label + big mono number + delta |
| `PriceDelta` | colored +/- value (pos green, neg red) — used dozens of times |
| `Sparkline` | tiny inline trend line |
| `BrandLogo` | mark/wordmark, theme-aware |

**Chrome:**
`MarketingNav`, `MarketingFooter`, `AppTopbar` (logo + index ticker + credit chip + user menu), `AppSidebar` (nav from the reference), `AppStatusBar`.

**Trading (the demo asset — most polish here):**
`TradeCandleChart` (lightweight-charts, in `.client.vue`), `TradeOrderBook` (bid/ask ladder), `TradeTape` (streaming trades), `TradeOrderForm` (buy/sell, market/limit), `TradeDepthChart`, `PositionPanel`, `MarketSelector`.

**Wallet / portfolio:**
`CreditBalanceCard` (unified AI credit + sub-credits + GPU credits), `AssetRow`, `AllocationDonut`, `TransactionRow`.

> Anything that needs `window`/`document` or a canvas (all charts) → suffix `.client.vue` so it never runs on the server.

---

## Phase 5 — Page migration (file → route map)

Now port each HTML file. Each becomes ONE page that **composes components and uses tokens** — strip every inline style and replace with the library.

| HTML file                         | Route             | Layout    | Render    |
| --------------------------------- | ----------------- | --------- | --------- |
| `Exascale Homepage.html`          | `/`               | marketing | prerender |
| `Exascale Signup.html`            | `/signup`         | marketing | prerender |
| `Login.html`                      | `/login`          | marketing | prerender |
| `Trader KYC.html`                 | `/onboarding/kyc` | marketing | ssr:false |
| `Exascale Trading Dashboard.html` | `/trade`          | app       | ssr:false |
| `Exascale Market Detail.html`     | `/markets/[slug]` | app       | ssr:false |
| `Exascale Wallet.html`            | `/wallet`         | app       | ssr:false |
| `Portfolio.html`                  | `/portfolio`      | app       | ssr:false |
| _(future B5)_ Index methodology   | `/benchmark`      | marketing | prerender |
| _(future C4)_ Trade history       | `/history`        | app       | ssr:false |
| _(future C5)_ Buy credits         | `/wallet/deposit` | app       | ssr:false |
| _(404)_                           | `[...slug].vue`   | marketing | —         |

**Per-page migration recipe (apply to every file):**

1. Create the page `.vue`, set `definePageMeta({ layout, middleware })`.
2. Paste the HTML `<body>` content into `<template>`.
3. Delete the page's `<head>`, its `<style>` block, and any chrome (nav/sidebar/footer) — those now come from the layout.
4. Replace every hardcoded color/size/font with `var(--token)`.
5. Replace repeated UI (buttons, cards, tables, numbers) with library components.
6. Move any leftover one-off CSS into a `<style scoped>` block.
7. Confirm the route renders identically to the original mockup.

Route rules in `nuxt.config.ts`:

```ts
routeRules: {
  '/':            { prerender: true },
  '/signup':      { prerender: true },
  '/login':       { prerender: true },
  '/benchmark':   { prerender: true },
  '/trade':       { ssr: false },
  '/markets/**':  { ssr: false },
  '/wallet':      { ssr: false },
  '/wallet/**':   { ssr: false },
  '/portfolio':   { ssr: false },
  '/history':     { ssr: false },
  '/onboarding/**': { ssr: false },
},
```

Mock auth guard (`app/middleware/auth.ts`) protects app pages; `useAuth()` is mock-only for now.

---

## Phase 6 — Mock data composables

All "live" behavior is simulated client-side. One composable per data stream, auto-imported, SSR-safe via `useState`.

| Composable               | Returns                     | Simulation                        |
| ------------------------ | --------------------------- | --------------------------------- |
| `useIndexPrice()`        | EAI-IDX price, change       | Brownian motion, tick every 2–3s  |
| `useMarketPrice(symbol)` | per-market price/change     | same, seeded per symbol           |
| `useCandles(symbol)`     | OHLC array                  | random-walk OHLC, realistic wicks |
| `useOrderBook(symbol)`   | bids/asks ladder            | power-law depth, jitters live     |
| `useTradeTape(symbol)`   | streaming trades            | log-normal sizes, pos/neg side    |
| `usePortfolio()`         | positions/balances          | static seed + small drift         |
| `useCredits()`           | unified + sub + GPU credits | static seed                       |
| `useAuth()`              | isLoggedIn, user            | mock toggle                       |

> **Realism matters.** Prices must look like a market (Brownian motion), not random noise. Order-book depth follows a power law. Trade sizes are log-normal. This is what makes the demo read as credible vs. a toy.

Initialize tickers in `onMounted` only (never SSR). Clean up intervals in `onUnmounted`.

---

## Phase 7 — Consistency QA pass

Walk every route and verify the contract in §10 holds. Then:

```bash
npx nuxt typecheck      # zero TS errors
npm run generate        # SSG build for Netlify
npm run preview         # verify the static output
```

Deploy `.output/public/` to Netlify.

---

## 8. Final project structure

```
exascale-app/
├── app/
│   ├── assets/css/        tokens.css · global.css
│   ├── assets/images/     logo.svg · textures
│   ├── components/
│   │   ├── Base/          Button Card Input Select Badge Table StatTile PriceDelta Sparkline
│   │   ├── Brand/         Logo
│   │   ├── Marketing/     Nav Footer Hero IndexTicker
│   │   ├── App/           Topbar Sidebar StatusBar CreditChip UserMenu
│   │   ├── Trade/         CandleChart.client OrderBook Tape OrderForm DepthChart PositionPanel MarketSelector
│   │   ├── Wallet/        CreditBalanceCard AssetRow TransactionRow
│   │   └── Portfolio/     AllocationDonut.client
│   ├── composables/       useIndexPrice useMarketPrice useCandles useOrderBook useTradeTape usePortfolio useCredits useAuth
│   ├── layouts/           marketing.vue · app.vue
│   ├── middleware/        auth.ts
│   ├── pages/
│   │   ├── index.vue
│   │   ├── signup.vue · login.vue · benchmark.vue
│   │   ├── trade.vue · wallet/index.vue · wallet/deposit.vue
│   │   ├── portfolio.vue · history.vue
│   │   ├── markets/[slug].vue
│   │   ├── onboarding/kyc.vue
│   │   └── [...slug].vue          (404)
│   └── utils/             format.ts (money, credits, pct, tabular)
├── public/                favicon.ico · og-image.png · fonts/
├── nuxt.config.ts
├── tsconfig.json
├── netlify.toml
└── package.json
```

---

## 9. Dependencies

```jsonc
{
  "dependencies": {
    "nuxt": "^4",
    "vue": "latest",
    "lucide-vue-next": "latest", // icons (one system)
    "lightweight-charts": "latest", // candlestick / depth charts
    "@vueuse/core": "latest", // composable utilities
    "date-fns": "latest", // time formatting
  },
  "devDependencies": {
    "vue-tsc": "latest",
    "typescript": "latest",
    "sass": "latest", // only if you use scss
  },
}
```

---

## 10. The consistency contract (hard rules — enforce in QA)

These rules are what make "same design everywhere" true. A page passes only if all hold:

1. **No hardcoded design values.** No raw hex, rgb, or px font/spacing in any page or component. Everything routes through `var(--token)` or the spacing/type scale. Grep for `#` and bare `px` to catch violations.
2. **No chrome in pages.** Nav, sidebar, footer, status bar exist only in layouts. A page never re-declares them.
3. **One component per concept.** One `BaseButton`, one `BaseCard`, one `PriceDelta`, one icon set. If you're tempted to write a second button, add a variant instead.
4. **All numbers are mono + tabular.** Every price, balance, percentage uses `--font-mono` + `tabular-nums`. No exceptions in the trading UI.
5. **Two themes, one token file.** Light/dark differ only by `data-theme` swapping token values. No page hardcodes a theme color.
6. **Charts are client-only.** Every canvas/chart component is `.client.vue`.
7. **Scoped styles only.** Component CSS is `scoped`. No global page-specific CSS leaking.
8. **Spacing from the scale.** Padding/margins use `--sp-*`. No arbitrary `13px`/`27px`.

If you change the brand color or radius, it must be a **one-line edit in `tokens.css`** that updates all 26 screens. If it isn't, the contract is broken somewhere — find the hardcoded value.

---

## 11. Suggested execution prompts (for Claude Code)

Run these in sequence. Don't batch — gate each on the previous passing.

1. _"Read all HTML files in `/source-html/`. Produce a token reconciliation table of every color, font, spacing, radius, and shadow used, flag conflicts, and generate `app/assets/css/tokens.css` + `global.css` using the canonical values from the migration plan. Output a decisions note."_
2. _"Scaffold a Nuxt 4 project per §1 and §8 of the plan: compatibilityVersion 4, strict TS, the two global stylesheets, the canonical `app.vue`, and the empty folder structure."_
3. _"Build the two layouts (`marketing.vue` light, `app.vue` dark) and the chrome components (MarketingNav/Footer, AppTopbar/Sidebar/StatusBar) per §3 and §4."_
4. _"Build the Base primitive components per §4. Render them all on a temporary `/styleguide` page so I can verify consistency."_
5. _"Build the Trade components against mock-data composables (§6). Make the candlestick chart use lightweight-charts in a `.client.vue`."_
6. _"Migrate `Exascale Trading Dashboard.html` → `/trade` using the per-page recipe in §5. Strip all inline styles, compose from the component library, keep the design identical."_
7. _(repeat #6 for each remaining HTML file)_
8. _"Run the §10 consistency contract as a checklist across every route. List violations. Then `npx nuxt typecheck` and `npm run generate`."_

---

## 12. Honest cautions

- **Do the audit (Phase 0) before anything else.** Skipping it is the #1 way the migration reintroduces the exact drift you're trying to kill.
- **The trading dashboard is the demo asset.** Spend disproportionate care on `/trade`. It's what investors judge.
- **Mock data must look real.** Brownian motion, power-law depth, log-normal sizes. Random noise reads as fake and undermines the pitch.
- **Resist Tailwind for v1** unless the audited HTML forces it. Two styling systems = drift returns.
- **The brand book may override tokens later.** When Tai's designer book lands, it should change **only `tokens.css`** — if it requires touching pages, the contract was violated.
