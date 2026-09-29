<script setup lang="ts">
/**
 * /markets/[slug] — one market on the paper venue (trading.yaml v1.1), live from the matching engine.
 *
 *   sub-topbar (breadcrumbs)
 *   header (price, 24h change, 24h sparkline, Trade button)
 *   stats (24h volume / high / low / spread; last trade, tick, status)
 *   candlestick chart — bars from paper trades; bars with no trade show the reference walk, muted
 *   details (about + specification, from the catalog)
 *   related markets (live)
 *   order book + recent trades (the real paper book and tape)
 *
 * Nothing here is generated in the browser. Real money is paused pending licensing; the page says so.
 */
import { createChart, ColorType, CrosshairMode, type ISeriesApi } from 'lightweight-charts'

definePageMeta({ layout: 'app', middleware: 'auth' })

interface Product {
  product_id: string; name: string; description: string; family: string; credit_type: string
  tick_size: string; quote_precision: number; tradeable: boolean
}
interface Summary {
  last: string; open_24h: string; high_24h: string; low_24h: string; change_pct_24h: string
  volume_24h: string; spread_bps: string
}
interface CandleApi { time: number; open: string; high: string; low: string; close: string; volume: string; source?: string }
interface Market { product_id: string; name: string; family: string; last: number; changePct24h: number; tradeable: boolean }

const route = useRoute()
const slug = computed(() => {
  const s = Array.isArray(route.params.slug) ? route.params.slug[0] : route.params.slug
  return (s || 'eai-idx').toUpperCase()
})

const product = ref<Product | null>(null)
const summary = ref<Summary | null>(null)
const status = ref<{ state: string; reason: string } | null>(null)
const loadErr = ref('')
const midPrice = ref(0)
const flashClass = ref<'flash-up' | 'flash-down' | ''>('')
const lastUpdated = ref('')
const previewCollapsed = ref(false)
useHead({ title: () => `${slug.value} · ${product.value?.name ?? 'Market'} — 1Trade` })

type TF = '1m' | '5m' | '15m' | '1h' | '4h' | '1d'
const TIMEFRAMES: TF[] = ['1m', '5m', '15m', '1h', '4h', '1d']
const tf = ref<TF>('1h')

const dp = computed(() => product.value?.quote_precision ?? 6)
const n = (s?: string) => Number(s ?? 0)
const deltaPct = computed(() => n(summary.value?.change_pct_24h))
const deltaAbs = computed(() => n(summary.value?.last) - n(summary.value?.open_24h))

// =====================================================
// Formatters
// =====================================================
function fmtPx(v: number, places = dp.value) {
  return v.toLocaleString('en-US', { minimumFractionDigits: places, maximumFractionDigits: places })
}
function fmtQty(v: number) {
  return v.toLocaleString('en-US', { maximumFractionDigits: v < 100 ? 4 : 0 })
}
function fmtUsd(v: number, places = 2) {
  return '$' + v.toLocaleString('en-US', { minimumFractionDigits: places, maximumFractionDigits: places })
}
function timestampNow() {
  return new Date().toLocaleTimeString('en-GB', { hour12: false, timeZone: 'UTC' }) + ' UTC'
}

// =====================================================
// Product + summary
// =====================================================
/** loadProduct reads the product and its 24h summary, and flashes the price when it moves. */
async function loadProduct() {
  try {
    const r = await $fetch<{ product: Product; summary: Summary }>(`/api/trading/products/${slug.value}`)
    product.value = r.product
    summary.value = r.summary
    loadErr.value = ''
  } catch {
    loadErr.value = 'This market is unavailable — retrying…'
  }
}

// =====================================================
// Chart (real candles; reference bars muted)
// =====================================================
const chartHost = ref<HTMLDivElement | null>(null)
let chart: ReturnType<typeof createChart> | null = null
let candleSeries: ISeriesApi<'Candlestick'> | null = null
let volumeSeries: ISeriesApi<'Histogram'> | null = null
let chartRO: ResizeObserver | null = null
const tradedBars = ref(0)
const totalBars = ref(0)

/** fetchCandles reads a product's candles for one interval. */
async function fetchCandles(id: string, interval: TF | '15m', limit: number) {
  const r = await $fetch<{ candles: CandleApi[] }>(`/api/trading/products/${id}/candles`, { query: { interval, limit } })
  return r.candles ?? []
}

