<script setup lang="ts">
/**
 * /markets/[slug] — Single-market detail page.
 *
 * Dark theme. Renders inside the `app` layout (sidebar + topbar are shared).
 * Layout per design:
 *   sub-topbar (breadcrumbs)
 *   header (huge price + delta + sparkline + Trade button)
 *   stats grid (2 rows × 4)
 *   primary candlestick chart (lightweight-charts)
 *   details (about + spec table)
 *   related markets (4 cards)
 *   preview (order book + tape, collapsible)
 */
import { createChart, ColorType, CrosshairMode } from 'lightweight-charts'

definePageMeta({ layout: 'app' })

interface MarketDef {
  sym: string
  name: string
  basePrice: number
  decimals: number
  desc: string
}

const MARKETS: Record<string, MarketDef> = {
  'eai-idx': {
    sym: 'EAI-IDX',
    name: 'AI Index',
    basePrice: 0.001005,
    decimals: 6,
    desc: 'The Exascale AI Index represents the unified price of AI inference, computed from observed market data and consumption metrics across the text, speech, image, video, and niche credit markets.',
  },
  'text-spot':  { sym: 'TEXT-SPOT',  name: 'Text Credit',   basePrice: 0.001210, decimals: 6, desc: 'Spot market for text-inference credits.' },
  'image-spot': { sym: 'IMAGE-SPOT', name: 'Image Credit',  basePrice: 0.008000, decimals: 6, desc: 'Spot market for image-generation credits.' },
  'h100-spot':  { sym: 'H100-SPOT',  name: 'H100 GPU Hour', basePrice: 2.99,     decimals: 4, desc: 'Spot market for H100 80GB SXM5 GPU hours.' },
}

const route = useRoute()
const slug = computed(() => {
  const s = Array.isArray(route.params.slug) ? route.params.slug[0] : route.params.slug
  return s || 'eai-idx'
})
const market = computed<MarketDef>(() => MARKETS[slug.value] ?? MARKETS['eai-idx']!)
useHead({ title: () => `${market.value.sym} · ${market.value.name} — Exascale` })

// =====================================================
// State
// =====================================================
const midPrice = ref(market.value.basePrice)
const open24h = ref(market.value.basePrice * 0.998)
const high24h = ref(market.value.basePrice * 1.007)
const low24h  = ref(market.value.basePrice * 0.991)
const flashClass = ref<'flash-up' | 'flash-down' | ''>('')
const lastUpdated = ref('14:23:47.123 UTC')
const previewCollapsed = ref(false)

type TF = '1m' | '5m' | '15m' | '1h' | '4h' | '1d' | '1w' | '1M' | '1y' | 'all'
const TIMEFRAMES: TF[] = ['1m', '5m', '15m', '1h', '4h', '1d', '1w', '1M', '1y', 'all']
const tf = ref<TF>('1d')

