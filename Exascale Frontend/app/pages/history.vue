<script setup lang="ts">
/**
 * /history — Trade history (C4).
 *
 * Dark, in-app. Dense Bloomberg-tier table with:
 *   - 4-stat summary header
 *   - Sticky filter bar (date / market / type / side / search)
 *   - Expandable rows showing audit hashes + fills + open-in-detail
 *   - Cryptographic-signed CSV / JSON / PDF exports
 *   - Pagination
 *   - Date dividers between days
 */
import { ChevronDown, ChevronRight, Copy, CheckCircle2, FileDown, Search, X, Download, ShieldCheck } from 'lucide-vue-next'

definePageMeta({ layout: 'app' })
useHead({ title: 'Trade History — Exascale' })

const toasts = useToasts()
const od = useOrderDetail()

// ─── Types ──────────────────────────────────────────────────
type RowType  = 'Trade' | 'Conversion' | 'Deposit' | 'Withdrawal' | 'Fee'
type Side     = 'buy' | 'sell' | '—'
type Result   = 'success' | 'fail'

interface Fill {
  ts: string
  qty: number
  px: number
  fee: number
  counterparty: string
}
interface Row {
  id: string                  // ord_… or tx_…
  ts: string                  // ISO with ms
  type: RowType
  market: string
  side: Side
  qty: number
  price: number
  total: number
  fee: number
  pnl?: number
  result: Result
  ip: string
  block: number
  hash: string
  fills?: Fill[]
}

