<script setup lang="ts">
/**
 * /compute/[id] — Compute instance detail (G1).
 * Dark, in-app. Overview · Logs · Snapshots · Events · Cost tabs.
 */
import {
  Cpu, MapPin, Activity, Square, RotateCcw, Maximize2, Trash2,
  Play, Pause, Download, Search, Camera, RefreshCw, ChevronRight, Clock,
  AlertTriangle, ShieldCheck, Circle, Copy
} from 'lucide-vue-next'

definePageMeta({ layout: 'app', middleware: 'auth' })

const route = useRoute()
const instanceId = computed(() => String(route.params.id || 'inst_8c4f2a1e'))

useHead({ title: () => `${instance.value.name} — Compute — 1Trade` })

interface Instance {
  id: string
  name: string
  region: string
  gpu: string
  gpuCount: number
  status: 'running' | 'stopped' | 'restarting' | 'provisioning' | 'terminating'
  spendToDate: number
  hourlyRate: number
  startedAt: string
  image: string
  diskGB: number
  ramGB: number
  vcpu: number
  ipv4: string
  sshUser: string
}

const instance = computed<Instance>(() => ({
  id: instanceId.value,
  name: 'training-llama3-frankfurt-04',
  region: 'eu-central-1 / Frankfurt zone-b',
  gpu: 'H100 SXM5 80GB',
  gpuCount: 8,
  status: 'running',
  spendToDate: 487.32,
  hourlyRate: 23.92,
  startedAt: '2026-05-22T08:14:00Z',
  image: 'exascale/cuda12.4-pytorch2.4-ubuntu22.04 (build #4218)',
  diskGB: 4096,
  ramGB: 1536,
  vcpu: 192,
  ipv4: '198.51.100.42',
  sshUser: 'exuser',
}))

const tabs = ['Overview', 'Logs', 'Snapshots', 'Events', 'Cost'] as const
type Tab = typeof tabs[number]
const activeTab = ref<Tab>('Overview')

// Live ticking ----------------------------------------
type GpuRow = { idx: number; util: number; mem: number; temp: number; power: number }
const gpus = ref<GpuRow[]>([])
const ramUsedGB = ref(932)
const ioRead  = ref<number[]>([])
const ioWrite = ref<number[]>([])
const netIn   = ref<number[]>([])
const netOut  = ref<number[]>([])

function seed(): GpuRow[] {
  return Array.from({ length: 8 }, (_, i) => ({
    idx: i,
    util:  85 + Math.round(Math.random() * 12),
    mem:   72 + Math.round(Math.random() * 22),
    temp:  64 + Math.round(Math.random() * 10),
    power: 580 + Math.round(Math.random() * 90),
  }))
}

function seedSpark(n: number, base: number, jitter: number) {
  return Array.from({ length: n }, () => base + (Math.random() - 0.5) * jitter)
}

onMounted(() => {
  gpus.value = seed()
  ioRead.value  = seedSpark(60, 240, 180)
  ioWrite.value = seedSpark(60, 60,  60)
  netIn.value   = seedSpark(60, 380, 220)
  netOut.value  = seedSpark(60, 120, 90)

  const t = setInterval(() => {
    gpus.value = gpus.value.map(g => {
      const drift = (Math.random() - 0.5) * 4
      return {
        ...g,
        util:  Math.max(40, Math.min(100, g.util + drift)),
        mem:   Math.max(40, Math.min(99,  g.mem  + drift * 0.4)),
        temp:  Math.max(55, Math.min(85,  g.temp + (Math.random() - 0.5) * 1.2)),
        power: Math.max(450, Math.min(700, g.power + (Math.random() - 0.5) * 18)),
      }
    })
    ramUsedGB.value = Math.max(800, Math.min(1500, ramUsedGB.value + (Math.random() - 0.5) * 12))

    const push = (arr: Ref<number[]>, base: number, jit: number) => {
      arr.value = [...arr.value.slice(1), base + (Math.random() - 0.5) * jit]
    }
    push(ioRead,  240, 200)
    push(ioWrite,  60,  70)
    push(netIn,   380, 220)
    push(netOut,  120,  90)
  }, 1500)
  onBeforeUnmount(() => clearInterval(t))
})