// =====================================================
// Formatters
// =====================================================
function fmtPx(n: number, dp = market.value.decimals) {
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtInt(n: number) {
  return Math.round(n).toLocaleString('en-US')
}
function fmtUsd(n: number, dp = 2) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function timestampNow() {
  const d = new Date()
  return d.toLocaleTimeString('en-GB', { hour12: false, timeZone: 'UTC' }) + '.' + String(d.getMilliseconds()).padStart(3, '0') + ' UTC'
}

const deltaPct = computed(() => ((midPrice.value - open24h.value) / open24h.value) * 100)
const deltaAbs = computed(() => midPrice.value - open24h.value)

// =====================================================
// Chart series (daily + intraday)
// =====================================================
interface Candle { time: number | string; open: number; high: number; low: number; close: number }
interface Vol { time: number | string; value: number; color: string }

const dailySeries: Candle[] = []
const dailyVol:    Vol[]    = []
const intradaySeries: Candle[] = []
const intradayVol:    Vol[]    = []

function isoDate(d: Date) { return d.toISOString().slice(0, 10) }

function buildDaily() {
  const N = 365
  const base = market.value.basePrice
  let p = base * 0.965
  const endDay = new Date('2026-05-19')
  endDay.setUTCHours(0, 0, 0, 0)
  for (let i = 0; i < N; i++) {
    const drift = base * 3e-7
    const noise = (Math.random() - 0.48) * base * 0.008
    const dd = Math.random() < 0.016 ? -(base * 0.008 + Math.random() * base * 0.018) : 0
    p = Math.max(base * 0.90, Math.min(base * 1.045, p + drift + noise + dd))
    const open = p
    const close = Math.max(base * 0.90, Math.min(base * 1.045, p + (Math.random() - 0.48) * base * 0.010))
    const high = Math.max(open, close) + Math.random() * base * 0.008
    const low  = Math.min(open, close) - Math.random() * base * 0.008
    p = close
    const d = new Date(endDay)
    d.setUTCDate(endDay.getUTCDate() - (N - 1 - i))
    const time = isoDate(d)
    dailySeries.push({ time, open, high, low, close })
    const vol = Math.exp(Math.random() * 1.6 + 11)
    dailyVol.push({ time, value: vol, color: close >= open ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)' })
  }
  const last = dailySeries[dailySeries.length - 1]!
  last.close = midPrice.value
  last.high  = Math.max(last.high, midPrice.value)
  last.low   = Math.min(last.low, midPrice.value)
}

function buildIntraday() {
  const N = 288
  const base = market.value.basePrice
  let p = base * 0.994
  const startTime = Math.floor(Date.now() / 1000) - N * 300
  for (let i = 0; i < N; i++) {
    const drift = (base - p) * 0.025
    const noise = (Math.random() - 0.5) * base * 0.0035
    const open = p
    const close = Math.max(base * 0.93, Math.min(base * 1.06, p + drift + noise))
    const swing = Math.random() * base * 0.0025 + base * 0.0005
    const high = Math.max(open, close) + swing
    const low  = Math.min(open, close) - swing
    const t = startTime + i * 300
    intradaySeries.push({ time: t, open, high, low, close })
    const vol = Math.exp(Math.random() * 1.4 + 6)
    intradayVol.push({ time: t, value: vol, color: close >= open ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)' })
    p = close
  }
  midPrice.value = intradaySeries[intradaySeries.length - 1]!.close
  high24h.value  = Math.max(...intradaySeries.map(c => c.high))
  low24h.value   = Math.min(...intradaySeries.map(c => c.low))
}

// =====================================================
// Chart init
// =====================================================
const chartHost = ref<HTMLDivElement | null>(null)
let chart: ReturnType<typeof createChart> | null = null
let candleSeries: ReturnType<NonNullable<typeof chart>['addCandlestickSeries']> | null = null
let volumeSeries: ReturnType<NonNullable<typeof chart>['addHistogramSeries']> | null = null
let chartRO: ResizeObserver | null = null

function initChart() {
  if (!chartHost.value) return
  chart = createChart(chartHost.value, {
    layout: {
      background: { type: ColorType.Solid, color: '#0A0B0E' },
      textColor: '#9A9A95',
      fontFamily: 'JetBrains Mono, ui-monospace, monospace',
      fontSize: 10,
    },
    grid: {
      vertLines: { color: 'rgba(255,255,255,0.03)' },
      horzLines: { color: 'rgba(255,255,255,0.03)' },
    },
    rightPriceScale: {
      borderColor: 'rgba(255,255,255,0.08)',
      scaleMargins: { top: 0.08, bottom: 0.28 },
    },
    timeScale: {
      borderColor: 'rgba(255,255,255,0.08)',
      timeVisible: true,
      secondsVisible: false,
    },
    crosshair: {
      mode: CrosshairMode.Normal,
      vertLine: { color: 'rgba(232,230,224,0.35)', width: 1, style: 0 },
      horzLine: { color: 'rgba(232,230,224,0.35)', width: 1, style: 0 },
    },
  })
  candleSeries = chart.addCandlestickSeries({
    upColor:        '#19C37D',
    downColor:      '#EF4444',
    borderUpColor:  '#19C37D',
    borderDownColor:'#EF4444',
    wickUpColor:    '#19C37D',
    wickDownColor:  '#EF4444',
    priceFormat: { type: 'price', precision: market.value.decimals, minMove: market.value.decimals === 6 ? 0.000001 : 0.0001 },
  })
  volumeSeries = chart.addHistogramSeries({
    priceFormat: { type: 'volume' },
    priceScaleId: '',
    color: 'rgba(232,230,224,0.18)',
  })
  volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.80, bottom: 0 } })

  applyTimeframe(tf.value)

  chartRO = new ResizeObserver(() => {
    if (chart && chartHost.value) {
      chart.applyOptions({ width: chartHost.value.clientWidth, height: chartHost.value.clientHeight })
    }
  })
  chartRO.observe(chartHost.value)
}

