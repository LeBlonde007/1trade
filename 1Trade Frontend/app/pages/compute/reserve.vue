<script setup lang="ts">
/**
 * /compute/reserve — reserved GPU capacity (F14), live on compute-control. The quote updates as the
 * form changes; buying prepays in the tier's GPU credits and sets the GPUs aside for the term. The
 * form holds one Idempotency-Key per distinct order, so a double click or a retry after "payment
 * pending" never charges twice.
 */
import type { Reservation, ReservationQuote, ReservedCapacity, Term } from '~/composables/useCompute'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Compute · Reserved capacity — 1Trade' })

const compute = useCompute()
const form = reactive<{ gpu_type: string; gpus: number; term: Term }>({ gpu_type: 'gpu_h100', gpus: 8, term: '1mo' })
const q = ref<ReservationQuote | null>(null)
const available = ref<number | null>(null)
const quoteError = ref('')
const list = ref<Reservation[]>([])
const capacity = ref<ReservedCapacity[]>([])
const loadError = ref('')
const buying = ref(false)
const buyError = ref('')
const bought = ref<Reservation | null>(null)

/** newKey mints an idempotency key. */
const newKey = () => globalThis.crypto?.randomUUID?.().replace(/-/g, '') ?? `res${Date.now()}`
const key = ref(newKey())
const TIER: Record<string, string> = { gpu_h100: 'H100 80GB', gpu_h200: 'H200 141GB' }
const TERMS: { id: Term; label: string }[] = [
  { id: '1mo', label: '1 month · 17% off' }, { id: '6mo', label: '6 months · 27% off' }, { id: '12mo', label: '12 months · 33% off' },
]
const fmt = (v: string) => Number(v).toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 6 })

let timer: ReturnType<typeof setTimeout> | undefined
/** refreshQuote re-prices the form (debounced); a new order also gets a new idempotency key. */
function refreshQuote() {
  key.value = newKey()
  bought.value = null
  buyError.value = ''
  if (timer) clearTimeout(timer)
  timer = setTimeout(async () => {
    quoteError.value = ''
    if (!Number.isInteger(form.gpus) || form.gpus < 1 || form.gpus > 256) {
      q.value = null
      quoteError.value = 'GPUs must be a whole number from 1 to 256.'
      return
    }
    try {
      const r = await compute.quote(form.gpu_type, form.gpus, form.term)
      q.value = r.quote
      available.value = r.available
    } catch (e: any) {
      q.value = null
      quoteError.value = e?.data?.message || e?.statusMessage || 'Could not price this reservation.'
    }
  }, 250)
}
watch(form, refreshQuote, { deep: true })

/** load fetches the tenant's reservations. */
async function load() {
  loadError.value = ''
  try {
    const r = await compute.reservations()
    list.value = r.data
    capacity.value = r.capacity
  } catch (e: any) {
    loadError.value = e?.statusCode === 503
      ? 'Reserved capacity is not enabled in this environment.'
      : (e?.data?.message || e?.statusMessage || 'Could not load your reservations.')
  }
}

/** buy prepays the quoted reservation; errors are shown inline with what to do next. */
async function buy() {
  if (!q.value || buying.value) return
  buying.value = true
  buyError.value = ''
  try {
    bought.value = await compute.reserve(form.gpu_type, form.gpus, form.term, key.value)
    await load()
    const r = await compute.quote(form.gpu_type, form.gpus, form.term)
    available.value = r.available
  } catch (e: any) {
    const code = e?.data?.code
    buyError.value = code === 'insufficient_credit'
      ? `Not enough ${TIER[form.gpu_type]} GPU credits. Top up or convert credits in the wallet, then try again.`
      : code === 'capacity_unavailable'
        ? 'Not enough free GPUs of this tier to set aside right now. Try fewer GPUs.'
        : code === 'payment_pending'
          ? 'Payment is not confirmed yet. Your GPUs are held — press Reserve again to retry (you will not be charged twice).'
          : e?.statusCode === 403
            ? 'Buying reserved capacity needs the admin or billing role.'
            : (e?.data?.message || e?.statusMessage || 'The reservation failed.')
  } finally {
    buying.value = false
  }
}

