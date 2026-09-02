<script setup lang="ts">
/**
 * /trade — Headline trading dashboard. Dense Bloomberg-style grid.
 * Renders inside the `app` layout (sidebar + topbar are shared).
 *
 * Layout (fills the main area, no scroll):
 *   ┌───────────────────────────────┬──────────────────┐
 *   │  chart       (1.55fr)         │  order book      │
 *   ├───────────────────────────────┼──────────────────┤
 *   │  order entry                  │  trade tape      │
 *   ├───────────────────────────────┴──────────────────┤
 *   │  positions footer (200px)                        │
 *   └──────────────────────────────────────────────────┘
 *
 * Live ticks: candles update every 3s, book every 1.5s, tape every 5s.
 */
import { createChart, ColorType, CrosshairMode } from 'lightweight-charts'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Trade — EAI-IDX · 1Trade' })

// =====================================================
// Types + state
// =====================================================
type TF = '1m' | '5m' | '15m' | '1h' | '4h' | '1d'
type Side = 'buy' | 'sell'
type OrderType = 'market' | 'limit' | 'stop'

interface BookLevel { price: number; qty: number; changed: boolean }
interface Position {
  market: string
  side: 'long' | 'short'
  size: number
  entry: number
  mark: number
}
interface TapeRow {
  side: 'up' | 'down'
  px: number
  qty: number
  time: string
  large: boolean
}

const TIMEFRAMES: TF[] = ['1m', '5m', '15m', '1h', '4h', '1d']

const tf = ref<TF>('5m')
const midPrice = ref(0.001005)
const open24h = 0.001003
const high24h = ref(0.001012)
const low24h  = ref(0.000996)
const flashHeader = ref<'up' | 'down' | null>(null)

const positions = reactive<Position[]>([
  { market: 'EAI-IDX',    side: 'long',  size: 50000, entry: 0.000980, mark: 0.001005 },
  { market: 'TEXT-SPOT',  side: 'long',  size: 12000, entry: 0.00118,  mark: 0.00120 },
  { market: 'IMAGE-SPOT', side: 'short', size: 5000,  entry: 0.00810,  mark: 0.00798 },
  { market: 'H100-SPOT',  side: 'long',  size: 8,     entry: 2.95,     mark: 2.99 },
])

const posTab = ref<'open' | 'closed' | 'all'>('open')

// =====================================================
// Formatters
// =====================================================
function fmtPx(n: number, dp = 6) {
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtUsd(n: number, dp = 2) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtInt(n: number) {
  return Math.round(n).toLocaleString('en-US')
}

// =====================================================
// Derived header values
// =====================================================
const deltaPct = computed(() => ((midPrice.value - open24h) / open24h) * 100)
const arrowUp = computed(() => deltaPct.value >= 0)
const headerPxClass = computed(() => {
  if (flashHeader.value === 'up') return 'flash-up'
  if (flashHeader.value === 'down') return 'flash-down'
  return ''
})

// =====================================================
// Lightweight charts
// =====================================================
const chartContainer = ref<HTMLDivElement | null>(null)
let chart: ReturnType<typeof createChart> | null = null
let candleSeries: ReturnType<NonNullable<typeof chart>['addCandlestickSeries']> | null = null
let volumeSeries: ReturnType<NonNullable<typeof chart>['addHistogramSeries']> | null = null
let chartRO: ResizeObserver | null = null

interface Candle { time: number; open: number; high: number; low: number; close: number }
const candles: Candle[] = []
let lastBarTime = 0
let lastBarOpen = 0.001005

function seedChart() {
  const N = 200
  let p = 0.001005
  const target = 0.001005
  const startTime = Math.floor(Date.now() / 1000) - N * 60
  for (let i = 0; i < N; i++) {
    const drift = (target - p) * 0.02
    const noise = (Math.random() - 0.5) * 0.0000035
    const open = p
    const close = Math.max(0.00094, Math.min(0.001065, p + drift + noise))
    const swing = Math.random() * 0.0000025 + 0.0000005
    const high = Math.max(open, close) + swing
    const low  = Math.min(open, close) - swing
    const t = startTime + i * 60
    candles.push({ time: t, open, high, low, close })
    p = close
  }
  midPrice.value = candles[candles.length - 1]!.close
  lastBarTime = candles[candles.length - 1]!.time
  lastBarOpen = candles[candles.length - 1]!.open
}

function initChart() {
  if (!chartContainer.value) return
  chart = createChart(chartContainer.value, {
    layout: {
      background: { type: ColorType.Solid, color: '#0A0A0A' },
      textColor: '#A8A196',
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
      vertLine: { color: 'rgba(232,230,224,0.30)', width: 1, style: 0 },
      horzLine: { color: 'rgba(232,230,224,0.30)', width: 1, style: 0 },
    },
    handleScroll: true,
    handleScale: true,
  })
  candleSeries = chart.addCandlestickSeries({
    upColor:      '#19C37D',
    downColor:    '#EF4444',
    borderUpColor:'#19C37D',
    borderDownColor:'#EF4444',
    wickUpColor:  '#19C37D',
    wickDownColor:'#EF4444',
    priceFormat:  { type: 'price', precision: 6, minMove: 0.000001 },
  })
  volumeSeries = chart.addHistogramSeries({
    priceFormat: { type: 'volume' },
    priceScaleId: '',
    color: 'rgba(232,230,224,0.18)',
  })
  volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.80, bottom: 0 } })

  const cdata = candles.map(c => ({ time: c.time as never, open: c.open, high: c.high, low: c.low, close: c.close }))
  const vdata = candles.map(c => ({
    time: c.time as never,
    value: Math.exp(Math.random() * 1.5 + 6),
    color: c.close >= c.open ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)',
  }))
  candleSeries.setData(cdata)
  volumeSeries.setData(vdata)
  chart.timeScale().fitContent()

  chartRO = new ResizeObserver(() => {
    if (chart && chartContainer.value) {
      chart.applyOptions({
        width:  chartContainer.value.clientWidth,
        height: chartContainer.value.clientHeight,
      })
    }
  })
  chartRO.observe(chartContainer.value)
}

