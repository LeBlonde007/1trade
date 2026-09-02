<script setup lang="ts">
/**
 * /datacenter — Datacenter Partner Dashboard.
 *
 * Internal-only in v1; partner-facing in v1.5. Dark mode, operational
 * surface. Standalone chrome — this is not the trader app.
 *
 * Sections:
 *   1. Partner header (Czech Data Center 1)
 *   2. Capacity overview (5 stats)
 *   3. 24h utilization chart (Chart.js, live)
 *   4. Revenue this month + payout split
 *   5. Payout history table
 *   6. Capacity status grid (1024 GPUs, 16 racks × 64)
 */
import { Chart, type ChartDataset } from 'chart.js/auto'

definePageMeta({ layout: false })
useHead({
  title: 'Czech DC1 · Partner Dashboard — 1Trade',
  htmlAttrs: { 'data-theme': 'dark' },
})

// =====================================================
// Constants — partner identity, contract terms
// =====================================================
const PARTNER = {
  name: 'Czech Data Center 1',
  code: 'EX-DC-PRG-01',
  region: 'PRG · Prague, CZ',
  activeSince: '2026-04-12',
  contact: 'partner-ops@czdc1.com',
  contractTerms: '#contract-terms',
  totalCapacity: 1024,
  share: 0.70,
  fee: 0.30,
  floorPrice: 2.50,
  avgPrice7d: 2.92,
  utilized: 847,
  maintenance: 8,
  offline: 4,
}

// Derived: 847 + 8 + 4 = 859 unavailable. Available = 1024 - 859 = 165.
const AVAILABLE = PARTNER.totalCapacity - PARTNER.utilized - PARTNER.maintenance - PARTNER.offline

// =====================================================
// Revenue numbers
// =====================================================
const REVENUE_TOTAL    = 1_234_500
const REVENUE_PARTNER  = Math.round(REVENUE_TOTAL * PARTNER.share)
const REVENUE_FEE      = REVENUE_TOTAL - REVENUE_PARTNER
const NEXT_PAYOUT_DATE = '2026-06-01'

// =====================================================
// Payout history
// =====================================================
interface Payout {
  period: string
  capacity: number     // GPU-hours
  gross: number
  fee: number
  partner: number
  status: 'paid' | 'pending' | 'projected'
  txHash?: string
}
const PAYOUTS: Payout[] = [
  { period: 'Apr 2026', capacity: 562_400, gross: 1_642_400, fee: 492_720, partner: 1_149_680, status: 'paid',      txHash: '0x7a3c91…f042' },
  { period: 'Mar 2026', capacity: 489_200, gross: 1_428_464, fee: 428_539, partner:   999_925, status: 'paid',      txHash: '0x4b8821…a107' },
  { period: 'Feb 2026', capacity: 421_800, gross: 1_231_656, fee: 369_497, partner:   862_159, status: 'paid',      txHash: '0x12fa55…b3d9' },
  { period: 'Jan 2026', capacity: 368_400, gross: 1_039_944, fee: 311_983, partner:   727_961, status: 'paid',      txHash: '0x9e0d11…2c4a' },
  { period: 'May 2026', capacity: 421_080, gross: 1_234_500, fee:  370_350, partner:  864_150, status: 'projected' },
]

// =====================================================
// Formatters
// =====================================================
function fmtInt(n: number) { return n.toLocaleString('en-US') }
function fmtUsd(n: number, dp = 0) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtUsd2(n: number) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// =====================================================
// 24h utilization series (hourly, live)
// =====================================================
interface UtilPoint { hour: string; pct: number }

function seedUtilization(): UtilPoint[] {
  // Build last 24 hours hourly with realistic shape:
  //   night (0-6 UTC): 70-78%, ramp from 6 to 12 to 88-92%, peak 12-18 around 90%, taper.
  const now = new Date()
  const out: UtilPoint[] = []
  for (let i = 23; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 3600_000)
    const hh = d.getUTCHours()
    let base = 78
    if (hh >= 0 && hh < 5)  base = 72
    if (hh >= 5 && hh < 9)  base = 76 + (hh - 5) * 3
    if (hh >= 9 && hh < 14) base = 86 + (Math.sin((hh - 9) * 0.6) * 3)
    if (hh >= 14 && hh < 19) base = 90
    if (hh >= 19) base = 86 - (hh - 19) * 2.2
    const noise = (Math.random() - 0.5) * 2.4
    const pct = Math.max(60, Math.min(96, base + noise))
    const label = String(hh).padStart(2, '0') + ':00'
    out.push({ hour: label, pct })
  }
  return out
}
const utilization = ref<UtilPoint[]>(seedUtilization())

