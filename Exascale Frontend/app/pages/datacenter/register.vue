<script setup lang="ts">
/**
 * /datacenter/register — Partner Capacity Registration form.
 *
 * Internal-only in v1; self-serve in v1.5+. Light admin tool surface,
 * max-width 960px. Five numbered sections:
 *   1 Identification · 2 Capacity · 3 Commercial · 4 Verification · 5 Activation
 *
 * Form is pre-populated with one in-flight partner ("Helsinki Data Center Oy")
 * for the screenshot/demo. Verification section runs a live, fake-but-realistic
 * NCCL benchmark progress so the page feels active.
 */

definePageMeta({ layout: false })
useHead({
  title: 'Register partner · Datacenter — Exascale',
  htmlAttrs: { 'data-theme': 'light' },
})

// Supply-side onboarding (F16) isn't built — gate the form behind a "coming soon" screen so partners
// can't sign up into an incomplete flow. Flip to true when the supply service ships.
const datacenterSignupLive = false

// =====================================================
// Pre-populated partner draft — Helsinki DC
// =====================================================
const form = reactive({
  // identification
  legalName: 'Helsinki Data Center Oy',
  tradingAs: 'HDC1',
  techName: 'Jukka Virtanen',
  techEmail: 'jukka.virtanen@hdc.fi',
  techPhone: '+358 9 1234 5678',
  bizName: 'Anna Korhonen',
  bizEmail: 'a.korhonen@hdc.fi',
  bizPhone: '+358 50 987 6543',
  country: 'FI',
  city: 'Helsinki',
  operatorSince: '2024-08-15',

  // capacity
  gpuType: 'h100',
  gpuCount: 512,
  interconnect: 'ib-ndr',
  bandwidth: '400 Gbps',
  topology: 'fat-tree',
  uptime: 99.95,
  maintenanceWindows: 'Tuesdays 02:00–04:00 UTC · Emergency: 24h notice',

  // commercial
  floorPrice: 2.45,
  revenueShare: 70,
  payoutCurrency: 'USD',
  payoutCadence: 'monthly',
  bankName: 'Nordea Bank Abp',
  iban: 'FI21 1234 5678 9012 34',
  swift: 'NDEAFIHHXXX',
  beneficiary: 'Helsinki Data Center Oy',
  beneficiaryAddress: 'Kaivokatu 10, 00100 Helsinki, Finland',

  // activation
  softLaunch: true,
  softLaunchPct: 20,
  softLaunchDuration: 14,
  activationDate: '2026-05-26',
})

const COUNTRIES: Array<[string, string]> = [
  ['FI', 'Finland'], ['SE', 'Sweden'], ['NO', 'Norway'], ['DK', 'Denmark'],
  ['DE', 'Germany'], ['FR', 'France'], ['NL', 'Netherlands'], ['IE', 'Ireland'],
  ['GB', 'United Kingdom'], ['US', 'United States'], ['CA', 'Canada'], ['SG', 'Singapore'],
  ['JP', 'Japan'], ['KR', 'South Korea'], ['CZ', 'Czechia'], ['EE', 'Estonia'],
]

const GPU_OPTIONS = [
  { value: 'h100', label: 'NVIDIA H100 · 80GB SXM5',  baseRate: 2.99 },
  { value: 'h200', label: 'NVIDIA H200 · 141GB SXM5', baseRate: 3.49 },
  { value: 'b200', label: 'NVIDIA B200 · 192GB',      baseRate: 4.95 },
  { value: 'mi300', label: 'AMD MI300X · 192GB',      baseRate: 2.85 },
]
const INTERCONNECTS = [
  { value: 'ib-ndr',   label: 'InfiniBand NDR · 400 Gbps' },
  { value: 'ib-hdr',   label: 'InfiniBand HDR · 200 Gbps' },
  { value: 'rocev2',   label: 'RoCE v2 · 200 Gbps' },
  { value: 'ethernet', label: 'Ethernet · 100 Gbps' },
]
const TOPOLOGIES = ['fat-tree', 'dragonfly+', 'rail-optimized', '3D torus']
const CURRENCIES = ['USD', 'EUR', 'GBP', 'JPY', 'SGD']

// =====================================================
// Derived numbers — soft-launch ramp, revenue projection
// =====================================================
const activeCount = computed(() =>
  form.softLaunch ? Math.round(form.gpuCount * form.softLaunchPct / 100) : form.gpuCount,
)
const monthlyHours = 730  // average month
const projectedMonthlyRevenue = computed(() =>
  activeCount.value * form.floorPrice * monthlyHours,
)
const projectedPartnerShare = computed(() =>
  projectedMonthlyRevenue.value * form.revenueShare / 100,
)
const projectedExascaleFee = computed(() =>
  projectedMonthlyRevenue.value - projectedPartnerShare.value,
)

function fmtUsd(n: number, dp = 0) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtInt(n: number) {
  return Math.round(n).toLocaleString('en-US')
}

// =====================================================
// Verification — live NCCL benchmark progress
// =====================================================
interface VerificationStep {
  id: string
  state: 'done' | 'running' | 'pending' | 'failed'
  title: string
  detail: string
  timestamp: string
  progress?: number    // 0–100 when running
  bench?: string       // running-only sub-detail (live)
}