function chartTick() {
  if (!candleSeries || !volumeSeries || candles.length === 0) return
  const now = Math.floor(Date.now() / 1000)
  const newBar = now - lastBarTime >= 60

  const drift = (0.001005 - midPrice.value) * 0.04
  const noise = (Math.random() - 0.5) * 0.0000028
  const next = Math.max(0.00094, Math.min(0.001065, midPrice.value + drift + noise))
  const prevMid = midPrice.value
  midPrice.value = next

  if (newBar) {
    lastBarTime += 60
    lastBarOpen = candles[candles.length - 1]!.close
    const bar: Candle = {
      time: lastBarTime,
      open: lastBarOpen,
      high: Math.max(lastBarOpen, next),
      low:  Math.min(lastBarOpen, next),
      close: next,
    }
    candles.push(bar)
    candleSeries.update({ time: bar.time as never, open: bar.open, high: bar.high, low: bar.low, close: bar.close })
    const vol = Math.exp(Math.random() * 1.5 + 6)
    volumeSeries.update({
      time: bar.time as never,
      value: vol,
      color: next >= lastBarOpen ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)',
    })
  } else {
    const cur = candles[candles.length - 1]!
    cur.close = next
    cur.high  = Math.max(cur.high, next)
    cur.low   = Math.min(cur.low, next)
    candleSeries.update({ time: cur.time as never, open: cur.open, high: cur.high, low: cur.low, close: cur.close })
  }

  high24h.value = Math.max(high24h.value, next)
  low24h.value  = Math.min(low24h.value, next)

  flashHeader.value = next >= prevMid ? 'up' : 'down'
  setTimeout(() => { flashHeader.value = null }, 700)

  syncPositions()
}

// =====================================================
// Order book
// =====================================================
const asks = ref<BookLevel[]>([])
const bids = ref<BookLevel[]>([])
const flashBook = ref(0)
const midFlash = ref<'up' | 'down' | null>(null)

function generateBook() {
  const mid = midPrice.value
  const halfSpread = mid * 0.0005
  const a: BookLevel[] = []
  const b: BookLevel[] = []
  for (let i = 0; i < 10; i++) {
    const baseQty = 5000 + Math.pow(i, 1.6) * 4500
    const jitter = (Math.random() - 0.5) * 1200
    a.push({ price: mid + halfSpread + i * mid * 0.0008, qty: Math.round(baseQty + jitter), changed: false })
    b.push({ price: mid - halfSpread - i * mid * 0.0008, qty: Math.round(baseQty + jitter), changed: false })
  }
  asks.value = a
  bids.value = b
}

function updateBook() {
  const sides: ('asks' | 'bids')[] = ['asks', 'bids']
  for (const side of sides) {
    const arr = side === 'asks' ? asks.value : bids.value
    const nChanged = 1 + Math.floor(Math.random() * 3)
    for (let i = 0; i < nChanged; i++) {
      const idx = Math.floor(Math.random() * arr.length)
      const delta = (Math.random() - 0.5) * 3000
      arr[idx]!.qty = Math.max(800, Math.round(arr[idx]!.qty + delta))
      arr[idx]!.changed = true
    }
  }
  asks.value = [...asks.value]
  bids.value = [...bids.value]
  flashBook.value++
  midFlash.value = deltaPct.value >= 0 ? 'up' : 'down'
  setTimeout(() => {
    asks.value.forEach(l => { l.changed = false })
    bids.value.forEach(l => { l.changed = false })
    asks.value = [...asks.value]
    bids.value = [...bids.value]
    midFlash.value = null
  }, 700)
}

