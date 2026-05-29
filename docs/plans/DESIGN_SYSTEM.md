# Design System — Exascale frontend (canonical)

> **Every frontend change conforms to this document.** It is the single source of visual truth for
> `apps/web/` (and the current `Exascale Frontend/`). Owned by `trading-frontend`; treated like a
> contract — you consume it, you don't quietly deviate from it. Enforced by `ENGINEERING_STANDARDS.md`
> §4, the `/ex-review` gate, and a pre-commit token check (§7 below).
>
> **Status:** v1 **placeholder** values — swap the token values when the brand book lands; the
> *structure* (token names, scale, rules) stays. The implementation lives in
> `apps/web/app/assets/css/tokens.css` — **a design change is a one-line edit there, never in a page
> or component.**

---

## How to use this (the rule)

1. **Tokens only.** Never hardcode a color, size, font, radius, or spacing value in a component.
   Use `var(--token)` or a shared component. The only file with raw values is `tokens.css`.
2. **Numbers are first-class.** Mono font + `tabular-nums`, right-aligned in tables, semantic
   color + shape (▲/▼). See §1 Number formatting.
3. **Match the surface.** Marketing = generous + light; product/trading UI = dense + dark. §3.
4. **When in doubt, restraint.** A trading venue earns trust through visual sobriety.
5. If you need a value that has no token, **add the token to `tokens.css`** (and here if it's
   systemic) — don't inline it.

---

## 1. Design tokens (A1)

> Placeholder values — swap when brand book lands. Token *names* are stable.

### Dark mode (primary for the trading product)
```
--surface-canvas:    #0A0B0E
--surface-elevated:  #14161B
--surface-overlay:   #1C1F26
--text-primary:      #E8E6E0
--text-secondary:    #9A9A95
--text-tertiary:     #5F5F5C
```

### Light mode (primary for marketing + compute/console dashboard)
```
--surface-canvas:    #F8F7F4
--surface-elevated:  #FFFFFF
--surface-overlay:   #FFFFFF
--text-primary:      #0A0B0E
--text-secondary:    #5F5F5C
--text-tertiary:     #9A9A95
```

### Borders
```
Dark:  rgba(255,255,255,0.08) default · 0.16 strong
Light: rgba(0,0,0,0.08) default · 0.16 strong
Focus ring: #4A90E2
```

### Brand accent (placeholder)
```
--brand-primary:        #C8F25C  (lime)
--brand-primary-hover:  #B8E548
--brand-accent:         #4A90E2
```

### Semantic (CRITICAL for trading UI — always pair color with shape)
```
--positive:         #19C37D   (bullish / gain / buy)   → ▲
--positive-subtle:  rgba(25,195,125,0.12)
--negative:         #EF4444   (bearish / loss / sell)  → ▼
--negative-subtle:  rgba(239,68,68,0.12)
--warning:          #F59E0B
--info:             #4A90E2
```
Color alone never carries meaning (accessibility): positive → ▲, negative → ▼, neutral → text-primary.

### Typography
- **Body / UI:** Inter, system-ui fallback. Numbers use `font-feature-settings: "tnum"`.
- **Display:** Inter (heavier weight) / Inter Display.
- **Monospace:** JetBrains Mono — prices, code, trading data.

### Type scale
```
display-xxl  96px / 700 / -2%    tracking
display-xl   64px / 700 / -1.5%  tracking
display-l    48px / 600 / -1%    tracking
h1           32px / 600
h2           24px / 600
h3           20px / 600
h4           18px / 600
body-l       18px / 400 / 1.55 line-height
body         16px / 400 / 1.55 line-height
body-s       14px / 400
caption      12px / 500
tiny         10px / 600 / 0.08em tracking / UPPERCASE
```

### Spacing (base 4)
`0, 4, 8, 12, 16, 24, 32, 48, 64, 96` px

### Radius — SHARP (institutional)
`none 0 · sm 2px · md 4px · lg 6px`

### Number formatting — always
- `font-variant-numeric: tabular-nums`
- Right-align in tables; decimal-align prices.
- Monospace for prices in the trading UI.
- Positive: green + ▲ · Negative: red + ▼ · Neutral: text-primary.

---

## 2. Mock data shape (A2)

> Use **realistic** mock data, not random numbers — movement signals "real market."
> **Note:** the canonical credit enum (`docs/contracts/credit-types.md`) uses `embeddings`, not
> "niche" — the old "NICHE" row is mapped to **embeddings** below to stay consistent with the contract.

### Prices — Brownian motion with mean reversion
```
AI Index current: $0.001005    (1 AI credit ≈ $0.001)
Sub-credit relative prices:
  text       ≈ $0.001210
  speech     ≈ $0.001200
  image      ≈ $0.008000
  video      ≈ $0.250000
  embeddings ≈ $0.000400      (was "NICHE" in the original spec)
GPU credits:
  gpu_h100 = $2.99   (1 GPU-hour)
  gpu_h200 = $3.49
```

### Order book — power-law depth
- Spread: 1% (e.g. bid $0.000990 / ask $0.001010). 10 levels each side.
- Best bid/ask: 5K–50K credit quantity. Deeper levels grow larger (power law).

### Trade tape — log-normal sizes
- 30–300 trades/minute. Median size 500 credits. Occasional large prints (50K–100K).
- Time format `HH:MM:SS.mmm`.

### Candles — plausible OHLC
- 1-minute candles. ~0.3% range/candle typical. Log-normal volume bars.

### Index history
- Normalized to 1.0000 at launch; currently 1.0024.
- 24h +0.18% · 7d −0.42% · 30d +2.31% · YTD +4.18%.
- Realistic curve: drift up with periodic 5–10% drawdowns.

