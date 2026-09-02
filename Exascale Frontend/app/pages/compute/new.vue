<script setup lang="ts">
/**
 * /compute/new — Provision a new GPU instance.
 *
 * Renders inside the `app` layout (dark). Three columns visible at once
 * (no stepwise wizard): Hardware · Configuration · Cost preview.
 * The cost card is sticky on the right and recomputes live.
 */

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Provision new instance · Compute — 1Trade' })

const router = useRouter()

// =====================================================
// Catalog
// =====================================================
type GpuKey = string
interface GpuOption {
  key: GpuKey
  name: string
  spec: string
  ratePerHour: number   // USD per GPU-hour
  recommended?: string
  disabled?: boolean
  status?: string
  tag?: string
}
// The GPU-type list IS the live rental catalog (compute-control /v1/compute/types) — the full lineup
// with datasheet specs + availability. Loaded in onMounted; available tiers are selectable, sold-out /
// coming-soon ones show a waitlist CTA. Empty fallback keeps the derived numbers safe before load.
const EMPTY_GPU: GpuOption = { key: '', name: '—', spec: '', ratePerHour: 0 }
const gpuOptions = ref<GpuOption[]>([])

/** mapToOption turns a catalog GpuType into the provision list's option shape. */
function mapToOption(t: import('~/composables/useCompute').GpuType): GpuOption {
  const status = t.status ?? 'available'
  const arch = t.specs?.architecture ?? ''
  const ff = t.specs?.form_factor ?? ''
  return {
    key: t.id,
    name: t.name,
    spec: [arch, ff].filter(Boolean).join(' · '),
    ratePerHour: Number(t.price_per_hour),
    status,
    disabled: status !== 'available',
    tag: status === 'coming_soon' ? 'Coming soon' : status === 'sold_out' ? 'Sold out' : undefined,
    recommended: t.specs ? `${t.specs.vram} · ${t.specs.mem_bandwidth} · FP8 ${t.specs.fp8_tflops} TFLOPS` : '',
  }
}

const REGIONS = [
  { value: 'us-east-1',      label: 'us-east-1 · N. Virginia',     latency: '12ms RTT · 32 GPU pool' },
  { value: 'eu-west-1',      label: 'eu-west-1 · Dublin',          latency: '88ms RTT · 24 GPU pool' },
  { value: 'ap-northeast-1', label: 'ap-northeast-1 · Tokyo',      latency: '142ms RTT · 16 GPU pool' },
]

const IMAGES = [
  { value: 'exa-ml',      label: '1Trade ML Stack · PyTorch 2.4 · CUDA 12.4', sub: 'Default · maintained image · 3.7 GB' },
  { value: 'exa-vllm',    label: '1Trade vLLM · 0.5.2',                       sub: 'Optimized inference server' },
  { value: 'exa-axolotl', label: '1Trade Axolotl · 0.4',                       sub: 'Fine-tuning toolkit' },
  { value: 'custom',      label: 'Custom image · paste image URI',               sub: 'docker.io / ghcr.io / private registry' },
  { value: 'byo',         label: 'Bring your own · Docker image URL',            sub: 'Untrusted images run in isolated VPC' },
]

const VPCS = [
  { value: 'default',  label: 'default · 10.0.0.0/16' },
  { value: 'training', label: 'training · 10.10.0.0/16' },
  { value: 'prod',     label: 'prod · 10.20.0.0/16' },
]

const COUNT_PRESETS = [1, 2, 4, 8, 16, 32]
const STORAGE_PER_GB_MONTH = 0.10

// =====================================================
// Form state — pre-populated for the demo
// =====================================================
const form = reactive({
  gpu: '' as GpuKey,
  count: 8,
  region: 'us-east-1',
  image: 'exa-ml',
  imageUri: '',                                        // for custom/byo
  sshKey: 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINrZ7q5sX5q+Yk8oN9p8X4w0Lq6vYx5gC4dFqJh2bP1k marcus.chen@frontier.lab',
  publicIp: true,
  vpc: 'default',
  infiniband: true,
  storageGb: 1000,
  instanceName: 'training-run-2026-05-21',
})

// Watch: enable IB when count > 1, lock it
watch(() => form.count, (c) => {
  if (c > 1) form.infiniband = true
})

// =====================================================
// Derived numbers
// =====================================================
const selectedGpu = computed(() =>
  gpuOptions.value.find(g => g.key === form.gpu) ?? gpuOptions.value[0] ?? EMPTY_GPU,
)