const maxBookQty = computed(() => {
  const all = [...asks.value.map(l => l.qty), ...bids.value.map(l => l.qty)]
  return Math.max(1, ...all)
})

// Asks shown deepest-at-top → best ask at bottom of asks list
const asksDisplay = computed(() => {
  let run = 0
  const cum = asks.value.map(l => (run += l.qty))
  return asks.value
    .map((l, i) => ({ ...l, cum: cum[i]! }))
    .slice()
    .reverse()
})
const bidsDisplay = computed(() => {
  let run = 0
  return bids.value.map((l) => ({ ...l, cum: (run += l.qty) }))
})

const bestAsk = computed(() => asks.value[0]?.price ?? 0)
const bestBid = computed(() => bids.value[0]?.price ?? 0)
const spread = computed(() => bestAsk.value - bestBid.value)
const spreadPct = computed(() => {
  const m = (bestAsk.value + bestBid.value) / 2
  return m === 0 ? 0 : (spread.value / m) * 100
})

// =====================================================
// Trade tape
// =====================================================
const MAX_TAPE = 40
const tape = ref<TapeRow[]>([])
const tapePaused = ref(false)

function addTrades() {
  const burst = 1 + Math.floor(Math.random() * 3)
  const list = [...tape.value]
  for (let i = 0; i < burst; i++) {
    const dir: 'up' | 'down' = Math.random() < 0.5 ? 'up' : 'down'
    const px = dir === 'up'
      ? midPrice.value + Math.random() * 0.0000015
      : midPrice.value - Math.random() * 0.0000015
    const ln = Math.exp(Math.random() * 2.6 + 4.4)
    const isLarge = Math.random() < 0.05
    const qty = Math.round(isLarge ? ln * 60 : ln)
    const now = new Date()
    const time = now.toLocaleTimeString('en-GB', { hour12: false }) + '.' + String(now.getMilliseconds()).padStart(3, '0')
    list.unshift({ side: dir, px, qty, time, large: isLarge })
    if (list.length > MAX_TAPE) list.pop()
  }
  if (!tapePaused.value) tape.value = list
}

function toggleTapePause() { tapePaused.value = !tapePaused.value }

// =====================================================
// Order entry
// =====================================================
const orderSide = ref<Side>('buy')
const orderType = ref<OrderType>('limit')
const limitPx = ref('0.001004')
const qty = ref('1,000')

function parseNum(s: string): number {
  const cleaned = s.replace(/[^0-9.]/g, '')
  return parseFloat(cleaned) || 0
}
const qtyN = computed(() => parseNum(qty.value))
const pxN  = computed(() => parseNum(limitPx.value))
const notional = computed(() => pxN.value * qtyN.value)
const fee = computed(() => notional.value * 0.01)
const total = computed(() => orderSide.value === 'buy' ? notional.value + fee.value : notional.value - fee.value)
const buyingPowerCredits = 10196841

function setLastPx() {
  limitPx.value = fmtPx(midPrice.value)
}
function bumpPx(dir: 1 | -1) {
  const step = 0.000001
  const next = Math.max(0, pxN.value + dir * step)
  limitPx.value = next.toFixed(6)
}
function bumpQty(dir: 1 | -1) {
  const step = qtyN.value < 100 ? 1 : qtyN.value < 1000 ? 10 : 100
  const next = Math.max(0, qtyN.value + dir * step)
  qty.value = fmtInt(next)
}
function quickFill(pct: number) {
  qty.value = fmtInt(Math.round(buyingPowerCredits * pct))
}

const submitLabel = computed(() => {
  const side = orderSide.value === 'buy' ? 'Buy' : 'Sell'
  const amount = total.value < 10 ? fmtUsd(total.value, 4) : fmtUsd(total.value, 2)
  return `${side} ${fmtInt(qtyN.value)} AI credits — ${amount}`
})

// =====================================================
// Positions
// =====================================================
function pnl(p: Position) {
  const dir = p.side === 'long' ? 1 : -1
  return (p.mark - p.entry) * p.size * dir
}
function pnlPct(p: Position) {
  const dir = p.side === 'long' ? 1 : -1
  return ((p.mark - p.entry) / p.entry) * 100 * dir
}
function posEntryFmt(p: Position) {
  if (p.market === 'H100-SPOT') return p.entry.toFixed(2)
  const dp = p.market === 'TEXT-SPOT' || p.market === 'IMAGE-SPOT' ? 5 : 6
  return fmtPx(p.entry, dp)
}
function posMarkFmt(p: Position) {
  if (p.market === 'H100-SPOT') return p.mark.toFixed(2)
  const dp = p.market === 'TEXT-SPOT' || p.market === 'IMAGE-SPOT' ? 5 : 6
  return fmtPx(p.mark, dp)
}

