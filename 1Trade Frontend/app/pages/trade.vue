<script setup lang="ts">
/**
 * /trade — the paper exchange (trading.yaml v1.1). Dense Bloomberg-style grid inside the `app` layout.
 *
 * Layout (fills the main area, no scroll):
 *   ┌───────────────────────────────┬──────────────────┐
 *   │  chart       (1.55fr)         │  order book      │
 *   ├───────────────────────────────┼──────────────────┤
 *   │  order entry                  │  trade tape      │
 *   ├───────────────────────────────┴──────────────────┤
 *   │  account: positions · orders · fills             │
 *   └──────────────────────────────────────────────────┘
 *
 * Everything on screen is the engine's paper venue: the book is the real paper book (kept two-sided by
 * the paper liquidity account), orders reserve paper cash or credits in the ledger and fill for real,
 * and positions, orders and fills are the caller's own. Bars with no trades are the reference walk and
 * are drawn muted — the engine labels them `source: reference`. Real money is paused (licence-gated):
 * a real-money account's order comes back 503 EXCHANGE_PAUSED and is shown as such.
 *
 * `?product=H100-SPOT` selects the product (default EAI-IDX).
 */
import { createChart, ColorType, CrosshairMode, type ISeriesApi } from 'lightweight-charts'

definePageMeta({ layout: 'app', middleware: 'auth' })

// =====================================================
// Types + state
// =====================================================
type TF = '1m' | '5m' | '15m' | '1h' | '4h' | '1d'
type Side = 'buy' | 'sell'
type OrderType = 'market' | 'limit'

interface BookLevel { price: number; qty: number; changed: boolean }
interface Product {
  product_id: string; name: string; credit_type: string; tick_size: string; quote_precision: number; tradeable: boolean
}
interface Summary { last: string; open_24h: string; high_24h: string; low_24h: string; change_pct_24h: string; volume_24h: string }
interface PositionApi {
  product_id: string; side: string; net_quantity: string; avg_entry_price: string; mark_price: string
  notional: string; unrealized_pnl: string; unrealized_pnl_pct: string; realized_pnl_total: string
}
interface OrderRow {
  order_id: string; product_id: string; side: string; order_type: string; quantity: string
  filled_quantity: string; limit_price: string | null; avg_fill_price: string | null; state: string
  reason: string | null; created_at: string
}
interface FillRow {
  fill_id: string; product_id: string; side: string; price: string; quantity: string
  notional: string; fee: string; liquidity: string; executed_at: string
}
interface TapeRow { side: 'up' | 'down'; px: number; qty: number; time: string; large: boolean }

const TIMEFRAMES: TF[] = ['1m', '5m', '15m', '1h', '4h', '1d']
const route = useRoute()
const PRODUCT_ID = String(route.query.product || 'EAI-IDX').toUpperCase()
useHead({ title: `Trade — ${PRODUCT_ID} · 1Trade` })

const product = ref<Product | null>(null)
const summary = ref<Summary | null>(null)
const tf = ref<TF>('5m')
const midPrice = ref(0)
const flashHeader = ref<'up' | 'down' | null>(null)

/** Display precision and tick for this product (from the catalog). */
const dp = computed(() => product.value?.quote_precision ?? 6)
const tick = computed(() => Number(product.value?.tick_size ?? '0.000001'))

/** loadProduct reads the product and its 24h summary (last, range, volume) from the engine. */
async function loadProduct() {
  try {
    const r = await $fetch<{ product: Product; summary: Summary }>(`/api/trading/products/${PRODUCT_ID}`)
    product.value = r.product
    summary.value = r.summary
  } catch { /* header keeps its last values */ }
}

// =====================================================
// Formatters
// =====================================================
function fmtPx(n: number, places = dp.value) {
  return n.toLocaleString('en-US', { minimumFractionDigits: places, maximumFractionDigits: places })
}
function fmtUsd(n: number, places = 2) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: places, maximumFractionDigits: places })
}
function fmtQty(n: number) {
  return n.toLocaleString('en-US', { maximumFractionDigits: n < 100 ? 4 : 0 })
}
function fmtTime(iso: string) {
  return new Date(iso).toLocaleTimeString('en-GB', { hour12: false })
}

