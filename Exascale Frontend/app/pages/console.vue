<script setup lang="ts">
/**
 * /console — the AI-company command center (F20). A dense, dark, institutional overview wired
 * entirely to the real platform via the BFF composables: balances, the live ledger tape, GPU
 * instances (F13), an inference quick-run, budget, API keys, purchases, and the audit trail.
 * Always live — no mock data. Numbers-first: balances/usage are the largest elements, mono + tabular,
 * semantic colour paired with ▲/▼. Tokens only.
 */
import { compact, full } from '~/utils/format'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Console — 1Trade' })

const { user } = useAuth()
const { models, load: loadCatalog } = useCatalog()
const { balances, transactions, loadBalances, loadTransactions } = useWallet()
const { running, error: infError, insufficientCredit, result, run } = useInference()
const { keys, newSecret, load: loadKeys, create: createKey, revoke: revokeKey, dismissSecret } = useKeys()
const { instances, types: gpuTypes, loadInstances, loadTypes } = useCompute()

// ── Inference quick-run ─────────────────────────────────────────────────────────────────────
const selectedModel = ref('llama-3.1-8b')
const prompt = ref('Write a one-line definition of a GPU.')
const textModels = computed(() => models.value.filter((m) => m.exascale.modality === 'text'))
const activeModel = computed(() => models.value.find((m) => m.id === selectedModel.value))
async function onRun() {
  if (!prompt.value.trim() || running.value) return
  try { await run(selectedModel.value, prompt.value) } catch { /* surfaced via infError */ }
  await Promise.allSettled([loadBalances(), loadTransactions()])
  synced.value = nowLabel()
}
const estCost = computed(() => {
  if (!result.value || !activeModel.value) return ''
  return ((Number(activeModel.value.exascale.price) * result.value.usage.total_tokens) / 1000).toFixed(6)
})

// ── API keys ────────────────────────────────────────────────────────────────────────────────
const newKeyName = ref('')
async function onCreateKey() {
  if (!newKeyName.value.trim()) return
  await createKey(newKeyName.value, ['inference:read'])
  newKeyName.value = ''
}
function copySecret() { if (newSecret.value) navigator.clipboard?.writeText(newSecret.value) }

// ── Budget ──────────────────────────────────────────────────────────────────────────────────
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
const budgetLevel = computed(() => (budgetPct.value >= 100 ? 'over' : budgetPct.value >= 80 ? 'warn' : budgetPct.value >= 50 ? 'mid' : 'ok'))

// ── Formatting ──────────────────────────────────────────────────────────────────────────────
// Credit amounts render compactly (250K · 1.2M); hover a hero/balance cell for the exact value (full).
function fmt(s: string): string { return compact(s) }
function fmtInt(n: number): string { return Math.round(n).toLocaleString('en-US') }
function money(n: number): string { return n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }
function isDebit(amount: string): boolean { return amount.trim().startsWith('-') }
function clock(t: string): string { return new Date(t).toLocaleTimeString('en-US', { hour12: false }) }
function nowLabel(): string { return new Date().toLocaleTimeString('en-US', { hour12: false }) }

// ── KPI deltas ──────────────────────────────────────────────────────────────────────────────
const todayKey = new Date().toDateString()
function todayNet(ct: string): number {
  return transactions.value
    .filter((t) => t.credit_type === ct && new Date(t.created_at).toDateString() === todayKey)
    .reduce((s, t) => s + Number(t.amount), 0)
}
const txTodayCount = computed(() => transactions.value.filter((t) => new Date(t.created_at).toDateString() === todayKey).length)

// ── Compute (live GPU instances, F13) ───────────────────────────────────────────────────────
const liveInstances = computed(() =>
  instances.value.filter((i) => ['running', 'provisioning', 'starting', 'stopping'].includes(i.state)),
)
const now = ref(Date.now())
let ticker: ReturnType<typeof setInterval> | null = null
// Hourly GPU burn = Σ running instances (count × tier $/hr) — a real, numbers-first spend rate.
const gpuBurn = computed(() => {
  const priceOf = (t: string) => Number(gpuTypes.value.find((g) => g.id === t)?.price_per_hour ?? 0)
  return instances.value
    .filter((i) => i.state === 'running')
    .reduce((s, i) => s + priceOf(i.gpu_type) * (i.count || 1), 0)
})
function uptime(i: { started_at: string | null }): string {
  if (!i.started_at) return '—'
  let s = Math.max(0, Math.floor((now.value - new Date(i.started_at).getTime()) / 1000))
  const h = Math.floor(s / 3600); s -= h * 3600
  const m = Math.floor(s / 60); s -= m * 60
  return h > 0 ? `${h}h ${m}m` : m > 0 ? `${m}m ${s}s` : `${s}s`
}
function stateClass(s: string): string {
  if (s === 'running') return 'st-run'
  if (s === 'stopped' || s === 'terminated') return 'st-off'
  return 'st-warn'
}