const fmtUSD = (n: number) => `$${n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

// Sparkline path
function spark(values: number[], w = 220, h = 36) {
  if (!values.length) return ''
  const lo = Math.min(...values), hi = Math.max(...values)
  const range = hi - lo || 1
  return values.map((v, i) => {
    const x = (i / (values.length - 1)) * w
    const y = h - ((v - lo) / range) * (h - 4) - 2
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

const startedDuration = computed(() => {
  const ms = Date.now() - new Date(instance.value.startedAt).getTime()
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  return `${h}h ${m}m`
})

// Logs -----------------------------------------------
interface LogLine { ts: string; stream: 'stdout' | 'stderr'; body: string }
const logBuffer = ref<LogLine[]>([])
const followLogs = ref(true)
const logSearch  = ref('')
const logFilterStream = ref<'all' | 'stdout' | 'stderr'>('all')
const logEl = ref<HTMLElement | null>(null)

const sampleLines = [
  ['stdout', 'epoch 14/30 step 4218/12600 loss 1.842 lr 1.4e-4 grad_norm 0.81 tok/s 14920'],
  ['stdout', 'epoch 14/30 step 4219/12600 loss 1.839 lr 1.4e-4 grad_norm 0.79 tok/s 14881'],
  ['stdout', 'checkpoint saved → s3://exa-runs/llama3-fra-04/ckpt-4200.pt (size 27.4 GiB)'],
  ['stderr', '[WARN] NCCL: ring 0 hop 6→7 latency 124us above p99 threshold (90us)'],
  ['stdout', 'epoch 14/30 step 4220/12600 loss 1.835 lr 1.4e-4 grad_norm 0.83 tok/s 15012'],
  ['stdout', 'rank 0: gpu0 mem 72.3 GiB · gpu1 71.9 · gpu2 72.1 · gpu3 71.8 · gpu4 72.4 · gpu5 71.9 · gpu6 72.0 · gpu7 72.2'],
  ['stdout', 'epoch 14/30 step 4221/12600 loss 1.831 lr 1.4e-4 grad_norm 0.77 tok/s 15098'],
  ['stderr', '[INFO] dataloader: worker-12 stall 0.42s (queue depth 3)'],
  ['stdout', 'epoch 14/30 step 4222/12600 loss 1.828 lr 1.4e-4 grad_norm 0.80 tok/s 15044'],
] as const

function nowStamp(off = 0) {
  const d = new Date(Date.now() + off)
  const pad = (n: number, l = 2) => String(n).padStart(l, '0')
  return `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}.${pad(d.getUTCMilliseconds(), 3)}`
}

onMounted(() => {
  logBuffer.value = sampleLines.map((l, i) => ({
    ts: nowStamp(-1000 * (sampleLines.length - i)),
    stream: l[0] as 'stdout' | 'stderr',
    body: l[1],
  }))
  const t = setInterval(() => {
    if (!followLogs.value) return
    const s = sampleLines[Math.floor(Math.random() * sampleLines.length)]
    logBuffer.value = [...logBuffer.value.slice(-200), { ts: nowStamp(), stream: s[0] as 'stdout' | 'stderr', body: s[1] }]
    nextTick(() => {
      if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
    })
  }, 900)
  onBeforeUnmount(() => clearInterval(t))
})

const filteredLogs = computed(() => {
  const q = logSearch.value.trim().toLowerCase()
  return logBuffer.value.filter(l => {
    if (logFilterStream.value !== 'all' && l.stream !== logFilterStream.value) return false
    if (q && !l.body.toLowerCase().includes(q)) return false
    return true
  })
})

// Snapshots ------------------------------------------
const snapshots = ref([
  { id: 'snap_4f81', label: 'pre-llama3-finetune',  sizeGB: 312, createdAt: '2026-05-22 08:11 UTC', auto: false },
  { id: 'snap_5c02', label: 'daily-auto-2026-05-23', sizeGB: 318, createdAt: '2026-05-23 02:00 UTC', auto: true },
  { id: 'snap_5e91', label: 'daily-auto-2026-05-24', sizeGB: 324, createdAt: '2026-05-24 02:00 UTC', auto: true },
])

// Events ---------------------------------------------
interface Event { ts: string; type: string; level: 'info' | 'warn' | 'success'; body: string; actor: string }
const events = ref<Event[]>([
  { ts: '2026-05-24 13:14:08 UTC', type: 'snapshot',      level: 'success', body: 'Snapshot daily-auto-2026-05-24 created (324 GiB)',                                                  actor: 'system' },
  { ts: '2026-05-23 02:00:11 UTC', type: 'snapshot',      level: 'success', body: 'Snapshot daily-auto-2026-05-23 created (318 GiB)',                                                  actor: 'system' },
  { ts: '2026-05-22 14:02:47 UTC', type: 'resize',        level: 'info',    body: 'Disk volume expanded 2048 → 4096 GiB (online)',                                                     actor: 'mark.s@deepminds.me' },
  { ts: '2026-05-22 08:14:02 UTC', type: 'provisioning',  level: 'success', body: 'Instance provisioned · 8× H100 SXM5 80GB · zone-b host hv-fra-b-218',                              actor: 'system' },
  { ts: '2026-05-22 08:13:41 UTC', type: 'provisioning',  level: 'info',    body: 'Image exascale/cuda12.4-pytorch2.4-ubuntu22.04 pulled (4.2 GiB, 31s)',                              actor: 'system' },
  { ts: '2026-05-22 08:13:09 UTC', type: 'request',      level: 'info',    body: 'Launch requested via API · key prod_eu_runs (4)',                                                   actor: 'mark.s@deepminds.me' },
])

// Cost -----------------------------------------------
const costSeries = computed(() => {
  // 48 hourly bars representing last 48h spend
  return Array.from({ length: 48 }, (_, i) => {
    const base = 23.92
    const jitter = Math.sin(i / 6) * 2 + (Math.random() - 0.5) * 1.4
    return Math.max(18, base + jitter)
  })
})

const monthlyProjection = computed(() => instance.value.hourlyRate * 24 * 30)

// Header status pill
const statusTone = computed(() => ({
  running:       'pos',
  stopped:       'neg',
  restarting:    'warn',
  provisioning:  'info',
  terminating:   'neg',
}[instance.value.status]))

const showTerminate = ref(false)
const terminateConfirm = ref('')
const toasts = useToasts?.() ?? null

function onTerminate() {
  if (terminateConfirm.value !== instance.value.name) return
  toasts?.push({ tone: 'warn', title: 'Termination scheduled', body: `${instance.value.name} will stop within 60s.` })
  showTerminate.value = false
}

function copyText(t: string) {
  navigator.clipboard?.writeText(t)
  toasts?.push({ tone: 'info', title: 'Copied', body: t.length > 48 ? t.slice(0, 48) + '…' : t })
}
</script>

<template>
  <div class="page">
    <!-- Header --------------------------------------- -->
    <header class="head">
      <div class="crumbs">
        <NuxtLink to="/compute">Compute</NuxtLink>
        <ChevronRight :size="12" :stroke-width="1.5" />
        <NuxtLink to="/compute">Instances</NuxtLink>
        <ChevronRight :size="12" :stroke-width="1.5" />
        <span class="mono">{{ instance.id }}</span>
      </div>

      <div class="head-main">
        <div class="head-left">
          <h1>{{ instance.name }}</h1>
          <div class="head-meta">
            <span class="meta-pill" :class="`tone-${statusTone}`">
              <span class="pill-dot" />
              {{ instance.status }}
            </span>
            <span class="meta-item"><MapPin :size="12" :stroke-width="1.7" />{{ instance.region }}</span>
            <span class="meta-item"><Cpu :size="12" :stroke-width="1.7" />{{ instance.gpuCount }}× {{ instance.gpu }}</span>
            <span class="meta-item"><Clock :size="12" :stroke-width="1.7" />up {{ startedDuration }}</span>
            <span class="meta-item mono id-pill">
              {{ instance.id }}
              <button class="copy-btn" @click="copyText(instance.id)"><Copy :size="11" :stroke-width="1.7" /></button>
            </span>
          </div>
        </div>

        <div class="head-right">
          <div class="kpi">
            <span class="caps">Spend to date</span>
            <span class="mono kpi-value">{{ fmtUSD(instance.spendToDate) }}</span>
            <span class="kpi-sub mono">{{ fmtUSD(instance.hourlyRate) }} / hr</span>
          </div>
        </div>
      </div>

      <div class="actions-bar">
        <button class="btn"><Pause :size="13" :stroke-width="1.7" /> Stop</button>
        <button class="btn"><RotateCcw :size="13" :stroke-width="1.7" /> Restart</button>
        <button class="btn"><Maximize2 :size="13" :stroke-width="1.7" /> Resize</button>
        <button class="btn"><Camera :size="13" :stroke-width="1.7" /> Snapshot</button>
        <span class="actions-spacer" />
        <button class="btn btn-danger" @click="showTerminate = true"><Trash2 :size="13" :stroke-width="1.7" /> Terminate</button>
      </div>

      <nav class="tabs">
        <button
          v-for="t in tabs"
          :key="t"
          class="tab"
          :class="{ active: activeTab === t }"
          @click="activeTab = t"
        >{{ t }}</button>
      </nav>
    </header>

    <main class="body">
      <!-- Overview ----------------------------------- -->
      <section v-show="activeTab === 'Overview'" class="overview">
        <div class="grid-3">
          <article class="card">
            <header class="card-head">
              <h3>Per-GPU utilization</h3>
              <span class="caps">live · 1.5s</span>
            </header>
            <ul class="gpu-list">
              <li v-for="g in gpus" :key="g.idx">
                <span class="gpu-label">GPU{{ g.idx }}</span>
                <div class="gpu-bar">
                  <div class="gpu-bar-fill" :style="{ width: g.util + '%' }" />
                </div>
                <span class="mono gpu-pct">{{ g.util.toFixed(0) }}%</span>
                <span class="mono gpu-aux">{{ g.mem.toFixed(0) }}% mem</span>
                <span class="mono gpu-aux temp" :class="{ hot: g.temp >= 78 }">{{ g.temp.toFixed(0) }}°C</span>
                <span class="mono gpu-aux">{{ g.power.toFixed(0) }} W</span>
              </li>
            </ul>
            <footer class="card-foot">
              <span class="mono">Σ util {{ (gpus.reduce((a, b) => a + b.util, 0) / 8).toFixed(1) }}%</span>
              <span class="dot-sep">·</span>
              <span class="mono">Σ pwr {{ gpus.reduce((a, b) => a + b.power, 0).toFixed(0) }} W</span>
              <span class="dot-sep">·</span>
              <span class="mono">NVLink: healthy</span>
            </footer>
          </article>

          <article class="card">
            <header class="card-head">
              <h3>System RAM</h3>
              <span class="caps">live</span>
            </header>
            <div class="ram-gauge">
              <svg viewBox="0 0 200 200" width="200" height="200">
                <circle cx="100" cy="100" r="86" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="14" />
                <circle
                  cx="100" cy="100" r="86" fill="none"
                  stroke="var(--info)" stroke-width="14"
                  stroke-linecap="butt"
                  :stroke-dasharray="`${(ramUsedGB / instance.ramGB) * 540} 540`"
                  transform="rotate(-90 100 100)"
                />
                <text x="100" y="96" text-anchor="middle" class="ram-num">{{ ramUsedGB.toFixed(0) }}</text>
                <text x="100" y="120" text-anchor="middle" class="ram-denom">/ {{ instance.ramGB }} GiB</text>
              </svg>
            </div>
            <ul class="kv-grid">
              <li><span class="caps">vCPU</span><span class="mono">{{ instance.vcpu }}</span></li>
              <li><span class="caps">Disk</span><span class="mono">{{ (instance.diskGB / 1024).toFixed(1) }} TiB</span></li>
              <li><span class="caps">IPv4</span><span class="mono">{{ instance.ipv4 }}</span></li>
              <li><span class="caps">SSH user</span><span class="mono">{{ instance.sshUser }}</span></li>
            </ul>
          </article>

          <article class="card">
            <header class="card-head">
              <h3>Throughput</h3>
              <span class="caps">last 90s</span>
            </header>
            <ul class="metric-list">
              <li>
                <div class="m-row">
                  <span class="caps">Disk read</span>
                  <span class="mono m-val">{{ ioRead.at(-1)?.toFixed(0) }} <span class="m-unit">MiB/s</span></span>
                </div>
                <svg :viewBox="`0 0 220 36`" class="sparkline">
                  <path :d="spark(ioRead)" />
                </svg>
              </li>
              <li>
                <div class="m-row">
                  <span class="caps">Disk write</span>
                  <span class="mono m-val">{{ ioWrite.at(-1)?.toFixed(0) }} <span class="m-unit">MiB/s</span></span>
                </div>
                <svg :viewBox="`0 0 220 36`" class="sparkline">
                  <path :d="spark(ioWrite)" />
                </svg>
              </li>
              <li>
                <div class="m-row">
                  <span class="caps">Net in</span>
                  <span class="mono m-val">{{ netIn.at(-1)?.toFixed(0) }} <span class="m-unit">MiB/s</span></span>
                </div>
                <svg :viewBox="`0 0 220 36`" class="sparkline">
                  <path :d="spark(netIn)" />
                </svg>
              </li>
              <li>
                <div class="m-row">
                  <span class="caps">Net out</span>
                  <span class="mono m-val">{{ netOut.at(-1)?.toFixed(0) }} <span class="m-unit">MiB/s</span></span>
                </div>
                <svg :viewBox="`0 0 220 36`" class="sparkline">
                  <path :d="spark(netOut)" />
                </svg>
              </li>
            </ul>
          </article>
        </div>

        <article class="card config">
          <header class="card-head">
            <h3>Configuration</h3>
          </header>
          <ul class="cfg-grid">
            <li>
              <span class="caps">Image</span>
              <span class="mono">{{ instance.image }}</span>
            </li>
            <li>
              <span class="caps">SSH command</span>
              <span class="mono ssh-line">
                ssh -i ~/.ssh/exascale_ed25519 {{ instance.sshUser }}@{{ instance.ipv4 }}
                <button class="copy-btn" @click="copyText(`ssh -i ~/.ssh/exascale_ed25519 ${instance.sshUser}@${instance.ipv4}`)"><Copy :size="11" :stroke-width="1.7" /></button>
              </span>
            </li>
            <li>
              <span class="caps">Tags</span>
              <span class="tags">
                <span class="tag">team:research</span>
                <span class="tag">project:llama3-rlhf</span>
                <span class="tag">env:prod</span>
              </span>
            </li>
          </ul>
        </article>
      </section>

      <!-- Logs ---------------------------------------- -->
      <section v-show="activeTab === 'Logs'" class="logs">
        <header class="logs-toolbar">
          <div class="search-wrap">
            <Search :size="13" :stroke-width="1.7" />
            <input v-model="logSearch" placeholder="Search logs…">
          </div>
          <div class="seg">
            <button :class="{ active: logFilterStream === 'all' }"    @click="logFilterStream = 'all'">All</button>
            <button :class="{ active: logFilterStream === 'stdout' }" @click="logFilterStream = 'stdout'">stdout</button>
            <button :class="{ active: logFilterStream === 'stderr' }" @click="logFilterStream = 'stderr'">stderr</button>
          </div>
          <span class="spacer" />
          <button class="btn-sm" @click="followLogs = !followLogs">
            <component :is="followLogs ? Pause : Play" :size="12" :stroke-width="1.7" />
            {{ followLogs ? 'Pause follow' : 'Resume follow' }}
          </button>
          <button class="btn-sm"><Download :size="12" :stroke-width="1.7" /> Download</button>
        </header>

        <div ref="logEl" class="log-pane">
          <pre v-for="(l, i) in filteredLogs" :key="i" class="log-line" :class="`stream-${l.stream}`">
            <span class="log-ts">{{ l.ts }}</span><span class="log-stream">{{ l.stream }}</span><span class="log-body">{{ l.body }}</span>
          </pre>
        </div>

        <footer class="log-foot">
          <span class="mono">{{ filteredLogs.length.toLocaleString() }} lines shown</span>
          <span class="dot-sep">·</span>
          <span class="mono">Buffer: last 200 lines · full history in object storage</span>
          <span class="dot-sep">·</span>
          <span class="status-dot" :class="{ live: followLogs }" />
          <span>{{ followLogs ? 'Live' : 'Paused' }}</span>
        </footer>
      </section>

      <!-- Snapshots ----------------------------------- -->
      <section v-show="activeTab === 'Snapshots'" class="snapshots">
        <header class="snap-head">
          <div>
            <h3>Volume snapshots</h3>
            <p class="caps">Crash-consistent · stored in eu-central-1 + cross-region replica</p>
          </div>
          <button class="btn"><Camera :size="13" :stroke-width="1.7" /> Take snapshot now</button>
        </header>

        <table class="snap-table">
          <thead>
            <tr>
              <th>Label</th>
              <th>ID</th>
              <th class="num">Size</th>
              <th>Created</th>
              <th>Origin</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in snapshots" :key="s.id">
              <td>{{ s.label }}</td>
              <td><span class="mono">{{ s.id }}</span></td>
              <td class="num mono">{{ s.sizeGB }} GiB</td>
              <td class="mono">{{ s.createdAt }}</td>
              <td>
                <span class="origin-pill" :class="{ auto: s.auto }">{{ s.auto ? 'auto-daily' : 'manual' }}</span>
              </td>
              <td class="action-cell">
                <button class="link-btn"><RefreshCw :size="11" :stroke-width="1.7" /> Restore</button>
                <button class="link-btn"><Download :size="11" :stroke-width="1.7" /> Export</button>
                <button class="link-btn danger"><Trash2 :size="11" :stroke-width="1.7" /></button>
              </td>
            </tr>
          </tbody>
        </table>

        <div class="snap-policy">
          <ShieldCheck :size="14" :stroke-width="1.7" />
          <span>Retention: 14 daily, 6 weekly, 6 monthly. Restoring a snapshot creates a new volume; current volume is preserved.</span>
        </div>
      </section>

      <!-- Events -------------------------------------- -->
      <section v-show="activeTab === 'Events'" class="events">
        <ul class="timeline">
          <li v-for="(e, i) in events" :key="i" :class="`level-${e.level}`">
            <span class="t-dot" />
            <div class="t-content">
              <div class="t-head">
                <span class="t-type">{{ e.type }}</span>
                <span class="mono t-ts">{{ e.ts }}</span>
              </div>
              <p>{{ e.body }}</p>
              <span class="t-actor mono">{{ e.actor }}</span>
            </div>
          </li>
        </ul>
      </section>

      <!-- Cost --------------------------------------- -->
      <section v-show="activeTab === 'Cost'" class="cost">
        <div class="cost-grid">
          <article class="card kpi-card">
            <span class="caps">Hourly rate</span>
            <span class="mono kpi-big">{{ fmtUSD(instance.hourlyRate) }}</span>
            <span class="kpi-sub">8× H100 SXM5 + storage + egress</span>
          </article>
          <article class="card kpi-card">
            <span class="caps">Spend MTD</span>
            <span class="mono kpi-big">{{ fmtUSD(instance.spendToDate) }}</span>
            <span class="kpi-sub mono pos">▲ on plan</span>
          </article>
          <article class="card kpi-card">
            <span class="caps">Projected month</span>
            <span class="mono kpi-big">{{ fmtUSD(monthlyProjection) }}</span>
            <span class="kpi-sub">Assumes 24/7 from now</span>
          </article>
          <article class="card kpi-card">
            <span class="caps">Budget remaining</span>
            <span class="mono kpi-big">{{ fmtUSD(25000 - instance.spendToDate) }}</span>
            <span class="kpi-sub">Of $25,000 project cap</span>
          </article>
        </div>

        <article class="card">
          <header class="card-head">
            <h3>Per-hour spend · last 48 h</h3>
            <span class="caps">USD</span>
          </header>
          <div class="cost-chart">
            <div
              v-for="(v, i) in costSeries"
              :key="i"
              class="cost-bar"
              :style="{ height: ((v / 28) * 100) + '%' }"
              :title="`-${48 - i}h: ${fmtUSD(v)}`"
            />
          </div>
          <div class="cost-axis">
            <span>-48h</span><span>-36h</span><span>-24h</span><span>-12h</span><span>now</span>
          </div>
        </article>

        <article class="card breakdown">
          <header class="card-head">
            <h3>Breakdown</h3>
          </header>
          <ul class="breakdown-list">
            <li><span>GPU compute (8× H100 SXM5)</span><span class="mono">{{ fmtUSD(8 * 2.74) }} / hr</span></li>
            <li><span>Provisioned IOPS storage (4 TiB SSD)</span><span class="mono">{{ fmtUSD(0.92) }} / hr</span></li>
            <li><span>System (vCPU + RAM)</span><span class="mono">{{ fmtUSD(0.48) }} / hr</span></li>
            <li><span>Egress (avg 380 MiB/s)</span><span class="mono">{{ fmtUSD(0.60) }} / hr</span></li>
            <li class="sep"></li>
            <li class="total"><span>Total</span><span class="mono">{{ fmtUSD(instance.hourlyRate) }} / hr</span></li>
          </ul>
        </article>
      </section>
    </main>

    <!-- Terminate confirm modal ---------------------- -->
    <Transition name="fade">
      <div v-if="showTerminate" class="modal-scrim" @click.self="showTerminate = false">
        <div class="modal">
          <header>
            <AlertTriangle :size="16" :stroke-width="1.7" />
            <h3>Terminate instance</h3>
          </header>
          <p>
            Terminating <span class="mono">{{ instance.name }}</span> will release
            <span class="mono">{{ instance.gpuCount }}× {{ instance.gpu }}</span>
            and destroy the root volume. Snapshots are preserved.
            This cannot be undone.
          </p>
          <label>
            <span class="caps">Type the instance name to confirm</span>
            <input v-model="terminateConfirm" :placeholder="instance.name">
          </label>
          <footer>
            <button class="btn" @click="showTerminate = false">Cancel</button>
            <button class="btn btn-danger" :disabled="terminateConfirm !== instance.name" @click="onTerminate">
              <Trash2 :size="13" :stroke-width="1.7" /> Terminate
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.page {
  background: var(--surface-canvas);
  color: var(--text-primary);
  min-height: 100vh;
  font-family: 'Inter', system-ui, sans-serif;
  font-feature-settings: 'tnum';
}

.mono { font-family: 'JetBrains Mono', ui-monospace, monospace; font-variant-numeric: tabular-nums; }

.caps {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}

.dot-sep { color: var(--text-tertiary); margin: 0 4px; }
.num { text-align: right; }
.pos { color: #19C37D; }

/* Header ------------------------------------------- */
.head {
  background: var(--surface-elevated);
  border-bottom: 1px solid rgba(255,255,255,0.08);
  padding: 16px 24px 0;
}
.crumbs {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-tertiary);
  margin-bottom: 12px;
}
.crumbs a { color: var(--text-secondary); text-decoration: none; }
.crumbs a:hover { color: var(--text-primary); }

.head-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  flex-wrap: wrap;
}
.head-left h1 {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  margin: 0 0 8px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.head-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--text-secondary);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.meta-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 3px 8px;
  border-radius: 2px;
}
.pill-dot { width: 6px; height: 6px; border-radius: 50%; }
.meta-pill.tone-pos  { background: rgba(25,195,125,0.12); color: #19C37D; }
.meta-pill.tone-pos  .pill-dot { background: #19C37D; box-shadow: 0 0 0 3px rgba(25,195,125,0.25); animation: pulse 2.5s infinite; }
.meta-pill.tone-warn { background: rgba(245,158,11,0.12); color: #F5A524; }
.meta-pill.tone-warn .pill-dot { background: #F5A524; }
.meta-pill.tone-neg  { background: rgba(239,68,68,0.12); color: #EF4444; }
.meta-pill.tone-neg  .pill-dot { background: #EF4444; }
.meta-pill.tone-info { background: rgba(74,144,226,0.12); color: var(--info); }
.meta-pill.tone-info .pill-dot { background: var(--info); }

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(25,195,125,0.25); }
  50%      { box-shadow: 0 0 0 5px rgba(25,195,125,0.08); }
}

.id-pill {
  background: rgba(255,255,255,0.04);
  padding: 2px 6px;
  border-radius: 2px;
  font-size: 11px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.copy-btn {
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  padding: 0;
  display: inline-flex;
}
.copy-btn:hover { color: var(--text-primary); }

.kpi { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; }
.kpi-value { font-size: 24px; font-weight: 600; color: var(--text-primary); }
.kpi-sub { font-size: 11px; color: var(--text-tertiary); }

.actions-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 16px 0;
}
.btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  background: rgba(255,255,255,0.04);
  color: var(--text-primary);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 2px;
  cursor: pointer;
  font-family: inherit;
  transition: background 100ms ease, border-color 100ms ease;
}
.btn:hover { background: rgba(255,255,255,0.08); border-color: rgba(255,255,255,0.16); }
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-danger { color: #EF4444; }
.btn-danger:hover { background: rgba(239,68,68,0.10); border-color: rgba(239,68,68,0.32); }
.btn-danger:not(:disabled) { color: #EF4444; }
.actions-spacer { flex: 1; }

.tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid rgba(255,255,255,0.08);
  margin: 0 -24px;
  padding: 0 24px;
}
.tab {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  font-family: inherit;
}
.tab:hover { color: var(--text-primary); }
.tab.active {
  color: var(--text-primary);
  border-bottom-color: var(--brand-accent);
}

/* Body --------------------------------------------- */
.body {
  padding: 20px 24px 64px;
  max-width: 1440px;
  margin: 0 auto;
}

.grid-3 {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1.1fr;
  gap: 16px;
  margin-bottom: 16px;
}
@media (max-width: 1280px) {
  .grid-3 { grid-template-columns: 1fr 1fr; }
}
@media (max-width: 880px) {
  .grid-3 { grid-template-columns: 1fr; }
}

.card {
  background: var(--surface-elevated);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 4px;
  padding: 16px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 12px;
}
.card-head h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.card-foot {
  display: flex;
  align-items: center;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid rgba(255,255,255,0.06);
  font-size: 11px;
  color: var(--text-tertiary);
}

/* GPU list */
.gpu-list { list-style: none; padding: 0; margin: 0; }
.gpu-list li {
  display: grid;
  grid-template-columns: 44px 1fr 48px 64px 56px 56px;
  gap: 8px;
  align-items: center;
  padding: 4px 0;
  font-size: 11px;
}
.gpu-label { color: var(--text-tertiary); font-size: 11px; }
.gpu-bar { background: rgba(255,255,255,0.06); height: 6px; border-radius: 1px; overflow: hidden; }
.gpu-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #19C37D, var(--info));
  transition: width 600ms ease;
}
.gpu-pct { font-size: 12px; font-weight: 500; text-align: right; }
.gpu-aux { font-size: 11px; color: var(--text-secondary); text-align: right; }
.gpu-aux.temp.hot { color: #F5A524; }

/* RAM */
.ram-gauge { display: flex; justify-content: center; padding: 4px 0 12px; }
.ram-num { fill: var(--text-primary); font-size: 36px; font-family: 'JetBrains Mono', ui-monospace, monospace; font-weight: 600; }
.ram-denom { fill: var(--text-tertiary); font-size: 12px; font-family: 'JetBrains Mono', ui-monospace, monospace; }
.kv-grid {
  list-style: none;
  padding: 12px 0 0;
  margin: 0;
  border-top: 1px solid rgba(255,255,255,0.06);
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 16px;
}
.kv-grid li { display: flex; justify-content: space-between; font-size: 12px; }
.kv-grid .caps { font-size: 10px; }

/* Throughput */
.metric-list { list-style: none; padding: 0; margin: 0; }
.metric-list li { padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,0.04); }
.metric-list li:last-child { border-bottom: none; }
.m-row { display: flex; justify-content: space-between; align-items: baseline; }
.m-val { font-size: 14px; font-weight: 500; }
.m-unit { color: var(--text-tertiary); font-size: 11px; font-weight: 400; margin-left: 2px; }
.sparkline { width: 100%; height: 36px; }
.sparkline path {
  fill: none;
  stroke: var(--info);
  stroke-width: 1.5;
  stroke-linejoin: round;
  stroke-linecap: round;
}

/* Config card */
.config { margin-top: 0; }
.cfg-grid {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 12px 16px;
  align-items: baseline;
}
.cfg-grid li { display: contents; }
.cfg-grid li > .caps { padding-top: 2px; }
.cfg-grid .mono { font-size: 12px; word-break: break-all; }
.ssh-line { display: inline-flex; align-items: center; gap: 8px; }
.tags { display: flex; gap: 4px; flex-wrap: wrap; }
.tag {
  background: rgba(255,255,255,0.04);
  padding: 2px 6px;
  border-radius: 2px;
  font-size: 11px;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

/* Logs --------------------------------------------- */
.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.search-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--surface-elevated);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 2px;
  padding: 0 8px;
  height: 28px;
  width: 280px;
}
.search-wrap svg { color: var(--text-tertiary); }
.search-wrap input {
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
  flex: 1;
  font-family: inherit;
}

.seg {
  display: inline-flex;
  background: var(--surface-elevated);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 2px;
  overflow: hidden;
}
.seg button {
  background: none;
  border: none;
  color: var(--text-secondary);
  padding: 0 10px;
  height: 28px;
  font-size: 11px;
  cursor: pointer;
  font-family: inherit;
}
.seg button.active { background: rgba(255,255,255,0.08); color: var(--text-primary); }

.spacer { flex: 1; }

.btn-sm {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  font-size: 11px;
  background: rgba(255,255,255,0.04);
  color: var(--text-primary);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 2px;
  cursor: pointer;
  font-family: inherit;
}
.btn-sm:hover { background: rgba(255,255,255,0.08); }

.log-pane {
  background: #0A0A0A;
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 4px;
  height: 540px;
  overflow-y: auto;
  padding: 8px 12px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.log-line {
  margin: 0;
  padding: 1px 0;
  font-size: 12px;
  line-height: 1.55;
  display: flex;
  gap: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.log-ts { color: var(--text-tertiary); flex-shrink: 0; }
.log-stream { color: var(--text-tertiary); flex-shrink: 0; width: 48px; }
.log-line.stream-stderr .log-stream { color: #F5A524; }
.log-line.stream-stderr .log-body   { color: #F8D58F; }
.log-body { color: var(--text-secondary); }

.log-foot {
  display: flex;
  align-items: center;
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-tertiary);
}
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-tertiary);
  margin: 0 6px 0 8px;
}
.status-dot.live { background: #19C37D; box-shadow: 0 0 0 3px rgba(25,195,125,0.25); }

/* Snapshots ---------------------------------------- */
.snap-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 12px;
}
.snap-head h3 { margin: 0; font-size: 14px; font-weight: 600; }
.snap-head p { margin: 4px 0 0; }

.snap-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface-elevated);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 4px;
  overflow: hidden;
  font-size: 12px;
}
.snap-table th {
  text-align: left;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255,255,255,0.08);
}
.snap-table td {
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255,255,255,0.04);
}
.snap-table tr:last-child td { border-bottom: none; }
.snap-table tr:hover { background: rgba(255,255,255,0.02); }
.origin-pill {
  display: inline-block;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(255,255,255,0.06);
  color: var(--text-secondary);
}
.origin-pill.auto { background: rgba(74,144,226,0.12); color: var(--info); }
.action-cell { display: flex; gap: 8px; justify-content: flex-end; }
.link-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
  font-family: inherit;
  padding: 2px 4px;
}
.link-btn:hover { color: var(--text-primary); }
.link-btn.danger:hover { color: #EF4444; }

.snap-policy {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  background: rgba(74,144,226,0.08);
  border-left: 2px solid var(--info);
  font-size: 12px;
  color: var(--text-secondary);
}
.snap-policy svg { color: var(--info); flex-shrink: 0; }

/* Events ------------------------------------------- */
.timeline {
  list-style: none;
  padding: 0;
  margin: 0;
  position: relative;
}
.timeline::before {
  content: '';
  position: absolute;
  left: 7px;
  top: 12px;
  bottom: 12px;
  width: 1px;
  background: rgba(255,255,255,0.08);
}
.timeline li {
  position: relative;
  padding: 12px 12px 12px 32px;
  border-bottom: 1px solid rgba(255,255,255,0.04);
}
.timeline li:last-child { border-bottom: none; }
.t-dot {
  position: absolute;
  left: 0;
  top: 18px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--surface-elevated);
  border: 2px solid var(--text-tertiary);
}
.level-success .t-dot { border-color: #19C37D; background: rgba(25,195,125,0.15); }
.level-warn    .t-dot { border-color: #F5A524; background: rgba(245,158,11,0.15); }
.level-info    .t-dot { border-color: var(--info); background: rgba(74,144,226,0.15); }
.t-head { display: flex; gap: 12px; align-items: baseline; margin-bottom: 2px; }
.t-type {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.t-ts { font-size: 11px; color: var(--text-tertiary); }
.timeline p { margin: 0; font-size: 13px; color: var(--text-secondary); }
.t-actor { font-size: 11px; color: var(--text-tertiary); margin-top: 4px; display: inline-block; }

/* Cost --------------------------------------------- */
.cost-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}
@media (max-width: 880px) {
  .cost-grid { grid-template-columns: 1fr 1fr; }
}
.kpi-card { display: flex; flex-direction: column; gap: 4px; }
.kpi-big { font-size: 24px; font-weight: 600; }
.kpi-sub { font-size: 11px; color: var(--text-tertiary); }

.cost-chart {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 180px;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255,255,255,0.08);
}
.cost-bar {
  flex: 1;
  background: var(--info);
  min-height: 4px;
  border-radius: 1px;
}
.cost-bar:hover { background: #6BA6E8; }
.cost-axis {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 10px;
  color: var(--text-tertiary);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.breakdown { margin-top: 16px; }
.breakdown-list { list-style: none; padding: 0; margin: 0; }
.breakdown-list li {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255,255,255,0.04);
  font-size: 13px;
}
.breakdown-list li.sep { padding: 0; border: none; }
.breakdown-list li.total {
  border-top: 1px solid rgba(255,255,255,0.08);
  border-bottom: none;
  font-weight: 600;
  margin-top: 4px;
  padding-top: 12px;
}

/* Modal -------------------------------------------- */
.modal-scrim {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(4px);
}
.modal {
  background: var(--surface-overlay);
  border: 1px solid rgba(255,255,255,0.16);
  border-radius: 6px;
  padding: 20px;
  width: 440px;
  max-width: calc(100vw - 32px);
}
.modal header { display: flex; gap: 8px; align-items: center; color: #EF4444; margin-bottom: 12px; }
.modal h3 { margin: 0; font-size: 16px; font-weight: 600; color: var(--text-primary); }
.modal p { margin: 0 0 16px; font-size: 13px; color: var(--text-secondary); line-height: 1.55; }
.modal label { display: block; margin-bottom: 16px; }
.modal label > .caps { display: block; margin-bottom: 6px; }
.modal input {
  width: 100%;
  background: rgba(0,0,0,0.4);
  border: 1px solid rgba(255,255,255,0.16);
  border-radius: 2px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--text-primary);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.modal input:focus { outline: none; border-color: #EF4444; }
.modal footer { display: flex; justify-content: flex-end; gap: 8px; }

.fade-enter-from, .fade-leave-to { opacity: 0; }
.fade-enter-active, .fade-leave-active { transition: opacity 120ms ease; }

@media (max-width: 560px) {
  .cost-grid { grid-template-columns: 1fr; }
}
</style>