function syncPositions() {
  positions[0]!.mark = midPrice.value
  positions[1]!.mark += (0.00120 - positions[1]!.mark) * 0.05 + (Math.random() - 0.5) * 0.000004
  positions[2]!.mark += (0.00798 - positions[2]!.mark) * 0.05 + (Math.random() - 0.5) * 0.00003
  positions[3]!.mark += (2.99 - positions[3]!.mark) * 0.05 + (Math.random() - 0.5) * 0.005
}

const totalPnl = computed(() => positions.reduce((s, p) => s + pnl(p), 0))
const totalValue = computed(() => 10247.83 + (totalPnl.value - 23.41))

// =====================================================
// Live loops
// =====================================================
let chartInterval: ReturnType<typeof setInterval> | null = null
let bookInterval:  ReturnType<typeof setInterval> | null = null
let tapeInterval:  ReturnType<typeof setInterval> | null = null

onMounted(() => {
  seedChart()
  initChart()
  generateBook()
  for (let i = 0; i < MAX_TAPE; i++) addTrades()
  chartInterval = setInterval(chartTick, 3000)
  bookInterval  = setInterval(updateBook, 1500)
  tapeInterval  = setInterval(addTrades, 5000)
})
onBeforeUnmount(() => {
  if (chartInterval) clearInterval(chartInterval)
  if (bookInterval)  clearInterval(bookInterval)
  if (tapeInterval)  clearInterval(tapeInterval)
  chartRO?.disconnect()
  chart?.remove()
})
</script>