onMounted(() => {
  refreshQuote()
  load()
})
onBeforeUnmount(() => timer && clearTimeout(timer))
</script>

<template>
  <div class="rs-page">
    <div class="subbar">
      <nav class="crumbs">
        <span>Account</span><span class="sep">›</span><NuxtLink to="/compute">Compute</NuxtLink>
        <span class="sep">›</span><span class="cur">Reserved capacity</span>
      </nav>
    </div>

    <main class="page">
      <section class="head">
        <h1>Reserved capacity</h1>
        <p class="sub">
          Prepay GPUs for a term at a discount. They are set aside for you — nobody else's work can take them — and
          your instances use them first, at no further charge.
        </p>
      </section>

      <section class="grid">
        <form class="card form" @submit.prevent="buy">
          <h2>New reservation</h2>
          <div class="row">
            <label><span>GPU</span>
              <select v-model="form.gpu_type">
                <option value="gpu_h100">NVIDIA H100 80GB</option>
                <option value="gpu_h200">NVIDIA H200 141GB</option>
              </select>
            </label>
            <label><span>GPUs</span>
              <input v-model.number="form.gpus" type="number" min="1" max="256" step="1" class="mono">
            </label>
          </div>
          <fieldset class="terms">
            <legend>Term</legend>
            <label v-for="t in TERMS" :key="t.id" class="term" :class="{ on: form.term === t.id }">
              <input v-model="form.term" type="radio" name="term" :value="t.id"> {{ t.label }}
            </label>
          </fieldset>
          <p v-if="quoteError" class="banner neg" role="alert">{{ quoteError }}</p>
          <p v-if="buyError" class="banner neg" role="alert">{{ buyError }}</p>
          <p v-if="bought" class="banner pos" aria-live="polite">
            Reserved {{ bought.gpus }} × {{ TIER[bought.gpu_type] }} until {{ bought.ends_at?.slice(0, 10) }}.
          </p>
          <button type="submit" class="btn-primary" :disabled="!q || buying || !!bought">
            {{ buying ? 'Reserving…' : bought ? 'Reserved' : 'Reserve and prepay' }}
          </button>
        </form>

        <div class="card quote" aria-live="polite">
          <h2>Quote</h2>
          <template v-if="q">
            <dl>
              <dt>GPU-hours in the term</dt><dd class="mono">{{ fmt(q.gpu_hours) }}</dd>
              <dt>At on-demand</dt><dd class="mono dim">{{ fmt(q.on_demand_credits) }}</dd>
              <dt>Discount</dt><dd class="mono">{{ q.discount_pct }}%</dd>
              <dt class="strong">You prepay</dt><dd class="mono strong">{{ fmt(q.price_credits) }} {{ TIER[q.gpu_type] }} credits</dd>
              <dt>You save</dt><dd class="mono pos-text">{{ fmt(q.saving_credits) }}</dd>
              <dt>Free to set aside now</dt><dd class="mono">{{ available ?? '—' }} GPUs</dd>
            </dl>
            <p class="small dim">
              One GPU credit is one on-demand GPU-hour of its tier. The price is taken from your GPU credit balance
              when you reserve; reservations are a prepaid commitment and are not cancellable.
            </p>
          </template>
          <p v-else class="small dim">Choose a GPU, a count and a term.</p>
        </div>
      </section>

      <section class="card">
        <header class="card-head"><h2>Your reservations</h2></header>
        <p v-if="loadError" class="banner neg" role="alert">{{ loadError }}</p>
        <div v-else-if="!list.length" class="empty">No reservations yet.</div>
        <div v-else class="table-wrap">
          <table>
            <thead>
              <tr><th>GPUs</th><th>Term</th><th class="num">Prepaid</th><th>Starts</th><th>Ends</th><th>Status</th></tr>
            </thead>
            <tbody>
              <tr v-for="r in list" :key="r.id">
                <td class="mono">{{ r.gpus }} × {{ TIER[r.gpu_type] ?? r.gpu_type }}</td>
                <td>{{ r.term }} · −{{ r.discount_pct }}%</td>
                <td class="num mono">{{ fmt(r.price_credits) }}</td>
                <td class="mono small">{{ r.starts_at?.slice(0, 10) ?? '—' }}</td>
                <td class="mono small">{{ r.ends_at?.slice(0, 10) ?? '—' }}</td>
                <td>
                  <span class="pill" :class="r.state === 'active' ? 'pos' : r.state === 'failed' ? 'neg' : 'dim'">{{ r.state.replace('_', ' ') }}</span>
                  <div v-if="r.failure" class="small dim">{{ r.failure }}</div>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-for="c in capacity" :key="c.gpu_type" class="small dim">
            {{ TIER[c.gpu_type] }}: {{ c.in_use_gpus }} of {{ c.reserved_gpus }} reserved GPUs in use now.
          </p>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.rs-page { min-height: 100%; background: var(--canvas); color: var(--text); }
