<script setup lang="ts">
/**
 * /portfolio — the trader's paper account. Live, no mock data.
 *
 * Sources (each loads independently; one failing never blanks the others):
 *   - paper cash from credit-ledger (/api/wallet/cash-balances) — real, with what open orders hold;
 *   - open paper positions from the matching engine (/api/trading/positions) — built from the
 *     caller's real paper fills, which settle in the ledger (real money stays paused, F22);
 *   - paper fills (/api/trading/fills) and cash movements (/api/wallet/cash-transactions) for the
 *     per-position fills and the activity feed.
 *
 * Cash and positions are shown side by side, never summed into one "account value": positions are
 * simulated and cash is a real ledger balance, so a single total would be a number nobody holds.
 * There is no performance chart: no account-value history exists yet, and the previous one was
 * generated with Math.random.
 */
import { ChevronDown, ChevronRight, RefreshCw } from 'lucide-vue-next'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Portfolio — 1Trade' })

/** A paper position as served by matching-engine /v1/trading/positions. */
interface EnginePosition {
  product_id: string
  side: 'long' | 'short'
  net_quantity: string
  avg_entry_price: string
  mark_price: string
  notional: string
  unrealized_pnl: string
  unrealized_pnl_pct: string
  realized_pnl_total: string
  is_paper: boolean
}

const wallet = useWallet()
const positions = ref<EnginePosition[]>([])
const fills = ref<EngineFill[]>([])
const cashTxs = ref<LedgerCashTx[]>([])
const loading = ref(true)
const errors = ref<string[]>([])

/** load fetches every source in parallel and keeps whatever succeeded. */
async function load() {
  loading.value = true
  const [pos, fl, cash, ctx] = await Promise.allSettled([
    $fetch<{ positions: EnginePosition[] }>('/api/trading/positions'),
    $fetch<{ fills: EngineFill[] }>('/api/trading/fills', { query: { limit: 200 } }),
    wallet.loadCashBalances(),
    $fetch<{ transactions: LedgerCashTx[] }>('/api/wallet/cash-transactions', { query: { limit: 50 } }),
  ])
  const errs: string[] = []
  if (pos.status === 'fulfilled') positions.value = pos.value.positions ?? []
  else errs.push('Positions could not be loaded from the matching engine.')
  if (fl.status === 'fulfilled') fills.value = fl.value.fills ?? []
  else errs.push('Fills could not be loaded from the matching engine.')
  if (cash.status === 'rejected') errs.push('Paper cash could not be loaded from the credit ledger.')
  if (ctx.status === 'fulfilled') cashTxs.value = ctx.value.transactions ?? []
  else errs.push('Cash movements could not be loaded from the credit ledger.')
  errors.value = errs
  loading.value = false
}
onMounted(load)

// ─── Cash (real ledger) ─────────────────────────────────────
const cash = computed(() => wallet.cashBalances.value.find((b) => b.currency === 'USD') ?? null)
/** cashAvailable is balance − locked, in fixed point. */
const cashAvailable = computed(() =>
  cash.value ? fromMicros(toMicros(cash.value.balance) - toMicros(cash.value.locked_amount)) : '0.000000')

// ─── Positions (the engine's paper venue) ─────────────
const totals = computed(() => ({
  value: sumDecimal(positions.value.map((p) => p.notional)),
  unrealized: sumDecimal(positions.value.map((p) => p.unrealized_pnl)),
  realized: sumDecimal(positions.value.map((p) => p.realized_pnl_total)),
}))

/** exposure is each market's share of total position value, for the allocation bar. */
const PALETTE = ['#D4AF37', '#4A90E2', '#F5A524', '#8FB8E6', '#E89818', '#6BA4E8', '#7E786C']
const exposure = computed(() => {
  const total = Number(totals.value.value) || 0
  return positions.value
    .map((p, i) => ({ market: p.product_id, value: p.notional, share: total ? (Number(p.notional) / total) * 100 : 0, color: PALETTE[i % PALETTE.length]! }))
    .sort((a, b) => b.share - a.share)
})

const openRow = ref<string | null>(null)
/** toggleRow expands one position to show its fills. */
function toggleRow(id: string) { openRow.value = openRow.value === id ? null : id }
/** fillsFor returns a product's fills, newest first (already ordered by the engine). */
function fillsFor(productID: string) { return fills.value.filter((f) => f.product_id === productID).slice(0, 10) }

// ─── Activity (fills + cash movements) ──────────────────────
const activity = computed(() => mergeHistory(fills.value, [], cashTxs.value).slice(0, 12))

