<script setup lang="ts">
/**
 * /markets — Markets index (M4).
 * Sortable live table of every credit market, grouped by family.
 * Dark, in-app.
 */
definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Markets — 1Trade' })

/**
 * A market row as the BFF delivers it. Shape follows the engine's product + summary rather than a
 * frontend invention, so the columns cannot drift from what the exchange actually publishes.
 */
interface Market {
  product_id: string
  name: string
  description: string
  family: string
  credit_type: string
  product_type: string
  quote_precision: number
  tradeable: boolean
  is_paper: boolean
  last: number
  changePct24h: number
  spreadBps: number
  volume24h: number
  high24h: number
  low24h: number
}
interface ExchangeStatus { state: string; reason: string; methodology_url: string }

// Live from the matching engine. Prices are simulated while the venue is paused, but simulated
// SERVER-side — this page used to run its own Brownian motion, which meant /markets and /trade
// could disagree about the same product. The engine is the single source of truth; the page only
// re-fetches.
const markets = ref<Market[]>([])
const exchangeStatus = ref<ExchangeStatus | null>(null)
const loading = ref(true)
const loadError = ref('')

async function loadMarkets() {
  try {
    const r = await $fetch<{ exchange_status: ExchangeStatus; markets: Market[] }>('/api/trading/markets')
    markets.value = r.markets ?? []
    exchangeStatus.value = r.exchange_status ?? null
    loadError.value = ''
  } catch {
    loadError.value = 'Could not reach the exchange. Retrying…'
  } finally {
    loading.value = false
  }
}

let tickInterval: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  void loadMarkets()
  tickInterval = setInterval(() => { void loadMarkets() }, 5000)
})
onBeforeUnmount(() => { if (tickInterval) clearInterval(tickInterval) })

// Filters + sort
const FAMILIES = ['All', 'Index', 'Text', 'Speech', 'Image', 'Video', 'GPU'] as const
type Family = typeof FAMILIES[number]
const fFamily = ref<Family>('All')
const query = ref('')

type SortKey = 'product_id' | 'last' | 'changePct24h' | 'spreadBps' | 'volume24h'
const sortKey = ref<SortKey>('volume24h')
const sortDesc = ref(true)

function setSort(k: SortKey) {
  if (sortKey.value === k) sortDesc.value = !sortDesc.value
  else { sortKey.value = k; sortDesc.value = true }
}

const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  let list = markets.value.filter((m) => {
    if (fFamily.value !== 'All' && m.family !== fFamily.value) return false
    if (q && !(m.product_id.toLowerCase().includes(q) || m.description.toLowerCase().includes(q))) return false
    return true
  })
  list = [...list].sort((a, b) => {
    const av = a[sortKey.value] as number | string
    const bv = b[sortKey.value] as number | string
    if (av < bv) return sortDesc.value ? 1 : -1
    if (av > bv) return sortDesc.value ? -1 : 1
    return 0
  })
  return list
})

// Group by family for the table
const grouped = computed(() => {
  const groups: { family: string; rows: Market[] }[] = []
  if (fFamily.value === 'All') {
    const order: string[] = ['Index', 'Text', 'Speech', 'Image', 'Video', 'GPU']
    for (const f of order) {
      const rows = visible.value.filter((m) => m.family === f)
      if (rows.length) groups.push({ family: f, rows })
    }
  } else {
    groups.push({ family: fFamily.value, rows: visible.value })
  }
  return groups
})

// Aggregate KPIs
const totals = computed(() => {
  const vol = markets.value.reduce((s, m) => s + m.volume24h, 0)
  const pos = markets.value.filter((m) => m.changePct24h >= 0).length
  const neg = markets.value.length - pos
  return { vol, pos, neg }
})

function fmtPx(m: Market): string {
  return '$' + m.last.toFixed(m.quote_precision)
}
function fmtDelta(p: number): string {
  return (p >= 0 ? '▲ ' : '▼ ') + Math.abs(p).toFixed(2) + '%'
}
function fmtVol(n: number): string {
  if (n >= 1_000_000) return '$' + (n / 1_000_000).toFixed(2) + 'M'
  if (n >= 1_000)     return '$' + (n / 1_000).toFixed(1) + 'K'
  return '$' + n.toFixed(0)
}

/** rangePath draws the 24h low→last→high band the engine publishes. The old sparkline replayed a
 *  client-side random walk, which was invented data dressed as history — the summary's real
 *  low/last/high is less detailed but true. */
function rangePath(m: Market): string {
  const lo = m.low24h, hi = m.high24h
  if (!(hi > lo)) return 'M0,11 L80,11'
  const y = (v: number) => 22 - ((v - lo) / (hi - lo)) * 22
  return `M0,${y(lo).toFixed(1)} L40,${y(m.last).toFixed(1)} L80,${y(hi).toFixed(1)}`
}