const hourlyGpuRate = computed(() => form.count * selectedGpu.value.ratePerHour)
const monthlyStorageCost = computed(() => form.storageGb * STORAGE_PER_GB_MONTH)

const cost24h = computed(() => hourlyGpuRate.value * 24)
const cost30d = computed(() => hourlyGpuRate.value * 24 * 30 + monthlyStorageCost.value)

const budgetRemaining = 13_152.68

// =====================================================
// Formatters
// =====================================================
function fmtUsd(n: number, dp = 2) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtGb(gb: number) {
  if (gb >= 1000) {
    const tb = gb / 1000
    return tb % 1 === 0 ? `${tb} TB` : `${tb.toFixed(1)} TB`
  }
  return `${gb} GB`
}
function fmtCount(n: number) {
  return n.toString()
}

// =====================================================
// CLI command preview (live)
// =====================================================
const cliPreview = computed(() => {
  const parts = [
    'exascale gpu create',
    `--type ${form.gpu}`,
    `--count ${form.count}`,
    `--region ${form.region}`,
    `--image ${form.image}`,
  ]
  if (form.storageGb > 0) parts.push(`--storage ${form.storageGb}GB`)
  if (!form.publicIp) parts.push('--no-public-ip')
  if (form.vpc !== 'default') parts.push(`--vpc ${form.vpc}`)
  if (form.infiniband && form.count > 1) parts.push('--ib')
  if (form.instanceName) parts.push(`--name ${form.instanceName}`)
  return parts.join(' \\\n  ')
})

const cliShortInline = 'exascale gpu create'

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch { return false }
}

const copiedFlag = ref<'' | 'short' | 'full'>('')
async function copyShort() {
  if (await copyText(cliShortInline)) {
    copiedFlag.value = 'short'
    setTimeout(() => { copiedFlag.value = '' }, 1400)
  }
}
async function copyFull() {
  if (await copyText(cliPreview.value)) {
    copiedFlag.value = 'full'
    setTimeout(() => { copiedFlag.value = '' }, 1400)
  }
}

// =====================================================
// Storage slider — log-ish scale via discrete steps
// =====================================================
// 0, 100, 250, 500, 1000, 2000, 4000, 8000, 10000 GB
const STORAGE_STOPS = [0, 100, 250, 500, 1000, 2000, 4000, 8000, 10000]
function storageStepIndex() {
  // Map current GB to nearest stop index for slider position
  const idx = STORAGE_STOPS.findIndex(v => v === form.storageGb)
  return idx === -1 ? 4 : idx
}
const storageIdx = ref(storageStepIndex())
watch(storageIdx, (i) => { form.storageGb = STORAGE_STOPS[i] ?? 1000 })

// =====================================================
// Provision action — fake transition
// =====================================================
const compute = useCompute()

// Load the live GPU catalog into the type list; default the selection to the first available tier.
onMounted(async () => {
  try {
    const types = await compute.loadTypes()
    gpuOptions.value = types.map(mapToOption)
    const firstAvailable = gpuOptions.value.find(g => !g.disabled) ?? gpuOptions.value[0]
    if (firstAvailable) form.gpu = firstAvailable.key
  } catch { /* keep the (empty) list; the form guards against it */ }
})

const provisioning = ref(false)
const provisionError = ref('')
async function provision() {
  if (provisioning.value || !canProvision.value) return
  provisioning.value = true
  provisionError.value = ''
  try {
    // Live create (F13). The ML-Stack image catalog ships 'stable'/'latest' this milestone; the page's
    // image presets all resolve to the maintained stable image. is_paper is server-derived.
    await compute.create({
      type: form.gpu || 'gpu_h100',
      count: form.count,
      image: 'stable',
      region: form.region,
    })
    router.push('/compute')
  } catch (e: unknown) {
    provisioning.value = false
    provisionError.value = (e as { data?: { message?: string } })?.data?.message || 'Could not provision the instance.'
  }
}

const canProvision = computed(() => {
  if (form.count < 1 || form.count > 32) return false
  if (!form.sshKey.trim()) return false
  if (selectedGpu.value.disabled) return false
  return true
})

const overBudget = computed(() => cost30d.value > budgetRemaining)
</script>