const utilNow = computed(() => utilization.value[utilization.value.length - 1]?.pct ?? 83)
const utilPeak = computed(() => Math.max(...utilization.value.map(p => p.pct)))
const utilTrough = computed(() => Math.min(...utilization.value.map(p => p.pct)))

// =====================================================
// Capacity grid — 16 racks × 64 GPUs (8 × 8)
// =====================================================
type CellState = 'in_use' | 'available' | 'maintenance' | 'offline'

interface Rack {
  id: string          // R01..R16
  cells: CellState[]  // 64 cells each
}

function buildRacks(): Rack[] {
  // Allocate: 847 in-use, 165 available, 8 maintenance, 4 offline
  const pool: CellState[] = []
  for (let i = 0; i < PARTNER.utilized; i++) pool.push('in_use')
  for (let i = 0; i < AVAILABLE; i++) pool.push('available')
  for (let i = 0; i < PARTNER.maintenance; i++) pool.push('maintenance')
  for (let i = 0; i < PARTNER.offline; i++) pool.push('offline')
  // Shuffle but cluster a bit: keep maintenance/offline rare and scattered.
  // For visual realism, sort so each rack is dense at top with in-use, with available cells trailing.
  // Simpler: shuffle then partition into 16 racks of 64.
  for (let i = pool.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[pool[i], pool[j]] = [pool[j]!, pool[i]!]
  }
  const racks: Rack[] = []
  for (let r = 0; r < 16; r++) {
    racks.push({
      id: 'R' + String(r + 1).padStart(2, '0'),
      cells: pool.slice(r * 64, (r + 1) * 64),
    })
  }
  return racks
}
const racks = ref<Rack[]>(buildRacks())

const cellCounts = computed(() => {
  const c = { in_use: 0, available: 0, maintenance: 0, offline: 0 }
  for (const r of racks.value) {
    for (const cell of r.cells) {
      c[cell]++
    }
  }
  return c
})

// =====================================================
// Live tick — utilization drifts; the grid occasionally flips an
// in_use ↔ available cell to simulate a job ending/starting.
// =====================================================
const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart<'line', number[]> | null = null
let utilTimer: ReturnType<typeof setInterval> | null = null
let gridTimer: ReturnType<typeof setInterval> | null = null
const clock = ref('—')

function tickClock() {
  const d = new Date()
  clock.value = d.toLocaleTimeString('en-GB', { hour12: false, timeZone: 'UTC' }) + ' UTC'
}
let clockTimer: ReturnType<typeof setInterval> | null = null

function initChart() {
  if (!chartCanvas.value) return
  const ctx = chartCanvas.value.getContext('2d')!
  const grad = ctx.createLinearGradient(0, 0, 0, 240)
  grad.addColorStop(0, 'rgba(200,242,92,0.18)')
  grad.addColorStop(1, 'rgba(200,242,92,0.00)')

  chart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: utilization.value.map(p => p.hour),
      datasets: [{
        label: 'Utilization',
        data: utilization.value.map(p => p.pct),
        borderColor: '#D4AF37',
        borderWidth: 1.6,
        backgroundColor: grad,
        fill: true,
        tension: 0.32,
        pointRadius: 0,
        pointHoverRadius: 4,
        pointHoverBorderColor: '#D4AF37',
        pointHoverBackgroundColor: '#121212',
        pointHoverBorderWidth: 1.5,
      } as ChartDataset<'line', number[]>],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: '#1A1A1A',
          borderColor: 'rgba(255,255,255,0.16)',
          borderWidth: 1,
          padding: 10,
          cornerRadius: 2,
          titleColor: '#A8A196',
          bodyColor: '#E8E2D6',
          titleFont: { family: "'JetBrains Mono', monospace", size: 10, weight: 'bold' },
          bodyFont:  { family: "'JetBrains Mono', monospace", size: 12 },
          displayColors: false,
          callbacks: {
            title: items => 'Hour · ' + items[0]!.label,
            label: item => 'Utilization  ' + (item.parsed.y as number).toFixed(1) + '%',
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          border: { display: false },
          ticks: {
            color: '#7E786C',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 12,
          },
        },
        y: {
          min: 50,
          max: 100,
          position: 'right',
          grid: { color: 'rgba(255,255,255,0.05)' },
          border: { display: false },
          ticks: {
            color: '#7E786C',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            stepSize: 10,
            callback: v => v + '%',
            padding: 8,
          },
        },
      },
    },
  })
}

