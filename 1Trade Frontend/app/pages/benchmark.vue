<script setup lang="ts">
/**
 * /benchmark — the AI Index methodology, rendered from the live index API: the current print, the
 * constituent weight schedule, the window/trim/volume-floor parameters and the hash-chained print
 * history all come from matching-engine /v1/index/*. The prose mirrors docs/index-methodology.md.
 * Every print is simulated today (`source: mock`) and the page says so. Public — no auth.
 */
import { Chart, type ChartDataset } from 'chart.js/auto'

definePageMeta({ layout: 'marketing' })
useHead({
  title: 'AI Index methodology — 1Trade',
  meta: [{ name: 'description', content: 'How the 1Trade AI Credit Index is computed, published and verified.' }],
})

interface IndexPrint {
  print_id: string
  value: string
  published_at: string
  methodology_version: string
  observation_count: number
  excluded_count?: number
  chain_hash: string
  prev_chain_hash?: string | null
  provisional: boolean
  source: 'mock' | 'computed'
}
interface Constituent { name: string; credit_type: string; weight: string; source: string; observed_at?: string | null }
interface Methodology {
  version: string
  effective_date: string
  window_minutes: number
  trim_pct: string
  volume_floor?: string
  constituents: Constituent[]
  is_mock: boolean
}

const SECTIONS = [
  { id: 'live', label: 'Current value' },
  { id: 'sec-1', label: 'What the index measures' },
  { id: 'sec-2', label: 'Inputs' },
  { id: 'sec-3', label: 'Constituents and weights' },
  { id: 'sec-4', label: 'Computation' },
  { id: 'sec-5', label: 'Audit chain' },
  { id: 'sec-6', label: 'Publication' },
  { id: 'sec-7', label: 'Changing the methodology' },
  { id: 'sec-8', label: 'Historical prints' },
  { id: 'sec-9', label: 'Known limitations' },
  { id: 'sec-10', label: 'API and contact' },
]

const latest = ref<IndexPrint | null>(null)
const nextAt = ref<string | null>(null)
const meth = ref<Methodology | null>(null)
const history = ref<IndexPrint[]>([])
const loadError = ref('')
const activeId = ref('live')
const countdown = ref('—')

/** creditsPerUsd turns a USD-per-credit print value into "credits per $1". */
function creditsPerUsd(v: string | undefined) {
  const n = Number(v)
  return n > 0 ? (1 / n).toLocaleString('en-US', { maximumFractionDigits: 1 }) : '—'
}
/** pctChange is the percent move from a to b, formatted with a sign. */
function pctChange(a?: string, b?: string) {
  const x = Number(a), y = Number(b)
  if (!(x > 0) || !(y > 0)) return null
  return ((y - x) / x) * 100
}
const change30 = computed(() => {
  const h = history.value
  return h.length > 1 ? pctChange(h[Math.max(0, h.length - 31)]!.value, h[h.length - 1]!.value) : null
})
const weightTotal = computed(() =>
  (meth.value?.constituents ?? []).reduce((s, c) => s + Number(c.weight), 0).toFixed(4),
)