// ─── Formatters (display only; values stay strings) ─────────
/** num groups a fixed-point string for display. */
function num(s: string | null | undefined, dp = 6): string {
  if (s === null || s === undefined || s === '') return '—'
  const n = Number(s)
  return Number.isFinite(n) ? n.toLocaleString('en-US', { maximumFractionDigits: dp }) : s
}
/** usd renders a fixed-point dollar string with cents. */
function usd(s: string | null | undefined): string {
  if (s === null || s === undefined || s === '') return '—'
  const n = Number(s)
  const abs = Math.abs(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  return (n < 0 ? '−$' : '$') + abs
}
/** signed renders a P&L string with an explicit sign. */
function signed(s: string): string { return Number(s) > 0 ? '+' + usd(s) : usd(s) }
/** tone is pos / neg / '' for colouring a signed value. */
function tone(s: string): string { const n = Number(s); return n > 0 ? 'pos' : n < 0 ? 'neg' : '' }
/** when renders an RFC 3339 timestamp as a short UTC date-time. */
function when(ts: string): string {
  const n = Date.parse(ts)
  return Number.isNaN(n) ? ts : new Date(n).toISOString().slice(5, 16).replace('T', ' ')
}
</script>

<template>
  <div class="portfolio-page">
    <header class="hd">
      <div>
        <h1>Portfolio</h1>
        <p class="hd-sub">Your paper account: cash from the credit ledger, positions from the matching engine.</p>
      </div>
      <button class="btn" type="button" :disabled="loading" @click="load">
        <RefreshCw :size="13" />
        Refresh
      </button>
    </header>

    <p class="notice">
      <span class="tag mono">PAPER</span>
      Paper trading: your orders fill on the paper books and settle in the ledger with paper cash and
      paper credits. Real-money trading is paused pending exchange licensing. No real money is involved.
    </p>
    <p v-for="e in errors" :key="e" class="err">{{ e }} <button type="button" class="link" @click="load">Retry</button></p>

    <!-- KPIs -->
    <section class="hero">
      <div class="kpi">
        <div class="lbl">Paper cash</div>
        <div class="val mono">{{ cash ? usd(cash.balance) : '—' }}</div>
        <div v-if="cash" class="sub mono">
          {{ usd(cashAvailable) }} available
          <span v-if="Number(cash.locked_amount) > 0"> · {{ usd(cash.locked_amount) }} held by open orders</span>
        </div>
        <div v-else-if="!loading" class="sub">No paper cash yet — it is granted when your account is activated.</div>
      </div>
      <div class="kpi">
        <div class="lbl">Position value <span class="dim">· paper</span></div>
        <div class="val mono">{{ usd(totals.value) }}</div>
        <div class="sub mono">{{ positions.length }} open {{ positions.length === 1 ? 'position' : 'positions' }}</div>
      </div>
      <div class="kpi">
        <div class="lbl">Unrealized P&amp;L</div>
        <div class="val mono" :class="tone(totals.unrealized)">{{ signed(totals.unrealized) }}</div>
        <div class="sub">Marked to the paper book's mid</div>
      </div>
      <div class="kpi">
        <div class="lbl">Realized P&amp;L</div>
        <div class="val mono" :class="tone(totals.realized)">{{ signed(totals.realized) }}</div>
        <div class="sub">From partial closes</div>
      </div>
    </section>

    <div class="grid">
      <!-- Positions -->
      <section class="card">
        <div class="card-h">
          <h2>Open positions</h2>
          <span class="dim mono">{{ positions.length }}</span>
        </div>
        <div v-if="loading && !positions.length" class="state">Loading positions…</div>
        <div v-else-if="!positions.length" class="state">No open positions.</div>
        <table v-else class="tbl">
          <thead>
            <tr>
              <th class="chev"></th>
              <th>Market</th>
              <th>Side</th>
              <th class="r">Qty</th>
              <th class="r">Avg entry</th>
              <th class="r">Mark</th>
              <th class="r">Value</th>
              <th class="r">Unrealized</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="p in positions" :key="p.product_id">
              <tr class="row" :class="{ open: openRow === p.product_id }" @click="toggleRow(p.product_id)">
                <td class="chev">
                  <ChevronRight v-if="openRow !== p.product_id" :size="11" />
                  <ChevronDown v-else :size="11" />
                </td>
                <td class="mono mkt">{{ p.product_id }}</td>
                <td><span class="side mono" :class="p.side">{{ p.side.toUpperCase() }}</span></td>
                <td class="r mono">{{ num(p.net_quantity, 2) }}</td>
                <td class="r mono">{{ num(p.avg_entry_price) }}</td>
                <td class="r mono">{{ num(p.mark_price) }}</td>
                <td class="r mono">{{ usd(p.notional) }}</td>
                <td class="r mono" :class="tone(p.unrealized_pnl)">
                  {{ signed(p.unrealized_pnl) }}
                  <span class="pct">{{ Number(p.unrealized_pnl_pct) > 0 ? '+' : '' }}{{ p.unrealized_pnl_pct }}%</span>
                </td>
              </tr>
              <tr v-if="openRow === p.product_id" class="detail-row">
                <td colspan="8">
                  <div class="detail">
                    <div class="d-eyebrow">Recent fills · {{ p.product_id }}</div>
                    <table v-if="fillsFor(p.product_id).length" class="fills">
                      <thead><tr><th>Time (UTC)</th><th>Side</th><th class="r">Qty</th><th class="r">Price</th><th class="r">Fee</th><th>Liquidity</th></tr></thead>
                      <tbody>
                        <tr v-for="f in fillsFor(p.product_id)" :key="f.fill_id">
                          <td class="mono">{{ when(f.executed_at) }}</td>
                          <td><span class="side mono" :class="f.side === 'buy' ? 'long' : 'short'">{{ f.side.toUpperCase() }}</span></td>
                          <td class="r mono">{{ num(f.quantity, 2) }}</td>
                          <td class="r mono">{{ num(f.price) }}</td>
                          <td class="r mono dim">{{ usd(f.fee) }}</td>
                          <td class="mono dim">{{ f.liquidity }}</td>
                        </tr>
                      </tbody>
                    </table>
                    <p v-else class="dim">No fills loaded for this market.</p>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </section>

      <!-- Right rail -->
      <div class="rail">
        <section class="card">
          <div class="card-h"><h2>Exposure by market</h2></div>
          <div v-if="!exposure.length" class="state">Nothing to allocate yet.</div>
          <template v-else>
            <div class="bar">
              <span v-for="x in exposure" :key="x.market" :style="{ width: x.share + '%', background: x.color }" :title="x.market + ' ' + x.share.toFixed(1) + '%'" />
            </div>
            <ul class="legend">
              <li v-for="x in exposure" :key="x.market">
                <span class="sw" :style="{ background: x.color }" />
                <span class="mono">{{ x.market }}</span>
                <span class="mono dim">{{ usd(x.value) }}</span>
                <span class="mono">{{ x.share.toFixed(1) }}%</span>
              </li>
            </ul>
          </template>
        </section>

        <section class="card">
          <div class="card-h">
            <h2>Recent activity</h2>
            <NuxtLink to="/history" class="link">View all</NuxtLink>
          </div>
          <div v-if="!activity.length" class="state">{{ loading ? 'Loading…' : 'No activity yet.' }}</div>
          <ul v-else class="feed">
            <li v-for="r in activity" :key="r.id">
              <span class="mono dim">{{ when(r.ts) }}</span>
              <span class="kind" :class="r.source">{{ r.source === 'cash' ? (r.operation === 'paper_grant' ? 'Grant' : r.operation === 'fee' ? 'Fee' : 'Cash') : (r.side ?? '').toUpperCase() }}</span>
              <span class="mono">{{ r.market }}</span>
              <span class="mono r" :class="r.source === 'cash' ? tone(r.quantity) : ''">
                {{ r.source === 'cash' ? signed(r.quantity) : compact(r.quantity) + ' @ ' + num(r.price) }}
              </span>
            </li>
          </ul>
        </section>
      </div>
    </div>

    <p class="foot dim">
      Performance history is not shown yet: there is no recorded account-value series to chart. It will
      appear once paper trades settle through the ledger.
    </p>
  </div>
</template>

<style scoped>
.portfolio-page { padding: 20px 24px 32px; color: var(--text); font-family: var(--font-sans); font-size: 12.5px; }
.portfolio-page .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.dim { color: var(--text-3); }
.pos { color: var(--pos); font-weight: 600; }
.neg { color: var(--neg); font-weight: 600; }
.r { text-align: right; }

.hd { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 14px; }
.hd h1 { font-family: var(--font-display); font-weight: 600; font-size: 26px; letter-spacing: -0.015em; margin: 0 0 4px; }
.hd-sub { color: var(--text-2); margin: 0; }
.btn {
  display: inline-flex; align-items: center; gap: 6px; padding: 7px 12px;
  font: 500 12px var(--font-sans); background: transparent; color: var(--text);
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm); cursor: pointer;
}
.btn:disabled { opacity: 0.5; cursor: default; }

