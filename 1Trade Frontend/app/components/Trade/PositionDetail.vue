<script setup lang="ts">
/**
 * Trade/PositionDetail — drawer for one position (E5).
 * Dark, in-app. Open with usePositionDetail().open(id).
 * Footer actions: Close · Reduce (slider) · Convert to limit · Set stop.
 */
import { X, AlertTriangle } from 'lucide-vue-next'

const pd = usePositionDetail()
const toasts = useToasts()

interface Lot {
  id: string
  ts: string
  side: 'buy' | 'sell'
  qty: number
  px: number
}

interface Position {
  id: string
  market: string
  marketDesc: string
  side: 'long' | 'short'
  size: number
  avgCost: number
  markPx: number
  currentValue: number
  unrealizedPnL: number
  realizedPnL: number
  pnlPct: number
  precision: number
  lots: Lot[]
  fills30: { day: string; pnl: number }[]
}

const MOCK: Record<string, Position> = {
  'pos_eai_long': {
    id: 'pos_eai_long',
    market: 'EAI-IDX',
    marketDesc: 'AI Index · spot',
    side: 'long',
    size: 50000,
    avgCost: 0.000980,
    markPx: 0.001005,
    currentValue: 50.25,
    unrealizedPnL: 1.25,
    realizedPnL: 0.42,
    pnlPct: 2.55,
    precision: 6,
    lots: [
      { id: 'l1', ts: '2026-05-21 09:42', side: 'buy', qty: 30000, px: 0.000975 },
      { id: 'l2', ts: '2026-05-22 14:08', side: 'buy', qty: 15000, px: 0.000984 },
      { id: 'l3', ts: '2026-05-23 11:12', side: 'buy', qty:  5000, px: 0.000990 },
    ],
    fills30: Array.from({ length: 30 }, (_, i) => ({
      day: 'd-' + (29 - i),
      pnl: (Math.random() - 0.42) * 0.4,
    })),
  },
  'pos_h100_long': {
    id: 'pos_h100_long',
    market: 'H100-SPOT',
    marketDesc: 'H100 GPU-hour · spot',
    side: 'long',
    size: 8,
    avgCost: 2.95,
    markPx: 2.99,
    currentValue: 23.92,
    unrealizedPnL: 0.32,
    realizedPnL: 0.0,
    pnlPct: 1.36,
    precision: 2,
    lots: [
      { id: 'l1', ts: '2026-05-23 16:04', side: 'buy', qty: 8, px: 2.95 },
    ],
    fills30: Array.from({ length: 30 }, (_, i) => ({
      day: 'd-' + (29 - i),
      pnl: (Math.random() - 0.5) * 0.6,
    })),
  },
}

const pos = computed<Position | null>(() => {
  if (!pd.positionId.value) return null
  return MOCK[pd.positionId.value] ?? null
})

// Reduce slider state
const reducePct = ref(100)

// Action confirmation state
type ActionKind = 'close' | 'reduce' | 'limit' | 'stop' | null
const pendingAction = ref<ActionKind>(null)

function confirmAction() {
  if (!pos.value || !pendingAction.value) return
  const action = pendingAction.value
  const m = pos.value.market
  pendingAction.value = null

  if (action === 'close') {
    toasts.push({ tone: 'pos', title: 'Position closed', body: 'Market sell · ' + m + ' · ' + pos.value.size.toLocaleString() + ' credits — fill logged in /history.' })
  } else if (action === 'reduce') {
    const qty = Math.round(pos.value.size * (reducePct.value / 100))
    toasts.push({ tone: 'pos', title: 'Position reduced', body: 'Closed ' + qty.toLocaleString() + ' of ' + m + ' (' + reducePct.value + '%).' })
  } else if (action === 'limit') {
    toasts.push({ tone: 'info', title: 'Limit converted', body: 'Opened limit close at mid for ' + m + '.' })
  } else if (action === 'stop') {
    toasts.push({ tone: 'info', title: 'Stop set', body: 'Trailing stop armed on ' + m + '.' })
  }
  pd.close()
}

