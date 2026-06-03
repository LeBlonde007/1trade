<script setup lang="ts">
/**
 * /console — the live platform console (F20). One dense, dark, institutional surface wired to the
 * real platform via the BFF composables: identity, model catalog, inference playground, wallet
 * (balances + transactions), buy-credits, and API keys. Always live (no mock mode). Tokens only —
 * no raw values here.
 */
definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Console — Exascale' })

const { user } = useAuth()
const { models, load: loadCatalog } = useCatalog()
const { balances, transactions, loadBalances, loadTransactions } = useWallet()
const { running, error: infError, insufficientCredit, result, run } = useInference()
const { checkout, loading: buying } = useBilling()
const { keys, newSecret, load: loadKeys, create: createKey, revoke: revokeKey, dismissSecret } = useKeys()

// ── Inference playground state ──────────────────────────────
const selectedModel = ref('llama-3.1-8b')
const prompt = ref('Write a one-line definition of a GPU.')
const textModels = computed(() => models.value.filter((m) => m.exascale.modality === 'text'))
const activeModel = computed(() => models.value.find((m) => m.id === selectedModel.value))

async function onRun() {
  if (!prompt.value.trim() || running.value) return
  try {
    await run(selectedModel.value, prompt.value)
  } catch { /* surfaced via infError / insufficientCredit */ }
  await loadBalances()
  await loadTransactions()
}

// Estimated credits for the last run (price × total_tokens / 1000), for display only.
const estCost = computed(() => {
  if (!result.value || !activeModel.value) return ''
  const price = Number(activeModel.value.exascale.price)
  return ((price * result.value.usage.total_tokens) / 1000).toFixed(6)
})

// ── Buy credits state ───────────────────────────────────────
const buyAmount = ref('100')
const buyType = ref('text')
async function onBuy() {
  const { checkout_url } = await checkout(buyAmount.value + '.000000', buyType.value, 'usd')
  window.open(checkout_url, '_blank')
}

// ── API keys state ──────────────────────────────────────────
const newKeyName = ref('')
async function onCreateKey() {
  if (!newKeyName.value.trim()) return
  await createKey(newKeyName.value, ['inference:read'])
  newKeyName.value = ''
}
function copySecret() {
  if (newSecret.value) navigator.clipboard?.writeText(newSecret.value)
}

// ── Formatting (mono + tabular, per the design system) ──────
function fmt(s: string, dp = 2): string {
  const n = Number(s)
  if (Number.isNaN(n)) return s
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: 6 })
}
function isDebit(amount: string): boolean {
  return amount.trim().startsWith('-')
}
const textBalance = computed(() => balances.value.find((b) => b.credit_type === 'text')?.balance ?? '0')

// ── KPI deltas — numbers-first hero tiles show today's net movement per credit type ─────────
// Net signed change today for a credit type (mint/purchase +, consumption −), from the live tx feed.
const todayKey = new Date().toDateString()
function todayNet(ct: string): number {
  return transactions.value
    .filter((t) => t.credit_type === ct && new Date(t.created_at).toDateString() === todayKey)
    .reduce((s, t) => s + Number(t.amount), 0)
}
// Activity pulse: count of ledger movements today (a small "the market is live" signal in the header).
const txTodayCount = computed(() =>
  transactions.value.filter((t) => new Date(t.created_at).toDateString() === todayKey).length,
)

// ── Budget + activity panels ────────────────────────────────
interface Budget { credit_type: string; monthly_limit: string }
interface Purchase { id: string; amount: string; credit_type: string; currency: string; status: string; created_at: string }
interface AuditEntry { id: string; action: string; target_type?: string; created_at: string }
const budget = useState<Budget | null>('console:budget', () => null)
const budgetInput = ref('')
const purchases = useState<Purchase[]>('console:purchases', () => [])
const auditEntries = useState<AuditEntry[]>('console:audit', () => [])

