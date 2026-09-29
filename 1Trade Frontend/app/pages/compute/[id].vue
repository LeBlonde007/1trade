<script setup lang="ts">
/**
 * /compute/[id] — one GPU instance, live from compute-control: state, GPU type and count, region,
 * image, how to connect, uptime and cost so far, plus stop / start / terminate. Telemetry and log
 * collection are not built yet, so the page says so instead of showing sample numbers.
 */
import type { Instance } from '~/composables/useCompute'

definePageMeta({ layout: 'app', middleware: 'auth' })

const route = useRoute()
const compute = useCompute()
const toasts = useToasts()

const id = computed(() => String(route.params.id))
const inst = ref<Instance | null>(null)
const error = ref('')
const loading = ref(true)
const now = ref(Date.now())

useHead({ title: () => `${inst.value?.id ?? 'Instance'} — Compute — 1Trade` })

/** HOURLY is the on-demand rate per GPU, matching the compute list page. */
const HOURLY: Record<string, number> = { gpu_h100: 2.99, gpu_h200: 3.49 }

/** load fetches the instance; a 404 shows "not found" rather than an error banner. */
async function load() {
  try {
    inst.value = await compute.getInstance(id.value)
    error.value = ''
  } catch (e: any) {
    error.value = e?.statusCode === 404 ? 'not_found' : (e?.data?.message || 'Could not load this instance.')
  } finally {
    loading.value = false
  }
}

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  load()
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })

const gpuName = computed(() => (inst.value?.gpu_type === 'gpu_h200' ? 'H200 141GB SXM5' : 'H100 80GB SXM5'))
const hourlyRate = computed(() => inst.value ? (HOURLY[inst.value.gpu_type] ?? 0) * inst.value.count : 0)
const uptimeSec = computed(() => {
  const s = inst.value?.started_at
  if (!s || inst.value?.state !== 'running') return 0
  return Math.max(0, Math.floor((now.value - new Date(s).getTime()) / 1000))
})
const costSoFar = computed(() => (uptimeSec.value / 3600) * hourlyRate.value)

/** fmtUsd formats dollars with two decimals. */
function fmtUsd(n: number) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
/** fmtUptime renders seconds as "2d 4h 05m". */
function fmtUptime(sec: number) {
  if (!sec) return '—'
  const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60)
  if (d) return `${d}d ${h}h ${String(m).padStart(2, '0')}m`
  if (h) return `${h}h ${String(m).padStart(2, '0')}m`
  return `${m}m ${String(sec % 60).padStart(2, '0')}s`
}

const tone = computed(() => ({
  running: 'pos', provisioning: 'info', starting: 'info', stopping: 'warn', stopped: 'neg', terminated: 'neg',
}[inst.value?.state ?? 'stopped']))

/** act runs a lifecycle action and replaces the instance with the server's answer. */
async function act(kind: 'stop' | 'start' | 'remove') {
  if (!inst.value) return
  try {
    inst.value = await compute[kind](inst.value.id)
    toasts.push({ tone: 'info', title: kind === 'remove' ? 'Instance terminated' : kind === 'stop' ? 'Stopping instance' : 'Starting instance' })
  } catch (e: any) {
    toasts.push({ tone: 'warn', title: 'Action failed', body: e?.data?.message || 'Try again in a moment.' })
  }
}

const confirmTerminate = ref(false)

/** copy puts text on the clipboard and confirms with a toast. */
function copy(t: string) {
  navigator.clipboard?.writeText(t).catch(() => {})
  toasts.push({ tone: 'info', title: 'Copied', body: t.length > 48 ? t.slice(0, 48) + '…' : t })
}
</script>

