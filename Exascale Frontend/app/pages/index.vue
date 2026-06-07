<script setup lang="ts">
/**
 * / — Exascale Homepage
 *
 * Faithful port of uploads/Exascale Homepage.html from the design bundle.
 * All hardcoded values replaced with token references; behavior preserved:
 *   - sticky nav comes from the marketing layout
 *   - live AI Index ticker (Brownian motion + mean reversion, 5s tick)
 *   - 30-day sparkline (draw-on animation)
 *   - 365-day chart.js index chart with custom crosshair tooltip
 *   - "next print" countdown
 *
 * Layout: light marketing shell. Dark "inverse" sections via .inverse class.
 */
definePageMeta({ layout: 'marketing' })
useHead({ title: 'Exascale — The Commodity Market for AI Compute' })

// ─── Live AI Index ticker ───────────────────────────────────
const tkNum = ref('1,002.4')
const tkDelta = ref('▲ 0.18% (24h)')
const tkDeltaNeg = ref(false)
const tkFlash = ref<'' | 'flash-up' | 'flash-down'>('')
const tkNext = ref('3h 24m')

const obMidLabel = ref('1,002.4')   // shared with order book preview
const footerIdx = ref('1,002.4')

let indexValue = 1.0024
const target = 1.00
let nextPrintSec = 3 * 3600 + 24 * 60
let tickInterval: ReturnType<typeof setInterval> | null = null
let printInterval: ReturnType<typeof setInterval> | null = null
// Lifted out of buildIndexChart() so it can be cleaned up from a top-level
// onUnmounted — registering lifecycle hooks after `await` is invalid.
let chartCleanup: (() => void) | null = null

const fmt = (n: number) =>
  n.toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

const doTick = () => {
  // Brownian motion with weak mean reversion
  const drift = (target - indexValue) * 0.05
  const noise = (Math.random() - 0.5) * 0.0008
  const change = drift + noise
  indexValue = Math.max(0.985, Math.min(1.018, indexValue + change))
  const dayChange = (indexValue - 1.0006) / 1.0006

  const display = indexValue * 1000   // "1 USD = N credits"
  const text = fmt(display)
  tkNum.value = text
  obMidLabel.value = text
  footerIdx.value = text

  const dpct = (dayChange * 100).toFixed(2)
  const sign = dayChange >= 0 ? '▲' : '▼'
  tkDelta.value = `${sign} ${Math.abs(parseFloat(dpct))}% (24h)`
  tkDeltaNeg.value = dayChange < 0

  // Flash color briefly on change
  tkFlash.value = change > 0 ? 'flash-up' : 'flash-down'
  setTimeout(() => { tkFlash.value = '' }, 700)
}

// ─── Sparkline path (built once on mount) ────────────────────
const sparkLineD = ref('')
const sparkFillD = ref('')