/** loadChart draws the selected timeframe. */
async function loadChart() {
  if (!candleSeries || !volumeSeries) return
  try {
    const rows = await fetchCandles(slug.value, tf.value, 200)
    const muted = 'rgba(168,161,150,0.35)'
    candleSeries.setData(rows.map((c) => {
      const bar = { time: c.time as never, open: n(c.open), high: n(c.high), low: n(c.low), close: n(c.close) }
      return c.source === 'reference' ? { ...bar, color: muted, wickColor: muted, borderColor: muted } : bar
    }))
    volumeSeries.setData(rows.map((c) => ({
      time: c.time as never, value: n(c.volume),
      color: n(c.close) >= n(c.open) ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)',
    })))
    tradedBars.value = rows.filter(c => c.source !== 'reference').length
    totalBars.value = rows.length
  } catch { /* keep the last series */ }
}

/** applyTimeframe switches the chart interval. */
async function applyTimeframe(t: TF) {
  tf.value = t
  await loadChart()
  chart?.timeScale().fitContent()
}

/** initChart builds the chart for the current product. */
function initChart() {
  if (!chartHost.value) return
  chart = createChart(chartHost.value, {
    layout: { background: { type: ColorType.Solid, color: '#0A0A0A' }, textColor: '#A8A196', fontFamily: 'JetBrains Mono, ui-monospace, monospace', fontSize: 10 },
    grid: { vertLines: { color: 'rgba(255,255,255,0.03)' }, horzLines: { color: 'rgba(255,255,255,0.03)' } },
    rightPriceScale: { borderColor: 'rgba(255,255,255,0.08)', scaleMargins: { top: 0.08, bottom: 0.28 } },
    timeScale: { borderColor: 'rgba(255,255,255,0.08)', timeVisible: true, secondsVisible: false },
    crosshair: {
      mode: CrosshairMode.Normal,
      vertLine: { color: 'rgba(232,230,224,0.35)', width: 1, style: 0 },
      horzLine: { color: 'rgba(232,230,224,0.35)', width: 1, style: 0 },
    },
  })
  candleSeries = chart.addCandlestickSeries({
    upColor: '#19C37D', downColor: '#EF4444', borderUpColor: '#19C37D', borderDownColor: '#EF4444',
    wickUpColor: '#19C37D', wickDownColor: '#EF4444',
    priceFormat: { type: 'price', precision: dp.value, minMove: n(product.value?.tick_size) || 0.000001 },
  })
  volumeSeries = chart.addHistogramSeries({ priceFormat: { type: 'volume' }, priceScaleId: '', color: 'rgba(232,230,224,0.18)' })
  volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.80, bottom: 0 } })
  chartRO = new ResizeObserver(() => {
    if (chart && chartHost.value) chart.applyOptions({ width: chartHost.value.clientWidth, height: chartHost.value.clientHeight })
  })
  chartRO.observe(chartHost.value)
}

// =====================================================
// Sparklines (24h of 15-minute closes)
// =====================================================
/** sparkPaths turns closes into an SVG line and fill of width × height. */
function sparkPaths(closes: number[], W: number, H: number, pad: number) {
  if (closes.length < 2) return { line: '', fill: '' }
  const min = Math.min(...closes), max = Math.max(...closes)
  const range = max - min || 1
  const stepX = W / (closes.length - 1)
  const usableH = H - pad * 2
  const line = closes.map((v, i) => (i === 0 ? 'M' : 'L') + (i * stepX).toFixed(1) + ' ' + (pad + usableH - ((v - min) / range) * usableH).toFixed(1)).join(' ')
  return { line, fill: `${line} L ${W} ${H} L 0 ${H} Z` }
}
const hdrSparkLine = ref('')
const hdrSparkFill = ref('')

/** loadHeaderSpark draws the header's 24h line. */
async function loadHeaderSpark() {
  try {
    const { line, fill } = sparkPaths((await fetchCandles(slug.value, '15m', 96)).map(c => n(c.close)), 320, 60, 6)
    hdrSparkLine.value = line
    hdrSparkFill.value = fill
  } catch { /* leave empty */ }
}

// =====================================================
// Order book + tape (the real paper book)
// =====================================================
interface BookLevel { price: number; qty: number; cum: number }
const asksPreview = ref<BookLevel[]>([])
const bidsPreview = ref<BookLevel[]>([])
const maxBookQty = computed(() => Math.max(1, ...asksPreview.value.map(l => l.qty), ...bidsPreview.value.map(l => l.qty)))
const bestAsk = ref(0)
const bestBid = ref(0)
const spread = computed(() => (bestAsk.value && bestBid.value) ? bestAsk.value - bestBid.value : 0)
const spreadPct = computed(() => midPrice.value ? (spread.value / midPrice.value) * 100 : 0)