function pushNewUtilPoint() {
  // Tweak the latest hour's value to simulate a live tick.
  const arr = [...utilization.value]
  const last = arr[arr.length - 1]!
  const drift = (85 - last.pct) * 0.12
  const noise = (Math.random() - 0.5) * 1.6
  const next = Math.max(60, Math.min(96, last.pct + drift + noise))
  arr[arr.length - 1] = { ...last, pct: next }
  utilization.value = arr
  if (chart) {
    chart.data.datasets[0]!.data = arr.map(p => p.pct) as never
    chart.update('none')
  }
}

function flipOneCell() {
  // Find a random rack + cell, flip between in_use and available occasionally.
  const next = racks.value.map(r => ({ ...r, cells: [...r.cells] }))
  const r = Math.floor(Math.random() * next.length)
  const c = Math.floor(Math.random() * next[r]!.cells.length)
  const cur = next[r]!.cells[c]!
  if (cur === 'in_use') next[r]!.cells[c] = 'available'
  else if (cur === 'available') next[r]!.cells[c] = 'in_use'
  // leave maintenance/offline alone for stability
  racks.value = next
}

onMounted(() => {
  tickClock()
  clockTimer = setInterval(tickClock, 1000)
  initChart()
  utilTimer = setInterval(pushNewUtilPoint, 3000)
  gridTimer = setInterval(flipOneCell, 1800)
})
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
  if (utilTimer) clearInterval(utilTimer)
  if (gridTimer) clearInterval(gridTimer)
  chart?.destroy()
})

// =====================================================
// Capacity status legend
// =====================================================
const LEGEND: Array<{ key: CellState; label: string; color: string; count: () => number }> = [
  { key: 'in_use',      label: 'In use · paid',  color: 'var(--pos)',  count: () => cellCounts.value.in_use },
  { key: 'available',   label: 'Available',      color: 'var(--info)', count: () => cellCounts.value.available },
  { key: 'maintenance', label: 'Maintenance',    color: 'var(--warn)', count: () => cellCounts.value.maintenance },
  { key: 'offline',     label: 'Offline',        color: 'var(--neg)',  count: () => cellCounts.value.offline },
]
</script>

