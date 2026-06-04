<script setup lang="ts">
/**
 * /enterprise/billing — Billing & payment. Honest + live (no mock): wired to the real F06 surface via
 * the BFF — the tenant's monthly budget (GET/PUT /api/billing/budget), credit balances + this-month
 * spend (the real ledger, useWallet), and purchase history (GET /api/billing/purchases). Spend/budget
 * are per credit type (there is no live USD feed — the exchange is paused), so we never roll them into
 * a single fabricated dollar figure. Top-ups go through the live Stripe-only /wallet/buy flow; in
 * sandbox MockStripe settles instantly and no card is ever stored.
 */
import { compact, full } from '~/utils/format'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Billing & payment — Exascale' })

interface Budget { credit_type: string; monthly_limit: string }
interface Purchase { id: string; amount: string; credit_type: string; currency: string; status: string; created_at: string }

const { user } = useAuth()
const { balances, transactions, loadBalances, loadTransactions } = useWallet()

const budget = ref<Budget | null>(null)
const budgetInput = ref('')
const purchases = ref<Purchase[]>([])
const loading = ref(true)
const saving = ref(false)
const saveErr = ref('')
const saveOk = ref(false)

/** loadBudget fetches the tenant's monthly budget (null until one is set). */
async function loadBudget() {
  budget.value = (await $fetch<{ budget: Budget | null }>('/api/billing/budget')).budget
  if (budget.value) budgetInput.value = String(Number(budget.value.monthly_limit))
}
/** loadPurchases fetches the real purchase history (most recent first, server-ordered). */
async function loadPurchases() {
  purchases.value = (await $fetch<{ purchases: Purchase[] }>('/api/billing/purchases')).purchases || []
}
/** saveBudget upserts the monthly budget for the budget credit type (billing/admin only → 403 surfaced). */
async function saveBudget() {
  if (!budgetInput.value || saving.value) return
  saving.value = true; saveErr.value = ''; saveOk.value = false
  try {
    await $fetch('/api/billing/budget', {
      method: 'PUT',
      body: { credit_type: budgetCt.value, monthly_limit: Number(budgetInput.value).toFixed(6) },
    })
    await loadBudget()
    saveOk.value = true
    setTimeout(() => { saveOk.value = false }, 2500)
  } catch (e: unknown) {
    const status = (e as { statusCode?: number })?.statusCode
    saveErr.value = status === 403 ? 'Billing or admin role required to change the budget.' : 'Could not save the budget.'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await Promise.allSettled([loadBalances(), loadTransactions(), loadBudget(), loadPurchases()])
  loading.value = false
})

// ── Spend this month (real ledger consumption, per credit type) ───────────────────────────────────
const month = new Date().getMonth()
const year = new Date().getFullYear()
/** usedThisMonth sums absolute consumption for one credit type in the current calendar month. */
function usedThisMonth(ct: string): number {
  return transactions.value
    .filter((t) => t.credit_type === ct && t.operation === 'consumption'
      && new Date(t.created_at).getMonth() === month && new Date(t.created_at).getFullYear() === year)
    .reduce((s, t) => s + Math.abs(Number(t.amount)), 0)
}

const budgetCt = computed(() => budget.value?.credit_type ?? 'text')
const budgetUsed = computed(() => usedThisMonth(budgetCt.value))
const budgetLimit = computed(() => Number(budget.value?.monthly_limit ?? 0))
const budgetPct = computed(() => (budgetLimit.value > 0 ? Math.min(100, (budgetUsed.value / budgetLimit.value) * 100) : 0))
const budgetLevel = computed(() => (budgetPct.value >= 100 ? 'over' : budgetPct.value >= 80 ? 'warn' : budgetPct.value >= 50 ? 'mid' : 'ok'))

