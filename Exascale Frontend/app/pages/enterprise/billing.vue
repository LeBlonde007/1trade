<script setup lang="ts">
/**
 * /enterprise/billing — Billing Dashboard (D4)
 *
 * Light-mode enterprise finance surface.
 * Month spend + forecast, breakdowns, cost alerts, invoice history.
 */
import { Chart, type ChartDataset } from 'chart.js/auto'

definePageMeta({ layout: false })
useHead({ title: 'Billing · Walmart Inc. — Exascale', htmlAttrs: { 'data-theme': 'light' } })

const ORG = {
  name: 'Walmart Inc.',
  enterpriseId: 'ent_wmt_8412',
  paymentMethod: 'Wire · Chase ••4421',
}

const PERIOD = {
  label: 'May 2026',
  daysIn: 19,
  daysInMonth: 31,
}

// ---------- Headline stats ----------
const mtdSpend   = 42847.23
const projected  = 67800.00
const budget     = 80000.00
const lastMonth  = 61238.49

const budgetPct      = computed(() => (mtdSpend / budget) * 100)
const projectedPct   = computed(() => (projected / budget) * 100)
const vsLastMonthPct = computed(() => ((projected - lastMonth) / lastMonth) * 100)
const burnPerDay     = computed(() => mtdSpend / PERIOD.daysIn)
const daysToBudget   = computed(() => Math.floor((budget - mtdSpend) / burnPerDay.value))
const status: 'on-track' | 'warning' | 'over' = 'on-track'

const statusCopy = {
  'on-track': { dot: 'pos',  label: 'On track',     hint: 'Projected under budget by $12,200 (15%)' },
  'warning':  { dot: 'warn', label: 'Watch',        hint: 'Projected within 5% of budget' },
  'over':     { dot: 'neg',  label: 'Over budget',  hint: 'Forecast exceeds budget' },
}[status]

// ---------- Breakdown by service ----------
interface Slice { id: string; label: string; amount: number; pct: number; color: string }
const services: Slice[] = [
  { id: 'compute',   label: 'Compute',      amount: 25708.34, pct: 60, color: '#4A90E2' },
  { id: 'inference', label: 'Inference',    amount: 10711.81, pct: 25, color: '#19C37D' },
  { id: 'storage',   label: 'Storage',      amount: 4284.72,  pct: 10, color: '#6B5B95' },
  { id: 'fees',      label: 'Trading fees', amount: 2142.36,  pct: 5,  color: '#F59E0B' },
]

const subAccounts: Slice[] = [
  { id: 'ai-research',   label: 'AI-Research-Team',     amount: 22280.56, pct: 52, color: '#4A90E2' },
  { id: 'cs-ai',         label: 'Customer-Service-AI',  amount: 11997.22, pct: 28, color: '#19C37D' },
  { id: 'marketing-img', label: 'Marketing-Image-Gen',  amount: 5141.67,  pct: 12, color: '#6B5B95' },
  { id: 'inference-dev', label: 'Inference-Dev',        amount: 2142.36,  pct: 5,  color: '#9CA3AF' },
  { id: 'other',         label: 'Other (ad-hoc)',       amount: 1285.42,  pct: 3,  color: '#D1D5DB' },
]

// Build conic-gradient for the services donut
const donutGradient = computed(() => {
  let acc = 0
  const stops = services.map((s) => {
    const start = acc
    acc += s.pct
    return `${s.color} ${start}% ${acc}%`
  })
  return `conic-gradient(${stops.join(', ')})`
})

// ---------- Daily spend (30 days) ----------
// Actual days 1-19; projected dotted days 20-31. Sat/Sun lower (weekend dips).
interface DailyPoint { day: number; dow: string; amount: number; projected?: boolean }
const dailySpend: DailyPoint[] = [
  { day: 1,  dow: 'Fri', amount: 2150 },
  { day: 2,  dow: 'Sat', amount: 1090 },
  { day: 3,  dow: 'Sun', amount: 850  },
  { day: 4,  dow: 'Mon', amount: 2620 },
  { day: 5,  dow: 'Tue', amount: 2840 },
  { day: 6,  dow: 'Wed', amount: 2580 },
  { day: 7,  dow: 'Thu', amount: 2780 },
  { day: 8,  dow: 'Fri', amount: 2540 },
  { day: 9,  dow: 'Sat', amount: 1220 },
  { day: 10, dow: 'Sun', amount: 970  },
  { day: 11, dow: 'Mon', amount: 2950 },
  { day: 12, dow: 'Tue', amount: 3120 },
  { day: 13, dow: 'Wed', amount: 2920 },
  { day: 14, dow: 'Thu', amount: 2850 },
  { day: 15, dow: 'Fri', amount: 2630 },
  { day: 16, dow: 'Sat', amount: 1310 },
  { day: 17, dow: 'Sun', amount: 1180 },
  { day: 18, dow: 'Mon', amount: 2980 },
  { day: 19, dow: 'Tue', amount: 3267 }, // today (MTD partial)
  // ---- projected ----
  { day: 20, dow: 'Wed', amount: 2820, projected: true },
  { day: 21, dow: 'Thu', amount: 2740, projected: true },
  { day: 22, dow: 'Fri', amount: 2510, projected: true },
  { day: 23, dow: 'Sat', amount: 1240, projected: true },
  { day: 24, dow: 'Sun', amount: 1090, projected: true },
  { day: 25, dow: 'Mon', amount: 2850, projected: true },
  { day: 26, dow: 'Tue', amount: 2990, projected: true },
  { day: 27, dow: 'Wed', amount: 2770, projected: true },
  { day: 28, dow: 'Thu', amount: 2680, projected: true },
  { day: 29, dow: 'Fri', amount: 2490, projected: true },
  { day: 30, dow: 'Sat', amount: 1260, projected: true },
  { day: 31, dow: 'Sun', amount: 1110, projected: true },
]

