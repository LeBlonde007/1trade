<script setup lang="ts">
/**
 * /history — Trade history (C4). Live, no mock data.
 *
 * One dense, time-ordered table over two real sources (merged in utils/history.ts):
 *   - paper fills from the matching engine's paper venue (real money paused, F22); these settle in the ledger as paper
 *     fills and are labelled that way on screen;
 *   - ledger transactions from credit-ledger — real, append-only, each with its own chain hash;
 *   - paper cash movements from credit-ledger (activation grant, trade legs, fees) — always paper.
 *
 * Filters, pagination and CSV/JSON export all operate on the loaded rows. Totals are summed in
 * fixed-point (utils/history.ts sumDecimal), never floats.
 */
import { ChevronDown, ChevronRight, Copy, CheckCircle2, FileDown, Search, X, RefreshCw } from 'lucide-vue-next'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Trade history — 1Trade' })

const route = useRoute()

// ─── Load ───────────────────────────────────────────────────
// Each source loads independently: a ledger outage must not hide paper fills, and vice versa.
const FETCH_LIMIT = 200
const rows = ref<HistoryRow[]>([])
const loading = ref(true)
const errors = ref<string[]>([])

/** load fetches both sources in parallel and merges whatever succeeded. */
async function load() {
  loading.value = true
  const [fills, txs, cash] = await Promise.allSettled([
    $fetch<{ fills: EngineFill[] }>('/api/trading/fills', { query: { limit: FETCH_LIMIT } }),
    $fetch<{ transactions: LedgerTx[] }>('/api/wallet/transactions', { query: { limit: FETCH_LIMIT } }),
    $fetch<{ transactions: LedgerCashTx[] }>('/api/wallet/cash-transactions', { query: { limit: FETCH_LIMIT } }),
  ])
  const errs: string[] = []
  if (fills.status === 'rejected') errs.push('Paper fills could not be loaded from the matching engine.')
  if (txs.status === 'rejected') errs.push('Ledger transactions could not be loaded from the credit ledger.')
  if (cash.status === 'rejected') errs.push('Paper cash movements could not be loaded from the credit ledger.')
  errors.value = errs
  rows.value = mergeHistory(
    fills.status === 'fulfilled' ? fills.value.fills ?? [] : [],
    txs.status === 'fulfilled' ? txs.value.transactions ?? [] : [],
    cash.status === 'fulfilled' ? cash.value.transactions ?? [] : [],
  )
  loading.value = false
}
onMounted(load)

// ─── Filters ────────────────────────────────────────────────
const KINDS: HistoryKind[] = ['Trade', 'Purchase', 'Conversion', 'Usage', 'Cash', 'Adjustment']
const DATE_RANGES: { key: string; label: string; days: number | null }[] = [
  { key: '7d', label: 'Last 7 days', days: 7 },
  { key: '30d', label: 'Last 30 days', days: 30 },
  { key: '90d', label: 'Last 90 days', days: 90 },
  { key: 'all', label: 'All loaded', days: null },
]
const SIDE_FILTERS: { key: 'all' | 'buy' | 'sell'; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'buy', label: 'Buy' },
  { key: 'sell', label: 'Sell' },
]

const fDate = ref('30d')
const fMarket = ref('all')
const fKind = ref<'all' | HistoryKind>('all')
const fSide = ref<'all' | 'buy' | 'sell'>('all')
const fQuery = ref(typeof route.query.order === 'string' ? route.query.order : '')

/** Markets present in the loaded rows — the dropdown never offers a market with nothing in it. */
const markets = computed(() => [...new Set(rows.value.map((r) => r.market))].sort())

