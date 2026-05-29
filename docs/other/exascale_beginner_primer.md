# Exascale — Beginner's Primer

> "I've never worked on trading software or modern frontend frameworks.
> What do I need to know before I touch the codebase?"
>
> This is that primer. Read top-to-bottom in ~30 minutes and you'll have
> enough vocabulary, mental model, and project orientation to make safe
> changes.

---

## 1. What Exascale actually is

A **trading venue** where the thing being traded is **AI compute** — tokens,
GPU-hours, image generations, etc. — the way commodity exchanges trade oil
or wheat.

Three groups of users meet on it:

- **AI companies** that need to *buy* compute predictably (instead of paying
  whatever AWS charges this hour).
- **Datacenters** that own GPUs and want to *sell* their unused capacity to
  the highest bidder.
- **Traders** in the middle, betting on whether compute will get more or
  less expensive — exactly like commodity speculators.

The unit traded is a **credit**. `1 AI credit ≈ $0.001 USD` (rough peg, the
market sets the actual rate). One credit buys you one dollar's worth of
work — could be 1M text tokens, or 100 image generations, or 0.0003 hours
of an H100 GPU. The price of each "sub-credit" floats.

**Why this is interesting:** AI compute today has no public price discovery.
A token from OpenAI costs whatever they say it costs today. Exascale makes
it a transparent commodity market with quotes, depth, and historical curves.

---

## 2. The glossary you must know (trading words)

You don't need an MBA. You need ~15 terms.

| Term | Plain-English meaning |
|---|---|
| **Order** | An instruction to buy or sell. Either *market* (right now at whatever price) or *limit* (only at $X or better). |
| **Bid** | The highest price a buyer is willing to pay right now. |
| **Ask** | The lowest price a seller will accept right now. |
| **Spread** | Ask minus bid. Tight spread = liquid market. Wide spread = thin. |
| **Order book** | The list of all current bids + asks, sorted by price. Has *depth* — more quantity available the further you walk from the mid-price. |
| **Mid price** | The midpoint between best bid and best ask. The "real" price most charts show. |
| **Fill** | The moment two orders match. A buyer's bid matches a seller's ask → trade happens. |
| **Tape** | The running feed of every fill the venue prints, in chronological order. |
| **Position** | What you currently hold after your buys / sells. "Long 50,000 AI credits" = you bought and still hold them. "Short 5,000" = you sold credits you didn't own (advanced). |
| **P&L** | Profit & Loss. Unrealized = on paper, position not yet closed. Realized = locked in. |
| **Slippage** | The gap between the price you expected and the price you actually got. Market orders against a thin book = high slippage. |
| **Index** | A weighted basket — Exascale's `EAI-IDX` blends all the major credit families into one number, like the S&P 500. |
| **Spot** | Buy/sell for delivery right now. |
| **Forward / futures** | Buy/sell today for delivery later (e.g., 30-day H100 forward = locks in GPU rate for delivery a month out). |
| **Paper trading** | Trading with fake money to practice. We give every new account `$10,000` in paper credit. |
| **Settlement** | The bookkeeping that locks in a trade — money + credits actually change hands. T+0 = same day, T+1 = next business day. |

**Two color rules everyone in finance follows** and our UI follows too:

- Green + ▲ = price went UP (bullish, good for longs).
- Red + ▼ = price went DOWN (bearish, good for shorts).
- Pair color with a shape (▲▼ arrows) so colorblind users aren't excluded.

---

## 3. The three personas (who the screens are for)

These names recur throughout the docs and the tour scripts. Memorize them.

- **Jordan Park — independent quant trader.** Opens the marketing site,
  signs up, KYC, lands on `/trade`. Cares about: live chart, order book,
  fast order entry, P&L vs the index.
- **Maya Chen — VP Engineering at an AI startup.** Procures compute in bulk.
  Cares about: team budgets, audit log, billing dashboard, API keys, SSO.
  Lives in `/enterprise/*` pages.