const verifyStartedAt = '14:04 UTC · 2026-05-21'
const steps = reactive<VerificationStep[]>([
  {
    id: 'reach',
    state: 'done',
    title: 'Network reachability',
    detail: 'TCP/443 + ICMP across 10 endpoints · 12.4ms RTT median · 0 packet loss',
    timestamp: '14:08:42 UTC',
  },
  {
    id: 'smi',
    state: 'done',
    title: 'GPU detection · NVIDIA SMI',
    detail: '512 × H100 80GB SXM5 detected · CUDA 12.4 · driver 550.54 · all healthy',
    timestamp: '14:09:18 UTC',
  },
  {
    id: 'nccl',
    state: 'running',
    title: 'NCCL collective benchmark',
    detail: 'all-reduce + all-gather across all GPUs · target 8.0 TB/s sustained',
    timestamp: '14:11:02 UTC',
    progress: 52,
    bench: '268 / 512 GPUs benchmarked · 8.4 TB/s achieved · ETA 4m 12s',
  },
  {
    id: 'sla',
    state: 'pending',
    title: 'Final SLA test',
    detail: '24-hour burn-in · network jitter + thermal soak · starts after NCCL completes',
    timestamp: '—',
  },
])

const verifyComplete = computed(() => steps.every(s => s.state === 'done'))
const verifyProgress = computed(() => {
  const total = steps.length
  const done = steps.filter(s => s.state === 'done').length
  const running = steps.find(s => s.state === 'running')
  const runningContribution = running ? (running.progress ?? 0) / 100 : 0
  return ((done + runningContribution) / total) * 100
})

let benchTimer: ReturnType<typeof setInterval> | null = null
function tickBenchmark() {
  const nccl = steps.find(s => s.id === 'nccl')
  if (!nccl || nccl.state !== 'running') return
  const cur = nccl.progress ?? 0
  const next = Math.min(100, cur + 0.7 + Math.random() * 1.4)
  nccl.progress = next
  const benched = Math.round((next / 100) * 512)
  const tb = (7.8 + Math.random() * 0.9).toFixed(1)
  const etaSec = Math.max(0, Math.round((100 - next) * 5.2))
  const m = Math.floor(etaSec / 60)
  const s = etaSec % 60
  nccl.bench = `${benched} / 512 GPUs benchmarked · ${tb} TB/s achieved · ETA ${m}m ${String(s).padStart(2, '0')}s`
  if (next >= 100) {
    nccl.state = 'done'
    nccl.timestamp = '14:18:55 UTC'
    nccl.detail = 'all-reduce + all-gather · 8.6 TB/s sustained · 512 GPUs · variance ≤ 1.4%'
    nccl.progress = 100
    nccl.bench = undefined
    // start SLA step
    const sla = steps.find(s => s.id === 'sla')
    if (sla) {
      sla.state = 'running'
      sla.timestamp = '14:19:01 UTC'
      sla.detail = '24-hour burn-in · 23h 59m remaining · jitter 0.8ms · GPU temps nominal'
      sla.progress = 1
    }
  }
}

const formClock = ref('14:13:08 UTC')
function tickClock() {
  const d = new Date()
  formClock.value = d.toLocaleTimeString('en-GB', { hour12: false, timeZone: 'UTC' }) + ' UTC'
}
let clockTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  tickClock()
  clockTimer = setInterval(tickClock, 1000)
  benchTimer = setInterval(tickBenchmark, 1200)
})
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
  if (benchTimer) clearInterval(benchTimer)
})

// =====================================================
// Section-level completion status (heuristic — purely cosmetic here)
// =====================================================
const sections = computed(() => [
  {
    n: '01', id: 'identification', title: 'Partner identification',
    status: (form.legalName && form.techEmail && form.country) ? 'complete' : 'incomplete',
  },
  {
    n: '02', id: 'capacity', title: 'Capacity details',
    status: (form.gpuCount > 0) ? 'complete' : 'incomplete',
  },
  {
    n: '03', id: 'commercial', title: 'Commercial terms',
    status: (form.iban && form.swift) ? 'complete' : 'incomplete',
  },
  {
    n: '04', id: 'verification', title: 'Capacity verification',
    status: verifyComplete.value ? 'complete' : 'in-progress',
  },
  {
    n: '05', id: 'activation', title: 'Activation',
    status: 'pending',
  },
])

function gpuSpec() {
  return GPU_OPTIONS.find(g => g.value === form.gpuType)?.label ?? form.gpuType
}
function interconnectLabel() {
  return INTERCONNECTS.find(i => i.value === form.interconnect)?.label ?? form.interconnect
}
</script>

