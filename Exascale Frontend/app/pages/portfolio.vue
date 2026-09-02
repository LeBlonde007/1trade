<script setup lang="ts">
/**
 * /portfolio — Account overview, performance, positions, allocation, activity.
 * Dark theme. Renders inside the `app` layout (sidebar + topbar already provided).
 *
 * The standalone design ships its own topbar/breadcrumbs; here we keep the
 * project's shared AppTopbar/AppSidebar and port the page body verbatim.
 */
import { Chart, type ChartDataset } from 'chart.js/auto'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Portfolio — 1Trade' })

// =====================================================
// Mock data — mirrors /tmp design's portfolio-data.js
// =====================================================
type PosCat = 'ai' | 'sub' | 'gpu'
type Side = 'long' | 'short'

interface Fill {
  ts: string
  side: 'buy' | 'sell'
  qty: number
  px: number
  total: number
  taker: boolean
}
interface Position {
  id: string
  market: string
  sub: string
  sym: string
  cat: PosCat
  side: Side
  size: number
  unit: string
  avgEntry: number
  current: number
  decimals: number
  status: 'open' | 'closed'
  opened?: string
  closed?: string
  pnlOverride?: number
  pnlPctOverride?: number
  fees: number
  fills: Fill[]
  // computed
  pnl: number
  pnlPct: number
  value: number
  notional: number
}

const ACCOUNT = {
  id: 'EX-PT-7A3C91',
  name: 'Marcus Chen',
  email: 'marcus.chen@frontier.lab',
  value: 10247.83,
  cash:  10118.46,
  todayPnl: 23.41,
  todayPct: 0.23,
  allTimePnl: 247.83,
  allTimePct: 2.48,
}

function buildPositions(): Position[] {
  const raw: Omit<Position, 'pnl' | 'pnlPct' | 'value' | 'notional'>[] = [
    {
      id: 'p-aiidx-001', market: 'AI-INDEX', sub: 'Composite AI compute',
      sym: 'AI', cat: 'ai', side: 'long', size: 50000, unit: 'credits',
      avgEntry: 0.000980, current: 0.001005, decimals: 6, status: 'open',
      opened: 'May 17, 2026 · 09:42 ET', fees: 0.14,
      fills: [
        { ts: 'May 17, 09:41:22.184', side: 'buy',  qty: 18000, px: 0.000978, total: 17.604, taker: false },
        { ts: 'May 17, 09:42:08.011', side: 'buy',  qty: 22000, px: 0.000980, total: 21.560, taker: false },
        { ts: 'May 17, 14:18:55.642', side: 'buy',  qty: 10000, px: 0.000984, total: 9.840,  taker: true  },
      ],
    },
    {
      id: 'p-text-002', market: 'TEXT-INDEX', sub: 'Text-generation sub-credit',
      sym: 'TX', cat: 'sub', side: 'long', size: 12000, unit: 'credits',
      avgEntry: 0.001180, current: 0.001200, decimals: 6, status: 'open',
      opened: 'May 18, 2026 · 11:05 ET', fees: 0.08,
      fills: [{ ts: 'May 18, 11:05:14.330', side: 'buy', qty: 12000, px: 0.001180, total: 14.160, taker: false }],
    },
    {
      id: 'p-image-003', market: 'IMAGE-INDEX', sub: 'Image-generation sub-credit',
      sym: 'IM', cat: 'sub', side: 'short', size: 5000, unit: 'credits',
      avgEntry: 0.008100, current: 0.007980, decimals: 6, status: 'open',
      opened: 'May 19, 2026 · 14:32 ET', fees: 0.21,
      fills: [{ ts: 'May 19, 14:32:08.420', side: 'sell', qty: 5000, px: 0.008100, total: 40.500, taker: false }],
    },
    {
      id: 'p-h100-004', market: 'H100-USD', sub: 'GPU-hour credit',
      sym: 'H1', cat: 'gpu', side: 'long', size: 8, unit: 'GPU-hr',
      avgEntry: 2.9500, current: 2.9900, decimals: 4, status: 'open',
      opened: 'May 19, 2026 · 16:11 ET', fees: 0.12,
      fills: [
        { ts: 'May 19, 16:10:48.092', side: 'buy', qty: 3, px: 2.9450, total: 8.835,  taker: false },
        { ts: 'May 19, 16:11:33.901', side: 'buy', qty: 5, px: 2.9530, total: 14.765, taker: true  },
      ],
    },
    {
      id: 'p-aiidx-clo-005', market: 'AI-INDEX', sub: 'Composite AI compute',
      sym: 'AI', cat: 'ai', side: 'long', size: 25000, unit: 'credits',
      avgEntry: 0.000975, current: 0.001002, decimals: 6, status: 'closed',
      closed: 'May 16, 2026 · 17:55 ET',
      pnlOverride: 0.68, pnlPctOverride: 2.77, fees: 0.06,
      fills: [
        { ts: 'May 14, 10:08:12.114', side: 'buy',  qty: 25000, px: 0.000975, total: 24.375, taker: false },
        { ts: 'May 16, 17:55:01.207', side: 'sell', qty: 25000, px: 0.001002, total: 25.050, taker: true  },
      ],
    },
    {
      id: 'p-h200-clo-006', market: 'H200-USD', sub: 'GPU-hour credit',
      sym: 'H2', cat: 'gpu', side: 'short', size: 4, unit: 'GPU-hr',
      avgEntry: 3.5200, current: 3.4800, decimals: 4, status: 'closed',
      closed: 'May 15, 2026 · 12:22 ET',
      pnlOverride: 0.16, pnlPctOverride: 1.14, fees: 0.04,
      fills: [
        { ts: 'May 15, 09:14:48.221', side: 'sell', qty: 4, px: 3.5200, total: 14.080, taker: false },
        { ts: 'May 15, 12:21:55.408', side: 'buy',  qty: 4, px: 3.4800, total: 13.920, taker: true  },
      ],
    },
  ]
  return raw.map(p => {
    const dir = p.side === 'long' ? 1 : -1
    const notional = p.avgEntry * p.size
    const pnl    = p.status === 'closed' ? p.pnlOverride! : (p.current - p.avgEntry) * p.size * dir
    const pnlPct = p.status === 'closed' ? p.pnlPctOverride! : (p.current - p.avgEntry) / p.avgEntry * 100 * dir
    const value  = p.status === 'closed' ? 0 : p.current * p.size
    return { ...p, pnl, pnlPct, value, notional }
  })
}
const POSITIONS = buildPositions()

const ALLOCATION = [
  { key: 'cash',     label: 'Cash (USD)',          color: '#6B7280', value: ACCOUNT.cash },
  { key: 'aiindex',  label: 'AI-INDEX',            color: '#D4AF37', value: 50.25 },
  { key: 'textidx',  label: 'TEXT-INDEX',          color: '#4A90E2', value: 14.40 },
  { key: 'imageidx', label: 'IMAGE-INDEX (short)', color: '#7AAEEE', value: 39.90 },
  { key: 'h100',     label: 'H100 GPU credits',    color: '#F5A524', value: 23.92 },
  { key: 'h200',     label: 'H200 GPU credits',    color: '#F0C674', value: 0.90 },
]
const ALLOC_TOTAL = ALLOCATION.reduce((s, x) => s + x.value, 0)