### Demo account — paper trading (`is_paper: true`)
```
Cash: $10,247.83
Long  50,000 ai_index   @ 0.000980 → +$1.25 (+2.55%)
Long  12,000 text       @ 0.00118  → +$0.24 (+1.69%)
Short  5,000 image      @ 0.00810  → +$0.60 (+1.48%)
Long       8 gpu_h100   @ 2.95     → +$0.32 (+1.36%)
Total value: $10,247.83 · Today P&L: +$23.41 (+0.45%)
```
> The trading mock-data surface belongs to the **kept-warm** trading demo (KW02) — labelled
> "Demo — exchange paused." Platform-console mock data (wallet, catalog, compute, billing) follows
> the same realism bar against the live contracts in `docs/contracts/`.

---

## 3. Aesthetic guardrails (A3)

**Exascale is institutional / technical / financial / modern.**

### Reference points (study these)
✓ Bloomberg Terminal (density, gravitas) · ✓ Polymarket (clean serious trading UI) ·
✓ Stripe (calm restraint) · ✓ Linear (dark-native, technical) · ✓ Vercel (minimal, monochrome).

### Anti-references (NEVER look like these)
✗ Robinhood (consumer-cheerful) · ✗ Coinbase consumer app (playful) · ✗ Web3/crypto (neon,
hex shapes, gradients) · ✗ generic AI startup (purple gradients, neural-net imagery) · ✗ stodgy
bank (navy + gold) · ✗ agency portfolio (asymmetric grids, oversized type for its own sake).

### Principles
1. **Numbers-first.** Numbers get the largest size in their context, mono/tabular font, semantic
   color. Everything else is supporting chrome.
2. **Density > whitespace (for product UI).** Marketing can breathe; product/trading UI is
   Bloomberg-dense — compress padding, tighten line-heights, smaller type than marketing.
3. **Boldness from precision, not maximalism.** Bold = information density, perfect alignment,
   intentional spacing, restrained color — NOT unusual fonts, asymmetric grids, decorative motion.
4. **Restraint.** Favor sobriety; it's how a venue earns trust.
5. **Live data where possible.** Animate price ticks / candle updates / rolling tape — but subtle,
   never jarring.

> The gut check: *"Would Larry Fink mistake this for a real Bloomberg-tier product, or does it
> look like a startup demo?"* Aim for the former.

---

## 4. Output rules (A4) — mockups vs. production

There are two contexts. Don't mix their rules.

### (a) Throwaway HTML mockups / Claude artifacts (exploration, investor demos)
- A single self-contained HTML artifact (inline CSS/JS, CDN libs OK), full-page, renders inline.
- Interactivity where it matters: live-updating mock data (`setInterval` ticks), hover/tabs/dropdowns.
  Simple — vanilla JS or alpine.js; no build tools.
- Allowed CDN libs: `chart.js`, `lightweight-charts` (best for candles), `alpine.js`, `lucide`,
  `tailwindcss` (play CDN — **mockups only**), `Tone.js` (only if audio cues).
- Design for 1440px (trading dashboard), degrade to 1024px; mobile lower priority for v1.
- Density appropriate to the surface. Perfect typography (tabular figures; mono prices; rhythm;
  letter-spacing on small-caps/uppercase). **No placeholder text** — realistic data per §2, never
  "Lorem ipsum" or "TITLE HERE."
- Output the full visual mockup with realistic data — not a wireframe, not a static chart image.

### (b) Production Nuxt app (`apps/web/`) — what actually ships
- **No Tailwind.** `tokens.css` is the sole styling language (per `nuxt.config.ts`). No inline
  styles, no hardcoded values — `var(--token)` or a shared component only.
- `<script setup lang="ts">` + Composition API; charts in `.client.vue` (SSR-unsafe libs).
- One data layer (`useApi()`) flips mock ↔ local ↔ staging with **zero UI change** — build mockup
  realism into the mock source, not into the components.
- Migrating a mockup into Nuxt = translate inline styles → tokens, Tailwind → token classes/components,
  vanilla JS → composables. The *look* is preserved; the *implementation* follows (b).

---

## 5. Theming mechanics

- Two themes on **one token foundation**: marketing/console = light, trading product = dark
  (`nuxt.config.ts` already wires light marketing + dark app).
- Theme switch = swapping the token values under a `[data-theme]` selector in `tokens.css`; pages
  and components never branch on theme.

---

## 6. Accessibility floor (WCAG AA)

- Color never the sole signal (semantic ▲/▼, icons, labels).
- Contrast ≥ AA for text on its surface (the placeholder palette is chosen to pass; re-verify when
  the brand book swaps values).
- Keyboard navigable; visible focus ring (`#4A90E2`); screen-reader labels on icon-only controls.

---

## 7. Enforcement (so this isn't just prose)

| Rule | Enforced by |
|---|---|
| Tokens only — no hardcoded hex/rgb/px-font in components | pre-commit `design-tokens-guard` (flags raw colors in `apps/web/**/*.{vue,ts}` outside `tokens.css`/`global.css`) + `ENGINEERING_STANDARDS.md` §4 |
| Numbers: mono + `tabular-nums`, sentence case, semantic ▲/▼ | `trading-frontend` review + `/ex-review` |
| Conforms to this doc | `/ex-start` loads it for any `apps/web/` work; `/ex-review` checks it; it's in `trading-frontend` Definition of Done |
| Institutional aesthetic (no crypto-flashy / anti-references) | `trading-frontend` review (the Larry Fink gut check) |
| Mock realism (no random numbers / no Lorem) | this doc §2 + review |

When the **brand book** arrives: update `tokens.css` values (and the placeholder values here),
record it as an ADR if it changes structure, and re-verify contrast. No component should need to
change — that's the point of tokens.
