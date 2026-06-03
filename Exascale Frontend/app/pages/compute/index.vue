<script setup lang="ts">
/**
 * /compute — GPU instances list (engineer-facing).
 *
 * Renders inside the `app` layout. CLI is the primary interface; this
 * page is for monitoring and quick actions. Dark, terminal-adjacent.
 */

definePageMeta({ layout: 'app' })
useHead({ title: 'Compute · Instances — Exascale' })

// =====================================================
// Types
// =====================================================
type InstanceStatus = 'running' | 'provisioning' | 'stopping' | 'stopped'
type GpuType = 'H100' | 'H200'

interface Instance {
  id: string
  name: string
  status: InstanceStatus
  gpu: GpuType
  gpuFull: string
  count: number
  region: string
  startedAt: number         // epoch seconds; null-equivalent (0) when stopped
  uptimeSec: number
  hourlyRate: number        // USD per hour total for the whole instance
  costSoFar: number
  // Specs (used on expansion)
  ram: string
  storage: string
  network: string
  os: string
  image: string
  sshHost: string
  sshKey: string
  // Live GPU utilization (% per GPU)
  util: number[]
}

// =====================================================
// Mock instances — 5 rows per spec
// =====================================================
const NOW = Math.floor(Date.now() / 1000)

const instances = reactive<Instance[]>([
  {
    id: 'i-7a3c91',
    name: 'training-run-2026-05-19',
    status: 'running',
    gpu: 'H100',
    gpuFull: 'H100 80GB SXM5',
    count: 8,
    region: 'us-east-1',
    startedAt: NOW - (4 * 3600 + 23 * 60 + 12),   // 4h 23m 12s
    uptimeSec: 4 * 3600 + 23 * 60 + 12,
    hourlyRate: 8 * 2.99,                          // $23.92/hr
    costSoFar: 0,                                  // recomputed on mount
    ram: '1.92 TB DDR5',
    storage: '16 TB NVMe (8 × 2 TB · RAID 0)',
    network: '400 Gbps InfiniBand NDR',
    os: 'Ubuntu 22.04 LTS',
    image: 'exascale-pytorch-2.4-cuda-12.4',
    sshHost: 'training-run-2026-05-19.tyo1.exascale.com',
    sshKey: 'SHA256:ZxVZ7p9k…7Qd2',
    util: [88, 92, 85, 91, 89, 94, 87, 90],
  },
  {
    id: 'i-4b8821',
    name: 'inference-test-llama3',
    status: 'running',
    gpu: 'H100',
    gpuFull: 'H100 80GB SXM5',
    count: 1,
    region: 'us-east-1',
    startedAt: NOW - (18 * 3600 + 4 * 60),
    uptimeSec: 18 * 3600 + 4 * 60,
    hourlyRate: 2.99,
    costSoFar: 0,
    ram: '256 GB',
    storage: '2 TB NVMe',
    network: '100 Gbps Ethernet',
    os: 'Ubuntu 22.04 LTS',
    image: 'exascale-vllm-0.5',
    sshHost: 'inference-test-llama3.tyo1.exascale.com',
    sshKey: 'SHA256:Lp4hF2x…aB91',
    util: [62],
  },
  {
    id: 'i-12fa55',
    name: 'fine-tune-mistral-7b',
    status: 'running',
    gpu: 'H200',
    gpuFull: 'H200 141GB SXM5',
    count: 1,
    region: 'eu-west-1',
    startedAt: NOW - (2 * 86400 + 6 * 3600 + 41 * 60),
    uptimeSec: 2 * 86400 + 6 * 3600 + 41 * 60,
    hourlyRate: 3.49,
    costSoFar: 0,
    ram: '256 GB',
    storage: '4 TB NVMe',
    network: '200 Gbps InfiniBand HDR',
    os: 'Ubuntu 22.04 LTS',
    image: 'exascale-axolotl-0.4',
    sshHost: 'fine-tune-mistral-7b.dub1.exascale.com',
    sshKey: 'SHA256:9eR8mW1…CC04',
    util: [77],
  },
  {
    id: 'i-9e0d11',
    name: 'mlperf-bench-h100',
    status: 'provisioning',
    gpu: 'H100',
    gpuFull: 'H100 80GB SXM5',
    count: 1,
    region: 'us-east-1',
    startedAt: 0,
    uptimeSec: 0,
    hourlyRate: 2.99,
    costSoFar: 0,
    ram: '256 GB',
    storage: '2 TB NVMe',
    network: '100 Gbps Ethernet',
    os: 'Ubuntu 22.04 LTS',
    image: 'exascale-mlperf-4.1',
    sshHost: '— pending provision',
    sshKey: '—',
    util: [0],
  },
  {
    id: 'i-stp-001',
    name: 'nightly-eval-2026-05-18',
    status: 'stopped',
    gpu: 'H100',
    gpuFull: 'H100 80GB SXM5',
    count: 2,
    region: 'us-east-1',
    startedAt: 0,
    uptimeSec: 1 * 3600 + 12 * 60,                 // ran for this much yesterday
    hourlyRate: 5.98,
    costSoFar: 7.18,
    ram: '512 GB',
    storage: '4 TB NVMe',
    network: '200 Gbps InfiniBand HDR',
    os: 'Ubuntu 22.04 LTS',
    image: 'exascale-pytorch-2.4-cuda-12.4',
    sshHost: '— instance stopped',
    sshKey: '—',
    util: [0, 0],
  },
])

// =====================================================
// Recompute live cost from uptime
// =====================================================
function recomputeCostNow(it: Instance) {
  if (it.status === 'running') {
    it.costSoFar = (it.uptimeSec / 3600) * it.hourlyRate
  }
}
for (const it of instances) recomputeCostNow(it)