<template>
  <div class="provision-page">
    <!-- Sub-topbar -->
    <div class="subbar">
      <nav class="crumbs">
        <NuxtLink to="/compute">Compute</NuxtLink>
        <span class="sep">›</span>
        <span class="strong">New instance</span>
      </nav>
      <div class="subbar-right">
        <span class="cli-inline" :class="{ copied: copiedFlag === 'short' }" @click="copyShort" title="Click to copy">
          <span class="cli-prompt">$</span>
          <span class="cli-cmd">{{ cliShortInline }}</span>
          <span class="cli-copy">{{ copiedFlag === 'short' ? '✓ Copied' : '⌘C' }}</span>
        </span>
        <NuxtLink to="/compute" class="exit-link">← Back to instances</NuxtLink>
      </div>
    </div>

    <main class="page">
      <!-- ============ Header ============ -->
      <section class="page-head">
        <div>
          <div class="eyebrow"><span class="dot" /> Compute · provisioning</div>
          <h1 class="page-title">Provision new instance</h1>
          <p class="page-sub">
            Or use the CLI:
            <code class="inline-cli" @click="copyShort">exascale gpu create</code>
            for full flag coverage. This form maps 1:1 to CLI flags — you can copy the equivalent command from the preview at any time.
          </p>
        </div>
      </section>

      <!-- ============ Three columns ============ -->
      <div class="cols">
        <!-- COLUMN 1 · HARDWARE -->
        <section class="col">
          <div class="col-head">
            <span class="col-num">01</span>
            <h2 class="col-title">Hardware</h2>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">GPU type</span>
            </div>
            <div class="gpu-list">
              <label
                v-for="g in gpuOptions"
                :key="g.key"
                class="gpu-card"
                :class="{ active: form.gpu === g.key, disabled: g.disabled }"
              >
                <input
                  v-model="form.gpu"
                  type="radio"
                  :value="g.key"
                  :disabled="g.disabled"
                />
                <span class="gpu-radio" />
                <div class="gpu-body">
                  <div class="gpu-line">
                    <span class="gpu-name">{{ g.name }}</span>
                    <span class="gpu-spec">{{ g.spec }}</span>
                    <span class="gpu-rate mono">{{ fmtUsd(g.ratePerHour) }}<span class="dim">/hr</span></span>
                  </div>
                  <div class="gpu-sub">
                    <template v-if="g.disabled">
                      <span class="gpu-tag">{{ g.tag }}</span>
                      <a href="#" class="gpu-link" @click.prevent>Sign up for waitlist →</a>
                    </template>
                    <template v-else>{{ g.recommended }}</template>
                  </div>
                </div>
              </label>
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Count</span>
              <span class="section-meta">
                <span class="mono big">{{ form.count }}</span>
                <span class="dim">×</span>
                <span class="dim">{{ selectedGpu.name }}</span>
              </span>
            </div>
            <div class="count-presets">
              <button
                v-for="p in COUNT_PRESETS"
                :key="p"
                type="button"
                class="preset"
                :class="{ active: form.count === p }"
                @click="form.count = p"
              >{{ p }}</button>
              <input
                v-model.number="form.count"
                class="count-input"
                type="number"
                min="1"
                max="32"
                aria-label="Custom GPU count"
              />
            </div>
            <input
              v-model.number="form.count"
              type="range"
              min="1"
              max="32"
              step="1"
              class="slider"
            />
            <div class="section-hint">
              Above 32 GPUs · <a href="#" @click.prevent>Contact sales for cluster size 32+ →</a>
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Region</span>
            </div>
            <div class="select-wrap">
              <select v-model="form.region" class="select">
                <option v-for="r in REGIONS" :key="r.value" :value="r.value">{{ r.label }}</option>
              </select>
            </div>
            <div class="section-hint mono">
              {{ REGIONS.find(r => r.value === form.region)?.latency }}
            </div>
          </div>
        </section>

        <!-- COLUMN 2 · CONFIGURATION -->
        <section class="col">
          <div class="col-head">
            <span class="col-num">02</span>
            <h2 class="col-title">Configuration</h2>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Base image</span>
            </div>
            <div class="select-wrap">
              <select v-model="form.image" class="select">
                <option v-for="i in IMAGES" :key="i.value" :value="i.value">{{ i.label }}</option>
              </select>
            </div>
            <div class="section-hint">
              {{ IMAGES.find(i => i.value === form.image)?.sub }}
            </div>
            <div v-if="form.image === 'custom' || form.image === 'byo'" class="extra-field">
              <input
                v-model="form.imageUri"
                class="input mono-input"
                type="text"
                placeholder="e.g. ghcr.io/walmart/train-pytorch:2.4.0"
              />
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">SSH access</span>
              <span class="section-meta">
                <a href="#" @click.prevent>Manage keys via CLI →</a>
              </span>
            </div>
            <textarea
              v-model="form.sshKey"
              class="textarea mono-input"
              rows="3"
              placeholder="Paste your SSH public key (ssh-ed25519 / ssh-rsa)"
            />
            <div class="section-hint">
              Detected: <span class="mono">{{ form.sshKey.startsWith('ssh-ed25519') ? 'ssh-ed25519' : form.sshKey.startsWith('ssh-rsa') ? 'ssh-rsa' : 'unrecognized' }}</span>
              · fingerprint <span class="mono dim">SHA256:Lp4hF2x…aB91</span>
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Networking</span>
            </div>
            <div class="toggle-row">
              <label class="toggle">
                <input v-model="form.publicIp" type="checkbox" />
                <span class="toggle-track"><span class="toggle-thumb" /></span>
                <span class="toggle-label">
                  <span class="t-title">Public IP</span>
                  <span class="t-sub">Reachable on the public internet · default on</span>
                </span>
              </label>
              <label class="toggle">
                <input
                  v-model="form.infiniband"
                  type="checkbox"
                  :disabled="form.count > 1"
                />
                <span class="toggle-track"><span class="toggle-thumb" /></span>
                <span class="toggle-label">
                  <span class="t-title">InfiniBand</span>
                  <span class="t-sub">
                    <template v-if="form.count > 1">
                      <span class="locked-tag">Locked · auto-on for multi-GPU</span>
                    </template>
                    <template v-else>NDR 400 Gbps interconnect · single-GPU optional</template>
                  </span>
                </span>
              </label>
            </div>
            <div class="vpc-row">
              <span class="section-k inline">— VPC</span>
              <div class="select-wrap inline">
                <select v-model="form.vpc" class="select small">
                  <option v-for="v in VPCS" :key="v.value" :value="v.value">{{ v.label }}</option>
                </select>
              </div>
              <span class="section-hint inline">Advanced · default works for most workloads</span>
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Persistent storage</span>
              <span class="section-meta mono">{{ fmtGb(form.storageGb) }}</span>
            </div>
            <input
              v-model.number="storageIdx"
              type="range"
              min="0"
              :max="STORAGE_STOPS.length - 1"
              step="1"
              class="slider"
            />
            <div class="slider-ticks">
              <span v-for="(s, i) in STORAGE_STOPS" :key="i" :class="{ active: storageIdx === i }">{{ s === 0 ? '0' : (s >= 1000 ? (s / 1000) + 'T' : s + 'G') }}</span>
            </div>
            <div class="section-hint">
              <span class="mono">${{ STORAGE_PER_GB_MONTH.toFixed(2) }}/GB-month</span>
              · independent of instance lifetime · backups separate
            </div>
          </div>

          <div class="section">
            <div class="section-head">
              <span class="section-k">Instance name</span>
            </div>
            <input
              v-model="form.instanceName"
              class="input mono-input"
              type="text"
              placeholder="e.g. training-run-2026-05-21"
            />
            <div class="section-hint">
              Used in the dashboard + DNS · letters / numbers / dashes only
            </div>
          </div>
        </section>

        <!-- COLUMN 3 · COST PREVIEW (sticky) -->
        <aside class="col cost-col">
          <div class="cost-card">
            <div class="col-head">
              <span class="col-num">03</span>
              <h2 class="col-title">Cost preview</h2>
            </div>

            <!-- Instance summary -->
            <div class="cost-section">
              <div class="cs-head">— Instance summary</div>
              <ul class="recap">
                <li>
                  <span class="r-k">GPU</span>
                  <span class="r-v mono">{{ form.count }} × {{ selectedGpu.name }} {{ selectedGpu.spec }}</span>
                </li>
                <li>
                  <span class="r-k">Region</span>
                  <span class="r-v mono">{{ form.region }}</span>
                </li>
                <li>
                  <span class="r-k">Image</span>
                  <span class="r-v">{{ IMAGES.find(i => i.value === form.image)?.label.split(' · ')[0] }}</span>
                </li>
                <li>
                  <span class="r-k">Storage</span>
                  <span class="r-v mono">{{ fmtGb(form.storageGb) }} persistent</span>
                </li>
                <li>
                  <span class="r-k">Network</span>
                  <span class="r-v">
                    {{ form.publicIp ? 'Public IP' : 'Private only' }}
                    <span v-if="form.infiniband && form.count > 1" class="dim">· InfiniBand NDR</span>
                  </span>
                </li>
              </ul>
            </div>

            <!-- Cost breakdown -->
            <div class="cost-section">
              <div class="cs-head">— Cost breakdown</div>
              <div class="cb-row">
                <span class="cb-k">GPU rate</span>
                <span class="cb-v mono">
                  {{ form.count }} × {{ fmtUsd(selectedGpu.ratePerHour) }}/hr
                  <strong>= {{ fmtUsd(hourlyGpuRate) }}/hr</strong>
                </span>
              </div>
              <div class="cb-row">
                <span class="cb-k">Storage</span>
                <span class="cb-v mono">
                  {{ form.storageGb }} GB × ${{ STORAGE_PER_GB_MONTH.toFixed(2) }}/mo
                  <strong>= {{ fmtUsd(monthlyStorageCost) }}/mo</strong>
                </span>
              </div>
              <div class="cb-row">
                <span class="cb-k">Egress</span>
                <span class="cb-v mono pos">Free · no charges</span>
              </div>
              <div class="cb-divider" />
              <div class="cb-row major">
                <span class="cb-k">Total</span>
                <span class="cb-v mono">
                  <strong>{{ fmtUsd(hourlyGpuRate) }}</strong><span class="dim">/hr</span>
                  <span class="plus">+</span>
                  <strong>{{ fmtUsd(monthlyStorageCost) }}</strong><span class="dim">/mo storage</span>
                </span>
              </div>
            </div>

            <!-- Estimates -->
            <div class="estimates">
              <div class="est">
                <div class="est-k">Estimated · 24h</div>
                <div class="est-v mono">{{ fmtUsd(cost24h) }}</div>
              </div>
              <div class="est">
                <div class="est-k">Estimated · 30d</div>
                <div class="est-v mono">{{ fmtUsd(cost30d) }}</div>
              </div>
            </div>

            <!-- Budget -->
            <div class="budget" :class="{ over: overBudget }">
              <div class="budget-row">
                <span class="b-k">Available balance</span>
                <span class="b-v mono">{{ fmtUsd(budgetRemaining) }}</span>
              </div>
              <div class="budget-row">
                <span class="b-k">After 30d run</span>
                <span class="b-v mono" :class="overBudget ? 'neg-text' : 'pos-text'">
                  {{ fmtUsd(budgetRemaining - cost30d) }}
                </span>
              </div>
              <div v-if="overBudget" class="budget-warn">
                30-day projection exceeds remaining budget. Raise the budget under
                <NuxtLink to="/enterprise/teams">Team management</NuxtLink>
                or shorten the workload.
              </div>
            </div>

            <!-- CLI parity -->
            <details class="cli-block">
              <summary>
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M3 4l3 3-3 3M8 11h6" />
                </svg>
                Equivalent CLI command
                <button type="button" class="cli-copy-btn" :class="{ copied: copiedFlag === 'full' }" @click.prevent="copyFull">
                  {{ copiedFlag === 'full' ? '✓ Copied' : 'Copy' }}
                </button>
              </summary>
              <pre class="cli-pre">{{ cliPreview }}</pre>
            </details>

            <button
              type="button"
              class="provision-btn"
              :disabled="!canProvision || provisioning"
              @click="provision"
            >
              <template v-if="provisioning">
                <span class="spinner" />
                Provisioning…
              </template>
              <template v-else>
                Provision {{ form.count }} × {{ selectedGpu.name }} · {{ fmtUsd(hourlyGpuRate) }}/hr
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square">
                  <path d="M3 8h10M9 4l4 4-4 4" />
                </svg>
              </template>
            </button>
            <div v-if="provisionError" class="provision-error">{{ provisionError }}</div>
            <div class="provision-note">
              Charges start the moment SSH is ready · typically ~90 seconds. Stop the instance any time from the dashboard or CLI.
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<style scoped>
.provision-page {
  --bd-soft: rgba(255, 255, 255, 0.05);
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
.neg-text { color: var(--neg); }
.pos { color: var(--pos); }

/* ============================================================
   Sub-topbar
   ============================================================ */
.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
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
.crumbs a { color: var(--text-2); text-decoration: none; }
.crumbs a:hover { color: var(--text); }
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .strong { color: var(--text); }
.subbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 14px;
}
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
  cursor: pointer;
  transition: background 120ms;
}
.cli-inline:hover { background: var(--hover); }
.cli-inline.copied { border-color: rgba(25, 195, 125, 0.45); }
.cli-prompt { color: var(--brand); font-weight: 600; }
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
.cli-inline.copied .cli-copy { color: var(--pos); border-color: rgba(25, 195, 125, 0.45); }
.exit-link {
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  text-decoration: none;
  letter-spacing: -0.005em;
}
.exit-link:hover { color: var(--text); }