<template>
  <!-- Datacenter onboarding (supply side, F16) isn't built yet — gate the registration so partners
       can't sign up into an incomplete flow. Flip `datacenterSignupLive` when the supply service ships. -->
  <div v-if="!datacenterSignupLive" class="reg-shell soon-shell" data-theme="light">
    <header class="topbar">
      <NuxtLink to="/" class="brand"><span class="brand-mark" />EXASCALE</NuxtLink>
    </header>
    <main class="soon-main">
      <div class="soon-card">
        <span class="soon-pill">Coming soon</span>
        <h1>Datacenter onboarding isn't open yet</h1>
        <p>We're finishing the supply-side onboarding — capacity registration, GPU attestation, and
          streamed payouts. Partner sign-up will open with that release.</p>
        <p class="soon-contact">Want to supply GPUs early? <a href="mailto:partners@exascale.ai">partners@exascale.ai</a></p>
        <NuxtLink to="/" class="soon-btn">← Back to Exascale</NuxtLink>
      </div>
    </main>
  </div>

  <div v-else class="reg-shell" data-theme="light">
    <!-- ============ Top chrome ============ -->
    <header class="topbar">
      <NuxtLink to="/datacenter" class="brand">
        <span class="brand-mark" />
        EXASCALE
      </NuxtLink>
      <nav class="crumbs">
        <NuxtLink to="/datacenter">Datacenter</NuxtLink>
        <span class="sep">›</span>
        <span class="cur">Register partner</span>
      </nav>
      <div class="top-right">
        <span class="env-pill">PARTNER OPS · INTERNAL</span>
        <span class="clock">
          <span class="pulse" />
          {{ formClock }}
        </span>
        <button type="button" class="ghost-btn">Save draft</button>
        <NuxtLink to="/datacenter" class="ghost-btn">Cancel</NuxtLink>
      </div>
    </header>

    <main class="page">
      <!-- Page header -->
      <section class="page-head">
        <div class="page-head-left">
          <div class="eyebrow"><span class="dot" />Partner onboarding · capacity registration</div>
          <h1 class="page-title">Register a new capacity partner</h1>
          <p class="page-sub">
            Provision a datacenter into the venue's compute underlying. Verification runs
            automatically and typically completes in ~15 minutes. Commercial terms here are
            binding once activated; revisions follow the standard partner amendment process.
          </p>
        </div>
        <aside class="page-head-right">
          <div class="form-progress">
            <span class="fp-lbl">Form progress</span>
            <div class="fp-bar">
              <div class="fp-fill" :style="{ width: verifyProgress + '%' }" />
            </div>
            <span class="fp-meta">{{ Math.round(verifyProgress) }}% · verification</span>
          </div>
          <div class="form-progress">
            <span class="fp-lbl">Sections</span>
            <span class="fp-meta">3 of 5 complete</span>
          </div>
        </aside>
      </section>

      <!-- ===================================================
           Section 01 — Identification
           =================================================== -->
      <article class="section" id="identification">
        <header class="section-head">
          <div class="sh-left">
            <span class="sh-num">01</span>
            <h2 class="sh-title">Partner identification</h2>
          </div>
          <span class="section-status complete">
            <span class="dot" /> Complete
          </span>
        </header>
        <div class="section-body">
          <div class="grid-2">
            <div class="field col-2">
              <label class="field-label"><span>— Legal entity name</span></label>
              <input v-model="form.legalName" class="input" type="text" />
              <span class="field-hint">As registered. Used on contracts and statements.</span>
            </div>
            <div class="field">
              <label class="field-label">
                <span>— Trading as / cluster code</span>
                <span class="req optional">Optional</span>
              </label>
              <input v-model="form.tradingAs" class="input mono-input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Operator since</span></label>
              <input v-model="form.operatorSince" class="input mono-input" type="date" />
            </div>
          </div>

          <div class="subhead">— Primary technical contact</div>
          <div class="grid-3">
            <div class="field">
              <label class="field-label"><span>— Full name</span></label>
              <input v-model="form.techName" class="input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Email</span></label>
              <input v-model="form.techEmail" class="input mono-input" type="email" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Phone</span></label>
              <input v-model="form.techPhone" class="input mono-input" type="tel" />
            </div>
          </div>

          <div class="subhead">— Primary business contact</div>
          <div class="grid-3">
            <div class="field">
              <label class="field-label"><span>— Full name</span></label>
              <input v-model="form.bizName" class="input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Email</span></label>
              <input v-model="form.bizEmail" class="input mono-input" type="email" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Phone</span></label>
              <input v-model="form.bizPhone" class="input mono-input" type="tel" />
            </div>
          </div>

          <div class="subhead">— Datacenter location</div>
          <div class="grid-2">
            <div class="field">
              <label class="field-label"><span>— Country</span></label>
              <select v-model="form.country" class="select">
                <option v-for="[code, name] in COUNTRIES" :key="code" :value="code">
                  {{ name }} · {{ code }}
                </option>
              </select>
            </div>
            <div class="field">
              <label class="field-label"><span>— City</span></label>
              <input v-model="form.city" class="input" type="text" />
            </div>
          </div>
        </div>
      </article>

      <!-- ===================================================
           Section 02 — Capacity
           =================================================== -->
      <article class="section" id="capacity">
        <header class="section-head">
          <div class="sh-left">
            <span class="sh-num">02</span>
            <h2 class="sh-title">Capacity details</h2>
          </div>
          <span class="section-status complete">
            <span class="dot" /> Complete
          </span>
        </header>
        <div class="section-body">
          <div class="grid-2">
            <div class="field">
              <label class="field-label"><span>— GPU type</span></label>
              <select v-model="form.gpuType" class="select">
                <option v-for="g in GPU_OPTIONS" :key="g.value" :value="g.value">{{ g.label }}</option>
              </select>
            </div>
            <div class="field">
              <label class="field-label">
                <span>— GPU count</span>
                <span class="req mono-inline">×{{ fmtInt(form.gpuCount) }}</span>
              </label>
              <input v-model.number="form.gpuCount" class="input mono-input" type="number" min="8" step="8" />
              <span class="field-hint">Step in multiples of 8 (one full rack node).</span>
            </div>
          </div>

          <div class="subhead">— Networking</div>
          <div class="grid-3">
            <div class="field">
              <label class="field-label"><span>— Interconnect</span></label>
              <select v-model="form.interconnect" class="select">
                <option v-for="i in INTERCONNECTS" :key="i.value" :value="i.value">{{ i.label }}</option>
              </select>
            </div>
            <div class="field">
              <label class="field-label"><span>— Bandwidth · per node</span></label>
              <input v-model="form.bandwidth" class="input mono-input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Topology</span></label>
              <select v-model="form.topology" class="select">
                <option v-for="t in TOPOLOGIES" :key="t" :value="t">{{ t }}</option>
              </select>
            </div>
          </div>

          <div class="subhead">— Service level</div>
          <div class="grid-2">
            <div class="field">
              <label class="field-label"><span>— Uptime SLA commitment</span></label>
              <div class="input-affix">
                <input v-model.number="form.uptime" class="input mono-input" type="number" step="0.01" min="95" max="100" />
                <span class="affix-right">%</span>
              </div>
              <span class="field-hint">Annual rolling. Below threshold triggers contractual remediation.</span>
            </div>
            <div class="field">
              <label class="field-label"><span>— Maintenance windows</span></label>
              <input v-model="form.maintenanceWindows" class="input" type="text" />
            </div>
          </div>
        </div>
      </article>

      <!-- ===================================================
           Section 03 — Commercial
           =================================================== -->
      <article class="section" id="commercial">
        <header class="section-head">
          <div class="sh-left">
            <span class="sh-num">03</span>
            <h2 class="sh-title">Commercial terms</h2>
          </div>
          <span class="section-status complete">
            <span class="dot" /> Complete
          </span>
        </header>
        <div class="section-body">
          <div class="grid-2">
            <div class="field">
              <label class="field-label"><span>— Pricing floor</span></label>
              <div class="input-affix">
                <span class="affix-left">$</span>
                <input v-model.number="form.floorPrice" class="input mono-input pad-l" type="number" step="0.01" min="0" />
                <span class="affix-right">/ GPU-hr</span>
              </div>
              <span class="field-hint">Lowest accepted clearing price for your capacity.</span>
            </div>
            <div class="field">
              <label class="field-label"><span>— Revenue share · partner</span></label>
              <div class="input-affix">
                <input v-model.number="form.revenueShare" class="input mono-input" type="number" min="50" max="90" step="1" />
                <span class="affix-right">%</span>
              </div>
              <span class="field-hint">
                Exascale fee:
                <strong>{{ 100 - form.revenueShare }}%</strong>
                · Standard band 60–75%
              </span>
            </div>
          </div>

          <div class="grid-2">
            <div class="field">
              <label class="field-label"><span>— Payout currency</span></label>
              <select v-model="form.payoutCurrency" class="select">
                <option v-for="c in CURRENCIES" :key="c" :value="c">{{ c }}</option>
              </select>
            </div>
            <div class="field">
              <label class="field-label"><span>— Payout cadence</span></label>
              <div class="radio-group" :style="{ '--cols': 2 }">
                <label
                  class="radio-opt"
                  :class="{ checked: form.payoutCadence === 'weekly' }"
                >
                  <input v-model="form.payoutCadence" type="radio" value="weekly" />
                  <span>Weekly · Fridays</span>
                </label>
                <label
                  class="radio-opt"
                  :class="{ checked: form.payoutCadence === 'monthly' }"
                >
                  <input v-model="form.payoutCadence" type="radio" value="monthly" />
                  <span>Monthly · 1st</span>
                </label>
              </div>
            </div>
          </div>

          <div class="subhead">— Bank wire details</div>
          <div class="grid-2">
            <div class="field col-2">
              <label class="field-label"><span>— Beneficiary</span></label>
              <input v-model="form.beneficiary" class="input" type="text" />
            </div>
            <div class="field col-2">
              <label class="field-label"><span>— Beneficiary address</span></label>
              <input v-model="form.beneficiaryAddress" class="input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— Bank</span></label>
              <input v-model="form.bankName" class="input" type="text" />
            </div>
            <div class="field">
              <label class="field-label"><span>— SWIFT / BIC</span></label>
              <input v-model="form.swift" class="input mono-input" type="text" />
            </div>
            <div class="field col-2">
              <label class="field-label"><span>— IBAN</span></label>
              <input v-model="form.iban" class="input mono-input" type="text" />
              <span class="field-hint">Verified against Nordea Bank · 4 fields matched · check digits valid</span>
            </div>
          </div>
        </div>
      </article>

      <!-- ===================================================
           Section 04 — Verification
           =================================================== -->
      <article class="section" id="verification">
        <header class="section-head">
          <div class="sh-left">
            <span class="sh-num">04</span>
            <h2 class="sh-title">Capacity verification</h2>
          </div>
          <span class="section-status" :class="verifyComplete ? 'complete' : 'in-progress'">
            <span class="dot" />
            {{ verifyComplete ? 'Complete' : 'In progress · ' + Math.round(verifyProgress) + '%' }}
          </span>
        </header>
        <div class="section-body">
          <div class="verify-meta">
            <span class="vm-k">Verification suite</span>
            <span class="vm-v">v2.4.1 · started {{ verifyStartedAt }}</span>
          </div>

          <ol class="verify-steps">
            <li
              v-for="(s, i) in steps"
              :key="s.id"
              class="verify-step"
              :class="s.state"
            >
              <div class="vs-rail">
                <span class="vs-icon">
                  <!-- done: filled disc with check -->
                  <svg v-if="s.state === 'done'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="square">
                    <circle cx="8" cy="8" r="7.5" fill="currentColor" stroke="none" />
                    <path d="M4.5 8.2l2.4 2.4L11.5 6" stroke="white" />
                  </svg>
                  <!-- running: half-filled disc -->
                  <svg v-else-if="s.state === 'running'" viewBox="0 0 16 16" fill="none">
                    <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.4" fill="none" />
                    <path d="M8 1.5 a 6.5 6.5 0 0 1 0 13 Z" fill="currentColor" />
                  </svg>
                  <!-- pending: outline -->
                  <svg v-else viewBox="0 0 16 16" fill="none">
                    <circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 2" fill="none" />
                  </svg>
                </span>
                <span v-if="i !== steps.length - 1" class="vs-line" />
              </div>
              <div class="vs-body">
                <div class="vs-row">
                  <span class="vs-title">{{ s.title }}</span>
                  <span class="vs-ts">{{ s.timestamp }}</span>
                </div>
                <p class="vs-detail">{{ s.detail }}</p>
                <template v-if="s.state === 'running' && s.progress !== undefined">
                  <div class="vs-bar">
                    <div class="vs-bar-fill" :style="{ width: s.progress + '%' }" />
                  </div>
                  <div class="vs-bench" v-if="s.bench">{{ s.bench }}</div>
                </template>
              </div>
            </li>
          </ol>

          <div class="verify-foot">
            <span>Next probe in {{ Math.max(1, 30 - new Date().getSeconds() % 30) }}s · runs every 30s</span>
            <a href="#">View full log →</a>
          </div>
        </div>
      </article>

      <!-- ===================================================
           Section 05 — Activation
           =================================================== -->
      <article class="section activation" id="activation">
        <header class="section-head">
          <div class="sh-left">
            <span class="sh-num">05</span>
            <h2 class="sh-title">Activation</h2>
          </div>
          <span class="section-status pending">
            <span class="dot" /> Pending verification
          </span>
        </header>
        <div class="section-body">
          <div class="summary-card">
            <div class="sc-row">
              <span class="sc-k">Partner</span>
              <span class="sc-v">{{ form.legalName }} <span class="sc-dim">· {{ form.tradingAs }}</span></span>
            </div>
            <div class="sc-row">
              <span class="sc-k">Capacity</span>
              <span class="sc-v">
                <strong>{{ fmtInt(activeCount) }}</strong> / {{ fmtInt(form.gpuCount) }} GPUs
                <span class="sc-dim">· {{ gpuSpec() }}</span>
              </span>
            </div>
            <div class="sc-row">
              <span class="sc-k">Networking</span>
              <span class="sc-v">{{ interconnectLabel() }} · {{ form.topology }}</span>
            </div>
            <div class="sc-row">
              <span class="sc-k">Floor / SLA</span>
              <span class="sc-v">${{ form.floorPrice.toFixed(2) }} / GPU-hr · {{ form.uptime.toFixed(2) }}% uptime</span>
            </div>
            <div class="sc-divider" />
            <div class="sc-row">
              <span class="sc-k">Projected monthly revenue</span>
              <span class="sc-v big">{{ fmtUsd(projectedMonthlyRevenue) }}</span>
            </div>
            <div class="sc-row">
              <span class="sc-k">Your share · {{ form.revenueShare }}%</span>
              <span class="sc-v pos">{{ fmtUsd(projectedPartnerShare) }}</span>
            </div>
            <div class="sc-row">
              <span class="sc-k">Exascale fee · {{ 100 - form.revenueShare }}%</span>
              <span class="sc-v">{{ fmtUsd(projectedExascaleFee) }}</span>
            </div>
            <div class="sc-foot">
              At ${{ form.floorPrice.toFixed(2) }} floor × {{ monthlyHours }} avg monthly hours × {{ form.revenueShare }}% share.
              Actual revenue depends on clearing prices and utilization.
            </div>
          </div>

          <label class="ramp-row" :class="{ checked: form.softLaunch }">
            <input v-model="form.softLaunch" type="checkbox" />
            <span class="check-box">
              <svg v-if="form.softLaunch" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="square">
                <path d="M3 8.5l3.2 3.2L13 5" />
              </svg>
            </span>
            <div class="ramp-body">
              <div class="ramp-title">Soft-launch this partner</div>
              <p class="ramp-sub">
                Start with <strong>{{ form.softLaunchPct }}% of capacity</strong>
                ({{ fmtInt(Math.round(form.gpuCount * form.softLaunchPct / 100)) }} GPUs) for
                <strong>{{ form.softLaunchDuration }} days</strong>, then auto-ramp to 100% on
                <span class="mono">{{ form.activationDate }} + 14d</span>. Reduces blast-radius if anything regresses on the venue side.
              </p>
              <div v-if="form.softLaunch" class="ramp-controls">
                <div class="rc-field">
                  <label>Initial %</label>
                  <input v-model.number="form.softLaunchPct" class="input mono-input narrow" type="number" min="10" max="50" step="5" />
                </div>
                <div class="rc-field">
                  <label>Ramp duration</label>
                  <input v-model.number="form.softLaunchDuration" class="input mono-input narrow" type="number" min="7" max="30" step="1" />
                  <span class="rc-unit">days</span>
                </div>
              </div>
            </div>
          </label>

          <div class="activation-row">
            <div class="field activation-date-field">
              <label class="field-label"><span>— Activation date</span></label>
              <input v-model="form.activationDate" class="input mono-input" type="date" />
              <span class="field-hint">Capacity becomes live on the venue at 00:00 UTC.</span>
            </div>
            <div class="activation-actions">
              <button type="button" class="btn secondary">Save &amp; finish later</button>
              <button type="button" class="btn primary" :disabled="!verifyComplete">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                  <path d="M8 3v10M3 8h10" />
                </svg>
                {{ verifyComplete ? 'Activate this capacity' : 'Awaiting verification…' }}
              </button>
            </div>
          </div>

          <div class="legal-note">
            By activating, you confirm Exascale's
            <a href="#">Partner Master Agreement</a>,
            <a href="#">Pricing Schedule v2.4</a>, and the
            <a href="#">Capacity Verification Report</a> generated above. Contract becomes
            effective on the activation date. Soft-launch ramp may be paused from
            ops at any time.
          </div>
        </div>
      </article>

      <!-- Page foot -->
      <footer class="page-foot">
        <span class="pf-k">Onboarding session</span>
        <span class="pf-v mono-inline">ONB-7A3C91 · operator: ops@exascale.com · v2.4.1</span>
      </footer>
    </main>
  </div>