const ASSET_BREAKDOWN = [
  { key: 'cash', label: 'Cash',        color: '#6B7280', value: ACCOUNT.cash },
  { key: 'ai',   label: 'AI Index',    color: '#D4AF37', value: 50.25 },
  { key: 'sub',  label: 'Sub-indices', color: '#4A90E2', value: 54.30 },
  { key: 'gpu',  label: 'GPU credits', color: '#F5A524', value: 24.82 },
]
const ASSET_TOTAL = ASSET_BREAKDOWN.reduce((s, x) => s + x.value, 0)

interface ActivityRow {
  ts: string
  market: string
  side: 'buy' | 'sell' | '—'
  qty: number | null
  px: number | null
  total: number
  type: string
  status: string
}
const ACTIVITY: ActivityRow[] = [
  { ts: 'May 19  16:32:48.711', market: 'H100-USD',    side: 'buy',  qty: 3,     px: 2.9450,   total: 8.835,  type: 'Limit',          status: 'filled' },
  { ts: 'May 19  14:32:08.420', market: 'IMAGE-INDEX', side: 'sell', qty: 5000,  px: 0.008100, total: 40.500, type: 'Limit',          status: 'filled' },
  { ts: 'May 19  09:18:02.110', market: 'AI-INDEX',    side: 'buy',  qty: 4000,  px: 0.000984, total: 3.936,  type: 'Market',         status: 'filled' },
  { ts: 'May 18  17:00:00.000', market: '—',           side: '—',    qty: null,  px: null,     total: 0,      type: 'EOD settlement', status: 'settled' },
  { ts: 'May 18  11:05:14.330', market: 'TEXT-INDEX',  side: 'buy',  qty: 12000, px: 0.001180, total: 14.160, type: 'Limit',          status: 'filled' },
  { ts: 'May 17  14:18:55.642', market: 'AI-INDEX',    side: 'buy',  qty: 10000, px: 0.000984, total: 9.840,  type: 'Limit',          status: 'filled' },
  { ts: 'May 17  09:42:08.011', market: 'AI-INDEX',    side: 'buy',  qty: 22000, px: 0.000980, total: 21.560, type: 'Limit',          status: 'filled' },
  { ts: 'May 16  17:55:01.207', market: 'AI-INDEX',    side: 'sell', qty: 25000, px: 0.001002, total: 25.050, type: 'Limit',          status: 'filled' },
  { ts: 'May 15  12:21:55.408', market: 'H200-USD',    side: 'buy',  qty: 4,     px: 3.4800,   total: 13.920, type: 'Market',         status: 'filled' },
  { ts: 'May 15  09:14:48.221', market: 'H200-USD',    side: 'sell', qty: 4,     px: 3.5200,   total: 14.080, type: 'Limit',          status: 'filled' },
]

const TICKERS = [
  { sym: 'AI-INDEX', px: 0.001005, chg:  0.41, dir: 'up'   },
  { sym: 'H100',     px: 2.9912,   chg:  0.18, dir: 'up'   },
  { sym: 'H200',     px: 3.8450,   chg: -0.27, dir: 'down' },
  { sym: 'TEXT',     px: 0.001210, chg:  1.84, dir: 'up'   },
  { sym: 'IMAGE',    px: 0.007980, chg: -0.62, dir: 'down' },
]

// =====================================================
// Formatters
// =====================================================
function fmtUSD(n: number, opts: { decimals?: number; signed?: boolean } = {}) {
  const decimals = opts.decimals ?? 2
  const sign = opts.signed && n > 0 ? '+' : ''
  const abs = Math.abs(n)
  const v = abs.toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
  return `${n < 0 ? '−' : sign}$${v}`
}
function fmtPct(n: number, decimals = 2) {
  return `${n > 0 ? '+' : n < 0 ? '−' : ''}${Math.abs(n).toFixed(decimals)}%`
}
function fmtNum(n: number, decimals = 0) {
  return n.toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
}
function fmtPx(n: number, decimals = 6) {
  return n.toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
}

// =====================================================
// Performance series
// =====================================================
type RangeKey = '24h' | '7d' | '30d' | '90d' | '1y' | 'all'
const RANGES: Record<RangeKey, { days: number; start: number; indexStart: number; indexEnd: number }> = {
  '24h': { days: 1,   start: 10224.42, indexStart: 1.00193, indexEnd: 1.00240 },
  '7d':  { days: 7,   start: 10298.18, indexStart: 1.00657, indexEnd: 1.00240 },
  '30d': { days: 30,  start: 10018.62, indexStart: 0.98019, indexEnd: 1.00240 },
  '90d': { days: 90,  start: 9847.20,  indexStart: 0.97842, indexEnd: 1.00240 },
  '1y':  { days: 365, start: 9620.10,  indexStart: 0.96241, indexEnd: 1.00240 },
  'all': { days: 540, start: 9000.00,  indexStart: 0.90015, indexEnd: 1.00240 },
}

function buildSeries(days: number, startVal: number, endVal: number) {
  const now = new Date('2026-05-19T16:30:00')
  const n = days <= 1 ? 48 : days <= 7 ? days * 8 : days <= 30 ? days : days <= 90 ? Math.floor(days / 2) : days <= 365 ? Math.floor(days / 4) : 60
  let path: number[] = []
  let cur = startVal
  for (let i = 0; i < n; i++) {
    const drift = (endVal - startVal) / n
    const vol = startVal * 0.003 * (days > 30 ? 1.8 : 1)
    const noise = (Math.sin(i * 1.7) + Math.cos(i * 0.9 + 1.2)) * vol * 0.4 + (Math.random() - 0.5) * vol
    cur += drift + noise
    path.push(cur)
  }
  const last = path[path.length - 1]!
  const diff = endVal - last
  for (let i = 0; i < path.length; i++) path[i]! += diff * (i / (path.length - 1))
  const msSpan = days * 24 * 3600 * 1000
  const out: { t: Date; v: number }[] = []
  for (let i = 0; i < n; i++) {
    const t = new Date(now.getTime() - msSpan + (i / (n - 1)) * msSpan)
    out.push({ t, v: path[i]! })
  }
  return out
}
function indexSeries(portfolio: { t: Date; v: number }[], startIdx: number, endIdx: number) {
  return portfolio.map((p, i) => {
    const frac = i / (portfolio.length - 1)
    const wobble = Math.sin(i * 0.45) * 0.004 + Math.cos(i * 0.21) * 0.003
    return { t: p.t, v: startIdx + (endIdx - startIdx) * frac + wobble }
  })
}
const SERIES = {} as Record<RangeKey, { portfolio: { t: Date; v: number }[]; index: { t: Date; v: number }[]; days: number }>
for (const k of Object.keys(RANGES) as RangeKey[]) {
  const r = RANGES[k]
  const port = buildSeries(r.days, r.start, ACCOUNT.value)
  const idx = indexSeries(port, r.indexStart, r.indexEnd)
  SERIES[k] = { portfolio: port, index: idx, days: r.days }
}

