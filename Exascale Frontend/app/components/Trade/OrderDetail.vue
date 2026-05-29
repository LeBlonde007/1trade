<script setup lang="ts">
/**
 * Trade/OrderDetail — slide-in drawer for a single order (E1).
 * Dark, in-app. Mounted globally in app layout; opened by useOrderDetail().
 */
import { X, CheckCircle2, Circle, Copy, ExternalLink } from 'lucide-vue-next'

const od = useOrderDetail()

// Mock order lookup — in real impl would fetch by id.
interface Fill {
  id: string
  ts: string         // HH:MM:SS.mmm
  qty: number
  px: number
  fee: number
  counterparty: string
}
interface OrderInfo {
  id: string
  market: string
  marketDesc: string
  side: 'buy' | 'sell'
  type: 'market' | 'limit' | 'stop' | 'stop-limit'
  status: 'FILLED' | 'WORKING' | 'PARTIAL' | 'CANCELLED' | 'REJECTED'
  qty: number
  filledQty: number
  avgPx: number
  limitPx?: number
  stopPx?: number
  tif: 'GTC' | 'IOC' | 'FOK' | 'DAY'
  subAccount: string
  ip: string
  client: 'web' | 'api' | 'palette'
  placedAt: string
  routedAt?: string
  firstFillAt?: string
  finalAt?: string
  fills: Fill[]
  routedVenues: number
  slippageBps: number
  queuePos?: number
  timeOnBookMs: number
}

const MOCK: Record<string, OrderInfo> = {
  ord_8c2a48f1: {
    id: 'ord_8c2a48f1',
    market: 'EAI-IDX',
    marketDesc: 'AI Index · spot',
    side: 'buy',
    type: 'market',
    status: 'FILLED',
    qty: 5000,
    filledQty: 5000,
    avgPx: 0.001005,
    tif: 'IOC',
    subAccount: 'AI-Research-Team',
    ip: '198.51.100.42',
    client: 'web',
    placedAt: '2026-05-24T14:31:54.992Z',
    routedAt: '2026-05-24T14:31:54.995Z',
    firstFillAt: '2026-05-24T14:31:55.041Z',
    finalAt: '2026-05-24T14:31:55.108Z',
    fills: [
      { id: 'f_001', ts: '14:31:55.041', qty: 1800, px: 0.001005, fee: 0.018, counterparty: 'mm_a14c…' },
      { id: 'f_002', ts: '14:31:55.071', qty: 2400, px: 0.001005, fee: 0.024, counterparty: 'mm_b22e…' },
      { id: 'f_003', ts: '14:31:55.108', qty:  800, px: 0.001006, fee: 0.008, counterparty: 'mm_a14c…' },
    ],
    routedVenues: 3,
    slippageBps: 0.4,
    timeOnBookMs: 116,
  },
}

const order = computed<OrderInfo | null>(() => {
  if (!od.orderId.value) return null
  return MOCK[od.orderId.value] ?? null
})

const statusTone = computed(() => {
  if (!order.value) return 'neutral'
  if (order.value.status === 'FILLED')    return 'pos'
  if (order.value.status === 'PARTIAL')   return 'warn'
  if (order.value.status === 'WORKING')   return 'info'
  if (order.value.status === 'CANCELLED') return 'mute'
  return 'neg'
})

function fmtPx(px: number): string {
  return '$' + (px < 1 ? px.toFixed(6) : px.toFixed(2))
}
function fmtQty(n: number): string {
  return n.toLocaleString('en-US')
}
function fmtUSD(n: number): string {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const copied = ref<string | null>(null)
function copy(s: string, key: string) {
  if (navigator.clipboard) navigator.clipboard.writeText(s).catch(() => {})
  copied.value = key
  setTimeout(() => { if (copied.value === key) copied.value = null }, 1200)
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && od.isOpen.value) od.close()
}

onMounted(() => { document.addEventListener('keydown', onKey) })
onBeforeUnmount(() => { document.removeEventListener('keydown', onKey) })