</template>

<style scoped>
/* Coming-soon gate (datacenter onboarding not yet open) */
.soon-shell { min-height: 100vh; display: flex; flex-direction: column; background: var(--canvas); color: var(--text); font-family: var(--font-sans); }
.soon-shell .topbar { display: flex; align-items: center; padding: 24px 40px; }
.soon-shell .brand { display: inline-flex; align-items: center; gap: 10px; font-family: var(--font-display); font-weight: 700; letter-spacing: -0.02em; color: var(--text); text-decoration: none; }
.soon-shell .brand-mark { width: 12px; height: 12px; background: var(--brand); display: inline-block; }
.soon-main { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; }
.soon-card { max-width: 480px; text-align: center; border: 1px solid var(--border); background: var(--elevated); border-radius: 2px; padding: 48px 40px; }
.soon-pill { display: inline-block; font-family: var(--font-mono); font-size: 11px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-2); border: 1px solid var(--border); border-radius: 2px; padding: 3px 8px; margin-bottom: 18px; }
.soon-card h1 { font-family: var(--font-display); font-size: 24px; font-weight: 600; margin: 0 0 12px; letter-spacing: -0.01em; }
.soon-card p { color: var(--text-2); font-size: 14px; line-height: 1.6; margin: 0 0 12px; }
.soon-contact a { color: var(--text); }
.soon-btn { display: inline-block; margin-top: 18px; color: var(--text); font-size: 13px; text-decoration: underline; text-underline-offset: 3px; }