// ── Derived KPIs + sync ─────────────────────────────────────────────────────────────────────
const modelCount = computed(() => models.value.length)
const keyCount = computed(() => keys.value.length)
const textBalance = computed(() => balances.value.find((b) => b.credit_type === 'text')?.balance ?? '0')
const synced = ref('')

async function refreshAll() {
  await Promise.allSettled([loadBalances(), loadTransactions(), loadInstances()])
  synced.value = nowLabel()
}
onMounted(async () => {
  ticker = setInterval(() => { now.value = Date.now() }, 1000)
  await Promise.allSettled([
    loadCatalog(), loadBalances(), loadTransactions(), loadKeys(),
    loadBudget(), loadActivity(), loadTypes(), loadInstances(),
  ])
  synced.value = nowLabel()
})
onBeforeUnmount(() => { if (ticker) clearInterval(ticker) })
</script>

<template>
  <div class="console">
    <!-- ── Header ─────────────────────────────────────────────────────────────── -->
    <header class="dash-head">
      <div class="dh-titles">
        <div class="eyebrow">Overview · {{ user?.is_paper ? 'sandbox' : 'live' }}</div>
        <h1 class="dash-title">Console</h1>
      </div>
      <div class="dh-actions">
        <button class="sync" title="Refresh" @click="refreshAll">
          <span class="sync-dot" /> Synced <span class="mono">{{ synced || '—' }}</span>
        </button>
        <span v-if="user?.is_paper" class="chip chip-paper">sandbox</span>
        <span v-else class="chip chip-live">live</span>
        <NuxtLink to="/compute/new" class="btn ghost">New GPU instance</NuxtLink>
        <NuxtLink to="/wallet/buy" class="btn brand">Buy credits</NuxtLink>
      </div>
    </header>

    <!-- ── Install the CLI (one-line, copy-able) ──────────────────────────────── -->
    <AppCliInstall />

    <!-- ── KPI command strip ──────────────────────────────────────────────────── -->
    <section class="kpis">
      <div v-for="b in balances" :key="b.credit_type" class="kpi">
        <div class="kpi-cap">{{ b.credit_type }}</div>
        <div class="kpi-num mono" :title="full(b.balance) + ' ' + b.credit_type">{{ fmt(b.balance) }}</div>
        <div class="kpi-delta mono" :class="todayNet(b.credit_type) < 0 ? 'neg' : todayNet(b.credit_type) > 0 ? 'pos' : 'flat'">
          <template v-if="todayNet(b.credit_type) !== 0">{{ todayNet(b.credit_type) < 0 ? '▼' : '▲' }} {{ fmt(String(Math.abs(todayNet(b.credit_type)))) }}</template>
          <template v-else>— no change today</template>
        </div>
      </div>
      <div class="kpi stat">
        <div class="kpi-cap">GPU burn</div>
        <div class="kpi-num mono">${{ money(gpuBurn) }}<span class="per">/hr</span></div>
        <div class="kpi-delta flat">{{ liveInstances.length }} instance{{ liveInstances.length === 1 ? '' : 's' }} live</div>
      </div>
      <div class="kpi stat">
        <div class="kpi-cap">Catalog</div>
        <div class="kpi-num mono">{{ modelCount }}</div>
        <div class="kpi-delta flat">models available</div>
      </div>
      <div class="kpi stat">
        <div class="kpi-cap">API keys</div>
        <div class="kpi-num mono">{{ keyCount }}</div>
        <div class="kpi-delta flat">{{ txTodayCount }} ledger moves today</div>
      </div>
      <NuxtLink v-if="!balances.length" to="/wallet/buy" class="kpi empty">No credits — buy some →</NuxtLink>
    </section>

    <!-- ── Main grid ──────────────────────────────────────────────────────────── -->
    <div class="grid">
      <div class="col">
        <!-- Wallet / balances -->
        <section class="panel">
          <header class="panel-h">
            <span class="panel-title">Wallet · balances</span>
            <NuxtLink class="panel-act" to="/wallet">Open →</NuxtLink>
          </header>
          <table class="tbl">
            <thead><tr><th>Credit</th><th class="r">Balance</th><th class="r">Today</th><th class="r">Locked</th></tr></thead>
            <tbody>
              <tr v-for="b in balances" :key="b.credit_type">
                <td class="ct">{{ b.credit_type }}</td>
                <td class="r mono strong" :title="full(b.balance)">{{ fmt(b.balance) }}</td>
                <td class="r mono" :class="todayNet(b.credit_type) < 0 ? 'neg' : todayNet(b.credit_type) > 0 ? 'pos' : 'muted'">
                  <template v-if="todayNet(b.credit_type) !== 0">{{ todayNet(b.credit_type) < 0 ? '▼' : '▲' }} {{ fmt(String(Math.abs(todayNet(b.credit_type)))) }}</template>
                  <template v-else>—</template>
                </td>
                <td class="r mono muted">{{ fmt(b.locked_amount) }}</td>
              </tr>
              <tr v-if="!balances.length"><td colspan="4" class="muted pad">No credits yet — <NuxtLink to="/wallet/buy">buy some →</NuxtLink></td></tr>
            </tbody>
          </table>
          <footer class="panel-foot">
            <NuxtLink to="/wallet/buy" class="link">Buy credits</NuxtLink>
            <NuxtLink to="/wallet" class="link">Convert / movements</NuxtLink>
          </footer>
        </section>

        <!-- GPU compute (F13) -->
        <section class="panel">
          <header class="panel-h">
            <span class="panel-title">GPU compute</span>
            <NuxtLink class="panel-act" to="/compute">Manage →</NuxtLink>
          </header>
          <table class="tbl">
            <thead><tr><th>Instance</th><th>Type</th><th>State</th><th class="r">GPUs</th><th class="r">Uptime</th></tr></thead>
            <tbody>
              <tr v-for="i in liveInstances" :key="i.id">
                <td class="mono muted">{{ i.id.slice(0, 12) }}</td>
                <td class="mono">{{ i.gpu_type }}</td>
                <td><span class="state" :class="stateClass(i.state)"><span class="sd" />{{ i.state }}</span></td>
                <td class="r mono">{{ i.count }}</td>
                <td class="r mono">{{ uptime(i) }}</td>
              </tr>
              <tr v-if="!liveInstances.length"><td colspan="5" class="muted pad">No running instances — <NuxtLink to="/compute/new">launch one →</NuxtLink></td></tr>
            </tbody>
          </table>
        </section>

        <!-- Inference quick-run -->
        <section class="panel">
          <header class="panel-h">
            <span class="panel-title">Inference</span>
            <select v-model="selectedModel" class="mini-select mono">
              <option v-for="m in textModels" :key="m.id" :value="m.id">{{ m.id }} · {{ fmt(m.exascale.price) }}/{{ m.exascale.unit }}</option>
            </select>
          </header>
          <div class="panel-b">
            <textarea v-model="prompt" class="prompt" rows="2" placeholder="Ask the model…" />
            <div class="run-row">
              <button class="btn brand sm" :disabled="running" @click="onRun">{{ running ? 'Running…' : 'Run' }}</button>
              <span v-if="result" class="usage mono">{{ result.usage.prompt_tokens }} in · {{ result.usage.completion_tokens }} out · ≈ {{ estCost }} cr</span>
            </div>
            <div v-if="insufficientCredit" class="note warn">Insufficient credit. <NuxtLink to="/wallet/buy">Buy text →</NuxtLink></div>
            <div v-else-if="infError" class="note neg">{{ infError }}</div>
            <pre v-if="result" class="output">{{ result.content }}</pre>
          </div>
        </section>
      </div>

      <div class="col">
        <!-- Activity tape -->
        <section class="panel tall">
          <header class="panel-h">
            <span class="panel-title">Activity</span>
            <span class="panel-meta mono"><span class="pulse" />{{ txTodayCount }} today</span>
          </header>
          <div class="tape">
            <table class="tbl">
              <tbody>
                <tr v-for="t in transactions" :key="t.tx_id">
                  <td class="mono muted tm">{{ clock(t.created_at) }}</td>
                  <td class="op">{{ t.operation }}</td>
                  <td class="ct">{{ t.credit_type }}</td>
                  <td class="r mono" :class="isDebit(t.amount) ? 'neg' : 'pos'">{{ isDebit(t.amount) ? '▼' : '▲' }} {{ fmt(t.amount.replace(/^[+-]/, '')) }}</td>
                </tr>
                <tr v-if="!transactions.length"><td colspan="4" class="muted pad">No activity yet.</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- Budget -->
        <section class="panel">
          <header class="panel-h"><span class="panel-title">Monthly budget</span><span class="panel-meta mono">text</span></header>
          <div class="panel-b">
            <div class="budget-row-in">
              <input v-model="budgetInput" class="input mono amt" inputmode="numeric" placeholder="limit" />
              <button class="btn ghost sm" @click="saveBudget">Save</button>
            </div>
            <div v-if="budget" class="meter">
              <div class="bar"><div class="fill" :class="'lvl-' + budgetLevel" :style="{ width: budgetPct + '%' }" /></div>
              <div class="meter-row mono">
                <span :class="'lvl-' + budgetLevel">{{ fmt(String(usedThisMonth)) }} used</span>
                <span class="muted">/ {{ fmt(budget.monthly_limit) }} · {{ budgetPct.toFixed(0) }}%</span>
              </div>
            </div>
            <p v-else class="hint">No budget set. Alerts at 50 / 80 / 100% of usage.</p>
          </div>
        </section>

        <!-- API keys -->
        <section class="panel">
          <header class="panel-h"><span class="panel-title">API keys</span><span class="panel-meta mono">{{ keyCount }}</span></header>
          <div class="panel-b">
            <div v-if="newSecret" class="secret">
              <div class="secret-h">Copy now — shown once.</div>
              <code class="mono">{{ newSecret }}</code>
              <div class="run-row"><button class="btn ghost sm" @click="copySecret">Copy</button><button class="btn ghost sm" @click="dismissSecret">Done</button></div>
            </div>
            <div class="key-new">
              <input v-model="newKeyName" class="input" placeholder="Key name (e.g. production)" />
              <button class="btn brand sm" @click="onCreateKey">Generate</button>
            </div>
            <table class="tbl">
              <tbody>
                <tr v-for="k in keys" :key="k.id">
                  <td class="mono muted">{{ k.prefix }}</td>
                  <td>{{ k.name }}</td>
                  <td class="r"><button class="link neg" @click="revokeKey(k.id)">Revoke</button></td>
                </tr>
                <tr v-if="!keys.length"><td colspan="3" class="muted pad">No keys yet.</td></tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </div>

    <!-- ── Bottom: purchases + audit ──────────────────────────────────────────── -->
    <div class="grid-2">
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Purchases</span></header>
        <table class="tbl">
          <thead><tr><th>Date</th><th class="r">Amount</th><th>Credit</th><th>Status</th></tr></thead>
          <tbody>
            <tr v-for="p in purchases" :key="p.id">
              <td class="mono muted">{{ new Date(p.created_at).toLocaleDateString() }}</td>
              <td class="r mono">{{ fmt(p.amount) }}</td>
              <td class="ct">{{ p.credit_type }}</td>
              <td><span :class="p.status === 'paid' ? 'pos' : 'muted'">{{ p.status }}</span></td>
            </tr>
            <tr v-if="!purchases.length"><td colspan="4" class="muted pad">No purchases yet.</td></tr>
          </tbody>
        </table>
      </section>
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Audit log</span></header>
        <table class="tbl">
          <thead><tr><th>Time</th><th>Action</th><th>Target</th></tr></thead>
          <tbody>
            <tr v-for="a in auditEntries" :key="a.id">
              <td class="mono muted">{{ clock(a.created_at) }}</td>
              <td class="mono">{{ a.action }}</td>
              <td class="muted">{{ a.target_type }}</td>
            </tr>
            <tr v-if="!auditEntries.length"><td colspan="3" class="muted pad">No activity yet.</td></tr>
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
  max-width: 1480px;
  margin: 0 auto;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.pos { color: var(--pos); }