- **Tom Reyes — capacity ops at a datacenter.** Sells GPU-hours into the
  venue. Cares about: fill rate, settlement statements, hardware health.
  Lives in `/datacenter/*` pages.

Every screen in the project belongs to one of these three audiences
(plus the public marketing site that lives at `/`).

---

## 4. The tech stack (one paragraph each, no jargon)

### Nuxt 4
A framework for building websites with Vue. The killer feature for us:
**file-based routing.** Create `app/pages/foo.vue` → you instantly have
a page at `/foo`. Create `app/pages/users/[id].vue` → `/users/123` works
and `id` is `'123'`.

### Vue 3
The component library. Each file ends in `.vue` and has three blocks:
- `<script setup>` — JavaScript (or TypeScript) that defines state + logic
- `<template>` — the HTML the component renders
- `<style scoped>` — CSS that *only* applies inside this component

Example minimal component:

```vue
<script setup>
const count = ref(0)
function increment() { count.value++ }
</script>

<template>
  <button @click="increment">Clicks: {{ count }}</button>
</template>

<style scoped>
button { padding: 8px 14px; }
</style>
```

Three concepts to know:
- **`ref()`** — wraps a value so Vue knows when it changes and re-renders.
  Read/write via `.value` in JS. In the template, just use the name.
- **`v-if` / `v-for`** — show conditionally / loop. `v-for="x in items"`.
- **`@click`** — listen for events. Same as `addEventListener('click', ...)`.

### TypeScript
JavaScript with type annotations. Don't be intimidated — 90% of the time
it's `const name: string = 'Jordan'` (just add `: type` after a variable).

**Important catch in this codebase:** Vue templates can only contain plain
JavaScript expressions, NOT TypeScript. So this is OK in `<script>`:
```ts
const order = orders.find(o => o.id === id) as Order
```
…but this in the `<template>` will silently break:
```vue
{{ (order as Order).price }}   <!-- BAD -->
```
Use the `<script>` block for any casting, then pass the cleaned-up value
to the template.

### Composables (`useFoo()`)
Reusable bits of logic that live in `app/composables/`. They're just
functions. Convention is the name starts with `use`.

```ts
// app/composables/useCounter.ts
export function useCounter() {
  const count = ref(0)
  return { count, increment: () => count.value++ }
}
```

Any component anywhere can call `useCounter()` and get the state.
We use this for: command palette state, notification drawer, tour state,
toast queue, user session, etc.

### Auto-import
Nuxt looks at `app/components/`, `app/composables/`, and a few standard
Vue APIs (`ref`, `computed`, `watch`, `onMounted`, …) and **imports them
for you**. You can use them in any file without typing `import`. The
name follows the path: `components/Base/Button.vue` becomes `<BaseButton>`,
`components/App/Topbar.vue` becomes `<AppTopbar>`.

### SSR / SPA
SSR = server renders the page once on the first request (good for SEO).
SPA = the browser handles everything after that. Nuxt does both by default.
Some pages (`/trade`, anything with live data) opt out of SSR via
`routeRules` in `nuxt.config.ts`.

---

## 5. Project layout

```
Exascale Frontend/
├── app/
│   ├── app.vue              ← top of the component tree; renders global overlays
│   ├── assets/css/
│   │   ├── tokens.css       ← design variables (colors, spacing, type) — read this first
│   │   └── global.css       ← base styles applied everywhere
│   ├── components/
│   │   ├── App/             ← in-app chrome (Topbar, Sidebar, CommandPalette, etc.)
│   │   ├── Base/            ← primitives (Button, Card, Input, Badge…)
│   │   ├── Brand/           ← logos
│   │   ├── Marketing/       ← public-site nav + footer
│   │   └── Trade/           ← trading-specific (CandleChart, OrderBook, OrderForm…)
│   ├── composables/         ← reusable logic (useCommandPalette, useToasts, useTour…)
│   ├── data/                ← static data files (tour scripts, market catalogs)
│   ├── layouts/
│   │   ├── app.vue          ← dark trading layout (sidebar + topbar + main)
│   │   └── marketing.vue    ← light marketing layout (nav + content + footer)
│   ├── pages/               ← every file here = a URL route
│   └── utils/               ← pure helper functions (formatPrice, formatUSD…)
├── nuxt.config.ts           ← Nuxt config (route rules, modules, fonts)
├── package.json
└── public/                  ← static files served as-is
```