async function loadBudget() {
  budget.value = (await $fetch<{ budget: Budget | null }>('/api/billing/budget')).budget
  if (budget.value) budgetInput.value = String(Number(budget.value.monthly_limit))
}
async function saveBudget() {
  if (!budgetInput.value) return
  await $fetch('/api/billing/budget', { method: 'PUT', body: { credit_type: 'text', monthly_limit: Number(budgetInput.value).toFixed(6) } })
  await loadBudget()
}
async function loadActivity() {
  purchases.value = (await $fetch<{ purchases: Purchase[] }>('/api/billing/purchases')).purchases || []
  auditEntries.value = (await $fetch<{ entries: AuditEntry[] }>('/api/account/audit')).entries || []
}

// Month-to-date consumption for the budget's credit type, from the transaction feed.
const usedThisMonth = computed(() => {
  const ct = budget.value?.credit_type ?? 'text'
  const m = new Date().getMonth()
  return transactions.value
    .filter((t) => t.credit_type === ct && t.operation === 'consumption' && new Date(t.created_at).getMonth() === m)
    .reduce((s, t) => s + Math.abs(Number(t.amount)), 0)
})
const budgetPct = computed(() => {
  const lim = Number(budget.value?.monthly_limit ?? 0)
  return lim > 0 ? Math.min(100, (usedThisMonth.value / lim) * 100) : 0
})
// 50/80/100% alert bands → semantic color.
const budgetLevel = computed(() => (budgetPct.value >= 100 ? 'over' : budgetPct.value >= 80 ? 'warn' : budgetPct.value >= 50 ? 'mid' : 'ok'))

onMounted(async () => {
  await Promise.allSettled([loadCatalog(), loadBalances(), loadTransactions(), loadKeys(), loadBudget(), loadActivity()])
})
</script>