/** loadBook reads ten levels of the paper book. */
async function loadBook() {
  try {
    const r = await $fetch<{ asks: { price: string; size: string; cumulative: string }[]; bids: { price: string; size: string; cumulative: string }[] }>(
      `/api/trading/products/${slug.value}/orderbook`, { query: { depth: 10 } })
    const map = (ls: { price: string; size: string; cumulative: string }[]) => (ls ?? []).map(l => ({ price: n(l.price), qty: n(l.size), cum: n(l.cumulative) }))
    const asks = map(r.asks), bids = map(r.bids)
    asksPreview.value = asks.slice().reverse()
    bidsPreview.value = bids
    bestAsk.value = asks[0]?.price ?? 0
    bestBid.value = bids[0]?.price ?? 0
    const prev = midPrice.value
    midPrice.value = bestAsk.value && bestBid.value ? (bestAsk.value + bestBid.value) / 2 : n(summary.value?.last)
    if (prev && midPrice.value !== prev) {
      flashClass.value = midPrice.value > prev ? 'flash-up' : 'flash-down'
      setTimeout(() => { flashClass.value = '' }, 700)
    }
    lastUpdated.value = timestampNow()
  } catch { /* keep the last book */ }
}

interface TapeRow { side: 'up' | 'down'; px: number; qty: number; time: string; large: boolean }
const MAX_TAPE = 18
const tape = ref<TapeRow[]>([])

/** loadTape reads the most recent paper prints. */
async function loadTape() {
  try {
    const r = await $fetch<{ trades: { price: string; quantity: string; aggressor_side: string; executed_at: string; block: boolean }[] }>(
      `/api/trading/products/${slug.value}/trades`, { query: { limit: MAX_TAPE } })
    tape.value = (r.trades ?? []).map(t => ({
      side: t.aggressor_side === 'buy' ? 'up' : 'down', px: n(t.price), qty: n(t.quantity),
      time: new Date(t.executed_at).toLocaleTimeString('en-GB', { hour12: false }), large: !!t.block,
    } as TapeRow))
  } catch { /* keep the last tape */ }
}

// =====================================================
// Related markets (same data as /markets)
// =====================================================
interface RelatedMarket { sym: string; name: string; px: number; dpct: number; color: 'pos' | 'neg'; places: number; sparkPath: string; sparkFill: string }
const related = ref<RelatedMarket[]>([])

/** loadRelated shows four other tradeable markets with their 24h lines. */
async function loadRelated() {
  try {
    const r = await $fetch<{ exchange_status: { state: string; reason: string }; markets: (Market & { quote_precision: number })[] }>('/api/trading/markets')
    status.value = r.exchange_status ?? null
    const others = (r.markets ?? []).filter(m => m.product_id !== slug.value && m.tradeable).slice(0, 4)
    related.value = await Promise.all(others.map(async (m) => {
      const { line, fill } = sparkPaths((await fetchCandles(m.product_id, '15m', 96).catch(() => [])).map(c => n(c.close)), 200, 36, 4)
      return { sym: m.product_id, name: m.name, px: m.last, dpct: m.changePct24h, color: (m.changePct24h >= 0 ? 'pos' : 'neg') as 'pos' | 'neg', places: m.quote_precision, sparkPath: line, sparkFill: fill }
    }))
  } catch { /* keep the last cards */ }
}

// =====================================================
// Lifecycle
// =====================================================
let fastInterval: ReturnType<typeof setInterval> | null = null
let slowInterval: ReturnType<typeof setInterval> | null = null

/** loadAll (re)loads everything for the current market. */
async function loadAll() {
  await loadProduct()
  if (!chart) initChart()
  await Promise.all([loadChart(), loadHeaderSpark(), loadBook(), loadTape(), loadRelated()])
  chart?.timeScale().fitContent()
}

watch(slug, async () => {
  chart?.remove()
  chart = null
  chartRO?.disconnect()
  await loadAll()
})

onMounted(async () => {
  await loadAll()
  fastInterval = setInterval(() => { void loadBook(); void loadTape() }, 3000)
  slowInterval = setInterval(() => { void loadProduct(); void loadChart(); void loadHeaderSpark() }, 15000)
})
onBeforeUnmount(() => {
  if (fastInterval) clearInterval(fastInterval)
  if (slowInterval) clearInterval(slowInterval)
  chartRO?.disconnect()
  chart?.remove()
})
</script>