// =====================================================
// Header (24h summary)
// =====================================================
const deltaPct = computed(() => Number(summary.value?.change_pct_24h ?? 0))
const arrowUp = computed(() => deltaPct.value >= 0)
const high24h = computed(() => Number(summary.value?.high_24h ?? 0))
const low24h = computed(() => Number(summary.value?.low_24h ?? 0))
const volume24h = computed(() => Number(summary.value?.volume_24h ?? 0))
const headerPxClass = computed(() => flashHeader.value === 'up' ? 'flash-up' : flashHeader.value === 'down' ? 'flash-down' : '')

// =====================================================
// Chart
// =====================================================
const chartContainer = ref<HTMLDivElement | null>(null)
let chart: ReturnType<typeof createChart> | null = null
let candleSeries: ISeriesApi<'Candlestick'> | null = null
let volumeSeries: ISeriesApi<'Histogram'> | null = null
let chartRO: ResizeObserver | null = null

interface Candle { time: number; open: number; high: number; low: number; close: number; volume: number; traded: boolean }
interface CandleApi { time: number; open: string; high: string; low: string; close: string; volume: string; source?: string }
const candles: Candle[] = []
const chartErr = ref('')
const tradedBars = computed(() => candles.filter(c => c.traded).length)

/** loadCandles pulls the OHLCV series for the selected timeframe. */
async function loadCandles(): Promise<boolean> {
  try {
    const r = await $fetch<{ candles: CandleApi[] }>(
      `/api/trading/products/${PRODUCT_ID}/candles`, { query: { interval: tf.value, limit: 200 } },
    )
    const rows = (r.candles ?? []).map((c) => ({
      time: c.time, open: Number(c.open), high: Number(c.high), low: Number(c.low), close: Number(c.close),
      volume: Number(c.volume), traded: c.source !== 'reference',
    }))
    if (!rows.length) return false
    candles.splice(0, candles.length, ...rows)
    chartErr.value = ''
    return true
  } catch {
    chartErr.value = 'Chart data unavailable — retrying…'
    return false
  }
}

/** applyCandles draws the series: traded bars in colour, reference bars (no trades) muted. */
function applyCandles() {
  if (!candleSeries || !volumeSeries || !candles.length) return
  const muted = 'rgba(168,161,150,0.35)'
  candleSeries.setData(candles.map((c) => {
    const bar = { time: c.time as never, open: c.open, high: c.high, low: c.low, close: c.close }
    return c.traded ? bar : { ...bar, color: muted, wickColor: muted, borderColor: muted }
  }))
  volumeSeries.setData(candles.map((c) => ({
    time: c.time as never, value: c.volume,
    color: c.close >= c.open ? 'rgba(25,195,125,0.30)' : 'rgba(239,68,68,0.30)',
  })))
}

/** initChart builds the chart once the series has loaded. */
function initChart() {
  if (!chartContainer.value) return
  chart = createChart(chartContainer.value, {
    layout: { background: { type: ColorType.Solid, color: '#0A0A0A' }, textColor: '#A8A196', fontFamily: 'JetBrains Mono, ui-monospace, monospace', fontSize: 10 },
    grid: { vertLines: { color: 'rgba(255,255,255,0.03)' }, horzLines: { color: 'rgba(255,255,255,0.03)' } },
    rightPriceScale: { borderColor: 'rgba(255,255,255,0.08)', scaleMargins: { top: 0.08, bottom: 0.28 } },
    timeScale: { borderColor: 'rgba(255,255,255,0.08)', timeVisible: true, secondsVisible: false },
    crosshair: {
      mode: CrosshairMode.Normal,
      vertLine: { color: 'rgba(232,230,224,0.30)', width: 1, style: 0 },
      horzLine: { color: 'rgba(232,230,224,0.30)', width: 1, style: 0 },
    },
    handleScroll: true,
    handleScale: true,
  })
  candleSeries = chart.addCandlestickSeries({
    upColor: '#19C37D', downColor: '#EF4444', borderUpColor: '#19C37D', borderDownColor: '#EF4444',
    wickUpColor: '#19C37D', wickDownColor: '#EF4444',
    priceFormat: { type: 'price', precision: dp.value, minMove: tick.value },
  })
  volumeSeries = chart.addHistogramSeries({ priceFormat: { type: 'volume' }, priceScaleId: '', color: 'rgba(232,230,224,0.18)' })
  volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.80, bottom: 0 } })
  applyCandles()
  chart.timeScale().fitContent()
  chartRO = new ResizeObserver(() => {
    if (chart && chartContainer.value) {
      chart.applyOptions({ width: chartContainer.value.clientWidth, height: chartContainer.value.clientHeight })
    }
  })
  chartRO.observe(chartContainer.value)
}

