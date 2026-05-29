# Deployment Guide — Exascale Nuxt 4 App

The Exascale trading platform is ready to deploy. Here's how to get it live on Netlify.

## ✓ Build Status

- ✅ Nuxt 4 project scaffolded
- ✅ Pages created (homepage, signup, trading dashboard, wallet, portfolio, markets)
- ✅ Layouts created (marketing light theme, app dark theme)
- ✅ Components created (AppSidebar, AppTopbar, MarketingNav)
- ✅ Design tokens & global styles
- ✅ Build succeeds: `npm run build` ✓
- ✅ Ready for deployment

## Deploy to Netlify

### Option 1: GitHub Integration (Recommended)

1. **Push to GitHub**
   ```bash
   cd /mnt/f/ex/nuxt-app
   git init
   git add .
   git commit -m "initial: Nuxt 4 app with trading dashboard"
   git remote add origin https://github.com/YOUR_USERNAME/exascale-nuxt.git
   git push -u origin main
   ```

2. **Connect to Netlify**
   - Go to https://app.netlify.com
   - Click "Add new site" → "Import an existing project"
   - Select GitHub, authorize, pick your repo
   - Build settings should auto-detect:
     - **Build command**: `npm run build`
     - **Publish directory**: `.output/public`
   - Click "Deploy site"

### Option 2: Netlify CLI

```bash
npm install -g netlify-cli

# Deploy the built output
netlify deploy --prod --dir=.output/public
```

### Option 3: Drag & Drop

For a quick demo (serves .output/public as static):
```bash
npm run build
# Drag .output/public folder to https://app.netlify.com/drop
```

---

## How the App Works

**Architecture:**
- **Mode**: Single-Page App (SPA) — client-side routing
- **Framework**: Nuxt 4 with Vue 3
- **Styling**: CSS custom properties (design tokens) + scoped component styles
- **Icons**: Emoji icons (can upgrade to Lucide later)

**Routes:**
- `/` — Homepage (light theme, marketing)
- `/signup` — Account creation (light theme)
- `/login` — Sign in (points to `/public/login/`)
- `/trade` — Trading dashboard (dark theme, app shell)
- `/wallet` — Asset balances (dark theme)
- `/portfolio` — Account overview (dark theme)
- `/markets/eai-idx` — Market detail (dynamic route, dark theme)

**Layouts:**
- `marketing.vue` — Light theme for public pages
- `app.vue` — Dark theme for authenticated pages (sidebar + topbar)

**Components:**
- `App/Sidebar.vue` — Icon nav with active state
- `App/Topbar.vue` — Account pill, notifications, user menu
- `MarketingNav.vue` — Sticky top nav for marketing pages

---

## What's Next

### Immediate (finish MVP)

1. **Connect real data sources**
   - `/api/markets` → fetch live market data
   - `/api/trades` → fetch trading history
   - Build `useMarket()`, `useTrades()` composables

2. **Implement authentication**
   - Wire up `/login` and `/signup` to backend
   - Add auth middleware
   - Protect `/trade`, `/wallet`, `/portfolio` routes

3. **Upgrade complex pages**
   - Implement `lightweight-charts` in `/trade` for candlestick chart
   - Implement `chart.js` for portfolio performance chart
   - Add real order book rendering
   - Add live position updates

4. **Deploy to Netlify**
   - Push to GitHub
   - Connect to Netlify (auto-deploys on git push)

### Later (polish)

- [ ] Upgrade emoji icons to Lucide Vue icons
- [ ] Add form validation & error handling
- [ ] Implement order submission flow
- [ ] Add WebSocket for live price updates
- [ ] Add Vitest unit tests
- [ ] Setup CI/CD pipeline
- [ ] Monitor performance & errors (Sentry integration)

---

## Local Development

```bash
# Install dependencies
npm install

# Dev server (http://localhost:3000)
npm run dev

# Type check
npm run typecheck

# Build for production
npm run build

# Preview the build
npm run preview

# Generate static version (if you want SSG later)
npm run generate
```

---

## Environment Variables

Create a `.env.local` file for local development:

```
NUXT_PUBLIC_API_BASE_URL=http://localhost:3000/api
NUXT_API_SECRET=your_secret_key_here
```

Access in components:
```ts
const config = useRuntimeConfig()
console.log(config.public.apiBaseUrl)  // Available client-side
console.log(config.apiSecret)           // Server-side only
```

---

## Troubleshooting

**Build fails with "instance unavailable"**
- Usually means a composable (like `useRoute()`) is being called during SSR pre-rendering
- Solution: Set `ssr: false` in `nuxt.config.ts` or disable prerendering for that route
- Currently using SPA mode (`ssr: false`), so all rendering happens client-side ✓

**Icons don't show**
- Currently using emoji icons (📊, 💼, etc.)
- To upgrade to Lucide icons: install `lucide-vue-next` and import properly
- See `app/components/App/Sidebar.vue` for the emoji pattern

**Can't import components**
- All components in `app/components/` are auto-imported by Nuxt
- No need for manual `import` statements
- If auto-import doesn't work, check `nuxt.config.ts` → `components` setting

**TypeScript errors in IDE**
- Run `npm run dev` once to generate `.nuxt/tsconfig.app.json`
- Reload your IDE (VS Code may need "Reload Window" via Command Palette)

---

## Performance Tips

- Components with `Lazy` prefix are code-split automatically (lazy-load on demand)
- `.client.vue` components only load in the browser
- `.server.vue` components only run on the server (SSR)
- Static assets in `/public` are served without processing
- CSS in `app/assets/css/` is bundled and minified

---

## Support

- **Nuxt docs**: https://nuxt.com/docs/4.x/guide
- **Vue docs**: https://vuejs.org
- **Netlify docs**: https://docs.netlify.com

---

**Status**: Ready for Netlify deployment ✅