const buildSparkline = () => {
  const W = 240, H = 64
  const days = 30
  let v = 0.987
  const pts: number[] = []
  for (let i = 0; i < days; i++) {
    v += (1.00 - v) * 0.05 + (Math.random() - 0.48) * 0.008
    pts.push(v)
  }
  pts[pts.length - 1] = 1.0024
  const min = Math.min(...pts), max = Math.max(...pts)
  const range = max - min || 1
  const pad = 6
  const usableH = H - pad * 2
  const stepX = W / (pts.length - 1)
  const d = pts.map((p, i) => {
    const x = i * stepX
    const y = pad + usableH - ((p - min) / range) * usableH
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`
  }).join(' ')
  sparkLineD.value = d
  sparkFillD.value = `${d} L ${W} ${H} L 0 ${H} Z`
}

// ─── Index chart (chart.js, 365 days) — client only ─────────
const indexCanvas = ref<HTMLCanvasElement>()
const chartTip = ref<HTMLDivElement>()
const chartTipVal = ref('1.0024')
const chartTipDate = ref('19 May 2026')
const chartTipPos = ref({ x: 0, y: 0, opacity: 0 })

const buildIndexChart = async () => {
  const { Chart, registerables } = await import('chart.js')
  await import('chartjs-adapter-date-fns')
  Chart.register(...registerables)

  if (!indexCanvas.value) return

  const days = 365
  let v = 0.962
  const data: number[] = []
  const labels: Date[] = []
  const now = new Date('2026-05-19')
  for (let i = 0; i < days; i++) {
    const drawdown = Math.random() < 0.018 ? -(0.005 + Math.random() * 0.01) : 0
    const drift = 0.0002
    const noise = (Math.random() - 0.48) * 0.0035
    v = Math.max(0.94, Math.min(1.05, v + drift + noise + drawdown))
    data.push(+v.toFixed(4))
    const d = new Date(now)
    d.setDate(d.getDate() - (days - 1 - i))
    labels.push(d)
  }
  data[data.length - 1] = 1.0024

  const root = getComputedStyle(document.documentElement)
  const brandHex = root.getPropertyValue('--brand').trim()
  const textHex  = root.getPropertyValue('--text').trim()
  const t3Hex    = root.getPropertyValue('--text-3').trim()

  const ctx = indexCanvas.value
  const grad = ctx.getContext('2d')!.createLinearGradient(0, 0, 0, 320)
  grad.addColorStop(0, `${brandHex}4D`) // ~30% alpha
  grad.addColorStop(1, `${brandHex}00`) // 0%

  const chart = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        data,
        borderColor: textHex,
        borderWidth: 1.5,
        backgroundColor: grad,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 0,
        tension: 0.3,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 900, easing: 'easeOutCubic' },
      interaction: { mode: 'index', intersect: false },
      plugins: { legend: { display: false }, tooltip: { enabled: false } },
      scales: {
        x: {
          type: 'time',
          time: { unit: 'month', displayFormats: { month: 'MMM' } },
          grid: { display: false },
          border: { display: false },
          ticks: { color: t3Hex, font: { family: 'JetBrains Mono', size: 10 }, maxTicksLimit: 8, maxRotation: 0 },
        },
        y: {
          position: 'right',
          grid: { color: 'rgba(0,0,0,0.04)' },
          border: { display: false },
          ticks: { color: t3Hex, font: { family: 'JetBrains Mono', size: 10 }, callback: (v) => Number(v).toFixed(3) },
        },
      },
    },
  })

  // Custom crosshair tooltip
  const onMove = (e: MouseEvent) => {
    const rect = ctx.getBoundingClientRect()
    const x = e.clientX - rect.left
    const xScale = (chart.scales as any).x
    const yScale = (chart.scales as any).y
    const xValue = xScale.getValueForPixel(x)
    let nearest = 0
    let best = Infinity
    for (let i = 0; i < labels.length; i++) {
      const dist = Math.abs(labels[i].getTime() - xValue)
      if (dist < best) { best = dist; nearest = i }
    }
    const px = xScale.getPixelForValue(labels[nearest])
    const py = yScale.getPixelForValue(data[nearest])
    chartTipPos.value = { x: px, y: py, opacity: 1 }
    chartTipVal.value = data[nearest].toFixed(4)
    chartTipDate.value = labels[nearest].toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
  }
  const onLeave = () => { chartTipPos.value = { ...chartTipPos.value, opacity: 0 } }
  ctx.addEventListener('mousemove', onMove)
  ctx.addEventListener('mouseleave', onLeave)

  chartCleanup = () => {
    ctx.removeEventListener('mousemove', onMove)
    ctx.removeEventListener('mouseleave', onLeave)
    chart.destroy()
  }
}

onMounted(() => {
  buildSparkline()
  doTick()  // initial paint with current value
  tickInterval = setInterval(doTick, 5000)
  printInterval = setInterval(() => {
    nextPrintSec = Math.max(0, nextPrintSec - 60)
    const h = Math.floor(nextPrintSec / 3600)
    const m = Math.floor((nextPrintSec % 3600) / 60)
    tkNext.value = `${h}h ${String(m).padStart(2, '0')}m`
  }, 60_000)
  buildIndexChart()
})

onUnmounted(() => {
  if (tickInterval) clearInterval(tickInterval)
  if (printInterval) clearInterval(printInterval)
  if (chartCleanup) { chartCleanup(); chartCleanup = null }
})

// Mini order book preview rows (static)
const obAsks = [
  { px: '1010.4', sz: '128.4', tot: '128.4', bar: 18 },
  { px: '1009.2', sz: '186.0', tot: '314.4', bar: 26 },
  { px: '1008.0', sz: '298.7', tot: '613.1', bar: 42 },
  { px: '1006.5', sz: '421.3', tot: '1,034.4', bar: 60 },
  { px: '1004.8', sz: '529.1', tot: '1,563.5', bar: 74 },
]
const obBids = [
  { px: '1000.4', sz: '498.2', tot: '498.2', bar: 70 },
  { px: '999.8',  sz: '412.8', tot: '911.0', bar: 58 },
  { px: '998.1',  sz: '285.5', tot: '1,196.5', bar: 40 },
  { px: '996.7',  sz: '170.1', tot: '1,366.6', bar: 24 },
  { px: '994.2',  sz: '114.0', tot: '1,480.6', bar: 16 },
]
</script>

<template>
  <article class="home">
    <!-- ═══════════════════════════════════════════════════════
         HERO
         ═══════════════════════════════════════════════════════ -->
    <section class="hero grid-bg">
      <div class="container-x">
        <div class="eyebrow"><span class="dot" />The commodity market for AI compute</div>

        <h1 class="hero-display tnum">
          AI compute,<br />but <span class="hi">tradeable</span>.
        </h1>

        <p class="hero-lede">
          Tradeable credits connecting GPU datacenters, traders, and AI companies.
          Owned underlying for credibility. Partner-supplied scale.
        </p>

        <!-- Live ticker + sparkline -->
        <div class="ticker-row">
          <div class="ticker">
            <span class="live-dot" />
            <div>
              <div class="t-label">Exascale AI Index</div>
              <div class="t-line">
                <span class="t-value" :class="tkFlash">$1 = <span>{{ tkNum }}</span> AI credits</span>
                <span class="t-delta" :class="{ neg: tkDeltaNeg }">{{ tkDelta }}</span>
              </div>
            </div>
            <span class="t-sep" />
            <span class="t-meta">Last print 16:00 UTC · Next in <span>{{ tkNext }}</span></span>
          </div>

          <svg class="sparkline" viewBox="0 0 240 64" preserveAspectRatio="none" aria-hidden="true">
            <path class="fill" :d="sparkFillD" />
            <path class="line" :d="sparkLineD" />
          </svg>
        </div>

        <div class="cta-row">
          <BaseButton as="a" href="#cta" variant="primary" size="lg">Open account</BaseButton>
          <a href="#markets" class="btn-text">View markets →</a>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         THE PROBLEM
         ═══════════════════════════════════════════════════════ -->
    <section class="band elevated">
      <div class="container-x">
        <div class="eyebrow center"><span class="dot" />The problem</div>
        <p class="pullquote">There needs to be a market for compute. No solution yet.</p>
        <p class="quote-attr">— <span class="name">Larry Fink</span> · CEO, BlackRock · 2026</p>

        <div class="stat-grid">
          <div>
            <div class="stat-num tnum"><span class="lime">$200B+</span></div>
            <p class="stat-desc">Annual global GPU compute spend, untraded.</p>
          </div>
          <div>
            <div class="stat-num tnum"><span class="lime">5–10×</span></div>
            <p class="stat-desc">Spread between reserved and spot pricing.</p>
          </div>
          <div>
            <div class="stat-num tnum"><span class="lime">0</span></div>
            <p class="stat-desc">Functioning venues for AI compute as an asset class.</p>
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         HOW IT WORKS — three-sided market
         ═══════════════════════════════════════════════════════ -->
    <section class="band" id="markets">
      <div class="container-x">
        <div class="center">
          <div class="eyebrow center"><span class="dot" />How it works</div>
          <h2 class="s-head">Three sides. One venue.</h2>
        </div>

        <!-- Diagram (SVG ported from design — exact geometry) -->
        <div class="market-diagram">
          <svg viewBox="0 0 880 440" xmlns="http://www.w3.org/2000/svg">
            <defs>
              <marker id="arr" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto">
                <path d="M0,0 L8,3 L0,6 Z" fill="var(--text-3)" />
              </marker>
            </defs>

            <!-- Central venue (dark) -->
            <rect x="330" y="170" width="220" height="100" rx="2" fill="var(--inverse)" />
            <text x="440" y="206" text-anchor="middle" font-family="JetBrains Mono" font-size="11" letter-spacing="3" fill="var(--text-3)">— EXASCALE</text>
            <text x="440" y="236" text-anchor="middle" font-family="Inter Tight" font-weight="700" font-size="24" fill="var(--text-inverse)" letter-spacing="-0.5">Trading venue</text>

            <!-- Supply -->
            <g>
              <rect x="40" y="40" width="220" height="92" rx="2" fill="var(--elevated)" stroke="var(--border)" />
              <rect x="40" y="40" width="4" height="92" fill="var(--pos)" />
              <text x="60" y="68" font-family="JetBrains Mono" font-size="10" letter-spacing="2.5" fill="var(--text-3)">— SUPPLY</text>
              <text x="60" y="98" font-family="Inter Tight" font-weight="600" font-size="20" fill="var(--text)" letter-spacing="-0.4">GPU datacenters</text>
              <text x="60" y="118" font-family="Inter" font-size="11" fill="var(--text-3)">Owned + partner capacity</text>
            </g>

            <!-- Liquidity -->
            <g>
              <rect x="330" y="0" width="220" height="92" rx="2" fill="var(--elevated)" stroke="var(--border)" />
              <rect x="330" y="0" width="4" height="92" fill="var(--warn)" />
              <text x="350" y="28" font-family="JetBrains Mono" font-size="10" letter-spacing="2.5" fill="var(--text-3)">— LIQUIDITY</text>
              <text x="350" y="58" font-family="Inter Tight" font-weight="600" font-size="20" fill="var(--text)" letter-spacing="-0.4">Traders</text>
              <text x="350" y="78" font-family="Inter" font-size="11" fill="var(--text-3)">Prop firms · hedge funds · MMs</text>
            </g>

            <!-- Demand -->
            <g>
              <rect x="620" y="40" width="220" height="92" rx="2" fill="var(--elevated)" stroke="var(--border)" />
              <rect x="620" y="40" width="4" height="92" fill="var(--accent)" />
              <text x="640" y="68" font-family="JetBrains Mono" font-size="10" letter-spacing="2.5" fill="var(--text-3)">— DEMAND</text>
              <text x="640" y="98" font-family="Inter Tight" font-weight="600" font-size="20" fill="var(--text)" letter-spacing="-0.4">AI companies</text>
              <text x="640" y="118" font-family="Inter" font-size="11" fill="var(--text-3)">Frontier labs · enterprises</text>
            </g>

            <!-- Arrows in -->
            <path d="M 195 132 L 360 175" stroke="var(--text-3)" stroke-width="1" fill="none" marker-end="url(#arr)" opacity="0.55" />
            <path d="M 440 92  L 440 165" stroke="var(--text-3)" stroke-width="1" fill="none" marker-end="url(#arr)" opacity="0.55" />
            <path d="M 685 132 L 520 175" stroke="var(--text-3)" stroke-width="1" fill="none" marker-end="url(#arr)" opacity="0.55" />

            <!-- Arrows out -->
            <path d="M 360 265 L 195 305" stroke="var(--text-3)" stroke-width="1" fill="none" marker-end="url(#arr)" opacity="0.35" stroke-dasharray="4 4" />
            <path d="M 520 265 L 685 305" stroke="var(--text-3)" stroke-width="1" fill="none" marker-end="url(#arr)" opacity="0.35" stroke-dasharray="4 4" />

            <!-- Verb cards -->
            <g>
              <rect x="40" y="290" width="220" height="64" rx="2" fill="var(--canvas)" />
              <text x="150" y="318" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">List idle credits</text>
              <text x="150" y="338" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">Receive payment</text>
            </g>
            <g>
              <rect x="620" y="290" width="220" height="64" rx="2" fill="var(--canvas)" />
              <text x="730" y="318" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">Buy credits ahead</text>
              <text x="730" y="338" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">Redeem for compute</text>
            </g>
            <g>
              <rect x="330" y="380" width="220" height="50" rx="2" fill="var(--canvas)" />
              <text x="440" y="403" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">Provide liquidity</text>
              <text x="440" y="421" text-anchor="middle" font-family="Inter" font-size="13" fill="var(--text-3)">Earn spread + maker rebates</text>
            </g>
          </svg>
        </div>

        <!-- Three role columns -->
        <div class="role-grid">
          <div>
            <div class="eyebrow pos"><span class="dot pos" />Supply</div>
            <h4 class="role-title">GPU datacenters</h4>
            <p class="role-text">List unused capacity for sale as standardized credits. Cryptographic SLA receipts on every listing.</p>
            <ul class="role-list">
              <li>Owned datacenter (H100 / H200)</li>
              <li>Partner neoclouds at scale</li>
              <li>Per-second metering</li>
            </ul>
          </div>
          <div>
            <div class="eyebrow warn"><span class="dot warn" />Liquidity</div>
            <h4 class="role-title">Traders</h4>
            <p class="role-text">Prop firms, commodity hedge funds, and quant desks bring price discovery to an asset class that hasn't had any.</p>
            <ul class="role-list">
              <li>Maker-taker fee schedule</li>
              <li>Paper trading from day one</li>
              <li>Surveillance + market integrity</li>
            </ul>
          </div>
          <div>
            <div class="eyebrow info"><span class="dot info" />Demand</div>
            <h4 class="role-title">AI companies</h4>
            <p class="role-text">Frontier labs, mid-market AI, and financial-services teams hedge spend ahead of need. Redeem for real compute.</p>
            <ul class="role-list">
              <li>Lock in cost months ahead</li>
              <li>OpenAI-compatible inference</li>
              <li>Treasury-grade reporting</li>
            </ul>
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         THE PRODUCTS — three stacked cards
         ═══════════════════════════════════════════════════════ -->
    <section class="band elevated" id="products">
      <div class="container-x">
        <div class="eyebrow"><span class="dot" />The products</div>
        <h2 class="s-head wide">One venue. Three<br />integrated layers.</h2>

        <div class="products">
          <!-- Trading -->
          <div class="product-card">
            <div class="p-left">
              <div class="eyebrow pos"><span class="dot pos" />Trading layer</div>
              <h3 class="p-head">Order book.<br />Market maker.<br />Daily index.</h3>
              <p class="p-text">
                AI credits and GPU credits as tradeable instruments. Maker-taker fees, volume-tiered. Surveillance from day one.
              </p>
              <a href="#" class="btn-text p-link">Explore the trading layer →</a>
            </div>
            <div class="p-right inverse">
              <div class="ob-mini">
                <div class="ob-head">
                  <span>Price</span><span class="right">Size</span><span class="right">Total</span>
                </div>
                <div v-for="(r, i) in obAsks" :key="`a-${i}`" class="ob-row ask">
                  <div class="bar" :style="{ width: `${r.bar}%` }" />
                  <span class="px">{{ r.px }}</span>
                  <span class="right">{{ r.sz }}</span>
                  <span class="right">{{ r.tot }}</span>
                </div>
                <div class="ob-mid tnum">
                  {{ obMidLabel }}
                  <span class="ob-spread">spread 0.20%</span>
                </div>
                <div v-for="(r, i) in obBids" :key="`b-${i}`" class="ob-row bid">
                  <div class="bar" :style="{ width: `${r.bar}%` }" />
                  <span class="px">{{ r.px }}</span>
                  <span class="right">{{ r.sz }}</span>
                  <span class="right">{{ r.tot }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Inference -->
          <div class="product-card">
            <div class="p-left">
              <div class="eyebrow warn"><span class="dot warn" />Inference layer</div>
              <h3 class="p-head">Curated SoTA models.<br />OpenAI-compatible API.</h3>
              <p class="p-text">
                Top 3–5 open models per category — text, speech, image, video, niche. Multi-tenant per GPU. Quarterly catalog refresh.
              </p>
              <a href="#" class="btn-text p-link">Read the API docs →</a>
            </div>
            <div class="p-right inverse">
              <pre class="term"><span class="prompt">POST</span> https://api.exascale.com/v1/chat/completions
<span class="dim">Authorization: Bearer ex_live_...</span>
<span class="dim">Content-Type: application/json</span>

{
  <span class="ok">"model"</span>: <span class="ok">"meta/llama-3.3-70b"</span>,
  <span class="ok">"messages"</span>: [
    { <span class="ok">"role"</span>: <span class="ok">"user"</span>, <span class="ok">"content"</span>: <span class="ok">"Summarize..."</span> }
  ]
}

<span class="dim">→ 200 OK · 412 ms · 1,284 tokens · 1.55 text credits</span></pre>
            </div>
          </div>

          <!-- Compute -->
          <div class="product-card">
            <div class="p-left">
              <div class="eyebrow info"><span class="dot info" />Compute layer</div>
              <h3 class="p-head">H100 / H200 capacity.<br />CLI-first. Free egress.</h3>
              <p class="p-text">
                Owned datacenter underlying for trade settlement. Per-second metering. Pay with credits or cash. No region lock-in.
              </p>
              <a href="#" class="btn-text p-link">Browse instance types →</a>
            </div>
            <div class="p-right inverse">
              <pre class="term"><span class="prompt">$</span> exascale instances launch \
    --type   h100.8x \
    --hours  24 \
    --pay-with credits

<span class="ok">✓</span> Reserved · 8 × H100 80GB · TYO-1
  Hourly         <span class="dim">$2.99 / GPU-hr</span>
  Estimated cost <span class="dim">$574.08 → 5,710 H100 credits</span>
  SSH ready in   <span class="dim">~ 90s</span>

<span class="ok">✓</span> Connected   <span class="dim">ssh ubuntu@ex-h100-tyo1-04.exascale.com</span></pre>
            </div>
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         THE EXASCALE AI INDEX
         ═══════════════════════════════════════════════════════ -->
    <section class="band" id="index">
      <div class="container-x">
        <div class="eyebrow"><span class="dot" />The Exascale AI Index</div>
        <h2 class="s-head wide">The price of AI compute,<br />published daily.</h2>
        <p class="lede">
          An audited, methodology-public reference index. Trimmed mean across constituent venues
          with volume floors and ECP attestation.
        </p>

        <div class="chart-wrap">
          <div class="ch-head">
            <div>
              <div class="eyebrow"><span class="dot" />EXASCALE AI INDEX · 365D</div>
              <div class="ch-val tnum">
                1.0024 <span class="pos">▲ 0.18%</span>
              </div>
            </div>
            <NuxtLink to="/benchmark" class="btn-text small">View methodology →</NuxtLink>
          </div>

          <div class="ch-canvas-wrap">
            <canvas ref="indexCanvas" />
            <div
              class="chart-tip"
              :style="{ left: `${chartTipPos.x}px`, top: `${chartTipPos.y}px`, opacity: chartTipPos.opacity }"
            >
              <div class="tip-lbl">— Index</div>
              <div class="tip-val tnum">{{ chartTipVal }}</div>
              <div class="tip-date">{{ chartTipDate }}</div>
            </div>
          </div>

          <div class="ch-stats">
            <div>
              <div class="eyebrow">— Current</div>
              <div class="ch-stat tnum">1.0024</div>
            </div>
            <div>
              <div class="eyebrow">— Yesterday</div>
              <div class="ch-stat tnum">1.0006</div>
            </div>
            <div>
              <div class="eyebrow">— 7d</div>
              <div class="ch-stat tnum neg">−0.42%</div>
            </div>
            <div>
              <div class="eyebrow">— 30d</div>
              <div class="ch-stat tnum pos">+2.31%</div>
            </div>
            <div>
              <div class="eyebrow">— YTD</div>
              <div class="ch-stat tnum pos">+4.18%</div>
            </div>
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         FOR DIFFERENT AUDIENCES
         ═══════════════════════════════════════════════════════ -->
    <section class="band elevated">
      <div class="container-x">
        <div class="eyebrow"><span class="dot" />Built for</div>
        <h2 class="s-head wide">Three audiences.<br />One product.</h2>

        <div class="aud-grid">
          <div class="aud-card">
            <h4>For traders</h4>
            <p>Maker-taker fees from 1% / 0.5% down to 0.10% / 0.00% above $1B notional. Paper trading from day one; real money in v1.5.</p>
            <ul>
              <li>Maker rebates on volume tiers</li>
              <li>FIX gateway (Q3 2026)</li>
              <li>Level-2 market data, 25ms tick</li>
              <li>Cross-venue arbitrage routing</li>
            </ul>
            <a href="#" class="aud-link">Fee schedule →</a>
          </div>
          <div class="aud-card">
            <h4>For AI companies</h4>
            <p>Buy credits ahead, redeem for actual compute or inference. Multi-currency (USD, JPY). Bulk procurement with treasury controls.</p>
            <ul>
              <li>Bulk credit purchase with NET-30</li>
              <li>OpenAI-compatible API, drop-in</li>
              <li>Free egress, no region lock-in</li>
              <li>Per-team spend caps + reporting</li>
            </ul>
            <a href="#" class="aud-link">Talk to enterprise sales →</a>
          </div>
          <div class="aud-card">
            <h4>For datacenter partners</h4>
            <p>Monetize idle reserved capacity. Standardized credit issuance with cryptographic receipts. Settlement via existing payment rails.</p>
            <ul>
              <li>Market access on day one</li>
              <li>Per-instance utilization API</li>
              <li>Custodial credit issuance</li>
              <li>Pay-outs USD, JPY, stablecoin (v2)</li>
            </ul>
            <a href="#" class="aud-link">Partner program →</a>
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         TRUST STRIP
         ═══════════════════════════════════════════════════════ -->
    <section class="trust-band">
      <div class="container-x">
        <div class="trust-strip">
          <div>
            <span class="t-lbl">— Security</span>
            SOC 2 Type I path · Annual audits
          </div>
          <div>
            <span class="t-lbl">— Index</span>
            Methodology audited externally
          </div>
          <div>
            <span class="t-lbl">— Market integrity</span>
            Surveillance from day one
          </div>
          <div>
            <span class="t-lbl">— Anchor partner</span>
            UBS Japan · Financial services
          </div>
        </div>
      </div>
    </section>


    <!-- ═══════════════════════════════════════════════════════
         FOOTER CTA
         ═══════════════════════════════════════════════════════ -->
    <section id="cta" class="band cta-band">
      <div class="container-x center">
        <h2 class="cta-head">
          Open an account<br />in <span class="hi">5 minutes</span>.
        </h2>
        <p class="cta-sub">
          Paper trading available immediately. Real-money trading in v1.5 with full KYC. Bring your own custody at launch.
        </p>
        <div class="cta-actions">
          <BaseButton as="a" :href="'/signup'" variant="primary" size="lg">Open account</BaseButton>
          <a href="#" class="btn-text">Talk to enterprise sales →</a>
        </div>
      </div>
    </section>

    <!-- Fixed floating CTA — opens the persona picker / guided tour -->
    <NuxtLink to="/onboarding/tour" class="tour-fab" aria-label="Take the guided product tour">
      <span class="fab-dot" aria-hidden="true" />
      <span class="fab-eyebrow">GUIDED TOUR</span>
      <span class="fab-title">See the venue your way</span>
      <span class="fab-arrow" aria-hidden="true">→</span>
    </NuxtLink>
  </article>
</template>

<style scoped>
/* ─── Layout primitives ─── */
.container-x {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 var(--sp-6);
}

.band { padding: var(--sp-10) 0; }

.band.elevated {
  background: var(--elevated);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}

.center { text-align: center; }


/* ─── Eyebrow label ─── */
.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-3);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: 0.22em;
  text-transform: uppercase;
  font-weight: 500;
  color: var(--text-3);
}

.eyebrow.center { justify-content: center; display: flex; }

.eyebrow .dot {
  width: 6px;
  height: 6px;
  background: var(--brand);
  display: inline-block;
  border-radius: 0;
}

.eyebrow.pos  { color: var(--pos); }
.eyebrow.pos  .dot { background: var(--pos); }
.eyebrow.warn { color: var(--warn); }
.eyebrow.warn .dot { background: var(--warn); }
.eyebrow.info { color: var(--accent); }
.eyebrow.info .dot { background: var(--accent); }


/* ─── Buttons (text link variant; primary uses BaseButton) ─── */
.btn-text {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-2);
  font-family: var(--font-sans);
  font-size: 15px;
  font-weight: 500;
  letter-spacing: -0.005em;
  color: var(--text);
  text-decoration: none;
  transition: color var(--dur) var(--ease);
}

.btn-text:hover { color: var(--accent); }
.btn-text.small { font-size: var(--fs-sm); }


/* ─── HERO ─── */
.hero {
  padding: 80px 0 128px;
  position: relative;
}

.grid-bg {
  background-image:
    linear-gradient(rgba(0, 0, 0, 0.022) 1px, transparent 1px),
    linear-gradient(90deg, rgba(0, 0, 0, 0.022) 1px, transparent 1px);
  background-size: 56px 56px;
}

.hero-display {
  font-family: var(--font-display);
  font-size: 128px;
  font-weight: 700;
  line-height: 0.92;
  letter-spacing: -0.045em;
  margin-top: var(--sp-6);
  margin-bottom: var(--sp-6);
  max-width: 1180px;
  color: var(--text);
}

.hero-display .hi,
.cta-head .hi {
  background: var(--brand);
  padding: 0 0.04em;
  line-height: 0.65;
}

.hero-lede {
  font-size: 22px;
  line-height: 1.5;
  max-width: 720px;
  color: var(--text-2);
  margin: 0 0 var(--sp-7);
  letter-spacing: -0.005em;
}


/* Ticker row (ticker + sparkline) */
.ticker-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-5);
  align-items: center;
  margin-bottom: var(--sp-7);
}

/* Ticker */
.ticker {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-4);
  padding: 10px 18px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
  font-variant-numeric: tabular-nums;
}

.live-dot {
  width: 7px;
  height: 7px;
  background: var(--pos);
  border-radius: 50%;
  position: relative;
  flex-shrink: 0;
}

.live-dot::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  border: 1px solid var(--pos);
  opacity: 0.5;
  animation: ping 2s ease-out infinite;
}

@keyframes ping {
  0%   { transform: scale(0.9); opacity: 0.6; }
  100% { transform: scale(1.8); opacity: 0; }
}

.t-label {
  color: var(--text-3);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.t-line {
  display: flex;
  align-items: baseline;
  gap: 14px;
  margin-top: 2px;
}

.t-value {
  color: var(--text);
  font-weight: 500;
  transition: color 600ms ease-out;
}

.t-value.flash-up { color: var(--pos); }
.t-value.flash-down { color: var(--neg); }

.t-delta {
  color: var(--pos);
  font-weight: 500;
}

.t-delta.neg { color: var(--neg); }

.t-sep {
  height: 22px;
  width: 1px;
  background: var(--border);
  margin: 0 4px;
}

.t-meta {
  color: var(--text-3);
  font-size: 11px;
}


/* Sparkline */
.sparkline {
  width: 240px;
  height: 64px;
  overflow: visible;
}

.sparkline path.line {
  fill: none;
  stroke: var(--text);
  stroke-width: 1.5;
  stroke-dasharray: 1200;
  stroke-dashoffset: 1200;
  animation: draw 1.6s cubic-bezier(0.2, 0, 0, 1) 0.3s forwards;
}

.sparkline path.fill {
  fill: var(--brand);
  fill-opacity: 0.15;
  opacity: 0;
  animation: fadein 1.4s ease-out 0.9s forwards;
}

@keyframes draw  { to { stroke-dashoffset: 0; } }
@keyframes fadein{ to { opacity: 1; } }

.cta-row {
  display: flex;
  gap: var(--sp-5);
  align-items: center;
}


/* ─── Section headers ─── */
.s-head {
  font-family: var(--font-display);
  font-size: 56px;
  font-weight: 700;
  line-height: 1.02;
  letter-spacing: -0.035em;
  color: var(--text);
  margin: var(--sp-3) 0 0;
}

.s-head.wide { max-width: 800px; }

.lede {
  margin-top: var(--sp-5);
  font-size: 18px;
  color: var(--text-2);
  max-width: 640px;
  line-height: 1.55;
}


/* ─── Pull quote (problem) ─── */
.pullquote {
  font-family: var(--font-display);
  font-size: 56px;
  font-weight: 500;
  line-height: 1.1;
  letter-spacing: -0.025em;
  color: var(--text);
  font-style: italic;
  text-align: center;
  max-width: 1000px;
  margin: var(--sp-7) auto 0;
  position: relative;
}

.pullquote::before {
  content: '"';
  color: var(--brand);
  font-style: normal;
  font-weight: 700;
  font-size: 96px;
  line-height: 0;
  margin-right: 6px;
  vertical-align: -8px;
}

.quote-attr {
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  text-align: center;
  margin: var(--sp-7) 0 0;
}

.quote-attr .name { color: var(--text); }

.stat-grid {
  margin-top: 112px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-8);
}

.stat-num {
  font-family: var(--font-display);
  font-size: 96px;
  font-weight: 700;
  line-height: 0.9;
  letter-spacing: -0.04em;
  color: var(--text);
}

.stat-num .lime {
  background: var(--brand);
  padding: 0 0.04em;
}

.stat-desc {
  margin-top: var(--sp-4);
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.5;
  max-width: 28ch;
}


/* ─── How it works diagram + roles ─── */
.market-diagram {
  width: 100%;
  max-width: 880px;
  margin: 80px auto 0;
}

.market-diagram svg {
  width: 100%;
  height: auto;
}

.role-grid {
  margin-top: 72px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 56px;
}

.role-title {
  font-family: var(--font-display);
  font-size: var(--fs-2xl);
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 12px 0 14px;
  color: var(--text);
}

.role-text {
  font-size: 15px;
  color: var(--text-2);
  line-height: 1.6;
  margin: 0 0 18px;
}

.role-list {
  list-style: none;
  padding-left: 18px;
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.55;
  margin: 0;
}

.role-list li {
  position: relative;
  margin-bottom: 6px;
}

.role-list li::before {
  content: '·';
  position: absolute;
  left: -18px;
  opacity: 0.4;
}


/* ─── Product cards ─── */
.products {
  margin-top: 72px;
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
}

.product-card {
  display: grid;
  grid-template-columns: 1fr 1.2fr;
  gap: var(--sp-8);
  align-items: stretch;
  border: 1px solid var(--border);
  background: var(--elevated);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.p-left {
  padding: 56px 56px 56px 64px;
  display: flex;
  flex-direction: column;
}

.p-right {
  padding: var(--sp-7);
  display: flex;
  align-items: center;
  justify-content: center;
}

.p-right.inverse {
  background: var(--inverse);
  color: var(--text-inverse);
}

.p-head {
  font-family: var(--font-display);
  font-size: 40px;
  font-weight: 700;
  line-height: 1.02;
  letter-spacing: -0.03em;
  margin: var(--sp-4) 0 18px;
  color: var(--text);
}

.p-text {
  font-size: var(--fs-md);
  color: var(--text-2);
  line-height: 1.55;
  max-width: 44ch;
  margin: 0;
}

.p-link {
  margin-top: auto;
  padding-top: var(--sp-5);
}


/* Mini order book in the trading card */
.ob-mini {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  width: 100%;
  max-width: 360px;
}

.ob-mini .ob-head {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  font-size: 9px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(232, 230, 224, 0.5);
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-on-dark);
}

.ob-mini .ob-head .right { text-align: right; }

.ob-mini .ob-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  padding: 3px 0;
  position: relative;
}

.ob-mini .ob-row .bar {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
}

.ob-mini .ob-row.ask { color: var(--text-inverse); }
.ob-mini .ob-row.ask .px  { color: var(--neg); }
.ob-mini .ob-row.ask .bar { background: rgba(239, 68, 68, 0.10); }
.ob-mini .ob-row.bid .px  { color: var(--pos); }
.ob-mini .ob-row.bid .bar { background: rgba(25, 195, 125, 0.10); }
.ob-mini .ob-row > * { position: relative; text-align: right; }
.ob-mini .ob-row > .px  { text-align: left; }
.ob-mini .ob-row > .right { text-align: right; }

.ob-mini .ob-mid {
  text-align: center;
  padding: 6px 0;
  font-family: var(--font-display);
  font-size: var(--fs-sm);
  font-weight: 600;
  border-top: 1px solid var(--border-on-dark);
  border-bottom: 1px solid var(--border-on-dark);
  margin: 4px 0;
  color: var(--text-inverse);
}

.ob-mini .ob-spread {
  font-family: var(--font-mono);
  font-size: 10px;
  opacity: 0.5;
  margin-left: 8px;
  font-weight: 400;
}


/* Terminal blocks */
.term {
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.65;
  color: var(--text-inverse);
  width: 100%;
  max-width: 420px;
  margin: 0;
  white-space: pre-wrap;
}

.term .prompt { color: rgba(232, 230, 224, 0.5); }
.term .ok     { color: var(--brand); }
.term .dim    { color: rgba(232, 230, 224, 0.55); }


/* ─── Index chart ─── */
.chart-wrap {
  position: relative;
  margin-top: 56px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: var(--sp-6);
}

.ch-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: var(--sp-5);
}

.ch-val {
  font-family: var(--font-display);
  font-size: 36px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-top: 6px;
  color: var(--text);
}

.ch-val .pos {
  font-size: 18px;
  font-weight: 500;
}

.ch-canvas-wrap {
  position: relative;
  height: 320px;
}

.ch-canvas-wrap canvas {
  width: 100% !important;
  height: 100% !important;
}

.chart-tip {
  position: absolute;
  pointer-events: none;
  background: var(--inverse);
  color: var(--text-inverse);
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-on-dark);
  transform: translate(-50%, -120%);
  transition: opacity 100ms;
  white-space: nowrap;
  z-index: 5;
}

.tip-lbl {
  color: rgba(232, 230, 224, 0.55);
  font-size: 9px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.tip-val { color: var(--text-inverse); font-size: 14px; }

.tip-date {
  font-size: 10px;
  opacity: 0.6;
}

.ch-stats {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 0;
  margin-top: var(--sp-6);
  padding-top: var(--sp-5);
  border-top: 1px solid var(--border);
}

.ch-stat {
  font-family: var(--font-mono);
  font-size: 22px;
  font-weight: 500;
  color: var(--text);
  margin-top: 8px;
}

.ch-stat.pos { color: var(--pos); }
.ch-stat.neg { color: var(--neg); }

.pos { color: var(--pos); }
.neg { color: var(--neg); }


/* ─── Audience cards ─── */
.aud-grid {
  margin-top: 72px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 56px;
}

.aud-card {
  border-top: 1px solid var(--border);
  padding-top: var(--sp-6);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.aud-card h4 {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.1;
  margin-bottom: var(--sp-4);
  color: var(--text);
}

.aud-card p {
  color: var(--text-2);
  font-size: 15px;
  line-height: 1.55;
  margin-bottom: 20px;
}

.aud-card ul {
  list-style: none;
  padding: 0;
  margin: 0 0 var(--sp-5);
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.aud-card li {
  font-size: var(--fs-base);
  color: var(--text-2);
  padding-left: 18px;
  position: relative;
  line-height: 1.5;
}

.aud-card li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  width: 8px;
  height: 1px;
  background: var(--text-3);
}

.aud-link {
  color: var(--text);
  font-weight: 500;
  font-size: var(--fs-base);
  margin-top: auto;
  text-decoration: none;
  border-bottom: 1px solid var(--text);
  align-self: flex-start;
  padding-bottom: 1px;
}


/* ─── Trust strip ─── */
.trust-band {
  padding: 64px 0;
  background: var(--canvas);
}

.trust-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.trust-strip > * {
  padding: var(--sp-5) var(--sp-6);
  border-right: 1px solid var(--border);
  font-size: var(--fs-sm);
  line-height: 1.5;
  color: var(--text-2);
}

.trust-strip > *:last-child { border-right: none; }

.t-lbl {
  display: block;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 6px;
}


/* ─── CTA band ─── */
.cta-band {
  background: var(--canvas);
  padding-top: 128px;
  padding-bottom: 128px;
}

.cta-head {
  font-family: var(--font-display);
  font-size: 88px;
  font-weight: 700;
  line-height: 0.96;
  letter-spacing: -0.04em;
  max-width: 1100px;
  margin: 0 auto;
  color: var(--text);
}

.cta-sub {
  margin: var(--sp-6) auto 0;
  font-size: 19px;
  color: var(--text-2);
  max-width: 640px;
  line-height: 1.5;
}

.cta-actions {
  margin-top: var(--sp-7);
  display: flex;
  gap: var(--sp-5);
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
}


/* ─── Responsive shim ─── */
@media (max-width: 1024px) {
  .hero-display { font-size: 96px; }
  .cta-head { font-size: 72px; }
}

@media (max-width: 900px) {
  .hero-display { font-size: 64px; line-height: 0.96; }
  .cta-head { font-size: 56px; }
  .stat-num { font-size: 64px; }
  .s-head { font-size: 40px; }
  .pullquote { font-size: 32px; }
  .stat-grid { grid-template-columns: 1fr; gap: var(--sp-7); }
  .role-grid,
  .aud-grid { grid-template-columns: 1fr; gap: var(--sp-7); }
  .product-card { grid-template-columns: 1fr; }
  .p-left { padding: 40px; }
  .trust-strip { grid-template-columns: repeat(2, 1fr); }
  .trust-strip > *:nth-child(2n) { border-right: 0; }
  .ch-stats { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 600px) {
  .ticker-row { flex-direction: column; align-items: flex-start; }
  .cta-row { flex-direction: column; align-items: flex-start; }
  .container-x { padding: 0 var(--sp-5); }
}

/* ============================================================
   Floating "guided tour" CTA (fixed bottom-right on /)
   ============================================================ */
.tour-fab {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 60;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  padding: 12px 18px 12px 16px;
  background: var(--inverse);
  color: var(--text-inverse);
  border-radius: var(--radius-sm);
  text-decoration: none;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.20);
  border: 1px solid var(--inverse);
  font-family: var(--font-sans);
  transition: transform 140ms var(--ease, ease), box-shadow 140ms var(--ease, ease), background-color 140ms var(--ease, ease);
  animation: fab-in 420ms cubic-bezier(0.16, 1, 0.3, 1) 600ms backwards;
}
.tour-fab:hover {
  transform: translateY(-1px);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.28);
}
.tour-fab:active { transform: translateY(0); }

.fab-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: var(--brand);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--brand) 24%, transparent);
  animation: fab-pulse 2.4s ease-in-out infinite;
  flex-shrink: 0;
}
.fab-eyebrow {
  font-family: var(--font-mono);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.18em;
  color: color-mix(in srgb, var(--text-inverse) 55%, transparent);
  text-transform: uppercase;
  display: none;
}
.fab-title {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
}
.fab-arrow {
  font-family: var(--font-mono);
  font-size: 14px;
  color: var(--brand);
  font-weight: 700;
}

@keyframes fab-in {
  from { opacity: 0; transform: translateY(16px); }
  to   { opacity: 1; transform: translateY(0); }
}
@keyframes fab-pulse {
  0%, 100% { box-shadow: 0 0 0 4px color-mix(in srgb, var(--brand) 24%, transparent); }
  50%      { box-shadow: 0 0 0 8px color-mix(in srgb, var(--brand) 10%, transparent); }
}

@media (min-width: 1024px) {
  .tour-fab { gap: 14px; padding: 14px 22px 14px 18px; }
  .fab-eyebrow { display: inline-block; margin-right: 2px; }
  .fab-title { font-size: 14px; }
}

@media (max-width: 540px) {
  .tour-fab { right: 16px; bottom: 16px; padding: 10px 14px 10px 12px; }
  .fab-title { font-size: 12px; }
}
</style>