/** chartTick re-reads the series. */
async function chartTick() {
  if (await loadCandles()) applyCandles()
}

watch(tf, async () => {
  if (await loadCandles()) {
    applyCandles()
    chart?.timeScale().fitContent()
  }
})

// =====================================================
// Order book (the real paper book)
// =====================================================
const asks = ref<BookLevel[]>([])
const bids = ref<BookLevel[]>([])
const bookErr = ref('')
const tapeErr = ref('')
const midFlash = ref<'up' | 'down' | null>(null)

interface BookApiLevel { price: string; size: string; cumulative: string }

/** loadBook reads the paper book's depth; a level flashes only when its size actually changed. */
async function loadBook() {
  try {
    const r = await $fetch<{ asks: BookApiLevel[]; bids: BookApiLevel[] }>(
      `/api/trading/products/${PRODUCT_ID}/orderbook`, { query: { depth: 10 } },
    )
    const prevA = new Map(asks.value.map((l) => [l.price, l.qty]))
    const prevB = new Map(bids.value.map((l) => [l.price, l.qty]))
    const map = (levels: BookApiLevel[], prev: Map<number, number>): BookLevel[] =>
      (levels ?? []).map((l) => {
        const price = Number(l.price)
        const qty = Number(l.size)
        const before = prev.get(price)
        return { price, qty, changed: before !== undefined && before !== qty }
      })
    asks.value = map(r.asks, prevA)
    bids.value = map(r.bids, prevB)
    const prevMid = midPrice.value
    if (asks.value.length && bids.value.length) midPrice.value = (asks.value[0]!.price + bids.value[0]!.price) / 2
    else if (summary.value) midPrice.value = Number(summary.value.last)
    if (prevMid && midPrice.value !== prevMid) {
      midFlash.value = flashHeader.value = midPrice.value > prevMid ? 'up' : 'down'
    }
    bookErr.value = ''
    setTimeout(() => {
      asks.value = asks.value.map((l) => ({ ...l, changed: false }))
      bids.value = bids.value.map((l) => ({ ...l, changed: false }))
      midFlash.value = flashHeader.value = null
    }, 700)
  } catch {
    bookErr.value = 'Book unavailable — retrying…'
  }
}

const maxBookQty = computed(() => Math.max(1, ...asks.value.map(l => l.qty), ...bids.value.map(l => l.qty)))
// Asks shown deepest-at-top → best ask at the bottom of the asks list.
const asksDisplay = computed(() => {
  let run = 0
  const cum = asks.value.map(l => (run += l.qty))
  return asks.value.map((l, i) => ({ ...l, cum: cum[i]! })).slice().reverse()
})
const bidsDisplay = computed(() => {
  let run = 0
  return bids.value.map((l) => ({ ...l, cum: (run += l.qty) }))
})
const bestAsk = computed(() => asks.value[0]?.price ?? 0)
const bestBid = computed(() => bids.value[0]?.price ?? 0)
const spread = computed(() => (bestAsk.value && bestBid.value) ? bestAsk.value - bestBid.value : 0)
const spreadPct = computed(() => {
  const m = (bestAsk.value + bestBid.value) / 2
  return m === 0 || !spread.value ? 0 : (spread.value / m) * 100
})

// =====================================================
// Trade tape (real paper prints)
// =====================================================
const MAX_TAPE = 40
const tape = ref<TapeRow[]>([])
const tapePaused = ref(false)

interface TradeApi { price: string; quantity: string; aggressor_side: string; executed_at: string; block: boolean }

/** loadTape reads recent prints; direction is the engine's aggressor side. */
async function loadTape() {
  if (tapePaused.value) return
  try {
    const r = await $fetch<{ trades: TradeApi[] }>(`/api/trading/products/${PRODUCT_ID}/trades`, { query: { limit: MAX_TAPE } })
    tape.value = (r.trades ?? []).map((t) => ({
      side: t.aggressor_side === 'buy' ? 'up' : 'down',
      px: Number(t.price), qty: Number(t.quantity), time: fmtTime(t.executed_at), large: !!t.block,
    } as TapeRow))
    tapeErr.value = ''
  } catch {
    tapeErr.value = 'Tape unavailable — retrying…'
  }
}