// April daily spend (same shape, last month) for the overlay line — slightly lower run-rate
const aprilDaily: number[] = [
  1980, 1010, 820, 2410, 2620, 2380, 2540, 2340, 1140, 940,
  2710, 2850, 2680, 2580, 2410, 1220, 1080, 2740, 2900, 2660,
  2490, 1180, 1020, 2710, 2860, 2680, 2540, 2380, 1170, 1030,
]
// Build cumulative for overlay
const aprilCumulative = aprilDaily.reduce<number[]>((acc, v) => {
  acc.push((acc[acc.length - 1] ?? 0) + v)
  return acc
}, [])

// ---------- Cost alerts ----------
interface Alert {
  id: string
  level: 'active' | 'inactive'
  tone: 'warn' | 'info'
  title: string
  body: string
  meta: string
}
const alerts: Alert[] = [
  {
    id: 'ai-research-80',
    level: 'active',
    tone: 'warn',
    title: 'AI-Research-Team approaching 80% of monthly budget',
    body: 'Sub-account spend $22,280 / $25,000 cap (89%). At current burn the cap will be hit on May 22, 2026 (3 days). No auto-stop configured.',
    meta: 'Owner: marcus.chen@walmart.com · Notify: jane.doe@walmart.com, finance-ops@walmart.com',
  },
  {
    id: 'cs-ai-30k',
    level: 'inactive',
    tone: 'info',
    title: 'Customer-Service-AI auto-stop at $30,000',
    body: 'Hard limit. Spend currently $11,997 / $30,000 (40%). Projected to reach $20,400 by month end — not expected to trigger.',
    meta: 'Owner: linda.martinez@walmart.com · Action on trigger: pause inference + page on-call',
  },
  {
    id: 'monthly-90',
    level: 'inactive',
    tone: 'info',
    title: 'Org-wide soft alert at 90% of monthly budget',
    body: 'Sends to billing-alerts Slack channel. Will trigger at $72,000. Projected to reach $67,800 — not expected to trigger.',
    meta: 'Notify: #billing-alerts · finance-ops@walmart.com',
  },
]