**The mental shortcut:** if you're adding a screen, look in `app/pages/`.
If you're adding a reusable piece of UI, look in `app/components/`. If
you're adding shared state or logic, look in `app/composables/`.

---

## 6. Design system rules (non-negotiable)

These are the rules every screen in the project follows. Break them and
the screen feels off.

1. **Sharp corners.** `border-radius: 2px` (or 4px max for cards). NEVER
   pill-shaped buttons. Institutional, not consumer.
2. **Numbers are king.** Every price, quantity, percentage, timestamp uses
   `font-family: var(--font-mono)` (JetBrains Mono) and
   `font-variant-numeric: tabular-nums`. Right-align numbers in tables.
3. **Three text colors per theme.** `var(--text)` for main, `var(--text-2)`
   for secondary, `var(--text-3)` for muted/tertiary. Don't pull other
   greys out of thin air.
4. **Green = up, red = down**, paired with ▲ / ▼. `var(--pos)` and
   `var(--neg)`. Never blue for "increase" or red for "alert" in a
   trading context — color carries meaning here.
5. **Brand lime.** `#C8F25C` (`var(--brand)`) is the one accent. Used
   sparingly on primary CTAs and the small mark in the logo. Don't paint
   walls with it.
6. **Density over whitespace** *for product UI* (`/trade`, `/portfolio`,
   `/compute`, `/inference`). Marketing pages (`/`, `/benchmark`) are
   generous; trading pages should look like Bloomberg, not like a
   consumer app.
7. **Light vs dark.** The whole trading product is **always dark**
   (`app.vue` layout locks `<html data-theme="dark">`). Marketing is
   always light. Enterprise admin (`/enterprise/*`) is light because
   compliance officers expect a "documenty" feel.
8. **Mono for everything code-like.** Email addresses, hashes, IPs,
   instance IDs, market symbols — all mono.

When in doubt, **open `app/assets/css/tokens.css`** — every color,
spacing, font, and radius is there.

---

## 7. Your first change (concrete walkthrough)

Say you want to add a new page at `/hello`:

1. Create `app/pages/hello.vue`:
   ```vue
   <script setup lang="ts">
   definePageMeta({ layout: 'app' })  // dark trading layout
   useHead({ title: 'Hello — Exascale' })

   const greeting = ref('Welcome to Exascale.')
   </script>

   <template>
     <div class="page">
       <h1>{{ greeting }}</h1>
       <p class="mono">$0.001005 ▲ 0.18%</p>
     </div>
   </template>

   <style scoped>
   .page { padding: 24px; }
   h1 { font-size: 22px; font-weight: 600; }
   .mono {
     font-family: var(--font-mono);
     font-variant-numeric: tabular-nums;
     color: var(--pos);
   }
   </style>
   ```
2. Start the dev server: `cd "/mnt/f/ex/Exascale Frontend" && npm run dev`
3. Visit `http://localhost:3000/hello`. The page is live with hot reload.
4. Notice you didn't `import` anything — `ref`, `definePageMeta`, `useHead`
   are auto-imported by Nuxt.

---

## 8. Common mistakes to avoid

- **Hardcoding colors.** `color: #FFFFFF` will look right today and wrong
  in light mode tomorrow. Always use `var(--text)`, `var(--pos)`, etc.
- **TypeScript casts in templates.** See §4. Won't error; will silently
  fail to compile parts of the template.
