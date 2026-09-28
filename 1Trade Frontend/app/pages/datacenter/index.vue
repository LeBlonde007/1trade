<script setup lang="ts">
/**
 * /datacenter — the partner's supply dashboard (F17), live on compute-control's supply API
 * (supply.yaml v1.0). Every number is real: registered and healthy GPUs from the agent's heartbeats,
 * GPUs in use from the live scheduling pool, GPU time served from the usage records that payouts
 * (F18) are computed from. No mock data.
 */
import type { SupplySource, SourceUsage, Payout, Agreement, AttestationStatus, AttestationLayer } from '~/composables/useSupply'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Datacenter · Supply — 1Trade' })

const { sources, loading, error, load, usage, attestation, act, payouts, agreement, dispute } = useSupply()
const statements = ref<Payout[]>([])
const terms = ref<Agreement | null>(null)
const payoutError = ref('')

/** loadPayouts fetches statements and terms (independent of the sources table). */
async function loadPayouts() {
  payoutError.value = ''
  try {
    const [st, ag] = await Promise.all([payouts(), agreement()])
    statements.value = st
    terms.value = ag
  } catch (e: any) {
    payoutError.value = e?.data?.message || e?.statusMessage || 'Could not load payouts.'
  }
}

/** raiseDispute asks for a reason and disputes the statement. */
async function raiseDispute(p: Payout) {
  const reason = prompt('What is wrong with this statement? (sent to 1Trade operations)')
  if (!reason?.trim()) return
  try {
    await dispute(p.id, reason.trim())
    await loadPayouts()
  } catch (e: any) {
    payoutError.value = e?.data?.message || e?.statusMessage || 'Could not raise the dispute.'
  }
}

/** disputable: pending or wired, and still inside the window. */
const disputable = (p: Payout) => (p.state === 'pending' || p.state === 'wired') && Date.parse(p.dispute_until) > Date.now()
const day = (iso: string) => iso.slice(0, 10)
const usd = (v: string) => Number(v).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })
const usageBy = ref<Record<string, SourceUsage | null>>({})
const attestBy = ref<Record<string, AttestationStatus | null>>({})
const busy = ref<string>('')
const actionError = ref('')

/** loadAll fetches the sources, then each non-retired source's 30-day usage and attestation. */
async function loadAll() {
  await load()
  await loadDetail()
}

/** loadDetail fetches usage and attestation for every non-retired source; a failure shows as a dash. */
async function loadDetail() {
  const live = sources.value.filter(s => s.state !== 'retired')
  const [use, att] = await Promise.all([
    Promise.allSettled(live.map(s => usage(s.id))),
    Promise.allSettled(live.map(s => attestation(s.id))),
  ])
  const nextUse: Record<string, SourceUsage | null> = {}
  const nextAtt: Record<string, AttestationStatus | null> = {}
  live.forEach((s, i) => {
    const u = use[i]!, a = att[i]!
    nextUse[s.id] = u.status === 'fulfilled' ? u.value : null
    nextAtt[s.id] = a.status === 'fulfilled' ? a.value : null
  })
  usageBy.value = nextUse
  attestBy.value = nextAtt
}

onMounted(() => {
  loadAll()
  loadPayouts()
  const t = setInterval(loadAll, 30_000) // heartbeats (and telemetry attestation) land every 30 s
  onBeforeUnmount(() => clearInterval(t))
})

const live = computed(() => sources.value.filter(s => s.state !== 'retired'))
const totals = computed(() => {
  let registered = 0, healthy = 0, inUse = 0, schedulable = 0
  for (const s of live.value) {
    registered += s.gpu_count
    healthy += s.gpus_healthy ?? 0
    inUse += s.gpus_in_use
    if (s.schedulable) schedulable++
  }
  return { registered, healthy, inUse, schedulable }
})

const TIER: Record<string, string> = { gpu_h100: 'H100', gpu_h200: 'H200' }