// ---------- Invoices ----------
interface Invoice {
  period: string
  amount: number
  status: 'in-progress' | 'paid' | 'pending'
  method: string
  issued: string
  due: string
  paid: string | null
  id: string
}
const invoices: Invoice[] = [
  { period: 'May 2026',      amount: 42847.23, status: 'in-progress', method: 'Wire · Chase ••4421', issued: '—',          due: '2026-06-03', paid: null,         id: 'inv_2026_05_wmt' },
  { period: 'April 2026',    amount: 61238.49, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2026-05-01', due: '2026-05-03', paid: '2026-05-03', id: 'inv_2026_04_wmt' },
  { period: 'March 2026',    amount: 58914.02, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2026-04-01', due: '2026-04-03', paid: '2026-04-02', id: 'inv_2026_03_wmt' },
  { period: 'February 2026', amount: 54201.18, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2026-03-01', due: '2026-03-03', paid: '2026-03-03', id: 'inv_2026_02_wmt' },
  { period: 'January 2026',  amount: 51338.77, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2026-02-01', due: '2026-02-03', paid: '2026-02-02', id: 'inv_2026_01_wmt' },
  { period: 'December 2025', amount: 48719.55, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2026-01-02', due: '2026-01-04', paid: '2026-01-04', id: 'inv_2025_12_wmt' },
  { period: 'November 2025', amount: 46022.30, status: 'paid',        method: 'Wire · Chase ••4421', issued: '2025-12-01', due: '2025-12-03', paid: '2025-12-03', id: 'inv_2025_11_wmt' },
]

// ---------- Chart.js daily spend ----------
const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

function fmtUSD(n: number, opts: { cents?: boolean } = {}): string {
  const o: Intl.NumberFormatOptions = {
    style: 'currency', currency: 'USD',
    minimumFractionDigits: opts.cents ? 2 : 0,
    maximumFractionDigits: opts.cents ? 2 : 0,
  }
  return new Intl.NumberFormat('en-US', o).format(n)
}

onMounted(() => {
  if (!chartCanvas.value) return

  const labels = dailySpend.map((d) => String(d.day))
  const actualData: (number | null)[] = dailySpend.map((d) => (d.projected ? null : d.amount))
  const projectedData: (number | null)[] = dailySpend.map((d, i) => {
    if (!d.projected) return null
    // Stitch start: include the last actual to bridge the line
    return d.amount
  })
  // To bridge actual → projected so the projection line connects, copy day 19 actual into projection start
  if (projectedData[19] !== null) {
    // already projected starts at 20; bridge: also write day 19 actual into projection slot for the connecting segment
  }
  // We'll instead render the projection as a separate bar series, no need for line stitching.

  const aprilOverlay = aprilDaily.slice(0, 31).map((v) => v)

  chart = new Chart(chartCanvas.value, {
    type: 'bar',
    data: {
      labels,
      datasets: [
        {
          type: 'bar',
          label: 'Spend (actual)',
          data: actualData,
          backgroundColor: '#0A0B0E',
          borderColor: '#0A0B0E',
          borderWidth: 0,
          borderRadius: 0,
          maxBarThickness: 18,
        } as ChartDataset<'bar'>,
        {
          type: 'bar',
          label: 'Spend (projected)',
          data: projectedData,
          backgroundColor: 'rgba(10,11,14,0.18)',
          borderColor: 'rgba(10,11,14,0.32)',
          borderWidth: 1,
          borderRadius: 0,
          borderDash: [3, 3],
          maxBarThickness: 18,
        } as ChartDataset<'bar'>,
        {
          type: 'line',
          label: 'April daily',
          data: aprilOverlay,
          borderColor: '#4A90E2',
          backgroundColor: 'transparent',
          borderWidth: 1.4,
          borderDash: [4, 3],
          pointRadius: 0,
          pointHoverRadius: 3,
          tension: 0.25,
          yAxisID: 'y',
        } as ChartDataset<'line'>,
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          display: true,
          position: 'top',
          align: 'end',
          labels: {
            boxWidth: 10,
            boxHeight: 10,
            font: { family: 'Inter', size: 11, weight: 500 },
            color: '#5F5F5C',
            padding: 14,
            usePointStyle: false,
          },
        },
        tooltip: {
          backgroundColor: '#0A0B0E',
          titleColor: '#E8E6E0',
          bodyColor: '#E8E6E0',
          borderColor: 'rgba(255,255,255,0.12)',
          borderWidth: 1,
          padding: 10,
          cornerRadius: 2,
          titleFont: { family: 'JetBrains Mono', size: 11, weight: 600 },
          bodyFont: { family: 'JetBrains Mono', size: 11 },
          callbacks: {
            title: (items) => 'May ' + items[0].label + ', 2026',
            label: (ctx) => {
              const v = ctx.parsed.y
              if (v === null) return ''
              return ctx.dataset.label + ': ' + fmtUSD(v, { cents: false })
            },
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: {
            color: '#9A9A95',
            font: { family: 'JetBrains Mono', size: 10 },
            autoSkip: false,
            maxRotation: 0,
            callback: function (value, idx) {
              const day = idx + 1
              if (day === 1 || day % 5 === 0 || day === 19) return String(day)
              return ''
            },
          },
          border: { color: 'rgba(0,0,0,0.16)' },
        },
        y: {
          beginAtZero: true,
          grid: { color: 'rgba(0,0,0,0.04)', drawTicks: false },
          ticks: {
            color: '#9A9A95',
            font: { family: 'JetBrains Mono', size: 10 },
            padding: 6,
            callback: (v) => '$' + (Number(v) / 1000).toFixed(1) + 'k',
          },
          border: { display: false },
        },
      },
    },
  })
})

onBeforeUnmount(() => {
  if (chart) chart.destroy()
  chart = null
})

// Format helpers used in template
function fmtPct(n: number, digits = 1): string {
  return n.toFixed(digits) + '%'
}
function statusToneClass(tone: string): string {
  return 't-' + tone
}
</script>

<template>
  <div class="billing-page" data-theme="light">
    <!-- Admin chrome -->
    <header class="admin-chrome">
      <NuxtLink to="/" class="brand"><span class="mark" />Exascale</NuxtLink>
      <NuxtLink to="/enterprise/onboarding" class="org" :title="ORG.enterpriseId">
        <span class="org-name">{{ ORG.name }}</span>
        <span class="org-pill">ENTERPRISE</span>
      </NuxtLink>
      <nav class="chrome-nav">
        <NuxtLink to="/enterprise/onboarding" class="ch-link">Onboarding</NuxtLink>
        <NuxtLink to="/enterprise/teams" class="ch-link">Team</NuxtLink>
        <NuxtLink to="/enterprise/audit" class="ch-link">Audit Log</NuxtLink>
        <a href="#" class="ch-link active" aria-current="page">Billing</a>
      </nav>
      <div class="who">
        <span class="who-email mono">patrick.obrien@walmart.com</span>
        <span class="avatar">P</span>
      </div>
    </header>

    <div class="crumb">
      <NuxtLink to="/enterprise/onboarding">Admin</NuxtLink>
      <span class="sep">›</span>
      <span class="cur">Billing</span>
    </div>

    <!-- Page head -->
    <section class="page-head">
      <div class="ph-left">
        <h1>Billing</h1>
        <p class="ph-sub">
          <span class="mono">{{ ORG.name }}</span>
          <span class="dot-sep">·</span>
          <span class="mono">{{ PERIOD.label }} · day {{ PERIOD.daysIn }} of {{ PERIOD.daysInMonth }}</span>
          <span class="dot-sep">·</span>
          <span class="mono">{{ ORG.paymentMethod }}</span>
        </p>
      </div>
      <div class="ph-right">
        <div class="period-select">
          <span class="ps-label">Period</span>
          <select class="ps-input">
            <option>May 2026 (current)</option>
            <option>April 2026</option>
            <option>March 2026</option>
            <option>Q2 2026 (to date)</option>
            <option>YTD 2026</option>
            <option>Last 12 months</option>
            <option>Custom…</option>
          </select>
        </div>
        <div class="exports">
          <button class="btn ghost" type="button">Export CSV</button>
          <button class="btn ghost" type="button">Export PDF</button>
          <button class="btn primary" type="button">Configure budget</button>
        </div>
      </div>
    </section>

    <!-- KPI strip -->
    <section class="kpi-strip">
      <div class="kpi">
        <div class="kpi-label">This month · spend</div>
        <div class="kpi-val mono">{{ fmtUSD(mtdSpend, { cents: true }) }}</div>
        <div class="kpi-foot">
          <span class="mono">{{ fmtUSD(burnPerDay, { cents: false }) }}/day</span>
          <span class="muted">avg burn</span>
        </div>
      </div>

      <div class="kpi">
        <div class="kpi-label">Projected · month end</div>
        <div class="kpi-val mono">{{ fmtUSD(projected) }}</div>
        <div class="kpi-foot">
          <span class="pos mono">▲ {{ fmtPct(vsLastMonthPct) }}</span>
          <span class="muted">vs April {{ fmtUSD(lastMonth) }}</span>
        </div>
      </div>

      <div class="kpi">
        <div class="kpi-label">
          Budget · monthly cap
          <BaseHelpDot
            title="Monthly budget cap"
            what="The hard ceiling on combined spend for this org this month, across all sub-accounts, all services (compute, inference, storage, fees)."
            why="When projected month-end approaches the cap, alerts fire to the owner + finance. At 100% of cap, ALL non-paying-customer workloads auto-pause unless 'soft cap' is set in Settings → Budget."
            field="billing.kpi.budget"
            placement="bottom"
          />
        </div>
        <div class="kpi-val mono">{{ fmtUSD(budget) }}</div>
        <div class="budget-bar" :title="fmtPct(budgetPct) + ' used · projected ' + fmtPct(projectedPct)">
          <div class="bb-bg">
            <div class="bb-fill actual" :style="{ width: budgetPct + '%' }" />
            <div class="bb-fill projected" :style="{ width: (projectedPct - budgetPct) + '%', left: budgetPct + '%' }" />
          </div>
          <div class="bb-legend">
            <span class="mono">{{ fmtPct(budgetPct) }}</span>
            <span class="muted">spent</span>
            <span class="sep">·</span>
            <span class="mono">{{ fmtPct(projectedPct) }}</span>
            <span class="muted">projected</span>
          </div>
        </div>
      </div>

      <div class="kpi">
        <div class="kpi-label">Status</div>
        <div class="kpi-status">
          <span class="status-dot" :class="statusToneClass(statusCopy.dot)" />
          <span class="status-label">{{ statusCopy.label }}</span>
        </div>
        <div class="kpi-foot">
          <span class="muted">{{ statusCopy.hint }}</span>
        </div>
      </div>
    </section>

    <!-- Charts row -->
    <section class="charts-row">
      <!-- Spend over time -->
      <div class="card chart-card">
        <header class="card-head">
          <div>
            <h2 class="card-title">Spend over time</h2>
            <p class="card-sub">
              Daily spend May 1–{{ PERIOD.daysInMonth }} · projected days <span class="mono">20–31</span> dashed · April overlay <span class="mono accent">— —</span>
            </p>
          </div>
          <div class="seg">
            <button class="seg-btn on" type="button">Daily</button>
            <button class="seg-btn" type="button">Weekly</button>
            <button class="seg-btn" type="button">Cumulative</button>
          </div>
        </header>
        <div class="chart-wrap">
          <canvas ref="chartCanvas" />
        </div>
      </div>

      <!-- Spend breakdown -->
      <div class="card breakdown-card">
        <header class="card-head">
          <div>
            <h2 class="card-title">Spend breakdown</h2>
            <p class="card-sub">By service · by sub-account</p>
          </div>
        </header>

        <!-- By service: donut -->
        <div class="breakdown-section">
          <div class="bd-sub-head">
            <span class="bd-eyebrow">By service</span>
            <span class="mono muted">{{ fmtUSD(mtdSpend, { cents: false }) }} total</span>
          </div>
          <div class="donut-row">
            <div class="donut" :style="{ background: donutGradient }">
              <div class="donut-hole">
                <div class="dh-val mono">100%</div>
                <div class="dh-label">of spend</div>
              </div>
            </div>
            <ul class="legend">
              <li v-for="s in services" :key="s.id">
                <span class="lg-sw" :style="{ background: s.color }" />
                <span class="lg-label">{{ s.label }}</span>
                <span class="lg-amount mono">{{ fmtUSD(s.amount, { cents: false }) }}</span>
                <span class="lg-pct mono">{{ s.pct }}%</span>
              </li>
            </ul>
          </div>
        </div>

        <!-- By sub-account: horizontal bars -->
        <div class="breakdown-section">
          <div class="bd-sub-head">
            <span class="bd-eyebrow">By sub-account</span>
            <span class="mono muted">5 active</span>
          </div>
          <ul class="hbars">
            <li v-for="s in subAccounts" :key="s.id">
              <div class="hb-row">
                <span class="hb-label mono">{{ s.label }}</span>
                <span class="hb-amount mono">{{ fmtUSD(s.amount, { cents: false }) }}</span>
                <span class="hb-pct mono">{{ s.pct }}%</span>
              </div>
              <div class="hb-track">
                <div class="hb-fill" :style="{ width: s.pct + '%', background: s.color }" />
              </div>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- Cost alerts -->
    <section class="card alerts-card">
      <header class="card-head">
        <div>
          <h2 class="card-title">Cost alerts</h2>
          <p class="card-sub">
            <span class="mono">1 active</span> · <span class="mono">2 armed (inactive)</span> · evaluated every 5 min on the rolling daily cumulative
          </p>
        </div>
        <button class="btn ghost" type="button">Configure alerts →</button>
      </header>
      <ul class="alerts">
        <li v-for="a in alerts" :key="a.id" class="alert" :class="'a-' + a.level + ' a-tone-' + a.tone">
          <div class="a-rail" />
          <div class="a-body">
            <div class="a-head">
              <span class="a-dot" :class="a.level === 'active' ? 'on' : 'off'" />
              <span class="a-title">{{ a.title }}</span>
              <span class="a-pill mono">{{ a.level === 'active' ? 'ACTIVE' : 'ARMED' }}</span>
            </div>
            <p class="a-text">{{ a.body }}</p>
            <p class="a-meta mono">{{ a.meta }}</p>
          </div>
          <div class="a-actions">
            <button v-if="a.level === 'active'" class="btn primary sm" type="button">Adjust cap</button>
            <button class="btn ghost sm" type="button">Edit</button>
            <button class="btn ghost sm" type="button">Mute 24h</button>
          </div>
        </li>
      </ul>
    </section>

    <!-- Invoices -->
    <section class="card invoices-card">
      <header class="card-head">
        <div>
          <h2 class="card-title">Invoices</h2>
          <p class="card-sub">
            Net-2 payment terms · auto-paid by wire · <a href="#" class="inline-link">View payment methods →</a>
          </p>
        </div>
        <button class="btn ghost" type="button">Download all (CSV)</button>
      </header>
      <div class="inv-wrap">
        <table class="inv-tbl">
          <thead>
            <tr>
              <th class="left">Period</th>
              <th class="right">Amount</th>
              <th class="left">Status</th>
              <th class="left">Payment method</th>
              <th class="left">Issued</th>
              <th class="left">Due / Paid</th>
              <th class="left">Invoice ID</th>
              <th class="right">Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="inv in invoices" :key="inv.id" :class="{ current: inv.status === 'in-progress' }">
              <td class="period">
                <span class="per-label">{{ inv.period }}</span>
                <span v-if="inv.status === 'in-progress'" class="per-tag mono">CURRENT · {{ PERIOD.daysIn }}/{{ PERIOD.daysInMonth }} DAYS</span>
              </td>
              <td class="amount mono">{{ fmtUSD(inv.amount, { cents: true }) }}</td>
              <td>
                <span class="inv-status" :class="'is-' + inv.status">
                  <span class="is-dot" />
                  {{ inv.status === 'in-progress' ? 'In progress' : inv.status === 'paid' ? 'Paid' : 'Pending' }}
                </span>
              </td>
              <td class="mono pm">{{ inv.method }}</td>
              <td class="mono date">{{ inv.issued }}</td>
              <td class="mono date">
                <span v-if="inv.paid" class="paid-on">paid {{ inv.paid }}</span>
                <span v-else>{{ inv.due }}</span>
              </td>
              <td class="mono id">{{ inv.id }}</td>
              <td class="right">
                <button v-if="inv.status === 'paid'" class="btn ghost sm" type="button">Download PDF</button>
                <button v-else class="btn ghost sm" type="button" disabled>Issues on {{ inv.due }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <footer class="page-foot">
      <span class="mono muted">Last refreshed 14:32 UTC · ledger block #14,287 · next invoice generates 2026-06-01 00:00 UTC</span>
      <a href="mailto:billing@exascale.com" class="foot-link">billing@exascale.com</a>
    </footer>
  </div>
</template>

<style scoped>
/* ============================================================
   Page shell (light admin)
   ============================================================ */
.billing-page {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}
.billing-page .mono   { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.billing-page .muted  { color: var(--text-3); }
.billing-page .accent { color: var(--accent); }
.billing-page .pos    { color: var(--pos); }
.billing-page .neg    { color: var(--neg); }

/* ============================================================
   Admin chrome (same shape as audit)
   ============================================================ */
.admin-chrome {
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  align-items: center;
  gap: 28px;
  padding: 18px 32px;
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
}
.brand {
  display: inline-flex; align-items: center; gap: 10px;
  font-family: var(--font-display); font-weight: 700; font-size: 17px;
  letter-spacing: -0.02em; color: var(--text); text-decoration: none;
}
.brand .mark { display:inline-block; width:11px; height:11px; background: var(--brand); }
.org {
  display: inline-flex; align-items: center; gap: 10px;
  padding: 6px 12px; border: 1px solid var(--border); border-radius: 2px;
  color: var(--text); text-decoration: none; background: var(--canvas);
}
.org-name { font-weight: 600; font-size: 13px; }
.org-pill {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; padding: 2px 6px; border-radius: 2px;
  background: rgba(74,144,226,0.10); color: #2563B0;
}
.chrome-nav { display: inline-flex; gap: 22px; align-items: center; }
.ch-link {
  color: var(--text-2); text-decoration: none; font-size: 13px;
  transition: color 160ms ease; position: relative; padding-bottom: 16px; margin-bottom: -16px;
}
.ch-link:hover { color: var(--text); }
.ch-link.active { color: var(--text); font-weight: 600; }
.ch-link.active::after {
  content: ''; position: absolute; left: 0; right: 0; bottom: -1px;
  height: 2px; background: var(--text);
}
.who { display: inline-flex; align-items: center; gap: 10px; }
.who-email { font-size: 12px; color: var(--text-2); }
.avatar {
  width: 28px; height: 28px; border-radius: 50%;
  background: var(--brand); color: var(--text);
  display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 12px;
}

/* Breadcrumb */
.crumb {
  padding: 14px 32px 0;
  font-size: 12px; color: var(--text-3);
  display: flex; align-items: center; gap: 8px;
}
.crumb a { color: var(--text-2); text-decoration: none; }
.crumb a:hover { color: var(--text); }
.crumb .cur { color: var(--text); font-weight: 500; }

/* ============================================================
   Page head
   ============================================================ */
.page-head {
  display: grid; grid-template-columns: 1fr auto; gap: 24px;
  padding: 16px 32px 22px;
  align-items: flex-end;
}
.ph-left h1 {
  font-family: var(--font-display); font-weight: 600;
  font-size: 28px; letter-spacing: -0.015em; margin: 0 0 6px;
}
.ph-sub {
  color: var(--text-2); font-size: 13px; margin: 0;
  display: inline-flex; gap: 6px; align-items: center; flex-wrap: wrap;
}
.dot-sep { color: var(--text-3); }

.ph-right { display: flex; flex-direction: column; gap: 12px; align-items: flex-end; }
.period-select {
  display: inline-flex; align-items: center; gap: 10px;
  padding: 8px 12px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.ps-label {
  font-family: var(--font-mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
}
.ps-input {
  font-family: var(--font-sans); font-size: 12.5px; color: var(--text);
  background: transparent; border: none; outline: none; cursor: pointer;
}
.exports { display: inline-flex; gap: 8px; }

.btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 12px;
  font-family: var(--font-sans); font-size: 12.5px; font-weight: 500;
  border: 1px solid var(--border-strong);
  background: var(--elevated); color: var(--text);
  border-radius: 2px; cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease;
}
.btn:hover:not(:disabled) { background: rgba(0,0,0,0.03); border-color: var(--text); }
.btn:disabled { color: var(--text-3); cursor: default; }
.btn.ghost { background: var(--elevated); }
.btn.primary { background: var(--brand); border-color: var(--brand); color: var(--text); font-weight: 600; }
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.sm { padding: 5px 10px; font-size: 11.5px; }

/* ============================================================
   KPI strip
   ============================================================ */
.kpi-strip {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--border); border-block: 1px solid var(--border);
  margin: 0 32px;
}
.kpi {
  background: var(--elevated);
  padding: 16px 18px 18px;
  display: flex; flex-direction: column; gap: 4px;
  min-height: 130px;
}
.kpi-label {
  font-family: var(--font-mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 4px;
}
.kpi-val {
  font-size: 28px; font-weight: 600; color: var(--text);
  letter-spacing: -0.015em; line-height: 1.1;
}
.kpi-foot {
  margin-top: auto;
  font-size: 11.5px;
  display: inline-flex; gap: 6px; align-items: baseline; flex-wrap: wrap;
}

/* Budget bar */
.budget-bar { margin-top: 8px; }
.bb-bg {
  position: relative;
  width: 100%; height: 8px;
  background: rgba(0,0,0,0.06);
  border-radius: 2px;
  overflow: hidden;
}
.bb-fill {
  position: absolute; top: 0; height: 100%;
}
.bb-fill.actual {
  left: 0;
  background: var(--text);
}
.bb-fill.projected {
  background: repeating-linear-gradient(
    -45deg,
    rgba(10,11,14,0.32), rgba(10,11,14,0.32) 4px,
    rgba(10,11,14,0.10) 4px, rgba(10,11,14,0.10) 8px
  );
}
.bb-legend {
  margin-top: 6px;
  font-size: 11px;
  display: inline-flex; gap: 5px; align-items: baseline; flex-wrap: wrap;
}
.bb-legend .sep { color: var(--text-3); }

/* Status KPI */
.kpi-status {
  display: inline-flex; align-items: center; gap: 10px;
  margin-top: 2px;
}
.status-dot {
  width: 12px; height: 12px; border-radius: 50%;
  display: inline-block;
}
.status-dot.t-pos  { background: var(--pos);  box-shadow: 0 0 0 3px rgba(25,195,125,0.16); }
.status-dot.t-warn { background: var(--warn); box-shadow: 0 0 0 3px rgba(245,158,11,0.16); }
.status-dot.t-neg  { background: var(--neg);  box-shadow: 0 0 0 3px rgba(239,68,68,0.16); }
.status-dot.t-info { background: var(--accent); box-shadow: 0 0 0 3px rgba(74,144,226,0.16); }
.status-label { font-size: 22px; font-weight: 600; letter-spacing: -0.01em; }

/* ============================================================
   Card primitive
   ============================================================ */
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.card-head {
  display: flex; justify-content: space-between; align-items: flex-start;
  gap: 16px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.card-title {
  font-family: var(--font-display); font-weight: 600; font-size: 17px;
  letter-spacing: -0.01em; margin: 0 0 3px;
}
.card-sub {
  margin: 0; color: var(--text-2); font-size: 12px;
}
.inline-link { color: var(--accent); text-decoration: none; }
.inline-link:hover { text-decoration: underline; }

.seg {
  display: inline-flex; padding: 2px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.seg-btn {
  padding: 5px 10px; font-family: var(--font-sans); font-size: 11.5px; font-weight: 500;
  background: transparent; color: var(--text-2);
  border: none; border-radius: 2px; cursor: pointer;
}
.seg-btn:hover { color: var(--text); }
.seg-btn.on { background: var(--elevated); color: var(--text); border: 1px solid var(--border); }

/* ============================================================
   Charts row
   ============================================================ */
.charts-row {
  display: grid; grid-template-columns: 1.55fr 1fr;
  gap: 16px;
  padding: 16px 32px;
}
.chart-card .chart-wrap { padding: 16px 18px 18px; height: 320px; position: relative; }

/* Breakdown card */
.breakdown-card { display: flex; flex-direction: column; }
.breakdown-section { padding: 16px 20px; border-bottom: 1px solid var(--border); }
.breakdown-section:last-child { border-bottom: none; }
.bd-sub-head {
  display: flex; justify-content: space-between; align-items: baseline;
  margin-bottom: 14px;
}
.bd-eyebrow {
  font-family: var(--font-mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-2);
}

/* Donut */
.donut-row { display: grid; grid-template-columns: 110px 1fr; gap: 18px; align-items: center; }
.donut {
  width: 110px; height: 110px; border-radius: 50%;
  position: relative;
  display: flex; align-items: center; justify-content: center;
}
.donut-hole {
  width: 64px; height: 64px; border-radius: 50%;
  background: var(--elevated);
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  border: 1px solid var(--border);
}
.dh-val   { font-size: 14px; font-weight: 700; color: var(--text); }
.dh-label { font-family: var(--font-mono); font-size: 9px; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-3); margin-top: 2px; }

.legend { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.legend li {
  display: grid; grid-template-columns: 10px 1fr auto auto;
  gap: 10px; align-items: center;
  font-size: 12px;
  padding: 3px 0;
}
.lg-sw { width: 10px; height: 10px; border-radius: 2px; }
.lg-label { color: var(--text); }
.lg-amount { color: var(--text); font-size: 11.5px; }
.lg-pct { color: var(--text-3); font-size: 11px; min-width: 32px; text-align: right; }

/* Horizontal bars */
.hbars { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 12px; }
.hb-row {
  display: grid; grid-template-columns: 1fr auto auto;
  gap: 10px; align-items: baseline;
  margin-bottom: 4px;
}
.hb-label { font-size: 11.5px; color: var(--text); }
.hb-amount { font-size: 11.5px; color: var(--text); }
.hb-pct { font-size: 10.5px; color: var(--text-3); min-width: 32px; text-align: right; }
.hb-track {
  height: 6px;
  background: rgba(0,0,0,0.05);
  border-radius: 2px;
  overflow: hidden;
}
.hb-fill { height: 100%; }

/* ============================================================
   Alerts
   ============================================================ */
.alerts-card { margin: 0 32px; }
.alerts { list-style: none; margin: 0; padding: 0; }
.alert {
  display: grid;
  grid-template-columns: 3px 1fr auto;
  gap: 16px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
}
.alert:last-child { border-bottom: none; }
.a-rail { background: var(--border); border-radius: 1px; }
.alert.a-tone-warn.a-active .a-rail { background: var(--warn); }
.alert.a-tone-info .a-rail { background: rgba(74,144,226,0.4); }

.a-body { min-width: 0; }
.a-head {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 6px;
}
.a-dot {
  width: 8px; height: 8px; border-radius: 50%;
}
.a-dot.on  { background: var(--warn); box-shadow: 0 0 0 3px rgba(245,158,11,0.18); animation: alert-pulse 2.4s ease-in-out infinite; }
.a-dot.off { background: rgba(0,0,0,0.18); }
@keyframes alert-pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(245,158,11,0.18); }
  50%      { box-shadow: 0 0 0 6px rgba(245,158,11,0.06); }
}
.a-title { font-weight: 600; font-size: 13.5px; color: var(--text); }
.a-pill {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; padding: 2px 6px; border-radius: 2px;
  background: rgba(0,0,0,0.05); color: var(--text-2);
}
.alert.a-active .a-pill { background: rgba(245,158,11,0.18); color: #8C5A05; }
.a-text { margin: 0 0 4px; color: var(--text); font-size: 12.5px; line-height: 1.5; }
.a-meta { font-size: 11px; color: var(--text-3); }

.a-actions { display: inline-flex; gap: 6px; align-items: flex-start; }

/* ============================================================
   Invoices
   ============================================================ */
.invoices-card { margin: 16px 32px 0; }
.inv-wrap { overflow-x: auto; }
.inv-tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}
.inv-tbl thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3);
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
  white-space: nowrap;
}
.inv-tbl th.right { text-align: right; }
.inv-tbl tbody td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
  white-space: nowrap;
}
.inv-tbl tbody tr.current { background: rgba(74,144,226,0.04); }
.inv-tbl td.amount { text-align: right; font-weight: 500; }
.inv-tbl td.right { text-align: right; }
.inv-tbl .period { display: flex; flex-direction: column; gap: 3px; }
.per-label { color: var(--text); font-weight: 500; }
.per-tag { font-size: 9.5px; font-weight: 700; letter-spacing: 0.12em; color: var(--accent); }

.inv-status {
  display: inline-flex; align-items: center; gap: 6px;
  font-family: var(--font-mono); font-size: 10.5px; font-weight: 600;
  letter-spacing: 0.06em; text-transform: uppercase;
}
.inv-status.is-paid        { color: var(--pos); }
.inv-status.is-in-progress { color: var(--accent); }
.inv-status.is-pending     { color: var(--warn); }
.is-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

.pm   { color: var(--text); }
.date { color: var(--text-2); font-size: 11.5px; }
.paid-on { color: var(--pos); }
.id   { color: var(--text-3); font-size: 11px; }

/* ============================================================
   Footer
   ============================================================ */
.page-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding: 22px 32px 28px;
  font-size: 11.5px;
}
.foot-link { color: var(--accent); text-decoration: none; }
.foot-link:hover { text-decoration: underline; }

@media (max-width: 1180px) {
  .page-head { grid-template-columns: 1fr; }
  .ph-right { align-items: flex-start; }
  .kpi-strip { grid-template-columns: repeat(2, 1fr); }
  .charts-row { grid-template-columns: 1fr; }
}
</style>