// ─── Mock data (20 rows — mix of trades, 1 conversion, 1 deposit, 1 withdrawal) ──
const rows: Row[] = [
  { id: 'ord_8c2a48f1', ts: '2026-05-24T14:31:55.108Z', type: 'Trade',      market: 'EAI-IDX',    side: 'buy',  qty: 5000,  price: 0.001005, total: 5.025,  fee: 0.05,  pnl:  +0.62, result: 'success', ip: '198.51.100.42', block: 14287, hash: '7a3f9c1e021', fills: [
    { ts: '14:31:55.041', qty: 1800, px: 0.001005, fee: 0.018, counterparty: 'mm_a14c' },
    { ts: '14:31:55.071', qty: 2400, px: 0.001005, fee: 0.024, counterparty: 'mm_b22e' },
    { ts: '14:31:55.108', qty:  800, px: 0.001006, fee: 0.008, counterparty: 'mm_a14c' },
  ] },
  { id: 'ord_2c8e91a1', ts: '2026-05-24T10:18:44.331Z', type: 'Trade',      market: 'TEXT-SPOT',  side: 'sell', qty: 8000,  price: 0.001210, total: 9.680,  fee: 0.097, pnl:  +1.20, result: 'success', ip: '198.51.100.42', block: 14273, hash: 'ad08c46e2f0' },
  { id: 'ord_5d4b22e0', ts: '2026-05-24T09:42:17.502Z', type: 'Trade',      market: 'H100-SPOT',  side: 'buy',  qty: 8,     price: 2.99,     total: 23.92,  fee: 0.239, pnl:  +0.32, result: 'success', ip: '198.51.100.42', block: 14268, hash: '80f4ad81ba8' },
  { id: 'tx_b341a2c',   ts: '2026-05-24T13:09:47.612Z', type: 'Deposit',    market: '—',          side: '—',    qty: 100000,price: 1,        total: 100000, fee: 15.00, result: 'success', ip: '192.0.2.91',   block: 14280, hash: 'b341a2ce0a4' },
  { id: 'tx_9c1ab402',  ts: '2026-05-23T16:55:08.220Z', type: 'Conversion', market: 'AI→TEXT',    side: '—',    qty: 1500,  price: 1.198,    total: 1.797,  fee: 0.018, result: 'success', ip: '198.51.100.42', block: 14220, hash: '9c1ab402df1' },
  { id: 'ord_6e2fa033', ts: '2026-05-23T14:22:39.880Z', type: 'Trade',      market: 'IMAGE-SPOT', side: 'buy',  qty: 300,   price: 0.008000, total: 2.400,  fee: 0.024, pnl:  -0.18, result: 'success', ip: '203.0.113.42',  block: 14201, hash: '6e2fa033c89' },
  { id: 'ord_91ee0a4b', ts: '2026-05-23T12:08:01.110Z', type: 'Trade',      market: 'EAI-IDX',    side: 'sell', qty: 3200,  price: 0.001002, total: 3.206,  fee: 0.032, pnl:  +0.41, result: 'success', ip: '198.51.100.42', block: 14188, hash: '91ee0a4b0a2' },
  { id: 'ord_44caea91', ts: '2026-05-23T11:01:55.620Z', type: 'Trade',      market: 'SPEECH-SPOT',side: 'buy',  qty: 2500,  price: 0.001200, total: 3.000,  fee: 0.030, pnl:  -0.04, result: 'success', ip: '198.51.100.42', block: 14179, hash: '44caea91d04' },
  { id: 'ord_22d011ee', ts: '2026-05-23T09:14:08.041Z', type: 'Trade',      market: 'H100-SPOT',  side: 'sell', qty: 4,     price: 2.97,     total: 11.88,  fee: 0.119, pnl:  +0.18, result: 'success', ip: '198.51.100.42', block: 14163, hash: '22d011ee7f3' },
  { id: 'ord_77af2918', ts: '2026-05-22T18:42:33.997Z', type: 'Trade',      market: 'EAI-IDX',    side: 'buy',  qty: 12000, price: 0.000998, total: 11.976, fee: 0.120, pnl:  +0.84, result: 'success', ip: '198.51.100.42', block: 14122, hash: '77af2918bc6' },
  { id: 'ord_18c0fa19', ts: '2026-05-22T15:55:12.080Z', type: 'Trade',      market: 'VIDEO-SPOT', side: 'buy',  qty: 8,     price: 0.250,    total: 2.000,  fee: 0.020, pnl:  +0.04, result: 'success', ip: '203.0.113.42',  block: 14104, hash: '18c0fa19df8' },
  { id: 'ord_bb39028f', ts: '2026-05-22T13:27:48.770Z', type: 'Trade',      market: 'TEXT-SPOT',  side: 'sell', qty: 14000, price: 0.001195, total: 16.730, fee: 0.167, pnl:  -0.12, result: 'success', ip: '198.51.100.42', block: 14086, hash: 'bb39028f102' },
  { id: 'ord_aa12_fail',ts: '2026-05-22T11:08:09.220Z', type: 'Trade',      market: 'EAI-IDX',    side: 'buy',  qty: 100000,price: 0.001004, total: 100.400,fee: 0,     result: 'fail',    ip: '198.51.100.42', block: 14074, hash: 'aa12faila01' },
  { id: 'tx_with_wmt',  ts: '2026-05-22T09:30:00.000Z', type: 'Withdrawal', market: '—',          side: '—',    qty: 5000,  price: 1,        total: 5000,   fee: 15.00, result: 'success', ip: '192.0.2.91',   block: 14060, hash: 'fa92ce30210' },
  { id: 'ord_0ec1d829', ts: '2026-05-21T16:00:14.300Z', type: 'Trade',      market: 'EAI-IDX',    side: 'sell', qty: 6800,  price: 0.001001, total: 6.807,  fee: 0.068, pnl:  +0.27, result: 'success', ip: '198.51.100.42', block: 13988, hash: '0ec1d829e44' },
  { id: 'ord_5f47120c', ts: '2026-05-21T13:48:55.610Z', type: 'Trade',      market: 'H100-SPOT',  side: 'buy',  qty: 12,    price: 2.96,     total: 35.52,  fee: 0.355, pnl:  +0.62, result: 'success', ip: '198.51.100.42', block: 13967, hash: '5f47120c980' },
  { id: 'tx_fee_admin', ts: '2026-05-21T10:00:00.000Z', type: 'Fee',        market: '—',          side: '—',    qty: 1,     price: 9.99,     total: 9.99,   fee: 0,     result: 'success', ip: '—',            block: 13941, hash: 'cc01febbb84' },
  { id: 'ord_3da9c8a0', ts: '2026-05-20T15:14:01.900Z', type: 'Trade',      market: 'TEXT-SPOT',  side: 'buy',  qty: 9500,  price: 0.001188, total: 11.286, fee: 0.113, pnl:  +0.31, result: 'success', ip: '198.51.100.42', block: 13880, hash: '3da9c8a0bf7' },
  { id: 'ord_7748ab12', ts: '2026-05-20T11:38:09.220Z', type: 'Trade',      market: 'EAI-IDX',    side: 'buy',  qty: 22000, price: 0.000999, total: 21.978, fee: 0.220, pnl:  +1.10, result: 'success', ip: '198.51.100.42', block: 13851, hash: '7748ab1290e' },
  { id: 'ord_e0c1448a', ts: '2026-05-20T09:02:48.412Z', type: 'Trade',      market: 'IMAGE-SPOT', side: 'sell', qty: 800,   price: 0.008012, total: 6.410,  fee: 0.064, pnl:  +0.22, result: 'success', ip: '198.51.100.42', block: 13832, hash: 'e0c1448aa10' },
]