<template>
  <div class="dc-shell" data-theme="dark">
    <!-- ============ Top chrome ============ -->
    <header class="topbar">
      <a href="#" class="brand">
        <span class="brand-mark" />
        1TRADE
      </a>
      <div class="env-row">
        <span class="env-pill">PARTNER · INTERNAL</span>
        <span class="env-sep">·</span>
        <span class="env-meta">{{ PARTNER.code }} · {{ PARTNER.region }}</span>
      </div>
      <div class="top-right">
        <span class="clock-live">
          <span class="pulse" />
          <span>Live · {{ clock }}</span>
        </span>
        <a :href="`mailto:${PARTNER.contact}`" class="top-link">{{ PARTNER.contact }}</a>
        <a :href="PARTNER.contractTerms" class="top-link">Contract terms →</a>
        <NuxtLink to="/datacenter/register" class="top-cta">+ Register partner</NuxtLink>
      </div>
    </header>

    <main class="page">
      <!-- ============ Partner header ============ -->
      <section class="partner-head">
        <div class="ph-left">
          <div class="eyebrow"><span class="dot" /> Datacenter partner · operations</div>
          <h1 class="ph-title">{{ PARTNER.name }}</h1>
          <div class="ph-meta">
            <span class="active-pill">
              <span class="pulse" />
              Active since {{ PARTNER.activeSince }}
            </span>
            <span class="meta-sep">·</span>
            <span>Cluster <strong>{{ PARTNER.code }}</strong></span>
            <span class="meta-sep">·</span>
            <span>{{ PARTNER.region }}</span>
          </div>
        </div>
        <div class="ph-right">
          <div class="kv">
            <span class="kv-k">Revenue share</span>
            <span class="kv-v">{{ (PARTNER.share * 100).toFixed(0) }}% / {{ (PARTNER.fee * 100).toFixed(0) }}%</span>
          </div>
          <div class="kv">
            <span class="kv-k">Pricing floor</span>
            <span class="kv-v">${{ PARTNER.floorPrice.toFixed(2) }} / GPU-hr</span>
          </div>
          <div class="kv">
            <span class="kv-k">Settlement</span>
            <span class="kv-v">Monthly · USD</span>
          </div>
        </div>
      </section>

      <!-- ============ Capacity overview stats ============ -->
      <section class="stats-row">
        <div class="stat featured">
          <div class="stat-lbl">— Contributed capacity</div>
          <div class="stat-val">{{ fmtInt(PARTNER.totalCapacity) }}</div>
          <div class="stat-sub">H100 GPUs · cluster total</div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Currently utilized</div>
          <div class="stat-val pos-num">{{ fmtInt(cellCounts.in_use) }}</div>
          <div class="stat-sub">
            <span class="pos">{{ ((cellCounts.in_use / PARTNER.totalCapacity) * 100).toFixed(1) }}%</span>
            · live
          </div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Available</div>
          <div class="stat-val">{{ fmtInt(cellCounts.available) }}</div>
          <div class="stat-sub">{{ ((cellCounts.available / PARTNER.totalCapacity) * 100).toFixed(1) }}% idle</div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Pricing floor</div>
          <div class="stat-val">${{ PARTNER.floorPrice.toFixed(2) }}</div>
          <div class="stat-sub">Your minimum · GPU-hr</div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Avg sale price · 7d</div>
          <div class="stat-val">${{ PARTNER.avgPrice7d.toFixed(2) }}</div>
          <div class="stat-sub">
            <span class="pos">+{{ (((PARTNER.avgPrice7d / PARTNER.floorPrice) - 1) * 100).toFixed(1) }}%</span>
            vs floor
          </div>
        </div>
      </section>

      <!-- ============ Utilization + revenue row ============ -->
      <section class="util-row">
        <div class="card util-card">
          <div class="card-head">
            <div>
              <span class="eyebrow card-eyebrow"><span class="dot" /> Utilization · 24h</span>
              <h2 class="card-ttl">
                {{ utilNow.toFixed(1) }}<span class="ttl-unit">%</span>
                <span class="util-meta">
                  Peak <strong>{{ utilPeak.toFixed(1) }}%</strong>
                  · Trough <strong>{{ utilTrough.toFixed(1) }}%</strong>
                </span>
              </h2>
            </div>
            <div class="card-head-right">
              <span class="live-chip">
                <span class="pulse" />
                LIVE
              </span>
            </div>
          </div>
          <div class="util-canvas-wrap">
            <canvas ref="chartCanvas" />
          </div>
          <div class="util-foot">
            <span>Hourly · last 24h · UTC</span>
            <a href="#">Download CSV →</a>
          </div>
        </div>

        <div class="card rev-card">
          <div class="card-head rev-head">
            <span class="eyebrow"><span class="dot" /> Revenue this month · May 2026</span>
          </div>
          <div class="rev-big">
            {{ fmtUsd(REVENUE_TOTAL) }}
            <span class="rev-unit">USD</span>
          </div>
          <div class="rev-split">
            <div class="rev-row">
              <span class="rev-k">Your share · {{ (PARTNER.share * 100).toFixed(0) }}%</span>
              <span class="rev-v pos">{{ fmtUsd(REVENUE_PARTNER) }}</span>
            </div>
            <div class="rev-bar">
              <div class="rev-bar-fill" :style="{ width: (PARTNER.share * 100) + '%' }" />
            </div>
            <div class="rev-row">
              <span class="rev-k">1Trade fee · {{ (PARTNER.fee * 100).toFixed(0) }}%</span>
              <span class="rev-v">{{ fmtUsd(REVENUE_FEE) }}</span>
            </div>
          </div>
          <div class="rev-foot">
            <div class="rev-foot-row">
              <span class="lbl">— Next payout</span>
              <span class="val">{{ NEXT_PAYOUT_DATE }}</span>
            </div>
            <div class="rev-foot-row">
              <span class="lbl">— Settlement rail</span>
              <span class="val">USD wire · T+1</span>
            </div>
            <div class="rev-foot-row">
              <span class="lbl">— Statement</span>
              <a href="#" class="val link">Download May 2026 →</a>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ Payout history ============ -->
      <section class="card payout-card">
        <div class="card-head">
          <span class="eyebrow"><span class="dot" /> Payout history</span>
          <div class="card-head-right">
            <a href="#" class="head-link">All statements →</a>
          </div>
        </div>
        <table class="payout-table">
          <thead>
            <tr>
              <th class="left">Period</th>
              <th>Capacity sold</th>
              <th>Gross</th>
              <th>1Trade fee</th>
              <th>Your payout</th>
              <th class="left">Settlement</th>
              <th class="center">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in PAYOUTS" :key="p.period" :class="{ projected: p.status === 'projected' }">
              <td class="left period">{{ p.period }}</td>
              <td>{{ fmtInt(p.capacity) }} <span class="dim">GPU-hr</span></td>
              <td>{{ fmtUsd(p.gross) }}</td>
              <td>{{ fmtUsd(p.fee) }}</td>
              <td class="emph">{{ fmtUsd(p.partner) }}</td>
              <td class="left mono dim">
                <template v-if="p.txHash">{{ p.txHash }}</template>
                <template v-else>— pending</template>
              </td>
              <td class="center">
                <span class="status-tag" :class="p.status">
                  <span class="dot" />
                  <template v-if="p.status === 'paid'">Paid</template>
                  <template v-else-if="p.status === 'pending'">Pending</template>
                  <template v-else>Projected</template>
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- ============ Capacity status grid ============ -->
      <section class="card grid-card">
        <div class="card-head">
          <div>
            <span class="eyebrow"><span class="dot" /> Capacity status · live</span>
            <h2 class="card-ttl">
              {{ fmtInt(PARTNER.totalCapacity) }} GPUs
              <span class="ttl-unit">· 16 racks × 64</span>
            </h2>
          </div>
          <div class="legend">
            <span v-for="l in LEGEND" :key="l.key" class="legend-item">
              <span class="legend-sw" :style="{ background: l.color }" />
              <span class="legend-label">{{ l.label }}</span>
              <span class="legend-count">{{ fmtInt(l.count()) }}</span>
            </span>
          </div>
        </div>

        <div class="rack-grid">
          <div v-for="rack in racks" :key="rack.id" class="rack">
            <div class="rack-head">
              <span class="rack-id">{{ rack.id }}</span>
              <span class="rack-stat">
                {{ rack.cells.filter(c => c === 'in_use').length }}/64
              </span>
            </div>
            <div class="rack-cells">
              <span
                v-for="(c, idx) in rack.cells"
                :key="idx"
                class="cell"
                :class="c"
                :title="`${rack.id}-${String(idx + 1).padStart(2, '0')} · ${c.replace('_', ' ')}`"
              />
            </div>
          </div>
        </div>

        <div class="grid-foot">
          <span>{{ PARTNER.code }} · {{ PARTNER.region }}</span>
          <span class="dim">Refreshes every 1.8s · grid generated from current allocation table</span>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.dc-shell {
  --canvas:        #0A0A0A;
  --elevated:      #121212;
  --overlay:       #1A1A1A;
  --hover:         #232323;
  --text:          #E8E2D6;
  --text-2:        #A8A196;
  --text-3:        #7E786C;
  --text-4:        rgba(255, 255, 255, 0.18);
  --border:        rgba(255, 255, 255, 0.08);
  --border-strong: rgba(255, 255, 255, 0.16);
  --border-soft:   rgba(255, 255, 255, 0.05);

  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.5;
  font-feature-settings: 'tnum' on, 'ss01' on;
  -webkit-font-smoothing: antialiased;
}