<template>
  <div class="trade-page">
    <div class="main-grid">
      <!-- ============ CHART ============ -->
      <section class="panel chart-panel">
        <header class="chart-head">
          <span class="sym">EAI-IDX</span>
          <span class="px" :class="headerPxClass">{{ fmtPx(midPrice) }}</span>
          <span class="delta" :class="arrowUp ? 'pos' : 'neg'">
            {{ arrowUp ? '▲' : '▼' }} {{ Math.abs(deltaPct).toFixed(2) }}%
          </span>
          <span class="stats">
            <span class="k">Vol</span> $1.2M
            <span class="k stat-spacer">H</span> {{ fmtPx(high24h) }}
            <span class="k stat-spacer">L</span> {{ fmtPx(low24h) }}
          </span>
          <div class="right">
            <div class="seg">
              <button
                v-for="t in TIMEFRAMES"
                :key="t"
                :class="{ active: tf === t }"
                type="button"
                @click="tf = t"
              >{{ t }}</button>
            </div>
            <button class="indicators-btn" type="button">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M8 3v10M3 8h10" />
              </svg>
              Indicators
            </button>
          </div>
        </header>
        <div class="panel-body">
          <div ref="chartContainer" class="chart-container" />
        </div>
      </section>

      <!-- ============ ORDER BOOK ============ -->
      <section class="panel book-panel">
        <header class="panel-head">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
            <path d="M3 4l5 3 5-3M3 8l5 3 5-3M3 12l5 3 5-3" />
          </svg>
          <span class="lbl">Order Book</span>
          <span class="lbl-badge">10 LVL</span>
        </header>
        <div class="panel-body book-body">
          <div class="ob-cols">
            <span>Price (USD)</span>
            <span>Size</span>
            <span>Total</span>
          </div>
          <div class="ob-rows">
            <div class="ob-side asks">
              <div
                v-for="(l, i) in asksDisplay"
                :key="'a-' + i"
                class="ob-row ask"
                :class="{ flash: l.changed }"
              >
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtInt(l.qty) }}</span>
                <span class="ob-tot">{{ fmtInt(l.cum) }}</span>
              </div>
            </div>
            <div class="ob-mid">
              <div
                class="midpx"
                :class="midFlash === 'up' ? 'flash-up' : midFlash === 'down' ? 'flash-down' : ''"
              >
                {{ fmtPx(midPrice) }}
                <span class="arrow" :class="arrowUp ? 'pos' : 'neg'">{{ arrowUp ? '▲' : '▼' }}</span>
              </div>
              <div class="spread">
                Spread {{ fmtPx(spread) }} / {{ spreadPct.toFixed(2) }}%
              </div>
            </div>
            <div class="ob-side bids">
              <div
                v-for="(l, i) in bidsDisplay"
                :key="'b-' + i"
                class="ob-row bid"
                :class="{ flash: l.changed }"
              >
                <div class="bar" :style="{ width: (l.qty / maxBookQty * 100) + '%' }" />
                <span class="ob-px">{{ fmtPx(l.price) }}</span>
                <span class="ob-qty">{{ fmtInt(l.qty) }}</span>
                <span class="ob-tot">{{ fmtInt(l.cum) }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ ORDER ENTRY ============ -->
      <section class="panel entry-panel">
        <div class="side-tabs">
          <button
            class="side-tab buy"
            :class="{ active: orderSide === 'buy' }"
            type="button"
            @click="orderSide = 'buy'"
          >Buy</button>
          <button
            class="side-tab sell"
            :class="{ active: orderSide === 'sell' }"
            type="button"
            @click="orderSide = 'sell'"
          >Sell</button>
        </div>
        <div class="entry-body">
          <div class="entry-form">
            <div class="field">
              <div class="field-row">
                <span class="lbl">Order type</span>
              </div>
              <div class="seg seg-sm">
                <button
                  type="button"
                  :class="{ active: orderType === 'market' }"
                  @click="orderType = 'market'"
                >Market</button>
                <button
                  type="button"
                  :class="{ active: orderType === 'limit' }"
                  @click="orderType = 'limit'"
                >Limit</button>
                <button type="button" disabled class="disabled-btn">Stop</button>
              </div>
            </div>

            <div class="field">
              <div class="field-row">
                <span class="lbl">Limit price</span>
                <button type="button" class="last-px" @click="setLastPx">Last {{ fmtPx(midPrice) }}</button>
              </div>
              <div class="input-row">
                <input v-model="limitPx" type="text" />
                <div class="ticks">
                  <button type="button" @click="bumpPx(1)">▲</button>
                  <button type="button" @click="bumpPx(-1)">▼</button>
                </div>
              </div>
            </div>

            <div class="field">
              <div class="field-row">
                <span class="lbl">Quantity (credits)</span>
              </div>
              <div class="input-row">
                <input v-model="qty" type="text" />
                <div class="ticks">
                  <button type="button" @click="bumpQty(1)">▲</button>
                  <button type="button" @click="bumpQty(-1)">▼</button>
                </div>
              </div>
              <div class="quick-fill">
                <button type="button" @click="quickFill(0.25)">25%</button>
                <button type="button" @click="quickFill(0.5)">50%</button>
                <button type="button" @click="quickFill(0.75)">75%</button>
                <button type="button" @click="quickFill(1)">100%</button>
              </div>
            </div>
          </div>

          <div class="entry-form summary-side">
            <div class="summary">
              <div class="sum-row">
                <span class="k">PRICE × QTY</span>
                <span class="v">{{ fmtUsd(pxN, 6) }} × {{ fmtInt(qtyN) }} = {{ fmtUsd(notional, notional < 10 ? 4 : 2) }}</span>
              </div>
              <div class="sum-row">
                <span class="k">TAKER FEE · 1.00%</span>
                <span class="v">+ {{ fmtUsd(fee, fee < 1 ? 4 : 2) }}</span>
              </div>
              <div class="sum-row">
                <span class="k">FUNDING (EST)</span>
                <span class="v muted">—</span>
              </div>
              <div class="sum-row total">
                <span class="k total-k">TOTAL</span>
                <span class="v">{{ fmtUsd(total, total < 10 ? 4 : 2) }}</span>
              </div>
            </div>

            <div class="balance">
              <span class="k">AVAILABLE</span>
              <span class="v">$10,247.83 USD</span>
              <span class="sep">·</span>
              <span class="v">{{ fmtInt(buyingPowerCredits) }}</span>
              <span class="k">credits buying power</span>
            </div>

            <button class="submit-btn" :class="orderSide" type="button">
              {{ submitLabel }}
            </button>
          </div>
        </div>
      </section>

      <!-- ============ TAPE ============ -->
      <section
        class="panel tape-panel"
        @mouseenter="tapePaused = true"
        @mouseleave="tapePaused = false"
      >
        <header class="panel-head">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
            <path d="M2 8h3l2-4 2 8 2-5 2 3h1" />
          </svg>
          <span class="lbl">Trades</span>
          <button class="tape-toggle" type="button" @click="toggleTapePause">
            <span class="dot" :class="{ warn: tapePaused }" />
            {{ tapePaused ? 'PAUSED' : 'AUTO-SCROLL' }}
          </button>
        </header>
        <div class="panel-body">
          <div class="tape">
            <div class="tape-cols">
              <span>Time</span>
              <span>Price</span>
              <span>Qty</span>
              <span />
            </div>
            <div class="tape-list">
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
      </section>
    </div>

    <!-- ============ POSITIONS FOOTER ============ -->
    <section class="positions">
      <div class="pos-head">
        <span class="pos-title">Positions</span>
        <div class="pos-tabs">
          <button
            class="pos-tab"
            :class="{ active: posTab === 'open' }"
            type="button"
            @click="posTab = 'open'"
          >Open<span class="count">· {{ positions.length }}</span></button>
          <button
            class="pos-tab"
            :class="{ active: posTab === 'closed' }"
            type="button"
            @click="posTab = 'closed'"
          >Closed</button>
          <button
            class="pos-tab"
            :class="{ active: posTab === 'all' }"
            type="button"
            @click="posTab = 'all'"
          >All</button>
        </div>
        <div class="pos-stats">
          <div class="stat">
            <span class="k">TODAY P&amp;L</span>
            <span class="v" :class="totalPnl >= 0 ? 'pos' : 'neg'">
              {{ totalPnl >= 0 ? '+' : '−' }}{{ fmtUsd(Math.abs(totalPnl)) }}
              {{ totalPnl >= 0 ? '▲' : '▼' }} {{ Math.abs(totalPnl / 10247.83 * 100).toFixed(2) }}%
            </span>
          </div>
          <div class="stat">
            <span class="k">TOTAL VALUE</span>
            <span class="v">{{ fmtUsd(totalValue) }}</span>
          </div>
        </div>
      </div>
      <div class="pos-table-wrap">
        <table class="pos-table">
          <thead>
            <tr>
              <th class="left">Market</th>
              <th class="left">Side</th>
              <th>Size</th>
              <th>Avg Entry</th>
              <th>Mark</th>
              <th>Notional</th>
              <th>P&amp;L $</th>
              <th>P&amp;L %</th>
              <th class="right-th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in positions" :key="p.market">
              <td class="left">{{ p.market }}</td>
              <td class="left">
                <span class="side-pill" :class="p.side">{{ p.side.toUpperCase() }}</span>
              </td>
              <td>{{ fmtInt(p.size) }}</td>
              <td>{{ posEntryFmt(p) }}</td>
              <td>{{ posMarkFmt(p) }}</td>
              <td>{{ fmtUsd(p.mark * p.size, (p.mark * p.size) < 100 ? 4 : 2) }}</td>
              <td :class="pnl(p) >= 0 ? 'pos' : 'neg'">
                {{ pnl(p) >= 0 ? '+' : '−' }}{{ fmtUsd(Math.abs(pnl(p))) }}
              </td>
              <td :class="pnl(p) >= 0 ? 'pos' : 'neg'">
                {{ pnl(p) >= 0 ? '▲' : '▼' }} {{ Math.abs(pnlPct(p)).toFixed(2) }}%
              </td>
              <td class="right-td"><button type="button" class="close-btn">CLOSE</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.trade-page {
  --text-4: rgba(255, 255, 255, 0.20);
  --pos-bar: rgba(25, 195, 125, 0.16);
  --neg-bar: rgba(239, 68, 68, 0.16);

  display: grid;
  grid-template-rows: 1fr 200px;
  height: 100%;
  min-height: 800px;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.5;
  font-feature-settings: 'tnum' on, 'ss01' on;
  overflow: hidden;
}

.pos { color: var(--pos); }
.neg { color: var(--neg); }
.muted { color: var(--text-2); }

/* ============================================
   Main grid: chart/book / entry/tape
   ============================================ */
.main-grid {
  display: grid;
  grid-template-columns: 1fr 340px;
  grid-template-rows: 1.55fr 1fr;
  grid-template-areas:
    "chart book"
    "entry tape";
  overflow: hidden;
  min-height: 0;
}
.panel {
  border-right: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
.book-panel,
.tape-panel { border-right: 0; }

.panel-head {
  height: 36px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 12px;
}
.panel-body {
  flex: 1;
  overflow: hidden;
  position: relative;
  min-height: 0;
}
.head-ic { width: 14px; height: 14px; color: var(--text-2); }
.lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
}
.lbl-badge {
  background: var(--elevated);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  margin-left: auto;
}

/* ============================================
   Chart panel
   ============================================ */
.chart-panel { grid-area: chart; }
.chart-head {
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--border);
  padding: 0 14px;
  gap: 14px;
  flex-wrap: nowrap;
  white-space: nowrap;
}
.chart-head .sym {
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 15px;
  letter-spacing: -0.01em;
  flex-shrink: 0;
}
.chart-head .px {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  font-weight: 600;
  transition: color 600ms;
}
.chart-head .px.flash-up   { color: var(--pos); }
.chart-head .px.flash-down { color: var(--neg); }
.chart-head .delta {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.02em;
}
.chart-head .delta.pos { background: var(--pos-soft); color: var(--pos); }
.chart-head .delta.neg { background: var(--neg-soft); color: var(--neg); }
.chart-head .stats {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
}
.chart-head .stats .k { color: var(--text-3); }
.chart-head .stats .stat-spacer { margin-left: 10px; }
.chart-head .right { margin-left: auto; display: flex; align-items: center; gap: 8px; }
.chart-container { position: absolute; inset: 0; }