.neg { color: var(--neg); }
.muted { color: var(--text-3); }
.strong { color: var(--text); font-weight: 600; }

/* ── Header ── */
.dash-head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.dash-title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 0; line-height: 1; }
.dh-actions { display: flex; align-items: center; gap: var(--sp-3); }
.sync { display: inline-flex; align-items: center; gap: 7px; background: transparent; border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 5px 10px; font-size: var(--fs-xs); color: var(--text-2); cursor: pointer; }
.sync:hover { border-color: var(--border-strong); color: var(--text); }
.sync-dot, .pulse { width: 6px; height: 6px; border-radius: var(--radius-full); background: var(--pos); display: inline-block; animation: pulse 2.4s ease-in-out infinite; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }
@media (prefers-reduced-motion: reduce) { .sync-dot, .pulse { animation: none; } }
.chip { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; padding: 3px var(--sp-2); border-radius: var(--radius-sm); }
.chip-live { background: var(--pos-soft); color: var(--pos); }
.chip-paper { color: var(--info); border: 1px solid var(--border); }

/* ── Buttons ── */
.btn { display: inline-flex; align-items: center; justify-content: center; height: 32px; padding: 0 var(--sp-4); border-radius: var(--radius-sm); font-size: var(--fs-sm); font-weight: 600; cursor: pointer; border: 1px solid transparent; text-decoration: none; transition: background 140ms, border-color 140ms; }
.btn.sm { height: 28px; padding: 0 var(--sp-3); }
.btn.brand { background: var(--brand); color: var(--text-on-accent); }
.btn.brand:hover { background: var(--brand-hov); }
.btn.brand:disabled { opacity: 0.5; cursor: default; }
.btn.ghost { background: transparent; color: var(--text); border-color: var(--border-strong); }
.btn.ghost:hover { background: var(--overlay); }