/* ============================================================
   Page
   ============================================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 32px 28px 80px;
}
.page-head {
  margin-bottom: 24px;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--border);
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
}
.page-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0;
  max-width: 700px;
  line-height: 1.55;
}
.inline-cli {
  font-family: var(--font-mono);
  font-size: 12.5px;
  background: var(--elevated);
  border: 1px solid var(--border);
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  color: var(--text);
  margin: 0 3px;
  cursor: pointer;
  letter-spacing: 0.02em;
}
.inline-cli:hover { background: var(--hover); }

/* ============================================================
   Three columns
   ============================================================ */
.cols {
  display: grid;
  grid-template-columns: 1fr 1fr 360px;
  gap: 24px;
  align-items: start;
}
.col {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 18px 22px 22px;
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.col-head {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--bd-soft);
  margin-bottom: 4px;
}
.col-num {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: var(--text-3);
}
.col-title {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
}

/* ============================================================
   Section primitives (used across columns)
   ============================================================ */
.section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.section-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
}
.section-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-2);
  font-weight: 600;
}
.section-k.inline {
  font-size: 10px;
  flex-shrink: 0;
  align-self: center;
}
.section-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}
.section-meta .big {
  font-family: var(--font-mono);
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.01em;
}
.section-meta a {
  color: var(--text-2);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
}
.section-meta a:hover { color: var(--text); }
.section-hint {
  font-size: 11.5px;
  color: var(--text-3);
  line-height: 1.5;
}
.section-hint a {
  color: var(--text);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
}
.section-hint a:hover { color: var(--brand); border-bottom-color: var(--brand); }
.section-hint.inline { font-size: 11px; color: var(--text-3); align-self: center; }
.section-hint.mono { font-family: var(--font-mono); letter-spacing: 0.04em; }