/* Segmented timeframe */
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
  height: 26px;
  padding: 0 10px;
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
.indicators-btn {
  height: 26px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-2);
  padding: 0 10px;
  font-family: var(--font-mono);
  font-size: 11px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.indicators-btn:hover { background: var(--hover); color: var(--text); }
.indicators-btn svg { width: 12px; height: 12px; stroke: currentColor; fill: none; stroke-width: 1.6; }

/* ============================================
   Order book
   ============================================ */
.book-panel { grid-area: book; }
.book-body {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  display: flex;
  flex-direction: column;
}
.ob-cols {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 8px 14px;
  color: var(--text-3);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
}
.ob-cols span { text-align: right; }
.ob-cols span:first-child { text-align: left; }

.ob-rows {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.ob-side {
  display: flex;
  flex-direction: column;
  flex: 1;
  justify-content: flex-end;
  overflow: hidden;
}
.ob-side.bids { justify-content: flex-start; }
.ob-row {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 0 14px;
  height: 18px;
  align-items: center;
  position: relative;
  cursor: pointer;
  transition: background 100ms;
}
.ob-row:hover { background: rgba(255, 255, 255, 0.03); }
.ob-row .bar {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  transition: width 250ms ease-out, opacity 250ms;
}
.ob-row.ask .bar { background: var(--neg-bar); }
.ob-row.bid .bar { background: var(--pos-bar); }
.ob-row > * { position: relative; text-align: right; }
.ob-row .ob-px { text-align: left; }
.ob-row.ask .ob-px { color: var(--neg); }
.ob-row.bid .ob-px { color: var(--pos); }
.ob-row .ob-qty { color: var(--text); }
.ob-row .ob-tot { color: var(--text-3); }
.ob-row.flash { background: rgba(255, 255, 255, 0.08); }

.ob-mid {
  height: 48px;
  flex-shrink: 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  background: var(--elevated);
}
.ob-mid .midpx {
  font-family: var(--font-mono);
  font-size: 15px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  transition: color 600ms;
}
.ob-mid .midpx.flash-up { color: var(--pos); }
.ob-mid .midpx.flash-down { color: var(--neg); }
.ob-mid .arrow { font-size: 11px; vertical-align: 2px; margin-left: 6px; }
.ob-mid .arrow.pos { color: var(--pos); }
.ob-mid .arrow.neg { color: var(--neg); }
.ob-mid .spread {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.04em;
}

/* ============================================
   Trade tape
   ============================================ */
.tape-panel { grid-area: tape; }
.tape-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--elevated);
  border: 1px solid var(--border);
  height: 22px;
  padding: 0 8px;
  border-radius: var(--radius-sm);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  cursor: pointer;
  letter-spacing: 0.06em;
  margin-left: auto;
}
.tape-toggle:hover { background: var(--hover); color: var(--text); }
.tape-toggle .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--pos);
}
.tape-toggle .dot.warn { background: var(--warn); }