const totalNotional = computed(() => {
  if (!order.value) return 0
  return order.value.fills.reduce((s, f) => s + f.qty * f.px, 0)
})
const totalFees = computed(() => {
  if (!order.value) return 0
  return order.value.fills.reduce((s, f) => s + f.fee, 0)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="od-bg">
      <div v-if="od.isOpen.value" class="od-backdrop" @click="od.close()" />
    </Transition>
    <Transition name="od">
      <aside
        v-if="od.isOpen.value && order"
        class="od-drawer"
        role="dialog"
        aria-modal="true"
        :aria-label="'Order ' + order.id"
      >
        <header class="od-head">
          <div class="od-id-row">
            <span class="od-id mono">{{ order.id }}</span>
            <button class="od-copy" type="button" :title="'Copy ' + order.id" @click="copy(order.id, 'id')">
              <Copy v-if="copied !== 'id'" :size="12" />
              <CheckCircle2 v-else :size="12" />
            </button>
            <span class="od-status" :class="'st-' + statusTone">
              <span class="dot" />{{ order.status }}
            </span>
          </div>
          <div class="od-mkt-row">
            <span class="od-mkt mono">{{ order.market }}</span>
            <span class="od-mkt-desc">{{ order.marketDesc }}</span>
            <span class="od-side" :class="order.side">{{ order.side.toUpperCase() }}</span>
            <span class="od-type mono">{{ order.type.toUpperCase() }}</span>
            <span class="od-tif mono">{{ order.tif }}</span>
          </div>
          <button class="od-close" aria-label="Close order detail" type="button" @click="od.close()">
            <X :size="18" />
          </button>
        </header>

        <!-- Summary KPIs -->
        <section class="od-kpis">
          <div class="od-kpi">
            <div class="kpi-label">Quantity</div>
            <div class="kpi-val mono">{{ fmtQty(order.filledQty) }} / {{ fmtQty(order.qty) }}</div>
            <div class="kpi-sub">{{ Math.round(order.filledQty / order.qty * 100) }}% filled</div>
          </div>
          <div class="od-kpi">
            <div class="kpi-label">Avg fill price</div>
            <div class="kpi-val mono">{{ fmtPx(order.avgPx) }}</div>
            <div class="kpi-sub">slippage {{ order.slippageBps.toFixed(1) }} bps</div>
          </div>
          <div class="od-kpi">
            <div class="kpi-label">Total notional</div>
            <div class="kpi-val mono">{{ fmtUSD(totalNotional) }}</div>
            <div class="kpi-sub">fee {{ fmtUSD(totalFees) }}</div>
          </div>
        </section>

        <!-- Timeline -->
        <section class="od-section">
          <div class="od-eyebrow">Timeline</div>
          <ol class="od-timeline">
            <li class="tl-step done">
              <span class="tl-icon"><CheckCircle2 :size="12" /></span>
              <span class="tl-text">Placed</span>
              <span class="tl-ts mono">{{ order.placedAt.slice(11, 23) }}</span>
            </li>
            <li v-if="order.routedAt" class="tl-step done">
              <span class="tl-icon"><CheckCircle2 :size="12" /></span>
              <span class="tl-text">Routed · {{ order.routedVenues }} venue<span v-if="order.routedVenues !== 1">s</span></span>
              <span class="tl-ts mono">{{ order.routedAt.slice(11, 23) }}</span>
            </li>
            <li v-if="order.firstFillAt" class="tl-step done">
              <span class="tl-icon"><CheckCircle2 :size="12" /></span>
              <span class="tl-text">First fill</span>
              <span class="tl-ts mono">{{ order.firstFillAt.slice(11, 23) }}</span>
            </li>
            <li v-if="order.finalAt" class="tl-step done">
              <span class="tl-icon"><CheckCircle2 :size="12" /></span>
              <span class="tl-text">Fully filled</span>
              <span class="tl-ts mono">{{ order.finalAt.slice(11, 23) }}</span>
            </li>
            <li v-else class="tl-step pending">
              <span class="tl-icon"><Circle :size="12" /></span>
              <span class="tl-text">Awaiting completion</span>
            </li>
          </ol>
        </section>

        <!-- Fills -->
        <section class="od-section">
          <div class="od-eyebrow">Fills · {{ order.fills.length }}</div>
          <table class="od-fills">
            <thead>
              <tr>
                <th>Time (UTC)</th>
                <th class="r">Qty</th>
                <th class="r">Price</th>
                <th class="r">Fee</th>
                <th>Counterparty</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="f in order.fills" :key="f.id">
                <td class="mono">{{ f.ts }}</td>
                <td class="r mono">{{ fmtQty(f.qty) }}</td>
                <td class="r mono">{{ fmtPx(f.px) }}</td>
                <td class="r mono">{{ fmtUSD(f.fee) }}</td>
                <td class="mono mute">{{ f.counterparty }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- Order params -->
        <section class="od-section">
          <div class="od-eyebrow">Order parameters</div>
          <dl class="od-kv">
            <dt>Side</dt>            <dd>{{ order.side.toUpperCase() }}</dd>
            <dt>Type</dt>            <dd>{{ order.type.toUpperCase() }}</dd>
            <dt>TIF</dt>             <dd class="mono">{{ order.tif }}</dd>
            <dt v-if="order.limitPx">Limit price</dt>
            <dd v-if="order.limitPx" class="mono">{{ fmtPx(order.limitPx) }}</dd>
            <dt v-if="order.stopPx">Stop price</dt>
            <dd v-if="order.stopPx" class="mono">{{ fmtPx(order.stopPx) }}</dd>
            <dt>Sub-account</dt>     <dd>{{ order.subAccount }}</dd>
            <dt>Origin IP</dt>       <dd class="mono">{{ order.ip }}</dd>
            <dt>Client</dt>          <dd class="mono">{{ order.client }}</dd>
          </dl>
        </section>

        <!-- Matching telemetry -->
        <section class="od-section">
          <div class="od-eyebrow">Matching telemetry</div>
          <dl class="od-kv">
            <dt>Routed venues</dt>    <dd class="mono">{{ order.routedVenues }}</dd>
            <dt>Slippage vs mid</dt>  <dd class="mono">{{ order.slippageBps.toFixed(1) }} bps</dd>
            <dt v-if="order.queuePos">Queue position</dt>
            <dd v-if="order.queuePos" class="mono">{{ order.queuePos }}</dd>
            <dt>Time on book</dt>     <dd class="mono">{{ order.timeOnBookMs }} ms</dd>
          </dl>
        </section>

        <!-- Footer actions -->
        <footer class="od-foot">
          <div class="od-acts">
            <button
              type="button"
              class="btn ghost"
              :disabled="order.status !== 'WORKING' && order.status !== 'PARTIAL'"
            >Cancel</button>
            <button
              type="button"
              class="btn ghost"
              :disabled="order.status !== 'WORKING' && order.status !== 'PARTIAL'"
            >Replace</button>
          </div>
          <NuxtLink :to="'/history?order=' + order.id" class="od-foot-link" @click="od.close()">
            View in /history
            <ExternalLink :size="11" />
          </NuxtLink>
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.od-backdrop {
  position: fixed; inset: 0;
  background: rgba(5, 6, 8, 0.55);
  z-index: 900;
}
.od-drawer {
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 480px;
  background: var(--overlay, var(--elevated));
  border-left: 1px solid var(--border-strong);
  box-shadow: -16px 0 36px rgba(0, 0, 0, 0.45);
  display: flex; flex-direction: column;
  color: var(--text);
  font-family: var(--font-sans);
  z-index: 1000;
  overflow-y: auto;
}
.od-drawer .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.od-drawer .mute { color: var(--text-3); }

/* Header */
.od-head {
  padding: 18px 20px 14px;
  border-bottom: 1px solid var(--border);
  position: relative;
}
.od-id-row {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 8px;
}
.od-id {
  font-size: 13px; font-weight: 600; color: var(--text);
  letter-spacing: -0.005em;
}
.od-copy {
  width: 22px; height: 22px;
  background: transparent; color: var(--text-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.od-copy:hover { color: var(--text); border-color: var(--text); }
.od-status {
  margin-left: auto;
  display: inline-flex; align-items: center; gap: 5px;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
}
.od-status .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.st-pos  { background: var(--pos-soft); color: var(--pos); }
.st-warn { background: rgba(245,158,11,0.16); color: var(--warn); }
.st-info { background: rgba(74,144,226,0.16); color: var(--info); }
.st-mute { background: rgba(255,255,255,0.06); color: var(--text-2); }
.st-neg  { background: var(--neg-soft); color: var(--neg); }

.od-mkt-row {
  display: flex; align-items: center; gap: 10px;
  flex-wrap: wrap;
  font-size: 12.5px;
}
.od-mkt { font-weight: 700; color: var(--text); }
.od-mkt-desc { color: var(--text-2); font-size: 11.5px; }
.od-side {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: 0.14em; padding: 2px 6px; border-radius: var(--radius-sm);
}
.od-side.buy  { background: var(--pos-soft); color: var(--pos); }
.od-side.sell { background: var(--neg-soft); color: var(--neg); }
.od-type, .od-tif {
  font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em;
  color: var(--text-3);
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.od-close {
  position: absolute; top: 14px; right: 14px;
  width: 28px; height: 28px;
  background: transparent; color: var(--text-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.od-close:hover { color: var(--text); border-color: var(--text); }

/* KPI strip */
.od-kpis {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px;
  background: var(--border);
  border-bottom: 1px solid var(--border);
}
.od-kpi { background: var(--overlay, var(--elevated)); padding: 12px 14px; }
.kpi-label {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 4px;
}
.kpi-val { font-size: 17px; font-weight: 600; color: var(--text); }
.kpi-sub { font-size: 10.5px; color: var(--text-3); margin-top: 2px; }

/* Sections */
.od-section {
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
}
.od-eyebrow {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3); margin-bottom: 10px;
}

/* Timeline */
.od-timeline { list-style: none; padding: 0; margin: 0; }
.tl-step {
  display: grid; grid-template-columns: 14px 1fr auto;
  gap: 10px; align-items: center;
  padding: 5px 0;
  font-size: 12px;
}
.tl-step.pending .tl-icon { color: var(--text-3); }
.tl-step.done .tl-icon    { color: var(--pos); }
.tl-text { color: var(--text); }
.tl-ts   { color: var(--text-3); font-size: 11px; }
.tl-step.pending .tl-text { color: var(--text-3); }

/* Fills table */
.od-fills {
  width: 100%; border-collapse: collapse;
  font-size: 11.5px;
}
.od-fills thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  padding: 6px 4px; border-bottom: 1px solid var(--border);
}
.od-fills th.r, .od-fills td.r { text-align: right; }
.od-fills tbody td {
  padding: 6px 4px;
  border-bottom: 1px solid var(--border);
}

/* Key/value */
.od-kv {
  display: grid; grid-template-columns: 130px 1fr;
  row-gap: 5px; column-gap: 12px; margin: 0;
  font-size: 12px;
}
.od-kv dt {
  font-family: var(--font-mono); font-size: 10px;
  letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--text-3);
}
.od-kv dd { margin: 0; color: var(--text); }

/* Footer */
.od-foot {
  margin-top: auto;
  padding: 12px 20px;
  display: flex; justify-content: space-between; align-items: center;
  border-top: 1px solid var(--border);
  background: rgba(255,255,255,0.02);
}
.od-acts { display: inline-flex; gap: 8px; }
.btn {
  padding: 7px 12px;
  font-family: var(--font-sans); font-size: 12px; font-weight: 500;
  background: transparent; color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease;
}
.btn:hover:not(:disabled) { background: rgba(255,255,255,0.04); border-color: var(--text); }
.btn:disabled { color: var(--text-3); cursor: default; opacity: 0.6; }
.od-foot-link {
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 11.5px; color: var(--accent); text-decoration: none;
}
.od-foot-link:hover { text-decoration: underline; }

/* Transitions */
.od-bg-enter-active, .od-bg-leave-active { transition: opacity 180ms ease; }
.od-bg-enter-from, .od-bg-leave-to { opacity: 0; }
.od-enter-active, .od-leave-active { transition: transform 220ms cubic-bezier(0.4, 0, 0.2, 1); }
.od-enter-from, .od-leave-to { transform: translateX(100%); }
</style>