<template>
  <div class="market-detail">
    <!-- Sub-topbar: breadcrumbs -->
    <div class="subbar">
      <nav class="breadcrumbs">
        <NuxtLink to="/markets">Markets</NuxtLink>
        <span class="sep">›</span>
        <span class="cur">{{ slug }}</span>
      </nav>
    </div>

    <main class="page">
      <p v-if="loadErr" class="stat-sub neg">{{ loadErr }}</p>
      <!-- ============ Market header ============ -->
      <section class="market-header">
        <div class="mh-left">
          <div class="row1">
            <span class="sym-tag">{{ slug }}</span>
            <span class="meta-pill"><span class="pulse-dot" />{{ status?.state === 'paper' ? 'Paper trading · 24/7' : 'Order entry paused' }}</span>
            <span class="meta-pill meta-2">Real money paused</span>
          </div>
          <h1>{{ product?.name ?? slug }}</h1>
          <div class="mh-price" :class="flashClass">{{ fmtPx(midPrice) }}</div>
          <div class="mh-delta-row">
            <span class="mh-delta" :class="{ neg: deltaPct < 0 }">
              {{ deltaPct >= 0 ? '▲' : '▼' }} {{ Math.abs(deltaPct).toFixed(2) }}%
            </span>
            <span class="mh-delta-abs">
              {{ deltaAbs >= 0 ? '+' : '−' }}${{ Math.abs(deltaAbs).toFixed(dp) }}
            </span>
          </div>
          <div class="mh-updated">
            Book as of <span>{{ lastUpdated || '—' }}</span>
            · Index prints <span class="upd-em">16:00 UTC</span>
          </div>
        </div>

        <div class="mh-right">
          <NuxtLink v-if="product?.tradeable && status?.state === 'paper'" :to="'/trade?product=' + slug" class="btn btn-primary">
            Trade on paper
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
          <div class="mh-spark">
            <div class="label-row">
              <span>— 24H</span>
              <span :class="deltaPct >= 0 ? 'pos' : 'neg'">
                {{ deltaPct >= 0 ? '▲' : '▼' }} {{ Math.abs(deltaPct).toFixed(2) }}%
              </span>
            </div>
            <svg viewBox="0 0 320 60" preserveAspectRatio="none">
              <path class="fill" :d="hdrSparkFill" />
              <path class="line" :d="hdrSparkLine" />
            </svg>
          </div>
        </div>
      </section>

      <!-- ============ Stats ============ -->
      <section class="stats-grid">
        <div class="stat">
          <span class="stat-lbl">— 24h volume</span>
          <span class="stat-val">{{ fmtQty(n(summary?.volume_24h)) }}</span>
          <span class="stat-sub">{{ product?.credit_type ?? '' }} credits traded</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— 24h high</span>
          <span class="stat-val">{{ fmtUsd(n(summary?.high_24h), dp) }}</span>
          <span class="stat-sub">paper trades</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— 24h low</span>
          <span class="stat-val">{{ fmtUsd(n(summary?.low_24h), dp) }}</span>
          <span class="stat-sub">paper trades</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— Spread</span>
          <span class="stat-val">{{ summary?.spread_bps ?? '—' }} bps</span>
          <span class="stat-sub">{{ fmtPx(spread) }} at the touch</span>
        </div>
      </section>

      <!-- ============ Primary chart ============ -->
      <section class="chart-section">
        <div class="chart-card">
          <div class="chart-head">
            <div class="seg">
              <button v-for="t in TIMEFRAMES" :key="t" type="button" :class="{ active: tf === t }" @click="applyTimeframe(t)">{{ t }}</button>
            </div>
            <div class="chart-head-right">
              <span class="indicator-pill">{{ tradedBars }} of {{ totalBars }} bars traded · grey = reference, no trades</span>
            </div>
          </div>
          <div class="chart-wrap">
            <div ref="chartHost" class="chart-host" />
          </div>
        </div>
      </section>

      <!-- ============ Details ============ -->
      <section class="details">
        <div>
          <h3>About this market</h3>
          <p>{{ product?.description }} — a spot market in {{ product?.credit_type }} credits, priced in USD.</p>
          <p>
            Trading is on paper: orders reserve paper cash or paper credits in the credit ledger, match by
            price and time, and settle there. Both sides of every book are quoted by a paper liquidity
            account so the market is tradeable at any hour. Real-money trading is paused pending exchange
            licensing.
          </p>
          <p v-if="slug === 'EAI-IDX'">
            The AI Index methodology — constituents, weights, window and the audit hash chain on every
            print — is public. Index values are provisional until the index service is in production.
          </p>
          <NuxtLink to="/benchmark" class="methodology-link">
            View the index methodology
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M5 11L11 5M6 5h5v5" />
            </svg>
          </NuxtLink>
        </div>

        <div>
          <h3>Specifications</h3>
          <table class="spec-table">
            <tbody>
              <tr><th>Product</th><td>{{ slug }} · spot</td></tr>
              <tr><th>Credit</th><td>{{ product?.credit_type }}</td></tr>
              <tr><th>Settlement</th><td>Paper — settled in the credit ledger</td></tr>
              <tr><th>Tick size</th><td>${{ product?.tick_size }}</td></tr>
              <tr><th>Order types</th><td>Market · limit (gtc, day) · IOC · FOK</td></tr>
              <tr><th>Trading hours</th><td>24 / 7 (paper)</td></tr>
              <tr><th>Currency</th><td>USD</td></tr>
              <tr><th>Fees</th><td>0.50% maker / 1.00% taker</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- ============ Related markets ============ -->
      <section class="related">
        <h3>Related markets</h3>
        <div class="related-grid">
          <NuxtLink v-for="m in related" :key="m.sym" :to="'/markets/' + m.sym.toLowerCase()" class="market-card">
            <span class="sym">{{ m.sym }}</span>
            <span class="name">{{ m.name }}</span>
            <div class="pxrow">
              <span class="px">{{ fmtPx(m.px, m.places) }}</span>
              <span class="delta" :class="m.color">{{ m.dpct >= 0 ? '▲' : '▼' }} {{ Math.abs(m.dpct).toFixed(2) }}%</span>
            </div>
            <svg viewBox="0 0 200 36" preserveAspectRatio="none">
              <path class="fill" :class="m.color" :d="m.sparkFill" />
              <path class="line" :class="m.color" :d="m.sparkPath" />
            </svg>
          </NuxtLink>
        </div>
      </section>

      <!-- ============ OB + Tape preview ============ -->
      <section class="preview" :class="{ collapsed: previewCollapsed }">
        <div class="preview-head" @click="previewCollapsed = !previewCollapsed">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="caret">
            <path d="M4 6l4 4 4-4" />
          </svg>
          <h3>Order book &amp; recent trades</h3>
          <span class="preview-sub">Paper · 10 levels</span>
        </div>
        <div class="preview-body">
          <div class="preview-card">
            <div class="preview-card-head">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
                <path d="M3 4l5 3 5-3M3 8l5 3 5-3M3 12l5 3 5-3" />
              </svg>
              <span class="lbl">Order book</span>
              <span class="lbl-badge">PAPER</span>
            </div>
            <div class="ob-prev">
              <div class="ob-cols"><span>Price (USD)</span><span>Size</span><span>Total</span></div>
              <div v-for="(l, i) in asksPreview" :key="'a-' + i" class="ob-row ask">
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtQty(l.qty) }}</span>
                <span class="ob-tot">{{ fmtQty(l.cum) }}</span>
              </div>
              <div class="ob-mid">
                {{ fmtPx(midPrice) }}
                <span class="ob-mid-spread">spread {{ fmtPx(spread) }} / {{ spreadPct.toFixed(2) }}%</span>
              </div>
              <div v-for="(l, i) in bidsPreview" :key="'b-' + i" class="ob-row bid">
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtQty(l.qty) }}</span>
                <span class="ob-tot">{{ fmtQty(l.cum) }}</span>
              </div>
            </div>
          </div>

          <div class="preview-card">
            <div class="preview-card-head">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
                <path d="M2 8h3l2-4 2 8 2-5 2 3h1" />
              </svg>
              <span class="lbl">Recent trades</span>
              <span class="lbl-badge">PAPER</span>
            </div>
            <div class="tape-prev">
              <div class="tape-cols"><span>Time</span><span>Price</span><span>Qty</span><span /></div>
              <p v-if="!tape.length" class="stat-sub">No paper trades in this market yet.</p>
              <div v-for="(t, i) in tape" :key="t.time + i" class="tape-row" :class="[t.side, { large: t.large, enter: i === 0 }]">
                <span class="t-time">{{ t.time }}</span>
                <span class="t-px">{{ fmtPx(t.px) }}</span>
                <span class="t-qty">{{ fmtQty(t.qty) }}</span>
                <span class="t-side">{{ t.side === 'up' ? '▲' : '▼' }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="preview-cta-row">
          <NuxtLink :to="'/trade?product=' + slug" class="preview-cta">
            Open the trading view
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.market-detail {
  --pos-bar: rgba(25, 195, 125, 0.16);
  --neg-bar: rgba(239, 68, 68, 0.16);
  --pos-sub: rgba(25, 195, 125, 0.12);
  --neg-sub: rgba(239, 68, 68, 0.12);

  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.55;
  color: var(--text);
  font-feature-settings: 'tnum' on, 'ss01' on;
  background: var(--canvas);
}
.mono { font-family: var(--font-mono); }
.pos { color: var(--pos); }
.neg { color: var(--neg); }
.dim { color: var(--text-3); }

/* Sub-topbar */
.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: var(--canvas);
}
.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.breadcrumbs a {
  color: var(--text-2);
  text-decoration: none;
}
.breadcrumbs a:hover { color: var(--text); }
.breadcrumbs .sep { color: var(--text-3); opacity: 0.6; }
.breadcrumbs .cur { color: var(--text); }