.pos { color: var(--pos); }
.neg { color: var(--neg); }
.dim { color: var(--text-3); }
.pos-num { color: var(--text); }

/* ============================================================
   Top chrome
   ============================================================ */
.topbar {
  height: 48px;
  background: var(--canvas);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
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
  border-right: 1px solid var(--border);
  padding-right: 24px;
  height: 32px;
}
.brand-mark {
  width: 8.89px;
  height: 20.6px;
  display: inline-block;
  flex: none;
  background: var(--brand);
  -webkit-mask: url('/brand/mark.svg') center / contain no-repeat;
  mask: url('/brand/mark.svg') center / contain no-repeat;
}
.env-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}
.env-pill {
  padding: 3px 8px;
  background: var(--info);
  color: #fff;
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  border-radius: var(--radius-sm);
  opacity: 0.95;
}
.env-sep { color: var(--text-4); }
.env-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-2);
}
.top-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 18px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.clock-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-2);
}
.clock-live .pulse {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}
@keyframes pulse {
  0%   { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0.6); }
  70%  { box-shadow: 0 0 0 6px rgba(25, 195, 125, 0); }
  100% { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0); }
}
.top-link {
  color: var(--text-2);
  text-decoration: none;
  font-family: var(--font-sans);
  font-size: 12px;
  letter-spacing: 0;
}
.top-link:hover { color: var(--text); }
.top-cta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--brand);
  color: #0E0E0E;
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: -0.005em;
  border-radius: var(--radius-sm);
  text-decoration: none;
  transition: background 120ms;
}
.top-cta:hover { background: var(--brand-hov); }

