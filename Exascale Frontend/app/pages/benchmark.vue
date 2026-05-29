<script setup lang="ts">
/**
 * /benchmark — Exascale AI Index Methodology · v1.2
 *
 * Whitepaper-style "credibility document". Three-pane shell:
 * left section nav · 720px reading column · right TOC scroll-spy.
 *
 * The formula in §3 is rendered with HTML typography (no KaTeX dep)
 * to keep the bundle lean; the source standalone used KaTeX via CDN.
 * The historical-prints chart in §10 uses Chart.js (already a dep).
 */
import { Chart, type ChartDataset } from 'chart.js/auto'

definePageMeta({ layout: 'marketing' })
useHead({
  title: 'AI Index Methodology · v1.2 — Exascale',
  meta: [{ name: 'description', content: 'Methodology document for the Exascale AI Index — the daily reference price for AI compute.' }],
})

// =====================================================
// Left-nav structure
// =====================================================
const LNAV_GROUPS = [
  {
    num: 'I',
    title: 'Overview',
    items: [
      { id: 'sec-1',   pre: '§1', label: 'Executive summary' },
      { id: 'live',    pre: '··', label: 'Current index value' },
    ],
  },
  {
    num: 'II',
    title: 'Methodology',
    items: [
      { id: 'sec-2', pre: '§2', label: 'Constituent inputs' },
      { id: 'sec-3', pre: '§3', label: 'Calculation formula' },
      { id: 'sec-4', pre: '§4', label: 'Outlier filtering' },
      { id: 'sec-5', pre: '§5', label: 'Volume floor' },
      { id: 'sec-6', pre: '§6', label: 'Manipulation resistance' },
    ],
  },
  {
    num: 'III',
    title: 'Operations',
    items: [
      { id: 'sec-7', pre: '§7', label: 'Publication schedule' },
      { id: 'sec-8', pre: '§8', label: 'Methodology versioning' },
      { id: 'sec-9', pre: '§9', label: 'Audit & oversight' },
    ],
  },
  {
    num: 'IV',
    title: 'Data',
    items: [
      { id: 'sec-10', pre: '§10', label: 'Historical prints' },
      { id: 'sec-11', pre: '§11', label: 'Constituent transparency' },
    ],
  },
  {
    num: 'V',
    title: 'Governance',
    items: [
      { id: 'sec-12', pre: '§12', label: 'Methodology committee' },
      { id: 'sec-13', pre: '§13', label: 'Complaint process' },
      { id: 'sec-14', pre: '§14', label: 'Contact' },
    ],
  },
]

const RTOC = [
  { id: 'sec-1',  label: 'Executive summary' },
  { id: 'sec-2',  label: 'Constituent inputs' },
  { id: 'sec-3',  label: 'Calculation formula' },
  { id: 'sec-4',  label: 'Outlier filtering' },
  { id: 'sec-5',  label: 'Volume floor' },
  { id: 'sec-6',  label: 'Manipulation resistance' },
  { id: 'sec-7',  label: 'Publication schedule' },
  { id: 'sec-8',  label: 'Methodology versioning' },
  { id: 'sec-9',  label: 'Audit & oversight' },
  { id: 'sec-10', label: 'Historical prints' },
  { id: 'sec-11', label: 'Constituent transparency' },
  { id: 'sec-12', label: 'Methodology committee' },
  { id: 'sec-13', label: 'Complaint process' },
  { id: 'sec-14', label: 'Contact' },
]

// =====================================================
// Constituent weights, version history, committee
// =====================================================
const CONSTITUENTS = [
  { name: 'Text credit spot',     weight: 0.42, source: 'EX · TEXT-INDEX',         updated: '14:23:18' },
  { name: 'Speech credit spot',   weight: 0.08, source: 'EX · SPEECH-INDEX',       updated: '14:18:04' },
  { name: 'Image credit spot',    weight: 0.15, source: 'EX · IMAGE-INDEX',        updated: '14:21:51' },
  { name: 'Video credit spot',    weight: 0.05, source: 'EX · VIDEO-INDEX',        updated: '13:55:09' },
  { name: 'Niche credit spot',    weight: 0.10, source: 'EX · NICHE-INDEX',        updated: '14:10:32' },
  { name: 'Inference cost avg.',  weight: 0.20, source: 'Derived · capacity util.',updated: '14:00:00' },
]

const VERSIONS = [
  { v: '1.2', eff: '2026-04-01', cls: 'Minor', summary: 'Weight schedule rebalanced; video credit weight lowered to 0.05 (from 0.07).' },
  { v: '1.1', eff: '2026-01-08', cls: 'Patch', summary: 'Clarified provisional-print reconciliation under §5.' },
  { v: '1.0', eff: '2025-10-15', cls: 'Major', summary: 'Initial release. Six constituents; 120-minute window; 95% trimming.' },
  { v: '0.9', eff: '2025-07-01', cls: 'Major', summary: 'Pre-launch pilot (consultation draft, no live prints).' },
]

const COMMITTEE = [
  { name: 'Dr. Rina Halpern',    role: 'Chair · External',       bio: 'Formerly Head of Index Research, FTSE Russell. Independent appointment, three-year term commencing 2025-09-01.' },
  { name: 'Marcus Kapoor',       role: 'Vice-chair · Internal',  bio: 'Chief Risk Officer, Exascale Markets. Non-voting on weight-schedule revisions per cooling-off rule.' },
  { name: 'Yui Tanaka',          role: 'Member · External',      bio: 'Director, Quantitative Research, Nomura Holdings. Two-year term.' },
  { name: 'Léon Beaumont',       role: 'Member · External',      bio: 'Adjunct Professor of Market Microstructure, INSEAD; formerly Deutsche Börse Index.' },
  { name: 'Priya Rao',           role: 'Member · Internal',      bio: 'Head of Market Data, Exascale Markets. Non-voting on revisions affecting data products.' },
  { name: 'Dr. Aiden O\'Connell', role: 'Observer · External',   bio: 'Representative of the audit firm; observer capacity, no vote.' },
]

// =====================================================
// Live state — countdown + active section
// =====================================================
const countdown = ref('03:24:18')
const activeId = ref('sec-1')