// =====================================================
// State
// =====================================================
const currentRange = ref<RangeKey>('30d')
const compareOn = ref(false)
const RANGE_KEYS: RangeKey[] = ['24h', '7d', '30d', '90d', '1y', 'all']
const POS_FILTERS: Array<'all' | 'open' | 'closed'> = ['all', 'open', 'closed']
const posFilter = ref<'all' | 'open' | 'closed'>('all')
const sortKey = ref<keyof Position>('pnl')
const sortDir = ref<'asc' | 'desc'>('desc')
const openRows = ref<Set<string>>(new Set())

const filteredPositions = computed(() => {
  const rows = POSITIONS.filter(p => posFilter.value === 'all' ? true : p.status === posFilter.value)
  const k = sortKey.value
  return rows.slice().sort((a, b) => {
    const va = (a[k] as number | string)
    const vb = (b[k] as number | string)
    if (typeof va === 'string') {
      return sortDir.value === 'asc' ? va.localeCompare(vb as string) : (vb as string).localeCompare(va)
    }
    return sortDir.value === 'asc' ? (va as number) - (vb as number) : (vb as number) - (va as number)
  })
})

function toggleRow(id: string) {
  if (openRows.value.has(id)) openRows.value.delete(id)
  else openRows.value.add(id)
  openRows.value = new Set(openRows.value)
}

function setSort(key: keyof Position) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = 'desc'
  }
}

function classifySym(cat: PosCat) {
  if (cat === 'ai') return 'lime'
  if (cat === 'sub') return 'blue'
  return 'orange'
}

// =====================================================
// Period summary
// =====================================================
const periodSummary = computed(() => {
  const s = SERIES[currentRange.value]
  const port = s.portfolio
  const idx = s.index
  const startV = port[0]!.v
  const endV   = port[port.length - 1]!.v
  const high   = Math.max(...port.map(p => p.v))
  const low    = Math.min(...port.map(p => p.v))
  const delta  = endV - startV
  const deltaP = (delta / startV) * 100
  const idxStart = idx[0]!.v
  const idxEnd   = idx[idx.length - 1]!.v
  const idxP = (idxEnd - idxStart) / idxStart * 100
  const ppDiff = deltaP - idxP
  return { startV, endV, high, low, delta, deltaP, ppDiff }
})

const rangeLabel = computed(() => ({
  '24h': '24h', '7d': '7d', '30d': '30d', '90d': '90d', '1y': '1y', 'all': 'all-time',
}[currentRange.value]))

// =====================================================
// Donut center state
// =====================================================
const donutHover = ref<number | null>(null)
const donutCenter = computed(() => {
  if (donutHover.value === null) {
    return { lbl: 'Total value', val: fmtUSD(ALLOC_TOTAL), pct: '100.00%' }
  }
  const a = ALLOCATION[donutHover.value]!
  return { lbl: a.label, val: fmtUSD(a.value), pct: (a.value / ALLOC_TOTAL * 100).toFixed(2) + '%' }
})

// =====================================================
// Live drifting hero value
// =====================================================
const heroBase = ref(ACCOUNT.value)
const heroWhole = computed(() => '$' + Math.floor(heroBase.value).toLocaleString('en-US'))
const heroCents = computed(() => (heroBase.value - Math.floor(heroBase.value)).toFixed(2).slice(1))

// =====================================================
// Chart.js init (client only)
// =====================================================
const perfCanvas = ref<HTMLCanvasElement | null>(null)
const donutCanvas = ref<HTMLCanvasElement | null>(null)
let perfChart: Chart<'line', number[]> | null = null
let donutChart: Chart<'doughnut'> | null = null

function formatDateLabel(d: Date, days: number) {
  if (days <= 1) return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
  if (days <= 90) return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  return d.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
}

function hexA(hex: string, a: number) {
  const h = hex.replace('#', '')
  return `rgba(${parseInt(h.slice(0, 2), 16)},${parseInt(h.slice(2, 4), 16)},${parseInt(h.slice(4, 6), 16)},${a})`
}

function initPerfChart() {
  if (!perfCanvas.value) return
  const ctx = perfCanvas.value.getContext('2d')!
  const grad = ctx.createLinearGradient(0, 0, 0, 360)
  grad.addColorStop(0, hexA('#D4AF37', 0.18))
  grad.addColorStop(1, hexA('#D4AF37', 0))
  const s = SERIES[currentRange.value]
  perfChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: s.portfolio.map(p => formatDateLabel(p.t, s.days)),
      datasets: [
        {
          label: 'Account value',
          data: s.portfolio.map(p => p.v),
          borderColor: '#D4AF37',
          borderWidth: 1.6,
          backgroundColor: grad,
          fill: true,
          tension: 0.18,
          pointRadius: 0,
          pointHoverRadius: 4,
          pointHoverBorderColor: '#D4AF37',
          pointHoverBackgroundColor: '#121212',
          pointHoverBorderWidth: 1.5,
        } as ChartDataset<'line', number[]>,
        {
          label: 'AI-INDEX',
          data: s.index.map(p => p.v),
          borderColor: '#4A90E2',
          borderWidth: 1.2,
          borderDash: [4, 3],
          backgroundColor: 'transparent',
          fill: false,
          tension: 0.18,
          pointRadius: 0,
          pointHoverRadius: 4,
          pointHoverBorderColor: '#4A90E2',
          pointHoverBackgroundColor: '#121212',
          pointHoverBorderWidth: 1.5,
          yAxisID: 'yIndex',
          hidden: true,
        } as ChartDataset<'line', number[]>,
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { intersect: false, mode: 'index' },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: '#1A1A1A',
          borderColor: 'rgba(255,255,255,0.16)',
          borderWidth: 1,
          padding: 10,
          cornerRadius: 2,
          titleColor: '#A8A196',
          titleFont: { family: "'JetBrains Mono', monospace", size: 10, weight: 'bold' },
          bodyColor: '#E8E2D6',
          bodyFont: { family: "'JetBrains Mono', monospace", size: 12 },
          displayColors: false,
          callbacks: {
            title: items => {
              const idx = items[0]!.dataIndex
              const t = SERIES[currentRange.value].portfolio[idx]!.t
              return (
                t.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
                + ' · '
                + t.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
              )
            },
            label: item => {
              if (item.datasetIndex === 0) return 'Account  ' + fmtUSD(item.parsed.y as number)
              return 'AI-INDEX  ' + (item.parsed.y as number).toFixed(5)
            },
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          border: { display: false },
          ticks: {
            color: '#7E786C',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 8,
          },
        },
        y: {
          position: 'right',
          grid: { color: 'rgba(255,255,255,0.05)' },
          border: { display: false },
          ticks: {
            color: '#7E786C',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            callback: v => '$' + (v as number).toLocaleString('en-US', { maximumFractionDigits: 0 }),
            padding: 8,
          },
        },
        yIndex: {
          position: 'left',
          display: false,
          grid: { display: false },
          ticks: { display: false },
        } as never,
      },
    },
  })
}