/** tickCountdown counts down to the next publication the API announced. */
function tickCountdown() {
  if (!nextAt.value) return
  const diff = Math.max(0, new Date(nextAt.value).getTime() - Date.now())
  const h = Math.floor(diff / 3.6e6), m = Math.floor((diff % 3.6e6) / 6e4), s = Math.floor((diff % 6e4) / 1e3)
  countdown.value = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// Historical chart -----------------------------------------------------------
const histCanvas = ref<HTMLCanvasElement | null>(null)
let histChart: Chart<'line', number[], string> | null = null
type RangeKey = '90' | '180' | '365' | '730'
const RANGE_KEYS: RangeKey[] = ['90', '180', '365', '730']
const RANGE_LABELS: Record<RangeKey, string> = { '90': '90D', '180': '180D', '365': '1Y', '730': '2Y' }
const currentRange = ref<RangeKey>('365')

/** loadHistory fetches the print series for the chosen range and redraws the chart. */
async function loadHistory(r: RangeKey) {
  currentRange.value = r
  try {
    history.value = (await $fetch<{ prints: IndexPrint[] }>('/api/index-service/history', { query: { days: r } })).prints
  } catch {
    history.value = []
  }
  drawChart()
}

/** drawChart renders the history as credits per $1 — the unit the rest of the site quotes. */
function drawChart() {
  if (!histCanvas.value) return
  const labels = history.value.map(p => p.published_at.slice(0, 10))
  const data = history.value.map(p => 1 / Number(p.value))
  if (histChart) {
    histChart.data.labels = labels
    histChart.data.datasets[0]!.data = data
    histChart.update()
    return
  }
  const ctx = histCanvas.value.getContext('2d')!
  histChart = new Chart<'line', number[], string>(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        label: 'AI-INDEX', data, borderColor: '#0E0E0E', borderWidth: 1.4, fill: false,
        tension: 0.15, pointRadius: 0, pointHoverRadius: 3,
      } as ChartDataset<'line', number[]>],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { display: false },
        tooltip: { callbacks: { label: item => `${(item.parsed.y as number).toFixed(1)} credits per $1` } },
      },
      scales: {
        x: { grid: { display: false }, ticks: { maxTicksLimit: 8, maxRotation: 0, color: '#767269', font: { family: "'JetBrains Mono', monospace", size: 10 } } },
        y: { position: 'right', grid: { color: 'rgba(14,14,14,0.06)' }, ticks: { color: '#767269', font: { family: "'JetBrains Mono', monospace", size: 10 } } },
      },
    },
  })
}