// ─── Filters ────────────────────────────────────────────────
const TYPES: RowType[] = ['Trade', 'Conversion', 'Deposit', 'Withdrawal', 'Fee']
const DATE_RANGES = ['Last 7 days', 'Last 30 days', 'Last 90 days', 'YTD 2026', 'Custom…']
const MARKETS = ['All markets', 'EAI-IDX', 'TEXT-SPOT', 'SPEECH-SPOT', 'IMAGE-SPOT', 'VIDEO-SPOT', 'H100-SPOT', 'H200-SPOT']
const SIDE_FILTERS: { key: 'all' | 'buy' | 'sell'; label: string }[] = [
  { key: 'all',  label: 'All' },
  { key: 'buy',  label: 'Buy' },
  { key: 'sell', label: 'Sell' },
]
const RESULT_FILTERS: { key: 'all' | 'success' | 'fail'; label: string }[] = [
  { key: 'all',     label: 'All' },
  { key: 'success', label: 'Success' },
  { key: 'fail',    label: 'Fail' },
]

const fDate    = ref('Last 30 days')
const fMarket  = ref('All markets')
const fType    = ref<'all' | RowType>('all')
const fSide    = ref<'all' | 'buy' | 'sell'>('all')
const fResult  = ref<'all' | 'success' | 'fail'>('all')
const fQuery   = ref('')

const filtered = computed(() => {
  const q = fQuery.value.trim().toLowerCase()
  return rows.filter((r) => {
    if (fMarket.value !== 'All markets' && r.market !== fMarket.value) return false
    if (fType.value   !== 'all' && r.type !== fType.value)              return false
    if (fSide.value   !== 'all' && r.side !== fSide.value)              return false
    if (fResult.value !== 'all' && r.result !== fResult.value)          return false
    if (q && !(r.id.toLowerCase().includes(q) || r.market.toLowerCase().includes(q) || r.hash.includes(q))) return false
    return true
  })
})

function resetFilters() {
  fDate.value   = 'Last 30 days'
  fMarket.value = 'All markets'
  fType.value   = 'all'
  fSide.value   = 'all'
  fResult.value = 'all'
  fQuery.value  = ''
}

const filtersActive = computed(() =>
  fMarket.value !== 'All markets' || fType.value !== 'all' || fSide.value !== 'all' ||
  fResult.value !== 'all' || fQuery.value.length > 0,
)

// ─── Summary stats (over visible rows) ──────────────────────
const stats = computed(() => {
  const trades = filtered.value.filter((r) => r.type === 'Trade' && r.result === 'success')
  return {
    count:    trades.length,
    volume:   trades.reduce((s, r) => s + r.total, 0),
    fees:     filtered.value.reduce((s, r) => s + r.fee, 0),
    realized: trades.reduce((s, r) => s + (r.pnl ?? 0), 0),
  }
})

// ─── Pagination (visual only — 20 rows fit on one page) ─────
const PAGE_SIZE = 20
const page = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))
const visibleRows = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return filtered.value.slice(start, start + PAGE_SIZE)
})