.subbar { padding: 10px 24px; border-bottom: 1px solid var(--border); font-size: 12px; color: var(--text-2); }
.crumbs a { color: var(--text-2); }
.crumbs .sep { margin: 0 6px; color: var(--text-3); }
.crumbs .cur { color: var(--text); }
.page { max-width: 1040px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 16px; }
h1 { font-family: var(--font-display); font-size: 26px; margin: 0; letter-spacing: -0.02em; }
h2 { font-size: 15px; margin: 0 0 10px; }
.sub { margin: 4px 0 0; color: var(--text-2); font-size: 13px; max-width: 720px; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
.form { display: flex; flex-direction: column; gap: 12px; }
.row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-2); }
input, select { background: var(--canvas); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; color: var(--text); font-size: 14px; }
.terms { border: 0; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; }
.terms legend { font-size: 12px; color: var(--text-2); margin-bottom: 4px; }
.term { flex-direction: row; align-items: center; gap: 8px; font-size: 13px; color: var(--text); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; cursor: pointer; }
.term.on { border-color: var(--brand); }
.term input { padding: 0; }
dl { display: grid; grid-template-columns: 1fr auto; gap: 6px 12px; margin: 0 0 10px; font-size: 13px; }
dt { color: var(--text-2); }
dd { margin: 0; text-align: right; }
.strong { font-weight: 600; color: var(--text); }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.btn-primary { background: var(--brand); color: var(--text-on-accent); border: 0; padding: 9px 14px; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; align-self: flex-start; }
.btn-primary:disabled { opacity: 0.5; cursor: default; }
.banner { margin: 0; padding: 8px 12px; border-radius: 6px; font-size: 13px; }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.banner.pos { background: var(--pos-soft, var(--sunken)); color: var(--pos); }
.pos-text { color: var(--pos); }
.small { font-size: 12px; }
.dim { color: var(--text-2); }
.card-head { display: flex; justify-content: space-between; align-items: baseline; }
.empty { padding: 16px 0; color: var(--text-2); font-size: 13px; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; font-weight: 500; color: var(--text-2); font-size: 12px; padding: 8px; border-bottom: 1px solid var(--border); }
td { padding: 10px 8px; border-bottom: 1px solid var(--border); vertical-align: top; }
.num { text-align: right; }
.pill { display: inline-block; font-size: 11px; padding: 1px 8px; border-radius: 999px; border: 1px solid currentColor; }
.pill.pos { color: var(--pos); }
.pill.neg { color: var(--neg); }
.pill.dim { color: var(--text-2); }
@media (max-width: 760px) { .grid, .row { grid-template-columns: 1fr; } }
</style>