function tickCountdown() {
  const now = new Date()
  const next = new Date(Date.UTC(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate(),
    17, 0, 0,
  ))
  if (next.getTime() <= now.getTime()) next.setUTCDate(next.getUTCDate() + 1)
  const diff = next.getTime() - now.getTime()
  const h = Math.floor(diff / 3.6e6)
  const m = Math.floor((diff % 3.6e6) / 6e4)
  const s = Math.floor((diff % 6e4) / 1e3)
  countdown.value = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// =====================================================
// Historical chart — Chart.js, 217 business days, ends at 1.0024
// =====================================================
const histCanvas = ref<HTMLCanvasElement | null>(null)
let histChart: Chart<'line', number[]> | null = null
let fullDates: Date[] = []
let fullVals: number[] = []
let currentDates: Date[] = []

type RangeKey = '90' | '180' | '365' | 'all'
const currentRange = ref<RangeKey>('365')
const RANGE_KEYS: RangeKey[] = ['90', '180', '365', 'all']
const RANGE_LABELS: Record<RangeKey, string> = { '90': '90D', '180': '180D', '365': '1Y', 'all': 'ALL' }

function buildSeries(days: number) {
  const out: number[] = []
  const startVal = 1.0000
  const endVal   = 1.0024
  let val = startVal
  for (let i = 0; i < days; i++) {
    const drift = (endVal - startVal) / days
    const small = (Math.sin(i * 0.42) + Math.cos(i * 0.27)) * 0.0015
    const noise = (Math.random() - 0.5) * 0.0018
    const ddPhase = Math.sin(i * 0.04) - 0.65
    const dd = ddPhase > 0 ? -ddPhase * 0.012 : 0
    val += drift + small + noise + dd
    out.push(val)
  }
  const last = out[out.length - 1]!
  const lift = endVal - last
  for (let i = 0; i < out.length; i++) out[i]! += lift * (i / (out.length - 1))
  out[0] = startVal
  out[out.length - 1] = endVal
  return out
}

function buildBusinessDays(days: number, end = new Date('2026-05-19T17:00:00Z')) {
  const out: Date[] = []
  const d = new Date(end)
  let count = 0
  while (count < days) {
    const wd = d.getUTCDay()
    if (wd !== 0 && wd !== 6) {
      out.unshift(new Date(d))
      count++
    }
    d.setUTCDate(d.getUTCDate() - 1)
  }
  return out
}

function initHistChart() {
  if (!histCanvas.value) return
  const ctx = histCanvas.value.getContext('2d')!
  const N = 217
  fullDates = buildBusinessDays(N)
  fullVals  = buildSeries(N)
  currentDates = fullDates
  const labels = fullDates.map(d => d.toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: '2-digit' }))
  const grad = ctx.createLinearGradient(0, 0, 0, 240)
  grad.addColorStop(0, 'rgba(14,14,14,0.10)')
  grad.addColorStop(1, 'rgba(14,14,14,0.00)')

  histChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        label: 'AI-INDEX',
        data: fullVals,
        borderColor: '#0E0E0E',
        borderWidth: 1.4,
        backgroundColor: grad,
        fill: true,
        tension: 0.15,
        pointRadius: 0,
        pointHoverRadius: 3,
        pointHoverBackgroundColor: '#fff',
        pointHoverBorderColor: '#0E0E0E',
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
          backgroundColor: '#FFFFFF',
          borderColor: 'rgba(14,14,14,0.20)',
          borderWidth: 1,
          padding: 10,
          cornerRadius: 2,
          titleColor: '#4A4A45',
          bodyColor: '#0E0E0E',
          titleFont: { family: "'JetBrains Mono', monospace", size: 10, weight: 'bold' },
          bodyFont:  { family: "'JetBrains Mono', monospace", size: 12 },
          displayColors: false,
          callbacks: {
            title: items => {
              const d = currentDates[items[0]!.dataIndex]!
              return d.toLocaleDateString('en-US', { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' })
            },
            label: item => 'AI-INDEX  ' + (item.parsed.y as number).toFixed(4),
          },
        },
      },
      scales: {
        x: {
          type: 'category',
          grid: { display: false },
          border: { color: 'rgba(14,14,14,0.20)' },
          ticks: {
            color: '#8A8A82',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 8,
            callback: (_v, idx) => {
              const d = currentDates[idx as number]
              if (!d) return ''
              return d.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
            },
          },
        },
        y: {
          position: 'right',
          grid: { color: 'rgba(14,14,14,0.06)' },
          border: { display: false },
          ticks: {
            color: '#8A8A82',
            font: { family: "'JetBrains Mono', monospace", size: 10 },
            callback: v => (v as number).toFixed(4),
            padding: 8,
          },
        },
      },
    },
  })

  requestAnimationFrame(() => requestAnimationFrame(() => histChart?.resize()))
}

function setRange(r: RangeKey) {
  currentRange.value = r
  if (!histChart) return
  const n = r === 'all' ? fullVals.length : Math.min(parseInt(r, 10), fullVals.length)
  const slice = fullVals.slice(-n)
  const dates = fullDates.slice(-n)
  histChart.data.labels = dates.map(d => d.toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: '2-digit' }))
  histChart.data.datasets[0]!.data = slice
  currentDates = dates
  histChart.update()
}

// =====================================================
// Scroll-spy with IntersectionObserver
// =====================================================
let io: IntersectionObserver | null = null
let cdTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  tickCountdown()
  cdTimer = setInterval(tickCountdown, 1000)
  initHistChart()

  io = new IntersectionObserver((entries) => {
    let best: IntersectionObserverEntry | null = null
    for (const e of entries) {
      if (e.isIntersecting && (!best || e.intersectionRatio > best.intersectionRatio)) best = e
    }
    if (best) activeId.value = best.target.id
  }, { rootMargin: '-30% 0px -60% 0px', threshold: [0, 0.25, 0.5, 1] })

  document.querySelectorAll<HTMLElement>('section.sec').forEach(s => io!.observe(s))
})
onBeforeUnmount(() => {
  if (cdTimer) clearInterval(cdTimer)
  io?.disconnect()
  histChart?.destroy()
})
</script>

