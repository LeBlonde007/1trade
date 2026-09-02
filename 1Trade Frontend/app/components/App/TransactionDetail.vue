<script setup lang="ts">
/**
 * App/TransactionDetail — drawer for one ledger row (F4).
 * Dark. Mount once in app layout.
 */
import { X, ArrowDownLeft, ArrowUpRight, RefreshCw, CreditCard, Receipt, Download, Copy, CheckCircle2, ShieldCheck } from 'lucide-vue-next'

const td = useTxDetail()

interface FeeLine { label: string; amount: number }
interface Tx {
  id: string
  type: 'buy' | 'sell' | 'conversion' | 'deposit' | 'withdrawal' | 'fee'
  ts: string
  amountIn?:  { ccy: string; value: number; precision: number }
  amountOut?: { ccy: string; value: number; precision: number }
  counterparty: string
  settlementState: 'settled' | 'pending' | 'failed'
  settledAt?: string
  fees: FeeLine[]
  orderId?: string
  audit: { block: number; hash: string }
  paymentMethod?: string
  geo?: string
}

const MOCK: Record<string, Tx> = {
  'tx_001': {
    id: 'tx_2026_05_24_8c2a',
    type: 'buy',
    ts: '2026-05-24T14:31:55.108Z',
    amountIn:  { ccy: 'EAI-IDX', value: 5000,   precision: 0 },
    amountOut: { ccy: 'USD',     value: 5.025,  precision: 4 },
    counterparty: 'venue · matching engine',
    settlementState: 'settled',
    settledAt: '2026-05-24T14:31:55.108Z',
    fees: [
      { label: 'Venue fee',        amount: 0.050 },
      { label: 'Matching rebate', amount: -0.005 },
    ],
    orderId: 'ord_8c2a48f1',
    audit: { block: 14286, hash: '9e0d8a1ac11' },
    geo: 'Bentonville, AR · US',
  },
  'tx_002': {
    id: 'tx_2026_05_24_b341',
    type: 'deposit',
    ts: '2026-05-24T13:09:47.612Z',
    amountIn:  { ccy: 'USD', value: 100000, precision: 2 },
    counterparty: 'Wire · Chase ••4421',
    settlementState: 'settled',
    settledAt: '2026-05-24T13:09:51.422Z',
    fees: [{ label: 'Wire fee', amount: 15.00 }],
    audit: { block: 14280, hash: 'b341a2ce0a4' },
    paymentMethod: 'Wire · Chase ••4421',
  },
  'tx_003': {
    id: 'tx_2026_05_24_2c8e',
    type: 'conversion',
    ts: '2026-05-24T10:18:44.331Z',
    amountIn:  { ccy: 'TEXT-SPOT', value: 8000, precision: 0 },
    amountOut: { ccy: 'EAI-IDX',   value: 9620, precision: 0 },
    counterparty: 'venue · book sweep',
    settlementState: 'settled',
    settledAt: '2026-05-24T10:18:44.402Z',
    fees: [{ label: 'Conversion fee', amount: 0.024 }],
    audit: { block: 14273, hash: 'ad08c46e2f0' },
  },
}

const tx = computed<Tx | null>(() => {
  if (!td.txId.value) return null
  return MOCK[td.txId.value] ?? null
})

const titleFor: Record<Tx['type'], string> = {
  buy: 'Credit purchase',
  sell: 'Credit sale',
  conversion: 'Credit conversion',
  deposit: 'Deposit',
  withdrawal: 'Withdrawal',
  fee: 'Fee',
}

const iconFor: Record<Tx['type'], unknown> = {
  buy: ArrowDownLeft,
  sell: ArrowUpRight,
  conversion: RefreshCw,
  deposit: CreditCard,
  withdrawal: ArrowUpRight,
  fee: Receipt,
}

function fmtAmount(a: { ccy: string; value: number; precision: number } | undefined): string {
  if (!a) return ''
  if (a.ccy === 'USD') return '$' + a.value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: a.precision })
  return a.value.toLocaleString('en-US') + ' ' + a.ccy
}
function fmtFee(n: number): string {
  const sign = n < 0 ? '-$' : '$'
  return sign + Math.abs(n).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })
}

const totalFee = computed(() => tx.value?.fees.reduce((s, f) => s + f.amount, 0) ?? 0)

const copied = ref<string | null>(null)
function copy(s: string, key: string) {
  if (navigator.clipboard) navigator.clipboard.writeText(s).catch(() => {})
  copied.value = key
  setTimeout(() => { if (copied.value === key) copied.value = null }, 1200)
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && td.isOpen.value) td.close()
}
onMounted(() => { document.addEventListener('keydown', onKey) })
onBeforeUnmount(() => { document.removeEventListener('keydown', onKey) })
</script>