// ── Credit balances (funded types first, then anything with month spend) ──────────────────────────
const CREDIT_LABEL: Record<string, string> = {
  text: 'Text', ai_index: 'AI index', speech: 'Speech', image: 'Image', video: 'Video',
  embeddings: 'Embeddings', gpu_h100: 'H100 GPU', gpu_h200: 'H200 GPU',
}
function ctLabel(ct: string): string { return CREDIT_LABEL[ct] ?? ct }
/** rows — every balance the tenant holds, plus credit types they spent this month even at zero balance. */
const rows = computed(() => {
  const byType = new Map<string, { credit_type: string; balance: string; locked: string }>()
  for (const b of balances.value) byType.set(b.credit_type, { credit_type: b.credit_type, balance: b.balance, locked: b.locked_amount })
  for (const t of transactions.value) {
    if (!byType.has(t.credit_type)) byType.set(t.credit_type, { credit_type: t.credit_type, balance: '0', locked: '0' })
  }
  return [...byType.values()]
    .map((r) => ({ ...r, used: usedThisMonth(r.credit_type) }))
    .sort((a, b) => Number(b.balance) - Number(a.balance) || b.used - a.used)
})

// ── Purchases ─────────────────────────────────────────────────────────────────────────────────────
const isPaid = (s: string) => ['paid', 'succeeded', 'completed'].includes(s.toLowerCase())
function statusClass(s: string): string { return isPaid(s) ? 'ok' : s.toLowerCase() === 'pending' ? 'pending' : 'other' }