<template>
  <div class="methodology">
    <div class="shell">
      <!-- ============ LEFT NAV ============ -->
      <aside class="lnav" aria-label="Document sections">
        <div v-for="g in LNAV_GROUPS" :key="g.num" class="lnav-group">
          <div class="lnav-group-head"><span class="num">{{ g.num }}</span>{{ g.title }}</div>
          <ul>
            <li v-for="i in g.items" :key="i.id">
              <a :href="'#' + i.id" :class="{ active: activeId === i.id }">
                <span class="pre">{{ i.pre }}</span>{{ i.label }}
              </a>
            </li>
          </ul>
        </div>
      </aside>

      <!-- ============ CONTENT ============ -->
      <main class="content">
        <!-- HEADER -->
        <header class="doc-head">
          <div class="doc-eyebrow">Methodology · benchmark document</div>
          <h1 class="doc-title">Exascale AI Index Methodology</h1>
          <div class="doc-version">
            <span>Version <strong>1.2</strong></span>
            <span>Effective <strong>2026-04-01</strong></span>
            <span>Next review <strong>2026-Q3</strong></span>
            <span class="pill"><span class="dot" /> IN EFFECT</span>
          </div>
          <div class="downloads">
            <a class="dl-link" href="#" download>
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"><path d="M8 2v8M4 7l4 4 4-4M3 13h10" /></svg>
              <span>PDF · full whitepaper</span>
              <span class="meta">(486 KB)</span>
            </a>
            <a class="dl-link" href="#" download>
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"><path d="M8 2v8M4 7l4 4 4-4M3 13h10" /></svg>
              <span>JSON · machine-readable</span>
              <span class="meta">(12 KB)</span>
            </a>
            <a class="dl-link" href="#" download>
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"><path d="M8 2v8M4 7l4 4 4-4M3 13h10" /></svg>
              <span>Schema · Avro</span>
              <span class="meta">(4 KB)</span>
            </a>
          </div>
        </header>

        <!-- LIVE INDEX BOX -->
        <div id="live" class="live-box">
          <div class="live-box-head">
            <span>— Current value · AI-INDEX</span>
            <span class="live"><span class="pulse" />Live · refreshed each print</span>
          </div>
          <div class="live-box-grid">
            <div class="live-box-cell">
              <div class="lbl">Current value</div>
              <div class="val">$1 = 1,002.4<span class="unit">AI credits</span></div>
              <div class="sub">24h Δ <span class="pos">▲ +0.18%</span> · 30d Δ <span class="pos">▲ +2.31%</span></div>
            </div>
            <div class="live-box-cell">
              <div class="lbl">Last print</div>
              <div class="val val-sm">16:00 UTC</div>
              <div class="sub">2026-05-19 · ref <span class="strong">0x7a3c91…f042</span></div>
            </div>
            <div class="live-box-cell">
              <div class="lbl">Next print in</div>
              <div class="val val-md">{{ countdown }}</div>
              <div class="sub">17:00 UTC daily · ICE business days</div>
            </div>
          </div>
        </div>

        <!-- §1 EXECUTIVE SUMMARY -->
        <section class="sec" id="sec-1">
          <div class="sec-head">
            <span class="sec-num">§ 1</span>
            <h2>Executive summary</h2>
          </div>
          <div class="sec-body">
            <p class="lede">
              The <strong>Exascale AI Index</strong> is a daily reference price for AI inference, expressed
              as the number of <em>AI credits</em> redeemable per one United States dollar at print
              time. The index is computed from observed transactions across the venue's constituent
              sub-credit markets, weighted by inference category, with statistically robust filtering
              for outliers and minimum-volume requirements.
            </p>
            <p>
              The index is intended as the canonical reference for cash- or physically-settled
              derivatives on AI compute, including the Exascale AI-INDEX spot and forward contracts.
              Its design borrows from established commodity benchmarks (LBMA Gold Price, Brent Dated,
              CME Henry Hub) and from equity-index methodology (S&amp;P, MSCI) where applicable to a
              continuously-traded, multi-constituent underlying.
            </p>
            <p>
              This document is normative. Any divergence between this document and an operational
              implementation is to be treated as a defect in the implementation. The methodology
              committee (§12) is the authority of last resort on interpretation.
            </p>
          </div>
        </section>

        <!-- §2 CONSTITUENT INPUTS -->
        <section class="sec" id="sec-2">
          <div class="sec-head">
            <span class="sec-num">§ 2</span>
            <h2>Constituent inputs</h2>
          </div>
          <div class="sec-body">
            <p>
              The index draws on six observable input streams, each capturing a distinct inference
              category. Five are spot prices on Exascale sub-credit markets; the sixth is a derived
              metric of weighted-average inference cost, computed from observed capacity utilisation
              across the GPU-credit underlying.
            </p>
            <ul>
              <li><strong>Text credit spot.</strong> Median execution price of TEXT-INDEX trades over the calculation window.</li>
              <li><strong>Speech credit spot.</strong> Median execution price of SPEECH-INDEX trades over the calculation window.</li>
              <li><strong>Image credit spot.</strong> Median execution price of IMAGE-INDEX trades over the calculation window.</li>
              <li><strong>Video credit spot.</strong> Median execution price of VIDEO-INDEX trades; thinner book, see §5.</li>
              <li><strong>Niche credit spot.</strong> Aggregate of long-tail inference categories (embedding, classification, fine-tune).</li>
              <li><strong>Inference cost average.</strong> Capacity-weighted USD-per-token cost derived from H100 and H200 GPU-credit utilisation; see Appendix A of the full whitepaper.</li>
            </ul>

            <h3 class="sub">Constituent weights <span class="ix">— effective 2026-04-01</span></h3>
            <p>
              Weights are set by the methodology committee at each quarterly review (§8). Weights
              reflect transaction-volume share over the preceding 90 calendar days, adjusted for
              inference-category representativeness. The current schedule is reproduced below.
            </p>

            <table class="tbl" aria-label="Current constituent weights">
              <thead>
                <tr>
                  <th>Constituent</th>
                  <th class="num-h">Weight</th>
                  <th>Source</th>
                  <th class="num-h">Last update (UTC)</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in CONSTITUENTS" :key="c.name">
                  <td>{{ c.name }}</td>
                  <td class="num">{{ c.weight.toFixed(2) }}</td>
                  <td class="mono-td">{{ c.source }}</td>
                  <td class="num">{{ c.updated }}</td>
                </tr>
                <tr class="total">
                  <td>Total</td>
                  <td class="num">1.00</td>
                  <td colspan="2" />
                </tr>
              </tbody>
            </table>
            <div class="tbl-cap"><strong>Table 1.</strong> Constituent weights in effect at the time of publication. Subsequent quarterly revisions are linked from <a href="#sec-8">§8</a>.</div>
          </div>
        </section>

        <!-- §3 CALCULATION FORMULA -->
        <section class="sec" id="sec-3">
          <div class="sec-head">
            <span class="sec-num">§ 3</span>
            <h2>Calculation formula</h2>
          </div>
          <div class="sec-body">
            <p>
              The index value <code>V(t)</code> at print time <code>t</code> is the trimmed,
              volume-weighted sum of constituent observations across the calculation window
              <code>[t − Δ, t]</code>:
            </p>

            <div class="formula-box" aria-label="Equation 1">
              <div class="formula-eq">
                <em>V</em>(<em>t</em>) =
                trimmedMean<sub class="trim">0.025, 0.975</sub>
                <span class="parens">(</span>
                <span class="sigma">Σ</span><sub class="sigi">i=1</sub><sup class="sign">N</sup>
                <em>P</em><sub>i</sub>(<em>t</em>)
                <em>w</em><sub>i</sub>(<em>t</em>)
                <span class="parens">)</span>
                × <em>κ</em>
              </div>
              <div class="formula-where">
                <dl>
                  <dt>P<sub>i</sub>(t)</dt><dd>observed price of constituent <em>i</em> within the calculation window</dd>
                  <dt>w<sub>i</sub>(t)</dt><dd>active weight of constituent <em>i</em> at time <em>t</em>; <span class="mono-inline">Σ w<sub>i</sub> = 1</span></dd>
                  <dt>N</dt><dd>number of constituents in effect; currently <strong>6</strong></dd>
                  <dt>Δ</dt><dd>calculation window; currently <strong>120 minutes</strong> rolling</dd>
                  <dt>κ</dt><dd>normalization scalar; <strong>1,000</strong> (so the index quotes near unity)</dd>
                </dl>
              </div>
            </div>

            <p>
              The trimmed mean operator excludes the lowest 2.5% and highest 2.5% of observations
              within the window before the volume-weighted sum is taken. The operator is applied
              jointly across all constituents (rather than per-constituent) so that genuine
              cross-asset moves are preserved while idiosyncratic spikes are suppressed.
            </p>
            <p>
              A reference implementation in Python, with deterministic test vectors covering 18
              published prints, is distributed with the JSON download above. The reference
              implementation is the authority for any computational ambiguity in the prose.
            </p>
          </div>
        </section>

        <!-- §4 OUTLIER FILTERING -->
        <section class="sec" id="sec-4">
          <div class="sec-head">
            <span class="sec-num">§ 4</span>
            <h2>Outlier filtering</h2>
          </div>
          <div class="sec-body">
            <p>
              The 95% trimmed mean was selected after comparative back-testing against
              Huber M-estimation, MAD-based winsorisation, and the unmodified arithmetic mean
              across <em>2,194 simulated prints</em> drawn from observed sub-credit volatility. The
              trimmed mean is the simplest estimator that meets two requirements:
              <strong>(i)</strong> a published, auditable cut-off; and
              <strong>(ii)</strong> bounded influence under a single-counterparty wash-trading attack
              up to 2.4% of window volume.
            </p>

            <div class="figure" aria-label="Trimmed mean visualization">
              <div class="figure-head">
                <span>— Figure 1 · Trimmed-mean operator</span>
                <span class="legend">
                  <span><span class="sw sw-dark" />Retained · 95%</span>
                  <span><span class="sw sw-tail" />Excluded tails · 2 × 2.5%</span>
                </span>
              </div>
              <div class="figure-body">
                <svg class="hist-svg" viewBox="0 0 720 200" preserveAspectRatio="none" aria-hidden="true">
                  <line x1="20" y1="180" x2="700" y2="180" stroke="rgba(14,14,14,0.20)" stroke-width="1" />
                  <path d="M 20,180 L 20,165 L 40,155 L 60,148 L 80,140 L 80,180 Z" fill="#F2EEE3" stroke="#0E0E0E" stroke-width="1" />
                  <path d="M 640,180 L 640,140 L 660,148 L 680,160 L 700,170 L 700,180 Z" fill="#F2EEE3" stroke="#0E0E0E" stroke-width="1" />
                  <path d="M 80,140 L 100,128 L 130,110 L 160,90 L 190,72 L 220,56 L 260,40 L 300,30 L 340,26 L 360,25 L 380,26 L 420,30 L 460,40 L 500,56 L 540,72 L 580,90 L 610,110 L 640,140 L 640,180 L 80,180 Z" fill="#0E0E0E" fill-opacity="0.86" />
                  <line x1="80" y1="20" x2="80" y2="180" stroke="#0E0E0E" stroke-width="1" stroke-dasharray="3 3" />
                  <line x1="640" y1="20" x2="640" y2="180" stroke="#0E0E0E" stroke-width="1" stroke-dasharray="3 3" />
                  <line x1="360" y1="20" x2="360" y2="180" stroke="#C8F25C" stroke-width="2" />
                  <text x="80" y="14" text-anchor="middle" fill="#4A4A45" font-family="JetBrains Mono" font-size="10" font-weight="600" letter-spacing="0.06em">P₂.₅</text>
                  <text x="640" y="14" text-anchor="middle" fill="#4A4A45" font-family="JetBrains Mono" font-size="10" font-weight="600" letter-spacing="0.06em">P₉₇.₅</text>
                  <text x="360" y="14" text-anchor="middle" fill="#0E0E0E" font-family="JetBrains Mono" font-size="10" font-weight="600" letter-spacing="0.06em">x̄ = V(t)</text>
                  <text x="50" y="196" text-anchor="middle" fill="#8A8A82" font-family="JetBrains Mono" font-size="9">2.5%</text>
                  <text x="360" y="196" text-anchor="middle" fill="#8A8A82" font-family="JetBrains Mono" font-size="9">Distribution of constituent-weighted observations · 120-min window</text>
                  <text x="670" y="196" text-anchor="middle" fill="#8A8A82" font-family="JetBrains Mono" font-size="9">2.5%</text>
                </svg>
              </div>
              <div class="figure-cap"><strong>Figure 1.</strong> Stylised distribution of constituent-weighted observations across the calculation window. The two shaded tails (2.5% each) are excluded before the volume-weighted mean is taken.</div>
            </div>

            <p>
              The 2.5% trimming threshold may be revised by the methodology committee with not less
              than 30 calendar days' notice. Any revision triggers a new methodology version
              (§8).
            </p>
          </div>
        </section>

        <!-- §5 VOLUME FLOOR -->
        <section class="sec" id="sec-5">
          <div class="sec-head">
            <span class="sec-num">§ 5</span>
            <h2>Volume floor</h2>
          </div>
          <div class="sec-body">
            <p>
              A valid print requires at least <strong>N = 240</strong> observations across the
              calculation window, of which not fewer than <strong>20</strong> must originate from
              each constituent except <em>video credit spot</em>, which has a lower floor of
              <strong>8</strong> observations in recognition of its thinner book.
            </p>
            <p>
              If the volume floor is not met at any constituent, the print is marked
              <em>provisional</em> and republished with the next-day reconciliation. If the floor
              is not met at the aggregate level, no print is issued; the previous print remains
              the prevailing reference and the methodology committee convenes within four
              business hours under standing protocol <code>VP-01</code>.
            </p>
            <p>
              In the 18 months of pre-launch observation, the aggregate floor was met on
              <strong>100.0%</strong> of business days; constituent floors were met on
              <strong>99.6%</strong> of business days, with three provisional prints, each
              reconciled within one publication cycle.
            </p>
          </div>
        </section>

        <!-- §6 MANIPULATION RESISTANCE -->
        <section class="sec" id="sec-6">
          <div class="sec-head">
            <span class="sec-num">§ 6</span>
            <h2>Manipulation resistance</h2>
          </div>
          <div class="sec-body">
            <p>
              The index incorporates the following defences against attempted manipulation. Each
              is documented in greater detail in the corresponding annex of the full whitepaper.
            </p>
            <ol>
              <li><strong>Cross-validation against multiple input sources.</strong> Each constituent's spot price is corroborated against the time-weighted mid-quote on the same market; large divergences trigger source isolation.</li>
              <li><strong>Outlier exclusion via trimmed mean.</strong> See §4. Bounded influence of any single counterparty's window activity.</li>
              <li><strong>Volume threshold for valid prints.</strong> See §5. A thin book defers to the prior print rather than admitting a thin observation.</li>
              <li><strong>Surveillance for marking-the-close patterns.</strong> Pattern matching against last-minute order-book pressure, with manual review for matches above the surveillance threshold.</li>
              <li><strong>Cryptographic audit chain on all prints.</strong> Each daily print is hashed and chained to the prior print; the chain root is published to a third-party transparency log within 60 seconds of issuance.</li>
              <li><strong>Member-trading restrictions.</strong> Methodology committee members are subject to a 5-day cooling-off window around methodology revisions and may not hold proprietary positions in any constituent during their tenure.</li>
            </ol>
          </div>
        </section>

        <!-- §7 PUBLICATION SCHEDULE -->
        <section class="sec" id="sec-7">
          <div class="sec-head">
            <span class="sec-num">§ 7</span>
            <h2>Publication schedule</h2>
          </div>
          <div class="sec-body">
            <p>
              The Exascale AI Index is published once per ICE business day at <strong>17:00 UTC</strong>.
              Publication latency is targeted at &lt; 60 seconds from the close of the calculation
              window; the observed median latency in 2026-Q1 was 14 seconds.
            </p>
            <p>
              A continuous intraday <em>indicative</em> value is also computed at five-second intervals
              and disseminated over the market data feed under the symbol <code>AI-INDEX.IV</code>.
              The intraday indicative is for reference only and does not constitute a published print.
            </p>
            <p>
              Holiday and exceptional-event handling follows the schedule maintained at
              <a href="#">exascale.com/calendar</a>. No print is issued on days where the venue is closed.
            </p>
          </div>
        </section>

        <!-- §8 METHODOLOGY VERSIONING -->
        <section class="sec" id="sec-8">
          <div class="sec-head">
            <span class="sec-num">§ 8</span>
            <h2>Methodology versioning</h2>
          </div>
          <div class="sec-body">
            <p>
              Methodology revisions follow semantic versioning with the following correspondence:
              <strong>major</strong> changes (formula structure, constituent set) carry a 90-day
              consultation period; <strong>minor</strong> changes (weight schedule, trimming
              threshold) carry a 30-day notification period; <strong>patch</strong> changes
              (typographical, clarification of intent) take effect at publication and do not
              re-rebase historical prints.
            </p>

            <table class="tbl" aria-label="Methodology version history">
              <thead>
                <tr>
                  <th>Version</th>
                  <th>Effective</th>
                  <th>Class</th>
                  <th>Summary</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="v in VERSIONS" :key="v.v">
                  <td class="mono-td">{{ v.v }}</td>
                  <td class="mono-td">{{ v.eff }}</td>
                  <td>{{ v.cls }}</td>
                  <td>{{ v.summary }}</td>
                </tr>
              </tbody>
            </table>
            <div class="tbl-cap"><strong>Table 2.</strong> Methodology version history. Earlier consultation drafts available on request.</div>
          </div>
        </section>

        <!-- §9 AUDIT & OVERSIGHT -->
        <section class="sec" id="sec-9">
          <div class="sec-head">
            <span class="sec-num">§ 9</span>
            <h2>Audit and oversight</h2>
          </div>
          <div class="sec-body">
            <p>
              The methodology is reviewed quarterly by the Exascale Methodology Committee (§12)
              and audited annually by an independent third-party benchmark administrator. The
              audit covers (a) conformity of the operational implementation to this document,
              (b) integrity of the cryptographic audit chain, and (c) governance practices of the
              methodology committee.
            </p>
            <div class="callout">
              <span class="lbl">Independent auditor</span>
              <p class="callout-p">
                <em>[Auditor name to be confirmed — appointment under review by the Audit and
                Oversight Committee. Expected attestation: ISAE 3000 (Revised), with scope per
                IOSCO Principles for Financial Benchmarks.]</em>
                <br />
                <a href="#" class="callout-link">Audit reports archive →</a>
              </p>
            </div>
            <p>
              The audit chain is rooted in a third-party transparency log operated by the
              <em>Cloudflare Merkle Town</em> service (RFC 6962). Independent verifiers may
              reconstruct the entire history of prints from the published audit roots and the
              per-print JSON manifests distributed under
              <a href="#">data.exascale.com/index/v1/</a>.
            </p>
          </div>
        </section>

        <!-- §10 HISTORICAL PRINTS -->
        <section class="sec" id="sec-10">
          <div class="sec-head">
            <span class="sec-num">§ 10</span>
            <h2>Historical prints</h2>
          </div>
          <div class="sec-body">
            <p>
              The chart below renders the full series of daily index prints from launch
              (2025-10-15) to the most recent business day. The index was normalised to a value of
              <strong>1.0000</strong> at launch; the most recent print is
              <strong>1.0024</strong>, a year-to-date change of <strong>+4.18%</strong>.
            </p>

            <div class="chart-card">
              <div class="chart-head">
                <h3 class="chart-ttl">AI-INDEX · daily prints<span class="chart-meta">· since launch · 217 business days</span></h3>
                <div class="ranges">
                  <button
                    v-for="r in RANGE_KEYS"
                    :key="r"
                    type="button"
                    class="r"
                    :class="{ active: currentRange === r }"
                    @click="setRange(r)"
                  >{{ RANGE_LABELS[r] }}</button>
                </div>
              </div>
              <div class="chart-body">
                <div class="chart-canvas-wrap"><canvas ref="histCanvas" /></div>
              </div>
              <div class="chart-foot">
                <span>Source · EX market data feed · published prints only · normalised to 1.0000 at launch</span>
                <a href="#" download>Download full historical data (CSV) →</a>
              </div>
            </div>

            <p>
              Prints prior to launch are not included; the consultation-period pilot
              (versions 0.9 — 0.9.4) was a paper exercise and did not include settlement.
              Researchers requesting the pilot series should contact
              <a href="mailto:methodology@exascale.com">methodology@exascale.com</a>.
            </p>
          </div>
        </section>

        <!-- §11 CONSTITUENT TRANSPARENCY -->
        <section class="sec" id="sec-11">
          <div class="sec-head">
            <span class="sec-num">§ 11</span>
            <h2>Constituent transparency</h2>
          </div>
          <div class="sec-body">
            <p>
              Per-constituent observed prices, window-volume figures, and exclusion counts are
              published with each print under the JSON manifest. The manifest is fully signed and
              chained to the audit log described in §9, and is downloadable on a five-second
              delay over the public data API.
            </p>
            <p>
              Subscribers to the institutional data feed receive the full per-fill detail in
              real time, including taker / maker flags, counterparty bucket (member /
              non-member), and the maker–taker contribution to each constituent's window-volume.
            </p>
          </div>
        </section>

        <!-- §12 METHODOLOGY COMMITTEE -->
        <section class="sec" id="sec-12">
          <div class="sec-head">
            <span class="sec-num">§ 12</span>
            <h2>Methodology committee</h2>
          </div>
          <div class="sec-body">
            <p>
              The methodology committee is responsible for (a) the quarterly review of constituent
              weights, (b) extraordinary reviews triggered under the volume-floor protocol (§5),
              and (c) approval of any methodology revision. Membership is mixed internal and
              external; the external chair holds a casting vote.
            </p>

            <div class="committee">
              <div v-for="m in COMMITTEE" :key="m.name" class="member">
                <div class="name">{{ m.name }}</div>
                <div class="role">{{ m.role }}</div>
                <div class="bio">{{ m.bio }}</div>
              </div>
            </div>

            <p>
              Committee minutes for non-confidential portions of each session are published at
              <a href="#">exascale.com/methodology/committee/minutes</a> within ten business days
              of the session.
            </p>
          </div>
        </section>

        <!-- §13 COMPLAINT PROCESS -->
        <section class="sec" id="sec-13">
          <div class="sec-head">
            <span class="sec-num">§ 13</span>
            <h2>Complaint process</h2>
          </div>
          <div class="sec-body">
            <p>
              Any market participant, vendor, or member of the public may submit a complaint
              regarding the integrity of a print, the conduct of the methodology committee, or
              the operation of the audit chain. Complaints are received at
              <a href="mailto:complaints@exascale.com">complaints@exascale.com</a> and
              acknowledged within two business days.
            </p>
            <p>
              Substantive complaints are investigated within ten business days. Where the
              investigation cannot be completed within the standard window, the complainant is
              notified of an extended timetable. Resolution outcomes are published in
              anonymised form at <a href="#">exascale.com/methodology/complaints</a>.
            </p>
          </div>
        </section>

        <!-- §14 CONTACT -->
        <section class="sec" id="sec-14">
          <div class="sec-head">
            <span class="sec-num">§ 14</span>
            <h2>Contact</h2>
          </div>
          <div class="sec-body">
            <p>
              General methodology enquiries:
              <a href="mailto:methodology@exascale.com">methodology@exascale.com</a>.
              Audit-chain and reconciliation enquiries:
              <a href="mailto:audit@exascale.com">audit@exascale.com</a>.
              Media enquiries should be routed through
              <a href="mailto:press@exascale.com">press@exascale.com</a>.
            </p>
            <p>
              Exascale Markets · 200 Pine Street · San Francisco, CA 94104 · United States.
              For corporate registration and counterparty diligence, see
              <a href="#">exascale.com/legal/entity</a>.
            </p>
          </div>
        </section>

        <!-- FOOTNOTES -->
        <div class="footnote">
          <p><sup>1</sup> The Exascale AI Index is a non-investible benchmark. Any reference to <em>"investing"</em> in the index refers to investment in derivatives whose settlement value is determined by reference to a print of the index.</p>
          <p><sup>2</sup> The trimmed mean is the operator <em>x̄<sub>α,β</sub>(X) := mean({ x ∈ X : Q<sub>α</sub>(X) ≤ x ≤ Q<sub>β</sub>(X) })</em>, where Q is the empirical quantile function.</p>
          <p><sup>3</sup> Cloudflare Merkle Town is referenced for illustrative purposes; the production audit log provider will be confirmed in the next minor revision.</p>
        </div>

        <!-- PAGE FOOTER -->
        <div class="doc-foot">
          <div class="contacts">
            <span><a href="mailto:methodology@exascale.com">methodology@exascale.com</a></span>
            <span><a href="mailto:audit@exascale.com">audit@exascale.com</a></span>
          </div>
          <div class="vers">
            <div class="v">Methodology v1.2</div>
            <div>Last updated 2026-04-01</div>
            <div>This document supersedes all prior methodology documents.</div>
          </div>
        </div>
      </main>

      <!-- ============ RIGHT TOC ============ -->
      <aside class="rtoc" aria-label="On this page">
        <div class="rtoc-head">— On this page</div>
        <ul>
          <li v-for="i in RTOC" :key="i.id">
            <a :href="'#' + i.id" :class="{ active: activeId === i.id }">{{ i.label }}</a>
          </li>
        </ul>
        <div class="rtoc-meta">
          <strong>v1.2</strong> · Effective 2026-04-01<br />
          Next quarterly review: <strong>2026-Q3</strong>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.methodology {
  --t-4: #B8B8B0;
  --bd-soft: rgba(14, 14, 14, 0.06);
  --font-serif: 'Source Serif 4', 'Charter', Georgia, serif;
  background: var(--canvas);
  color: var(--text);
  font-size: 14px;
  line-height: 1.55;
  -webkit-font-smoothing: antialiased;
  font-feature-settings: 'ss01';
}
.pos { color: var(--pos); }
.mono-inline { font-family: var(--font-mono); }