const filtered = computed(() => {
  const q = fQuery.value.trim().toLowerCase()
  const days = DATE_RANGES.find((d) => d.key === fDate.value)?.days ?? null
  const since = days === null ? 0 : Date.now() - days * 86_400_000
  return rows.value.filter((r) => {
    if (since && (Date.parse(r.ts) || 0) < since) return false
    if (fMarket.value !== 'all' && r.market !== fMarket.value) return false
    if (fKind.value !== 'all' && r.kind !== fKind.value) return false
    if (fSide.value !== 'all' && r.side !== fSide.value) return false
    if (q && ![r.id, r.orderId, r.market, r.chainHash, r.referenceId].some((v) => v?.toLowerCase().includes(q))) return false
    return true
  })
})

/** resetFilters restores the defaults (last 30 days, everything else open). */
function resetFilters() {
  fDate.value = '30d'
  fMarket.value = 'all'
  fKind.value = 'all'
  fSide.value = 'all'
  fQuery.value = ''
}

const filtersActive = computed(() =>
  fDate.value !== '30d' || fMarket.value !== 'all' || fKind.value !== 'all' || fSide.value !== 'all' || fQuery.value.length > 0,
)

// ─── Summary (over the filtered rows, fixed-point) ──────────
const stats = computed(() => {
  const trades = filtered.value.filter((r) => r.source === 'engine')
  // "Ledger entries" counts credit and cash rows alike; both are real ledger rows.
  return {
    trades: trades.length,
    notional: sumDecimal(trades.map((r) => r.total)),
    fees: sumDecimal(trades.map((r) => r.fee)),
    ledger: filtered.value.length - trades.length,
  }
})

// ─── Pagination ─────────────────────────────────────────────
const PAGE_SIZE = 25
const page = ref(1)
watch(filtered, () => { page.value = 1 })
const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))
const visibleRows = computed(() => filtered.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))

// ─── Date dividers ──────────────────────────────────────────
interface RenderItem { kind: 'divider' | 'row'; date?: string; row?: HistoryRow }
const rendered = computed<RenderItem[]>(() => {
  const out: RenderItem[] = []
  let lastDate = ''
  for (const r of visibleRows.value) {
    const d = isoUTC(r.ts).slice(0, 10)
    if (d !== lastDate) {
      out.push({ kind: 'divider', date: d })
      lastDate = d
    }
    out.push({ kind: 'row', row: r })
  }
  return out
})

// ─── Row expansion + copy ───────────────────────────────────
const expandedId = ref<string | null>(null)
/** toggleExpand opens one row's detail at a time. */
function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

const copied = ref<string | null>(null)
/** copy writes s to the clipboard and flashes a check on the button keyed by key. */
function copy(s: string, key: string, ev?: Event) {
  ev?.stopPropagation()
  navigator.clipboard?.writeText(s).catch(() => {})
  copied.value = key
  setTimeout(() => { if (copied.value === key) copied.value = null }, 1200)
}