/** toggleTapePause freezes the tape so a row can be read. */
function toggleTapePause() { tapePaused.value = !tapePaused.value }

// =====================================================
// Account: paper cash, credits, positions, orders, fills
// =====================================================
const cash = ref(0)        // available paper USD
const holdings = ref(0)    // available paper credits of this product's type
const positions = ref<PositionApi[]>([])
const orders = ref<OrderRow[]>([])
const fills = ref<FillRow[]>([])
const posTab = ref<'positions' | 'orders' | 'fills'>('positions')
const positionsErr = ref('')

/** loadBalances reads the caller's paper cash and credits from the ledger (available = balance − locked). */
async function loadBalances() {
  try {
    const [c, b] = await Promise.all([
      $fetch<{ balances: { balance: string; locked_amount: string }[] }>('/api/wallet/cash-balances'),
      $fetch<{ balances: { credit_type: string; balance: string; locked_amount?: string; is_paper: boolean }[] }>('/api/wallet/balances'),
    ])
    cash.value = (c.balances ?? []).reduce((s, x) => s + Number(x.balance) - Number(x.locked_amount || 0), 0)
    const ct = product.value?.credit_type
    holdings.value = (b.balances ?? []).filter(x => x.is_paper && x.credit_type === ct)
      .reduce((s, x) => s + Number(x.balance) - Number(x.locked_amount || 0), 0)
  } catch { /* keep the last good values */ }
}

/** loadAccount refreshes positions, orders and fills. */
async function loadAccount() {
  try {
    const [p, o, f] = await Promise.all([
      $fetch<{ positions: PositionApi[] }>('/api/trading/positions'),
      $fetch<{ orders: OrderRow[] }>('/api/trading/orders', { query: { limit: 50 } }),
      $fetch<{ fills: FillRow[] }>('/api/trading/fills', { query: { limit: 50 } }),
    ])
    positions.value = p.positions ?? []
    orders.value = o.orders ?? []
    fills.value = f.fills ?? []
    positionsErr.value = ''
  } catch {
    positionsErr.value = 'Account unavailable — retrying…'
  }
}

const openOrders = computed(() => orders.value.filter(o => o.state === 'open' || o.state === 'partially_filled'))
// A closed position still carries realized P&L (counted in the stats) but is not an open position.
const openPositions = computed(() => positions.value.filter(p => Number(p.net_quantity) !== 0))
const unrealized = computed(() => positions.value.reduce((s, p) => s + Number(p.unrealized_pnl), 0))
const realized = computed(() => positions.value.reduce((s, p) => s + Number(p.realized_pnl_total), 0))
const positionsValue = computed(() => positions.value.reduce((s, p) => s + (p.side === 'long' ? Number(p.notional) : -Number(p.notional)), 0))
const totalValue = computed(() => cash.value + positionsValue.value)

/** cancelOrder withdraws one resting order. */
async function cancelOrder(id: string) {
  try {
    await $fetch(`/api/trading/orders/${id}`, { method: 'DELETE' })
  } catch (e: unknown) {
    showResult(false, errMessage(e, 'Cancel failed.'))
  }
  await refreshAfterOrder()
}

/** closePosition flattens a position with a market order on the other side. */
async function closePosition(p: PositionApi) {
  const qtyAbs = Math.abs(Number(p.net_quantity))
  await placeOrder({ product_id: p.product_id, side: p.side === 'long' ? 'sell' : 'buy', order_type: 'market', quantity: qtyAbs.toFixed(6) })
}

// =====================================================
// Order entry
// =====================================================
const orderSide = ref<Side>('buy')
const orderType = ref<OrderType>('limit')
const limitPx = ref('')
const qty = ref('')
const submitting = ref(false)
const orderMsg = ref('')
const orderOk = ref(false)