/* ============================================================
   Page
   ============================================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 32px 40px 80px;
}

/* Partner header */
.partner-head {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 32px;
  padding-bottom: 28px;
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
}
.eyebrow .dot {
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.ph-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 36px;
  letter-spacing: -0.025em;
  line-height: 1.05;
  margin: 10px 0 8px;
  color: var(--text);
}
.ph-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.ph-meta strong { color: var(--text); font-weight: 500; }
.ph-meta .meta-sep { color: var(--text-4); }
.active-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  background: rgba(25, 195, 125, 0.10);
  color: var(--pos);
  border: 1px solid rgba(25, 195, 125, 0.25);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.active-pill .pulse {
  width: 5px;
  height: 5px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}

.ph-right {
  display: flex;
  gap: 32px;
  align-items: flex-end;
}
.kv {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}
.kv-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.kv-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 14px;
  color: var(--text);
  font-weight: 500;
}

/* ============================================================
   Stats row
   ============================================================ */
.stats-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr 1fr 1fr;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  margin-bottom: 24px;
}
.stat {
  padding: 20px 22px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-right: 1px solid var(--border);
}
.stat:last-child { border-right: none; }
.stat.featured {
  background: linear-gradient(180deg, rgba(200, 242, 92, 0.04), transparent 80%);
}
.stat.featured::before {
  content: '';
  display: block;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--brand);
}
.stat.featured { position: relative; }

.stat-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.stat-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 30px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--text);
}
.stat.featured .stat-val { color: var(--text); }
.stat-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
}
.stat-sub .pos { color: var(--pos); }

/* ============================================================
   Util + revenue row
   ============================================================ */
.util-row {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 24px;
  margin-bottom: 24px;
}
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-soft);
  gap: 16px;
}
.card-eyebrow { margin-bottom: 0; }
.card-ttl {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 6px 0 0;
  color: var(--text);
  line-height: 1;
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
}
.card-ttl .ttl-unit {
  font-family: var(--font-mono);
  font-size: 14px;
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0;
}
.util-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.04em;
  margin-left: 14px;
}
.util-meta strong { color: var(--text); font-weight: 500; }

.live-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--pos);
  background: rgba(25, 195, 125, 0.10);
  border: 1px solid rgba(25, 195, 125, 0.25);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
}
.live-chip .pulse {
  width: 5px;
  height: 5px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse 2.4s infinite;
}
.card-head-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.util-canvas-wrap {
  position: relative;
  height: 240px;
  padding: 14px 20px 4px;
}
.util-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px 14px;
  border-top: 1px solid var(--border-soft);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.util-foot a {
  color: var(--text);
  text-decoration: none;
}
.util-foot a:hover { text-decoration: underline; }