html { scroll-behavior: smooth; }

/* ============================================================
   Page grid — three pane
   ============================================================ */
.shell {
  max-width: 1280px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr) 220px;
  gap: 48px;
  padding: 48px 32px 96px;
  align-items: start;
}

/* ============================================================
   Left nav
   ============================================================ */
.lnav {
  position: sticky;
  top: 80px;
  max-height: calc(100vh - 96px);
  overflow-y: auto;
  padding-right: 8px;
  font-size: 13px;
}
.lnav::-webkit-scrollbar { width: 6px; }
.lnav::-webkit-scrollbar-thumb { background: var(--border); border-radius: var(--radius-sm); }

.lnav-group { margin-bottom: 18px; }
.lnav-group-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--bd-soft);
  display: flex;
  align-items: center;
  gap: 8px;
}
.lnav-group-head .num {
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--t-4);
  letter-spacing: 0.04em;
}
.lnav ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.lnav li a {
  display: block;
  color: var(--text-2);
  text-decoration: none;
  padding: 5px 10px 5px 14px;
  font-size: 13px;
  line-height: 1.35;
  border-left: 2px solid transparent;
  transition: color 120ms, border-color 120ms, background 120ms;
}
.lnav li a:hover { color: var(--text); }
.lnav li a.active {
  color: var(--text);
  border-left-color: var(--text);
  background: var(--sunken);
  font-weight: 500;
}
.lnav li a .pre {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-right: 8px;
}
.lnav li a.active .pre { color: var(--text); }