.reg-shell {
  --bd-soft: rgba(14, 14, 14, 0.06);
  --t-4:     #B8B8B0;

  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  font-feature-settings: 'ss01';
}
.mono-inline {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}
.pos { color: var(--pos); }
.dim { color: var(--text-3); }

/* ============================================================
   Top chrome
   ============================================================ */
.topbar {
  height: 56px;
  background: var(--elevated);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 32px;
  gap: 24px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 14px;
  letter-spacing: -0.005em;
  color: var(--text);
  text-decoration: none;
  padding-right: 24px;
  border-right: 1px solid var(--border);
  height: 36px;
}
.brand-mark {
  width: 12px;
  height: 12px;
  background: var(--brand);
  display: inline-block;
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
.crumbs a {
  color: var(--text-2);
  text-decoration: none;
}
.crumbs a:hover { color: var(--text); }
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .cur { color: var(--text); }

.top-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 16px;
}
.env-pill {
  padding: 3px 8px;
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border: 1px solid rgba(74, 144, 226, 0.25);
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  border-radius: var(--radius-sm);
}
.clock {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
}
.clock .pulse {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}
@keyframes pulse {
  0%   { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0.6); }
  70%  { box-shadow: 0 0 0 6px rgba(22, 163, 74, 0); }
  100% { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0); }
}
.ghost-btn {
  height: 30px;
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  background: transparent;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
}
.ghost-btn:hover {
  background: var(--canvas);
  color: var(--text);
  border-color: rgba(14, 14, 14, 0.32);
}