.notice {
  display: flex; align-items: center; gap: 8px; margin: 0 0 12px; padding: 8px 12px;
  color: var(--text-2); background: var(--elevated); border: 1px solid var(--border);
}
.tag { padding: 2px 6px; border-radius: var(--radius-sm); font-size: 9.5px; font-weight: 700; letter-spacing: 0.14em; background: rgba(245,158,11,0.16); color: var(--warn); }
.err { margin: 0 0 12px; padding: 8px 12px; color: var(--neg); background: var(--neg-soft); border: 1px solid var(--neg); }
.link { background: none; border: none; padding: 0; margin-left: 6px; color: var(--accent); font: inherit; cursor: pointer; text-decoration: none; }
.link:hover { text-decoration: underline; }

.hero { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; background: var(--border); border: 1px solid var(--border); margin-bottom: 12px; }
.kpi { background: var(--elevated); padding: 14px 16px; min-width: 0; }
.lbl { font: 600 9.5px var(--font-mono); letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3); margin-bottom: 6px; }
.lbl .dim { text-transform: none; letter-spacing: 0; font-weight: 500; }
.val { font-size: 24px; font-weight: 600; letter-spacing: -0.01em; }
.sub { margin-top: 4px; color: var(--text-2); font-size: 11.5px; }

.grid { display: grid; grid-template-columns: minmax(0, 1fr) 380px; gap: 12px; align-items: start; }
.rail { display: flex; flex-direction: column; gap: 12px; min-width: 0; }
.card { background: var(--elevated); border: 1px solid var(--border); min-width: 0; overflow-x: auto; }
.card-h { display: flex; justify-content: space-between; align-items: center; padding: 10px 14px; border-bottom: 1px solid var(--border); }
.card-h h2 { margin: 0; font: 600 12.5px var(--font-sans); }
.state { padding: 28px 16px; text-align: center; color: var(--text-2); }