.tape {
  height: 100%;
  overflow: hidden;
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

.tape-list {
  overflow-y: hidden;
  height: calc(100% - 32px);
}
.tape-row {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 16px;
  padding: 0 14px;
  height: 18px;
  align-items: center;
  color: var(--text);
}
.tape-row.up .t-px,
.tape-row.up .t-side { color: var(--pos); }
.tape-row.down .t-px,
.tape-row.down .t-side { color: var(--neg); }
.tape-row .t-time { color: var(--text-3); }
.tape-row .t-qty { text-align: right; }
.tape-row .t-px { text-align: right; }
.tape-row .t-side {
  text-align: center;
  font-size: 10px;
}
.tape-row.large { background: rgba(255, 255, 255, 0.025); }
.tape-row.enter { animation: tapeIn 200ms ease-out; }
@keyframes tapeIn {
  from { transform: translateY(-8px); opacity: 0; }
  to   { transform: translateY(0); opacity: 1; }
}

/* ============================================
   Order entry
   ============================================ */
.entry-panel { grid-area: entry; }
.side-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.side-tab {
  height: 40px;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  letter-spacing: -0.005em;
  border-bottom: 2px solid transparent;
  transition: color 120ms, border-color 120ms, background 120ms;
}
.side-tab:hover { color: var(--text); background: rgba(255, 255, 255, 0.02); }
.side-tab.active.buy {
  color: var(--pos);
  border-bottom-color: var(--pos);
  background: var(--pos-soft);
}
.side-tab.active.sell {
  color: var(--neg);
  border-bottom-color: var(--neg);
  background: var(--neg-soft);
}

.entry-body {
  padding: 16px 20px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  flex: 1;
  min-height: 0;
}
.entry-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}
.entry-form.summary-side { justify-content: space-between; }
.seg-sm { height: 24px; }
.seg-sm button { height: 22px; font-size: 10px; padding: 0 8px; }
.seg-sm button.disabled-btn {
  opacity: 0.4;
  cursor: not-allowed;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.field-row .last-px {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-2);
  background: var(--elevated);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  border: 1px solid var(--border);
}
.field-row .last-px:hover { background: var(--hover); color: var(--text); }