/* ============================================================
   GPU radio cards
   ============================================================ */
.gpu-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gpu-card {
  position: relative;
  display: grid;
  grid-template-columns: 16px 1fr;
  gap: 12px;
  padding: 12px 14px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 120ms, border-color 120ms;
}
.gpu-card:hover { background: var(--hover); border-color: var(--border-strong); }
.gpu-card.active {
  background: rgba(200, 242, 92, 0.04);
  border-color: var(--brand);
}
.gpu-card.disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.gpu-card.disabled:hover {
  background: var(--canvas);
  border-color: var(--border);
}
.gpu-card input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}
.gpu-radio {
  width: 14px;
  height: 14px;
  border: 1px solid var(--border-strong);
  border-radius: 50%;
  background: var(--canvas);
  align-self: center;
  position: relative;
}
.gpu-card.active .gpu-radio { border-color: var(--brand); }
.gpu-card.active .gpu-radio::after {
  content: '';
  position: absolute;
  inset: 3px;
  background: var(--brand);
  border-radius: 50%;
}
.gpu-line {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.gpu-name {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text);
}
.gpu-spec {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.gpu-rate {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text);
  font-weight: 600;
}
.gpu-sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-2);
  display: flex;
  align-items: center;
  gap: 10px;
}
.gpu-tag {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  padding: 2px 7px;
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text-3);
}
.gpu-link {
  color: var(--text);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
  font-size: 12px;
}
.gpu-link:hover { color: var(--brand); border-bottom-color: var(--brand); }

