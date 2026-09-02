/**
 * 1TRADE — Nuxt 4 config
 *
 * Two-theme app (light marketing + dark trading) on ONE design-token foundation.
 * Hybrid rendering: marketing pages prerender (SEO), app pages SPA (live data).
 *
 * Tokens live in app/assets/css/tokens.css.
 * Any design change should be a ONE-LINE edit in that file — never here, never in a page.
 */
export default defineNuxtConfig({
  // Nuxt 4 mode
  future: {
    compatibilityVersion: 4,
  },

  // TypeScript
  typescript: {
    typeCheck: false, // turn on once components stabilize
    strict: true,
  },

  // Global stylesheets injected on every page
  css: [
    '~/assets/css/tokens.css',
    '~/assets/css/global.css',
  ],

  // Page <head> defaults — fonts only (everything else per-page via useHead)
  app: {
    head: {
      title: '1TRADE — The Global Exchange for AI Compute',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Tradeable credits connecting GPU datacenters, traders, and AI companies.' },
        // Brand chrome: browsers tint their UI with this on mobile.
        { name: 'theme-color', content: '#0A0A0A' },
      ],
      link: [
        // The mark on a midnight squircle — carries its own ground so it reads
        // against both light and dark browser chrome.
        { rel: 'icon', type: 'image/svg+xml', href: '/brand/favicon.svg' },
        // Self-host in production; CDN is fine for the dev/demo build.
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter+Tight:wght@400;500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500;600&display=swap',
        },
      ],
    },
  },

  // Hybrid rendering per §5 of the migration plan
  routeRules: {
    '/':              { prerender: true },
    '/signup':        { prerender: true },
    '/login':         { prerender: true },
    '/benchmark':     { prerender: true },
    '/trade':         { ssr: false },
    '/markets/**':    { ssr: false },
    '/wallet':        { ssr: false },
    '/wallet/**':     { ssr: false },
    '/portfolio':     { ssr: false },
    '/history':       { ssr: false },
    '/onboarding/**': { ssr: false },
    '/datacenter':    { ssr: false },
    '/datacenter/**': { ssr: false },
    '/settings':      { ssr: false },
    '/settings/**':   { ssr: false },
    '/enterprise/**': { ssr: false },
    '/compute':       { ssr: false },
    '/compute/**':    { ssr: false },
    '/inference':     { ssr: false },
    '/console':       { ssr: false },
    '/states':        { ssr: false },
  },

  // Live-only (F20). There is no mock mode — the Nitro BFF (server/api/**) always proxies to the real
  // platform services and screens render real data, so `npm run dev` needs the platform stack running.
  // Upstream URLs are server-only (private).
  runtimeConfig: {
    platformCoreUrl: process.env.PLATFORM_CORE_URL || 'http://localhost:8001',
    gatewayUrl: process.env.INFERENCE_GATEWAY_URL || 'http://localhost:8085',
    ledgerUrl: process.env.CREDIT_LEDGER_URL || 'http://localhost:8002',
    computeUrl: process.env.COMPUTE_CONTROL_URL || 'http://localhost:8086',
  },

  // No Tailwind for v1 — keeps tokens.css the sole styling language.
  // Add @nuxtjs/tailwindcss later only if the audited HTML proves it's needed.
  modules: [],

  // Vite tuning — pre-bundle the heavy chart libs so dev server boots fast
  vite: {
    optimizeDeps: {
      include: ['lightweight-charts', '@vueuse/core', 'lucide-vue-next', 'date-fns'],
    },
  },

  devtools: { enabled: true },
})