.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 32px 40px 80px;
}

/* ============================================================
   Market header
   ============================================================ */
.market-header {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 40px;
  padding: 32px 0 48px;
  border-bottom: 1px solid var(--border);
}
.mh-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.mh-left .row1 {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 4px;
}
.sym-tag {
  background: var(--elevated);
  border: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 11px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  color: var(--text-2);
  letter-spacing: 0.06em;
}
.meta-pill {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.meta-pill.meta-2 { color: var(--text-2); }
.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--pos);
  position: relative;
}
.pulse-dot::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 1px solid var(--pos);
  animation: ping 2.4s ease-out infinite;
}
@keyframes ping {
  0%   { transform: scale(0.9); opacity: 0.6; }
  100% { transform: scale(2);   opacity: 0; }
}

.mh-left h1 {
  font-family: var(--font-display);
  font-size: 44px;
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.05;
  margin: 0;
}
.mh-price {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 88px;
  font-weight: 600;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 16px 0 12px;
  transition: color 600ms;
}
.mh-price.flash-up   { color: var(--pos); }
.mh-price.flash-down { color: var(--neg); }

.mh-delta-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 8px;
}
.mh-delta {
  background: var(--pos-sub);
  color: var(--pos);
  font-family: var(--font-mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  padding: 5px 10px;
  border-radius: var(--radius-sm);
  font-weight: 500;
  letter-spacing: 0.02em;
}
.mh-delta.neg { background: var(--neg-sub); color: var(--neg); }
.mh-delta-abs {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text-2);
}
.mh-updated {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.02em;
  margin-top: auto;
  padding-top: 12px;
}
.mh-updated .upd-em { color: var(--text-2); }

