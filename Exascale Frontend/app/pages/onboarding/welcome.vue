<script setup lang="ts">
/**
 * /onboarding/welcome — First-login welcome (D2-B)
 *
 * Light, layout-less. Lands after KYC is complete + email verified.
 * Single primary CTA → /trade. Secondary: 60-second product tour.
 *
 * Brief warmth, no consumer-cheer. Trust-building via real numbers.
 */
definePageMeta({ layout: false })
useHead({ title: 'Welcome to Exascale — Exascale', htmlAttrs: { 'data-theme': 'light' } })

const route = useRoute()
const firstName = ref<string>(typeof route.query.name === 'string' ? route.query.name : 'Jane')

// Live AI Index (mean reversion to 1.0024)
const indexValue = ref<number>(1.0024)
const indexPct   = ref<number>(0.18)
let tickInterval: ReturnType<typeof setInterval> | null = null

interface MarketRow { sym: string; px: number; deltaPct: number; precision: number }
const markets = ref<MarketRow[]>([
  { sym: 'EAI-IDX',   px: 0.001005, deltaPct: 0.18, precision: 6 },
  { sym: 'TEXT-SPOT', px: 0.001210, deltaPct: 1.84, precision: 6 },
  { sym: 'H100-SPOT', px: 2.99,     deltaPct: 0.18, precision: 2 },
])

function tickIndex() {
  let v = indexValue.value
  v = v + (1.0024 - v) * 0.04 + (Math.random() - 0.5) * 0.0006
  v = Math.max(0.992, Math.min(1.015, v))
  indexValue.value = v
  indexPct.value = (v - 1.0) * 100

  // Markets drift around the same regime
  const ratio = v / 1.0024
  markets.value = markets.value.map((m, i) => {
    const base = i === 0 ? 0.001005 : i === 1 ? 0.001210 : 2.99
    const jitter = i === 2 ? (Math.random() - 0.5) * 0.004 : (Math.random() - 0.5) * 0.0000008
    return { ...m, px: base * ratio + jitter }
  })
}

onMounted(() => { tickInterval = setInterval(tickIndex, 2400) })
onBeforeUnmount(() => { if (tickInterval) clearInterval(tickInterval) })

const indexDisplay = computed(() => (indexValue.value * 1000).toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 }))
const indexDeltaStr = computed(() => {
  const arrow = indexPct.value >= 0 ? '▲' : '▼'
  return arrow + ' ' + Math.abs(indexPct.value).toFixed(2) + '%'
})
const indexDeltaPos = computed(() => indexPct.value >= 0)

function fmtPx(px: number, precision: number): string {
  return '$' + px.toFixed(precision)
}
function fmtDelta(pct: number): string {
  const arrow = pct >= 0 ? '▲' : '▼'
  return arrow + ' ' + Math.abs(pct).toFixed(2) + '%'
}
</script>

<template>
  <div class="welcome" data-theme="light">
    <header class="chrome">
      <NuxtLink to="/" class="brand"><span class="mark" />Exascale</NuxtLink>
      <NuxtLink to="/login" class="chrome-link">Sign out</NuxtLink>
    </header>

    <div class="center">
      <div class="hero">
        <h1>Welcome to Exascale, {{ firstName }}.</h1>
        <p class="sub">Your paper trading account is ready.</p>

        <div class="cards">
          <div class="card c-1">
            <div class="cap">Paper balance</div>
            <div class="big mono">$10,000.00</div>
            <div class="sub-line">USD ready · risk-free practice</div>
          </div>

          <div class="card c-2">
            <div class="cap">AI Index live</div>
            <div class="big mono">$1 = {{ indexDisplay }}</div>
            <div class="sub-line">
              credits per dollar ·
              <span class="mono" :class="{ pos: indexDeltaPos, neg: !indexDeltaPos }">{{ indexDeltaStr }}</span>
            </div>
          </div>

          <div class="card c-3">
            <div class="cap">Markets open</div>
            <ul class="markets-list">
              <li v-for="m in markets" :key="m.sym" class="market-row">
                <span class="sym mono">{{ m.sym }}</span>
                <span class="px mono">{{ fmtPx(m.px, m.precision) }}</span>
                <span class="delta mono pos">{{ fmtDelta(m.deltaPct) }}</span>
              </li>
            </ul>
          </div>
        </div>

        <div class="cta-row">
          <NuxtLink to="/trade" class="btn-primary">
            Start trading
            <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
          <NuxtLink to="/onboarding/tour" class="tour-link">Or take the guided product tour</NuxtLink>
        </div>
      </div>
    </div>

    <footer class="page-foot">
      <span class="mono muted">
        Account EX-PT-7A3C91 · paper-trading mode · capital trading requires further verification
      </span>
      <a href="#" class="foot-link">What's paper trading? →</a>
    </footer>
  </div>