// SVG sparkline path (kept for other series)
function sparkPath(pts: number[]): string {
  if (!pts.length) return ''
  const w = 80, h = 22
  const min = Math.min(...pts)
  const max = Math.max(...pts)
  const range = max - min || 1
  const stepX = w / (pts.length - 1)
  return pts
    .map((p, i) => {
      const x = i * stepX
      const y = h - ((p - min) / range) * h
      return (i === 0 ? 'M' : 'L') + x.toFixed(1) + ',' + y.toFixed(1)
    })
    .join(' ')
}
function sparkColor(p: number): string {
  return p >= 0 ? 'var(--pos)' : 'var(--neg)'
}

const pinned = reactive<Record<string, boolean>>({})
function togglePin(sym: string) { pinned[sym] = !pinned[sym] }
</script>

<template>
  <div class="markets-page">
    <header class="mk-head">
      <div class="mk-title-row">
        <div>
          <h1 class="mk-title">Markets</h1>
          <p class="mk-sub">
            <span class="mono">{{ markets.length }}</span> instruments live ·
            24h volume <span class="mono">{{ fmtVol(totals.vol) }}</span> ·
            <span class="pos mono">▲ {{ totals.pos }}</span> /
            <span class="neg mono">▼ {{ totals.neg }}</span>
          </p>
        </div>
        <div class="mk-actions">
          <NuxtLink to="/markets/heat" class="mk-act-link">Heat map →</NuxtLink>
          <NuxtLink to="/watchlist"    class="mk-act-link">Watchlist →</NuxtLink>
        </div>
      </div>

      <div class="mk-filters">
        <div class="chips">
          <button
            v-for="f in FAMILIES"
            :key="f"
            type="button"
            class="chip"
            :class="{ on: fFamily === f }"
            @click="fFamily = f"
          >{{ f }}</button>
        </div>
        <input v-model="query" class="mk-search" type="text" placeholder="Search symbol or description…" />
      </div>
    </header>

    <div class="mk-tbl-wrap">
      <table class="mk-tbl">
        <thead>
          <tr>
            <th class="pin"></th>
            <th class="sortable" :class="{ on: sortKey === 'sym' }" @click="setSort('sym')">Symbol</th>
            <th>Description</th>
            <th class="r sortable" :class="{ on: sortKey === 'px' }" @click="setSort('px')">Last</th>
            <th class="r sortable" :class="{ on: sortKey === 'deltaPct' }" @click="setSort('deltaPct')">24h Δ</th>
            <th class="r sortable" :class="{ on: sortKey === 'spreadBps' }" @click="setSort('spreadBps')">Spread</th>
            <th class="r sortable" :class="{ on: sortKey === 'vol24h' }" @click="setSort('vol24h')">Vol 24h</th>
            <th class="r">60m</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="g in grouped" :key="g.family">
            <tr v-if="fFamily === 'All'" class="group-row">
              <td colspan="9">
                <span class="g-name mono">{{ g.family.toUpperCase() }}</span>
                <span class="g-count mono">{{ g.rows.length }} markets</span>
              </td>
            </tr>
            <tr v-for="m in g.rows" :key="m.product_id" class="row">
              <td class="pin">
                <button class="pin-btn" :class="{ pinned: pinned[m.product_id] }" :title="(pinned[m.product_id] ? 'Unpin' : 'Pin') + ' ' + m.product_id" @click="togglePin(m.product_id)">
                  <svg viewBox="0 0 12 12" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M6 1l1.5 3.2 3.5.5-2.5 2.4.6 3.4L6 8.9 2.9 10.5l.6-3.4L1 4.7l3.5-.5z" />
                  </svg>
                </button>
              </td>
              <td class="sym">
                <NuxtLink :to="'/markets/' + m.product_id.toLowerCase()" class="sym-link mono">{{ m.product_id }}</NuxtLink>
              </td>
              <td class="desc">{{ m.description }}</td>
              <td class="r mono px">{{ fmtPx(m) }}</td>
              <td class="r mono delta" :class="m.changePct24h >= 0 ? 'pos' : 'neg'">{{ fmtDelta(m.changePct24h) }}</td>
              <td class="r mono spread">{{ m.spreadBps.toFixed(1) }} bps</td>
              <td class="r mono vol">{{ fmtVol(m.volume24h) }}</td>
              <td class="r">
                <svg :width="80" :height="22" viewBox="0 0 80 22" class="spark">
                  <path :d="rangePath(m)" fill="none" :stroke="sparkColor(m.changePct24h)" stroke-width="1.2" />
                </svg>
              </td>
              <td class="r">
                <NuxtLink :to="'/markets/' + m.product_id.toLowerCase()" class="open-link">Open →</NuxtLink>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
/* Exchange status — deliberately prominent. It reports a paused venue and simulated market data;
   burying it would misrepresent the product while the licence is pending. */