.mh-right {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: flex-end;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 52px;
  padding: 0 28px;
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.01em;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  cursor: pointer;
  text-decoration: none;
  transition: background 120ms, border-color 120ms, color 120ms, filter 120ms;
  white-space: nowrap;
}
.btn-primary {
  background: var(--brand);
  color: var(--canvas);
  width: 100%;
}
.btn-primary:hover { background: var(--brand-hov); }
.btn svg { width: 16px; height: 16px; }

.btn-ghost {
  background: transparent;
  color: var(--text-2);
  height: auto;
  padding: 0;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.btn-ghost:hover { color: var(--text); }
.btn-ghost .ic { width: 14px; height: 14px; }

.mh-spark {
  width: 100%;
  height: 76px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  background: var(--canvas);
}
.mh-spark .label-row {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--text-3);
  letter-spacing: 0.16em;
  text-transform: uppercase;
  margin-bottom: 4px;
}
.mh-spark svg {
  width: 100%;
  height: calc(100% - 18px);
  display: block;
  overflow: visible;
}
.mh-spark path.line {
  fill: none;
  stroke: var(--pos);
  stroke-width: 1.4;
  stroke-dasharray: 1500;
  stroke-dashoffset: 1500;
  animation: draw 1.4s cubic-bezier(0.2, 0, 0, 1) 0.2s forwards;
}
.mh-spark path.fill {
  fill: var(--pos);
  fill-opacity: 0;
  animation: sparkFill 1.4s ease-out 0.6s forwards;
}
@keyframes draw      { to { stroke-dashoffset: 0; } }
@keyframes sparkFill { to { fill-opacity: 0.12; } }