/* ── KPI strip ── */
.kpis { display: grid; grid-template-columns: repeat(auto-fill, minmax(184px, 1fr)); gap: var(--sp-3); }
.kpi { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); padding: var(--sp-3) var(--sp-4); transition: border-color 140ms; }
.kpi:hover { border-color: var(--border-strong); }
.kpi-cap { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.14em; color: var(--text-3); }
.kpi-num { font-size: var(--fs-2xl); font-weight: 600; letter-spacing: -0.01em; line-height: 1.1; margin-top: var(--sp-1); }
.kpi-num .per { font-size: var(--fs-sm); color: var(--text-3); font-weight: 400; margin-left: 2px; }
.kpi-delta { font-size: var(--fs-xs); margin-top: var(--sp-2); }
.kpi-delta.pos { color: var(--pos); } .kpi-delta.neg { color: var(--neg); } .kpi-delta.flat { color: var(--text-3); }
.kpi.stat .kpi-num { color: var(--text); }
.kpi.empty { display: flex; align-items: center; justify-content: center; color: var(--text-3); text-decoration: none; border-style: dashed; }
.kpi.empty:hover { color: var(--text); border-color: var(--brand); }

/* ── Grid + panels ── */
.grid { display: grid; grid-template-columns: 1.55fr 1fr; gap: var(--sp-4); align-items: start; }
.col { display: flex; flex-direction: column; gap: var(--sp-4); }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-4); }
.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); display: flex; flex-direction: column; overflow: hidden; }
.panel-h { display: flex; justify-content: space-between; align-items: center; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.panel-title { font-size: var(--fs-sm); font-weight: 600; letter-spacing: -0.005em; }
.panel-act { font-size: var(--fs-xs); color: var(--text-2); text-decoration: none; font-family: var(--font-mono); }
.panel-act:hover { color: var(--text); }
.panel-meta { font-size: var(--fs-xs); color: var(--text-3); display: inline-flex; align-items: center; gap: 6px; }
.panel-b { padding: var(--sp-4); }
.panel-foot { display: flex; gap: var(--sp-4); padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border); }
.link { font-size: var(--fs-xs); color: var(--text-2); text-decoration: none; font-family: var(--font-mono); cursor: pointer; background: none; border: 0; padding: 0; }
.link:hover { color: var(--text); }
.link.neg { color: var(--neg); }