function fmtPx(px: number, p: number): string {
  return '$' + (p === 2 ? px.toFixed(2) : px.toFixed(p))
}
function fmtPnL(n: number): string {
  const sign = n >= 0 ? '+' : '-'
  return sign + '$' + Math.abs(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
function fmtQty(n: number): string {
  return n.toLocaleString('en-US')
}

function onKey(e: KeyboardEvent) {
  if (!pd.isOpen.value) return
  if (e.key === 'Escape') {
    if (pendingAction.value) pendingAction.value = null
    else pd.close()
  }
}
onMounted(() => { document.addEventListener('keydown', onKey) })
onBeforeUnmount(() => { document.removeEventListener('keydown', onKey) })

const projectedCrystallization = computed(() => {
  if (!pos.value) return 0
  if (!pendingAction.value) return 0
  if (pendingAction.value === 'close')  return pos.value.unrealizedPnL
  if (pendingAction.value === 'reduce') return pos.value.unrealizedPnL * (reducePct.value / 100)
  return 0
})

const actionCopy = computed(() => {
  if (!pendingAction.value || !pos.value) return null
  const p = pos.value
  if (pendingAction.value === 'close') return {
    title: 'Close ' + p.market + ' position',
    body: 'Market sell ' + fmtQty(p.size) + ' credits at the prevailing bid. This will crystallize ' + fmtPnL(projectedCrystallization.value) + ' of P&L and write a fill to /history.',
    cta: 'Close at market',
    danger: true,
  }
  if (pendingAction.value === 'reduce') return {
    title: 'Reduce ' + p.market + ' by ' + reducePct.value + '%',
    body: 'Closes ' + fmtQty(Math.round(p.size * (reducePct.value / 100))) + ' of ' + fmtQty(p.size) + ' credits. Estimated realized P&L: ' + fmtPnL(projectedCrystallization.value) + '.',
    cta: 'Reduce position',
    danger: false,
  }
  if (pendingAction.value === 'limit') return {
    title: 'Convert to limit close',
    body: 'Opens a limit-sell at the current mid (' + fmtPx(p.markPx, p.precision) + ') for the full position. The position stays open until the limit fills.',
    cta: 'Place limit',
    danger: false,
  }
  if (pendingAction.value === 'stop') return {
    title: 'Set trailing stop on ' + p.market,
    body: '5% trailing stop arms below the mark. If price falls 5% from any subsequent high, a market sell triggers.',
    cta: 'Arm stop',
    danger: false,
  }
  return null
})
</script>

<template>
  <Teleport to="body">
    <Transition name="pd-bg">
      <div v-if="pd.isOpen.value" class="pd-backdrop" @click="pd.close()" />
    </Transition>
    <Transition name="pd">
      <aside
        v-if="pd.isOpen.value && pos"
        class="pd-drawer"
        role="dialog"
        aria-modal="true"
        :aria-label="'Position ' + pos.market"
      >
        <header class="pd-head">
          <div class="pd-mkt-row">
            <span class="pd-mkt mono">{{ pos.market }}</span>
            <span class="pd-desc">{{ pos.marketDesc }}</span>
            <span class="pd-side" :class="pos.side">{{ pos.side.toUpperCase() }}</span>
          </div>
          <div class="pd-size mono">{{ fmtQty(pos.size) }} credits</div>
          <button class="pd-close" aria-label="Close drawer" type="button" @click="pd.close()">
            <X :size="18" />
          </button>
        </header>

        <!-- KPI strip -->
        <section class="pd-kpis">
          <div class="pd-kpi">
            <div class="kpi-label">Avg cost</div>
            <div class="kpi-val mono">{{ fmtPx(pos.avgCost, pos.precision) }}</div>
          </div>
          <div class="pd-kpi">
            <div class="kpi-label">Mark</div>
            <div class="kpi-val mono">{{ fmtPx(pos.markPx, pos.precision) }}</div>
          </div>
          <div class="pd-kpi">
            <div class="kpi-label">Current value</div>
            <div class="kpi-val mono">${{ pos.currentValue.toFixed(2) }}</div>
          </div>
          <div class="pd-kpi">
            <div class="kpi-label">Unrealized P&L</div>
            <div class="kpi-val mono" :class="pos.unrealizedPnL >= 0 ? 'pos' : 'neg'">
              {{ fmtPnL(pos.unrealizedPnL) }}
            </div>
            <div class="kpi-sub" :class="pos.pnlPct >= 0 ? 'pos' : 'neg'">
              {{ pos.pnlPct >= 0 ? '▲' : '▼' }} {{ Math.abs(pos.pnlPct).toFixed(2) }}%
            </div>
          </div>
        </section>

        <!-- Lifetime realized -->
        <section class="pd-section">
          <div class="pd-eyebrow">Realized P&L · lifetime</div>
          <div class="lifetime mono" :class="pos.realizedPnL >= 0 ? 'pos' : 'neg'">
            {{ fmtPnL(pos.realizedPnL) }}
          </div>
        </section>

        <!-- Lots (FIFO) -->
        <section class="pd-section">
          <div class="pd-eyebrow">Open lots · FIFO</div>
          <table class="pd-lots">
            <thead>
              <tr><th>When</th><th>Side</th><th class="r">Qty</th><th class="r">Price</th></tr>
            </thead>
            <tbody>
              <tr v-for="l in pos.lots" :key="l.id">
                <td class="mono">{{ l.ts }}</td>
                <td><span class="lot-side" :class="l.side">{{ l.side.toUpperCase() }}</span></td>
                <td class="r mono">{{ fmtQty(l.qty) }}</td>
                <td class="r mono">{{ fmtPx(l.px, pos.precision) }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- 30-day fills strip -->
        <section class="pd-section">
          <div class="pd-eyebrow">Daily P&L · last 30 days</div>
          <div class="strip">
            <span
              v-for="(f, i) in pos.fills30"
              :key="i"
              class="strip-bar"
              :class="f.pnl >= 0 ? 'pos' : 'neg'"
              :style="{ height: Math.min(28, Math.max(2, Math.abs(f.pnl) * 50)) + 'px' }"
              :title="f.day + ' · ' + fmtPnL(f.pnl)"
            />
          </div>
        </section>

        <!-- Reduce slider (lives near actions for context) -->
        <section class="pd-section">
          <div class="pd-eyebrow">Reduce size</div>
          <div class="reduce-row">
            <input v-model.number="reducePct" type="range" min="0" max="100" step="5" class="reduce-slider" />
            <span class="reduce-pct mono">{{ reducePct }}%</span>
            <span class="reduce-qty mono">= {{ fmtQty(Math.round(pos.size * (reducePct / 100))) }} credits</span>
          </div>
        </section>

        <!-- Footer actions -->
        <footer class="pd-foot">
          <div class="pd-acts">
            <button type="button" class="btn danger" @click="pendingAction = 'close'">Close position</button>
            <button type="button" class="btn ghost"  @click="pendingAction = 'reduce'">Reduce</button>
            <button type="button" class="btn ghost"  @click="pendingAction = 'limit'">Convert to limit</button>
            <button type="button" class="btn ghost"  @click="pendingAction = 'stop'">Set stop</button>
          </div>
        </footer>

        <!-- Confirmation overlay -->
        <Transition name="confirm">
          <div v-if="pendingAction && actionCopy" class="pd-confirm">
            <div class="confirm-card">
              <div class="confirm-icon">
                <AlertTriangle :size="20" :stroke-width="1.8" />
              </div>
              <h3 class="confirm-title">{{ actionCopy.title }}</h3>
              <p class="confirm-body">{{ actionCopy.body }}</p>
              <div class="confirm-acts">
                <button type="button" class="btn ghost" @click="pendingAction = null">Cancel</button>
                <button type="button" class="btn" :class="{ danger: actionCopy.danger, primary: !actionCopy.danger }" @click="confirmAction">
                  {{ actionCopy.cta }}
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.pd-backdrop {
  position: fixed; inset: 0;
  background: rgba(5, 6, 8, 0.55);
  z-index: 900;
}
.pd-drawer {
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
.pd-drawer .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.pd-drawer .pos { color: var(--pos); }
.pd-drawer .neg { color: var(--neg); }

/* Head */
.pd-head { padding: 18px 20px 14px; border-bottom: 1px solid var(--border); position: relative; }
.pd-mkt-row { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
.pd-mkt { font-weight: 700; font-size: 15px; color: var(--text); }
.pd-desc { color: var(--text-2); font-size: 11.5px; }
.pd-side {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: 0.14em; padding: 2px 6px; border-radius: var(--radius-sm);
}
.pd-side.long  { background: var(--pos-soft); color: var(--pos); }
.pd-side.short { background: var(--neg-soft); color: var(--neg); }
.pd-size { font-size: 13px; color: var(--text-2); }
.pd-close {
  position: absolute; top: 14px; right: 14px;
  width: 28px; height: 28px;
  background: transparent; color: var(--text-2);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.pd-close:hover { color: var(--text); border-color: var(--text); }

/* KPI strip */
.pd-kpis {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--border);
  border-bottom: 1px solid var(--border);
}
.pd-kpi { background: var(--overlay, var(--elevated)); padding: 10px 12px; }
.kpi-label {
  font-family: var(--font-mono); font-size: 9px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 4px;
}
.kpi-val { font-size: 15px; font-weight: 600; color: var(--text); }
.kpi-sub { font-size: 10px; margin-top: 2px; font-family: var(--font-mono); }

/* Sections */
.pd-section { padding: 14px 20px; border-bottom: 1px solid var(--border); }
.pd-eyebrow {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3); margin-bottom: 10px;
}
.lifetime { font-size: 22px; font-weight: 600; }

/* Lots */
.pd-lots { width: 100%; border-collapse: collapse; font-size: 11.5px; }
.pd-lots thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  padding: 6px 4px; border-bottom: 1px solid var(--border);
}
.pd-lots th.r, .pd-lots td.r { text-align: right; }
.pd-lots tbody td { padding: 6px 4px; border-bottom: 1px solid var(--border); }
.lot-side {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.12em; padding: 1px 5px; border-radius: var(--radius-sm);
}
.lot-side.buy  { background: var(--pos-soft); color: var(--pos); }
.lot-side.sell { background: var(--neg-soft); color: var(--neg); }

/* Strip */
.strip {
  display: flex; gap: 2px; align-items: flex-end; height: 32px;
}
.strip-bar { width: 8px; background: var(--text-3); border-radius: 1px 1px 0 0; opacity: 0.85; }
.strip-bar.pos { background: var(--pos); }
.strip-bar.neg { background: var(--neg); }

/* Reduce row */
.reduce-row { display: flex; align-items: center; gap: 12px; }
.reduce-slider { flex: 1; accent-color: var(--brand); }
.reduce-pct { font-size: 14px; font-weight: 600; width: 48px; text-align: right; }
.reduce-qty { font-size: 11px; color: var(--text-3); }

/* Footer */
.pd-foot {
  margin-top: auto;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
  background: rgba(255,255,255,0.02);
}
.pd-acts { display: grid; grid-template-columns: repeat(2, 1fr); gap: 6px; }
.btn {
  padding: 9px 12px;
  font-family: var(--font-sans); font-size: 12px; font-weight: 600;
  background: transparent; color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease;
}
.btn:hover:not(:disabled) { background: rgba(255,255,255,0.04); border-color: var(--text); }
.btn.danger { background: var(--neg-soft); border-color: var(--neg); color: var(--neg); }
.btn.danger:hover { background: var(--neg); color: #fff; }
.btn.primary { background: var(--brand); border-color: var(--brand); color: var(--text-on-accent); }
.btn.primary:hover { background: var(--brand-hov); }

/* Confirmation modal overlay */
.pd-confirm {
  position: absolute; inset: 0;
  background: rgba(5, 6, 8, 0.75);
  display: flex; align-items: center; justify-content: center;
  padding: 20px;
  z-index: 10;
}
.confirm-card {
  width: 100%;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 20px;
}
.confirm-icon {
  width: 36px; height: 36px;
  background: var(--neg-soft);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--neg);
  margin-bottom: 10px;
}
.confirm-title {
  font-family: var(--font-display); font-weight: 600;
  font-size: 16px; margin: 0 0 6px;
}
.confirm-body { color: var(--text-2); font-size: 12.5px; line-height: 1.55; margin: 0 0 16px; }
.confirm-acts { display: flex; justify-content: flex-end; gap: 8px; }

/* Transitions */
.pd-bg-enter-active, .pd-bg-leave-active { transition: opacity 180ms ease; }
.pd-bg-enter-from, .pd-bg-leave-to { opacity: 0; }
.pd-enter-active, .pd-leave-active { transition: transform 220ms cubic-bezier(0.4, 0, 0.2, 1); }
.pd-enter-from, .pd-leave-to { transform: translateX(100%); }
.confirm-enter-active, .confirm-leave-active { transition: opacity 160ms ease; }
.confirm-enter-from, .confirm-leave-to { opacity: 0; }
</style>