function initDonut() {
  if (!donutCanvas.value) return
  const ctx = donutCanvas.value.getContext('2d')!
  donutChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: ALLOCATION.map(a => a.label),
      datasets: [{
        data: ALLOCATION.map(a => a.value),
        backgroundColor: ALLOCATION.map(a => a.color),
        borderColor: '#121212',
        borderWidth: 2,
        hoverBorderColor: '#121212',
        hoverBorderWidth: 2,
        spacing: 1,
      } as never],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '72%',
      plugins: { legend: { display: false }, tooltip: { enabled: false } },
      onHover: (_e, els) => {
        if (!donutCanvas.value) return
        donutCanvas.value.style.cursor = els.length ? 'pointer' : 'default'
        donutHover.value = els.length ? els[0]!.index : null
      },
    },
  })
}

function applyRange(key: RangeKey) {
  currentRange.value = key
  if (!perfChart) return
  const s = SERIES[key]
  perfChart.data.labels = s.portfolio.map(p => formatDateLabel(p.t, s.days))
  perfChart.data.datasets[0]!.data = s.portfolio.map(p => p.v) as never
  perfChart.data.datasets[1]!.data = s.index.map(p => p.v) as never
  perfChart.data.datasets[1]!.hidden = !compareOn.value
  perfChart.update()
}

function toggleCompare() {
  compareOn.value = !compareOn.value
  if (!perfChart) return
  perfChart.data.datasets[1]!.hidden = !compareOn.value
  perfChart.update()
}

let heroTick: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  initPerfChart()
  initDonut()
  heroTick = setInterval(() => {
    const delta = (Math.random() - 0.45) * 0.05
    heroBase.value += delta
  }, 2200)
})
onBeforeUnmount(() => {
  perfChart?.destroy()
  donutChart?.destroy()
  if (heroTick) clearInterval(heroTick)
})
</script>