// ── Formatting ────────────────────────────────────────────────────────────────────────────────────
function dt(s: string): string {
  return new Date(s).toLocaleString('en-US', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}
</script>

<template>
  <div class="billing">
    <header class="head">
      <div>
        <div class="eyebrow">Account · Billing</div>
        <h1 class="title">Billing &amp; payment</h1>
        <p class="sub">Your monthly budget, credit balances, and purchase history — live from the ledger.</p>
      </div>
      <span v-if="user?.is_paper" class="chip paper">sandbox</span>
      <span v-else class="chip live">live</span>
    </header>

    <!-- Monthly budget (live get/set) -->
    <section class="panel">
      <header class="panel-h">
        <span class="panel-title">Monthly budget</span>
        <span class="panel-meta mono">{{ ctLabel(budgetCt) }} credits</span>
      </header>
      <div class="budget">
        <div class="budget-set">
          <label class="lbl">Limit · {{ ctLabel(budgetCt) }} / month</label>
          <div class="set-row">
            <input v-model="budgetInput" class="input mono" inputmode="numeric" placeholder="e.g. 250000" @keydown.enter="saveBudget" />
            <button type="button" class="btn" :disabled="saving || !budgetInput" @click="saveBudget">{{ saving ? 'Saving…' : 'Save' }}</button>
          </div>
          <p v-if="saveErr" class="msg err">{{ saveErr }}</p>
          <p v-else-if="saveOk" class="msg ok">✓ Budget saved.</p>
          <p v-else class="hint">Soft alerts fire at 50 / 80 / 100% of the limit. Usage is metered from the ledger.</p>
        </div>
        <div class="budget-meter">
          <template v-if="budget && budgetLimit > 0">
            <div class="meter-top">
              <span class="meter-used mono" :class="'lvl-' + budgetLevel" :title="full(String(budgetUsed))">{{ compact(String(budgetUsed)) }}</span>
              <span class="meter-of mono">used / {{ compact(budget.monthly_limit) }}</span>
            </div>
            <div class="bar"><div class="fill" :class="'lvl-' + budgetLevel" :style="{ width: budgetPct + '%' }" /></div>
            <div class="meter-foot">
              <span class="mono" :class="'lvl-' + budgetLevel">{{ budgetPct.toFixed(0) }}%</span>
              <span class="muted">this month · resets on the 1st</span>
            </div>
          </template>
          <div v-else class="meter-empty muted">No budget set — spend is unlimited within your balance.</div>
        </div>
      </div>
    </section>

    <div class="grid">
      <!-- Credit balances + month spend -->
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Credit balances</span><span class="panel-meta mono">{{ rows.length }}</span></header>
        <table class="tbl">
          <thead><tr><th>Credit</th><th class="r">Balance</th><th class="r">Locked</th><th class="r">Spent · MTD</th></tr></thead>
          <tbody>
            <tr v-if="loading"><td colspan="4" class="pad muted">Loading…</td></tr>
            <tr v-for="r in rows" v-else :key="r.credit_type">
              <td class="strong">{{ ctLabel(r.credit_type) }}</td>
              <td class="r mono" :title="full(r.balance)">{{ compact(r.balance) }}</td>
              <td class="r mono muted">{{ Number(r.locked) > 0 ? compact(r.locked) : '—' }}</td>
              <td class="r mono" :title="full(String(r.used))">{{ r.used > 0 ? compact(String(r.used)) : '—' }}</td>
            </tr>
            <tr v-if="!loading && !rows.length"><td colspan="4" class="pad muted">No credits yet — buy some to get started.</td></tr>
          </tbody>
        </table>
      </section>

      <!-- Buy + payment method -->
      <div class="col">
        <section class="panel">
          <header class="panel-h"><span class="panel-title">Add credits</span></header>
          <div class="buy">
            <p class="buy-copy">Top up through Stripe Checkout. Choose the credit type and amount on the next screen.</p>
            <NuxtLink to="/wallet/buy" class="btn primary block">Buy credits →</NuxtLink>
          </div>
        </section>

        <section class="panel">
          <header class="panel-h"><span class="panel-title">Payment method</span></header>
          <div class="pay">
            <div class="pay-row">
              <span class="pay-k">Processor</span>
              <span class="pay-v">Stripe <span class="muted">· Checkout (hosted)</span></span>
            </div>
            <div class="pay-row">
              <span class="pay-k">Mode</span>
              <span class="pay-v">
                <template v-if="user?.is_paper">Sandbox — settles instantly, no charge</template>
                <template v-else>Live — real Stripe charge</template>
              </span>
            </div>
            <div class="pay-row">
              <span class="pay-k">Card on file</span>
              <span class="pay-v muted">None — Exascale never stores card data; Stripe holds it.</span>
            </div>
          </div>
        </section>
      </div>
    </div>

    <!-- Purchase history (real) -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Purchase history</span><span class="panel-meta mono">{{ purchases.length }}</span></header>
      <table class="tbl">
        <thead><tr><th>Date</th><th>Credit</th><th class="r">Amount</th><th>Currency</th><th>Status</th></tr></thead>
        <tbody>
          <tr v-if="loading"><td colspan="5" class="pad muted">Loading…</td></tr>
          <tr v-for="p in purchases" v-else :key="p.id">
            <td class="mono muted">{{ dt(p.created_at) }}</td>
            <td>{{ ctLabel(p.credit_type) }}</td>
            <td class="r mono" :title="full(p.amount)">{{ compact(p.amount) }}</td>
            <td class="mono upper">{{ p.currency }}</td>
            <td><span class="st" :class="statusClass(p.status)"><span class="dot" />{{ p.status }}</span></td>
          </tr>
          <tr v-if="!loading && !purchases.length"><td colspan="5" class="pad muted">No purchases yet. <NuxtLink to="/wallet/buy" class="link">Buy credits →</NuxtLink></td></tr>
        </tbody>
      </table>
    </section>

    <footer class="foot muted">
      Spend and budget are tracked per credit type — there is no single dollar figure because the price
      index (the exchange) is paused. Need invoices, sub-account budgets, or wire/ACH? Those arrive with
      enterprise billing (M4).
    </footer>
  </div>
</template>

<style scoped>
.billing { background: var(--canvas); color: var(--text); font-family: var(--font-sans); padding: var(--sp-5); display: flex; flex-direction: column; gap: var(--sp-4); max-width: 1100px; margin: 0 auto; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.muted { color: var(--text-3); }
.strong { font-weight: 600; }
.upper { text-transform: uppercase; }
.r { text-align: right; }
.link { color: var(--brand); text-decoration: none; }

.head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 6px; line-height: 1; }
.sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0; max-width: 580px; }
.chip { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.1em; padding: 3px var(--sp-2); border-radius: var(--radius-sm); }
.chip.live { background: var(--pos-soft); color: var(--pos); }
.chip.paper { color: var(--info); border: 1px solid var(--border); }