function applyTimeframe(t: TF) {
  tf.value = t
  if (!candleSeries || !volumeSeries) return
  const useDaily = ['1d', '1w', '1M', '1y', 'all'].includes(t)
  if (useDaily) {
    const sliceN = t === '1w' ? 90 : t === '1M' ? 180 : 365
    const c = dailySeries.slice(-sliceN)
    const v = dailyVol.slice(-sliceN)
    candleSeries.setData(c.map(x => ({ ...x, time: x.time as never })))
    volumeSeries.setData(v.map(x => ({ ...x, time: x.time as never })))
  } else {
    let n = 200
    if (t === '1m')  n = 180
    if (t === '5m')  n = 200
    if (t === '15m') n = 160
    if (t === '1h')  n = 120
    if (t === '4h')  n = 60
    const c = intradaySeries.slice(-n)
    const v = intradayVol.slice(-n)
    candleSeries.setData(c.map(x => ({ ...x, time: x.time as never })))
    volumeSeries.setData(v.map(x => ({ ...x, time: x.time as never })))
  }
  chart?.timeScale().fitContent()
}

// =====================================================
// Header sparkline (60 points from intraday close)
// =====================================================
const hdrSparkLine = ref('')
const hdrSparkFill = ref('')
function buildHdrSpark() {
  const sub = intradaySeries.slice(-60).map(c => c.close)
  if (!sub.length) return
  const W = 320, H = 60
  const min = Math.min(...sub), max = Math.max(...sub)
  const range = max - min || 1
  const stepX = W / (sub.length - 1)
  const pad = 6
  const usableH = H - pad * 2
  const path = sub.map((v, i) => {
    const x = i * stepX
    const y = pad + usableH - ((v - min) / range) * usableH
    return (i === 0 ? 'M' : 'L') + x.toFixed(1) + ' ' + y.toFixed(1)
  }).join(' ')
  hdrSparkLine.value = path
  hdrSparkFill.value = path + ' L ' + W + ' ' + H + ' L 0 ' + H + ' Z'
}

// =====================================================
// Order book + tape preview
// =====================================================
interface BookLevel { price: number; qty: number; cum: number }
const asksPreview = ref<BookLevel[]>([])
const bidsPreview = ref<BookLevel[]>([])
const maxBookQty = computed(() =>
  Math.max(1, ...asksPreview.value.map(l => l.qty), ...bidsPreview.value.map(l => l.qty)),
)
const spread = computed(() => {
  const a = asksPreview.value[0]?.price ?? 0
  const b = bidsPreview.value[0]?.price ?? 0
  return a - b
})
const spreadPct = computed(() => {
  if (midPrice.value === 0) return 0
  return (spread.value / midPrice.value) * 100
})

function generateBook() {
  const mid = midPrice.value
  const half = mid * 0.0005
  const asks: { price: number; qty: number }[] = []
  const bids: { price: number; qty: number }[] = []
  for (let i = 0; i < 8; i++) {
    const baseQty = 5000 + Math.pow(i, 1.6) * 4500
    asks.push({ price: mid + half + i * mid * 0.0008, qty: Math.round(baseQty + (Math.random() - 0.5) * 800) })
    bids.push({ price: mid - half - i * mid * 0.0008, qty: Math.round(baseQty + (Math.random() - 0.5) * 800) })
  }
  let aRun = 0
  const aDisplay: BookLevel[] = asks.map(l => ({ ...l, cum: (aRun += l.qty) }))
  let bRun = 0
  const bDisplay: BookLevel[] = bids.map(l => ({ ...l, cum: (bRun += l.qty) }))
  asksPreview.value = aDisplay.slice().reverse()
  bidsPreview.value = bDisplay
}