/* ============================================================
   Content
   ============================================================ */
.content {
  max-width: 720px;
  width: 100%;
  font-family: var(--font-sans);
  color: var(--text);
}

.doc-head {
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 32px;
}
.doc-eyebrow {
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
.doc-eyebrow::before {
  content: '';
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.doc-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 34px;
  letter-spacing: -0.025em;
  line-height: 1.1;
  margin: 0 0 14px;
  color: var(--text);
}
.doc-version {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 18px;
}
.doc-version strong { color: var(--text); font-weight: 500; }
.doc-version .pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  border: 1px solid var(--border);
  background: var(--elevated);
  border-radius: var(--radius-sm);
  font-size: 11px;
}
.doc-version .pill .dot {
  width: 5px;
  height: 5px;
  background: var(--pos);
  border-radius: 50%;
}

.downloads { display: flex; gap: 8px; margin-top: 4px; flex-wrap: wrap; }
.dl-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  color: var(--text);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: -0.005em;
  transition: background 120ms, border-color 120ms;
}
.dl-link:hover {
  background: var(--canvas);
  border-color: rgba(14, 14, 14, 0.32);
}
.dl-link svg { width: 12px; height: 12px; }
.dl-link .meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

/* Live index box */
.live-box {
  border: 1px solid var(--border-strong);
  background: var(--elevated);
  border-radius: var(--radius-sm);
  margin-bottom: 40px;
  position: relative;
  overflow: hidden;
}
.live-box::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--brand);
}
.live-box-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 18px;
  border-bottom: 1px solid var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.live-box-head .live {
  color: var(--text-2);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.live-box-head .pulse {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  box-shadow: 0 0 0 0 rgba(22, 163, 74, 0.6);
  animation: pulse-anim 2.4s infinite;
}
@keyframes pulse-anim {
  0%   { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0.6); }
  70%  { box-shadow: 0 0 0 6px rgba(22, 163, 74, 0); }
  100% { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0); }
}
.live-box-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr;
}
.live-box-cell {
  padding: 18px 22px;
  border-right: 1px solid var(--bd-soft);
}
.live-box-cell:last-child { border-right: 0; }
.live-box .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 6px;
}
.live-box .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 24px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.01em;
}
.live-box .val.val-sm { font-size: 18px; }
.live-box .val.val-md { font-size: 22px; }
.live-box .val .unit {
  font-size: 12px;
  color: var(--text-3);
  font-weight: 400;
  margin-left: 4px;
  letter-spacing: 0.04em;
}
.live-box .sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  margin-top: 4px;
  letter-spacing: 0.04em;
}
.live-box .sub .strong { color: var(--text); }