/* ── Tables (dense) ── */
.tbl { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.tbl th { text-align: left; font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); font-weight: 500; padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); }
.tbl td { padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); white-space: nowrap; }
.tbl tbody tr:last-child td { border-bottom: 0; }
.tbl tbody tr { transition: background 120ms; }
.tbl tbody tr:hover td { background: var(--overlay); }
.tbl .r { text-align: right; }
.tbl .ct { text-transform: uppercase; font-size: var(--fs-xs); letter-spacing: 0.04em; color: var(--text-2); font-family: var(--font-mono); }
.tbl .op { color: var(--text-2); text-transform: capitalize; }
.tbl .tm { font-size: var(--fs-xs); }
.pad { padding: var(--sp-4) !important; white-space: normal; }
.pad a { color: var(--brand); text-decoration: none; }

/* ── State pill (compute) ── */
.state { display: inline-flex; align-items: center; gap: 6px; font-size: var(--fs-xs); text-transform: capitalize; }
.state .sd { width: 6px; height: 6px; border-radius: var(--radius-full); }
.state.st-run { color: var(--pos); } .state.st-run .sd { background: var(--pos); animation: pulse 2.4s ease-in-out infinite; }
.state.st-warn { color: var(--warn); } .state.st-warn .sd { background: var(--warn); }
.state.st-off { color: var(--text-3); } .state.st-off .sd { background: var(--text-3); }