// ─── Date dividers ──────────────────────────────────────────
interface RenderItem {
  kind: 'divider' | 'row'
  date?: string
  row?: Row
}
const rendered = computed<RenderItem[]>(() => {
  const out: RenderItem[] = []
  let lastDate = ''
  for (const r of visibleRows.value) {
    const d = r.ts.slice(0, 10)
    if (d !== lastDate) {
      out.push({ kind: 'divider', date: d })
      lastDate = d
    }
    out.push({ kind: 'row', row: r })
  }
  return out
})

// ─── Row expansion ──────────────────────────────────────────
const expandedId = ref<string | null>(null)
function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

// ─── Copy / actions ─────────────────────────────────────────
const copied = ref<string | null>(null)
function copy(s: string, key: string, ev?: Event) {
  if (ev) ev.stopPropagation()
  if (navigator.clipboard) navigator.clipboard.writeText(s).catch(() => {})
  copied.value = key
  setTimeout(() => { if (copied.value === key) copied.value = null }, 1200)
}

function openOrderDetail(row: Row, ev?: Event) {
  if (ev) ev.stopPropagation()
  if (row.type !== 'Trade') return
  od.open(row.id)
}

function doExport(kind: 'csv' | 'json' | 'pdf') {
  toasts.push({
    tone:  'info',
    title: 'Export prepared',
    body:  kind.toUpperCase() + ' export of ' + filtered.value.length + ' rows · signed with venue key kid_8a91… · ready to download.',
  })
}

// ─── Formatters ─────────────────────────────────────────────
function fmtTime(ts: string): string { return ts.slice(11, 19) }
function fmtDate(d: string): string {
  const date = new Date(d + 'T00:00:00Z')
  return date.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
}
function fmtPx(p: number, m: string): string {
  if (m === 'H100-SPOT' || m === 'H200-SPOT' || m === 'VIDEO-SPOT' || m === '—') return p.toFixed(p < 1 ? 6 : 2)
  return p.toFixed(6)
}
function fmtQty(n: number): string {
  return n.toLocaleString('en-US')
}
function fmtUSD(n: number, opts: { cents?: boolean; sign?: boolean } = {}): string {
  const sign = opts.sign && n > 0 ? '+' : ''
  return sign + '$' + n.toLocaleString('en-US', {
    minimumFractionDigits: opts.cents === false ? 0 : 2,
    maximumFractionDigits: opts.cents === false ? 0 : 2,
  })
}
function fmtPnL(n: number | undefined): string {
  if (n === undefined) return '—'
  return (n >= 0 ? '+$' : '-$') + Math.abs(n).toFixed(2)
}
function typeTone(t: RowType): string {
  if (t === 'Trade')      return 'neutral'
  if (t === 'Conversion') return 'info'
  if (t === 'Deposit')    return 'pos'
  if (t === 'Withdrawal') return 'warn'
  return 'mute'
}
</script>