/* Section */
section.sec {
  margin-bottom: 56px;
  scroll-margin-top: 80px;
}
section.sec .sec-head {
  display: flex;
  align-items: baseline;
  gap: 14px;
  margin-bottom: 16px;
}
section.sec .sec-num {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-3);
  letter-spacing: 0.08em;
  font-variant-numeric: tabular-nums;
  padding-top: 6px;
  flex-shrink: 0;
  width: 38px;
}
section.sec h2 {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 24px;
  letter-spacing: -0.018em;
  line-height: 1.2;
  margin: 0;
  color: var(--text);
}
.sec-body { padding-left: 52px; }
.sec-body p,
.sec-body ul,
.sec-body ol {
  font-family: var(--font-serif);
  font-size: 16px;
  line-height: 1.62;
  color: var(--text);
  margin: 0 0 16px;
}
.sec-body .lede { font-size: 17px; line-height: 1.6; }
.sec-body strong { font-weight: 600; }
.sec-body em { font-style: italic; color: var(--text-2); }
.sec-body a {
  color: var(--text);
  text-decoration: underline;
  text-decoration-color: var(--text-3);
  text-decoration-thickness: 1px;
  text-underline-offset: 2px;
}
.sec-body a:hover { text-decoration-color: var(--text); }
.sec-body code {
  font-family: var(--font-mono);
  font-size: 0.88em;
  background: var(--sunken);
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  color: var(--text);
}
.sec-body ul, .sec-body ol { padding-left: 24px; }
.sec-body ul li, .sec-body ol li {
  margin-bottom: 8px;
  padding-left: 4px;
}
.sec-body ul li::marker { color: var(--text-3); }
.sec-body ol li::marker {
  color: var(--text-3);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

h3.sub {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 17px;
  letter-spacing: -0.005em;
  color: var(--text);
  margin: 32px 0 12px;
  display: flex;
  align-items: baseline;
  gap: 10px;
}
h3.sub .ix {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

/* ============================================================
   Tables
   ============================================================ */
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-sans);
  font-size: 13.5px;
  margin: 16px 0 8px;
  border-top: 1px solid var(--border);
  border-bottom: 2px solid var(--text);
}
.tbl thead th {
  text-align: left;
  padding: 8px 10px;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  border-bottom: 1px solid var(--text);
  background: transparent;
  white-space: nowrap;
}
.tbl thead th.num-h { text-align: right; }
.tbl tbody td {
  padding: 9px 10px;
  border-bottom: 1px solid var(--bd-soft);
  color: var(--text);
  vertical-align: top;
}
.tbl tbody tr:last-child td { border-bottom: 0; }
.tbl tbody td.num {
  text-align: right;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}