<template>
  <div class="inst-page">
    <NuxtLink to="/compute" class="back">← All instances</NuxtLink>

    <p v-if="loading" class="dim">Loading instance…</p>

    <div v-else-if="error === 'not_found'" class="card">
      <h1>Instance not found</h1>
      <p class="dim">There is no instance <span class="mono">{{ id }}</span> on your account.</p>
    </div>

    <p v-else-if="error" class="err">{{ error }}</p>

    <template v-else-if="inst">
      <header class="head">
        <div>
          <h1 class="mono">{{ inst.id }}</h1>
          <p class="sub">
            <span class="pill" :class="tone">{{ inst.state }}</span>
            {{ inst.count }} × {{ gpuName }} · {{ inst.region }}
            <span v-if="inst.is_paper" class="pill info">sandbox</span>
          </p>
        </div>
        <div class="actions">
          <button v-if="inst.state === 'running'" class="btn secondary" :disabled="compute.busy.value" @click="act('stop')">Stop</button>
          <button v-if="inst.state === 'stopped'" class="btn secondary" :disabled="compute.busy.value" @click="act('start')">Start</button>
          <template v-if="inst.state !== 'terminated'">
            <button v-if="!confirmTerminate" class="btn danger" @click="confirmTerminate = true">Terminate</button>
            <template v-else>
              <button class="btn danger" :disabled="compute.busy.value" @click="act('remove'); confirmTerminate = false">Confirm terminate</button>
              <button class="btn ghost" @click="confirmTerminate = false">Cancel</button>
            </template>
          </template>
        </div>
      </header>

      <div class="grid">
        <section class="card">
          <h2>Connect</h2>
          <template v-if="inst.connect?.ssh">
            <code class="cmd">{{ inst.connect.ssh }}</code>
            <button class="btn ghost sm" @click="copy(inst.connect.ssh!)">Copy SSH command</button>
          </template>
          <p v-else class="dim">Connection details appear once the instance is running.</p>
          <dl v-if="inst.connect?.jupyter || inst.connect?.http">
            <template v-if="inst.connect.jupyter"><dt>Jupyter</dt><dd class="mono">{{ inst.connect.jupyter }}</dd></template>
            <template v-if="inst.connect.http"><dt>HTTP</dt><dd class="mono">{{ inst.connect.http }}</dd></template>
          </dl>
        </section>

        <section class="card">
          <h2>Cost</h2>
          <dl>
            <dt>Rate</dt><dd class="mono">{{ fmtUsd(hourlyRate) }} / hr</dd>
            <dt>Uptime</dt><dd class="mono">{{ fmtUptime(uptimeSec) }}</dd>
            <dt>Cost this run</dt><dd class="mono emph">{{ fmtUsd(costSoFar) }}</dd>
            <dt>Projected · 24h</dt><dd class="mono">{{ fmtUsd(hourlyRate * 24) }}</dd>
          </dl>
          <p class="dim small">Billed per second from your GPU credits while running. Every debit is in your <NuxtLink to="/wallet">wallet</NuxtLink>.</p>
        </section>

        <section class="card">
          <h2>Configuration</h2>
          <dl>
            <dt>Image</dt><dd class="mono">{{ inst.image }}</dd>
            <dt>Supply</dt><dd class="mono">{{ inst.supply_source_id }}</dd>
            <dt>Idle stop</dt><dd class="mono">{{ inst.idle_stop_minutes ? inst.idle_stop_minutes + ' min' : 'off' }}</dd>
            <dt>Reserved</dt><dd>{{ inst.reserved ? 'yes — prepaid capacity' : 'no — on-demand' }}</dd>
            <dt>Created</dt><dd class="mono">{{ inst.created_at.slice(0, 19).replace('T', ' ') }} UTC</dd>
          </dl>
        </section>

        <section class="card">
          <h2>Telemetry and logs</h2>
          <p class="dim">
            GPU telemetry and log collection are not built yet. Connect over SSH and run
            <code>nvidia-smi</code> for utilization, memory and temperature.
          </p>
        </section>
      </div>
    </template>
  </div>
</template>

<style scoped>
.inst-page { max-width: 1100px; margin: 0 auto; padding: 24px; color: var(--text); }
.back { color: var(--text-3); font-size: var(--fs-sm); text-decoration: none; }
.back:hover { color: var(--text); }
.head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; margin: 16px 0 20px; flex-wrap: wrap; }
h1 { font-size: 22px; margin: 0 0 6px; }
.sub { margin: 0; color: var(--text-2); font-size: var(--fs-sm); display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.pill { font-family: var(--font-mono); font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; padding: 2px 8px; border-radius: var(--radius-sm); border: 1px solid var(--border); }
.pill.pos { color: var(--pos); } .pill.neg { color: var(--neg); } .pill.warn { color: var(--warn); } .pill.info { color: var(--brand); }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
@media (max-width: 800px) { .grid { grid-template-columns: 1fr; } }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 16px 18px; }
.card h2 { font-size: 13px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-3); margin: 0 0 12px; font-weight: 500; }
dl { display: grid; grid-template-columns: 140px 1fr; gap: 6px 12px; margin: 0 0 10px; font-size: var(--fs-sm); }
dt { color: var(--text-3); }
dd { margin: 0; word-break: break-all; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.emph { color: var(--text); font-weight: 600; }
.cmd { display: block; font-family: var(--font-mono); font-size: var(--fs-sm); background: var(--canvas); border: 1px solid var(--border); padding: 10px 12px; border-radius: var(--radius-sm); margin-bottom: 10px; word-break: break-all; }
code { font-family: var(--font-mono); }
.dim { color: var(--text-3); font-size: var(--fs-sm); line-height: 1.5; }
.small { font-size: var(--fs-xs); margin: 0; }
.err { color: var(--neg); }
.btn { height: 34px; padding: 0 14px; border-radius: var(--radius-sm); border: 1px solid var(--border-strong); background: var(--elevated); color: var(--text); font-family: var(--font-sans); font-size: 13px; cursor: pointer; }
.btn:hover:enabled { background: var(--hover); }
.btn:disabled { opacity: 0.45; cursor: not-allowed; }
.btn.ghost { background: transparent; border-color: var(--border); color: var(--text-2); }
.btn.danger { color: var(--neg); border-color: var(--neg); background: transparent; }
.btn.sm { height: 28px; padding: 0 10px; font-size: var(--fs-xs); }
</style>