<template>
  <div class="console">
    <!-- Page header (global nav + identity live in AppTopbar / AppSidebar) -->
    <header class="page-head">
      <div class="ph-titles">
        <div class="eyebrow">Overview</div>
        <h1 class="page-title">Console</h1>
      </div>
      <div class="ph-meta">
        <span class="bal mono">{{ fmt(textBalance) }} <span class="unit">text</span></span>
        <span v-if="txTodayCount" class="pulse mono"><span class="dot" />{{ txTodayCount }} today</span>
        <span class="mode-live">live</span>
        <span v-if="user?.is_paper" class="paper">sandbox</span>
      </div>
    </header>

    <!-- KPI tiles — numbers-first: the balance is the largest element, with today's net movement -->
    <section class="kpis">
      <div v-for="b in balances" :key="b.credit_type" class="kpi">
        <div class="kpi-label">{{ b.credit_type }}</div>
        <div class="kpi-num mono">{{ fmt(b.balance) }}</div>
        <div class="kpi-foot mono">
          <span
            v-if="todayNet(b.credit_type) !== 0"
            class="delta"
            :class="todayNet(b.credit_type) < 0 ? 'neg' : 'pos'"
          >{{ todayNet(b.credit_type) < 0 ? '▼' : '▲' }} {{ fmt(String(Math.abs(todayNet(b.credit_type)))) }}</span>
          <span v-else class="delta muted">— flat</span>
          <span v-if="Number(b.locked_amount) > 0" class="locked">{{ fmt(b.locked_amount) }} locked</span>
        </div>
      </div>
      <NuxtLink v-if="!balances.length" to="/wallet/buy" class="kpi empty">
        No credits yet — buy some →
      </NuxtLink>
    </section>

    <div class="grid">
      <!-- Inference playground -->
      <section class="card play">
        <div class="card-h">
          <h2>Inference</h2>
          <select v-model="selectedModel" class="select mono">
            <option v-for="m in textModels" :key="m.id" :value="m.id">
              {{ m.id }} · {{ fmt(m.exascale.price) }}/{{ m.exascale.unit }}
            </option>
          </select>
        </div>
        <textarea v-model="prompt" class="prompt" rows="3" placeholder="Ask the model…"></textarea>
        <div class="row">
          <button class="primary" :disabled="running" @click="onRun">{{ running ? 'Running…' : 'Run' }}</button>
          <span v-if="result" class="usage mono">
            {{ result.usage.prompt_tokens }} in · {{ result.usage.completion_tokens }} out ·
            {{ result.usage.total_tokens }} tok · ≈ {{ estCost }} credits
          </span>
        </div>

        <div v-if="insufficientCredit" class="banner warn">
          Insufficient credit to run this model. <a @click="buyType = 'text'">Buy text credits →</a>
        </div>
        <div v-else-if="infError" class="banner neg">{{ infError }}</div>
        <pre v-if="result" class="output">{{ result.content }}</pre>
      </section>

      <!-- Right rail -->
      <div class="rail">
        <!-- Buy credits -->
        <section class="card">
          <div class="card-h"><h2>Buy credits</h2></div>
          <div class="buy-row">
            <input v-model="buyAmount" class="input mono amt" inputmode="numeric" />
            <select v-model="buyType" class="select mono">
              <option value="text">text</option>
              <option value="speech">speech</option>
              <option value="image">image</option>
            </select>
            <button class="primary" :disabled="buying" @click="onBuy">{{ buying ? '…' : 'Checkout' }}</button>
          </div>
          <p class="hint">Opens Stripe checkout. On settlement the wallet credits automatically.</p>
        </section>

        <!-- Monthly budget -->
        <section class="card">
          <div class="card-h"><h2>Monthly budget</h2></div>
          <div class="buy-row">
            <input v-model="budgetInput" class="input mono amt" inputmode="numeric" placeholder="limit" />
            <span class="unit">text credits / mo</span>
            <button class="primary" @click="saveBudget">Save</button>
          </div>
          <div v-if="budget" class="budget-meter">
            <div class="bar"><div class="fill" :class="'lvl-' + budgetLevel" :style="{ width: budgetPct + '%' }"></div></div>
            <div class="budget-row mono">
              <span :class="'lvl-text-' + budgetLevel">{{ fmt(String(usedThisMonth)) }} used</span>
              <span class="muted">/ {{ fmt(budget.monthly_limit) }} · {{ budgetPct.toFixed(0) }}%</span>
            </div>
          </div>
          <p v-else class="hint">No budget set. Alerts fire at 50 / 80 / 100% of usage.</p>
        </section>

        <!-- API keys -->
        <section class="card">
          <div class="card-h"><h2>API keys</h2></div>
          <div v-if="newSecret" class="banner secret">
            <div class="secret-h">Copy your key now — it won't be shown again.</div>
            <code class="mono">{{ newSecret }}</code>
            <div class="row">
              <button class="ghost" @click="copySecret">Copy</button>
              <button class="ghost" @click="dismissSecret">Done</button>
            </div>
          </div>
          <div class="key-new">
            <input v-model="newKeyName" class="input" placeholder="Key name (e.g. production)" />
            <button class="primary" @click="onCreateKey">Generate</button>
          </div>
          <table class="tbl">
            <tbody>
              <tr v-for="k in keys" :key="k.id">
                <td class="mono prefix">{{ k.prefix }}</td>
                <td class="kname">{{ k.name }}</td>
                <td class="acts"><button class="ghost xs" @click="revokeKey(k.id)">Revoke</button></td>
              </tr>
              <tr v-if="!keys.length"><td colspan="3" class="muted">No keys yet.</td></tr>
            </tbody>
          </table>
        </section>
      </div>
    </div>

    <!-- Wallet / transactions -->
    <section class="card wallet">
      <div class="card-h"><h2>Recent transactions</h2></div>
      <table class="tbl tx">
        <thead>
          <tr><th>Time</th><th>Operation</th><th>Credit</th><th class="r">Amount</th><th class="r">Balance after</th></tr>
        </thead>
        <tbody>
          <tr v-for="t in transactions" :key="t.tx_id">
            <td class="mono muted">{{ new Date(t.created_at).toLocaleTimeString() }}</td>
            <td>{{ t.operation }}</td>
            <td>{{ t.credit_type }}</td>
            <td class="r mono" :class="isDebit(t.amount) ? 'neg' : 'pos'">
              {{ isDebit(t.amount) ? '▼' : '▲' }} {{ fmt(t.amount.replace(/^[+-]/, '')) }}
            </td>
            <td class="r mono">{{ fmt(t.balance_after) }}</td>
          </tr>
          <tr v-if="!transactions.length"><td colspan="5" class="muted">No transactions yet.</td></tr>
        </tbody>
      </table>
    </section>

    <!-- Activity: purchases + audit -->
    <div class="grid">
      <section class="card">
        <div class="card-h"><h2>Purchases</h2></div>
        <table class="tbl">
          <thead><tr><th>Date</th><th class="r">Amount</th><th>Credit</th><th>Status</th></tr></thead>
          <tbody>
            <tr v-for="p in purchases" :key="p.id">
              <td class="mono muted">{{ new Date(p.created_at).toLocaleDateString() }}</td>
              <td class="r mono">{{ fmt(p.amount) }}</td>
              <td>{{ p.credit_type }}</td>
              <td><span :class="p.status === 'paid' ? 'pos' : 'muted'">{{ p.status }}</span></td>
            </tr>
            <tr v-if="!purchases.length"><td colspan="4" class="muted">No purchases yet.</td></tr>
          </tbody>
        </table>
      </section>

      <section class="card">
        <div class="card-h"><h2>Audit log</h2></div>
        <table class="tbl">
          <thead><tr><th>Time</th><th>Action</th><th>Target</th></tr></thead>
          <tbody>
            <tr v-for="a in auditEntries" :key="a.id">
              <td class="mono muted">{{ new Date(a.created_at).toLocaleTimeString() }}</td>
              <td class="mono">{{ a.action }}</td>
              <td class="muted">{{ a.target_type }}</td>
            </tr>
            <tr v-if="!auditEntries.length"><td colspan="3" class="muted">No activity yet.</td></tr>
          </tbody>
        </table>
      </section>
    </div>
  </div>