<template>
  <div class="portfolio-page">
    <!-- Sub-topbar: crumbs + ticker + Paper pill -->
    <div class="subbar">
      <div class="crumbs">
        <span>EX</span>
        <span class="sep">/</span>
        <span>Account</span>
        <span class="sep">/</span>
        <span class="here">Portfolio</span>
      </div>
      <div class="market-ticker">
        <span v-for="t in TICKERS" :key="t.sym" class="ticker-item">
          <span class="sym">{{ t.sym }}</span>
          <span class="px">{{ t.px < 1 ? t.px.toFixed(6) : t.px.toFixed(4) }}</span>
          <span class="chg" :class="t.dir">{{ t.dir === 'up' ? '▲' : '▼' }} {{ Math.abs(t.chg).toFixed(2) }}%</span>
        </span>
      </div>
      <div class="subbar-right">
        <span class="env-pill"><span class="dot" /> Paper</span>
      </div>
    </div>

    <div class="page">
      <!-- ================= HERO ================= -->
      <section class="hero">
        <div class="hero-left">
          <div class="hero-eyebrow">
            <span>Portfolio overview</span>
            <span class="acct-pill"><span class="dot" /> Paper trading</span>
            <span class="acct-id">{{ ACCOUNT.id }}</span>
          </div>

          <div class="hero-value">
            <span>{{ heroWhole }}</span>
            <span class="cents">{{ heroCents }}</span>
            <span class="ccy">USD</span>
          </div>

          <div class="hero-stats">
            <div class="hero-stat">
              <span class="lbl">Today</span>
              <span class="val up">
                <span class="arr">▲</span>
                <span>+$23.41</span>
                <span class="pct">+0.23%</span>
              </span>
            </div>
            <div class="hero-stat">
              <span class="lbl">All-time</span>
              <span class="val up">
                <span class="arr">▲</span>
                <span>+$247.83</span>
                <span class="pct">+2.48%</span>
              </span>
            </div>
            <div class="hero-stat">
              <span class="lbl">Cash available</span>
              <span class="val plain">
                <span>$10,118.46</span>
              </span>
            </div>
            <div class="hero-stat">
              <span class="lbl">Open positions</span>
              <span class="val plain">
                <span>4</span>
                <span class="pct dim">2 closed today</span>
              </span>
            </div>
          </div>
        </div>

        <div class="hero-actions">
          <div class="btn-row">
            <span class="tooltip-host">
              <button class="btn disabled" disabled>
                <svg viewBox="0 0 16 16"><path d="M8 3v8M5 8l3-3 3 3" /></svg>
                Deposit
              </button>
              <span class="tooltip">Available after real-money upgrade</span>
            </span>
            <span class="tooltip-host">
              <button class="btn disabled" disabled>
                <svg viewBox="0 0 16 16"><path d="M8 13V5M5 8l3 3 3-3" /></svg>
                Withdraw
              </button>
              <span class="tooltip">Available after real-money upgrade</span>
            </span>
            <button class="btn primary">
              Upgrade to real money
              <svg viewBox="0 0 16 16"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
            </button>
          </div>
          <div class="hero-meta">
            <span class="live"><span class="pulse" />Live data · NYSE 14:32 ET</span>
            <span>·</span>
            <span>Statement YTD →</span>
          </div>
        </div>
      </section>

      <!-- ================= PERFORMANCE ================= -->
      <section class="card perf-card">
        <div class="card-head">
          <div class="perf-title">
            <span class="eyebrow">— Performance · account value</span>
            <span class="ttl">
              {{ fmtUSD(periodSummary.endV) }}
              <span class="perf-delta" :class="periodSummary.delta >= 0 ? 'up' : 'down'">
                {{ periodSummary.delta >= 0 ? '▲' : '▼' }}
                {{ fmtUSD(periodSummary.delta, { signed: true }) }}
                ({{ fmtPct(periodSummary.deltaP) }})
              </span>
              <span class="perf-range">· {{ rangeLabel }}</span>
            </span>
          </div>
          <div class="perf-controls">
            <span class="toggle" :class="{ on: compareOn }" role="switch" :aria-checked="compareOn" @click="toggleCompare">
              <span class="toggle-switch" />
              <span>Compare to AI-INDEX</span>
              <BaseHelpDot
                title="Compare to AI-INDEX"
                what="Overlays the AI-INDEX (EAI-IDX) performance on the same chart, rebased to your portfolio's starting value."
                why="Tells you whether you actually beat the market or just rode it. Lifted from any sleeve under the index = you added skill; below = you didn't."
                field="portfolio.compare.idx"
                placement="bottom"
                tone="on-dark"
              />
            </span>
            <div class="tab-bar">
              <button
                v-for="k in RANGE_KEYS"
                :key="k"
                class="tab"
                :class="{ active: currentRange === k }"
                @click="applyRange(k)"
              >{{ k.toUpperCase() }}</button>
              <BaseHelpDot
                title="Time range"
                what="Switches the chart and KPI summary between 24h, 7d, 30d, 90d, 1y, and full account lifetime."
                why="Short ranges show recent moves; longer ranges show the underlying trajectory. Compare 24h spikes against 1y context before reacting."
                field="portfolio.chart.range"
                placement="left"
                tone="on-dark"
              />
            </div>
          </div>
        </div>
        <div class="perf-body">
          <div class="perf-canvas-wrap">
            <canvas ref="perfCanvas" />
          </div>
        </div>
        <div class="perf-summary">
          <div class="item">
            <div class="lbl">Period start</div>
            <div class="val">{{ fmtUSD(periodSummary.startV) }}</div>
          </div>
          <div class="item">
            <div class="lbl">Period high</div>
            <div class="val">{{ fmtUSD(periodSummary.high) }}</div>
          </div>
          <div class="item">
            <div class="lbl">Period low</div>
            <div class="val">{{ fmtUSD(periodSummary.low) }}</div>
          </div>
          <div class="item">
            <div class="lbl">Period Δ</div>
            <div class="val">
              {{ fmtUSD(periodSummary.delta, { signed: true }) }}
              <span class="chg" :class="periodSummary.deltaP >= 0 ? 'up' : 'down'">{{ fmtPct(periodSummary.deltaP) }}</span>
            </div>
          </div>
          <div class="item">
            <div class="lbl">vs AI-INDEX</div>
            <div class="val">
              <span :class="periodSummary.ppDiff >= 0 ? 'up' : 'down'">
                {{ periodSummary.ppDiff >= 0 ? '+' : '−' }}{{ Math.abs(periodSummary.ppDiff).toFixed(2) }} pp
              </span>
            </div>
          </div>
          <div class="item">
            <div class="lbl">Sharpe (period)</div>
            <div class="val">1.48</div>
          </div>
        </div>
      </section>

      <!-- ================= POSITIONS + ALLOCATION ================= -->
      <div class="row-2col">
        <section class="card">
          <div class="card-head">
            <div class="tab-filter">
              <button
                v-for="f in POS_FILTERS"
                :key="f"
                class="tf"
                :class="{ active: posFilter === f }"
                @click="posFilter = f"
              >
                {{ f[0].toUpperCase() + f.slice(1) }}
                <span class="count">{{ f === 'all' ? POSITIONS.length : POSITIONS.filter(p => p.status === f).length }}</span>
              </button>
            </div>
            <div class="card-head-actions">
              <button class="btn sm">
                <svg viewBox="0 0 16 16"><path d="M3 4h10M5 8h6M7 12h2" /></svg>
                Filter
              </button>
              <button class="btn sm">
                <svg viewBox="0 0 16 16"><path d="M8 3v10M3 8l5 5 5-5" /></svg>
                Export CSV
              </button>
            </div>
          </div>

          <table class="table positions-table">
            <thead>
              <tr>
                <th @click="setSort('market')">Market <span class="sort">{{ sortKey === 'market' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="center">Side</th>
                <th class="num" @click="setSort('size')">Size <span class="sort">{{ sortKey === 'size' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="num" @click="setSort('avgEntry')">Avg entry <span class="sort">{{ sortKey === 'avgEntry' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="num" @click="setSort('current')">Mark <span class="sort">{{ sortKey === 'current' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="num" @click="setSort('value')">Value <span class="sort">{{ sortKey === 'value' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="num" :class="{ sorted: sortKey === 'pnl' }" @click="setSort('pnl')">P&amp;L <span class="sort">{{ sortKey === 'pnl' ? (sortDir === 'asc' ? '↑' : '↓') : '↕' }}</span></th>
                <th class="num" @click="setSort('pnlPct')">P&amp;L %</th>
                <th class="center caret-th"></th>
              </tr>
            </thead>
            <tbody>
              <template v-for="p in filteredPositions" :key="p.id">
                <tr class="row-main" :class="{ open: openRows.has(p.id) }" @click="toggleRow(p.id)">
                  <td>
                    <div class="mkt">
                      <span class="sym-square" :class="classifySym(p.cat)">{{ p.sym }}</span>
                      <span class="info">
                        <span class="name">
                          {{ p.market }}
                          <span v-if="p.status === 'closed'" class="closed-tag">· closed</span>
                        </span>
                        <span class="sub">{{ p.sub }}</span>
                      </span>
                    </div>
                  </td>
                  <td class="center">
                    <span class="side-tag" :class="p.side">
                      <span class="arr">{{ p.side === 'long' ? '▲' : '▼' }}</span>{{ p.side.toUpperCase() }}
                    </span>
                  </td>
                  <td class="num">{{ fmtNum(p.size) }} <span class="dim sm-unit">{{ p.unit }}</span></td>
                  <td class="num">{{ fmtPx(p.avgEntry, p.decimals) }}</td>
                  <td class="num">{{ fmtPx(p.current, p.decimals) }}</td>
                  <td class="num">
                    <span v-if="p.status === 'closed'" class="dim">—</span>
                    <template v-else>{{ fmtUSD(p.value) }}</template>
                  </td>
                  <td class="num" :class="p.pnl > 0 ? 'pnl-up' : p.pnl < 0 ? 'pnl-down' : 'pnl-flat'">
                    <span class="tiny-arr">{{ p.pnl >= 0 ? '▲' : '▼' }}</span>{{ fmtUSD(p.pnl, { signed: true }) }}
                  </td>
                  <td class="num" :class="p.pnl > 0 ? 'pnl-up' : p.pnl < 0 ? 'pnl-down' : 'pnl-flat'">
                    {{ fmtPct(p.pnlPct) }}
                  </td>
                  <td class="center">
                    <span class="caret"><svg viewBox="0 0 10 10"><path d="M3 1l4 4-4 4" /></svg></span>
                  </td>
                </tr>
                <tr v-if="openRows.has(p.id)" class="row-expand">
                  <td colspan="9">
                    <div class="expand-inner">
                      <div class="kv">
                        <div class="k">Status</div>
                        <div class="v">{{ p.status === 'closed' ? 'Closed · ' + p.closed : 'Open · since ' + p.opened }}</div>
                      </div>
                      <div class="kv">
                        <div class="k">Notional</div>
                        <div class="v">{{ fmtUSD(p.notional, { decimals: 3 }) }}</div>
                      </div>
                      <div class="kv">
                        <div class="k">Fees paid</div>
                        <div class="v">{{ fmtUSD(p.fees, { decimals: 3 }) }}</div>
                      </div>
                      <div class="kv">
                        <div class="k">Position id</div>
                        <div class="v subtle">{{ p.id }}</div>
                      </div>
                      <div class="timeline">
                        <div class="ttl">Fills · {{ p.fills.length }} total</div>
                        <div v-for="(f, fi) in p.fills" :key="fi" class="fill-row">
                          <span class="ts">{{ f.ts }}</span>
                          <span>
                            <span class="side-tag micro" :class="f.side === 'buy' ? 'long' : 'short'">
                              <span class="arr">{{ f.side === 'buy' ? '▲' : '▼' }}</span>{{ f.side.toUpperCase() }}
                            </span>
                          </span>
                          <span>{{ fmtNum(f.qty) }} {{ p.unit }}</span>
                          <span class="px">{{ fmtPx(f.px, p.decimals) }}</span>
                          <span class="dim">{{ f.taker ? 'Taker (0 bps)' : 'Maker (0 bps)' }}</span>
                          <span class="right">{{ fmtUSD(f.total, { decimals: 3 }) }}</span>
                        </div>
                      </div>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </section>

        <!-- Allocation -->
        <aside class="card alloc-card">
          <div class="card-head">
            <span class="ttl">Allocation</span>
            <span class="eyebrow">by market</span>
          </div>
          <div class="alloc-body">
            <div class="donut-wrap">
              <canvas ref="donutCanvas" />
              <div class="donut-center">
                <div class="lbl">{{ donutCenter.lbl }}</div>
                <div class="val">{{ donutCenter.val }}</div>
                <div class="pct">{{ donutCenter.pct }}</div>
              </div>
            </div>

            <div class="legend">
              <div
                v-for="(a, i) in ALLOCATION"
                :key="a.key"
                class="legend-row"
                :class="{ hovered: donutHover === i }"
                @mouseover="donutHover = i"
                @mouseleave="donutHover = null"
              >
                <span class="swatch" :style="{ background: a.color }" />
                <span class="name">{{ a.label }}</span>
                <span class="pct">{{ (a.value / ALLOC_TOTAL * 100).toFixed(2) }}%</span>
                <span class="val">{{ fmtUSD(a.value) }}</span>
              </div>
            </div>

            <div>
              <div class="section-eyebrow">Asset breakdown</div>
              <div class="asset-bar">
                <div
                  v-for="b in ASSET_BREAKDOWN"
                  :key="b.key"
                  class="seg"
                  :style="{ width: (b.value / ASSET_TOTAL * 100) + '%', background: b.color }"
                  :title="`${b.label} · ${(b.value / ASSET_TOTAL * 100).toFixed(2)}%`"
                />
              </div>
              <div class="asset-legend">
                <div v-for="b in ASSET_BREAKDOWN" :key="b.key" class="row">
                  <span class="name"><span class="swatch" :style="{ background: b.color }" />{{ b.label }}</span>
                  <span class="val">{{ (b.value / ASSET_TOTAL * 100).toFixed(2) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <!-- ================= RECENT ACTIVITY ================= -->
      <section class="card activity-card">
        <div class="card-head">
          <span class="ttl">Recent activity</span>
          <a href="#">View full history →</a>
        </div>
        <table class="table">
          <thead>
            <tr>
              <th class="ts-col">Timestamp</th>
              <th>Market</th>
              <th class="center">Side</th>
              <th class="num">Quantity</th>
              <th class="num">Price</th>
              <th class="num">Total (USD)</th>
              <th>Type</th>
              <th class="center">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(a, i) in ACTIVITY" :key="i">
              <template v-if="a.side === '—'">
                <td class="mono dim">{{ a.ts }}</td>
                <td colspan="6" class="settle">{{ a.type }}  ·  daily P&amp;L reconciled, fees swept</td>
                <td class="center"><span class="status-tag closed"><span class="dot" /> {{ a.status }}</span></td>
              </template>
              <template v-else>
                <td class="mono dim">{{ a.ts }}</td>
                <td><span class="mono mkt-name">{{ a.market }}</span></td>
                <td class="center">
                  <span class="side-tag" :class="a.side === 'buy' ? 'long' : 'short'">
                    <span class="arr">{{ a.side === 'buy' ? '▲' : '▼' }}</span>{{ a.side.toUpperCase() }}
                  </span>
                </td>
                <td class="num">{{ fmtNum(a.qty ?? 0) }}</td>
                <td class="num">{{ fmtPx(a.px ?? 0, (a.market === 'H100-USD' || a.market === 'H200-USD') ? 4 : 6) }}</td>
                <td class="num">{{ fmtUSD(a.total, { decimals: 3 }) }}</td>
                <td><span class="mono type-cell">{{ a.type }}</span></td>
                <td class="center"><span class="status-tag pos-status"><span class="dot" /> {{ a.status }}</span></td>
              </template>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* ============================================
   Page shell — sits inside the app layout (dark)
   ============================================ */
.portfolio-page {
  --text-4:      #3F3F3D;
  --border-soft: rgba(255, 255, 255, 0.05);
  --brand-soft:  rgba(200, 242, 92, 0.12);
  --sunken:      #07080A;

  font-family: var(--font-sans);
  font-size: 13px;
  font-feature-settings: 'ss01';
  -webkit-font-smoothing: antialiased;
  color: var(--text);
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.dim { color: var(--text-3); }
.subtle { color: var(--text-2); }
.right { text-align: right; }

/* ============================================
   Sub-topbar — crumbs + ticker (sits under app topbar)
   ============================================ */
.subbar {
  height: 48px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 16px;
  background: var(--canvas);
}
.crumbs {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.08em;
  color: var(--text-3);
  text-transform: uppercase;
  display: flex;
  align-items: center;
  gap: 8px;
}
.crumbs .sep { color: var(--text-4); }
.crumbs .here { color: var(--text); }

.market-ticker {
  display: flex;
  align-items: center;
  gap: 18px;
  margin-left: 24px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  font-variant-numeric: tabular-nums;
}
.ticker-item { display: flex; align-items: center; gap: 6px; }
.ticker-item .sym { color: var(--text-3); letter-spacing: 0.04em; }
.ticker-item .px { color: var(--text); }
.ticker-item .chg.up { color: var(--pos); }
.ticker-item .chg.down { color: var(--neg); }

.subbar-right { margin-left: auto; display: flex; align-items: center; gap: 14px; }
.env-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  padding: 4px 8px;
  background: var(--info);
  color: #fff;
  border-radius: var(--radius-sm);
  font-weight: 600;
  opacity: 0.9;
}
.env-pill .dot { width: 5px; height: 5px; border-radius: 50%; background: #fff; }

/* ============================================
   Page container
   ============================================ */
.page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 32px 24px 80px;
  width: 100%;
}

/* ============================================
   Hero
   ============================================ */
.hero {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: flex-end;
  gap: 32px;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 28px;
}
.hero-left { min-width: 0; }
.hero-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.hero-eyebrow .acct-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  background: var(--brand-soft);
  color: var(--brand);
  border: 1px solid rgba(200, 242, 92, 0.25);
  border-radius: var(--radius-sm);
  letter-spacing: 0.18em;
}
.hero-eyebrow .acct-pill .dot { width: 5px; height: 5px; background: var(--brand); border-radius: 50%; }
.hero-eyebrow .acct-id { color: var(--text-2); font-weight: 500; letter-spacing: 0.06em; }

.hero-value {
  font-family: var(--font-mono);
  font-size: 64px;
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.hero-value .cents { color: var(--text-2); font-size: 0.65em; margin-left: -8px; }
.hero-value .ccy {
  font-family: var(--font-mono);
  font-size: 14px;
  color: var(--text-3);
  letter-spacing: 0.16em;
  font-weight: 500;
  padding-bottom: 10px;
}

.hero-stats { display: flex; gap: 32px; margin-top: 16px; align-items: baseline; }
.hero-stat { display: flex; flex-direction: column; gap: 4px; }
.hero-stat .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.hero-stat .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 15px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}
.hero-stat .val.up { color: var(--pos); }
.hero-stat .val.down { color: var(--neg); }
.hero-stat .val.plain { color: var(--text); }
.hero-stat .val .pct { color: inherit; font-size: 12px; opacity: 0.78; }
.hero-stat .val .pct.dim { color: var(--text-3); opacity: 1; }
.hero-stat .val .arr {
  font-size: 9px;
  display: inline-block;
  transform: translateY(-1px);
}

.hero-actions { display: flex; flex-direction: column; gap: 12px; align-items: flex-end; }
.btn-row { display: flex; gap: 8px; }
.tooltip-host { position: relative; }
.tooltip-host .tooltip {
  position: absolute;
  bottom: calc(100% + 8px);
  right: 0;
  background: var(--overlay);
  color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  font-size: 11px;
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  transform: translateY(2px);
  transition: opacity 120ms, transform 120ms;
  font-family: var(--font-sans);
  z-index: 5;
}
.tooltip-host:hover .tooltip { opacity: 1; transform: translateY(0); }

.hero-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}
.hero-meta .live { display: inline-flex; align-items: center; gap: 6px; }
.hero-meta .pulse {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--pos);
  box-shadow: 0 0 0 0 var(--pos);
  animation: pulseRing 2.4s infinite;
}
@keyframes pulseRing {
  0%   { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0.6); }
  70%  { box-shadow: 0 0 0 6px rgba(25, 195, 125, 0); }
  100% { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0); }
}

/* ============================================
   Buttons
   ============================================ */
.btn {
  height: 32px;
  padding: 0 14px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-strong);
  background: var(--elevated);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 120ms, border-color 120ms, color 120ms;
}
.btn:hover { background: var(--overlay); border-color: rgba(255, 255, 255, 0.22); }
.btn:disabled,
.btn.disabled {
  background: var(--elevated);
  color: var(--text-3);
  border-color: var(--border);
  cursor: not-allowed;
}
.btn.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
}
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.sm { height: 26px; padding: 0 10px; font-size: 12px; gap: 6px; }
.btn svg { width: 13px; height: 13px; stroke: currentColor; fill: none; stroke-width: 1.6; }

/* ============================================
   Card
   ============================================ */
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-soft);
  gap: 16px;
}
.card-head .ttl {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.005em;
}
.card-head .eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.card-head-actions { display: flex; gap: 8px; }

/* tab bar */
.tab-bar {
  display: inline-flex;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.tab {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  color: var(--text-2);
  padding: 5px 11px;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.04em;
  cursor: pointer;
  transition: background 120ms, color 120ms;
  font-variant-numeric: tabular-nums;
}
.tab:last-child { border-right: 0; }
.tab:hover { color: var(--text); background: var(--overlay); }
.tab.active { background: var(--overlay); color: var(--text); }
.tab.active::before {
  content: '';
  display: inline-block;
  width: 4px; height: 4px;
  background: var(--brand);
  margin-right: 6px;
  vertical-align: 2px;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-2);
  cursor: pointer;
  user-select: none;
}
.toggle-switch {
  position: relative;
  width: 28px;
  height: 16px;
  background: var(--border);
  border-radius: 999px;
  transition: background 120ms;
}
.toggle-switch::after {
  content: '';
  position: absolute;
  left: 2px;
  top: 2px;
  width: 12px;
  height: 12px;
  background: var(--text-2);
  border-radius: 50%;
  transition: transform 160ms cubic-bezier(0.2, 0, 0, 1), background 120ms;
}
.toggle.on .toggle-switch { background: var(--brand); }
.toggle.on .toggle-switch::after { transform: translateX(12px); background: var(--canvas); }
.toggle.on { color: var(--text); }

/* ============================================
   Performance chart
   ============================================ */
.perf-card { margin-bottom: 24px; }
.perf-title { display: flex; flex-direction: column; gap: 4px; }
.perf-title .ttl {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}
.perf-delta {
  font-size: 13px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  margin-left: 4px;
}
.perf-delta.up { color: var(--pos); }
.perf-delta.down { color: var(--neg); }
.perf-range {
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 12px;
  margin-left: 6px;
}
.perf-controls { display: flex; align-items: center; gap: 14px; }
.perf-body { padding: 18px 18px 14px; position: relative; }
.perf-canvas-wrap { position: relative; height: 360px; width: 100%; }
.perf-summary {
  display: flex;
  gap: 28px;
  padding: 14px 18px 18px;
  border-top: 1px solid var(--border-soft);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}
.perf-summary .item .lbl {
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.perf-summary .item .val {
  font-size: 13px;
  color: var(--text);
  margin-top: 4px;
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.perf-summary .item .val .chg { font-size: 11px; }
.perf-summary .item .val .up { color: var(--pos); }
.perf-summary .item .val .down { color: var(--neg); }

/* ============================================
   Two-column row
   ============================================ */
.row-2col {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 380px;
  gap: 24px;
  margin-bottom: 24px;
}
@media (max-width: 1100px) {
  .row-2col { grid-template-columns: 1fr; }
}

/* ============================================
   Tables
   ============================================ */
.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.table thead th {
  text-align: left;
  padding: 10px 16px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  font-weight: 600;
  color: var(--text-3);
  background: var(--sunken);
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}
.table thead th.num { text-align: right; }
.table thead th.center { text-align: center; }
.table thead th .sort {
  display: inline-block;
  margin-left: 4px;
  color: var(--text-4);
  font-size: 9px;
  vertical-align: 1px;
}
.table thead th.sorted { color: var(--text); }
.table thead th.sorted .sort { color: var(--brand); }
.table thead th:hover { color: var(--text-2); }
.table thead th.caret-th { cursor: default; width: 40px; }
.table tbody td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-soft);
  font-variant-numeric: tabular-nums;
}
.table tbody td.num { text-align: right; font-family: var(--font-mono); }
.table tbody td.center { text-align: center; }
.table tbody td.mono { font-family: var(--font-mono); }

.row-main { cursor: pointer; transition: background 120ms; }
.row-main:hover { background: var(--overlay); }
.row-main.open { background: var(--overlay); }
.row-main.open td { border-bottom-color: transparent; }

.row-expand td {
  padding: 0;
  background: var(--sunken);
  border-bottom: 1px solid var(--border-soft);
}
.expand-inner {
  padding: 16px 22px 18px;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 18px 28px;
}
.expand-inner .kv .k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 4px;
  font-weight: 600;
}
.expand-inner .kv .v {
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.expand-inner .timeline {
  grid-column: 1 / -1;
  border-top: 1px solid var(--border-soft);
  padding-top: 14px;
  margin-top: 4px;
}
.expand-inner .timeline .ttl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 8px;
}
.fill-row {
  display: grid;
  grid-template-columns: 110px 70px 90px 90px 1fr 90px;
  gap: 16px;
  padding: 6px 0;
  font-family: var(--font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: var(--text-2);
  border-bottom: 1px dashed var(--border-soft);
}
.fill-row:last-child { border-bottom: 0; }
.fill-row .ts { color: var(--text-3); }
.fill-row .px { color: var(--text); }
.fill-row .right { text-align: right; }

/* Market cell */
.mkt { display: flex; align-items: center; gap: 10px; }
.mkt .sym-square {
  width: 24px; height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--text);
  border: 1px solid var(--border-strong);
}
.mkt .sym-square.lime   { background: var(--brand);  color: var(--canvas); border-color: var(--brand); }
.mkt .sym-square.blue   { background: var(--info);   color: #fff;          border-color: var(--info); }
.mkt .sym-square.orange { background: #F5A524;       color: var(--canvas); border-color: #F5A524; }
.mkt .info { display: flex; flex-direction: column; line-height: 1.25; }
.mkt .name {
  font-family: var(--font-mono);
  font-size: 12.5px;
  letter-spacing: 0.02em;
  color: var(--text);
  font-weight: 500;
}
.mkt .closed-tag { color: var(--text-3); font-weight: 400; letter-spacing: 0; margin-left: 4px; }
.mkt .sub { font-family: var(--font-sans); font-size: 11px; color: var(--text-3); }

.side-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.side-tag.long  { background: var(--pos-soft); color: var(--pos); }
.side-tag.short { background: var(--neg-soft); color: var(--neg); }
.side-tag .arr { font-size: 9px; }
.side-tag.micro { padding: 1px 6px; font-size: 9px; }

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  color: var(--text-3);
  border: 1px solid var(--border);
}
.status-tag.closed { color: var(--text-3); }
.status-tag.pos-status { color: var(--pos); border-color: rgba(25, 195, 125, 0.3); }
.status-tag .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }

.pnl-up   { color: var(--pos); }
.pnl-down { color: var(--neg); }
.pnl-flat { color: var(--text); }
.tiny-arr { font-size: 10px; margin-right: 4px; }
.sm-unit { font-size: 11px; }

/* Position tab filter */
.tab-filter {
  display: inline-flex;
  border-bottom: 1px solid var(--border);
  gap: 0;
  margin: -14px -18px 0;
  padding: 0 18px;
}
.tab-filter .tf {
  background: transparent;
  border: 0;
  color: var(--text-3);
  padding: 12px 14px;
  font-family: var(--font-sans);
  font-size: 12.5px;
  cursor: pointer;
  position: relative;
  letter-spacing: -0.005em;
}
.tab-filter .tf .count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-4);
  margin-left: 6px;
  letter-spacing: 0.04em;
}
.tab-filter .tf:hover { color: var(--text-2); }
.tab-filter .tf.active { color: var(--text); }
.tab-filter .tf.active .count { color: var(--text-2); }
.tab-filter .tf.active::after {
  content: '';
  position: absolute;
  left: 14px;
  right: 14px;
  bottom: -1px;
  height: 2px;
  background: var(--brand);
}

