# Nuxt 4 — Complete Reference & Best Practices

> Source: https://nuxt.com/docs/4.x/guide
> Compiled: 2026-05-20
> Purpose: Reference guide for migrating 1Trade HTML mockups into a Nuxt 4 app.

---

## Table of Contents

1. [Installation](#1-installation)
2. [Project Structure](#2-project-structure)
3. [app.vue — The Entry Point](#3-appvue--the-entry-point)
4. [Layouts](#4-layouts)
5. [Pages & File-Based Routing](#5-pages--file-based-routing)
6. [Components](#6-components)
7. [Composables](#7-composables)
8. [Auto-Imports](#8-auto-imports)
9. [Styling](#9-styling)
10. [TypeScript](#10-typescript)
11. [Rendering Modes](#11-rendering-modes)
12. [nuxt.config.ts — Key Options](#12-nuxtconfigts--key-options)
13. [Navigation & Routing](#13-navigation--routing)
14. [Assets vs Public](#14-assets-vs-public)
15. [Best Practices Checklist](#15-best-practices-checklist)

---

## 1. Installation

### Requirements
- Node.js **22.x or newer** (active LTS)
- VS Code + Volar (Vue) extension, or WebStorm

### Scaffold a new project

```bash
# Creates a new Nuxt 4 project in a folder called <project-name>
npm create nuxt@latest <project-name>

# Start the dev server (opens browser automatically)
cd <project-name>
npm run dev -- -o
```

### Build & Deploy

```bash
# Type-check only (no build)
npx nuxt typecheck

# Development server
npm run dev

# Production build (SSR)
npm run build

# Static site generation (for Netlify / CDN deploy)
npm run generate

# Preview the generated static output
npm run preview
```

---

## 2. Project Structure

```
my-nuxt-app/
├── app/                    ← All your application code lives here (Nuxt 4)
│   ├── assets/             ← Files processed by build tool (CSS, fonts, images)
│   │   └── css/
│   │       └── tokens.css  ← Design tokens (CSS custom properties)
│   ├── components/         ← Auto-imported Vue components
│   │   ├── App/
│   │   │   ├── Sidebar.vue ← Used as <AppSidebar />
│   │   │   └── Topbar.vue  ← Used as <AppTopbar />
│   ├── composables/        ← Auto-imported Vue composables (use-prefix)
│   │   └── useMarket.ts
│   ├── layouts/            ← Page layout wrappers
│   │   ├── default.vue     ← Fallback layout (every page uses this unless overridden)
│   │   ├── marketing.vue   ← Light theme — homepage, signup, login
│   │   └── app.vue         ← Dark theme — trading dashboard, wallet, portfolio
│   ├── middleware/         ← Route middleware (auth guards, redirects)
│   │   └── auth.ts
│   ├── pages/              ← File-based routing (each .vue = a route)
│   │   ├── index.vue       ← Route: /
│   │   ├── signup.vue      ← Route: /signup
│   │   ├── trade.vue       ← Route: /trade
│   │   ├── wallet.vue      ← Route: /wallet
│   │   ├── portfolio.vue   ← Route: /portfolio
│   │   └── markets/
│   │       └── [slug].vue  ← Route: /markets/:slug (dynamic)
│   └── utils/              ← Auto-imported utility functions (non-reactive)
├── public/                 ← Static files served as-is (no build processing)
│   └── login/
│       └── index.html      ← Standalone HTML pages served directly
├── server/                 ← Server-only code (API routes, middleware)
│   └── api/
│       └── index.ts
├── app.vue                 ← Root component (wraps NuxtLayout + NuxtPage)
├── nuxt.config.ts          ← Nuxt configuration
├── tsconfig.json           ← TypeScript config (extends .nuxt/tsconfig.app.json)
├── netlify.toml            ← Netlify deployment config
└── package.json
```

> **Nuxt 4 key change**: Application code now lives in `app/` subdirectory
> (not at the root like Nuxt 3). The `pages/`, `components/`, `layouts/`,
> `composables/` directories all go inside `app/`.

---

## 3. app.vue — The Entry Point

The root component. Everything you add here (JS and CSS) is global and
included on every page.

```vue
<!-- app.vue -->
<template>
  <!--
    NuxtLayout reads the `layout` property set by each page's definePageMeta().
    Falls back to layouts/default.vue if no layout is specified.
    NuxtPage renders the matched page component for the current route.
    NuxtLoadingIndicator shows a top progress bar during page navigation.
  -->
  <NuxtLoadingIndicator />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

> **Rule**: `NuxtLayout` must wrap `NuxtPage`. If you add a header or footer
> directly in `app.vue` it will appear on EVERY page — put shared chrome in
> layouts instead.

---

## 4. Layouts

Layouts wrap pages with reusable UI (navbars, sidebars, footers).
Files live in `app/layouts/`. They are loaded **asynchronously** by default.

### Default layout (`layouts/default.vue`)

Applied to every page that doesn't specify a layout.

```vue
<!-- app/layouts/default.vue -->
<template>
  <!-- Single root element is required for layout transitions -->
  <div>
    <AppTopbar />
    <!--
      <slot /> renders the page's content here.
      This is the only required part of any layout.
    -->
    <slot />
    <AppFooter />
  </div>
</template>
```

### Named layout (`layouts/marketing.vue`)

```vue
<!-- app/layouts/marketing.vue -->
<template>
  <!-- Light-mode marketing shell — homepage, signup, login -->
  <div class="marketing-shell">
    <MarketingNav />
    <main>
      <slot />
    </main>
  </div>
</template>

<style>
.marketing-shell {
  background: var(--surface-canvas);
  min-height: 100vh;
}
</style>
```

### Named layout (`layouts/app.vue`)

```vue
<!-- app/layouts/app.vue -->
<template>
  <!-- Dark-mode trading app shell — authenticated pages -->
  <div class="app-shell">
    <AppTopbar />
    <AppSidebar />
    <main class="app-main">
      <!--
        <slot /> renders whichever page is currently active.
        Each page (trade, wallet, markets) fills this slot.
      -->
      <slot />
    </main>
  </div>
</template>
```

### How a page selects its layout

```vue
<!-- app/pages/trade.vue -->
<script setup lang="ts">
// definePageMeta is a COMPILER MACRO — runs at build time.
// It sets metadata for this route. layout must be a static string.
definePageMeta({
  layout: 'app',          // Uses layouts/app.vue
})
</script>
```

```vue
<!-- app/pages/index.vue -->
<script setup lang="ts">
definePageMeta({
  layout: 'marketing',   // Uses layouts/marketing.vue
})
</script>
```

### Passing props to layouts (Nuxt 4.4+)

```vue
<script setup lang="ts">
definePageMeta({
  layout: {
    name: 'app',
    props: {
      showSidebar: true,
      title: 'Trading Dashboard',
    },
  },
})
</script>
```

### Rules
- Layout files **must have a single root element** (transitions break otherwise)
- Layout names are normalized to **kebab-case** (`someLayout` → `some-layout`)
- Use `layout: false` in `definePageMeta` to completely opt out of layouts
- Nested directory: `layouts/desktop/default.vue` → name is `desktop-default`

---

## 5. Pages & File-Based Routing

Every `.vue` file in `app/pages/` becomes a route. No router config needed.

### Basic page

```vue
<!-- app/pages/wallet.vue → route: /wallet -->
<script setup lang="ts">
// Page-level metadata — layout, middleware, title, transitions
definePageMeta({
  layout: 'app',
  title: 'Wallet',
})
</script>

<template>
  <div class="wallet-page">
    <h1>Wallet</h1>
  </div>
</template>
```

### Dynamic routes

```
app/pages/markets/[slug].vue   → /markets/eai-idx, /markets/text-spot
app/pages/trade/[symbol].vue   → /trade/EAI-IDX
```

```vue
<!-- app/pages/markets/[slug].vue -->
<script setup lang="ts">
// useRoute() gives access to the current route object
// route.params.slug = 'eai-idx' when visiting /markets/eai-idx
const route = useRoute()
const slug = computed(() => route.params.slug as string)

definePageMeta({
  layout: 'app',
})
</script>

<template>
  <div>
    <h1>{{ slug.toUpperCase() }}</h1>
  </div>
</template>
```

### Catch-all route (404 handler)

```
app/pages/[...slug].vue   → matches /any/path/that/doesnt/exist
```

```vue
<!-- app/pages/[...slug].vue -->
<script setup lang="ts">
definePageMeta({ layout: 'marketing' })
const route = useRoute()
// route.params.slug is an array: ['some', 'deep', 'path']
</script>

<template>
  <div>
    <h1>Page not found</h1>
    <NuxtLink to="/">Go home →</NuxtLink>
  </div>
</template>
```

### definePageMeta options

```ts
definePageMeta({
  layout: 'app',                  // Which layout to use
  title: 'Trading Dashboard',     // Custom meta title
  middleware: 'auth',             // Run auth.ts middleware before loading
  keepalive: true,                // Preserve component state on route change
  pageTransition: { name: 'fade', mode: 'out-in' }, // Page transition
  
  // Custom route validation — return false to show 404
  validate: (route) => {
    return typeof route.params.slug === 'string'
  },
})
```

### Client-only & server-only pages

```
pages/admin.client.vue    ← Only renders on the client (no SSR)
pages/feed.server.vue     ← Only renders on the server (excluded from client bundle)
```

---

## 6. Components

All `.vue` files in `app/components/` are **auto-imported** — no need to
`import` them manually anywhere.

### Naming convention (path → component name)

```
app/components/Button.vue          → <Button />
app/components/App/Sidebar.vue     → <AppSidebar />
app/components/App/Topbar.vue      → <AppTopbar />
app/components/Trade/OrderBook.vue → <TradeOrderBook />
```

> Nested directory structure is reflected in the component name.
> This avoids name collisions across a large codebase.

### Basic component

```vue
<!-- app/components/App/Sidebar.vue -->
<script setup lang="ts">
// useRoute() returns the current route — use it to highlight the active nav item
const route = useRoute()

// Sidebar nav items with their corresponding routes
const navItems = [
  { icon: 'candlestick-chart', label: 'Trade',     to: '/trade' },
  { icon: 'list',              label: 'Markets',   to: '/markets/eai-idx' },
  { icon: 'trending-up',       label: 'Index',     to: '/index' },
  { icon: 'briefcase',         label: 'Portfolio', to: '/portfolio' },
  { icon: 'clock',             label: 'History',   to: '/history' },
  { icon: 'wallet',            label: 'Wallet',    to: '/wallet' },
  { icon: 'server',            label: 'Compute',   to: '/compute' },
  { icon: 'message-square',    label: 'Inference', to: '/inference' },
]

// Returns true if the given route path matches the current URL
const isActive = (to: string) => route.path.startsWith(to)
</script>

<template>
  <nav class="sidebar">
    <NuxtLink
      v-for="item in navItems"
      :key="item.to"
      :to="item.to"
      :class="['sidebar-item', { active: isActive(item.to) }]"
      :aria-label="item.label"
    >
      <!-- Lucide icon by name -->
      <component :is="item.icon" :size="20" />
      <span class="sidebar-label">{{ item.label }}</span>
    </NuxtLink>
  </nav>
</template>
```

### Lazy loading (defer until needed)

```vue
<!-- Prefix with Lazy — component chunk is only loaded when it renders -->
<LazyTradeOrderBook v-if="showOrderBook" />
<LazyPortfolioChart />
```

### Client-only component

```vue
<!-- app/components/LiveTicker.client.vue -->
<!-- This component only runs in the browser — safe to use window/document here -->
<script setup lang="ts">
// window is available here — this code never runs on the server
const price = ref(0.001005)
onMounted(() => {
  setInterval(() => {
    price.value += (Math.random() - 0.5) * 0.000002
  }, 3000)
})
</script>

<template>
  <span class="price">{{ price.toFixed(6) }}</span>
</template>
```

### Dynamic components

```vue
<script setup lang="ts">
import { resolveComponent } from 'vue'

// Use resolveComponent when the component name is dynamic
const chartType = ref<'line' | 'candle'>('candle')
const ChartComponent = computed(() =>
  resolveComponent(chartType.value === 'candle' ? 'TradeCandleChart' : 'TradeLineChart')
)
</script>

<template>
  <component :is="ChartComponent" />
</template>
```

---

## 7. Composables

Reusable stateful logic that can share reactive state across components.
Files in `app/composables/` are auto-imported.

### Creating a composable

```ts
// app/composables/useMarketPrice.ts
// Naming: file starts with 'use', will be available as useMarketPrice()

export const useMarketPrice = (symbol: string) => {
  // useState persists state across SSR and client — safe for shared state
  const price = useState(`price-${symbol}`, () => 0)
  const change = useState(`change-${symbol}`, () => 0)

  // Simulate live price ticks — only runs client-side
  const startTicking = () => {
    setInterval(() => {
      const delta = (Math.random() - 0.5) * 0.000002
      price.value += delta
      change.value = (delta / price.value) * 100
    }, 3000)
  }

  return { price: readonly(price), change: readonly(change), startTicking }
}
```

```vue
<!-- Used in any component without importing -->
<script setup lang="ts">
const { price, change, startTicking } = useMarketPrice('EAI-IDX')

onMounted(() => {
  startTicking() // Start ticking only after the DOM is ready
})
</script>
```

### useState — SSR-safe shared state

```ts
// useState key must be unique across the whole app
// Initial value is only used on first call — subsequent calls return existing state
const counter = useState('counter', () => 0)
counter.value++
```

### useFetch — data fetching with SSR support

```ts
// Runs on both server (for SSR) and client (for client-side navigation)
// Data is automatically deduped and cached
const { data: markets, pending, error } = await useFetch('/api/markets')
```

### useAsyncData — custom async operations

```ts
// When you need more control than useFetch provides
const { data } = await useAsyncData('markets', async () => {
  const res = await $fetch('/api/markets')
  return res.markets
})
```

### Rules for composables

```ts
// ✅ Correct — called inside <script setup> or a composable
const route = useRoute()

// ✅ Correct — called inside a composable function
export const useMyComposable = () => {
  const config = useRuntimeConfig() // Fine here
}

// ❌ Wrong — called at module top level (outside Nuxt context)
const config = useRuntimeConfig() // Throws "Nuxt instance is unavailable"
```

---

## 8. Auto-Imports

Nuxt automatically imports from these directories — no `import` statement needed:

| Source | What's imported |
|--------|----------------|
| `app/components/` | All Vue components |
| `app/composables/` | All composables (files starting with `use`) |
| `app/utils/` | All utility functions |
| Vue | `ref`, `computed`, `reactive`, `watch`, `onMounted`, etc. |
| Nuxt | `useFetch`, `useRoute`, `useRouter`, `useState`, `navigateTo`, `definePageMeta`, etc. |

### Explicit import (when auto-import conflicts)

```ts
// Import from #imports to be explicit about where something comes from
import { ref, computed } from '#imports'
import { useFetch } from '#imports'
```

### Auto-import from third-party packages

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  imports: {
    presets: [
      {
        from: 'date-fns',
        imports: ['format', 'parseISO'],
      },
    ],
  },
})
```

### Disable auto-imports (when you want explicit control)

```ts
export default defineNuxtConfig({
  imports: {
    autoImport: false, // Disable composable auto-import
  },
  components: {
    dirs: [],           // Disable component auto-import
  },
})
```

---

## 9. Styling

### Global CSS (applied to every page)

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  css: [
    '~/assets/css/tokens.css',  // Design tokens (CSS custom properties)
    '~/assets/css/global.css',  // Reset, typography, base styles
  ],
})
```

### CSS custom properties (design tokens file)

```css
/* app/assets/css/tokens.css */

/* ─── Dark theme (trading app) ─── */
:root[data-theme="dark"],
.dark {
  --canvas:    #0A0B0E;
  --elevated:  #14161B;
  --overlay:   #1C1F26;
  --hover:     #1F2229;
  --text:      #E8E6E0;
  --text-2:    #9A9A95;
  --text-3:    #5F5F5C;
  --border:    rgba(255, 255, 255, 0.08);
  --border-2:  rgba(255, 255, 255, 0.14);
  --brand:     #C8F25C;
  --brand-hov: #B8E548;
  --accent:    #4A90E2;
  --pos:       #19C37D;
  --pos-sub:   rgba(25, 195, 125, 0.12);
  --neg:       #EF4444;
  --neg-sub:   rgba(239, 68, 68, 0.12);
}

/* ─── Light theme (marketing / admin) ─── */
:root,
:root[data-theme="light"] {
  --surface-canvas:   #F8F7F4;
  --surface-elevated: #FFFFFF;
  --text-primary:     #0A0B0E;
  --text-secondary:   #5F5F5C;
  --border:           rgba(0, 0, 0, 0.08);
  --brand-primary:    #C8F25C;
  --positive:         #19C37D;
  --negative:         #EF4444;
}

/* ─── Typography ─── */
:root {
  --font-sans: 'Inter', system-ui, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
  --font-display: 'Inter Tight', 'Inter', sans-serif;

  /* Always use tabular figures for numbers in trading UI */
  --font-feature-tnum: 'tnum';
}
```

### Scoped styles (component-level)

```vue
<style scoped>
/* These styles only apply to THIS component — no leaking */
.sidebar-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  color: var(--text-2);
  transition: color 0.15s;
}

.sidebar-item.active {
  color: var(--brand);
  border-left: 3px solid var(--brand);
}
</style>
```

### CSS modules (class name hashing)

```vue
<template>
  <!-- $style.price gets a hashed class name like "price_xk3m1" -->
  <span :class="$style.price">{{ price }}</span>
</template>

<style module>
.price {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text);
}
</style>
```

### Dynamic styles with v-bind

```vue
<script setup lang="ts">
// Reactive value that drives CSS — updates when the ref changes
const priceColor = computed(() =>
  priceChange.value > 0 ? 'var(--pos)' : 'var(--neg)'
)
</script>

<style scoped>
.price-display {
  /* v-bind() in CSS reads the JavaScript variable reactively */
  color: v-bind(priceColor);
}
</style>
```

### SCSS support

```bash
npm install -D sass
```

```ts
// nuxt.config.ts — inject shared variables into every SCSS file
export default defineNuxtConfig({
  vite: {
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: '@use "~/assets/scss/_variables.scss" as *;',
        },
      },
    },
  },
})
```

### Tailwind CSS (via module)

```bash
npm install -D @nuxtjs/tailwindcss
```

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  modules: ['@nuxtjs/tailwindcss'],
})
```

---

## 10. TypeScript

### Setup

```bash
npm install --save-dev vue-tsc typescript
```

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  typescript: {
    typeCheck: true,   // Run tsc on build and dev
    strict: true,      // Enable all strict checks (recommended)
  },
})
```

### tsconfig.json (at project root)

```json
{
  "extends": "./.nuxt/tsconfig.app.json"
}
```

> Nuxt generates `.nuxt/tsconfig.app.json` automatically when you run dev.
> Never edit `.nuxt/` files directly — they are regenerated.

### Typed page metadata

```ts
// Augment the PageMeta interface to add custom properties
// Place in: app/types/page-meta.d.ts
declare module '#app' {
  interface PageMeta {
    requiresAuth?: boolean
    pageTitle?: string
  }
}
export {}
```

```vue
<script setup lang="ts">
definePageMeta({
  requiresAuth: true,   // TypeScript now validates this
  pageTitle: 'Wallet',
})
</script>
```

### Typed composables

```ts
// app/composables/useMarket.ts
interface Market {
  symbol: string
  price: number
  change24h: number
  volume24h: number
}

export const useMarket = (slug: string) => {
  const market = useState<Market | null>(`market-${slug}`, () => null)
  return { market: readonly(market) }
}
```

---

## 11. Rendering Modes

### Static site generation (SSG) — best for Netlify

Generates fully static HTML at build time. Best for presentation/demo apps.

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  // Generate a static site — all pages pre-rendered to HTML
  // Deploy the .output/public/ folder to Netlify
  ssr: true,  // Pages are rendered at build time
})

// Run: npm run generate
// Output: .output/public/
```

### Client-side rendering (SPA)

No server needed — pure Vue app. Good for app-like screens behind auth.

```ts
export default defineNuxtConfig({
  ssr: false,  // Renders in browser only, no server HTML
})
```

### Hybrid rendering (per-route rules) — recommended for this project

```ts
export default defineNuxtConfig({
  routeRules: {
    '/':           { prerender: true },   // Homepage: static HTML
    '/signup':     { prerender: true },   // Signup: static
    '/trade':      { ssr: false },        // Trading dashboard: SPA (needs live data)
    '/markets/**': { ssr: false },        // Market pages: SPA
    '/wallet':     { ssr: false },        // Wallet: SPA
    '/portfolio':  { ssr: false },        // Portfolio: SPA
    '/admin/**':   { ssr: false },        // Admin: client-only
  },
})
```

---

## 12. nuxt.config.ts — Key Options

```ts
// nuxt.config.ts
export default defineNuxtConfig({

  // ─── Compatibility ───────────────────────────────────────────
  // Enable Nuxt 4 compatibility mode
  future: {
    compatibilityVersion: 4,
  },

  // ─── TypeScript ──────────────────────────────────────────────
  typescript: {
    typeCheck: true,
    strict: true,
  },

  // ─── Global CSS ─────────────────────────────────────────────
  // These stylesheets are injected on every page
  css: [
    '~/assets/css/tokens.css',
    '~/assets/css/global.css',
  ],

  // ─── App <head> defaults ─────────────────────────────────────
  app: {
    head: {
      title: '1Trade — The Commodity Market for AI Compute',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Trade AI compute credits.' },
      ],
      link: [
        // Google Fonts
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter+Tight:wght@400;500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500;600&display=swap',
        },
      ],
    },
  },

  // ─── Modules ──────────────────────────────────────────────────
  modules: [
    '@nuxtjs/tailwindcss',  // Tailwind CSS (marketing pages)
  ],

  // ─── Route rules ─────────────────────────────────────────────
  routeRules: {
    '/':       { prerender: true },
    '/signup': { prerender: true },
    '/trade':  { ssr: false },
    '/wallet': { ssr: false },
  },

  // ─── Vite config ─────────────────────────────────────────────
  vite: {
    optimizeDeps: {
      // Pre-bundle heavy chart libraries for faster dev server
      include: ['lightweight-charts', 'chart.js'],
    },
  },

  // ─── Runtime config ──────────────────────────────────────────
  // Values accessible server-side via useRuntimeConfig()
  // public values are also available client-side
  runtimeConfig: {
    apiSecret: process.env.API_SECRET,  // Server-only
    public: {
      apiBase: process.env.API_BASE_URL || 'https://api.1trade.com',
    },
  },

  // ─── Path aliases ─────────────────────────────────────────────
  // ~ and @ are built-in aliases for the project root
  // Add custom aliases here if needed
  alias: {
    '@components': '~/app/components',
  },

  // ─── Dev tools ────────────────────────────────────────────────
  devtools: { enabled: true },

})
```

---

## 13. Navigation & Routing

### NuxtLink — declarative navigation

```vue
<template>
  <!--
    NuxtLink is the Nuxt version of Vue Router's <RouterLink>.
    - Generates an <a> tag with correct href for SEO
    - Handles client-side navigation (no full page reload)
    - Automatically prefetches linked pages when they enter the viewport
  -->
  <NuxtLink to="/trade">Go to Trading Dashboard</NuxtLink>
  <NuxtLink to="/markets/eai-idx">AI Index Market</NuxtLink>

  <!-- External links — use a plain <a> tag instead -->
  <a href="https://docs.1trade.com" target="_blank" rel="noopener">Docs</a>

  <!-- Dynamic routes -->
  <NuxtLink :to="`/markets/${market.slug}`">{{ market.name }}</NuxtLink>

  <!-- With active class styling -->
  <NuxtLink
    to="/wallet"
    active-class="active"       <!-- Applied when route matches exactly -->
    exact-active-class="exact"  <!-- Applied on exact match only -->
  >
    Wallet
  </NuxtLink>
</template>
```

### navigateTo() — programmatic navigation

```ts
// Always await or return navigateTo — otherwise the redirect may not complete

// Navigate to a route
await navigateTo('/trade')

// Navigate with query params
await navigateTo({ path: '/markets', query: { filter: 'active' } })

// External redirect (must set external: true for full URLs)
await navigateTo('https://docs.1trade.com', { external: true })

// Replace current history entry (no back button)
await navigateTo('/login', { replace: true })
```

### useRoute() — reading current route

```ts
const route = useRoute()

// Current path: '/markets/eai-idx'
console.log(route.path)

// Dynamic params: { slug: 'eai-idx' }
console.log(route.params.slug)

// Query string: ?tab=orderbook → { tab: 'orderbook' }
console.log(route.query.tab)

// Metadata set by definePageMeta
console.log(route.meta.title)
```

### useRouter() — imperative router control

```ts
const router = useRouter()

// Go back
router.back()

// Push to route
router.push('/trade')

// Replace without history entry
router.replace('/login')
```

### Route middleware (auth guards)

```ts
// app/middleware/auth.ts
// Runs before every page that declares middleware: 'auth'
export default defineNuxtRouteMiddleware((to, from) => {
  const { isLoggedIn } = useAuth() // Your auth composable

  if (!isLoggedIn.value) {
    // Redirect to login, preserving the intended destination
    return navigateTo({
      path: '/login',
      query: { redirect: to.fullPath },
    })
  }
})
```

```vue
<!-- app/pages/trade.vue -->
<script setup lang="ts">
definePageMeta({
  layout: 'app',
  middleware: 'auth',  // Runs auth.ts before this page loads
})
</script>
```

### Global middleware (runs on every route)

```ts
// app/middleware/analytics.global.ts
// The .global suffix means it runs on EVERY route change automatically
export default defineNuxtRouteMiddleware((to) => {
  // Track page views
  console.log('Navigating to:', to.path)
})
```

---

## 14. Assets vs Public

| Directory | Use for | Processed by build? | Referenced as |
|-----------|---------|---------------------|---------------|
| `app/assets/` | CSS, SCSS, fonts, images that need optimization | Yes (Vite) | `~/assets/...` in CSS/JS |
| `public/` | Static files served as-is (favicons, OG images, standalone HTML) | No | `/filename.ext` (absolute URL) |

```vue
<!-- Referencing an asset (processed by Vite — gets hashed filename) -->
<img src="~/assets/images/logo.svg" alt="1Trade" />

<!-- Referencing a public file (served directly — URL stays the same) -->
<img src="/og-image.png" alt="OG Image" />
```

```css
/* In CSS — assets are referenced with url() */
@font-face {
  font-family: 'Inter';
  /* public/ files use absolute path */
  src: url('/fonts/Inter.woff2') format('woff2');
}

/* OR using assets/ (processed) */
background-image: url('~/assets/images/bg.png');
```

---

## 15. Best Practices Checklist

### Architecture

- [ ] One layout per visual mode (`marketing`, `app`) — don't repeat nav/sidebar in pages
- [ ] `definePageMeta` sets layout, middleware, and title — never set these in `onMounted`
- [ ] Shared state lives in composables (not component data) — use `useState` for SSR safety
- [ ] Server-only code goes in `server/` — never import Node.js modules in `app/`

### Performance

- [ ] Prefix rarely-used components with `Lazy` — they're code-split automatically
- [ ] Use `.client.vue` suffix for components that need `window`/`document`
- [ ] Pre-bundle heavy libraries (chart.js, lightweight-charts) in `vite.optimizeDeps`
- [ ] Use `npm run generate` (SSG) for static pages — fastest Netlify deploys

### TypeScript

- [ ] `typeCheck: true` in nuxt.config — catch errors at build time
- [ ] Type all `useState` calls with generics: `useState<Market | null>('key', () => null)`
- [ ] Augment `PageMeta` interface for custom `definePageMeta` properties
- [ ] Run `npx nuxt typecheck` before every deploy

### Styling

- [ ] CSS custom properties in `tokens.css` — single source of truth for colors
- [ ] Use `scoped` styles on all components — prevents style leaking
- [ ] `font-variant-numeric: tabular-nums` on all price/number displays
- [ ] Never use inline `style=""` for design values — use CSS vars

### Routing

- [ ] Use `<NuxtLink>` (not `<a>`) for all internal navigation
- [ ] Always `await navigateTo()` — otherwise redirects may silently fail
- [ ] Auth guard in `app/middleware/auth.ts` — applied per-page via `definePageMeta`
- [ ] Validate dynamic params with `definePageMeta({ validate })` to avoid broken states

### onMounted usage (for chart libraries)

```vue
<script setup lang="ts">
// Chart.js and lightweight-charts require a DOM canvas element.
// They MUST be initialized in onMounted() — the DOM doesn't exist on the server.
const chartRef = ref<HTMLDivElement | null>(null)

onMounted(() => {
  if (!chartRef.value) return

  // Safe to access window/document here — we're in the browser
  const chart = createChart(chartRef.value, {
    width: chartRef.value.clientWidth,
    height: 400,
  })

  // Clean up on component unmount to prevent memory leaks
  onUnmounted(() => {
    chart.remove()
  })
})
</script>

<template>
  <!-- ref="chartRef" gives us a reference to this DOM element -->
  <div ref="chartRef" class="chart-container" />
</template>
```

---

## Quick Reference Card

```
# Create project
npm create nuxt@latest my-app

# Run dev
npm run dev

# Type check
npx nuxt typecheck

# Build (SSR)
npm run build

# Generate (SSG → Netlify)
npm run generate
# Output: .output/public/

# Key composables
useRoute()          → current route (params, query, meta)
useRouter()         → programmatic navigation
useState()          → SSR-safe shared state
useFetch()          → HTTP requests with caching
useRuntimeConfig()  → env vars / config
navigateTo()        → programmatic redirect

# Key macros (compiler-time, not runtime)
definePageMeta()    → layout, middleware, title, transitions
defineNuxtConfig()  → nuxt.config.ts configuration
```