/* ============================================================
   Page
   ============================================================ */
.page {
  max-width: 960px;
  margin: 0 auto;
  padding: 40px 32px 96px;
}

.page-head {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 40px;
  align-items: end;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 32px;
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
  margin-bottom: 12px;
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
  font-size: 32px;
  letter-spacing: -0.022em;
  line-height: 1.1;
  margin: 0 0 10px;
  color: var(--text);
}
.page-sub {
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.55;
  margin: 0;
  max-width: 580px;
}

.page-head-right {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: stretch;
}
.form-progress {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.fp-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.fp-bar {
  height: 4px;
  background: var(--sunken);
  border-radius: 2px;
  overflow: hidden;
}
.fp-fill {
  height: 100%;
  background: var(--text);
  transition: width 400ms ease-out;
}
.fp-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
  font-variant-numeric: tabular-nums;
}

/* ============================================================
   Section card
   ============================================================ */
.section {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 20px;
  scroll-margin-top: 80px;
}
.section-head {
  padding: 18px 24px;
  border-bottom: 1px solid var(--bd-soft);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}
.sh-left {
  display: flex;
  align-items: baseline;
  gap: 14px;
}
.sh-num {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.16em;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  padding-top: 4px;
}
.sh-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 20px;
  letter-spacing: -0.015em;
  margin: 0;
}
.section-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.section-status .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.section-status.complete {
  background: rgba(22, 163, 74, 0.10);
  color: var(--pos);
  border: 1px solid rgba(22, 163, 74, 0.25);
}
.section-status.in-progress {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border: 1px solid rgba(74, 144, 226, 0.25);
}
.section-status.pending {
  background: transparent;
  color: var(--text-3);
  border: 1px dashed var(--border-strong);
}
.section-status.incomplete {
  background: transparent;
  color: var(--text-3);
  border: 1px solid var(--border);
}

