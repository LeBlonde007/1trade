<script setup lang="ts">
/**
 * /console — the live platform console (F20). One dense, dark, institutional surface wired to the
 * real platform via the BFF composables: identity, model catalog, inference playground, wallet
 * (balances + transactions), buy-credits, and API keys. `EXASCALE_API_MODE=local` makes every panel
 * live; mock mode shows realistic canned data. Tokens only — no raw values here.
 */
definePageMeta({ layout: false, middleware: 'auth' })
useHead({ title: 'Console — Exascale', htmlAttrs: { 'data-theme': 'dark' } })

const { user, logout } = useAuth()
const { models, load: loadCatalog } = useCatalog()
const { balances, transactions, loadBalances, loadTransactions } = useWallet()
const { running, error: infError, insufficientCredit, result, run } = useInference()
const { checkout, loading: buying } = useBilling()
const { keys, newSecret, load: loadKeys, create: createKey, revoke: revokeKey, dismissSecret } = useKeys()

const apiMode = useRuntimeConfig().public.apiMode

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

onMounted(async () => {
  await Promise.allSettled([loadCatalog(), loadBalances(), loadTransactions(), loadKeys()])
})
</script>

<template>
  <div class="console" data-theme="dark">
    <!-- Topbar -->
    <header class="topbar">
      <div class="brand">
        <span class="mark">EXASCALE</span>
        <span class="sub">Console</span>
        <span class="mode" :class="apiMode === 'local' ? 'mode-live' : 'mode-mock'">
          {{ apiMode === 'local' ? 'live' : 'demo data' }}
        </span>
      </div>
      <div class="who">
        <span class="bal mono">{{ fmt(textBalance) }} <span class="unit">text</span></span>
        <span v-if="user?.is_paper" class="paper">paper</span>
        <span class="email">{{ user?.email }}</span>
        <button class="ghost" @click="logout().then(() => navigateTo('/login'))">Sign out</button>
      </div>
    </header>

    <!-- Balances strip -->
    <section class="strip">
      <div v-for="b in balances" :key="b.credit_type" class="bal-tile">
        <div class="label">{{ b.credit_type }}</div>
        <div class="num mono">{{ fmt(b.balance) }}</div>
        <div class="sub-meta">locked {{ fmt(b.locked_amount) }}</div>
      </div>
      <div v-if="!balances.length" class="bal-tile empty">No credits yet — buy some →</div>
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
  </div>
</template>

<style scoped>
.console {
  min-height: 100vh;
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

/* Topbar */
.topbar { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.brand { display: flex; align-items: baseline; gap: var(--sp-3); }
.mark { font-weight: 700; letter-spacing: var(--ls-wide); font-size: var(--fs-md); }
.sub { color: var(--text-3); font-size: var(--fs-sm); }
.mode { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); padding: 2px var(--sp-2); border-radius: var(--radius-sm); }
.mode-live { background: var(--pos-soft); color: var(--pos); }
.mode-mock { background: var(--overlay); color: var(--text-3); border: 1px solid var(--border); }
.who { display: flex; align-items: center; gap: var(--sp-4); font-size: var(--fs-sm); }
.who .bal { font-size: var(--fs-md); }
.who .unit { color: var(--text-3); font-size: var(--fs-xs); }
.paper { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--warn); border: 1px solid var(--border); padding: 1px var(--sp-2); border-radius: var(--radius-sm); }
.email { color: var(--text-2); }

/* Balances strip */
.strip { display: flex; gap: var(--sp-3); flex-wrap: wrap; }
.bal-tile { flex: 1; min-width: 160px; background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); padding: var(--sp-3) var(--sp-4); }
.bal-tile.empty { color: var(--text-3); display: flex; align-items: center; }
.bal-tile .label { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: var(--ls-wide); color: var(--text-3); }
.bal-tile .num { font-size: var(--fs-2xl); font-weight: 600; margin-top: var(--sp-1); }
.bal-tile .sub-meta { font-size: var(--fs-xs); color: var(--text-3); margin-top: 2px; }

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
.tbl .r { text-align: right; }
.tbl .muted { color: var(--text-3); }
.tbl .prefix { color: var(--text-2); }
.tbl .kname { color: var(--text); }
.tbl .acts { text-align: right; }
.pos { color: var(--pos); }
.neg { color: var(--neg); }

@media (max-width: 1100px) { .grid { grid-template-columns: 1fr; } }
</style>