interface TapeRow { side: 'up' | 'down'; px: number; qty: number; time: string; large: boolean }
const MAX_TAPE = 18
const tape = ref<TapeRow[]>([])
function addTrade() {
  const burst = 1 + Math.floor(Math.random() * 2)
  const next = [...tape.value]
  for (let i = 0; i < burst; i++) {
    const side: 'up' | 'down' = Math.random() < 0.5 ? 'up' : 'down'
    const base = midPrice.value
    const px = side === 'up' ? base + Math.random() * base * 0.0015 : base - Math.random() * base * 0.0015
    const ln = Math.exp(Math.random() * 2.6 + 4.4)
    const isLarge = Math.random() < 0.05
    const qty = Math.round(isLarge ? ln * 60 : ln)
    const now = new Date()
    const time = now.toLocaleTimeString('en-GB', { hour12: false }) + '.' + String(now.getMilliseconds()).padStart(3, '0')
    next.unshift({ side, px, qty, time, large: isLarge })
    if (next.length > MAX_TAPE) next.pop()
  }
  tape.value = next
}

// =====================================================
// Live tick
// =====================================================
function chartTick() {
  const base = market.value.basePrice
  const drift = (base - midPrice.value) * 0.04
  const noise = (Math.random() - 0.5) * base * 0.0028
  const prev = midPrice.value
  midPrice.value = Math.max(base * 0.94, Math.min(base * 1.065, midPrice.value + drift + noise))

  // Update last bars
  if (intradaySeries.length && candleSeries) {
    const last = intradaySeries[intradaySeries.length - 1]!
    last.close = midPrice.value
    last.high  = Math.max(last.high, midPrice.value)
    last.low   = Math.min(last.low, midPrice.value)
    if (!['1d', '1w', '1M', '1y', 'all'].includes(tf.value)) {
      candleSeries.update({ ...last, time: last.time as never })
    }
  }
  if (dailySeries.length && candleSeries) {
    const last = dailySeries[dailySeries.length - 1]!
    last.close = midPrice.value
    last.high  = Math.max(last.high, midPrice.value)
    last.low   = Math.min(last.low, midPrice.value)
    if (['1d', '1w', '1M', '1y', 'all'].includes(tf.value)) {
      candleSeries.update({ ...last, time: last.time as never })
    }
  }

  high24h.value = Math.max(high24h.value, midPrice.value)
  low24h.value  = Math.min(low24h.value, midPrice.value)
  lastUpdated.value = timestampNow()

  flashClass.value = midPrice.value >= prev ? 'flash-up' : 'flash-down'
  setTimeout(() => { flashClass.value = '' }, 700)

  updateRelated()
}

// =====================================================
// Related markets
// =====================================================
interface RelatedMarket {
  sym: string
  slug: string
  name: string
  px: number
  dpct: number
  color: 'pos' | 'neg'
  flash: '' | 'flash-up' | 'flash-down'
  sparkPath: string
  sparkFill: string
}
const RELATED_DEFS: Array<Omit<RelatedMarket, 'flash' | 'sparkPath' | 'sparkFill'>> = [
  { sym: 'TEXT-SPOT',  slug: 'text-spot',  name: 'Text credits',   px: 0.001210, dpct:  0.42, color: 'pos' },
  { sym: 'SPEECH-SPOT',slug: 'speech-spot',name: 'Speech credits', px: 0.001200, dpct: -0.18, color: 'neg' },
  { sym: 'IMAGE-SPOT', slug: 'image-spot', name: 'Image credits',  px: 0.008000, dpct:  1.24, color: 'pos' },
  { sym: 'VIDEO-SPOT', slug: 'video-spot', name: 'Video credits',  px: 0.250000, dpct: -0.65, color: 'neg' },
]