/* Revenue card */
.rev-card { display: flex; flex-direction: column; }
.rev-head {
  border-bottom: 1px solid var(--border-soft);
}
.rev-big {
  padding: 18px 22px 8px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 40px;
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1;
  color: var(--text);
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.rev-unit {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.16em;
}
.rev-split { padding: 4px 22px 8px; }
.rev-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 6px 0;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
}
.rev-k {
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.rev-v { color: var(--text); }
.rev-v.pos { color: var(--pos); }
.rev-bar {
  height: 8px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
  margin: 4px 0;
  overflow: hidden;
  position: relative;
}
.rev-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--pos), rgba(25, 195, 125, 0.55));
  border-right: 1px solid var(--border-soft);
}

.rev-foot {
  margin-top: auto;
  padding: 16px 22px;
  border-top: 1px solid var(--border-soft);
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.rev-foot-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}
.rev-foot-row .lbl {
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-weight: 600;
}
.rev-foot-row .val {
  color: var(--text);
  font-family: var(--font-mono);
}
.rev-foot-row .val.link {
  color: var(--brand);
  text-decoration: none;
}
.rev-foot-row .val.link:hover { text-decoration: underline; }

/* ============================================================
   Payout history
   ============================================================ */
.payout-card { margin-bottom: 24px; }
.head-link {
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  text-decoration: none;
}
.head-link:hover { color: var(--text); }

.payout-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.payout-table thead th {
  text-align: right;
  padding: 10px 18px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.payout-table th.left,
.payout-table td.left { text-align: left; }
.payout-table th.center,
.payout-table td.center { text-align: center; }

.payout-table tbody td {
  padding: 12px 18px;
  border-bottom: 1px solid var(--border-soft);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
  color: var(--text);
}
.payout-table tbody td.period { color: var(--text); font-weight: 500; }
.payout-table tbody td.mono { font-family: var(--font-mono); }
.payout-table tbody td.emph { color: var(--pos); }
.payout-table tbody tr:hover { background: rgba(255, 255, 255, 0.025); }
.payout-table tbody tr.projected { opacity: 0.7; }

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.status-tag .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.status-tag.paid {
  background: rgba(25, 195, 125, 0.10);
  color: var(--pos);
  border: 1px solid rgba(25, 195, 125, 0.25);
}
.status-tag.pending {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border: 1px solid rgba(245, 158, 11, 0.25);
}
.status-tag.projected {
  background: transparent;
  color: var(--text-3);
  border: 1px dashed var(--border-strong);
}

/* ============================================================
   Capacity grid
   ============================================================ */
.grid-card .card-head {
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}
.legend {
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
  align-items: center;
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}
.legend-sw {
  width: 9px;
  height: 9px;
  display: inline-block;
}
.legend-label { color: var(--text-2); }
.legend-count {
  color: var(--text);
  font-variant-numeric: tabular-nums;
  padding-left: 4px;
  border-left: 1px solid var(--border);
  margin-left: 2px;
}

.rack-grid {
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}
.rack {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px;
}
.rack-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
  margin-bottom: 8px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
}
.rack-id {
  color: var(--text);
  font-weight: 600;
}
.rack-stat {
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}
.rack-cells {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 2px;
}
.cell {
  aspect-ratio: 1;
  border-radius: 1px;
  transition: background 240ms ease-out, transform 80ms;
}
.cell.in_use      { background: var(--pos); }
.cell.available   { background: var(--info); opacity: 0.42; }
.cell.maintenance { background: var(--warn); }
.cell.offline     { background: var(--neg); }
.cell:hover { transform: scale(1.25); cursor: crosshair; }

.grid-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px 14px;
  border-top: 1px solid var(--border-soft);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
}

/* ============================================================
   Responsive (1024px)
   ============================================================ */
@media (max-width: 1200px) {
  .stats-row { grid-template-columns: repeat(3, 1fr); }
  .stat:nth-child(4) { border-right: 1px solid var(--border); }
  .stat:nth-child(3) { border-right: none; }
  .util-row { grid-template-columns: 1fr; }
  .rack-grid { grid-template-columns: repeat(2, 1fr); }
  .ph-right { gap: 18px; }
  .partner-head { grid-template-columns: 1fr; }
}
@media (max-width: 800px) {
  .stats-row { grid-template-columns: 1fr 1fr; }
  .stat { border-right: none; }
  .rack-grid { grid-template-columns: 1fr; }
}
</style>