/* ============================================================
   Stats grid
   ============================================================ */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  border-bottom: 1px solid var(--border);
}
.stat {
  padding: 24px 28px 24px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-right: 1px solid var(--border);
  padding-left: 28px;
}
.stat:first-child { padding-left: 0; }
.stat:last-child { border-right: none; padding-right: 0; }
.stat .stat-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
}
.stat .stat-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 24px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.01em;
}
.stat .stat-val.dim { color: var(--text-3); }
.stat .stat-val.pos { color: var(--pos); }
.stat .stat-val.neg { color: var(--neg); }
.stat .stat-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
}
.stat .stat-sub.pos { color: var(--pos); }
.stat .stat-sub.neg { color: var(--neg); }
.stat .stat-sub.dim { color: var(--text-3); }
.stats-grid.row-2 { border-bottom: none; }

/* ============================================================
   Primary chart
   ============================================================ */
.chart-section { padding: 48px 0; }
.chart-card {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.chart-head {
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.seg {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.seg button {
  background: transparent;
  color: var(--text-2);
  border: none;
  font-family: var(--font-mono);
  font-size: 11px;
  height: 28px;
  padding: 0 12px;
  cursor: pointer;
  border-right: 1px solid var(--border);
  transition: background 120ms, color 120ms;
}
.seg button:last-child { border-right: none; }
.seg button:hover { background: var(--hover); color: var(--text); }
.seg button.active {
  background: rgba(200, 242, 92, 0.10);
  color: var(--brand);
}
.chart-head-right {
  margin-left: auto;
  display: flex;
  gap: 8px;
  align-items: center;
}
.indicator-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 11px;
  padding: 0 10px;
  height: 28px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.indicator-pill .x { color: var(--text-3); margin-left: 2px; }
.indicator-pill .x:hover { color: var(--neg); }
.indicator-pill:hover { background: var(--hover); color: var(--text); }
.add-indicator {
  background: transparent;
  border: 1px dashed var(--border-strong);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 11px;
  padding: 0 10px;
  height: 28px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.add-indicator:hover { color: var(--text); border-color: var(--brand); }
.add-indicator svg { width: 12px; height: 12px; }

.chart-wrap {
  height: 520px;
  position: relative;
}
.chart-host { position: absolute; inset: 0; }

/* ============================================================
   Details
   ============================================================ */
.details {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 56px;
  padding: 16px 0 48px;
  border-bottom: 1px solid var(--border);
}
.details h3 {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0 0 20px;
}
.details p {
  color: var(--text-2);
  font-size: 15px;
  line-height: 1.65;
  margin: 0 0 16px;
  max-width: 60ch;
}
.methodology-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  color: var(--text);
  font-size: 14px;
  font-weight: 500;
  border-bottom: 1px solid var(--text);
  padding-bottom: 1px;
  text-decoration: none;
}
.methodology-link:hover {
  color: var(--brand);
  border-bottom-color: var(--brand);
}
.methodology-link svg { width: 14px; height: 14px; }

.spec-table {
  width: 100%;
  border-collapse: collapse;
}
.spec-table th, .spec-table td {
  text-align: left;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
  font-size: 14px;
  vertical-align: top;
}
.spec-table th {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  width: 44%;
}
.spec-table td {
  color: var(--text);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  font-weight: 500;
}
.spec-table td a {
  color: var(--brand);
  border-bottom: 1px solid currentColor;
  padding-bottom: 1px;
  text-decoration: none;
}
.spec-table .regular { font-weight: 400; }

/* ============================================================
   Related markets
   ============================================================ */
.related {
  padding: 48px 0;
  border-bottom: 1px solid var(--border);
}
.related h3 {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0 0 24px;
}
.related-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}
.market-card {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 18px 20px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: background 120ms, border-color 120ms;
  text-decoration: none;
}
.market-card:hover {
  background: var(--elevated);
  border-color: var(--border-strong);
}
.market-card .sym {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}
.market-card .name {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
  margin-bottom: 6px;
}
.market-card .pxrow {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.market-card .px {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
  transition: color 600ms;
}
.market-card .px.flash-up   { color: var(--pos); }
.market-card .px.flash-down { color: var(--neg); }
.market-card .delta {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  font-weight: 500;
}
.market-card .delta.pos { color: var(--pos); }
.market-card .delta.neg { color: var(--neg); }
.market-card svg {
  width: 100%;
  height: 36px;
  display: block;
  margin-top: 6px;
}
.market-card svg .line { fill: none; stroke-width: 1.2; }
.market-card svg .line.pos { stroke: var(--pos); }
.market-card svg .line.neg { stroke: var(--neg); }
.market-card svg .fill.pos { fill: var(--pos); fill-opacity: 0.08; }
.market-card svg .fill.neg { fill: var(--neg); fill-opacity: 0.08; }

/* ============================================================
   OB + Tape preview
   ============================================================ */
.preview { padding: 48px 0 64px; }
.preview-head {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  cursor: pointer;
}
.preview-head h3 {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0;
}
.preview-head .caret {
  width: 16px;
  height: 16px;
  color: var(--text-2);
  transition: transform 200ms;
}
.preview.collapsed .preview-head .caret { transform: rotate(-90deg); }
.preview-sub {
  margin-left: auto;
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
}
.preview-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  overflow: hidden;
  transition: max-height 280ms ease-out, opacity 200ms;
  max-height: 720px;
}
.preview.collapsed .preview-body {
  max-height: 0;
  opacity: 0;
}
.preview-card {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.preview-card-head {
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 12px;
}
.preview-card-head .head-ic { width: 14px; height: 14px; color: var(--text-2); }
.preview-card-head .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.20em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
}
.preview-card-head .lbl-badge {
  background: var(--elevated);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  margin-left: auto;
}
.preview-cta-row {
  display: flex;
  justify-content: center;
  padding-top: 24px;
}
.preview-cta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--brand);
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.005em;
  padding: 12px 20px;
  border: 1px solid var(--brand);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 120ms;
  text-decoration: none;
}
.preview-cta:hover { background: rgba(200, 242, 92, 0.10); }
.preview-cta svg { width: 14px; height: 14px; }