function genSparkPaths(trend: number) {
  const W = 200, H = 36
  const N = 30
  const pts: number[] = []
  let v = 1.0
  for (let i = 0; i < N; i++) {
    v += (Math.random() - (trend < 0 ? 0.55 : 0.45)) * 0.012
    pts.push(v)
  }
  const min = Math.min(...pts), max = Math.max(...pts)
  const range = max - min || 1
  const stepX = W / (pts.length - 1)
  const pad = 4
  const usableH = H - pad * 2
  const line = pts.map((p, i) => {
    const x = i * stepX
    const y = pad + usableH - ((p - min) / range) * usableH
    return (i === 0 ? 'M' : 'L') + x.toFixed(1) + ' ' + y.toFixed(1)
  }).join(' ')
  return { line, fill: line + ` L ${W} ${H} L 0 ${H} Z` }
}

const related = ref<RelatedMarket[]>(
  RELATED_DEFS.map(m => {
    const { line, fill } = genSparkPaths(m.dpct)
    return { ...m, flash: '', sparkPath: line, sparkFill: fill }
  }),
)

function updateRelated() {
  related.value = related.value.map(m => {
    const noise = (Math.random() - 0.5) * 0.0008
    const newDpct = m.dpct + noise
    const drift = Math.random() * (m.px < 0.01 ? 0.000003 : m.px < 0.1 ? 0.00002 : 0.0008)
    const sign = Math.random() < 0.5 ? -1 : 1
    const newPx = Math.max(m.px * 0.985, m.px + drift * sign)
    const flash: '' | 'flash-up' | 'flash-down' = newPx >= m.px ? 'flash-up' : 'flash-down'
    return {
      ...m,
      px: newPx,
      dpct: newDpct,
      color: (newDpct >= 0 ? 'pos' : 'neg') as 'pos' | 'neg',
      flash,
    }
  })
  setTimeout(() => {
    related.value = related.value.map(m => ({ ...m, flash: '' }))
  }, 700)
}

function relatedPxDecimals(px: number) {
  return px >= 0.1 ? 4 : 6
}

// =====================================================
// Watch slug — rebuild series when market changes
// =====================================================
let tickInterval: ReturnType<typeof setInterval> | null = null
let bookInterval: ReturnType<typeof setInterval> | null = null
let tapeInterval: ReturnType<typeof setInterval> | null = null

function clearIntervals() {
  if (tickInterval) clearInterval(tickInterval)
  if (bookInterval) clearInterval(bookInterval)
  if (tapeInterval) clearInterval(tapeInterval)
  tickInterval = bookInterval = tapeInterval = null
}

watch(slug, () => {
  // Rebuild series for new market
  dailySeries.length = 0
  dailyVol.length = 0
  intradaySeries.length = 0
  intradayVol.length = 0
  midPrice.value = market.value.basePrice
  open24h.value = market.value.basePrice * 0.998
  buildDaily()
  buildIntraday()
  buildHdrSpark()
  generateBook()
  tape.value = []
  for (let i = 0; i < MAX_TAPE; i++) addTrade()
  if (chart) {
    chart.remove()
    chart = null
    initChart()
  }
})

onMounted(() => {
  buildDaily()
  buildIntraday()
  initChart()
  buildHdrSpark()
  generateBook()
  for (let i = 0; i < MAX_TAPE; i++) addTrade()
  tickInterval = setInterval(chartTick, 3000)
  bookInterval = setInterval(generateBook, 4000)
  tapeInterval = setInterval(addTrade, 5000)
})
onBeforeUnmount(() => {
  clearIntervals()
  chartRO?.disconnect()
  chart?.remove()
})
</script>