/** gpuHours renders fixed-point GPU-seconds as GPU-hours, 1 decimal (display only). */
function gpuHours(u: SourceUsage | null | undefined): string {
  if (!u) return '—'
  const secs = u.by_tier.reduce((a, t) => a + Number(t.gpu_seconds), 0)
  return (secs / 3600).toLocaleString('en-US', { maximumFractionDigits: 1, minimumFractionDigits: 1 })
}

/** ago renders a heartbeat time relative to now. */
function ago(iso: string | null): string {
  if (!iso) return 'never'
  const s = Math.max(0, Math.round((Date.now() - Date.parse(iso)) / 1000))
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.round(s / 60)}m ago`
  return `${Math.round(s / 3600)}h ago`
}

const LAYER: Record<AttestationLayer, string> = {
  kyb: 'KYB review', hardware: 'GPU identity', challenge: 'challenge', telemetry: 'telemetry', bond: 'bond',
}

/** attestLabel summarises a source's attestation as "passed / 5". */
function attestLabel(a: AttestationStatus | null | undefined): string {
  return a ? `${5 - a.missing.length} / 5` : '—'
}

/** attestProblem names the first failing or drifted layer (with the reason), else what is still missing. */
function attestProblem(a: AttestationStatus | null | undefined): string {
  if (!a || a.complete) return ''
  for (const l of a.missing) {
    const r = a.layers[l]
    if (r && r.state !== 'pass') return `${LAYER[l]}: ${r.state}${r.detail ? ' — ' + r.detail : ''}`
  }
  return 'Awaiting ' + a.missing.map(l => LAYER[l]).join(', ')
}

/** why explains, in one phrase, why a source is or is not taking work. */
function why(s: SupplySource): string {
  if (s.state === 'pending') return 'Activates when attestation passes'
  if (s.state === 'suspended') return 'Suspended — draining'
  if (s.schedulable) return 'Taking work'
  if (!s.last_heartbeat_at) return 'Waiting for the agent’s first heartbeat'
  if ((s.gpus_healthy ?? 0) === 0) return 'Agent reports no healthy GPUs'
  return 'No heartbeat for 2 min — not taking new work'
}

/** run performs a lifecycle action, confirming the irreversible one. */
async function run(s: SupplySource, action: 'suspend' | 'resume' | 'retire') {
  if (action === 'retire' && !confirm(`Retire “${s.name}”? It stops taking work, drains, and cannot be reactivated.`)) return
  busy.value = s.id + action
  actionError.value = ''
  try {
    await act(s.id, action)
  } catch (e: any) {
    actionError.value = e?.data?.message || e?.statusMessage || `Could not ${action} ${s.name}.`
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <div class="dc-page">
    <div class="subbar">
      <nav class="crumbs"><span>Account</span><span class="sep">›</span><span class="cur">Datacenter</span></nav>
    </div>

    <main class="page">
      <section class="head">
        <div>
          <h1>Supply</h1>
          <p class="sub">GPUs your datacenter contributes to the 1Trade pool. Scheduled exactly like 1Trade’s own capacity.</p>
        </div>
        <NuxtLink to="/datacenter/register" class="btn-primary">Register capacity</NuxtLink>
      </section>

      <p v-if="error" class="banner neg" role="alert">{{ error }}</p>
      <p v-if="actionError" class="banner neg" role="alert">{{ actionError }}</p>

      <section class="stats" aria-label="Totals">
        <div class="stat"><span class="lbl">Registered GPUs</span><span class="val mono">{{ totals.registered }}</span></div>
        <div class="stat"><span class="lbl">Healthy</span><span class="val mono">{{ totals.healthy }}</span></div>
        <div class="stat"><span class="lbl">In use now</span><span class="val mono">{{ totals.inUse }}</span></div>
        <div class="stat"><span class="lbl">Sources taking work</span><span class="val mono">{{ totals.schedulable }} / {{ live.length }}</span></div>
      </section>

      <section class="card">
        <header class="card-head"><h2>Sources</h2><span class="dim">Refreshes every 30 s</span></header>
        <div v-if="loading && !sources.length" class="empty">Loading…</div>
        <div v-else-if="!live.length && !error" class="empty">
          <p>No capacity registered yet.</p>
          <NuxtLink to="/datacenter/register" class="link">Register your first GPUs →</NuxtLink>
        </div>
        <div v-else class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Source</th><th>GPU</th><th class="num">Healthy / registered</th><th class="num">In use</th>
                <th class="num">Util.</th><th>Heartbeat</th><th class="num">GPU-h · 30d</th><th>Attestation</th><th>Status</th><th />
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in live" :key="s.id">
                <td><div class="name">{{ s.name }}</div><div class="dim small">{{ s.region }} · {{ s.sla_tier }}</div></td>
                <td class="mono">{{ TIER[s.gpu_type] ?? s.gpu_type }}</td>
                <td class="num mono">{{ s.gpus_healthy ?? '—' }} / {{ s.gpu_count }}</td>
                <td class="num mono">{{ s.gpus_in_use }}</td>
                <td class="num mono">{{ s.utilization_pct == null ? '—' : s.utilization_pct + '%' }}</td>
                <td class="mono small">{{ ago(s.last_heartbeat_at) }}</td>
                <td class="num mono">{{ gpuHours(usageBy[s.id]) }}</td>
                <td>
                  <span class="mono" :class="attestBy[s.id]?.complete ? 'pos-text' : ''">{{ attestLabel(attestBy[s.id]) }}</span>
                  <div v-if="attestProblem(attestBy[s.id])" class="small dim">{{ attestProblem(attestBy[s.id]) }}</div>
                </td>
                <td>
                  <span class="pill" :class="s.schedulable ? 'pos' : s.state === 'suspended' ? 'warn' : 'dim'">{{ s.state }}</span>
                  <div class="small dim">{{ why(s) }}</div>
                </td>
                <td class="actions">
                  <button v-if="s.state === 'active'" type="button" :disabled="!!busy" @click="run(s, 'suspend')">Suspend</button>
                  <button v-if="s.state === 'suspended'" type="button" :disabled="!!busy" @click="run(s, 'resume')">Resume</button>
                  <button type="button" class="danger" :disabled="!!busy" @click="run(s, 'retire')">Retire</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="grid2">
        <div class="card">
          <header class="card-head"><h2>Connect your agent</h2></header>
          <p class="small">
            Your agent reports health every 30 seconds. A source takes work only while it is active, has
            heartbeated in the last 2 minutes, and reports healthy GPUs; otherwise running work finishes and
            nothing new lands (a drain).
          </p>
          <pre class="mono code">curl -X POST https://api.1trade.io/v1/supply/sources/&lt;SOURCE_ID&gt;/heartbeat \
  -H "Authorization: Bearer $ONETRADE_TOKEN" \
  -d '{"gpus_healthy": 16, "utilization_pct": 42, "ecc_errors": 0}'</pre>
        </div>
        <div class="card">
          <header class="card-head"><h2>Activation &amp; terms</h2></header>
          <p class="small">
            New sources start <strong>pending</strong> and activate by themselves once all five attestation
            layers pass: KYB review and bond (1Trade), and from your agent a signed GPU identity report, a
            timed challenge, and clean telemetry. ECC errors or a failed layer suspend the source until it
            is clean again.
          </p>
          <p v-if="terms" class="small mono">
            <template v-for="(rate, tier) in terms.rates" :key="tier">{{ TIER[tier] ?? tier }} ${{ usd(rate) }}/GPU-h · </template>
            fee {{ Number(terms.fee_percent) }}% · holdback {{ Number(terms.holdback_percent) }}% · dispute window {{ terms.dispute_days }} days
          </p>
          <p v-else class="small dim">Your payout terms have not been set yet. Usage is recorded meanwhile and is paid once they are.</p>
        </div>
      </section>

      <section class="card">
        <header class="card-head"><h2>Payouts</h2><span class="dim">Statements are computed from the GPU time your sources served</span></header>
        <p v-if="payoutError" class="banner neg" role="alert">{{ payoutError }}</p>
        <div v-if="!statements.length" class="empty">No statements yet. 1Trade closes payout cycles monthly.</div>
        <div v-else class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Period</th><th class="num">GPU-h</th><th class="num">Gross</th><th class="num">Fee</th>
                <th class="num">Holdback</th><th class="num">Released</th><th>Status</th><th />
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in statements" :key="p.id">
                <td class="mono small">{{ day(p.period_start) }} → {{ day(p.period_end) }}
                  <div v-if="p.is_paper" class="small dim">Simulated — sandbox usage, no money moves</div>
                </td>
                <td class="num mono">{{ Number(p.gpu_hours).toFixed(2) }}</td>
                <td class="num mono">${{ usd(p.gross) }}</td>
                <td class="num mono">${{ usd(p.fee) }}</td>
                <td class="num mono">${{ usd(p.holdback) }}</td>
                <td class="num mono">${{ usd(p.released) }}</td>
                <td>
                  <span class="pill" :class="p.state === 'settled' ? 'pos' : p.state === 'disputed' ? 'warn' : 'dim'">{{ p.state }}</span>
                  <div v-if="p.wire_reference" class="small dim mono">{{ p.wire_reference }}</div>
                  <div v-if="p.state === 'disputed'" class="small dim">In review with 1Trade</div>
                </td>
                <td class="actions">
                  <button v-if="disputable(p)" type="button" @click="raiseDispute(p)">Dispute</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.dc-page { min-height: 100%; background: var(--canvas); color: var(--text); }
.subbar { padding: 10px 24px; border-bottom: 1px solid var(--border); font-size: 12px; color: var(--text-2); }
.crumbs .sep { margin: 0 6px; color: var(--text-3); }
.crumbs .cur { color: var(--text); }
.page { max-width: 1200px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 20px; }
.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
h1 { font-family: var(--font-display); font-size: 26px; margin: 0; letter-spacing: -0.02em; }
h2 { font-size: 14px; margin: 0; }
.sub { margin: 4px 0 0; color: var(--text-2); font-size: 13px; }
.btn-primary { background: var(--brand); color: var(--text-on-accent); padding: 8px 14px; border-radius: 6px; font-size: 13px; text-decoration: none; font-weight: 600; }
.banner { margin: 0; padding: 8px 12px; border-radius: 6px; font-size: 13px; }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.stat { background: var(--elevated); border: 1px solid var(--border); border-radius: 8px; padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; }
.lbl { font-size: 11px; color: var(--text-2); letter-spacing: 0.02em; }
.val { font-size: 22px; }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: 8px; padding: 14px 16px; }
.card-head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 10px; }
.card-head .dim { font-size: 12px; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; font-weight: 500; color: var(--text-2); font-size: 11px; padding: 6px 8px; border-bottom: 1px solid var(--border); white-space: nowrap; }
td { padding: 10px 8px; border-bottom: 1px solid var(--border); vertical-align: top; }
.num { text-align: right; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.name { font-weight: 500; }
.small { font-size: 12px; }
.dim { color: var(--text-2); }
.pill { display: inline-block; font-size: 11px; padding: 1px 8px; border-radius: 999px; border: 1px solid currentColor; text-transform: lowercase; }
.pill.pos { color: var(--pos); }
.pill.warn { color: var(--warn); }
.pill.dim { color: var(--text-2); }
.pos-text { color: var(--pos); }
.actions { white-space: nowrap; text-align: right; }
.actions button { background: none; border: 1px solid var(--border); color: var(--text); border-radius: 5px; padding: 3px 8px; font-size: 12px; cursor: pointer; margin-left: 4px; }
.actions button.danger { color: var(--neg); }
.actions button:disabled { opacity: 0.5; cursor: default; }
.empty { padding: 24px 8px; color: var(--text-2); font-size: 13px; }
.link { color: var(--text); }
.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.code { background: var(--sunken); border: 1px solid var(--border); border-radius: 6px; padding: 10px; font-size: 12px; overflow-x: auto; white-space: pre; }
@media (max-width: 760px) {
  .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .grid2 { grid-template-columns: 1fr; }
}
</style>