.tbl tbody td.mono-td {
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text-2);
}
.tbl tbody tr.total td {
  border-top: 1px solid var(--text);
  border-bottom: 0;
  font-weight: 600;
  padding-top: 12px;
}
.tbl-cap {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-3);
  margin-top: 8px;
}
.tbl-cap strong { color: var(--text-2); font-weight: 500; }

/* ============================================================
   Formula
   ============================================================ */
.formula-box {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 28px 24px;
  margin: 20px 0 16px;
  text-align: center;
  position: relative;
}
.formula-box::before {
  content: 'Equation 1';
  position: absolute;
  top: 8px;
  right: 12px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--t-4);
}
.formula-eq {
  font-family: 'Source Serif 4', 'Charter', Georgia, serif;
  font-size: 21px;
  line-height: 1.4;
  color: var(--text);
  font-feature-settings: 'ss01';
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: center;
}
.formula-eq em { font-style: italic; }
.formula-eq sub { font-size: 0.62em; vertical-align: -0.4em; }
.formula-eq sup { font-size: 0.62em; vertical-align: 0.5em; }
.formula-eq .parens { font-size: 1.1em; }
.formula-eq .sigma { font-size: 1.45em; font-family: 'Source Serif 4', Georgia, serif; }
.formula-eq .sigi { font-size: 0.55em; }
.formula-eq .sign { font-size: 0.55em; }
.formula-eq .trim { font-family: var(--font-mono); font-size: 0.6em; }