- **Adding `.value` in templates.** Inside `<template>`, refs unwrap
  automatically — just use the name (`{{ count }}`, not `{{ count.value }}`).
- **Forgetting `definePageMeta({ layout: 'app' })`** on a trading page.
  Without it, your page won't have the topbar, sidebar, or dark theme.
- **Calling lifecycle hooks after `await`.** This errors:
  ```ts
  onMounted(async () => {
    const x = await fetch(...)
    onUnmounted(() => { ... })   // ← BROKEN
  })
  ```
  Register `onUnmounted` at the top level of `<script setup>`, before any
  await happens.
- **Using `setInterval` without cleanup.** Always pair with `onUnmounted`
  that clears it.

---

## 9. Where to look for what

| If you need to… | Look at |
|---|---|
| Understand the product strategy | `docs/exascale_persona_demo_flows.md` |
| Find every screen and its status | `docs/exascale_mvp_screens_checklist.md` |
| Get prompts to design a new screen | `docs/exascale_mvp_screens_prompts.md` (MVP) + `docs/exascale_v15_screens_prompts.md` (v1.5) |
| See what's missing from the full exchange demo | `docs/exascale_exchange_demo_gaps.md` |
| Change a color, spacing, font | `app/assets/css/tokens.css` (one file rules everything) |
| Add a new page | `app/pages/<your-route>.vue` |
| Add a reusable component | `app/components/Base/` (primitives) or `app/components/App/` (chrome) |
| Add shared state across pages | `app/composables/useFoo.ts` |
| Change which pages get SSR | `nuxt.config.ts` → `routeRules` |
| Add an icon | We use `lucide-vue-next`. `import { CheckCircle2 } from 'lucide-vue-next'` then `<CheckCircle2 :size="16" />` in template |
| Draw a chart | `chart.js` for line / bar, `lightweight-charts` for candlesticks (both already in `package.json`) |

---

## 10. The three commands you'll actually run

```bash
cd "/mnt/f/ex/Exascale Frontend"

npm install           # one time, after pulling new deps
npm run dev           # start dev server at http://localhost:3000
npm run build         # production build (rarely needed locally)
```

That's the loop: `npm run dev` → edit a file → save → browser reloads
automatically. No build step, no compiler dance, no waiting.

---

## 11. Mental model — what's special about a trading UI

A few things make trading UIs feel different from normal web apps:

- **Numbers update in real time.** Use `setInterval` + `ref` to simulate
  in mockups. Use subtle flashes (`--pos-soft`, `--neg-soft` background
  on cells that just changed). Animations should be quiet, not jarring.
- **Layouts are dense.** Look at `/trade` — chart + book + tape + form +
  positions all visible on one screen. Don't be afraid of 11–13px fonts
  in tables. Bloomberg ships *10*-pixel cells in places.
- **Mono digits prevent layout shift.** Without `tabular-nums`, a "1"
  and a "2" have different widths and your prices jitter as they tick.
- **Latency matters socially.** Even in a mock, a price that takes 600ms
  to update feels broken. Use 1–3 second intervals so changes feel
  rhythmic without being chaotic.
- **Negative space conveys trust.** Borrowed from Bloomberg / Vercel /
  Linear / Stripe — restrained color, perfect alignment, no decoration
  for decoration's sake. Boldness comes from *precision*, not from
  oversized type or asymmetric grids.

---

## 12. If you remember nothing else

1. **Read `tokens.css` first.** It's the design system in one file.
2. **Pages live in `app/pages/`.** File path = URL.
3. **Sharp corners, mono numbers, green up / red down with arrows.**
4. **In-app = dark, marketing = light, enterprise admin = light.** Set
   via the layout — don't fight it per-page.
5. **`npm run dev`** and watch it hot-reload as you go.

You'll absorb the rest by reading existing pages. `/portfolio`, `/trade`,
and `/enterprise/billing` are good examples of the three main density
levels — start there.

Welcome to the venue.