.tbl { width: 100%; border-collapse: collapse; font-size: 12px; }
.tbl th {
  text-align: left; font: 600 9.5px var(--font-mono); letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3); padding: 9px 10px; border-bottom: 1px solid var(--border); background: var(--canvas); white-space: nowrap;
}
.tbl th.r { text-align: right; }
.tbl td { padding: 8px 10px; border-bottom: 1px solid var(--border); white-space: nowrap; }
.chev { width: 22px; padding-left: 14px !important; color: var(--text-3); }
.row { cursor: pointer; }
.row:hover { background: rgba(255,255,255,0.025); }
.row.open { background: rgba(74,144,226,0.06); }
.mkt { font-weight: 600; letter-spacing: 0.04em; }
.pct { display: block; font-size: 10.5px; font-weight: 500; opacity: 0.8; }
.side { display: inline-block; padding: 2px 6px; border-radius: var(--radius-sm); font-size: 9.5px; font-weight: 700; letter-spacing: 0.14em; }
.side.long { background: var(--pos-soft); color: var(--pos); }
.side.short { background: var(--neg-soft); color: var(--neg); }

.detail-row td { padding: 0 !important; background: var(--canvas); }
.detail { padding: 12px 20px 14px; }
.d-eyebrow { font: 700 9.5px var(--font-mono); letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3); margin-bottom: 8px; }
.fills { width: 100%; border-collapse: collapse; font-size: 11.5px; }
.fills th { text-align: left; font: 600 9px var(--font-mono); letter-spacing: 0.12em; color: var(--text-3); padding: 4px 6px; border-bottom: 1px solid var(--border); }
.fills th.r { text-align: right; }
.fills td { padding: 4px 6px; border-bottom: 1px solid var(--border); }

.bar { display: flex; height: 10px; margin: 14px; background: var(--canvas); }
.legend { list-style: none; margin: 0; padding: 0 14px 12px; }
.legend li { display: grid; grid-template-columns: 10px 1fr auto 52px; gap: 8px; align-items: center; padding: 4px 0; font-size: 11.5px; }
.legend li > :last-child { text-align: right; }
.sw { width: 8px; height: 8px; }

.feed { list-style: none; margin: 0; padding: 4px 14px 10px; }
.feed li { display: grid; grid-template-columns: 86px 34px 76px minmax(0, 1fr); gap: 8px; align-items: center; padding: 6px 0; border-bottom: 1px solid var(--border); font-size: 11.5px; white-space: nowrap; }
.feed li > * { overflow: hidden; text-overflow: ellipsis; }
.feed li:last-child { border-bottom: none; }
.kind { font: 700 9.5px var(--font-mono); letter-spacing: 0.1em; color: var(--text-2); }
.kind.cash { color: var(--warn); }

.foot { margin: 14px 0 0; font-size: 11.5px; }

@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .portfolio-page { padding: 16px 14px 28px; }
  .hero { grid-template-columns: repeat(2, 1fr); }
  .hd { flex-direction: column; align-items: flex-start; gap: 10px; }
}
</style>