<template>
  <Teleport to="body">
    <Transition name="td-bg">
      <div v-if="td.isOpen.value" class="td-backdrop" @click="td.close()" />
    </Transition>
    <Transition name="td">
      <aside
        v-if="td.isOpen.value && tx"
        class="td-drawer"
        role="dialog"
        aria-modal="true"
        :aria-label="'Transaction ' + tx.id"
      >
        <header class="td-head">
          <div class="td-icon"><component :is="iconFor[tx.type]" :size="20" :stroke-width="1.6" /></div>
          <div class="td-title-block">
            <div class="td-title">{{ titleFor[tx.type] }}</div>
            <div class="td-id-row">
              <span class="td-id mono">{{ tx.id }}</span>
              <button class="td-copy" type="button" :title="'Copy ' + tx.id" @click="copy(tx.id, 'id')">
                <Copy v-if="copied !== 'id'" :size="11" />
                <CheckCircle2 v-else :size="11" />
              </button>
            </div>
          </div>
          <span class="td-state" :class="'st-' + tx.settlementState">
            <span class="dot" />{{ tx.settlementState.toUpperCase() }}
          </span>
          <button class="td-close" aria-label="Close" type="button" @click="td.close()">
            <X :size="18" />
          </button>
        </header>

        <!-- Amounts -->
        <section class="td-amounts">
          <div v-if="tx.amountIn" class="amount">
            <div class="amount-label">Amount in</div>
            <div class="amount-val mono pos">+ {{ fmtAmount(tx.amountIn) }}</div>
          </div>
          <div v-if="tx.amountOut" class="amount">
            <div class="amount-label">Amount out</div>
            <div class="amount-val mono">{{ fmtAmount(tx.amountOut) }}</div>
          </div>
        </section>

        <!-- KV details -->
        <section class="td-section">
          <div class="td-eyebrow">Settlement</div>
          <dl class="td-kv">
            <dt>Counterparty</dt>     <dd>{{ tx.counterparty }}</dd>
            <dt>Initiated</dt>        <dd class="mono">{{ tx.ts }}</dd>
            <dt v-if="tx.settledAt">Settled</dt>
            <dd v-if="tx.settledAt" class="mono">{{ tx.settledAt }}</dd>
            <dt v-if="tx.paymentMethod">Payment</dt>
            <dd v-if="tx.paymentMethod" class="mono">{{ tx.paymentMethod }}</dd>
            <dt v-if="tx.geo">Origin</dt>
            <dd v-if="tx.geo" class="mono">{{ tx.geo }}</dd>
          </dl>
        </section>

        <!-- Fees -->
        <section class="td-section">
          <div class="td-eyebrow">Fees</div>
          <table class="td-fees">
            <tbody>
              <tr v-for="(f, i) in tx.fees" :key="i">
                <td class="fee-lbl">{{ f.label }}</td>
                <td class="fee-amt mono" :class="f.amount < 0 ? 'pos' : ''">{{ fmtFee(f.amount) }}</td>
              </tr>
              <tr class="fee-total">
                <td class="fee-lbl">Net fees</td>
                <td class="fee-amt mono" :class="totalFee < 0 ? 'pos' : ''">{{ fmtFee(totalFee) }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <!-- Linked order -->
        <section v-if="tx.orderId" class="td-section">
          <div class="td-eyebrow">Linked order</div>
          <div class="linked-order mono">{{ tx.orderId }}</div>
          <button type="button" class="link-row" @click="() => { td.close(); useOrderDetail().open(tx!.orderId!) }">
            View order detail →
          </button>
        </section>

        <!-- Audit -->
        <section class="td-section audit">
          <div class="td-eyebrow"><ShieldCheck :size="11" class="audit-icon" /> Audit chain</div>
          <dl class="td-kv">
            <dt>Block</dt>
            <dd class="mono">#{{ tx.audit.block.toLocaleString() }}</dd>
            <dt>Hash</dt>
            <dd class="mono hash-row">
              <span>{{ tx.audit.hash }}…</span>
              <button class="td-copy" type="button" :title="'Copy ' + tx.audit.hash" @click="copy(tx.audit.hash, 'hash')">
                <Copy v-if="copied !== 'hash'" :size="11" />
                <CheckCircle2 v-else :size="11" />
              </button>
            </dd>
          </dl>
          <NuxtLink :to="'/enterprise/audit?hash=' + tx.audit.hash" class="audit-link" @click="td.close()">
            Verify in audit log →
          </NuxtLink>
        </section>

        <!-- Footer -->
        <footer class="td-foot">
          <button class="btn ghost" type="button">
            <Download :size="13" />
            Download receipt (PDF)
          </button>
          <button class="btn ghost" type="button">Dispute</button>
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.td-backdrop {
  position: fixed; inset: 0;
  background: rgba(5, 6, 8, 0.55);
  z-index: 900;
}
.td-drawer {
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 460px;
  background: var(--overlay, var(--elevated));
  border-left: 1px solid var(--border-strong);
  box-shadow: -16px 0 36px rgba(0, 0, 0, 0.45);
  display: flex; flex-direction: column;
  color: var(--text);
  font-family: var(--font-sans);
  z-index: 1000;
  overflow-y: auto;
}
.td-drawer .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.td-drawer .pos { color: var(--pos); }
.td-drawer .neg { color: var(--neg); }

.td-head {
  display: grid;
  grid-template-columns: 36px 1fr auto;
  gap: 12px;
  align-items: flex-start;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  position: relative;
}
.td-icon {
  width: 36px; height: 36px;
  background: rgba(255,255,255,0.04);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--text);
}
.td-title-block { min-width: 0; }
.td-title { font-weight: 600; font-size: 14px; color: var(--text); }
.td-id-row { display: flex; align-items: center; gap: 6px; margin-top: 3px; }
.td-id { font-size: 11px; color: var(--text-2); }
.td-copy {
  width: 20px; height: 20px;
  background: transparent; color: var(--text-3);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.td-copy:hover { color: var(--text); border-color: var(--text); }

.td-state {
  position: absolute;
  top: 16px; right: 56px;
  display: inline-flex; align-items: center; gap: 5px;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
}
.td-state .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.st-settled { background: var(--pos-soft); color: var(--pos); }
.st-pending { background: rgba(245,158,11,0.16); color: var(--warn); }
.st-failed  { background: var(--neg-soft); color: var(--neg); }

.td-close {
  position: absolute; top: 12px; right: 14px;
  width: 28px; height: 28px;
  background: transparent; color: var(--text-2);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.td-close:hover { color: var(--text); border-color: var(--text); }

/* Amount cards */
.td-amounts {
  display: grid; grid-template-columns: 1fr 1fr; gap: 1px;
  background: var(--border);
  border-bottom: 1px solid var(--border);
}
.amount { background: var(--overlay, var(--elevated)); padding: 14px 18px; }
.amount-label {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 6px;
}
.amount-val { font-size: 19px; font-weight: 600; color: var(--text); }

/* Sections */
.td-section { padding: 14px 20px; border-bottom: 1px solid var(--border); }
.td-section.audit { background: rgba(74,144,226,0.03); }
.td-eyebrow {
  display: inline-flex; align-items: center; gap: 5px;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3); margin-bottom: 10px;
}
.audit-icon { color: var(--accent); }

.td-kv {
  display: grid; grid-template-columns: 110px 1fr;
  row-gap: 5px; column-gap: 12px; margin: 0;
  font-size: 12px;
}
.td-kv dt {
  font-family: var(--font-mono); font-size: 10px;
  letter-spacing: 0.06em; text-transform: uppercase;
  color: var(--text-3);
}
.td-kv dd { margin: 0; color: var(--text); word-break: break-all; }

.td-fees { width: 100%; border-collapse: collapse; font-size: 12px; }
.td-fees tbody td { padding: 5px 0; border-bottom: 1px solid var(--border); }
.td-fees tbody tr:last-child td { border-bottom: none; }
.td-fees .fee-amt { text-align: right; font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.fee-total td { font-weight: 600; padding-top: 8px !important; }

.linked-order {
  font-size: 12px; color: var(--text); font-weight: 500;
  margin-bottom: 6px;
}
.link-row {
  background: transparent; border: none; padding: 0;
  color: var(--accent); font-size: 11.5px; cursor: pointer;
  font-family: var(--font-sans);
}
.link-row:hover { text-decoration: underline; }

.hash-row { display: flex; align-items: center; gap: 6px; }
.audit-link {
  display: inline-block;
  margin-top: 8px;
  color: var(--accent);
  text-decoration: none;
  font-size: 11.5px;
}
.audit-link:hover { text-decoration: underline; }

.td-foot {
  margin-top: auto;
  padding: 12px 20px;
  display: flex; justify-content: space-between; gap: 8px;
  border-top: 1px solid var(--border);
  background: rgba(255,255,255,0.02);
}
.btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 12px;
  font-family: var(--font-sans); font-size: 12px; font-weight: 500;
  background: transparent; color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.btn:hover { background: rgba(255,255,255,0.04); border-color: var(--text); }

.td-bg-enter-active, .td-bg-leave-active { transition: opacity 180ms ease; }
.td-bg-enter-from, .td-bg-leave-to { opacity: 0; }
.td-enter-active, .td-leave-active { transition: transform 220ms cubic-bezier(0.4, 0, 0.2, 1); }
.td-enter-from, .td-leave-to { transform: translateX(100%); }
</style>