/* ============================================================
   Count presets + slider
   ============================================================ */
.count-presets {
  display: grid;
  grid-template-columns: repeat(6, 1fr) 1fr;
  gap: 6px;
}
.preset {
  height: 32px;
  background: var(--canvas);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  font-weight: 600;
  border-radius: var(--radius-sm);
  cursor: pointer;
  letter-spacing: 0.04em;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.preset:hover { background: var(--hover); color: var(--text); }
.preset.active {
  background: rgba(200, 242, 92, 0.10);
  color: var(--brand);
  border-color: var(--brand);
}
.count-input {
  height: 32px;
  width: 100%;
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  color: var(--text);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  font-weight: 600;
  text-align: center;
  border-radius: var(--radius-sm);
  outline: none;
  -webkit-appearance: none;
  appearance: none;
  -moz-appearance: textfield;
}
.count-input::-webkit-outer-spin-button,
.count-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.count-input:focus { border-color: var(--accent); box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.20); }

/* Range slider */
.slider {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  background: transparent;
  margin: 0;
  cursor: pointer;
}
.slider::-webkit-slider-runnable-track {
  height: 4px;
  background: var(--canvas);
  border-radius: 2px;
  border: 1px solid var(--border);
}
.slider::-moz-range-track {
  height: 4px;
  background: var(--canvas);
  border-radius: 2px;
  border: 1px solid var(--border);
}
.slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  background: var(--brand);
  border: 2px solid var(--canvas);
  border-radius: 50%;
  margin-top: -6px;
  box-shadow: 0 0 0 1px var(--border-strong);
}
.slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  background: var(--brand);
  border: 2px solid var(--canvas);
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--border-strong);
}
.slider:focus { outline: none; }
.slider-ticks {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.slider-ticks .active { color: var(--brand); }

/* ============================================================
   Select + input + textarea
   ============================================================ */
.select-wrap { position: relative; }
.select-wrap.inline { flex: 1; }
.select {
  height: 38px;
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 36px 0 12px;
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  width: 100%;
  outline: none;
  -webkit-appearance: none;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg width='10' height='6' viewBox='0 0 10 6' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1L5 5L9 1' stroke='%239A9A95' stroke-width='1.4' fill='none' stroke-linecap='square'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  cursor: pointer;
}
.select.small { height: 32px; font-size: 12px; }
.select:hover { border-color: rgba(255, 255, 255, 0.24); }
.select:focus { border-color: var(--accent); box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.20); }