.input-row {
  display: grid;
  grid-template-columns: 1fr 28px;
  height: 34px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  overflow: hidden;
}
.input-row input {
  background: transparent;
  border: none;
  color: var(--text);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 14px;
  font-weight: 500;
  padding: 0 12px;
  outline: none;
  text-align: right;
}
.input-row input:focus {
  outline: 1px solid var(--accent);
  outline-offset: -1px;
}
.input-row .ticks {
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--border);
}
.input-row .ticks button {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid var(--border);
  transition: background 120ms, color 120ms;
}
.input-row .ticks button:last-child { border-bottom: none; }
.input-row .ticks button:hover { background: var(--hover); color: var(--text); }

.quick-fill {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 4px;
  margin-top: 4px;
}
.quick-fill button {
  height: 22px;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 120ms, color 120ms;
}
.quick-fill button:hover { background: var(--hover); color: var(--text); }

.summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: auto;
}
.sum-row {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
.sum-row .k { color: var(--text-3); letter-spacing: 0.04em; }
.sum-row .v { color: var(--text); }
.sum-row.total {
  padding-top: 8px;
  margin-top: 4px;
  border-top: 1px solid var(--border);
  font-size: 13px;
  font-weight: 600;
}
.sum-row .total-k { color: var(--text); }

.balance {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-2);
  letter-spacing: 0.04em;
  padding: 6px 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  text-align: center;
}
.balance .v { color: var(--text); }
.balance .k { color: var(--text-2); }
.balance .sep { margin: 0 8px; color: var(--text-3); }

.submit-btn {
  height: 44px;
  width: 100%;
  border: none;
  border-radius: var(--radius-sm);
  color: #fff;
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 14px;
  letter-spacing: -0.005em;
  cursor: pointer;
  transition: filter 120ms;
}
.submit-btn:hover { filter: brightness(1.08); }
.submit-btn.buy  { background: var(--pos); }
.submit-btn.sell { background: var(--neg); }

/* ============================================
   Positions footer
   ============================================ */
.positions {
  border-top: 1px solid var(--border);
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
.pos-head {
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 16px;
  border-bottom: 1px solid var(--border);
}
.pos-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 13px;
}
.pos-tabs { display: inline-flex; gap: 0; }
.pos-tab {
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: 12px;
  height: 26px;
  padding: 0 10px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  font-family: var(--font-sans);
  transition: color 120ms, border-color 120ms;
}
.pos-tab:hover { color: var(--text); }
.pos-tab.active { color: var(--brand); border-bottom-color: var(--brand); }
.pos-tab .count {
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10px;
  margin-left: 4px;
}
.pos-stats {
  margin-left: auto;
  display: flex;
  gap: 24px;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.pos-stats .stat {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
}
.pos-stats .k {
  color: var(--text-3);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.pos-stats .v { color: var(--text); }
.pos-stats .v.pos { color: var(--pos); }
.pos-stats .v.neg { color: var(--neg); }

.pos-table-wrap {
  flex: 1;
  overflow: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.pos-table {
  width: 100%;
  border-collapse: collapse;
}
.pos-table th, .pos-table td {
  padding: 6px 14px;
  text-align: right;
  border-bottom: 1px solid var(--border);
  font-weight: 500;
}
.pos-table th.left, .pos-table td.left { text-align: left; }
.pos-table th.right-th, .pos-table td.right-td { text-align: right; }
.pos-table thead th {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  padding: 5px 14px;
  border-bottom-color: var(--border-strong);
}
.pos-table tbody tr {
  transition: background 120ms;
  cursor: pointer;
}
.pos-table tbody tr:hover { background: rgba(255, 255, 255, 0.025); }
.side-pill {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  letter-spacing: 0.06em;
  font-family: var(--font-mono);
  font-weight: 500;
}
.side-pill.long  { background: var(--pos-soft); color: var(--pos); }
.side-pill.short { background: var(--neg-soft); color: var(--neg); }
.close-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  padding: 3px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  letter-spacing: 0.04em;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.close-btn:hover {
  background: var(--neg-soft);
  color: var(--neg);
  border-color: var(--neg);
}

/* ── Mobile: unfreeze the fixed 2×2 trading board into a single scrolling column. Basic pass — the
   board is a paused-exchange placeholder; a purpose-built mobile trade view is a later task. ── */
@media (max-width: 768px) {
  .trade-page { display: block; height: auto; min-height: 0; overflow: visible; }
  .main-grid {
    grid-template-columns: 1fr;
    grid-template-rows: auto;
    grid-template-areas: "chart" "entry" "book" "tape";
    overflow: visible;
  }
  .panel { border-right: 0; }
  .chart-panel { min-height: 300px; }
  .entry-panel, .book-panel, .tape-panel { min-height: 340px; }
}
</style>