/** parseNum reads a typed number, ignoring thousands separators. */
function parseNum(s: string): number {
  return parseFloat(s.replace(/[^0-9.]/g, '')) || 0
}
const qtyN = computed(() => parseNum(qty.value))
const pxN = computed(() => orderType.value === 'market' ? (orderSide.value === 'buy' ? bestAsk.value : bestBid.value) || midPrice.value : parseNum(limitPx.value))
const notional = computed(() => pxN.value * qtyN.value)
const fee = computed(() => notional.value * 0.01)
const total = computed(() => orderSide.value === 'buy' ? notional.value + fee.value : notional.value - fee.value)
/** buyingPower is what the caller can trade now: paper cash ÷ (price + fee) to buy, credits held to sell. */
const buyingPower = computed(() => orderSide.value === 'sell' ? holdings.value : (pxN.value > 0 ? cash.value / (pxN.value * 1.01) : 0))
const creditLabel = computed(() => product.value?.name?.replace(/ credit$/i, '').toLowerCase() ?? 'credits')

/** setLastPx fills the limit price from the mid, on tick. */
function setLastPx() {
  limitPx.value = onTick(midPrice.value).toFixed(dp.value)
}
/** bumpPx moves the limit price one tick. */
function bumpPx(dir: 1 | -1) {
  limitPx.value = Math.max(0, onTick(pxN.value + dir * tick.value)).toFixed(dp.value)
}
/** bumpQty nudges the quantity. */
function bumpQty(dir: 1 | -1) {
  const step = qtyN.value < 100 ? 1 : qtyN.value < 1000 ? 10 : 100
  qty.value = fmtQty(Math.max(0, qtyN.value + dir * step))
}
/** quickFill sets the quantity to a share of buying power. */
function quickFill(pct: number) {
  qty.value = fmtQty(Math.floor(buyingPower.value * pct * 1e6) / 1e6)
}
/** onTick rounds a price to the product's tick. */
function onTick(p: number) {
  return Math.round(p / tick.value) * tick.value
}

const submitLabel = computed(() => {
  const side = orderSide.value === 'buy' ? 'Buy' : 'Sell'
  return `${side} ${fmtQty(qtyN.value)} ${PRODUCT_ID} — ${fmtUsd(total.value, total.value < 10 ? 4 : 2)}`
})

/** errMessage extracts the engine's message from a failed $fetch. */
function errMessage(e: unknown, fallback: string) {
  const d = (e as { data?: { message?: string; data?: { message?: string } } })?.data
  return d?.data?.message || d?.message || fallback
}

/** showResult shows the ticket's outcome line. */
function showResult(ok: boolean, msg: string) {
  orderOk.value = ok
  orderMsg.value = msg
}

const REASONS: Record<string, string> = {
  insufficient_cash: 'not enough paper cash', insufficient_credit: 'not enough credits to sell',
  risk_unavailable: 'the ledger could not reserve funds — try again', unfilled_remainder: 'no more liquidity at that price',
  fok_unfilled: 'could not fill completely', self_trade: 'it would have traded with your own order',
  user_cancel: 'cancelled by you', day_expired: 'expired at the end of the day',
}

/** describe turns the engine's order into one line for the ticket. */
function describe(o: OrderRow) {
  const filled = Number(o.filled_quantity)
  const avg = o.avg_fill_price ? ` at ${fmtPx(Number(o.avg_fill_price))}` : ''
  switch (o.state) {
    case 'filled': return [true, `Filled ${fmtQty(filled)}${avg}.`] as const
    case 'open': return [true, `Resting on the book at ${fmtPx(Number(o.limit_price))}.`] as const
    case 'partially_filled': return [true, `Filled ${fmtQty(filled)}${avg}; the rest rests on the book.`] as const
    case 'cancelled': return [filled > 0, filled > 0 ? `Filled ${fmtQty(filled)}${avg}; ${REASONS[o.reason ?? ''] ?? 'the rest was cancelled'}.` : `Not filled: ${REASONS[o.reason ?? ''] ?? o.reason}.`] as const
    default: return [false, `Rejected: ${REASONS[o.reason ?? ''] ?? o.reason}.`] as const
  }
}

/** placeOrder sends one order with a fresh idempotency key and reports the engine's answer. */
async function placeOrder(body: Record<string, string | undefined>) {
  if (submitting.value) return
  submitting.value = true
  orderMsg.value = ''
  try {
    const o = await $fetch<OrderRow>('/api/trading/orders', {
      method: 'POST', body, headers: { 'Idempotency-Key': crypto.randomUUID() },
    })
    const [ok, msg] = describe(o)
    showResult(ok, msg)
  } catch (e: unknown) {
    showResult(false, errMessage(e, 'Order could not be submitted.'))
  } finally {
    submitting.value = false
    await refreshAfterOrder()
  }
}