.formula-where {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-2);
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed var(--border);
  letter-spacing: 0.02em;
  text-align: left;
}
.formula-where dl {
  margin: 0;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 4px 18px;
}
.formula-where dt {
  font-family: var(--font-mono);
  color: var(--text);
  font-weight: 500;
}
.formula-where dd {
  margin: 0;
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 13px;
}

/* ============================================================
   Figures
   ============================================================ */
.figure {
  margin: 20px 0;
  border: 1px solid var(--border);
  background: var(--elevated);
  border-radius: var(--radius-sm);
}
.figure-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--bd-soft);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.figure-head .legend {
  display: inline-flex;
  gap: 14px;
  text-transform: none;
  letter-spacing: 0.04em;
  font-size: 11px;
}
.figure-head .legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-2);
}
.figure-head .legend .sw {
  width: 9px;
  height: 9px;
  display: inline-block;
}
.figure-head .legend .sw-dark { background: var(--text); }
.figure-head .legend .sw-tail { background: var(--sunken); border: 1px solid var(--border-strong); }
.figure-body { padding: 16px 18px 14px; }
.figure-cap {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  padding: 0 16px 12px;
}
.figure-cap strong { color: var(--text-2); font-weight: 500; }
.hist-svg { width: 100%; height: 200px; display: block; }

/* ============================================================
   Chart card
   ============================================================ */
.chart-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin: 16px 0;
}
.chart-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 14px 18px;
  border-bottom: 1px solid var(--bd-soft);
}
.chart-ttl {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.005em;
  margin: 0;
  color: var(--text);
}
.chart-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.04em;
  margin-left: 6px;
}
.ranges {
  display: inline-flex;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.ranges .r {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  padding: 5px 10px;
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  cursor: pointer;
  font-variant-numeric: tabular-nums;
}
.ranges .r:last-child { border-right: 0; }
.ranges .r.active { background: var(--sunken); color: var(--text); }
.ranges .r:hover { color: var(--text); }
.chart-body { padding: 14px 18px; }
.chart-canvas-wrap { position: relative; height: 240px; width: 100%; }
.chart-foot {
  padding: 12px 18px;
  border-top: 1px solid var(--bd-soft);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.chart-foot a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
}
.chart-foot a:hover { text-decoration: underline; }

/* Callout */
.callout {
  border-left: 3px solid var(--border-strong);
  background: var(--elevated);
  padding: 14px 18px;
  font-family: var(--font-sans);
  font-size: 13.5px;
  color: var(--text-2);
  line-height: 1.55;
  margin: 18px 0 16px;
  border-radius: var(--radius-sm);
}
.callout strong { color: var(--text); font-weight: 500; }
.callout .lbl {
  display: inline-block;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 6px;
}
.callout-p { margin: 0; }
.callout-p em { font-style: italic; }
.callout-link {
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  text-decoration: underline;
  text-decoration-color: var(--text-3);
}

/* Committee */
.committee {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  margin: 12px 0 16px;
}
.committee .member {
  padding: 14px 18px;
  border-right: 1px solid var(--bd-soft);
  border-bottom: 1px solid var(--bd-soft);
  font-family: var(--font-sans);
}
.committee .member:nth-child(2n) { border-right: 0; }
.committee .member:nth-last-child(-n+2) { border-bottom: 0; }
.committee .name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}
.committee .role {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 2px;
}
.committee .bio {
  font-size: 12.5px;
  color: var(--text-2);
  margin-top: 8px;
  line-height: 1.5;
}

/* Footnote */
.footnote {
  border-top: 1px solid var(--border);
  padding-top: 12px;
  margin-top: 24px;
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.5;
}
.footnote sup {
  font-family: var(--font-mono);
  color: var(--text);
  font-weight: 600;
}

/* Doc footer */
.doc-foot {
  margin-top: 64px;
  padding-top: 24px;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-3);
}
.doc-foot .contacts {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--text-2);
}
.doc-foot .contacts a { color: var(--text); text-decoration: none; }
.doc-foot .contacts a:hover { text-decoration: underline; }
.doc-foot .vers { text-align: right; }
.doc-foot .vers .v { color: var(--text); font-weight: 500; }

/* ============================================================
   Right TOC
   ============================================================ */
.rtoc {
  position: sticky;
  top: 80px;
  max-height: calc(100vh - 96px);
  font-size: 12.5px;
  padding-left: 8px;
  border-left: 1px solid var(--border);
}
.rtoc-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin: 0 0 12px 12px;
}
.rtoc ul { list-style: none; padding: 0; margin: 0; }
.rtoc li a {
  display: block;
  color: var(--text-3);
  text-decoration: none;
  padding: 4px 12px;
  line-height: 1.4;
  border-left: 2px solid transparent;
  margin-left: -1px;
  font-size: 12.5px;
  transition: color 120ms, border-color 120ms;
}
.rtoc li a:hover { color: var(--text); }
.rtoc li a.active {
  color: var(--text);
  border-left-color: var(--text);
  font-weight: 500;
}
.rtoc-meta {
  margin-top: 18px;
  padding: 12px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  background: var(--sunken);
  letter-spacing: 0.04em;
  line-height: 1.55;
  border-radius: var(--radius-sm);
}
.rtoc-meta strong { color: var(--text); font-weight: 500; }

/* Print */
@media print {
  .lnav, .rtoc { display: none !important; }
  .shell {
    grid-template-columns: 1fr;
    padding: 0;
    max-width: none;
  }
  .content { max-width: none; }
  section.sec { page-break-inside: avoid; }
}

/* Responsive */
@media (max-width: 1200px) {
  .shell { grid-template-columns: 200px minmax(0, 1fr); }
  .rtoc { display: none; }
}
@media (max-width: 900px) {
  .shell {
    grid-template-columns: 1fr;
    gap: 24px;
    padding: 32px 16px 64px;
  }
  .lnav { position: static; max-height: none; }
  .sec-body { padding-left: 0; }
  section.sec .sec-num { width: auto; padding-top: 0; }
  .live-box-grid { grid-template-columns: 1fr; }
  .live-box-cell { border-right: 0; border-bottom: 1px solid var(--bd-soft); }
  .live-box-cell:last-child { border-bottom: 0; }
  .committee { grid-template-columns: 1fr; }
  .committee .member { border-right: 0; }
}
</style>