.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden; }
.panel-h { display: flex; justify-content: space-between; align-items: center; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.panel-title { font-size: var(--fs-sm); font-weight: 600; }
.panel-meta { font-size: var(--fs-xs); color: var(--text-3); }

.grid { display: grid; grid-template-columns: 1.6fr 1fr; gap: var(--sp-4); align-items: start; }
.col { display: flex; flex-direction: column; gap: var(--sp-4); }

/* budget */
.budget { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-5); padding: var(--sp-4); align-items: center; }
.lbl { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); display: block; margin-bottom: 6px; }
.set-row { display: flex; gap: var(--sp-2); }
.input { flex: 1; background: var(--canvas); border: 1px solid var(--border-strong); border-radius: var(--radius-sm); color: var(--text); padding: 0 var(--sp-3); height: 34px; font-size: var(--fs-sm); text-align: right; outline: none; }
.input:focus { border-color: var(--brand); }
.btn { height: 34px; padding: 0 var(--sp-4); border-radius: var(--radius-sm); border: 1px solid var(--border-strong); background: var(--overlay); color: var(--text); font-size: var(--fs-sm); font-weight: 500; cursor: pointer; white-space: nowrap; }
.btn:hover { border-color: var(--brand); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn.primary { background: var(--brand); color: #0a0a0a; border-color: var(--brand); font-weight: 600; }
.btn.block { display: flex; align-items: center; justify-content: center; width: 100%; text-decoration: none; }
.hint, .msg { font-size: var(--fs-xs); margin: var(--sp-2) 0 0; }
.hint { color: var(--text-3); }
.msg.err { color: var(--neg); }
.msg.ok { color: var(--pos); }

.budget-meter { display: flex; flex-direction: column; gap: var(--sp-2); }
.meter-top { display: flex; align-items: baseline; gap: var(--sp-2); }
.meter-used { font-size: var(--fs-2xl); font-weight: 600; letter-spacing: -0.02em; }
.meter-of { font-size: var(--fs-xs); color: var(--text-3); }
.bar { height: 8px; background: var(--canvas); border: 1px solid var(--border); border-radius: 999px; overflow: hidden; }
.fill { height: 100%; border-radius: 999px; transition: width 300ms; }
.meter-foot { display: flex; justify-content: space-between; font-size: var(--fs-xs); }
.meter-empty { font-size: var(--fs-sm); }
.lvl-ok { color: var(--pos); } .fill.lvl-ok { background: var(--pos); }
.lvl-mid { color: var(--info); } .fill.lvl-mid { background: var(--info); }
.lvl-warn { color: var(--warn); } .fill.lvl-warn { background: var(--warn); }
.lvl-over { color: var(--neg); } .fill.lvl-over { background: var(--neg); }

/* tables */
.tbl { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.tbl th { text-align: left; font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); font-weight: 500; padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); }
.tbl th.r { text-align: right; }
.tbl td { padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); vertical-align: middle; }
.tbl tbody tr:last-child td { border-bottom: 0; }
.pad { padding: var(--sp-5) var(--sp-4); text-align: center; }

.st { display: inline-flex; align-items: center; gap: 6px; font-size: var(--fs-xs); text-transform: capitalize; }
.st .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.st.ok { color: var(--pos); }
.st.pending { color: var(--warn); }
.st.other { color: var(--text-3); }

/* buy + pay */
.buy { padding: var(--sp-4); display: flex; flex-direction: column; gap: var(--sp-3); }
.buy-copy { font-size: var(--fs-xs); color: var(--text-2); margin: 0; line-height: 1.5; }
.pay { padding: var(--sp-2) var(--sp-4) var(--sp-3); }
.pay-row { display: flex; justify-content: space-between; gap: var(--sp-3); padding: var(--sp-2) 0; border-bottom: 1px solid var(--border); font-size: var(--fs-sm); }
.pay-row:last-child { border-bottom: 0; }
.pay-k { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); padding-top: 2px; }
.pay-v { text-align: right; }

.foot { font-size: var(--fs-xs); line-height: 1.6; max-width: 760px; }

@media (max-width: 860px) {
  .grid { grid-template-columns: 1fr; }
  .budget { grid-template-columns: 1fr; gap: var(--sp-4); }
}
</style>
