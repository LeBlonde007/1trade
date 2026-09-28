<script setup lang="ts">
/**
 * /datacenter — the partner's supply dashboard (F17), live on compute-control's supply API
 * (supply.yaml v1.0). Every number is real: registered and healthy GPUs from the agent's heartbeats,
 * GPUs in use from the live scheduling pool, GPU time served from the usage records that payouts
 * (F18) are computed from. No mock data.
 */
import type { SupplySource, SourceUsage } from '~/composables/useSupply'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Datacenter · Supply — 1Trade' })

const { sources, loading, error, load, usage, act } = useSupply()
const usageBy = ref<Record<string, SourceUsage | null>>({})
const busy = ref<string>('')
const actionError = ref('')

/** loadAll fetches the sources, then each non-retired source's 30-day usage. */
async function loadAll() {
  await load()
  const live = sources.value.filter(s => s.state !== 'retired')
  const results = await Promise.allSettled(live.map(s => usage(s.id)))
  const next: Record<string, SourceUsage | null> = {}
  live.forEach((s, i) => {
    const r = results[i]!
    next[s.id] = r.status === 'fulfilled' ? r.value : null
  })
  usageBy.value = next
}

onMounted(() => {
  loadAll()
  const t = setInterval(load, 30_000) // heartbeats land every 30 s
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

/** why explains, in one phrase, why a source is or is not taking work. */
function why(s: SupplySource): string {
  if (s.state === 'pending') return 'Awaiting activation by 1Trade'
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
                <th class="num">Util.</th><th>Heartbeat</th><th class="num">GPU-h · 30d</th><th>Status</th><th />
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
          <header class="card-head"><h2>Activation &amp; payouts</h2></header>
          <p class="small">
            New sources start <strong>pending</strong> and are activated by 1Trade after review. Automated GPU
            attestation is on the way.
          </p>
          <p class="small">
            Every GPU-second your sources serve is recorded against them (the GPU-hours above).
            Payouts are computed from these records; the payout statement arrives with the payouts release.
          </p>
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