let io: IntersectionObserver | null = null
let cdTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  try {
    const [l, m] = await Promise.all([
      $fetch<{ print: IndexPrint; next_publication_at: string }>('/api/index-service/latest'),
      $fetch<Methodology>('/api/index-service/methodology'),
    ])
    latest.value = l.print
    nextAt.value = l.next_publication_at
    meth.value = m
  } catch {
    loadError.value = 'The index service did not answer. Values will show once it is reachable.'
  }
  tickCountdown()
  cdTimer = setInterval(tickCountdown, 1000)
  await nextTick()
  loadHistory(currentRange.value)

  io = new IntersectionObserver((entries) => {
    let best: IntersectionObserverEntry | null = null
    for (const e of entries) if (e.isIntersecting && (!best || e.intersectionRatio > best.intersectionRatio)) best = e
    if (best) activeId.value = best.target.id
  }, { rootMargin: '-30% 0px -60% 0px', threshold: [0, 0.25, 0.5, 1] })
  document.querySelectorAll<HTMLElement>('section.sec, #live').forEach(s => io!.observe(s))
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
      <aside class="lnav" aria-label="Document sections">
        <div class="lnav-group">
          <div class="lnav-group-head">Methodology</div>
          <ul>
            <li v-for="i in SECTIONS" :key="i.id">
              <a :href="'#' + i.id" :class="{ active: activeId === i.id }">{{ i.label }}</a>
            </li>
          </ul>
        </div>
      </aside>

      <main class="content">
        <header class="doc-head">
          <div class="doc-eyebrow">Methodology · published document</div>
          <h1 class="doc-title">1Trade AI Credit Index methodology</h1>
          <div class="doc-version">
            <span>Version <strong>{{ meth?.version ?? '—' }}</strong></span>
            <span>Effective <strong>{{ meth?.effective_date ?? '—' }}</strong></span>
            <span class="pill"><span class="dot" /> SIMULATED PRINTS</span>
          </div>
        </header>

        <div class="callout">
          <p class="callout-p">
            <strong>The index is not yet in production.</strong> Every print served today is simulated and
            says so (<span class="mono-inline">source: mock</span>, <span class="mono-inline">provisional: true</span>).
            Do not quote a value from this index as a market price. What is real today is the audit
            chain in §5: prints are hash-chained exactly as described and can be verified independently.
          </p>
        </div>

        <p v-if="loadError" class="callout-p neg">{{ loadError }}</p>

        <div id="live" class="live-box">
          <div class="live-box-head">
            <span>— Current value · AI-INDEX</span>
            <span class="live"><span class="pulse" />Simulated · provisional</span>
          </div>
          <div class="live-box-grid">
            <div class="live-box-cell">
              <div class="lbl">Current value</div>
              <div class="val">$1 = {{ creditsPerUsd(latest?.value) }}<span class="unit">AI credits</span></div>
              <div class="sub">
                <span class="mono-inline">{{ latest?.value ?? '—' }}</span> USD per credit
                <template v-if="change30 !== null"> · 30d Δ <span :class="change30 >= 0 ? 'pos' : 'neg'">{{ change30 >= 0 ? '+' : '' }}{{ change30.toFixed(2) }}%</span></template>
              </div>
            </div>
            <div class="live-box-cell">
              <div class="lbl">Last print</div>
              <div class="val val-sm">{{ latest ? latest.published_at.slice(11, 16) + ' UTC' : '—' }}</div>
              <div class="sub">
                {{ latest?.published_at.slice(0, 10) ?? '—' }} · hash
                <span class="strong">{{ latest ? latest.chain_hash.slice(0, 10) + '…' : '—' }}</span>
              </div>
            </div>
            <div class="live-box-cell">
              <div class="lbl">Next print in</div>
              <div class="val val-md">{{ countdown }}</div>
              <div class="sub">16:00 UTC daily</div>
            </div>
          </div>
        </div>

        <section id="sec-1" class="sec">
          <div class="sec-head"><span class="sec-num">§ 1</span><h2>What the index measures</h2></div>
          <div class="sec-body">
            <p class="lede">
              The AI Credit Index is a reference price for one unit of general-purpose AI compute,
              expressed as a fixed-point USD value per AI credit. The site also shows it inverted, as
              AI credits per US dollar.
            </p>
            <p>
              It is a <strong>reference</strong> index, not a settlement price. Nothing on the platform
              is settled, margined or liquidated against it.
            </p>
          </div>
        </section>

        <section id="sec-2" class="sec">
          <div class="sec-head"><span class="sec-num">§ 2</span><h2>Inputs</h2></div>
          <div class="sec-body">
            <p>
              Real-money trading is not open, so there are no real trade prints to observe; paper
              trades are never used. Computing the index from trades is therefore impossible today, and
              inventing trade prints would be dishonest. The current phase observes platform
              transactions instead:
            </p>
            <table class="tbl" aria-label="Observation sources">
              <thead><tr><th>Observation source</th><th>What it is</th></tr></thead>
              <tbody>
                <tr><td>Realized consumption rates</td><td>USD-equivalent actually charged per consumed sub-credit</td></tr>
                <tr><td>Prepaid purchase prices</td><td>What customers paid per credit class, at purchase</td></tr>
                <tr><td>Reserved-capacity transaction prices</td><td>Agreed prices on reserved GPU capacity</td></tr>
              </tbody>
            </table>
            <p>
              When real-money trading opens, trade prints and surveillance exclusion flags join the
              observation set. The methodology version will change and the change will be disclosed.
            </p>
          </div>
        </section>

        <section id="sec-3" class="sec">
          <div class="sec-head"><span class="sec-num">§ 3</span><h2>Constituents and weights</h2></div>
          <div class="sec-body">
            <p>
              Every constituent is a canonical credit type. The schedule below is served live by
              <span class="mono-inline">GET /v1/index/methodology</span>. This document and the API are the same schedule, so any
              disagreement between them is a bug.
            </p>
            <table class="tbl" aria-label="Current constituent weights">
              <thead>
                <tr><th>Constituent</th><th>Credit type</th><th class="num-h">Weight</th><th>Source</th><th class="num-h">Observed (UTC)</th></tr>
              </thead>
              <tbody>
                <tr v-if="!meth"><td colspan="5" class="mono-td">Loading…</td></tr>
                <tr v-for="c in meth?.constituents ?? []" :key="c.credit_type">
                  <td>{{ c.name }}</td>
                  <td class="mono-td">{{ c.credit_type }}</td>
                  <td class="num">{{ c.weight }}</td>
                  <td class="mono-td">{{ c.source }}</td>
                  <td class="num">{{ c.observed_at ? c.observed_at.slice(11, 19) : '—' }}</td>
                </tr>
                <tr v-if="meth" class="total"><td>Total</td><td /><td class="num">{{ weightTotal }}</td><td colspan="2" /></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section id="sec-4" class="sec">
          <div class="sec-head"><span class="sec-num">§ 4</span><h2>Computation</h2></div>
          <div class="sec-body">
            <ol>
              <li>Collect observations per constituent within a <strong>{{ meth?.window_minutes ?? '—' }}-minute window</strong> ending at publication. A constituent with no observation in its window is excluded from that print.</li>
              <li>Apply the <strong>volume floor</strong>: an observation below <span class="mono-inline">{{ meth?.volume_floor ?? '—' }}</span> in notional is discarded as too thin to be price-forming.</li>
              <li>Take a <strong>{{ meth?.trim_pct ?? '—' }}% trimmed mean</strong> per constituent, removing the extreme tails so a single unusual transaction cannot move the print.</li>
              <li>Combine constituents by the §3 weights.</li>
              <li>Round to 6 decimal places. All index arithmetic is fixed-point, never floating point.</li>
            </ol>
            <p>Every print reports <span class="mono-inline">observation_count</span> and <span class="mono-inline">excluded_count</span>, so the sample behind a value is visible.</p>
          </div>
        </section>

        <section id="sec-5" class="sec">
          <div class="sec-head"><span class="sec-num">§ 5</span><h2>Audit chain</h2></div>
          <div class="sec-body">
            <p>Prints are immutable and hash-chained:</p>
            <div class="formula-box"><div class="formula-eq mono-inline">chain_hash = SHA256( prev_chain_hash ‖ canonical_json(print) )</div></div>
            <p>
              <span class="mono-inline">canonical_json</span> uses a fixed key order (print_id, value, published_at,
              methodology_version, observation_count, excluded_count, provisional, source), values to exactly
              6 decimals and RFC 3339 UTC timestamps. The genesis print chains from the empty string.
            </p>
            <p>
              <strong>To verify:</strong> fetch a series from <span class="mono-inline">GET /v1/index/history</span>,
              re-derive each hash from the previous print's hash and the canonical form, and compare.
              Altering any print breaks every hash after it. The credit ledger uses the same construction.
            </p>
          </div>
        </section>

        <section id="sec-6" class="sec">
          <div class="sec-head"><span class="sec-num">§ 6</span><h2>Publication</h2></div>
          <div class="sec-body">
            <ul>
              <li>Daily at <strong>16:00 UTC</strong>. Every response states the next publication time.</li>
              <li>A print is <span class="mono-inline">provisional</span> until it is finalised. Today every print is provisional.</li>
              <li>Prints are never edited. A correction is a new print citing the one it supersedes; the superseded print stays in the chain.</li>
            </ul>
          </div>
        </section>

        <section id="sec-7" class="sec">
          <div class="sec-head"><span class="sec-num">§ 7</span><h2>Changing the methodology</h2></div>
          <div class="sec-body">
            <p>
              A methodology change is proposed in writing with its rationale and expected effect,
              reviewed, published here with a new version number and effective date, and only then
              deployed. A version change is never silent: each print states the
              <span class="mono-inline">methodology_version</span> that produced it.
            </p>
            <table class="tbl" aria-label="Version history">
              <thead><tr><th>Version</th><th>Effective</th><th>Change</th></tr></thead>
              <tbody>
                <tr><td class="mono-td">1.2</td><td class="mono-td">2026-04-01</td><td>Current. Six constituents, 120-minute window, 95% trim, 25,000 volume floor.</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section id="sec-8" class="sec">
          <div class="sec-head"><span class="sec-num">§ 8</span><h2>Historical prints</h2></div>
          <div class="sec-body">
            <div class="chart-card">
              <div class="chart-head">
                <h3 class="chart-ttl">AI-INDEX · daily prints<span class="chart-meta">· {{ history.length }} prints · simulated</span></h3>
                <div class="ranges">
                  <button v-for="k in RANGE_KEYS" :key="k" type="button" class="r" :class="{ active: currentRange === k }" @click="loadHistory(k)">{{ RANGE_LABELS[k] }}</button>
                </div>
              </div>
              <div class="chart-body">
                <div class="chart-canvas-wrap"><canvas ref="histCanvas" /></div>
              </div>
              <div class="chart-foot">
                <span>Credits per $1 · each print hash-chained</span>
                <a :href="`/api/index-service/history?days=${currentRange}`" target="_blank" rel="noopener">Raw prints (JSON) →</a>
              </div>
            </div>
          </div>
        </section>

        <section id="sec-9" class="sec">
          <div class="sec-head"><span class="sec-num">§ 9</span><h2>Known limitations</h2></div>
          <div class="sec-body">
            <ul>
              <li><strong>Values are simulated today.</strong> <span class="mono-inline">source: mock</span> marks every print.</li>
              <li><strong>No trade prints.</strong> The current phase observes platform transactions only (§2).</li>
              <li><strong>Thin history.</strong> The platform is young, so early observation counts are small.</li>
              <li><strong>No independent audit yet.</strong> The chain is verifiable by construction, but no third party has attested to it.</li>
              <li><strong>Single operator.</strong> 1Trade computes an index over its own platform's transactions. Publishing the methodology, weights, observation counts and audit chain mitigates that conflict; it does not remove it.</li>
              <li><strong>Administered prices are not discovered prices.</strong> Today's observations are at prices 1Trade set, so an index over them would largely restate that price table. That is why the index stays labelled simulated until prices vary by transaction or real trade prints exist.</li>
            </ul>
          </div>
        </section>

        <section id="sec-10" class="sec">
          <div class="sec-head"><span class="sec-num">§ 10</span><h2>API and contact</h2></div>
          <div class="sec-body">
            <table class="tbl" aria-label="Index endpoints">
              <tbody>
                <tr><td class="mono-td">GET /v1/index/latest</td><td>Current print and next publication time</td></tr>
                <tr><td class="mono-td">GET /v1/index/history?days=N</td><td>Print series, oldest first, hash-chained</td></tr>
                <tr><td class="mono-td">GET /v1/index/methodology</td><td>Live constituent and weight schedule</td></tr>
              </tbody>
            </table>
            <p>Questions or concerns about the methodology: <a href="mailto:methodology@1trade.com">methodology@1trade.com</a>.</p>
          </div>
        </section>
      </main>

      <aside class="rtoc" aria-label="On this page">
        <div class="rtoc-head">— On this page</div>
        <ul>
          <li v-for="i in SECTIONS" :key="i.id">
            <a :href="'#' + i.id" :class="{ active: activeId === i.id }">{{ i.label }}</a>
          </li>
        </ul>
        <div class="rtoc-meta">
          <strong>v{{ meth?.version ?? '—' }}</strong> · Effective {{ meth?.effective_date ?? '—' }}
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.methodology {
  --t-4: #B5AEA1;
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
.neg { color: var(--neg); }
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