// ─── Export (real files of the filtered rows) ───────────────
/** doExport downloads the filtered rows as CSV or JSON, exactly as loaded (fixed-point strings). */
function doExport(kind: 'csv' | 'json') {
  const body = kind === 'csv' ? toCSV(filtered.value) : JSON.stringify(filtered.value, null, 2)
  const blob = new Blob([body], { type: kind === 'csv' ? 'text/csv' : 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `1trade-history-${new Date().toISOString().slice(0, 10)}.${kind}`
  a.click()
  URL.revokeObjectURL(url)
}

// ─── Formatters (display only; values stay strings) ─────────
/** isoUTC normalises any RFC 3339 timestamp to UTC ISO form; unparseable input is returned as-is. */
function isoUTC(ts: string): string {
  const n = Date.parse(ts)
  return Number.isNaN(n) ? ts : new Date(n).toISOString()
}
/** fmtTime renders the HH:MM:SS part of a timestamp, in UTC. */
function fmtTime(ts: string): string { return isoUTC(ts).slice(11, 19) }
/** fmtDate renders a YYYY-MM-DD divider label. */
function fmtDate(d: string): string {
  return new Date(d + 'T00:00:00Z').toLocaleDateString('en-US', { weekday: 'short', day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
}
/** fmtNum groups a fixed-point string for display, keeping up to dp decimals. */
function fmtNum(s: string | null | undefined, dp = 6): string {
  if (s === null || s === undefined || s === '') return '—'
  const n = Number(s)
  return Number.isFinite(n) ? n.toLocaleString('en-US', { maximumFractionDigits: dp }) : s
}
/** fmtSigned is fmtNum with an explicit + on credits, for ledger deltas. */
function fmtSigned(s: string): string {
  return (s.startsWith('-') ? '' : '+') + fmtNum(s)
}
/** kindTone maps a row kind onto the type-badge colour. */
function kindTone(k: HistoryKind): string {
  if (k === 'Trade') return 'neutral'
  if (k === 'Conversion') return 'info'
  if (k === 'Purchase') return 'pos'
  if (k === 'Usage') return 'warn'
  if (k === 'Cash') return 'pos'
  return 'mute'
}
</script>

<template>
  <div class="history-page">
    <!-- Head -->
    <header class="hd">
      <div class="hd-left">
        <h1>Trade history</h1>
        <p class="hd-sub">
          <span class="mono">{{ filtered.length }}</span> events
          <span v-if="filtersActive" class="dim">· filtered</span>
          <span class="dim">· paper fills from the matching engine + real ledger entries</span>
        </p>
      </div>
      <div class="hd-acts">
        <button class="btn ghost" type="button" :disabled="loading" @click="load">
          <RefreshCw :size="13" />
          Refresh
        </button>
        <button class="btn ghost" type="button" :disabled="!filtered.length" @click="doExport('csv')">
          <FileDown :size="13" />
          Export CSV
        </button>
        <button class="btn ghost" type="button" :disabled="!filtered.length" @click="doExport('json')">
          <FileDown :size="13" />
          Export JSON
        </button>
      </div>
    </header>

    <!-- Staged-venue notice: the exchange is paused pending licensing (F22). -->
    <p class="notice">
      <span class="notice-tag mono">PAPER</span>
      Paper trading: trades below are your paper fills, settled in the ledger with paper cash and paper
      credits. Ledger entries (purchases, conversions, usage, trade settlements) are hash-chained.
      Real-money trading is paused pending exchange licensing.
    </p>

    <p v-for="e in errors" :key="e" class="err">{{ e }} <button type="button" class="link" @click="load">Retry</button></p>

    <!-- Stat strip -->
    <section class="stats">
      <div class="stat">
        <div class="lbl">Paper trades</div>
        <div class="val mono">{{ stats.trades.toLocaleString() }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Paper notional</div>
        <div class="val mono" :title="stats.notional">${{ fmtNum(stats.notional, 2) }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Paper fees</div>
        <div class="val mono" :title="stats.fees">${{ fmtNum(stats.fees, 2) }}</div>
      </div>
      <div class="stat">
        <div class="lbl">Ledger entries</div>
        <div class="val mono">{{ stats.ledger.toLocaleString() }}</div>
      </div>
    </section>

    <!-- Filter bar (sticky) -->
    <section class="filters">
      <div class="f">
        <label class="f-lbl" for="h-date">Date</label>
        <select id="h-date" v-model="fDate" class="f-input">
          <option v-for="d in DATE_RANGES" :key="d.key" :value="d.key">{{ d.label }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-lbl" for="h-market">Market</label>
        <select id="h-market" v-model="fMarket" class="f-input mono">
          <option value="all">All markets</option>
          <option v-for="m in markets" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-lbl" for="h-type">Type</label>
        <select id="h-type" v-model="fKind" class="f-input">
          <option value="all">All types</option>
          <option v-for="k in KINDS" :key="k" :value="k">{{ k }}</option>
        </select>
      </div>
      <div class="f">
        <span class="f-lbl">Side</span>
        <div class="chips">
          <button v-for="s in SIDE_FILTERS" :key="s.key" type="button" class="chip" :class="{ on: fSide === s.key }" @click="fSide = s.key">{{ s.label }}</button>
        </div>
      </div>
      <div class="f f-grow">
        <label class="f-lbl" for="h-search">Search</label>
        <div class="search-wrap">
          <Search :size="13" class="search-icon" />
          <input id="h-search" v-model="fQuery" type="text" class="f-input search" placeholder="order id, tx id, market, chain hash…" />
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
      <div v-if="loading && !rows.length" class="state">Loading history…</div>
      <div v-else-if="!filtered.length" class="state">
        {{ rows.length ? 'No events match these filters.' : 'No history yet — buy credits, run inference, or convert to see ledger entries here.' }}
        <button v-if="filtersActive" type="button" class="link" @click="resetFilters">Reset filters</button>
      </div>
      <table v-else class="tbl">
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
            <th>Mode</th>
            <th>Reference</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(item, i) in rendered" :key="i">
            <tr v-if="item.kind === 'divider'" class="divider-row">
              <td colspan="11">
                <span class="div-date">{{ fmtDate(item.date!) }}</span>
              </td>
            </tr>
            <template v-else>
              <tr class="row" :class="{ open: expandedId === item.row!.id }" @click="toggleExpand(item.row!.id)">
                <td class="chev">
                  <ChevronRight v-if="expandedId !== item.row!.id" :size="11" />
                  <ChevronDown v-else :size="11" />
                </td>
                <td class="mono ts">{{ fmtTime(item.row!.ts) }}</td>
                <td><span class="type" :class="'t-' + kindTone(item.row!.kind)">{{ item.row!.kind }}</span></td>
                <td class="mono mkt">{{ item.row!.market }}</td>
                <td>
                  <span v-if="item.row!.side" class="side mono" :class="item.row!.side">{{ item.row!.side.toUpperCase() }}</span>
                  <span v-else class="dim">—</span>
                </td>
                <td class="r mono">{{ item.row!.source !== 'engine' ? fmtSigned(item.row!.quantity) : fmtNum(item.row!.quantity) }}</td>
                <td class="r mono">{{ item.row!.price ?? '—' }}</td>
                <td class="r mono">{{ item.row!.total ? '$' + fmtNum(item.row!.total, 2) : '—' }}</td>
                <td class="r mono dim">{{ item.row!.fee ? '$' + fmtNum(item.row!.fee) : '—' }}</td>
                <td><span class="mode mono" :class="{ paper: item.row!.isPaper }">{{ item.row!.isPaper ? 'PAPER' : 'REAL' }}</span></td>
                <td class="hash-cell">
                  <span class="mono dim id">{{ truncateId(item.row!.chainHash ?? item.row!.orderId ?? item.row!.id, 10, 4) }}</span>
                  <button class="copy" type="button" title="Copy" @click="copy(item.row!.chainHash ?? item.row!.orderId ?? item.row!.id, item.row!.id, $event)">
                    <Copy v-if="copied !== item.row!.id" :size="10" />
                    <CheckCircle2 v-else :size="10" />
                  </button>
                </td>
              </tr>

              <!-- Expanded detail row -->
              <tr v-if="expandedId === item.row!.id" class="detail-row">
                <td colspan="11">
                  <div class="detail">
                    <div v-if="item.row!.source === 'engine'" class="d-left">
                      <div class="d-eyebrow">Paper fill · matching engine</div>
                      <dl class="kv">
                        <dt>Fill ID</dt><dd class="mono">{{ item.row!.id }}</dd>
                        <dt>Order ID</dt><dd class="mono">{{ item.row!.orderId }}</dd>
                        <dt>Executed</dt><dd class="mono">{{ item.row!.ts }}</dd>
                        <dt>Liquidity</dt><dd class="mono">{{ item.row!.liquidity }}</dd>
                        <dt>Price</dt><dd class="mono">{{ item.row!.price }}</dd>
                        <dt>Quantity</dt><dd class="mono">{{ item.row!.quantity }}</dd>
                        <dt>Notional</dt><dd class="mono">{{ item.row!.total }}</dd>
                        <dt>Fee</dt><dd class="mono">{{ item.row!.fee }}</dd>
                      </dl>
                      <p class="d-note">
                        Paper fill — settled in the credit ledger with paper value; the ledger's settlement
                        entries carry the hash chain.
                      </p>
                    </div>
                    <div v-else class="d-left">
                      <div class="d-eyebrow">{{ item.row!.source === 'cash' ? 'Paper cash entry · credit ledger' : 'Ledger entry · credit ledger' }}</div>
                      <dl class="kv">
                        <dt>Tx ID</dt><dd class="mono">{{ item.row!.id }}</dd>
                        <dt>Operation</dt><dd class="mono">{{ item.row!.operation }}</dd>
                        <dt>{{ item.row!.source === 'cash' ? 'Currency' : 'Credit type' }}</dt><dd class="mono">{{ item.row!.market }}</dd>
                        <dt>Amount</dt><dd class="mono">{{ item.row!.quantity }}</dd>
                        <dt>Balance after</dt><dd class="mono">{{ item.row!.balanceAfter }}</dd>
                        <dt>Reference</dt><dd class="mono">{{ item.row!.referenceId || '—' }}</dd>
                        <dt>Recorded</dt><dd class="mono">{{ item.row!.ts }}</dd>
                        <dt>Chain hash</dt><dd class="mono">{{ item.row!.chainHash || '—' }}</dd>
                      </dl>
                      <p class="d-note">
                        Append-only: chain_hash = SHA-256(prev_chain_hash ‖ canonical_json(row)).
                      </p>
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
    <footer v-if="filtered.length" class="pager">
      <div class="pg-left">
        Showing <span class="mono">{{ (page - 1) * PAGE_SIZE + 1 }}–{{ (page - 1) * PAGE_SIZE + visibleRows.length }}</span>
        of <span class="mono">{{ filtered.length }}</span> events
        <span class="dim">· most recent {{ FETCH_LIMIT }} per source</span>
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
  display: grid; grid-template-columns: minmax(0, 720px); gap: 20px;
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





.d-note { font-size: 12px; color: var(--text-2); margin: 0 0 12px; }

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

/* Staged-venue notice, errors, empty/loading states */
.notice {
  display: flex; align-items: center; gap: 8px;
  margin: 0 0 12px; padding: 8px 12px;
  font-size: 12px; color: var(--text-2);
  background: var(--elevated); border: 1px solid var(--border);
}
.notice-tag, .mode {
  padding: 2px 6px; border-radius: var(--radius-sm);
  font-size: 9.5px; font-weight: 700; letter-spacing: 0.14em;
  background: rgba(255,255,255,0.06); color: var(--text-2);
}
.notice-tag, .mode.paper { background: rgba(245,158,11,0.16); color: var(--warn); }
.err {
  margin: 0 0 12px; padding: 8px 12px; font-size: 12px;
  color: var(--neg); background: var(--neg-soft); border: 1px solid var(--neg);
}
.state { padding: 32px 16px; text-align: center; color: var(--text-2); font-size: 12.5px; }
.link {
  background: none; border: none; padding: 0; margin-left: 6px;
  color: var(--accent); font: inherit; cursor: pointer;
}
.link:hover { text-decoration: underline; }

/* ── Mobile: 2-col stats, single-column detail, horizontally scrollable tables. ── */
@media (max-width: 640px) {
  .history-page { padding: 16px 14px 28px; }
  .stats { grid-template-columns: repeat(2, 1fr); }
  .detail { grid-template-columns: 1fr; }
  .tbl-wrap { overflow-x: auto; }
}
</style>