/** submitOrder places the ticket's order. */
function submitOrder() {
  if (qtyN.value <= 0) return showResult(false, 'Enter a quantity.')
  if (orderType.value === 'limit' && pxN.value <= 0) return showResult(false, 'Enter a limit price.')
  return placeOrder({
    product_id: PRODUCT_ID, side: orderSide.value, order_type: orderType.value,
    quantity: qtyN.value.toFixed(6),
    limit_price: orderType.value === 'limit' ? onTick(pxN.value).toFixed(6) : undefined,
    time_in_force: orderType.value === 'limit' ? 'gtc' : undefined,
  })
}

/** refreshAfterOrder updates everything an order can change. */
async function refreshAfterOrder() {
  await Promise.all([loadBook(), loadTape(), loadAccount(), loadBalances(), loadProduct()])
}

// =====================================================
// Live loops
// =====================================================
let chartInterval: ReturnType<typeof setInterval> | null = null
let bookInterval: ReturnType<typeof setInterval> | null = null
let tapeInterval: ReturnType<typeof setInterval> | null = null
let accountInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await loadProduct()
  await loadCandles()
  initChart()
  await loadBook()
  if (!limitPx.value && midPrice.value) setLastPx()
  void loadTape()
  void loadAccount()
  void loadBalances()
  chartInterval = setInterval(() => { void chartTick(); void loadProduct() }, 5000)
  bookInterval = setInterval(() => { void loadBook() }, 2000)
  tapeInterval = setInterval(() => { void loadTape() }, 3000)
  accountInterval = setInterval(() => { void loadAccount(); void loadBalances() }, 5000)
})
onBeforeUnmount(() => {
  for (const t of [chartInterval, bookInterval, tapeInterval, accountInterval]) if (t) clearInterval(t)
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
          <span class="sym">{{ PRODUCT_ID }}</span>
          <span class="paper-badge" title="Paper trading: paper cash and credits only. Real-money trading is paused pending exchange licensing.">PAPER</span>
          <span class="px" :class="headerPxClass">{{ fmtPx(midPrice) }}</span>
          <span class="delta" :class="arrowUp ? 'pos' : 'neg'">
            {{ arrowUp ? '▲' : '▼' }} {{ Math.abs(deltaPct).toFixed(2) }}%
          </span>
          <span class="stats">
            <span class="k">Vol 24h</span> {{ fmtQty(volume24h) }}
            <span class="k stat-spacer">H</span> {{ fmtPx(high24h) }}
            <span class="k stat-spacer">L</span> {{ fmtPx(low24h) }}
            <span v-if="tradedBars < candles.length" class="k stat-spacer ref-note" title="Bars with no paper trades show the reference price and are drawn grey.">grey bars = reference, no trades</span>
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
          </div>
        </header>
        <div class="panel-body">
          <div ref="chartContainer" class="chart-container" />
          <p v-if="chartErr" class="panel-err">{{ chartErr }}</p>
        </div>
      </section>

      <!-- ============ ORDER BOOK ============ -->
      <section class="panel book-panel">
        <header class="panel-head">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="head-ic">
            <path d="M3 4l5 3 5-3M3 8l5 3 5-3M3 12l5 3 5-3" />
          </svg>
          <span class="lbl">Order Book</span>
          <span class="lbl-badge">PAPER · 10 LVL</span>
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
                <span class="ob-qty">{{ fmtQty(l.qty) }}</span>
                <span class="ob-tot">{{ fmtQty(l.cum) }}</span>
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
                <span class="ob-qty">{{ fmtQty(l.qty) }}</span>
                <span class="ob-tot">{{ fmtQty(l.cum) }}</span>
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
              </div>
            </div>

            <div v-if="orderType === 'limit'" class="field">
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
                <span class="lbl">Quantity ({{ PRODUCT_ID }})</span>
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
                <span class="v">{{ orderType === 'market' ? '≈ ' : '' }}{{ fmtUsd(pxN, dp) }} × {{ fmtQty(qtyN) }} = {{ fmtUsd(notional, notional < 10 ? 4 : 2) }}</span>
              </div>
              <div class="sum-row">
                <span class="k">TAKER FEE · 1.00%</span>
                <span class="v">+ {{ fmtUsd(fee, fee < 1 ? 4 : 2) }}</span>
              </div>
              <div class="sum-row total">
                <span class="k total-k">TOTAL</span>
                <span class="v">{{ fmtUsd(total, total < 10 ? 4 : 2) }}</span>
              </div>
            </div>

            <div class="balance">
              <span class="k">AVAILABLE</span>
              <span class="v">{{ fmtUsd(cash) }} paper</span>
              <span class="sep">·</span>
              <span class="v">{{ fmtQty(holdings) }}</span>
              <span class="k">{{ PRODUCT_ID }} held</span>
            </div>

            <button class="submit-btn" :class="orderSide" type="button" :disabled="submitting" @click="submitOrder">
              {{ submitting ? 'Submitting…' : submitLabel }}
            </button>
            <!-- The engine's answer, in words: filled, resting, or why not (a real-money account is
                 refused with EXCHANGE_PAUSED — shown verbatim, never as an acceptance). -->
            <p v-if="orderMsg" class="order-msg" :class="orderOk ? 'ok' : 'paused'">{{ orderMsg }}</p>
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
              <p v-if="!tape.length" class="pos-empty">No paper trades yet. Place an order to print the first one.</p>
              <div
                v-for="(t, i) in tape"
                :key="t.time + i"
                class="tape-row"
                :class="[t.side, { large: t.large, enter: i === 0 }]"
              >
                <span class="t-time">{{ t.time }}</span>
                <span class="t-px">{{ fmtPx(t.px) }}</span>
                <span class="t-qty">{{ fmtQty(t.qty) }}</span>
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
        <span class="pos-title">Account</span>
        <div class="pos-tabs">
          <button
            class="pos-tab"
            :class="{ active: posTab === 'positions' }"
            type="button"
            @click="posTab = 'positions'"
          >Positions<span class="count">· {{ openPositions.length }}</span></button>
          <button
            class="pos-tab"
            :class="{ active: posTab === 'orders' }"
            type="button"
            @click="posTab = 'orders'"
          >Orders<span class="count">· {{ openOrders.length }} open</span></button>
          <button
            class="pos-tab"
            :class="{ active: posTab === 'fills' }"
            type="button"
            @click="posTab = 'fills'"
          >Fills<span class="count">· {{ fills.length }}</span></button>
        </div>
        <div class="pos-stats">
          <div class="stat">
            <span class="k">UNREALIZED P&amp;L</span>
            <span class="v" :class="unrealized >= 0 ? 'pos' : 'neg'">{{ unrealized >= 0 ? '+' : '−' }}{{ fmtUsd(Math.abs(unrealized)) }}</span>
          </div>
          <div class="stat">
            <span class="k">REALIZED</span>
            <span class="v" :class="realized >= 0 ? 'pos' : 'neg'">{{ realized >= 0 ? '+' : '−' }}{{ fmtUsd(Math.abs(realized)) }}</span>
          </div>
          <div class="stat">
            <span class="k">PAPER VALUE</span>
            <span class="v">{{ fmtUsd(totalValue) }}</span>
          </div>
        </div>
      </div>
      <div class="pos-table-wrap">
        <p v-if="positionsErr" class="pos-empty">{{ positionsErr }}</p>
        <table v-if="posTab === 'positions'" class="pos-table">
          <thead>
            <tr>
              <th class="left">Market</th><th class="left">Side</th><th>Size</th><th>Avg entry</th><th>Mark</th>
              <th>Notional</th><th>P&amp;L $</th><th>P&amp;L %</th><th class="right-th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!openPositions.length"><td colspan="9" class="pos-empty">No open positions — your paper fills build them.</td></tr>
            <tr v-for="p in openPositions" :key="p.product_id">
              <td class="left">{{ p.product_id }}</td>
              <td class="left"><span class="side-pill" :class="p.side">{{ p.side.toUpperCase() }}</span></td>
              <td>{{ fmtQty(Math.abs(Number(p.net_quantity))) }}</td>
              <td>{{ Number(p.avg_entry_price).toFixed(6) }}</td>
              <td>{{ Number(p.mark_price).toFixed(6) }}</td>
              <td>{{ fmtUsd(Number(p.notional)) }}</td>
              <td :class="Number(p.unrealized_pnl) >= 0 ? 'pos' : 'neg'">{{ Number(p.unrealized_pnl) >= 0 ? '+' : '−' }}{{ fmtUsd(Math.abs(Number(p.unrealized_pnl))) }}</td>
              <td :class="Number(p.unrealized_pnl) >= 0 ? 'pos' : 'neg'">{{ p.unrealized_pnl_pct }}%</td>
              <td class="right-td">
                <button v-if="Number(p.net_quantity) !== 0" type="button" class="close-btn" :disabled="submitting" @click="closePosition(p)">CLOSE</button>
              </td>
            </tr>
          </tbody>
        </table>
        <table v-else-if="posTab === 'orders'" class="pos-table">
          <thead>
            <tr>
              <th class="left">Time</th><th class="left">Market</th><th class="left">Side</th><th class="left">Type</th>
              <th>Qty</th><th>Filled</th><th>Limit</th><th class="left">State</th><th class="right-th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!orders.length"><td colspan="9" class="pos-empty">No orders yet.</td></tr>
            <tr v-for="o in orders" :key="o.order_id">
              <td class="left">{{ fmtTime(o.created_at) }}</td>
              <td class="left">{{ o.product_id }}</td>
              <td class="left"><span class="side-pill" :class="o.side === 'buy' ? 'long' : 'short'">{{ o.side.toUpperCase() }}</span></td>
              <td class="left">{{ o.order_type }}</td>
              <td>{{ fmtQty(Number(o.quantity)) }}</td>
              <td>{{ fmtQty(Number(o.filled_quantity)) }}</td>
              <td>{{ o.limit_price ? Number(o.limit_price).toFixed(6) : 'market' }}</td>
              <td class="left" :title="o.reason ?? ''">{{ o.state.replace('_', ' ') }}{{ o.reason && o.state !== 'filled' ? ' · ' + (REASONS[o.reason] ?? o.reason) : '' }}</td>
              <td class="right-td">
                <button v-if="o.state === 'open' || o.state === 'partially_filled'" type="button" class="close-btn" @click="cancelOrder(o.order_id)">CANCEL</button>
              </td>
            </tr>
          </tbody>
        </table>
        <table v-else class="pos-table">
          <thead>
            <tr>
              <th class="left">Time</th><th class="left">Market</th><th class="left">Side</th>
              <th>Price</th><th>Qty</th><th>Notional</th><th>Fee</th><th class="left">Liquidity</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!fills.length"><td colspan="8" class="pos-empty">No fills yet.</td></tr>
            <tr v-for="f in fills" :key="f.fill_id">
              <td class="left">{{ fmtTime(f.executed_at) }}</td>
              <td class="left">{{ f.product_id }}</td>
              <td class="left"><span class="side-pill" :class="f.side === 'buy' ? 'long' : 'short'">{{ f.side.toUpperCase() }}</span></td>
              <td>{{ Number(f.price).toFixed(6) }}</td>
              <td>{{ fmtQty(Number(f.quantity)) }}</td>
              <td>{{ fmtUsd(Number(f.notional)) }}</td>
              <td>{{ fmtUsd(Number(f.fee), 4) }}</td>
              <td class="left">{{ f.liquidity }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.paper-badge { font-family: var(--font-mono); font-size: 10px; letter-spacing: 0.14em; color: var(--warn); border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent); border-radius: var(--radius-sm); padding: 1px 6px; }
.ref-note { color: var(--text-3); font-size: 10px; }
.panel-err { position: absolute; top: 8px; left: 12px; margin: 0; color: var(--warn); font-size: var(--fs-xs); }
.close-btn:disabled { opacity: 0.5; cursor: default; }
.pos-empty { text-align: center; color: var(--text-3); padding: var(--sp-5) var(--sp-4); font-size: var(--fs-sm); }
/* Order result — the paused refusal is informational, not an error the user caused. */
.order-msg {
  margin: var(--sp-3) 0 0; padding: var(--sp-3);
  font-size: var(--fs-sm); line-height: var(--lh-base);
  border-radius: var(--radius-sm); border: 1px solid;
}
.order-msg.paused { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 42%, transparent); background: color-mix(in srgb, var(--warn) 10%, transparent); }
.order-msg.ok     { color: var(--pos); border-color: color-mix(in srgb, var(--pos) 42%, transparent); background: color-mix(in srgb, var(--pos) 10%, transparent); }
.submit-btn:disabled { opacity: 0.6; cursor: default; }
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