.caret {
  width: 14px;
  height: 14px;
  display: inline-grid;
  place-items: center;
  color: var(--text-3);
  transition: transform 160ms;
}
.row-main.open .caret { transform: rotate(90deg); color: var(--text); }
.caret svg { width: 10px; height: 10px; stroke: currentColor; fill: none; stroke-width: 1.6; }

/* ============================================
   Allocation card
   ============================================ */
.alloc-card { padding: 0; }
.alloc-body {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.donut-wrap {
  position: relative;
  width: 100%;
  height: 200px;
  display: grid;
  place-items: center;
}
.donut-center {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  pointer-events: none;
  text-align: center;
}
.donut-center .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.donut-center .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 20px;
  color: var(--text);
  font-weight: 500;
  margin-top: 4px;
}
.donut-center .pct {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  margin-top: 2px;
}

.legend {
  display: flex;
  flex-direction: column;
  gap: 1px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.legend-row {
  display: grid;
  grid-template-columns: 14px 1fr auto auto;
  gap: 12px;
  padding: 9px 12px;
  align-items: center;
  cursor: pointer;
  font-size: 12.5px;
  transition: background 120ms;
  border-bottom: 1px solid var(--border-soft);
}
.legend-row:last-child { border-bottom: 0; }
.legend-row:hover,
.legend-row.hovered { background: var(--overlay); }
.legend-row .swatch { width: 10px; height: 10px; border-radius: 1px; }
.legend-row .name { color: var(--text); font-family: var(--font-sans); }
.legend-row .pct {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  font-variant-numeric: tabular-nums;
}
.legend-row .val {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  min-width: 70px;
  text-align: right;
}

.section-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.section-eyebrow::before { content: '—'; color: var(--text-4); }

.asset-bar {
  height: 16px;
  display: flex;
  overflow: hidden;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  background: var(--canvas);
}
.asset-bar .seg {
  height: 100%;
  position: relative;
  cursor: pointer;
  transition: filter 120ms;
}
.asset-bar .seg:hover { filter: brightness(1.15); }
.asset-bar .seg + .seg { border-left: 1px solid var(--canvas); }

.asset-legend {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 16px;
  margin-top: 12px;
  font-size: 12px;
}
.asset-legend .row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.asset-legend .name {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-2);
  font-family: var(--font-sans);
}
.asset-legend .swatch { width: 8px; height: 8px; }
.asset-legend .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text);
  font-size: 12px;
}

/* ============================================
   Activity
   ============================================ */
.activity-card .card-head a {
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  text-decoration: none;
  letter-spacing: 0;
}
.activity-card .card-head a:hover { color: var(--text); }
.ts-col { width: 170px; }
.mkt-name { font-size: 12.5px; letter-spacing: 0.02em; }
.type-cell {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-2);
}
.settle {
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.04em;
}

/* ── Mobile: stack the expanded-holding detail grid; wide fill rows scroll within the panel. ── */
@media (max-width: 640px) {
  .expand-inner { grid-template-columns: 1fr 1fr; gap: 12px 18px; }
  .fill-row { min-width: 520px; }
}
@media (max-width: 480px) {
  .expand-inner { grid-template-columns: 1fr; }
}
</style>