</template>

<style scoped>
.welcome {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.55;
  display: flex;
  flex-direction: column;
}
.welcome .mono   { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.welcome .muted  { color: var(--text-3); }
.welcome .pos    { color: var(--pos); font-weight: 600; }
.welcome .neg    { color: var(--neg); font-weight: 600; }

/* Chrome */
.chrome {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28px 48px;
  flex-shrink: 0;
}
.brand {
  display: inline-flex; align-items: center; gap: 10px;
  font-family: var(--font-display); font-weight: 700; font-size: 18px;
  letter-spacing: -0.02em; color: var(--text); text-decoration: none;
}
.brand .mark { display: inline-block; width: 12px; height: 12px; background: var(--brand); }
.chrome-link {
  color: var(--text-2); text-decoration: none; font-size: 13px;
  transition: color 160ms ease;
}
.chrome-link:hover { color: var(--text); }

/* Hero */
.center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 32px;
}
.hero {
  width: 100%;
  max-width: 1100px;
  text-align: center;
}
.hero h1 {
  font-family: var(--font-display);
  font-size: 52px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.08;
  margin: 0 0 12px;
  color: var(--text);
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 80ms forwards;
}
.hero .sub {
  font-size: 17px;
  color: var(--text-2);
  margin: 0 0 48px;
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 200ms forwards;
}
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* Cards */
.cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin: 0 0 40px;
  text-align: left;
}
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: 24px;
  min-height: 168px;
  display: flex;
  flex-direction: column;
  opacity: 0;
  transform: translateY(8px);
  animation: cardIn 600ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
  transition: border-color 160ms ease;
}
.card.c-1 { animation-delay: 340ms; }
.card.c-2 { animation-delay: 420ms; }
.card.c-3 { animation-delay: 500ms; }
.card:hover { border-color: var(--border-strong); }
@keyframes cardIn { to { opacity: 1; transform: translateY(0); } }

.cap {
  font-family: var(--font-mono);
  font-size: 10px; font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 18px;
}
.big {
  font-size: 28px; font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text);
  line-height: 1.1;
  margin-bottom: 6px;
}
.sub-line {
  font-size: 13px;
  color: var(--text-2);
}

/* Markets list (card 3) */
.markets-list { list-style: none; margin: 0; padding: 0; }
.market-row {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 12px;
  align-items: center;
  padding: 9px 0;
  font-family: var(--font-mono);
}
.market-row + .market-row { border-top: 1px solid var(--border); }
.market-row .sym {
  color: var(--text); font-weight: 600;
  font-size: 11px; letter-spacing: 0.06em;
}
.market-row .px {
  color: var(--text); font-size: 13px; font-weight: 500;
}
.market-row .delta {
  font-size: 11px; font-weight: 600;
}

/* CTA row */
.cta-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 620ms forwards;
}
.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 15px 30px;
  background: var(--brand);
  color: var(--text);
  border: 1px solid var(--brand);
  border-radius: 2px;
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 15px;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease, transform 80ms ease;
}
.btn-primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn-primary:active { transform: translateY(1px); }
.tour-link {
  color: var(--text-2);
  font-size: 13px;
  text-decoration: underline;
  text-underline-offset: 3px;
  text-decoration-thickness: 1px;
  font-family: var(--font-sans);
}
.tour-link:hover { color: var(--text); }

/* Footer */
.page-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 48px 22px;
  font-size: 11.5px;
  border-top: 1px solid var(--border);
  background: var(--elevated);
}
.foot-link { color: var(--accent); text-decoration: none; }
.foot-link:hover { text-decoration: underline; }

@media (max-width: 900px) {
  .cards { grid-template-columns: 1fr; }
  .hero h1 { font-size: 36px; }
  .chrome, .page-foot { padding-left: 24px; padding-right: 24px; }
}
</style>