<template>
  <div class="market-detail">
    <!-- Sub-topbar: breadcrumbs -->
    <div class="subbar">
      <nav class="breadcrumbs">
        <NuxtLink to="/trade">Markets</NuxtLink>
        <span class="sep">›</span>
        <span class="cur">{{ market.sym }}</span>
      </nav>
    </div>

    <main class="page">
      <!-- ============ Market header ============ -->
      <section class="market-header">
        <div class="mh-left">
          <div class="row1">
            <span class="sym-tag">{{ market.sym }}</span>
            <span class="meta-pill"><span class="pulse-dot" />Open · 24/7</span>
            <span class="meta-pill meta-2">Cash-settled · USD</span>
          </div>
          <h1>{{ market.name }}</h1>
          <div class="mh-price" :class="flashClass">{{ fmtPx(midPrice) }}</div>
          <div class="mh-delta-row">
            <span class="mh-delta" :class="{ neg: deltaPct < 0 }">
              {{ deltaPct >= 0 ? '▲' : '▼' }} {{ Math.abs(deltaPct).toFixed(2) }}%
            </span>
            <span class="mh-delta-abs">
              {{ deltaAbs >= 0 ? '+' : '−' }}${{ Math.abs(deltaAbs).toFixed(market.decimals) }}
            </span>
          </div>
          <div class="mh-updated">
            Last updated <span>{{ lastUpdated }}</span>
            · Next index print
            <span class="upd-em">16:00 UTC</span>
          </div>
        </div>

        <div class="mh-right">
          <NuxtLink to="/trade" class="btn btn-primary">
            Trade this market
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
          <button class="btn-ghost" type="button">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" class="ic">
              <path d="M8 1.5l1.95 4.13L14.5 6.3l-3.4 3.13.86 4.57L8 11.84 4.04 14l.86-4.57L1.5 6.3l4.55-.67z" />
            </svg>
            Add to watchlist
          </button>
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

      <!-- ============ Stats row 1 ============ -->
      <section class="stats-grid">
        <div class="stat">
          <span class="stat-lbl">— 24h volume</span>
          <span class="stat-val">$1.24M</span>
          <span class="stat-sub pos">+12.3% vs avg</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— 24h high</span>
          <span class="stat-val">{{ fmtUsd(high24h, market.decimals) }}</span>
          <span class="stat-sub">@ 09:42 UTC</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— 24h low</span>
          <span class="stat-val">{{ fmtUsd(low24h, market.decimals) }}</span>
          <span class="stat-sub">@ 02:18 UTC</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— All-time high</span>
          <span class="stat-val">{{ fmtUsd(market.basePrice * 1.04, market.decimals) }}</span>
          <span class="stat-sub">12 Mar 2026</span>
        </div>
      </section>

      <!-- ============ Stats row 2 ============ -->
      <section class="stats-grid row-2">
        <div class="stat">
          <span class="stat-lbl">— 7d change</span>
          <span class="stat-val neg">−0.42%</span>
          <span class="stat-sub">${{ (market.basePrice * 0.0042).toFixed(market.decimals) }}</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— 30d change</span>
          <span class="stat-val pos">+2.31%</span>
          <span class="stat-sub">${{ (market.basePrice * 0.0231).toFixed(market.decimals) }}</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— YTD change</span>
          <span class="stat-val pos">+4.18%</span>
          <span class="stat-sub">Since 01 Jan 2026</span>
        </div>
        <div class="stat">
          <span class="stat-lbl">— Open interest</span>
          <span class="stat-val dim">—</span>
          <span class="stat-sub dim">Coming v1.5</span>
        </div>
      </section>

      <!-- ============ Primary chart ============ -->
      <section class="chart-section">
        <div class="chart-card">
          <div class="chart-head">
            <div class="seg">
              <button
                v-for="t in TIMEFRAMES"
                :key="t"
                type="button"
                :class="{ active: tf === t }"
                @click="applyTimeframe(t)"
              >{{ t }}</button>
            </div>
            <div class="chart-head-right">
              <span class="indicator-pill">MA(20) <span class="x">×</span></span>
              <span class="indicator-pill">RSI(14) <span class="x">×</span></span>
              <span class="indicator-pill">BB(20,2) <span class="x">×</span></span>
              <button class="add-indicator" type="button">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                  <path d="M8 3v10M3 8h10" />
                </svg>
                Indicators
              </button>
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
          <p>{{ market.desc }}</p>
          <p>
            The index is published daily at 16:00 UTC by an independent administrator,
            methodology audited externally on a quarterly cadence. Intra-day prints shown
            here are real-time observations on the constituent venues, weighted by
            trailing-30-day volume and trimmed at the 5th and 95th percentiles to limit
            outlier influence.
          </p>
          <p>
            <span class="mono">{{ market.sym }}</span> is cash-settled in USD. Counterparties may also
            redeem against any of the constituent sub-credit markets at the index's daily
            fixing rate, with no settlement fees on physical redemption above 100,000 credits.
          </p>
          <NuxtLink to="/benchmark" class="methodology-link">
            View full methodology
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M5 11L11 5M6 5h5v5" />
            </svg>
          </NuxtLink>
        </div>

        <div>
          <h3>Specifications</h3>
          <table class="spec-table">
            <tbody>
              <tr><th>Settlement</th><td>Cash-settled in USD</td></tr>
              <tr><th>Tick size</th><td>${{ (market.decimals === 6 ? 0.000001 : 0.0001).toFixed(market.decimals) }}</td></tr>
              <tr><th>Minimum order</th><td>100 credits · $0.10</td></tr>
              <tr><th>Trading hours</th><td>24 / 7 · 365</td></tr>
              <tr><th>Currency</th><td>USD <span class="dim regular">(JPY pricing available)</span></td></tr>
              <tr>
                <th>Fees · retail tier</th>
                <td>0.50% maker / 1.00% taker
                  <span class="dim regular"><br /><a href="#">Fee schedule →</a></span>
                </td>
              </tr>
              <tr><th>Constituents</th><td>TEXT · SPEECH · IMAGE · VIDEO · NICHE</td></tr>
              <tr><th>Index administrator</th><td>Exascale Index Co.</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- ============ Related markets ============ -->
      <section class="related">
        <h3>Related markets</h3>
        <div class="related-grid">
          <NuxtLink
            v-for="m in related"
            :key="m.sym"
            :to="'/markets/' + m.slug"
            class="market-card"
          >
            <span class="sym">{{ m.sym }}</span>
            <span class="name">{{ m.name }}</span>
            <div class="pxrow">
              <span class="px" :class="m.flash">{{ fmtPx(m.px, relatedPxDecimals(m.px)) }}</span>
              <span class="delta" :class="m.color">
                {{ m.dpct >= 0 ? '▲' : '▼' }} {{ Math.abs(m.dpct).toFixed(2) }}%
              </span>
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
          <span class="preview-sub">Live · 10 levels</span>
        </div>
        <div class="preview-body">
          <div class="preview-card">
            <div class="preview-card-head">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
                <path d="M3 4l5 3 5-3M3 8l5 3 5-3M3 12l5 3 5-3" />
              </svg>
              <span class="lbl">Order book</span>
              <span class="lbl-badge">10 LVL</span>
            </div>
            <div class="ob-prev">
              <div class="ob-cols">
                <span>Price (USD)</span>
                <span>Size</span>
                <span>Total</span>
              </div>
              <div
                v-for="(l, i) in asksPreview"
                :key="'a-' + i"
                class="ob-row ask"
              >
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtInt(l.qty) }}</span>
                <span class="ob-tot">{{ fmtInt(l.cum) }}</span>
              </div>
              <div class="ob-mid">
                {{ fmtPx(midPrice) }}
                <span class="ob-mid-spread">spread {{ fmtPx(spread) }} / {{ spreadPct.toFixed(2) }}%</span>
              </div>
              <div
                v-for="(l, i) in bidsPreview"
                :key="'b-' + i"
                class="ob-row bid"
              >
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtInt(l.qty) }}</span>
                <span class="ob-tot">{{ fmtInt(l.cum) }}</span>
              </div>
            </div>
          </div>

          <div class="preview-card">
            <div class="preview-card-head">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
                <path d="M2 8h3l2-4 2 8 2-5 2 3h1" />
              </svg>
              <span class="lbl">Recent trades</span>
              <span class="lbl-badge">LIVE</span>
            </div>
            <div class="tape-prev">
              <div class="tape-cols">
                <span>Time</span>
                <span>Price</span>
                <span>Qty</span>
                <span />
              </div>
              <div
                v-for="(t, i) in tape"
                :key="t.time + i"
                class="tape-row"
                :class="[t.side, { large: t.large, enter: i === 0 }]"
              >
                <span class="t-time">{{ t.time }}</span>
                <span class="t-px">{{ fmtPx(t.px) }}</span>
                <span class="t-qty">{{ fmtInt(t.qty) }}</span>
                <span class="t-side">{{ t.side === 'up' ? '▲' : '▼' }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="preview-cta-row">
          <NuxtLink to="/trade" class="preview-cta">
            View full trading view
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
</style>