.section-body { padding: 22px 24px 24px; }

.subhead {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin: 24px 0 12px;
  padding-top: 18px;
  border-top: 1px dashed var(--bd-soft);
}
.section-body > .subhead:first-child { margin-top: 0; padding-top: 0; border-top: 0; }

/* ============================================================
   Form fields
   ============================================================ */
.grid-2,
.grid-3 {
  display: grid;
  gap: 14px;
}
.grid-2 { grid-template-columns: 1fr 1fr; }
.grid-3 { grid-template-columns: 1fr 1fr 1fr; }
.field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field.col-2 { grid-column: span 2; }
.field-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.field-label .req {
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: none;
  font-size: 11px;
  font-family: var(--font-sans);
}
.field-label .req.optional { color: var(--text-3); }
.field-hint {
  font-size: 11.5px;
  color: var(--text-3);
  font-family: var(--font-sans);
  line-height: 1.45;
}
.field-hint strong { color: var(--text); font-weight: 500; }

.input,
.select {
  height: 38px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text);
  width: 100%;
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
  font-feature-settings: 'ss01';
  -webkit-appearance: none;
  appearance: none;
}
.input::placeholder { color: var(--t-4); }
.input:hover,
.select:hover { border-color: rgba(14, 14, 14, 0.32); }
.input:focus,
.select:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.mono-input {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13.5px;
  letter-spacing: 0.02em;
}
.narrow { max-width: 100px; }

.select {
  background-image: url("data:image/svg+xml,%3Csvg width='10' height='6' viewBox='0 0 10 6' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1L5 5L9 1' stroke='%234A4A45' stroke-width='1.4' fill='none' stroke-linecap='square'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 32px;
  cursor: pointer;
}

.input-affix {
  position: relative;
  display: flex;
  align-items: center;
}
.input-affix .input { padding-right: 64px; }
.input-affix .pad-l { padding-left: 26px; }
.input-affix .affix-right,
.input-affix .affix-left {
  position: absolute;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-3);
  pointer-events: none;
}
.input-affix .affix-right { right: 12px; }
.input-affix .affix-left  { left: 12px; }

/* Radio group */
.radio-group {
  display: grid;
  grid-template-columns: repeat(var(--cols, 2), 1fr);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
  height: 38px;
}
.radio-opt {
  position: relative;
  cursor: pointer;
  background: var(--elevated);
  padding: 0 14px;
  border-right: 1px solid var(--border-strong);
  font-size: 13px;
  color: var(--text-2);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 120ms, color 120ms;
}
.radio-opt:last-child { border-right: 0; }
.radio-opt input { position: absolute; opacity: 0; pointer-events: none; }
.radio-opt:hover { background: var(--sunken); }
.radio-opt.checked {
  background: var(--text);
  color: var(--elevated);
}
.radio-opt.checked::before {
  content: '';
  width: 6px;
  height: 6px;
  background: var(--brand);
  display: inline-block;
}

/* ============================================================
   Verification timeline
   ============================================================ */