<template>
  <div class="history-page">
    <!-- Head -->
    <header class="hd">
      <div class="hd-left">
        <h1>Trade History</h1>
        <p class="hd-sub">
          <span class="mono">{{ filtered.length }}</span> events
          <span v-if="filtersActive" class="dim">· filtered</span>
          <span class="dim">·</span>
          <span class="mono">{{ stats.count }}</span> trades
          <span class="dim">·</span>
          <span class="mono">{{ fmtUSD(stats.volume) }}</span> volume
        </p>
      </div>
      <div class="hd-acts">
        <button class="btn ghost" type="button" @click="doExport('csv')">
          <FileDown :size="13" />
          Export CSV
        </button>
        <button class="btn ghost" type="button" @click="doExport('json')">
          <FileDown :size="13" />
          Export JSON
        </button>
        <button class="btn primary" type="button" @click="doExport('pdf')">
          <ShieldCheck :size="13" />
          Compliance report
        </button>
      </div>
    </header>

    <!-- Stat strip -->
    <section class="stats">
      <div class="stat">
        <div class="lbl">Total trades</div>
        <div class="val mono">{{ stats.count.toLocaleString() }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Total volume</div>
        <div class="val mono">{{ fmtUSD(stats.volume) }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Total fees</div>
        <div class="val mono">{{ fmtUSD(stats.fees) }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Realized P&amp;L</div>
        <div class="val mono" :class="stats.realized >= 0 ? 'pos' : 'neg'">
          {{ fmtPnL(stats.realized) }}
        </div>
      </div>
    </section>

    <!-- Filter bar (sticky) -->
    <section class="filters">
      <div class="f">
        <label class="f-lbl">Date</label>
        <select v-model="fDate" class="f-input">
          <option v-for="d in DATE_RANGES" :key="d" :value="d">{{ d }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-lbl">Market</label>
        <select v-model="fMarket" class="f-input mono">
          <option v-for="m in MARKETS" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-lbl">Type</label>
        <select v-model="fType" class="f-input">
          <option value="all">All types</option>
          <option v-for="t in TYPES" :key="t" :value="t">{{ t }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-lbl">Side</label>
        <div class="chips">
          <button v-for="s in SIDE_FILTERS" :key="s.key" type="button" class="chip" :class="{ on: fSide === s.key }" @click="fSide = s.key">{{ s.label }}</button>
        </div>
      </div>
      <div class="f">
        <label class="f-lbl">Result</label>
        <div class="chips">
          <button v-for="r in RESULT_FILTERS" :key="r.key" type="button" class="chip" :class="{ on: fResult === r.key }" @click="fResult = r.key">{{ r.label }}</button>
        </div>
      </div>
      <div class="f f-grow">
        <label class="f-lbl">Search</label>
        <div class="search-wrap">
          <Search :size="13" class="search-icon" />
          <input v-model="fQuery" type="text" class="f-input search" placeholder="order id, market, audit hash…" />
          <button v-if="fQuery" type="button" class="clear" aria-label="Clear search" @click="fQuery = ''">
            <X :size="12" />
          </button>
        </div>
      </div>
      <div class="f">
        <button v-if="filtersActive" type="button" class="reset-btn" @click="resetFilters">Reset filters</button>
      </div>
    </section>

    <!-- Table -->
    <section class="tbl-wrap">
      <table class="tbl">
        <thead>
          <tr>
            <th class="chev"></th>
            <th class="th-time">Time (UTC)</th>
            <th>Type</th>
            <th>Market</th>
            <th>Side</th>
            <th class="r">Qty</th>
            <th class="r">Price</th>
            <th class="r">Total</th>
            <th class="r">Fee</th>
            <th class="r">P&amp;L</th>
            <th>Result</th>
            <th>Order ID</th>
            <th>Audit hash</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(item, i) in rendered" :key="i">
            <tr v-if="item.kind === 'divider'" class="divider-row">
              <td colspan="13">
                <span class="div-date">{{ fmtDate(item.date!) }}</span>
              </td>
            </tr>
            <template v-else>
              <tr
                class="row"
                :class="{ open: expandedId === item.row!.id, fail: item.row!.result === 'fail' }"
                @click="toggleExpand(item.row!.id)"
              >
                <td class="chev">
                  <ChevronRight v-if="expandedId !== item.row!.id" :size="11" />
                  <ChevronDown  v-else :size="11" />
                </td>
                <td class="mono ts">{{ fmtTime(item.row!.ts) }}</td>
                <td>
                  <span class="type" :class="'t-' + typeTone(item.row!.type)">{{ item.row!.type }}</span>
                </td>
                <td class="mono mkt">{{ item.row!.market }}</td>
                <td>
                  <span v-if="item.row!.side !== '—'" class="side mono" :class="item.row!.side">{{ item.row!.side.toUpperCase() }}</span>
                  <span v-else class="dim">—</span>
                </td>
                <td class="r mono">{{ fmtQty(item.row!.qty) }}</td>
                <td class="r mono">{{ item.row!.market === '—' ? '—' : fmtPx(item.row!.price, item.row!.market) }}</td>
                <td class="r mono">{{ fmtUSD(item.row!.total) }}</td>
                <td class="r mono dim">{{ item.row!.fee ? fmtUSD(item.row!.fee) : '—' }}</td>
                <td class="r mono" :class="item.row!.pnl !== undefined ? (item.row!.pnl >= 0 ? 'pos' : 'neg') : ''">
                  {{ fmtPnL(item.row!.pnl) }}
                </td>
                <td>
                  <span class="result" :class="item.row!.result">
                    <span class="r-dot" />{{ item.row!.result === 'success' ? 'OK' : 'FAIL' }}
                  </span>
                </td>
                <td class="mono dim id">{{ item.row!.id }}</td>
                <td class="hash-cell">
                  <span class="mono dim">{{ item.row!.hash }}…</span>
                  <button class="copy" type="button" :title="'Copy ' + item.row!.hash" @click="copy(item.row!.hash, item.row!.id, $event)">
                    <Copy v-if="copied !== item.row!.id" :size="10" />
                    <CheckCircle2 v-else :size="10" />
                  </button>
                </td>
              </tr>

              <!-- Expanded detail row -->
              <tr v-if="expandedId === item.row!.id" class="detail-row">
                <td colspan="13">
                  <div class="detail">
                    <div class="d-left">
                      <div class="d-eyebrow">Audit chain</div>
                      <dl class="kv">
                        <dt>Block</dt><dd class="mono">#{{ item.row!.block.toLocaleString() }}</dd>
                        <dt>Hash</dt><dd class="mono">{{ item.row!.hash }}3e4a7c12d9f0…</dd>
                        <dt>Algorithm</dt><dd class="mono">SHA-256 · ed25519</dd>
                        <dt>IP origin</dt><dd class="mono">{{ item.row!.ip }}</dd>
                        <dt>Event ID</dt><dd class="mono">{{ item.row!.id }}</dd>
                      </dl>
                      <NuxtLink :to="'/enterprise/audit?hash=' + item.row!.hash" class="audit-link">
                        Verify in audit log →
                      </NuxtLink>
                    </div>

                    <div class="d-right">
                      <div class="d-eyebrow">{{ item.row!.fills ? 'Fills · ' + item.row!.fills.length : 'Event detail' }}</div>
                      <table v-if="item.row!.fills" class="fills">
                        <thead>
                          <tr><th>Time</th><th class="r">Qty</th><th class="r">Price</th><th class="r">Fee</th><th>Counterparty</th></tr>
                        </thead>
                        <tbody>
                          <tr v-for="(f, idx) in item.row!.fills" :key="idx">
                            <td class="mono">{{ f.ts }}</td>
                            <td class="r mono">{{ fmtQty(f.qty) }}</td>
                            <td class="r mono">{{ fmtPx(f.px, item.row!.market) }}</td>
                            <td class="r mono">{{ fmtUSD(f.fee) }}</td>
                            <td class="mono dim">{{ f.counterparty }}</td>
                          </tr>
                        </tbody>
                      </table>
                      <p v-else class="d-note">
                        Single-event {{ item.row!.type.toLowerCase() }} —
                        <span v-if="item.row!.type === 'Deposit' || item.row!.type === 'Withdrawal'">settled to bank ••4421.</span>
                        <span v-else-if="item.row!.type === 'Conversion'">venue book sweep · no counterparty match.</span>
                        <span v-else>see audit chain for full payload.</span>
                      </p>
                      <div class="d-acts">
                        <button v-if="item.row!.type === 'Trade'" type="button" class="btn ghost sm" @click="openOrderDetail(item.row!, $event)">
                          Open order detail →
                        </button>
                        <button type="button" class="btn ghost sm">
                          <Download :size="11" />
                          Download row receipt
                        </button>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </template>
        </tbody>
      </table>
    </section>

    <!-- Pager -->
    <footer class="pager">
      <div class="pg-left">
        Showing <span class="mono">{{ visibleRows.length }}</span> of <span class="mono">{{ filtered.length }}</span> events
        <span class="dim">· signed exports include the full chain proof for this filter</span>
      </div>
      <div class="pg-right">
        <button class="pg-btn" type="button" :disabled="page <= 1" @click="page--">← Newer</button>
        <span class="pg-page mono">Page {{ page }} / {{ totalPages }}</span>
        <button class="pg-btn" type="button" :disabled="page >= totalPages" @click="page++">Older →</button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.history-page {
  padding: 20px 24px 32px;
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 12.5px;
}
.history-page .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.history-page .dim  { color: var(--text-3); }
.history-page .pos  { color: var(--pos); font-weight: 600; }
.history-page .neg  { color: var(--neg); font-weight: 600; }

/* Head */
.hd {
  display: flex; justify-content: space-between; align-items: flex-end;
  margin-bottom: 16px;
}
.hd h1 {
  font-family: var(--font-display); font-weight: 600;
  font-size: 26px; letter-spacing: -0.015em; margin: 0 0 4px;
}
.hd-sub { color: var(--text-2); font-size: 12.5px; margin: 0; }
.hd-acts { display: inline-flex; gap: 6px; }

.btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 7px 12px;
  font-family: var(--font-sans); font-size: 12px; font-weight: 500;
  background: transparent; color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease;
}
.btn:hover:not(:disabled) { background: rgba(255,255,255,0.04); border-color: var(--text); }
.btn:disabled { color: var(--text-3); cursor: default; opacity: 0.5; }
.btn.primary { background: var(--brand); color: var(--text-on-accent); border-color: var(--brand); font-weight: 600; }
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.sm { padding: 5px 9px; font-size: 11.5px; }

/* Stats strip */
.stats {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--border);
  border: 1px solid var(--border);
  margin-bottom: 12px;
}
.stat { background: var(--elevated); padding: 12px 16px; }
.lbl {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 4px;
}
.val { font-size: 22px; font-weight: 600; color: var(--text); letter-spacing: -0.01em; }

/* Filter bar */
.filters {
  position: sticky; top: 0; z-index: 5;
  display: flex; align-items: flex-end; gap: 10px;
  padding: 12px 14px;
  background: var(--elevated);
  border: 1px solid var(--border);
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.f { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.f-grow { flex: 1; min-width: 200px; }
.f-lbl {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
}
.f-input {
  height: 30px;
  padding: 0 10px;
  font-family: var(--font-sans); font-size: 12px; color: var(--text);
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  outline: none;
  min-width: 130px;
}
.f-input.mono { font-family: var(--font-mono); }
.f-input:focus { border-color: var(--accent); }

.search-wrap { position: relative; }
.search-icon { position: absolute; left: 8px; top: 50%; transform: translateY(-50%); color: var(--text-3); }
.search { padding-left: 26px; padding-right: 26px; width: 100%; }
.clear {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  width: 18px; height: 18px;
  background: transparent; border: none; color: var(--text-3);
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: var(--radius-sm); cursor: pointer;
}
.clear:hover { color: var(--text); background: rgba(255,255,255,0.06); }

.chips { display: inline-flex; gap: 3px; }
.chip {
  height: 30px; padding: 0 10px;
  font-family: var(--font-sans); font-size: 11.5px; font-weight: 500;
  background: var(--canvas); color: var(--text-2);
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
  cursor: pointer;
}
.chip:hover { color: var(--text); border-color: var(--text); }
.chip.on    { background: var(--text); color: var(--canvas); border-color: var(--text); }

.reset-btn {
  height: 30px; padding: 0 10px;
  background: transparent; color: var(--accent);
  border: 1px solid var(--accent);
  border-radius: var(--radius-sm);
  font-family: var(--font-sans); font-size: 11.5px; font-weight: 600;
  cursor: pointer;
}
.reset-btn:hover { background: rgba(74,144,226,0.10); }

/* Table */
.tbl-wrap {
  background: var(--elevated);
  border: 1px solid var(--border);
  overflow-x: auto;
}
.tbl { width: 100%; border-collapse: collapse; font-size: 12px; }
.tbl thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3);
  padding: 10px 10px;
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
  white-space: nowrap;
  position: sticky; top: 0;
}
.tbl th.r { text-align: right; }
.tbl th.chev, .tbl td.chev { width: 22px; padding-left: 14px; }
.tbl th.th-time { width: 88px; }

.tbl tbody td {
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
  white-space: nowrap;
}
.tbl tbody td.r { text-align: right; }

.row { cursor: pointer; transition: background-color 80ms ease; }
.row:hover { background: rgba(255,255,255,0.025); }
.row.open  { background: rgba(74,144,226,0.06); }
.row.fail .ts { color: var(--neg); }
.row td.chev { color: var(--text-3); }

.divider-row td {
  background: var(--canvas);
  padding: 6px 16px !important;
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: 0.16em; color: var(--text-3);
  border-bottom: 1px solid var(--border);
}
.div-date { text-transform: uppercase; }

.ts { color: var(--text); font-size: 11.5px; }
.type {
  display: inline-block;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: 10px; font-weight: 600;
  letter-spacing: 0.06em;
}
.type.t-neutral { background: rgba(255,255,255,0.06); color: var(--text); }
.type.t-info    { background: rgba(74,144,226,0.14); color: var(--info); }
.type.t-pos     { background: var(--pos-soft); color: var(--pos); }
.type.t-warn    { background: rgba(245,158,11,0.16); color: var(--warn); }
.type.t-mute    { background: rgba(255,255,255,0.04); color: var(--text-2); }

.mkt { color: var(--text); font-weight: 600; font-size: 11.5px; letter-spacing: 0.04em; }

.side {
  display: inline-block;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: 9.5px; font-weight: 700; letter-spacing: 0.14em;
}
.side.buy  { background: var(--pos-soft); color: var(--pos); }
.side.sell { background: var(--neg-soft); color: var(--neg); }

.result {
  display: inline-flex; align-items: center; gap: 4px;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.1em;
}
.result.success { color: var(--pos); }
.result.fail    { color: var(--neg); }
.result .r-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }

.id   { font-size: 11px; }
.hash-cell { display: flex; align-items: center; gap: 6px; }
.copy {
  width: 18px; height: 18px;
  background: var(--canvas); color: var(--text-2);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.copy:hover { color: var(--text); border-color: var(--text); }

/* Expanded detail row */
.detail-row td { padding: 0 !important; background: var(--canvas); }
.detail {
  display: grid; grid-template-columns: 1fr 1.4fr; gap: 20px;
  padding: 14px 20px 18px;
  border-top: 1px dashed var(--border);
  border-bottom: 1px solid var(--border);
}
.d-eyebrow {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3); margin-bottom: 10px;
}
.kv {
  display: grid; grid-template-columns: 110px 1fr;
  row-gap: 5px; column-gap: 12px; margin: 0;
  font-size: 11.5px;
}
.kv dt { font-family: var(--font-mono); font-size: 10px; letter-spacing: 0.06em; text-transform: uppercase; color: var(--text-3); }
.kv dd { margin: 0; color: var(--text); word-break: break-all; }

.audit-link {
  display: inline-block; margin-top: 10px;
  font-size: 11.5px; color: var(--accent); text-decoration: none;
}
.audit-link:hover { text-decoration: underline; }

.fills { width: 100%; border-collapse: collapse; font-size: 11.5px; }
.fills thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9px; font-weight: 600;
  letter-spacing: 0.12em; color: var(--text-3);
  padding: 4px 6px;
  border-bottom: 1px solid var(--border);
}
.fills th.r, .fills td.r { text-align: right; }
.fills tbody td { padding: 4px 6px; border-bottom: 1px solid var(--border); }
.fills tbody tr:last-child td { border-bottom: none; }

.d-note { font-size: 12px; color: var(--text-2); margin: 0 0 12px; }
.d-acts { display: inline-flex; gap: 6px; margin-top: 10px; }

/* Pager */
.pager {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 12px;
  font-size: 12px;
}
.pg-left { color: var(--text-2); }
.pg-right { display: inline-flex; align-items: center; gap: 10px; }
.pg-btn {
  padding: 6px 11px; font-family: var(--font-sans); font-size: 11.5px;
  background: var(--elevated); color: var(--text);
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm); cursor: pointer;
}
.pg-btn:disabled { color: var(--text-3); cursor: default; opacity: 0.5; }
.pg-page { color: var(--text-2); }
</style>
