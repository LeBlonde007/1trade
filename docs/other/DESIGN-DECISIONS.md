# Exascale — Design Decisions (Phase 0 Reconciliation Note)

> One-page log of every conflict resolved during the token audit.
> All Vue pages, components, and layouts reference `tokens.css` exclusively.
> If the design changes, edit tokens.css. No page touches.

## Sources audited

| Source | Role | Authority |
|---|---|---|
| `/mnt/f/ex/brand-book.css` | Canonical brand document | **Primary source of truth** |
| `Exascale Trading Dashboard.html` | Dark-mode trading UI reference | Authoritative for dark surfaces |
| `Exascale Wallet.html`, `Market Detail.html` | Dark-mode app screens | Cross-check (consistent with Trading Dashboard) |
| `Exascale Homepage.html`, `Signup.html` | Light-mode marketing | Cross-check (consistent with brand book) |
| Migration plan §0 | Strategic guidance | Token structure (sp-*, fs-*, dur-*) adopted |

## Theme strategy

**Two themes, one token file.** `data-theme="light"` and `data-theme="dark"` swap the
SAME token names; primitives (brand, fonts, radii, spacing, motion) are shared.

- Light theme → marketing pages (`/`, `/signup`, `/login`, `/benchmark`, `/onboarding/kyc`)
- Dark theme → trading app (`/trade`, `/markets/*`, `/wallet`, `/portfolio`, `/history`)

## Conflicts resolved

### 1. Dark surface canvas
- Brand book: `--surface-inverse: #0E0E0E` (used as inverse blocks in light book)
- Trading dashboard: `--canvas: #0A0B0E` (full-app dark surface)
- **Decision:** Use `#0A0B0E` for `--canvas` in dark theme. Slightly cooler than brand book's
  inverse, which is correct for a full-app dark UI vs. a marketing inverse block.

### 2. Positive (green) color
- Brand book: `#16A34A` (Tailwind green-600 — more institutional)
- Trading dashboard: `#19C37D` (slightly brighter for readability on dark canvas)
- **Decision:** Both. `--pos: #16A34A` in light theme; `--pos: #19C37D` in dark theme.
  Same semantic meaning, tuned per surface contrast.

### 3. Negative (red) color
- Brand book: `#DC2626` (Tailwind red-600)
- Trading dashboard: `#EF4444` (Tailwind red-500)
- **Decision:** Same rule as positive. `--neg: #DC2626` light, `--neg: #EF4444` dark.

### 4. Typography stacks
- Brand book lists `'Inter Tight', 'Inter'` for display + body (Inter Tight is heavier).
- Trading mockups use `'Inter'` for body and `'Inter Tight'` for display.
- **Decision:** `--font-sans: 'Inter Tight', 'Inter', system-ui, ...` — Inter Tight covers
  both since its 400/500 weights work as body. Saves loading two stacks.

### 5. Radii
- Both sources agree on sharp/institutional: `2/4/6 px`.
- **Decision:** Adopt brand-book scale exactly: `--radius-sm: 2`, `--radius-md: 4`, `--radius-lg: 6`. Added `--radius-xl: 8`, `--radius-full: 999` as extensions.

### 6. Spacing scale
- Brand book uses ad-hoc spacing in the CSS (`16, 24, 32, 48, 64, 96, 120, 160`).
- **Decision:** Adopt the migration plan's 4px-based scale (`--sp-1` through `--sp-10`)
  — superset of brand book values, no information lost, enforces consistency.

### 7. Brand lime
- Both sources agree: `#C8F25C` with hover `#B8E548`.
- **Decision:** Locked. This is the single signature color and never changes between themes.

### 8. Borders
- Brand book light: `#E4E0D6` (warm) + `#C9C5B8` (stronger).
- Dashboard dark: `rgba(255,255,255,0.08)` + `rgba(255,255,255,0.14)`.
- **Decision:** Both, per theme. Same token names (`--border`, `--border-strong`).

## What is NOT in tokens (intentionally excluded)

- Component-specific values (button heights, input padding, table row height) live in
  `Base*.vue` component styles — they ARE referenced from tokens but expressed at the
  component level so a token never has a "button-only" use case.
- Grid layout values (`--nav-w`, `--app-sb-w`, `--topbar-h`) are layout primitives only —
  used by `layouts/marketing.vue` and `layouts/app.vue`.

## Enforcement (Phase 7 contract)

- `grep -nE '#[0-9a-fA-F]{3,6}' app/**/*.vue` should return **zero** matches outside
  `assets/css/tokens.css`.
- `grep -nE '\b[0-9]+px\b' app/**/*.vue` should match **only** in component overrides
  for known reasons (documented inline).
- Any failure means the design contract is broken — fix the page, not the test.
