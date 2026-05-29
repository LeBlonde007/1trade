# Exascale — Nuxt 4 App

The commodity market for AI compute credits, built with Nuxt 4.

## Quick Start

```bash
# Install dependencies
npm install

# Dev server (localhost:3000)
npm run dev

# Build for production (SSR)
npm run build

# Static site generation (for Netlify)
npm run generate

# Type check
npm run typecheck
```

## Architecture

- **`app/`** — All application code (pages, components, layouts, composables)
- **`app/pages/`** — File-based routing (each `.vue` file = a route)
- **`app/layouts/`** — Page wrappers (marketing, app shell)
- **`app/components/`** — Reusable Vue components (auto-imported)
- **`app/composables/`** — Stateful logic with `useState`, `useFetch`
- **`app/assets/css/`** — Global styles and design tokens
- **`public/`** — Static files served as-is

## Pages

- `/` — Homepage (marketing)
- `/signup` — Account creation (marketing)
- `/login` — Sign in (served from `/public/login/`)
- `/trade` — Trading dashboard (app shell)
- `/wallet` — Asset balances (app shell)
- `/portfolio` — Account overview (app shell)
- `/markets/[slug]` — Market detail (dynamic route, app shell)
- `/index` — AI Index (coming soon)
- `/compute` — GPU provisioning (coming soon)
- `/inference` — Model playground (coming soon)

## Deployment

### Netlify

1. Push to GitHub
2. Connect repository to Netlify
3. Build command: `npm run generate`
4. Publish directory: `.output/public`
5. Deploy!

The `netlify.toml` file handles routing and caching.

## Design System

All colors, typography, and spacing use CSS custom properties defined in:
- `app/assets/css/tokens.css` — Dark theme (app), Light theme (marketing)
- `app/assets/css/global.css` — Typography, resets, utilities

### Theme Variables

**Dark theme (trading app):**
- `--canvas` — Background
- `--elevated` — Card backgrounds
- `--text` — Primary text
- `--text-2` — Secondary text
- `--brand` — Lime accent
- `--pos` — Green (gains)
- `--neg` — Red (losses)

**Light theme (marketing):**
- Same tokens, different values
- Apply with `.light-theme` class

## TypeScript

TypeScript is configured and ready. Files use `.ts` and `.vue <script setup lang="ts">` by default.

Enable type checking in dev:
```bash
npm run typecheck
```

## Lucide Icons

Icon names from [lucide.dev](https://lucide.dev). Use component names:

```vue
<component :is="'CandlestickChart'" :size="20" />
<component :is="'Bell'" :size="18" />
<component :is="'Settings'" :size="20" />
```

## Next Steps

1. **Implement real data** — Replace mock data in pages with API calls
2. **Add authentication** — Wire up `/login` and `/signup` to backend
3. **Convert complex pages** — Migrate full HTML mockups into Vue components
4. **Add charts** — Implement lightweight-charts, chart.js in trading/market pages
5. **Setup tests** — Add Vitest and component testing

---

Built with [Nuxt 4](https://nuxt.com/docs/4.x/guide)