/* ── Activity tape ── */
.panel.tall .tape { max-height: 420px; overflow-y: auto; }

/* ── Inference ── */
.mini-select, .input, .prompt { background: var(--canvas); color: var(--text); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-2) var(--sp-3); font-family: var(--font-sans); font-size: var(--fs-sm); }
.mini-select { font-family: var(--font-mono); font-size: var(--fs-xs); max-width: 240px; }
.prompt { width: 100%; resize: vertical; }
.prompt:focus, .input:focus, .mini-select:focus { outline: none; border-color: var(--border-focus); }
.run-row { display: flex; align-items: center; gap: var(--sp-3); margin-top: var(--sp-3); }
.usage { font-size: var(--fs-xs); color: var(--text-2); }
.output { background: var(--canvas); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-3); margin-top: var(--sp-3); white-space: pre-wrap; font-family: var(--font-mono); font-size: var(--fs-sm); max-height: 180px; overflow-y: auto; }
.note { margin-top: var(--sp-3); font-size: var(--fs-sm); }
.note.warn { color: var(--warn); } .note.neg { color: var(--neg); }
.note a { color: var(--brand); text-decoration: none; }

/* ── Budget + keys ── */
.budget-row-in, .key-new { display: flex; gap: var(--sp-2); }
.budget-row-in .amt { flex: 1; text-align: right; }
.key-new { margin-bottom: var(--sp-3); } .key-new .input { flex: 1; }
.hint { font-size: var(--fs-xs); color: var(--text-3); margin: var(--sp-2) 0 0; }
.meter { margin-top: var(--sp-3); }
.bar { height: 6px; background: var(--overlay); border-radius: var(--radius-full); overflow: hidden; }
.fill { height: 100%; transition: width var(--dur, 200ms) ease; }
.fill.lvl-ok { background: var(--pos); } .fill.lvl-mid { background: var(--brand); } .fill.lvl-warn { background: var(--warn); } .fill.lvl-over { background: var(--neg); }
.meter-row { display: flex; justify-content: space-between; font-size: var(--fs-xs); margin-top: var(--sp-2); }
.lvl-ok { color: var(--pos); } .lvl-mid { color: var(--text); } .lvl-warn { color: var(--warn); } .lvl-over { color: var(--neg); }
.secret { background: var(--pos-soft); border: 1px solid var(--pos); border-radius: var(--radius-sm); padding: var(--sp-3); margin-bottom: var(--sp-3); }
.secret-h { font-size: var(--fs-xs); color: var(--text-2); margin-bottom: var(--sp-2); }
.secret code { display: block; word-break: break-all; font-size: var(--fs-sm); margin-bottom: var(--sp-2); }

@media (max-width: 1180px) {
  .grid { grid-template-columns: 1fr; }
  .grid-2 { grid-template-columns: 1fr; }
}
</style>