// =====================================================
// Resource summary (right sidebar)
// =====================================================
const monthBudget = 15_000
const monthCostBase = 1_847.32                     // base; live spend tracks above

const runningGpuCount = computed(() =>
  instances.filter(i => i.status === 'running').reduce((s, i) => s + i.count, 0),
)
const provisioningGpuCount = computed(() =>
  instances.filter(i => i.status === 'provisioning').reduce((s, i) => s + i.count, 0),
)
const activeGpuCount = computed(() => runningGpuCount.value + provisioningGpuCount.value)

const liveSpendDelta = ref(0)
const monthCost = computed(() => monthCostBase + liveSpendDelta.value)
const budgetAvailable = computed(() => monthBudget - monthCost.value)
const budgetPct = computed(() => Math.min(100, (monthCost.value / monthBudget) * 100))
const budgetTone = computed<'pos' | 'warn' | 'neg'>(() => {
  const p = budgetPct.value
  if (p >= 95) return 'neg'
  if (p >= 80) return 'warn'
  return 'pos'
})

// =====================================================
// Formatters
// =====================================================
function fmtUsd(n: number, dp = 2) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtUsdShort(n: number) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
function fmtUptime(sec: number) {
  if (sec === 0) return '—'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${String(m).padStart(2, '0')}m`
  if (h > 0) return `${h}h ${String(m).padStart(2, '0')}m`
  return `${m}m`
}
function statusLabel(s: InstanceStatus) {
  if (s === 'running')      return 'Running'
  if (s === 'provisioning') return 'Provisioning'
  if (s === 'stopping')     return 'Stopping'
  return 'Stopped'
}

// =====================================================
// Expand / collapse rows
// =====================================================
const expandedIds = ref<Set<string>>(new Set(['i-7a3c91']))  // first row pre-expanded
function toggleRow(id: string) {
  const next = new Set(expandedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedIds.value = next
}
function isExpanded(id: string) {
  return expandedIds.value.has(id)
}

// =====================================================
// Stop / SSH actions
// =====================================================
async function stopInstance(it: Instance) {
  if (it.status !== 'running') return
  it.status = 'stopping'                       // optimistic
  try {
    await compute.stop(it.id)                  // live: releases GPUs, ends billing
  } catch {
    it.status = 'running'                      // revert on failure
    return
  }
  await loadLive()
}
function copySsh(it: Instance) {
  if (typeof navigator !== 'undefined') {
    navigator.clipboard?.writeText(`ssh ubuntu@${it.sshHost}`).catch(() => {})
  }
}

// =====================================================
// CLI copy
// =====================================================
const cliInline = 'exascale gpu create --type h100 --count 8'
async function copyCli() {
  if (typeof navigator !== 'undefined') {
    await navigator.clipboard?.writeText(cliInline).catch(() => {})
  }
}

// =====================================================
// Logs (terminal preview for the expanded instance)
// =====================================================
const LOG_LINES: Record<string, string[]> = {
  'i-7a3c91': [
    '[14:08:42] exascale-runner v2.4.1 · 8 × H100 80GB · CUDA 12.4',
    '[14:08:43] cgroups attached · NVIDIA driver 550.54 · NCCL 2.21.5',
    '[14:08:46] pulling image exascale-pytorch-2.4-cuda-12.4 (3.7 GB) …',
    '[14:09:18] image ready · launching torchrun --nproc_per_node=8',
    '[14:09:22] [rank 0] init_process_group · world_size=8 · backend=nccl',
    '[14:09:24] [rank 0] dataset · /mnt/ds/c4-en.parquet · 364.2M rows',
    '[14:09:31] [rank 0] model loaded · 7.24B params · bf16',
    '[14:09:35] [rank 0] step      0 · loss 5.482 · lr 1.0e-5 · 2.1 tok/s/gpu',
    '[14:11:08] [rank 0] step    100 · loss 3.946 · lr 2.0e-5 · 18,420 tok/s/gpu',
    '[14:14:42] [rank 0] step    500 · loss 2.873 · lr 5.0e-5 · 19,840 tok/s/gpu',
    '[14:22:11] [rank 0] step  1,000 · loss 2.401 · lr 1.0e-4 · 20,156 tok/s/gpu',
    '[14:48:30] [rank 0] step  5,000 · loss 1.847 · lr 1.0e-4 · 20,318 tok/s/gpu',
    '[15:32:55] [rank 0] step 10,000 · loss 1.541 · lr 8.7e-5 · 20,402 tok/s/gpu',
    '[16:18:21] [rank 0] step 15,000 · loss 1.382 · lr 7.4e-5 · 20,388 tok/s/gpu',
    '[17:04:08] [rank 0] step 20,000 · loss 1.276 · lr 6.1e-5 · 20,414 tok/s/gpu',
    '[17:48:39] [rank 0] step 25,000 · loss 1.198 · lr 4.8e-5 · 20,422 tok/s/gpu',
    '[18:02:12] [rank 0] checkpoint · /mnt/ckpt/step-25000 · 13.4 GB',
    '[18:02:48] [rank 0] resume · step 25,000 · 21,940 tok/s/gpu (peak)',
    '[18:18:55] [rank 0] step 27,500 · loss 1.142 · lr 3.5e-5 · 20,418 tok/s/gpu',
    '[18:31:22] [rank 0] step 28,915 · loss 1.118 · lr 3.1e-5 · running ──',
  ],
  'i-4b8821': [
    '[20:11:02] vllm-server starting · model meta-llama/Meta-Llama-3-8B-Instruct',
    '[20:11:04] cuda kernels compiled · 1 × H100 80GB',
    '[20:11:09] model loaded · 8.03B params · bf16 · 17.4 GB resident',
    '[20:11:10] HTTP server listening on :8000 · OpenAI-compatible',
    '[20:11:42] POST /v1/chat/completions · 412 tokens · 245ms',
    '[20:12:14] POST /v1/chat/completions · 188 tokens · 110ms',
    '[20:13:22] POST /v1/chat/completions · 1,084 tokens · 678ms',
    '[ … 1,847 more requests · 99.6% under 500ms · throughput 142 RPM ]',
  ],
}

// =====================================================
// Live ticks — every 3s drift util + add uptime + cost
// =====================================================
let liveTimer: ReturnType<typeof setInterval> | null = null
let provisionTimer: ReturnType<typeof setTimeout> | null = null

function tickLive() {
  for (const it of instances) {
    if (it.status === 'running') {
      it.uptimeSec += 3
      recomputeCostNow(it)
      // drift utilization slightly
      for (let i = 0; i < it.util.length; i++) {
        const target = it.gpu === 'H200' ? 78 : 90
        const drift = (target - it.util[i]!) * 0.06
        const noise = (Math.random() - 0.5) * 4
        it.util[i] = Math.max(35, Math.min(99, it.util[i]! + drift + noise))
      }
      // bump live monthly spend
      liveSpendDelta.value += (it.hourlyRate / 3600) * 3
    }
  }
}

// =====================================================
// Live data (F13) — replace the SSR placeholder rows with real instances from compute-control.
// =====================================================
const compute = useCompute()
const HOURLY: Record<string, number> = { gpu_h100: 2.99, gpu_h200: 3.49 }

/** mapLive converts a compute-control instance into this page's richer row shape. Fields the API
 *  doesn't report (per-GPU util, RAM/storage specs) render as placeholders — never invented values. */
function mapLive(i: import('~/composables/useCompute').Instance): Instance {
  const gpu: GpuType = i.gpu_type === 'gpu_h200' ? 'H200' : 'H100'
  const status: InstanceStatus =
    i.state === 'running' ? 'running'
    : i.state === 'stopping' ? 'stopping'
    : (i.state === 'provisioning' || i.state === 'starting') ? 'provisioning'
    : 'stopped'
  const startedAt = i.started_at ? Math.floor(new Date(i.started_at).getTime() / 1000) : 0
  const uptimeSec = status === 'running' && startedAt ? Math.max(0, Math.floor(Date.now() / 1000) - startedAt) : 0
  const hourlyRate = (HOURLY[i.gpu_type] ?? 0) * i.count
  const sshHost = i.connect?.ssh ? i.connect.ssh.replace(/^ssh\s+\w+@/, '') : '— instance stopped'
  return {
    id: i.id, name: i.id, status, gpu, gpuFull: gpu === 'H200' ? 'H200 141GB SXM5' : 'H100 80GB SXM5',
    count: i.count, region: i.region, startedAt, uptimeSec, hourlyRate,
    costSoFar: (uptimeSec / 3600) * hourlyRate,
    ram: '—', storage: '—', network: '—', os: 'Ubuntu 22.04 LTS', image: i.image,
    sshHost, sshKey: '—', util: Array.from({ length: i.count }, () => 0),
  }
}

/** loadLive fetches non-terminated instances and replaces the table rows in place (keeps reactivity). */
async function loadLive() {
  try {
    const live = await compute.loadInstances()
    const rows = live.filter(i => i.state !== 'terminated').map(mapLive)
    instances.splice(0, instances.length, ...rows)
  } catch {
    // leave the current rows on a transient error; the empty state covers a truly empty list
  }
}

onMounted(() => {
  loadLive()
  liveTimer = setInterval(tickLive, 3000)
})
onBeforeUnmount(() => {
  if (liveTimer) clearInterval(liveTimer)
  if (provisionTimer) clearTimeout(provisionTimer)
})

// =====================================================
// Sort / filter
// =====================================================
type SortKey = 'name' | 'status' | 'gpu' | 'count' | 'region' | 'uptimeSec' | 'costSoFar' | 'hourlyRate'
const sortKey = ref<SortKey>('uptimeSec')
const sortDir = ref<'asc' | 'desc'>('desc')
const filter = ref<'all' | 'running' | 'stopped'>('all')
const FILTERS: Array<'all' | 'running' | 'stopped'> = ['all', 'running', 'stopped']

function setSort(k: SortKey) {
  if (sortKey.value === k) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = k; sortDir.value = 'desc' }
}
const sortedInstances = computed(() => {
  const rows = instances.filter(i => {
    if (filter.value === 'all') return true
    if (filter.value === 'running') return i.status === 'running' || i.status === 'provisioning' || i.status === 'stopping'
    return i.status === 'stopped'
  })
  const k = sortKey.value
  return rows.slice().sort((a, b) => {
    const va = (a[k] as number | string)
    const vb = (b[k] as number | string)
    if (typeof va === 'string') {
      return sortDir.value === 'asc' ? va.localeCompare(vb as string) : (vb as string).localeCompare(va)
    }
    return sortDir.value === 'asc' ? (va as number) - (vb as number) : (vb as number) - (va as number)
  })
})

const sortIndicator = (k: SortKey) => {
  if (sortKey.value !== k) return '↕'
  return sortDir.value === 'asc' ? '↑' : '↓'
}

// Reservation card
const RESERVATION = {
  gpus: 8,
  type: 'H100 80GB SXM5',
  term: '1-year',
  endsAt: '2027-02-15',
  rate: 1.79,  // discounted reserved rate
  utilization: 96,
}
</script>

<template>
  <div class="compute-page">
    <!-- Sub-topbar -->
    <div class="subbar">
      <nav class="crumbs">
        <span>Account</span>
        <span class="sep">›</span>
        <span class="strong">Compute</span>
        <span class="sep">›</span>
        <span class="strong">Instances</span>
      </nav>
      <div class="subbar-right">
        <span class="region-pill">
          <span class="dot pos" /> us-east-1 + 2 others
        </span>
        <span class="cli-inline" @click="copyCli" title="Click to copy">
          <span class="cli-prompt">$</span>
          <span class="cli-cmd">{{ cliInline }}</span>
          <span class="cli-copy">⌘C</span>
        </span>
      </div>
    </div>

    <main class="page">
      <!-- ============ Page header ============ -->
      <section class="page-head">
        <div>
          <div class="eyebrow"><span class="dot" /> Compute · GPU rentals</div>
          <h1 class="page-title">Compute Instances</h1>
          <p class="page-sub">
            GPU rentals for training, fine-tuning, and custom workloads. The CLI is the source of truth — this view is for monitoring and quick actions.
          </p>
        </div>
        <div class="head-actions">
          <span class="cli-block">
            <span class="cli-k">via CLI</span>
            <code>{{ cliInline }}</code>
          </span>
          <NuxtLink to="/compute/new" class="btn primary">
            Provision new
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
        </div>
      </section>

      <div class="layout">
        <!-- ============ Main column ============ -->
        <div class="main-col">
          <!-- Toolbar -->
          <div class="toolbar">
            <div class="filter-row">
              <button
                v-for="f in FILTERS"
                :key="f"
                type="button"
                class="filter-chip"
                :class="{ active: filter === f }"
                @click="filter = f"
              >
                {{ f === 'all' ? 'All' : f === 'running' ? 'Active' : 'Stopped' }}
                <span class="chip-count">
                  {{ f === 'all'
                    ? instances.length
                    : f === 'running'
                      ? instances.filter(i => i.status !== 'stopped').length
                      : instances.filter(i => i.status === 'stopped').length
                  }}
                </span>
              </button>
            </div>
            <div class="toolbar-right">
              <span class="live-tag">
                <span class="pulse" />
                LIVE · uptime + cost tick every 3s
              </span>
            </div>
          </div>

          <!-- Instances table -->
          <div class="table-wrap">
            <table class="instances">
              <thead>
                <tr>
                  <th class="caret-th" />
                  <th class="left" @click="setSort('name')">
                    Name <span class="sort">{{ sortIndicator('name') }}</span>
                  </th>
                  <th class="left" @click="setSort('status')">
                    Status <span class="sort">{{ sortIndicator('status') }}</span>
                  </th>
                  <th class="left" @click="setSort('gpu')">
                    Type <span class="sort">{{ sortIndicator('gpu') }}</span>
                  </th>
                  <th class="num" @click="setSort('count')">
                    GPUs <span class="sort">{{ sortIndicator('count') }}</span>
                  </th>
                  <th class="left" @click="setSort('region')">
                    Region <span class="sort">{{ sortIndicator('region') }}</span>
                  </th>
                  <th class="num" @click="setSort('uptimeSec')">
                    Uptime <span class="sort">{{ sortIndicator('uptimeSec') }}</span>
                  </th>
                  <th class="num" @click="setSort('costSoFar')">
                    Cost so far <span class="sort">{{ sortIndicator('costSoFar') }}</span>
                  </th>
                  <th class="num" @click="setSort('hourlyRate')">
                    $/hr <span class="sort">{{ sortIndicator('hourlyRate') }}</span>
                  </th>
                  <th class="right-th">Actions</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="it in sortedInstances" :key="it.id">
                  <tr class="row" :class="{ open: isExpanded(it.id), stopped: it.status === 'stopped' }" @click="toggleRow(it.id)">
                    <td class="caret-td">
                      <span class="caret">
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                          <path d="M5 4l4 4-4 4" />
                        </svg>
                      </span>
                    </td>
                    <td class="left name-cell">
                      <div class="name-wrap">
                        <span class="iname mono">{{ it.name }}</span>
                        <span class="iid">{{ it.id }}</span>
                      </div>
                    </td>
                    <td class="left">
                      <span class="status-tag" :class="it.status">
                        <span class="dot" />
                        {{ statusLabel(it.status) }}
                      </span>
                    </td>
                    <td class="left mono">{{ it.gpuFull }}</td>
                    <td class="num mono">{{ it.count }}</td>
                    <td class="left mono dim">{{ it.region }}</td>
                    <td class="num mono">
                      <span v-if="it.status === 'provisioning'" class="dim">—</span>
                      <template v-else>{{ fmtUptime(it.uptimeSec) }}</template>
                    </td>
                    <td class="num mono">
                      <span v-if="it.status === 'provisioning'" class="dim">—</span>
                      <template v-else>{{ fmtUsd(it.costSoFar) }}</template>
                    </td>
                    <td class="num mono">
                      {{ fmtUsd(it.hourlyRate) }}<span class="dim">/hr</span>
                    </td>
                    <td class="right-td">
                      <div class="row-actions" @click.stop>
                        <button
                          type="button"
                          class="btn-mini"
                          :disabled="it.status !== 'running'"
                          @click="copySsh(it)"
                        >
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                            <path d="M3 5h10v6H3zM6 11v3h4v-3" />
                          </svg>
                          SSH
                        </button>
                        <button
                          type="button"
                          class="btn-mini stop"
                          :disabled="it.status === 'stopped' || it.status === 'stopping'"
                          @click="stopInstance(it)"
                        >
                          <svg viewBox="0 0 16 16" fill="currentColor"><rect x="4" y="4" width="8" height="8" /></svg>
                          {{ it.status === 'stopping' ? 'Stopping…' : 'Stop' }}
                        </button>
                        <button type="button" class="btn-mini">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                            <path d="M3 4h10M3 8h10M3 12h7" />
                          </svg>
                          Logs
                        </button>
                      </div>
                    </td>
                  </tr>

                  <!-- ============ Expanded detail row ============ -->
                  <tr v-if="isExpanded(it.id)" class="row-expand">
                    <td colspan="10">
                      <div class="expand-grid">
                        <!-- Specs -->
                        <div class="ex-card">
                          <div class="ex-head">— Specs</div>
                          <dl class="spec-dl">
                            <dt>GPU</dt><dd>{{ it.count }} × {{ it.gpuFull }}</dd>
                            <dt>RAM</dt><dd>{{ it.ram }}</dd>
                            <dt>Storage</dt><dd>{{ it.storage }}</dd>
                            <dt>Network</dt><dd>{{ it.network }}</dd>
                            <dt>OS</dt><dd>{{ it.os }}</dd>
                            <dt>Image</dt><dd>{{ it.image }}</dd>
                          </dl>
                        </div>

                        <!-- SSH -->
                        <div class="ex-card">
                          <div class="ex-head">
                            — SSH access
                            <button type="button" class="ex-head-btn" @click="copySsh(it)" :disabled="it.status !== 'running'">
                              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                              Copy
                            </button>
                          </div>
                          <code class="ssh-cmd">
                            <span class="cli-prompt">$</span>
                            ssh ubuntu@{{ it.sshHost }}
                          </code>
                          <dl class="spec-dl tight">
                            <dt>Key fingerprint</dt><dd>{{ it.sshKey }}</dd>
                            <dt>Port</dt><dd>22 (default)</dd>
                            <dt>User</dt><dd>ubuntu</dd>
                          </dl>
                        </div>

                        <!-- Per-GPU utilization -->
                        <div class="ex-card util-card">
                          <div class="ex-head">— Per-GPU utilization</div>
                          <div class="util-grid" :style="{ '--cols': it.util.length > 4 ? 8 : it.util.length }">
                            <div
                              v-for="(u, i) in it.util"
                              :key="i"
                              class="util-cell"
                              :class="{ idle: u < 30 }"
                            >
                              <div class="util-bar-wrap">
                                <div class="util-bar" :style="{ height: u + '%' }" />
                              </div>
                              <span class="util-pct mono">{{ Math.round(u) }}%</span>
                              <span class="util-name mono">gpu{{ i }}</span>
                            </div>
                          </div>
                        </div>

                        <!-- Cost breakdown -->
                        <div class="ex-card">
                          <div class="ex-head">— Cost breakdown</div>
                          <dl class="spec-dl">
                            <dt>Rate</dt><dd>{{ fmtUsd(it.hourlyRate) }} / hr · {{ fmtUsd(it.hourlyRate / it.count) }} per GPU</dd>
                            <dt>Uptime</dt><dd>{{ it.status === 'provisioning' ? '— pending' : fmtUptime(it.uptimeSec) }}</dd>
                            <dt>Cost so far</dt><dd class="emph">{{ fmtUsd(it.costSoFar) }}</dd>
                            <dt>Projected · 24h</dt><dd>{{ fmtUsd(it.hourlyRate * 24) }}</dd>
                            <dt>Projected · 30d</dt><dd>{{ fmtUsd(it.hourlyRate * 24 * 30) }}</dd>
                          </dl>
                        </div>

                        <!-- Logs (full width below) -->
                        <div class="ex-card logs-card">
                          <div class="ex-head">
                            — Logs · tail -20
                            <span class="head-meta">
                              <span class="pulse" />
                              streaming · stdout merged
                            </span>
                          </div>
                          <pre class="logs"><span v-for="(line, idx) in (LOG_LINES[it.id] || ['[ — no logs · instance ' + it.status + ' ]'])" :key="idx" class="log-line">{{ line }}</span></pre>
                          <div class="logs-foot">
                            <a href="#">Open full log stream →</a>
                            <span class="dim">log-id <span class="mono">{{ it.id }}.stdout.log</span></span>
                          </div>
                        </div>
                      </div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>

          <!-- ============ Empty / launch CTA card ============ -->
          <section class="empty-card">
            <div class="empty-mark" aria-hidden="true">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <rect x="2.5" y="3.5" width="11" height="9" />
                <path d="M2.5 7h11M5 10.5h1M7.5 10.5h1" />
              </svg>
            </div>
            <div class="empty-body">
              <div class="empty-title">Need more capacity? Spin one up.</div>
              <div class="empty-sub">CLI is fastest and gives you full flag coverage. Web takes you through a guided wizard.</div>
            </div>
            <div class="empty-paths">
              <NuxtLink to="/compute/new" class="btn secondary">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                  <rect x="2.5" y="3.5" width="11" height="9" />
                  <path d="M2.5 7h11" />
                </svg>
                Via web
              </NuxtLink>
              <span class="empty-or">or</span>
              <code class="empty-cli">
                <span class="cli-prompt">$</span>
                <span>{{ cliInline }}</span>
                <button type="button" class="empty-copy" @click="copyCli" title="Copy">⌘C</button>
              </code>
            </div>
          </section>
        </div>

        <!-- ============ Resource sidebar ============ -->
        <aside class="side">
          <div class="side-card">
            <div class="side-head">
              <span class="eyebrow"><span class="dot" /> Current usage</span>
            </div>
            <div class="usage-row">
              <div class="usage-num mono">{{ activeGpuCount }}</div>
              <div class="usage-meta">
                <span class="usage-unit">GPUs</span>
                <span class="usage-sub mono">
                  {{ runningGpuCount }} running · {{ provisioningGpuCount }} provisioning
                </span>
              </div>
            </div>
            <div class="usage-detail">
              <span class="status-dot run" /> {{ runningGpuCount }} running ·
              <span class="status-dot prov" /> {{ provisioningGpuCount }} provisioning ·
              <span class="status-dot stop" /> {{ instances.filter(i => i.status === 'stopped').reduce((s, i) => s + i.count, 0) }} stopped
            </div>
          </div>

          <div class="side-card">
            <div class="side-head">
              <span class="eyebrow"><span class="dot" /> This month · cost</span>
              <span class="side-meta">May 2026</span>
            </div>
            <div class="cost-num mono">{{ fmtUsd(monthCost) }}</div>
            <div class="cost-sub">
              <span :class="budgetTone === 'pos' ? 'pos-text' : budgetTone === 'warn' ? 'warn-text' : 'neg-text'">
                {{ budgetPct.toFixed(1) }}%
              </span>
              of {{ fmtUsd(monthBudget) }} budget
            </div>
            <div class="budget-bar">
              <div class="budget-fill" :class="budgetTone" :style="{ width: budgetPct + '%' }" />
            </div>
            <div class="cost-foot">
              <span class="cf-k">Available</span>
              <span class="cf-v mono">{{ fmtUsd(budgetAvailable) }}</span>
            </div>
            <div class="cost-foot">
              <span class="cf-k">Daily burn · 7d avg</span>
              <span class="cf-v mono">{{ fmtUsd(Math.round(monthCost / 21 * 100) / 100) }}</span>
            </div>
            <div class="cost-foot">
              <span class="cf-k">Alerts</span>
              <a href="#" class="cf-link">Email + Slack at 80%</a>
            </div>
          </div>

          <div class="side-card">
            <div class="side-head">
              <span class="eyebrow"><span class="dot" /> Reserved capacity</span>
              <a href="#" class="side-head-link">Manage →</a>
            </div>
            <div class="reserve-num mono">{{ RESERVATION.gpus }} × {{ RESERVATION.type }}</div>
            <div class="reserve-meta mono">
              {{ RESERVATION.term }} · ends {{ RESERVATION.endsAt }}
            </div>
            <div class="reserve-rate">
              <span class="rr-k">Rate</span>
              <span class="rr-v mono">{{ fmtUsd(RESERVATION.rate) }}<span class="dim">/hr</span></span>
              <span class="rr-discount">−40% vs on-demand</span>
            </div>
            <div class="reserve-util">
              <div class="ru-row">
                <span class="ru-k">Utilization · 30d</span>
                <span class="ru-v mono">{{ RESERVATION.utilization }}%</span>
              </div>
              <div class="ru-bar">
                <div class="ru-fill" :style="{ width: RESERVATION.utilization + '%' }" />
              </div>
              <div class="ru-foot dim">≥ 80% target · rolling</div>
            </div>
          </div>

          <div class="side-card help">
            <div class="help-title">Looking for the API?</div>
            <p class="help-body">
              All actions on this page have CLI + REST equivalents. Reach for the CLI when scripting; this UI is for ad-hoc monitoring.
            </p>
            <a href="#" class="help-link">CLI reference →</a>
            <a href="#" class="help-link">REST API · /v1/gpu →</a>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<style scoped>
.compute-page {
  --bd-soft: rgba(255, 255, 255, 0.05);
  --warn-text: var(--warn);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.55;
  color: var(--text);
  font-feature-settings: 'tnum' on, 'ss01' on;
  background: var(--canvas);
  min-height: 100%;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.dim { color: var(--text-3); }
.pos-text { color: var(--pos); }
.warn-text { color: var(--warn); }
.neg-text { color: var(--neg); }

/* ============================================================
   Sub-topbar
   ============================================================ */
.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: var(--canvas);
  gap: 16px;
}
.crumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .strong { color: var(--text); }
.subbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 14px;
}
.region-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
}
.region-pill .dot { width: 5px; height: 5px; border-radius: 50%; background: var(--pos); }

.cli-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--elevated);
  border: 1px solid var(--border);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text);
  letter-spacing: 0.02em;
  cursor: pointer;
}
.cli-inline:hover { background: var(--hover); }
.cli-prompt { color: var(--brand); font-weight: 600; }
.cli-cmd { color: var(--text); }
.cli-copy {
  margin-left: 4px;
  padding: 1px 5px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
  font-size: 9.5px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}

/* ============================================================
   Page
   ============================================================ */
.page {
  max-width: 1440px;
  margin: 0 auto;
  padding: 32px 28px 80px;
}
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}
.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.eyebrow .dot {
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.page-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 30px;
  letter-spacing: -0.022em;
  line-height: 1.1;
  margin: 0 0 8px;
  color: var(--text);
}
.page-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0;
  max-width: 640px;
}

.head-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.cli-block {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  font-family: var(--font-mono);
  font-size: 11.5px;
}
.cli-block .cli-k {
  font-size: 9.5px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.cli-block code {
  color: var(--text);
  font-family: var(--font-mono);
  letter-spacing: 0.02em;
}

.btn {
  height: 38px;
  padding: 0 16px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: background 120ms, border-color 120ms, color 120ms;
}
.btn svg { width: 14px; height: 14px; }
.btn.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
  font-weight: 600;
}
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.secondary {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover { background: var(--hover); border-color: rgba(255, 255, 255, 0.24); }

/* ============================================================
   Layout: main + sidebar
   ============================================================ */
.layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 24px;
  align-items: start;
}

/* ============================================================
   Toolbar (filter chips)
   ============================================================ */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.filter-row {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.filter-chip {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  color: var(--text-2);
  padding: 5px 12px;
  font-family: var(--font-sans);
  font-size: 12px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.filter-chip:last-child { border-right: 0; }
.filter-chip:hover { background: var(--hover); color: var(--text); }
.filter-chip.active {
  background: var(--overlay);
  color: var(--text);
  font-weight: 500;
}
.chip-count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  background: var(--canvas);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-variant-numeric: tabular-nums;
}
.filter-chip.active .chip-count { color: var(--text); }

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 14px;
}
.live-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}
.live-tag .pulse {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}
@keyframes pulse {
  0%   { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0.55); }
  70%  { box-shadow: 0 0 0 6px rgba(25, 195, 125, 0); }
  100% { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0); }
}

/* ============================================================
   Instances table
   ============================================================ */
.table-wrap {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  margin-bottom: 24px;
}
.instances {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.instances thead th {
  text-align: right;
  padding: 9px 12px;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
}
.instances thead th.left { text-align: left; }
.instances thead th.num { text-align: right; }
.instances thead th.right-th { text-align: right; cursor: default; }
.instances thead th.caret-th { width: 32px; cursor: default; padding: 0; }
.instances thead th .sort {
  display: inline-block;
  margin-left: 4px;
  color: var(--text-3);
  font-size: 10px;
}
.instances thead th:hover { color: var(--text-2); }

.instances tbody td {
  padding: 12px;
  border-bottom: 1px solid var(--bd-soft);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
  color: var(--text);
  vertical-align: middle;
}
.instances tbody td.left { text-align: left; font-family: var(--font-sans); }
.instances tbody td.left.mono { font-family: var(--font-mono); font-size: 12px; }
.instances tbody td.right-td { text-align: right; }
.instances tbody td.caret-td {
  padding: 0 0 0 6px;
  width: 32px;
  text-align: center;
}
.instances tbody .row { cursor: pointer; transition: background 90ms; }
.instances tbody .row:hover { background: rgba(255, 255, 255, 0.025); }
.instances tbody .row.open { background: rgba(255, 255, 255, 0.03); }
.instances tbody .row.stopped td { opacity: 0.7; }

.caret {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  color: var(--text-3);
  transition: transform 160ms;
}
.row.open .caret { transform: rotate(90deg); color: var(--text); }
.caret svg { width: 12px; height: 12px; }

.name-cell { font-family: var(--font-mono); }
.name-wrap { display: flex; flex-direction: column; gap: 2px; }
.iname {
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.02em;
}
.iid {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

/* Status tags */
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
  border: 1px solid transparent;
}
.status-tag .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.status-tag.running {
  background: rgba(25, 195, 125, 0.10);
  color: var(--pos);
  border-color: rgba(25, 195, 125, 0.25);
}
.status-tag.running .dot { animation: pulse-dot 2.4s infinite; }
.status-tag.provisioning {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border-color: rgba(74, 144, 226, 0.25);
}
.status-tag.provisioning .dot { animation: pulse-dot 1.4s infinite; }
.status-tag.stopping {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border-color: rgba(245, 158, 11, 0.25);
}
.status-tag.stopped {
  background: transparent;
  color: var(--text-3);
  border-color: var(--border);
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* Row actions */
.row-actions { display: inline-flex; gap: 4px; justify-content: flex-end; }
.btn-mini {
  height: 26px;
  padding: 0 10px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.06em;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  text-transform: uppercase;
  font-weight: 600;
}
.btn-mini:hover { background: var(--hover); color: var(--text); border-color: var(--border-strong); }
.btn-mini:disabled { opacity: 0.45; cursor: not-allowed; }
.btn-mini.stop:hover:enabled {
  background: var(--neg-soft);
  color: var(--neg);
  border-color: var(--neg);
}
.btn-mini svg { width: 11px; height: 11px; }

/* ============================================================
   Expanded detail row
   ============================================================ */
.row-expand td {
  padding: 0;
  background: var(--canvas);
  border-bottom: 1px solid var(--bd-soft);
}
.expand-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  padding: 18px 18px 22px;
}
.expand-grid .ex-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 14px 16px;
}
.expand-grid .ex-card.logs-card,
.expand-grid .ex-card.util-card { grid-column: span 2; }
.expand-grid .ex-card.logs-card { grid-column: span 4; }

.ex-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.ex-head .head-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  letter-spacing: 0.08em;
  color: var(--text-3);
  text-transform: none;
}
.ex-head .head-meta .pulse {
  width: 5px;
  height: 5px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}
.ex-head-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.ex-head-btn:hover { background: var(--hover); color: var(--text); }
.ex-head-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.ex-head-btn svg { width: 11px; height: 11px; }

.spec-dl {
  display: grid;
  grid-template-columns: 110px 1fr;
  gap: 6px 14px;
  margin: 0;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}
.spec-dl.tight {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--bd-soft);
}
.spec-dl dt {
  color: var(--text-3);
  font-size: 10px;
  letter-spacing: 0.10em;
  text-transform: uppercase;
  font-weight: 600;
  align-self: center;
}
.spec-dl dd {
  margin: 0;
  color: var(--text);
  letter-spacing: 0.02em;
}
.spec-dl dd.emph {
  font-size: 14px;
  font-weight: 600;
  color: var(--brand);
}

.ssh-cmd {
  display: block;
  font-family: var(--font-mono);
  font-size: 12.5px;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  color: var(--text);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow-x: auto;
}
.ssh-cmd .cli-prompt { margin-right: 8px; }

/* Per-GPU utilization */
.util-grid {
  display: grid;
  grid-template-columns: repeat(var(--cols, 8), minmax(0, 1fr));
  gap: 6px;
  padding-top: 4px;
}
.util-cell {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 3px;
  min-width: 0;
}
.util-bar-wrap {
  width: 100%;
  height: 64px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 1px;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: flex-end;
}
.util-bar {
  width: 100%;
  background: linear-gradient(180deg, var(--brand), rgba(200, 242, 92, 0.65));
  transition: height 600ms ease-out;
}
.util-cell.idle .util-bar { background: var(--border-strong); }
.util-pct {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text);
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1.1;
  text-align: center;
}
.util-name {
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--text-3);
  letter-spacing: 0.08em;
  text-transform: uppercase;
  line-height: 1.1;
  text-align: center;
}

/* Logs */
.logs {
  margin: 0;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 12px 14px;
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text);
  line-height: 1.5;
  max-height: 320px;
  overflow-y: auto;
  letter-spacing: 0.01em;
  white-space: pre;
}
.log-line {
  display: block;
  white-space: pre;
}
.log-line:last-child {
  color: var(--brand);
}
.logs-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.logs-foot a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
}
.logs-foot a:hover { text-decoration: underline; }

/* ============================================================
   Empty / launch CTA
   ============================================================ */
.empty-card {
  background: var(--elevated);
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 18px 22px;
  display: grid;
  grid-template-columns: 56px 1fr auto;
  gap: 18px;
  align-items: center;
}
.empty-mark {
  width: 56px;
  height: 56px;
  display: grid;
  place-items: center;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-2);
}
.empty-mark svg { width: 22px; height: 22px; }
.empty-title {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
  margin-bottom: 4px;
}
.empty-sub {
  font-family: var(--font-sans);
  font-size: 12.5px;
  color: var(--text-2);
}
.empty-paths {
  display: flex;
  align-items: center;
  gap: 12px;
}
.empty-or {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}
.empty-cli {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text);
  letter-spacing: 0.02em;
}
.empty-copy {
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 9.5px;
  padding: 1px 6px;
  border-radius: 2px;
  cursor: pointer;
  letter-spacing: 0.06em;
}
.empty-copy:hover { color: var(--text); border-color: var(--border-strong); }

/* ============================================================
   Sidebar
   ============================================================ */
.side {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.side-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px 18px;
}
.side-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.side-head .eyebrow { margin: 0; }
.side-head-link {
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  text-decoration: none;
}
.side-head-link:hover { color: var(--text); text-decoration: underline; }
.side-meta {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

/* Usage card */
.usage-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.usage-num {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 36px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
  line-height: 1;
}
.usage-meta { display: flex; flex-direction: column; gap: 2px; }
.usage-unit {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}
.usage-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}
.usage-detail {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin: 0 4px 0 2px;
  vertical-align: 1px;
}
.status-dot.run  { background: var(--pos); }
.status-dot.prov { background: var(--accent); }
.status-dot.stop { background: var(--text-3); }

/* Cost card */
.cost-num {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
  line-height: 1;
  margin-bottom: 4px;
}
.cost-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
  margin-bottom: 10px;
}
.budget-bar {
  height: 6px;
  background: var(--canvas);
  border-radius: 999px;
  overflow: hidden;
  margin-bottom: 12px;
}
.budget-fill {
  height: 100%;
  border-radius: 999px;
  transition: width 400ms ease-out;
}
.budget-fill.pos  { background: linear-gradient(90deg, var(--pos), rgba(25, 195, 125, 0.65)); }
.budget-fill.warn { background: linear-gradient(90deg, var(--warn), rgba(245, 158, 11, 0.65)); }
.budget-fill.neg  { background: linear-gradient(90deg, var(--neg), rgba(239, 68, 68, 0.65)); }
.cost-foot {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding-top: 6px;
  border-top: 1px solid var(--bd-soft);
  margin-top: 6px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.cost-foot:first-of-type { border-top: 0; padding-top: 0; margin-top: 0; }
.cf-k { color: var(--text-3); }
.cf-v { color: var(--text); }
.cf-link {
  color: var(--text-2);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
}
.cf-link:hover { color: var(--text); }

/* Reserved capacity */
.reserve-num {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.005em;
  margin-bottom: 4px;
}
.reserve-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-bottom: 10px;
}
.reserve-rate {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 8px 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 12px;
}
.rr-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.rr-v {
  font-family: var(--font-mono);
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.rr-discount {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--pos);
  margin-left: auto;
  letter-spacing: 0.04em;
}
.reserve-util .ru-row {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  margin-bottom: 4px;
}
.reserve-util .ru-k { color: var(--text-3); }
.reserve-util .ru-v { color: var(--text); font-weight: 600; }
.reserve-util .ru-bar {
  height: 4px;
  background: var(--canvas);
  border-radius: 999px;
  overflow: hidden;
}
.reserve-util .ru-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--brand), rgba(200, 242, 92, 0.65));
}
.reserve-util .ru-foot {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  margin-top: 6px;
}

/* Help card */
.side-card.help { background: rgba(255, 255, 255, 0.02); }
.help-title {
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 13px;
  color: var(--text);
  margin-bottom: 6px;
  letter-spacing: -0.005em;
}
.help-body {
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.55;
  margin: 0 0 10px;
}
.help-link {
  display: block;
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text);
  text-decoration: none;
  padding: 4px 0;
  letter-spacing: 0.02em;
  border-top: 1px solid var(--bd-soft);
}
.help-link:first-of-type { border-top: 0; }
.help-link:hover { color: var(--brand); }

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1200px) {
  .layout { grid-template-columns: 1fr; }
  .side { position: static; flex-direction: row; flex-wrap: wrap; }
  .side-card { flex: 1 1 280px; }
  .expand-grid { grid-template-columns: repeat(2, 1fr); }
  .expand-grid .ex-card.util-card { grid-column: span 2; }
}
@media (max-width: 720px) {
  .page-head { flex-direction: column; align-items: stretch; }
  .head-actions { flex-direction: column; align-items: stretch; }
  .cli-block { justify-content: space-between; }
  .expand-grid { grid-template-columns: 1fr; }
  .expand-grid .ex-card.util-card,
  .expand-grid .ex-card.logs-card { grid-column: span 1; }
  .empty-card { grid-template-columns: 1fr; }
  .empty-paths { flex-direction: column; align-items: stretch; }
}
</style>