.verify-meta {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding-bottom: 18px;
  margin-bottom: 20px;
  border-bottom: 1px dashed var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.vm-k { color: var(--text-3); font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; }
.vm-v { color: var(--text); font-variant-numeric: tabular-nums; }

.verify-steps {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
}
.verify-step {
  display: grid;
  grid-template-columns: 28px 1fr;
  gap: 14px;
  padding-bottom: 18px;
}
.verify-step:last-child { padding-bottom: 0; }
.vs-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.vs-icon {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.vs-icon svg { width: 18px; height: 18px; }
.verify-step.done .vs-icon    { color: var(--pos); }
.verify-step.running .vs-icon { color: var(--accent); }
.verify-step.pending .vs-icon { color: var(--text-3); }
.verify-step.failed .vs-icon  { color: var(--neg); }

.vs-line {
  flex: 1;
  width: 1px;
  background: var(--border);
  margin: 4px 0;
  min-height: 14px;
}
.verify-step.done .vs-line     { background: var(--pos); opacity: 0.4; }
.verify-step.running .vs-line  { background: var(--accent); opacity: 0.3; }

.vs-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 1px;
  min-width: 0;
}
.vs-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
}
.vs-title {
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 14px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.vs-ts {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.vs-detail {
  font-size: 13px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.5;
}
.vs-bar {
  height: 4px;
  background: var(--sunken);
  border-radius: 2px;
  overflow: hidden;
  margin-top: 6px;
}
.vs-bar-fill {
  height: 100%;
  background: var(--accent);
  transition: width 600ms ease-out;
}
.vs-bench {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--accent);
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
  margin-top: 4px;
}

.verify-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 22px;
  padding-top: 16px;
  border-top: 1px dashed var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.verify-foot a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
}
.verify-foot a:hover { text-decoration: underline; }

/* ============================================================
   Activation
   ============================================================ */
.section.activation {
  border-color: rgba(14, 14, 14, 0.18);
}

.summary-card {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 18px 22px 14px;
  margin-bottom: 18px;
}
.sc-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 6px 0;
  font-size: 13px;
}
.sc-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.sc-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text);
  font-size: 13px;
}
.sc-v strong { font-weight: 600; }
.sc-v.big {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.sc-v.pos { color: var(--pos); font-size: 16px; font-weight: 500; }
.sc-dim { color: var(--text-3); font-weight: 400; }
.sc-divider {
  height: 1px;
  background: var(--border);
  margin: 8px 0;
}
.sc-foot {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--bd-soft);
  font-family: var(--font-sans);
  font-size: 11.5px;
  color: var(--text-3);
  line-height: 1.55;
}

/* Ramp toggle row */
.ramp-row {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 12px;
  padding: 16px 20px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  cursor: pointer;
  margin-bottom: 18px;
  position: relative;
}
.ramp-row.checked {
  background: rgba(200, 242, 92, 0.10);
  border-color: rgba(14, 14, 14, 0.30);
}
.ramp-row input { position: absolute; opacity: 0; pointer-events: none; }
.ramp-row .check-box {
  width: 18px;
  height: 18px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-top: 1px;
}
.ramp-row.checked .check-box {
  background: var(--text);
  border-color: var(--text);
  color: var(--brand);
}
.ramp-row.checked .check-box svg { width: 12px; height: 12px; }

.ramp-body { min-width: 0; }
.ramp-title {
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 14px;
  color: var(--text);
  letter-spacing: -0.005em;
  margin-bottom: 4px;
}
.ramp-sub {
  font-size: 13px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.55;
}
.ramp-sub .mono { font-family: var(--font-mono); color: var(--text); }

.ramp-controls {
  display: flex;
  gap: 24px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed rgba(14, 14, 14, 0.18);
}
.rc-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rc-field label {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.rc-field {
  display: flex;
  flex-direction: row;
  align-items: end;
  gap: 8px;
}
.rc-field label {
  margin-bottom: 6px;
}
.rc-field .rc-unit {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding-bottom: 10px;
}

/* Activation actions row */
.activation-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 24px;
  align-items: end;
  padding-top: 18px;
  border-top: 1px dashed var(--bd-soft);
}
.activation-date-field { max-width: 240px; }
.activation-actions {
  display: flex;
  gap: 10px;
}
.btn {
  height: 44px;
  padding: 0 22px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: background 120ms, border-color 120ms, color 120ms;
}
.btn svg { width: 16px; height: 16px; }
.btn.primary {
  background: var(--brand);
  color: var(--text);
  border-color: var(--brand);
  font-weight: 600;
}
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.primary:disabled {
  background: var(--sunken);
  color: var(--text-3);
  border-color: var(--border);
  cursor: not-allowed;
}
.btn.secondary {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover {
  background: var(--canvas);
  border-color: rgba(14, 14, 14, 0.32);
}

.legal-note {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px dashed var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  line-height: 1.7;
  letter-spacing: 0.02em;
}
.legal-note a {
  color: var(--text-2);
  text-decoration: underline;
  text-decoration-color: var(--text-3);
}
.legal-note a:hover { color: var(--text); }

/* Page footer */
.page-foot {
  margin-top: 28px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.pf-k { font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; }
.pf-v { color: var(--text-2); }

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 880px) {
  .page-head { grid-template-columns: 1fr; }
  .grid-2, .grid-3 { grid-template-columns: 1fr; }
  .field.col-2 { grid-column: auto; }
  .activation-row { grid-template-columns: 1fr; }
  .activation-actions { width: 100%; flex-direction: column; }
  .btn { width: 100%; justify-content: center; }
}
</style>