/* OB preview */
.ob-prev {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  padding: 10px 0;
}
.ob-cols {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 6px 14px;
  color: var(--text-3);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
}
.ob-cols span { text-align: right; }
.ob-cols span:first-child { text-align: left; }
.ob-row {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 0 14px;
  height: 20px;
  align-items: center;
  position: relative;
}
.ob-row .bar {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
}
.ob-row.ask .bar { background: var(--neg-bar); }
.ob-row.bid .bar { background: var(--pos-bar); }
.ob-row > * { position: relative; text-align: right; }
.ob-row .ob-px { text-align: left; }
.ob-row.ask .ob-px { color: var(--neg); }
.ob-row.bid .ob-px { color: var(--pos); }
.ob-row .ob-qty { color: var(--text); }
.ob-row .ob-tot { color: var(--text-3); }
.ob-mid {
  text-align: center;
  padding: 6px 0;
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  margin: 4px 0;
  background: var(--elevated);
}
.ob-mid-spread {
  color: var(--text-3);
  font-size: 10px;
  font-weight: 500;
  margin-left: 8px;
}

/* Tape preview */
.tape-prev {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
.tape-cols {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 16px;
  padding: 8px 14px;
  color: var(--text-3);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
}
.tape-cols span:nth-child(2),
.tape-cols span:nth-child(3) { text-align: right; }
.tape-row {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 16px;
  padding: 0 14px;
  height: 20px;
  align-items: center;
  color: var(--text);
}
.tape-row.up .t-px,
.tape-row.up .t-side { color: var(--pos); }
.tape-row.down .t-px,
.tape-row.down .t-side { color: var(--neg); }
.tape-row .t-time { color: var(--text-3); }
.tape-row .t-qty { text-align: right; }
.tape-row .t-px  { text-align: right; }
.tape-row .t-side { text-align: center; font-size: 10px; }
.tape-row.large { background: rgba(255, 255, 255, 0.025); }
.tape-row.enter { animation: tapeIn 200ms ease-out; }
@keyframes tapeIn {
  from { transform: translateY(-6px); opacity: 0; }
  to   { transform: translateY(0); opacity: 1; }
}

/* Responsive */
@media (max-width: 1100px) {
  .market-header { grid-template-columns: 1fr; }
  .mh-right { align-items: stretch; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .stat { border-right: none; }
  .related-grid { grid-template-columns: repeat(2, 1fr); }
  .preview-body { grid-template-columns: 1fr; }
  .details { grid-template-columns: 1fr; gap: 32px; }
}
@media (max-width: 560px) {
  .stats-grid { grid-template-columns: 1fr; }
  .related-grid { grid-template-columns: 1fr; }
}
</style>