</template>

<style scoped>
.console {
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  padding: var(--sp-5);
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  max-width: var(--content-max, 1440px);
  margin: 0 auto;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }

/* Page header */
.page-head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--text-3); }
.page-title { font-size: var(--fs-xl, 28px); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 0; color: var(--text); }
.ph-meta { display: flex; align-items: center; gap: var(--sp-3); font-size: var(--fs-sm); }
.ph-meta .bal { font-size: var(--fs-md); }
.ph-meta .unit { color: var(--text-3); font-size: var(--fs-xs); }
.mode-live { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); padding: 2px var(--sp-2); border-radius: var(--radius-sm); background: var(--pos-soft); color: var(--pos); }
.paper { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--info); border: 1px solid var(--border); padding: 1px var(--sp-2); border-radius: var(--radius-sm); }
/* Live activity pulse — subtle "the ledger is moving" signal (principle 5: live, but restrained) */
.pulse { display: inline-flex; align-items: center; gap: 6px; font-size: var(--fs-xs); color: var(--text-3); }
.pulse .dot { width: 6px; height: 6px; border-radius: var(--radius-full); background: var(--pos); animation: pulse 2.4s ease-in-out infinite; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }
@media (prefers-reduced-motion: reduce) { .pulse .dot { animation: none; } }

/* KPI tiles (numbers-first: the balance is the largest element in its tile) */
.kpis { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: var(--sp-3); }
.kpi { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); padding: var(--sp-3) var(--sp-4); transition: border-color var(--dur) var(--ease); }
.kpi:hover { border-color: var(--border-strong); }
.kpi-label { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--text-3); }
.kpi-num { font-size: var(--fs-2xl, 30px); font-weight: 600; letter-spacing: -0.01em; line-height: 1.1; margin-top: var(--sp-1); }
.kpi-foot { display: flex; align-items: baseline; justify-content: space-between; gap: var(--sp-2); font-size: var(--fs-xs); margin-top: var(--sp-2); }
.kpi-foot .delta.pos { color: var(--pos); }
.kpi-foot .delta.neg { color: var(--neg); }
.kpi-foot .delta.muted, .kpi-foot .locked { color: var(--text-3); }
.kpi.empty { color: var(--text-3); display: flex; align-items: center; justify-content: center; text-decoration: none; border-style: dashed; }
.kpi.empty:hover { color: var(--text); border-color: var(--brand); }

/* Grid */
.grid { display: grid; grid-template-columns: 1.6fr 1fr; gap: var(--sp-4); }
.rail { display: flex; flex-direction: column; gap: var(--sp-4); }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); padding: var(--sp-4); }
.card-h { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--sp-3); }
.card-h h2 { font-size: var(--fs-md); font-weight: 600; margin: 0; }