.mk-status {
  display: flex; align-items: center; gap: var(--sp-3); flex-wrap: wrap;
  padding: var(--sp-3) var(--sp-4); margin-bottom: var(--sp-4);
  border: 1px solid color-mix(in srgb, var(--warn) 42%, transparent);
  background: color-mix(in srgb, var(--warn) 10%, transparent);
  border-radius: var(--radius-sm);
  font-size: var(--fs-sm); color: var(--text-2);
}
.mk-status-pill {
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide); text-transform: uppercase;
  color: var(--warn); border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
  border-radius: 2px; padding: 2px 7px; flex: none;
}
.mk-status-txt { flex: 1; min-width: 0; }
.mk-status-link { color: var(--brand); text-decoration: none; font-weight: 600; }
.mk-status-link:hover { text-decoration: underline; }
.mk-status-err {
  border-color: color-mix(in srgb, var(--neg) 45%, transparent);
  background: color-mix(in srgb, var(--neg) 10%, transparent);
  color: var(--neg);
}
.markets-page {
  padding: 16px 24px 32px;
  color: var(--text);
}
.markets-page .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.pos { color: var(--pos); font-weight: 600; }
.neg { color: var(--neg); font-weight: 600; }

/* Head */
.mk-head { margin-bottom: 14px; }
.mk-title-row {
  display: flex; justify-content: space-between; align-items: flex-end;
  margin-bottom: 14px;
}
.mk-title {
  font-family: var(--font-display); font-weight: 600;
  font-size: 26px; letter-spacing: -0.015em; margin: 0 0 4px;
}
.mk-sub { color: var(--text-2); font-size: 12.5px; margin: 0; }
.mk-actions { display: inline-flex; gap: 18px; }
.mk-act-link {
  color: var(--text-2); text-decoration: none; font-size: 12.5px;
}
.mk-act-link:hover { color: var(--text); }

.mk-filters {
  display: flex; justify-content: space-between; align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}
.chips { display: inline-flex; gap: 4px; }
.chip {
  padding: 5px 10px;
  background: transparent; color: var(--text-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-family: var(--font-sans); font-size: 11.5px; font-weight: 500;
  cursor: pointer;
}
.chip:hover { color: var(--text); border-color: var(--border-strong); }
.chip.on    { background: var(--text); color: var(--canvas); border-color: var(--text); }
.mk-search {
  width: 280px;
  height: 30px;
  padding: 0 10px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text); font-size: 12px;
  outline: none;
}
.mk-search:focus { border-color: var(--accent); }

/* Table */
.mk-tbl-wrap { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); overflow-x: auto; }
.mk-tbl { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.mk-tbl thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3);
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
  white-space: nowrap;
}
.mk-tbl th.r { text-align: right; }
.mk-tbl th.sortable { cursor: pointer; user-select: none; }
.mk-tbl th.sortable.on { color: var(--text); }
.mk-tbl th.pin { width: 32px; padding-left: 14px; }

.mk-tbl tbody tr.row {
  border-bottom: 1px solid var(--border);
  transition: background-color 80ms ease;
}
.mk-tbl tbody tr.row:hover { background: rgba(255,255,255,0.025); }
.mk-tbl tbody td { padding: 9px 12px; vertical-align: middle; white-space: nowrap; }
.mk-tbl tbody td.r { text-align: right; }
.mk-tbl tbody td.pin { padding-left: 14px; }

.pin-btn {
  width: 22px; height: 22px;
  background: transparent;
  border: none;
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--text-3);
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.pin-btn:hover  { color: var(--text); }
.pin-btn.pinned { color: var(--brand); }

.group-row td {
  background: var(--canvas);
  padding: 8px 14px !important;
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.18em;
  color: var(--text-3);
}
.g-name { color: var(--text-2); font-weight: 700; margin-right: 10px; }
.g-count { color: var(--text-3); }

.sym-link { color: var(--text); font-weight: 700; text-decoration: none; }
.sym-link:hover { color: var(--brand); }
.desc { color: var(--text-2); font-size: 11.5px; }
.px { color: var(--text); font-weight: 500; }
.spread, .vol { color: var(--text); }
.spark { display: inline-block; vertical-align: middle; }
.open-link { color: var(--text-2); text-decoration: none; font-size: 11.5px; }
.open-link:hover { color: var(--text); }

/* ── Mobile: the header + filters wrap; the market table scrolls horizontally (already wrapped). ── */
@media (max-width: 640px) {
  .markets-page, .mk-page { padding-left: 14px; padding-right: 14px; }
  .mk-title-row { flex-wrap: wrap; gap: 8px; }
  .mk-actions { flex-wrap: wrap; }
  .mk-filters { flex-wrap: wrap; gap: 8px; }
}
</style>
