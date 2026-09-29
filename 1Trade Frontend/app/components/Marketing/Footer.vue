<script setup lang="ts">
/**
 * MarketingFooter — dark inverse site footer, used across all marketing pages. Every link goes to a
 * real page; the AI Index figure is the live EAI-IDX paper market (refreshed every 30 s).
 */
const footerIdx = ref('—')
const footerChange = ref(0)
let tickInterval: ReturnType<typeof setInterval> | null = null

const cols: { title: string; links: { label: string; to: string }[] }[] = [
  { title: 'Product', links: [
    { label: 'Trading', to: '/markets' }, { label: 'Inference', to: '/inference' }, { label: 'Compute', to: '/compute' },
    { label: 'Index', to: '/benchmark' }, { label: 'Status', to: '/status' },
  ] },
  { title: 'Markets', links: [
    { label: 'AI Index', to: '/markets/eai-idx' }, { label: 'Text credits', to: '/markets/text-spot' },
    { label: 'Image credits', to: '/markets/image-spot' }, { label: 'Video credits', to: '/markets/video-spot' },
    { label: 'GPU credits', to: '/markets/h100-spot' },
  ] },
  { title: 'Compute', links: [
    { label: 'GPU instances', to: '/compute/new' }, { label: 'Reserved capacity', to: '/compute/reserve' },
    { label: 'Clusters', to: '/compute/clusters' }, { label: 'GPU catalog', to: '/compute/catalog' },
    { label: 'Supply your GPUs', to: '/signup' },
  ] },
  { title: 'Company', links: [
    { label: 'Index methodology', to: '/benchmark' }, { label: 'Open an account', to: '/signup' },
    { label: 'Contact', to: 'mailto:contact@1trade.com' },
  ] },
  { title: 'Legal', links: [
    { label: 'Terms', to: '/legal#terms' }, { label: 'Privacy', to: '/legal#privacy' },
    { label: 'Trading rules', to: '/legal#trading-rules' }, { label: 'Risk disclosure', to: '/legal#risk' },
    { label: 'Compliance', to: '/legal#compliance' },
  ] },
]

/** loadIndex reads the EAI-IDX paper market: credits per dollar and the 24h change. */
async function loadIndex() {
  try {
    const r = await $fetch<{ summary: { last: string; change_pct_24h: string } }>('/api/trading/products/EAI-IDX')
    const last = Number(r.summary.last)
    if (last > 0) footerIdx.value = Math.round(1 / last).toLocaleString('en-US')
    footerChange.value = Number(r.summary.change_pct_24h)
  } catch { /* keep the last value */ }
}

onMounted(() => {
  void loadIndex()
  tickInterval = setInterval(() => { void loadIndex() }, 30000)
})

onUnmounted(() => {
  if (tickInterval) clearInterval(tickInterval)
})
</script>

<template>
  <!--
    The footer is always a dark band. It used to borrow the LIGHT theme's --inverse
    (= dark) plus hardcoded light greys, which broke the moment a page above it was
    itself dark: --inverse flipped to cream while the greys stayed light. It now
    declares data-theme="dark" and reads ordinary tokens, so it is self-contained and
    correct under a light OR a dark page.
  -->
  <footer class="site" data-theme="dark">
    <div class="container-x">
      <div class="row top">
        <div class="brand-col">
          <div class="brand">
            <BrandLogo variant="full" size="lg" />
          </div>
          <p class="tagline">
            The global exchange for AI compute. Tradeable credits backed by owned datacenter capacity.
          </p>
        </div>

        <div v-for="col in cols" :key="col.title" class="col">
          <h6>{{ col.title }}</h6>
          <ul>
            <li v-for="link in col.links" :key="link.label">
              <a v-if="link.to.startsWith('mailto:')" :href="link.to">{{ link.label }}</a>
              <NuxtLink v-else :to="link.to">{{ link.label }}</NuxtLink>
            </li>
          </ul>
        </div>
      </div>

      <div class="row bar">
        <div class="bar-copy">
          © 2026 1Trade, Inc. · contact@1trade.com · San Francisco · Tokyo
        </div>
        <div class="bar-status">
          <NuxtLink to="/status" class="bar-status-link"><span class="status-dot" />System status</NuxtLink>
          <span class="bar-sep" />
          <span class="tnum">
            AI Index (paper market) · $1 = <span class="bar-num">{{ footerIdx }}</span> credits ·
            <span :class="footerChange >= 0 ? 'pos' : 'neg'">{{ footerChange >= 0 ? '▲' : '▼' }} {{ Math.abs(footerChange).toFixed(2) }}%</span>
          </span>
        </div>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.bar-status-link { color: inherit; text-decoration: none; display: inline-flex; align-items: center; gap: 6px; }
.bar-status-link:hover { color: var(--text); }
.site {
  background: var(--canvas);
  color: var(--text);
  padding: 80px 0 32px;
  border-top: 1px solid var(--border);
}

.container-x {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 var(--sp-6);
}

.row.top {
  display: grid;
  grid-template-columns: 2fr repeat(5, 1fr);
  gap: var(--sp-7);
  padding-bottom: 56px;
  border-bottom: 1px solid var(--border);
}

.brand {
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 20px;
  letter-spacing: -0.02em;
  color: var(--text);
  display: inline-flex;
  align-items: center;
}

/* mark + wordmark come from <BrandLogo> */

.tagline {
  margin-top: 20px;
  color: var(--text-2);
  font-size: 13px;
  line-height: 1.6;
  max-width: 28ch;
}

h6 {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--text-3);
  margin: 0 0 var(--sp-4);
  font-weight: 500;
}

ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

li { margin-bottom: 10px; }

li a {
  color: var(--text-2);
  font-size: var(--fs-base);
  text-decoration: none;
  transition: color var(--dur) var(--ease);
}

li a:hover { color: var(--text); }

.row.bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 28px;
  flex-wrap: wrap;
  gap: var(--sp-4);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}

.bar-copy { color: var(--text-3); }

.bar-status {
  display: flex;
  gap: var(--sp-5);
  align-items: center;
  color: var(--text-2);
}

.bar-num { color: var(--text); }

.bar-sep {
  height: 12px;
  width: 1px;
  background: var(--border);
}

.tnum { font-variant-numeric: tabular-nums; }
.pos  { color: var(--pos); }
.neg  { color: var(--neg); }

/* Pulse dot */
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--pos);
  margin-right: 8px;
  position: relative;
  vertical-align: 1px;
}

.status-dot::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 1px solid var(--pos);
  animation: ping 2.4s ease-out infinite;
}

@keyframes ping {
  0%   { transform: scale(0.9); opacity: 0.6; }
  100% { transform: scale(1.8); opacity: 0; }
}

@media (max-width: 1024px) {
  .row.top { grid-template-columns: 2fr repeat(2, 1fr); }
}

@media (max-width: 600px) {
  .row.top { grid-template-columns: 1fr; gap: var(--sp-6); }
  .row.bar { flex-direction: column; align-items: flex-start; }
}
</style>