/* Playground */
.prompt, .input, .select { background: var(--canvas); color: var(--text); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-2) var(--sp-3); font-family: var(--font-sans); font-size: var(--fs-sm); }
.prompt { width: 100%; resize: vertical; }
.prompt:focus, .input:focus, .select:focus { outline: none; border-color: var(--border-focus); }
.row { display: flex; align-items: center; gap: var(--sp-3); margin-top: var(--sp-3); }
.usage { font-size: var(--fs-xs); color: var(--text-2); }
.output { background: var(--sunken, var(--canvas)); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-3); margin-top: var(--sp-3); white-space: pre-wrap; font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text); }

/* Buttons */
.primary { background: var(--brand); color: var(--text-on-accent); border: none; border-radius: var(--radius-sm); padding: var(--sp-2) var(--sp-4); font-weight: 600; font-size: var(--fs-sm); cursor: pointer; }
.primary:hover { background: var(--brand-hov); }
.primary:disabled { opacity: 0.5; cursor: default; }
.ghost { background: transparent; color: var(--text-2); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-1) var(--sp-3); font-size: var(--fs-sm); cursor: pointer; }
.ghost:hover { color: var(--text); border-color: var(--border-strong); }
.ghost.xs { font-size: var(--fs-xs); padding: 1px var(--sp-2); }

/* Banners */
.banner { margin-top: var(--sp-3); padding: var(--sp-3); border-radius: var(--radius-sm); font-size: var(--fs-sm); }
.banner.warn { background: var(--overlay); border: 1px solid var(--warn); color: var(--text); }
.banner.warn a { color: var(--brand); cursor: pointer; }
.banner.neg { background: var(--neg-soft); border: 1px solid var(--neg); color: var(--text); }
.banner.secret { background: var(--pos-soft); border: 1px solid var(--pos); }
.banner.secret .secret-h { font-size: var(--fs-xs); color: var(--text-2); margin-bottom: var(--sp-2); }
.banner.secret code { display: block; word-break: break-all; font-size: var(--fs-sm); margin-bottom: var(--sp-2); }

/* Buy + keys */
.buy-row { display: flex; gap: var(--sp-2); }
.buy-row .amt { width: 90px; text-align: right; }
.hint { font-size: var(--fs-xs); color: var(--text-3); margin: var(--sp-2) 0 0; }
.key-new { display: flex; gap: var(--sp-2); margin-bottom: var(--sp-3); }
.key-new .input { flex: 1; }

/* Tables */
.tbl { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.tbl th { text-align: left; font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--text-3); font-weight: 500; padding: var(--sp-2); border-bottom: 1px solid var(--border); }
.tbl td { padding: var(--sp-2); border-bottom: 1px solid var(--border); }
.tbl tbody tr { transition: background var(--dur) var(--ease); }
.tbl tbody tr:hover td { background: var(--overlay); }
.tbl .r { text-align: right; }
.tbl .muted { color: var(--text-3); }
.tbl .prefix { color: var(--text-2); }
.tbl .kname { color: var(--text); }
.tbl .acts { text-align: right; }
.pos { color: var(--pos); }
.neg { color: var(--neg); }

/* Budget meter */
.buy-row .unit { font-size: var(--fs-xs); color: var(--text-3); align-self: center; }
.budget-meter { margin-top: var(--sp-3); }
.bar { height: 6px; background: var(--overlay); border-radius: var(--radius-full); overflow: hidden; }
.fill { height: 100%; transition: width var(--dur) var(--ease); }
.fill.lvl-ok { background: var(--pos); }
.fill.lvl-mid { background: var(--brand); }
.fill.lvl-warn { background: var(--warn); }
.fill.lvl-over { background: var(--neg); }
.budget-row { display: flex; justify-content: space-between; font-size: var(--fs-xs); margin-top: var(--sp-2); }
.lvl-text-ok { color: var(--pos); }
.lvl-text-mid { color: var(--text); }
.lvl-text-warn { color: var(--warn); }
.lvl-text-over { color: var(--neg); }

@media (max-width: 1100px) { .grid { grid-template-columns: 1fr; } }
</style>