.input, .textarea {
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  width: 100%;
  height: 38px;
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
}
.textarea {
  height: auto;
  padding: 10px 12px;
  resize: vertical;
  min-height: 80px;
  line-height: 1.5;
  word-break: break-all;
}
.input:focus, .textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.20);
}
.mono-input {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  letter-spacing: 0.02em;
}
.extra-field { margin-top: 4px; }

/* ============================================================
   Toggle row (Networking)
   ============================================================ */
.toggle-row {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
}
.toggle {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 14px;
  align-items: center;
  padding: 12px 14px;
  cursor: pointer;
  border-bottom: 1px solid var(--bd-soft);
  position: relative;
}
.toggle:last-child { border-bottom: 0; }
.toggle input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}
.toggle-track {
  width: 30px;
  height: 16px;
  background: var(--border-strong);
  border-radius: 999px;
  position: relative;
  transition: background 120ms;
  flex-shrink: 0;
}
.toggle-thumb {
  position: absolute;
  left: 2px;
  top: 2px;
  width: 12px;
  height: 12px;
  background: var(--text-2);
  border-radius: 50%;
  transition: transform 160ms cubic-bezier(0.2, 0, 0, 1), background 120ms;
}
.toggle input:checked ~ .toggle-track {
  background: var(--brand);
}
.toggle input:checked ~ .toggle-track .toggle-thumb {
  transform: translateX(14px);
  background: var(--canvas);
}
.toggle input:disabled ~ .toggle-track { opacity: 0.6; cursor: not-allowed; }
.toggle-label { display: flex; flex-direction: column; gap: 2px; }
.t-title {
  font-family: var(--font-sans);
  font-weight: 500;
  font-size: 13px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.t-sub {
  font-size: 12px;
  color: var(--text-3);
}
.locked-tag {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.10em;
  color: var(--accent);
  background: rgba(74, 144, 226, 0.10);
  border: 1px solid rgba(74, 144, 226, 0.25);
  padding: 1px 6px;
  border-radius: 2px;
  text-transform: uppercase;
}

.vpc-row {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  align-items: center;
  margin-top: 12px;
}
.vpc-row .section-hint.inline {
  text-align: right;
}

/* ============================================================
   Cost preview column (sticky)
   ============================================================ */
.cost-col {
  position: sticky;
  top: 16px;
  padding: 0;
  background: transparent;
  border: 0;
  align-self: start;
}
.cost-card {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 18px 22px 22px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.cost-card .col-head { padding-bottom: 12px; margin-bottom: 0; }

.cost-section { display: flex; flex-direction: column; gap: 8px; }
.cs-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}

.recap {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.recap li {
  display: grid;
  grid-template-columns: 80px 1fr;
  gap: 10px;
  align-items: baseline;
  padding: 4px 0;
  font-size: 12px;
  border-bottom: 1px dashed var(--bd-soft);
}
.recap li:last-child { border-bottom: 0; }
.r-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.r-v {
  font-family: var(--font-sans);
  color: var(--text);
  font-size: 12px;
}
.r-v.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.cb-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 4px 0;
  font-size: 12px;
}
.cb-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.cb-v {
  color: var(--text);
  font-family: var(--font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.cb-v.pos { color: var(--pos); font-weight: 500; }
.cb-v strong { color: var(--text); font-weight: 600; }
.cb-divider {
  height: 1px;
  background: var(--bd-soft);
  margin: 4px 0;
}
.cb-row.major .cb-v {
  font-size: 13px;
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}
.cb-row.major .plus { color: var(--text-3); font-size: 11px; }

.estimates {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1px;
  background: var(--border);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.est {
  padding: 12px 14px;
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.est-k {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.est-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.01em;
}

.budget {
  border: 1px solid var(--bd-soft);
  background: var(--canvas);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.budget.over {
  border-color: rgba(239, 68, 68, 0.30);
  background: rgba(239, 68, 68, 0.04);
}
.budget-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 12px;
}
.b-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.b-v { font-family: var(--font-mono); color: var(--text); font-size: 12px; }
.budget-warn {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px dashed rgba(239, 68, 68, 0.30);
  font-size: 11.5px;
  color: var(--text);
  line-height: 1.5;
}
.budget-warn a {
  color: var(--neg);
  text-decoration: none;
  border-bottom: 1px solid var(--neg);
}

/* CLI parity disclosure */
.cli-block {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.cli-block summary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
  cursor: pointer;
  list-style: none;
  user-select: none;
}
.cli-block summary::-webkit-details-marker { display: none; }
.cli-block summary svg { width: 12px; height: 12px; color: var(--text-3); }
.cli-block summary:hover { color: var(--text); }
.cli-block summary:hover svg { color: var(--text); }
.cli-copy-btn {
  margin-left: auto;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  padding: 2px 7px;
  border-radius: 2px;
  cursor: pointer;
  text-transform: uppercase;
}
.cli-copy-btn:hover { color: var(--text); border-color: var(--border-strong); }
.cli-copy-btn.copied {
  color: var(--pos);
  border-color: rgba(25, 195, 125, 0.40);
}
.cli-pre {
  margin: 0;
  padding: 10px 14px 12px;
  background: rgba(0, 0, 0, 0.20);
  border-top: 1px solid var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text);
  letter-spacing: 0.02em;
  line-height: 1.55;
  white-space: pre;
  overflow-x: auto;
}

/* ============================================================
   Provision button
   ============================================================ */
.provision-btn {
  height: 52px;
  width: 100%;
  background: var(--brand);
  color: var(--canvas);
  border: 0;
  border-radius: var(--radius-sm);
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 700;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  transition: background 120ms, filter 120ms;
}
.provision-btn:hover:enabled { background: var(--brand-hov); }
.provision-btn:disabled {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-3);
  cursor: not-allowed;
}
.provision-btn svg { width: 16px; height: 16px; }
@keyframes spin { to { transform: rotate(360deg); } }
.spinner {
  width: 14px;
  height: 14px;
  border: 1.6px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}
.provision-note {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  line-height: 1.55;
  text-align: center;
}
.provision-error {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--neg);
  background: var(--neg-soft);
  border: 1px solid var(--neg);
  border-radius: 6px;
  padding: 8px 10px;
  text-align: center;
}

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1200px) {
  .cols { grid-template-columns: 1fr 1fr; }
  .cost-col { grid-column: span 2; position: static; }
}
@media (max-width: 800px) {
  .cols { grid-template-columns: 1fr; }
  .cost-col { grid-column: auto; }
  .count-presets { grid-template-columns: repeat(3, 1fr); }
  .vpc-row { grid-template-columns: 1fr; }
  .vpc-row .section-hint.inline { text-align: left; }
}
</style>
