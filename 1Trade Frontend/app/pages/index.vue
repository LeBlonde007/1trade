<script setup lang="ts">
/**
 * / — 1TRADE landing page.
 *
 * Order: thesis -> who it's for -> what you can buy -> how it works.
 *
 * The exchange thesis opens the page. Because that opening necessarily ends on a caveat
 * ("we are not taking trading accounts before then"), two things immediately follow it so
 * a visitor is not left thinking nothing here works yet:
 *   1. a one-line definition under the headline saying plainly what is sold, so the
 *      "new oil" metaphor lands on a fact rather than instead of one;
 *   2. the audience section, where two of the three sides read Live.
 * The hero ("Buy compute once") then acts as proof of the definition rather than
 * competing with it for the opening slot.
 *
 * The trading layer is paused pending a licence (CLAUDE.md § current direction), so every
 * exchange surface here stays explicitly staged — the "In design" pill, paper-mode wording
 * and the Illustrative label on the order book. Do not quietly drop those.
 *
 * An earlier version made the three-sided-market argument twice ("Three sides. One venue."
 * and "Three audiences. One product.") plus a third section reusing the same headline
 * construction; that is now the single audience section.
 *
 * Dark by intent — gold is 1.98:1 on cream and cannot carry the brand on a light ground.
 * The layout honours `theme` from page meta, so /benchmark and /status stay light.
 *
 * Live behaviour preserved from the original: index ticker (Brownian motion + mean
 * reversion, 5s), 30-day sparkline, 365-day chart.js index chart with crosshair tooltip,
 * next-print countdown, scroll reveal.
 */
definePageMeta({ layout: 'marketing', theme: 'dark' })
useHead({ title: '1TRADE — Buy compute once. Spend it anywhere.' })

// ─── Live index ticker ──────────────────────────────────────
const tkNum = ref('1,002.4')
const tkDelta = ref('▲ 0.18% (24h)')
const tkDeltaNeg = ref(false)
const tkFlash = ref<'' | 'flash-up' | 'flash-down'>('')
const tkNext = ref('3h 24m')

const obMidLabel = ref('1,002.4')
const footerIdx = ref('1,002.4')

let indexValue = 1.0024
const target = 1.00
let nextPrintSec = 3 * 3600 + 24 * 60
let tickInterval: ReturnType<typeof setInterval> | null = null
let printInterval: ReturnType<typeof setInterval> | null = null
let chartCleanup: (() => void) | null = null

const fmt = (n: number) =>
  n.toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

const doTick = () => {
  const drift = (target - indexValue) * 0.05
  const noise = (Math.random() - 0.5) * 0.0008
  const change = drift + noise
  indexValue = Math.max(0.985, Math.min(1.018, indexValue + change))
  const dayChange = (indexValue - 1.0006) / 1.0006

  const text = fmt(indexValue * 1000)
  tkNum.value = text
  obMidLabel.value = text
  footerIdx.value = text

  const dpct = (dayChange * 100).toFixed(2)
  tkDelta.value = `${dayChange >= 0 ? '▲' : '▼'} ${Math.abs(parseFloat(dpct))}% (24h)`
  tkDeltaNeg.value = dayChange < 0

  tkFlash.value = change > 0 ? 'flash-up' : 'flash-down'
  setTimeout(() => { tkFlash.value = '' }, 700)
}

// ─── Sparkline ──────────────────────────────────────────────
const sparkLineD = ref('')
const sparkFillD = ref('')

const buildSparkline = () => {
  const W = 240, H = 64, days = 30
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

// ─── 365-day index chart (client only) ──────────────────────
const indexCanvas = ref<HTMLCanvasElement>()
const chartTipVal = ref('1.0024')
const chartTipDate = ref('19 May 2026')
const chartTipPos = ref({ x: 0, y: 0, opacity: 0 })

const buildIndexChart = async () => {
  const { Chart, registerables } = await import('chart.js')
  await import('chartjs-adapter-date-fns')
  Chart.register(...registerables)
  if (!indexCanvas.value) return

  const root = getComputedStyle(document.documentElement)
  const brandHex = root.getPropertyValue('--brand').trim() || '#D4AF37'
  const t3Hex = root.getPropertyValue('--text-3').trim() || '#7E786C'
  // The original hardcoded a black 4%-alpha grid, which is invisible on midnight.
  // Derive it from the theme's own hairline instead.
  const gridCol = root.getPropertyValue('--border').trim() || 'rgba(216,197,163,0.12)'

  const days = 365
  let v = 0.962
  const data: number[] = []
  const labels: Date[] = []
  const now = new Date('2026-05-19')
  for (let i = 0; i < days; i++) {
    const drawdown = Math.random() < 0.018 ? -(0.005 + Math.random() * 0.01) : 0
    const noise = (Math.random() - 0.48) * 0.0035
    v = Math.max(0.94, Math.min(1.05, v + 0.0002 + noise + drawdown))
    data.push(+v.toFixed(4))
    const d = new Date(now)
    d.setDate(d.getDate() - (days - 1 - i))
    labels.push(d)
  }

  const ctx = indexCanvas.value
  const g = ctx.getContext('2d')
  if (!g) return
  const grad = g.createLinearGradient(0, 0, 0, ctx.clientHeight || 320)
  grad.addColorStop(0, `${brandHex}4D`)
  grad.addColorStop(1, `${brandHex}00`)

  const chart = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        data,
        borderColor: brandHex,
        borderWidth: 1.5,
        backgroundColor: grad,
        fill: true,
        pointRadius: 0,
        pointHoverRadius: 3,
        pointHoverBackgroundColor: brandHex,
        tension: 0.15,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: { legend: { display: false }, tooltip: { enabled: false } },
      scales: {
        x: {
          type: 'time',
          time: { unit: 'month' },
          border: { display: false },
          grid: { display: false },
          ticks: { color: t3Hex, font: { family: 'JetBrains Mono', size: 10 }, maxTicksLimit: 8, maxRotation: 0 },
        },
        y: {
          position: 'right',
          border: { display: false },
          grid: { color: gridCol },
          ticks: { color: t3Hex, font: { family: 'JetBrains Mono', size: 10 }, callback: (val) => Number(val).toFixed(3) },
        },
      },
    },
  })

  const onMove = (e: MouseEvent) => {
    const pts = chart.getElementsAtEventForMode(e, 'index', { intersect: false }, false)
    if (!pts.length) return
    const i = pts[0]!.index
    chartTipVal.value = (data[i] ?? 0).toFixed(4)
    chartTipDate.value = (labels[i] ?? new Date()).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
    chartTipPos.value = { x: pts[0]!.element.x, y: pts[0]!.element.y, opacity: 1 }
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

// ─── Scroll reveal ──────────────────────────────────────────
// JS-gated so a no-JS / crawler load still shows everything; CSS forces all
// sections visible under prefers-reduced-motion.
const revealReady = ref(false)
let revealObserver: IntersectionObserver | null = null

const setupReveal = () => {
  revealReady.value = true
  nextTick(() => {
    const els = Array.from(document.querySelectorAll<HTMLElement>('.reveal'))
    if (!('IntersectionObserver' in window)) { els.forEach((e) => e.classList.add('in-view')); return }
    revealObserver = new IntersectionObserver((entries) => {
      for (const en of entries) {
        if (!en.isIntersecting) continue
        en.target.classList.add('in-view')
        revealObserver?.unobserve(en.target)
      }
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 })
    els.forEach((e) => revealObserver!.observe(e))
  })
}

onMounted(() => {
  buildSparkline()
  doTick()
  tickInterval = setInterval(doTick, 5000)
  printInterval = setInterval(() => {
    nextPrintSec = Math.max(0, nextPrintSec - 60)
    const h = Math.floor(nextPrintSec / 3600)
    const m = Math.floor((nextPrintSec % 3600) / 60)
    tkNext.value = `${h}h ${String(m).padStart(2, '0')}m`
  }, 60_000)
  buildIndexChart()
  setupReveal()
})

onUnmounted(() => {
  if (tickInterval) clearInterval(tickInterval)
  if (printInterval) clearInterval(printInterval)
  if (chartCleanup) { chartCleanup(); chartCleanup = null }
  revealObserver?.disconnect()
})

// ─── Content ────────────────────────────────────────────────
// Counts are the live catalogue as served by inference-gateway + compute-control.
const MODEL_COUNT = 20
const MODALITY_COUNT = 9

/** What you can actually buy today, with the numbers behind each claim. */
const layers = [
  {
    id: 'models',
    kicker: 'Inference',
    title: 'Models',
    stat: MODEL_COUNT,
    statLabel: `models · ${MODALITY_COUNT} modalities`,
    body: 'Curated open and frontier models behind one OpenAI-compatible endpoint. Metered per thousand tokens, per image, per video — you see the credit cost of a call before you make it.',
    items: ['llama-3.1-70b', 'claude-opus-4.8', 'gpt-5', 'deepseek-v3.2', 'flux-schnell', 'bge-m3'],
    to: '/signup',
    cta: 'Browse the catalogue',
  },
  {
    id: 'compute',
    kicker: 'Compute',
    title: 'GPU instances',
    stat: 8,
    statLabel: 'tiers · L40S → GB300',
    body: 'Rent by the hour or reserve a dated block. Multi-GPU jobs are gang-scheduled, so a job either gets every GPU it asked for or waits — it never half-starts and burns your balance.',
    items: ['L40S 48GB', 'A100 80GB', 'H100 80GB', 'H200 141GB', 'B200 192GB', 'GB300 NVL72'],
    to: '/signup',
    cta: 'See instance types',
  },
  {
    id: 'credits',
    kicker: 'Settlement',
    title: 'Prepaid credits',
    stat: 1,
    statLabel: 'balance · every product',
    body: 'Buy AI credits once. They convert into whichever specific credit a job needs — text, speech, image, video, embeddings, or a GPU-hour on a given tier. No per-product wallets to top up.',
    items: ['ai_index', 'text', 'speech', 'image', 'video', 'gpu_h100'],
    to: '/signup',
    cta: 'How credits work',
  },
]

/** The credit mechanic, which the previous page never explained.
 *  `codes` renders as chips rather than inline markup, so no step needs v-html. */
const flow = [
  {
    step: 'Buy',
    body: 'Card, ACH or wire buys the umbrella credit, priced off the published index rather than a private rate card.',
    codes: ['ai_index'],
  },
  {
    step: 'Convert',
    body: 'Convert it into whatever a job actually consumes — chat and code, media, embeddings, or an hour on a named GPU tier.',
    codes: ['text', 'image', 'video', 'speech', 'embeddings', 'gpu_h100'],
  },
  {
    step: 'Spend',
    body: 'Every call and every GPU-hour debits the matching credit. The gateway meters usage; the ledger settles it about a second later.',
    codes: [],
  },
  {
    step: 'Audit',
    body: 'Each movement appends to a hash-chained ledger, so any balance can be replayed from the chain and proven rather than trusted.',
    codes: ['hash(prev ‖ row)'],
  },
]

/** Who the platform serves. One section, not three. */
const audiences = [
  {
    who: 'AI companies',
    live: true,
    body: 'Ship on 20 models without negotiating GPU contracts. Prepay, budget per team, and see exactly what each feature costs to run.',
    to: '/signup',
    cta: 'Start building',
  },
  {
    who: 'Datacenters',
    live: true,
    body: 'Put idle capacity to work. Attested GPUs, escrowed payouts streamed as your hardware serves real jobs.',
    to: '/datacenter/register',
    cta: 'Register capacity',
  },
  {
    who: 'Traders',
    live: false,
    // Wording is constrained by the F22 framing guide's banned-term list for customer-facing
    // surfaces (see docs/plans/features/F22-licensing-track.md — the terms are listed there, not
    // repeated here so an automated audit of this file stays meaningful). Describe price discovery
    // and hedging only, until counsel signs the framing off.
    body: 'Discover a public price for compute and hedge exposure to it. Order book, market maker and dated contracts are built and running in paper mode; the venue opens once licensed.',
    to: '/benchmark',
    cta: 'Read the methodology',
  },
]

/** Mini order-book preview — illustrative, shown in the exchange section only. */
const obAsks = [
  { px: '1010.4', sz: '128.4', bar: 18 },
  { px: '1009.2', sz: '186.0', bar: 26 },
  { px: '1008.0', sz: '298.7', bar: 42 },
  { px: '1006.5', sz: '421.3', bar: 60 },
]
const obBids = [
  { px: '1000.4', sz: '498.2', bar: 70 },
  { px: '999.8', sz: '412.8', bar: 58 },
  { px: '998.1', sz: '285.5', bar: 40 },
  { px: '996.7', sz: '170.1', bar: 24 },
]

// ─── API snippet + copy ─────────────────────────────────────
// Env-var placeholders rather than a literal host: the production domain is still
// being settled, and a wrong hostname in a quickstart is worse than none.
const SNIPPET = `curl "$ONETRADE_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer $ONETRADE_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "llama-3.1-8b",
    "messages": [{"role": "user", "content": "Say hi in three words"}]
  }'`

const RESPONSE = `{
  "model": "llama-3.1-8b",
  "choices": [{ "message": { "content": "Hi there friend" } }],
  "usage": { "prompt_tokens": 14, "completion_tokens": 4, "total_tokens": 18 },
  "x_1trade": { "credit_type": "text", "credits_debited": "0.090000" }
}`

const copied = ref(false)
/** Copy the quickstart to the clipboard, with a short confirmation. */
const copySnippet = async () => {
  try {
    await navigator.clipboard.writeText(SNIPPET)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1800)
  } catch { /* clipboard blocked — the snippet is selectable anyway */ }
}
</script>

<template>
  <article class="hm" :class="{ 'reveal-on': revealReady }">

    <!-- ══════════════ EXCHANGE — the thesis. Leads the page: the client sees this first. ══════════════ -->
    <section id="exchange" class="hm-band hm-exch hm-exch-lead">
      <div class="hm-wrap hm-exch-grid">
        <div>
          <p class="hm-kicker">Where this goes</p>
          <h2 class="hm-h2 hm-h2-lg">Compute is the new oil.<br><span class="hm-gold">1TRADE is the exchange.</span></h2>
          <!-- Says the category in one line. Without it the opening is metaphor, then a
               caveat that the venue is not open — a visitor reaches the fold without ever
               being told plainly what is sold. This makes the metaphor land on a fact. -->
          <p class="hm-def">
            1TRADE sells AI inference and GPU time for prepaid credits — and is building the
            exchange those credits will trade on.
          </p>
          <p class="hm-sub">
            Every commodity that mattered eventually got a market: a public price, someone willing
            to quote both sides, and instruments to hedge with. Compute has none of that yet — it
            has bilateral contracts and waiting lists.
          </p>
          <p class="hm-sub">
            The platform below is the foundation: real capacity, a real unit of account, a real
            audit trail. The venue is designed and built against a mock matching engine today.
          </p>
          <p class="hm-status">
            <span class="hm-status-pill">In design</span>
            Order book, market maker and dated contracts are specified and running in paper mode.
            The venue opens when the licence does — we are not taking trading accounts before then.
          </p>
        </div>

        <!-- Illustrative depth preview. Labelled, so it is not mistaken for a live book. -->
        <aside class="hm-ob" aria-label="Illustrative order book">
          <div class="hm-ob-head"><span>Order book</span><span class="hm-ob-tag">Illustrative</span></div>
          <div class="hm-ob-rows">
            <div v-for="a in obAsks" :key="'a' + a.px" class="hm-ob-r">
              <span class="hm-ob-bar hm-ob-bar-a" :style="{ width: a.bar + '%' }" />
              <span class="hm-ob-px hm-neg">{{ a.px }}</span><span class="hm-ob-sz">{{ a.sz }}</span>
            </div>
            <div class="hm-ob-mid"><span>{{ obMidLabel }}</span><span class="hm-ob-mid-l">mid</span></div>
            <div v-for="b in obBids" :key="'b' + b.px" class="hm-ob-r">
              <span class="hm-ob-bar hm-ob-bar-b" :style="{ width: b.bar + '%' }" />
              <span class="hm-ob-px hm-pos">{{ b.px }}</span><span class="hm-ob-sz">{{ b.sz }}</span>
            </div>
          </div>
        </aside>
      </div>
    </section>

    <!-- ══════════════ AUDIENCES — placed second: two of the three sides are Live, which answers the
         "is any of this usable today?" question the opening caveat raises ══════════════ -->
    <section class="hm-band hm-band-alt">
      <div class="hm-wrap">
        <header class="hm-shead reveal">
          <p class="hm-kicker">Who it's for</p>
          <h2 class="hm-h2">Three sides of one market.</h2>
        </header>
        <div class="hm-aud">
          <article v-for="a in audiences" :key="a.who" class="hm-aud-i reveal">
            <div class="hm-aud-top">
              <h3 class="hm-aud-t">{{ a.who }}</h3>
              <span class="hm-badge" :class="a.live ? 'is-live' : 'is-soon'">{{ a.live ? 'Live' : 'Paused' }}</span>
            </div>
            <p class="hm-aud-b">{{ a.body }}</p>
            <NuxtLink :to="a.to" class="hm-link-cta hm-link-sm">{{ a.cta }} →</NuxtLink>
          </article>
        </div>
      </div>
    </section>

    <!-- ══════════════ HERO — what you can buy today ══════════════ -->
    <section class="hm-hero">
      <div class="hm-wrap hm-hero-grid">
        <div class="hm-hero-copy">
          <p class="hm-eyebrow"><span class="hm-live" />Inference · GPUs · prepaid credits — live now</p>
          <h1 class="hm-h1">Buy compute once.<br>Spend it <span class="hm-gold">anywhere</span>.</h1>
          <p class="hm-lede">
            {{ MODEL_COUNT }} models across {{ MODALITY_COUNT }} modalities, GPU instances from L40S
            to GB300, and an OpenAI-compatible API — all drawn from a single prepaid balance,
            metered to six decimal places.
          </p>
          <div class="hm-cta-row">
            <BaseButton as="a" :href="'/signup'" variant="primary" size="lg">Start building</BaseButton>
            <a href="#api" class="hm-link-cta">Read the API →</a>
          </div>
          <p class="hm-hero-note">No minimum. Credits never expire. Paper mode by default.</p>
        </div>

        <!-- The characteristic artefact of this product: a metered API call. -->
        <div class="hm-term" aria-label="Example API request and response">
          <div class="hm-term-bar">
            <span class="hm-term-dot" /><span class="hm-term-t">chat/completions</span>
            <button class="hm-copy" type="button" @click="copySnippet">
              {{ copied ? 'Copied' : 'Copy' }}
            </button>
          </div>
          <pre class="hm-term-body"><code>{{ SNIPPET }}</code></pre>
          <div class="hm-term-split">response</div>
          <pre class="hm-term-body hm-term-res"><code>{{ RESPONSE }}</code></pre>
        </div>
      </div>
    </section>

    <!-- ══════════════ PLATFORM — the three things you buy ══════════════ -->
    <section id="platform" class="hm-band">
      <div class="hm-wrap">
        <header class="hm-shead reveal">
          <p class="hm-kicker">The platform</p>
          <h2 class="hm-h2">Three products. One balance.</h2>
        </header>

        <div class="hm-layers">
          <article v-for="l in layers" :key="l.id" class="hm-layer reveal">
            <p class="hm-kicker hm-kicker-sm">{{ l.kicker }}</p>
            <h3 class="hm-layer-t">{{ l.title }}</h3>
            <p class="hm-stat"><span class="hm-stat-n">{{ l.stat }}</span><span class="hm-stat-l">{{ l.statLabel }}</span></p>
            <p class="hm-layer-b">{{ l.body }}</p>
            <ul class="hm-chips">
              <li v-for="i in l.items" :key="i">{{ i }}</li>
            </ul>
            <NuxtLink :to="l.to" class="hm-link-cta hm-link-sm">{{ l.cta }} →</NuxtLink>
          </article>
        </div>
      </div>
    </section>

    <!-- ══════════════ CREDITS — the mechanic ══════════════ -->
    <section id="credits" class="hm-band hm-band-alt">
      <div class="hm-wrap">
        <header class="hm-shead reveal">
          <p class="hm-kicker">Credits</p>
          <h2 class="hm-h2">Prepay in one unit.<br>Spend it as any other.</h2>
          <p class="hm-sub">
            An <code>ai_index</code> credit is the umbrella unit. It converts into whatever a job
            actually consumes, so you fund the account rather than each product.
          </p>
        </header>

        <ol class="hm-flow">
          <li v-for="(f, i) in flow" :key="f.step" class="hm-flow-i reveal">
            <span class="hm-flow-n">{{ String(i + 1).padStart(2, '0') }}</span>
            <h3 class="hm-flow-t">{{ f.step }}</h3>
            <p class="hm-flow-b">{{ f.body }}</p>
            <ul v-if="f.codes.length" class="hm-chips hm-chips-tight">
              <li v-for="c in f.codes" :key="c">{{ c }}</li>
            </ul>
          </li>
        </ol>
      </div>
    </section>

    <!-- ══════════════ API ══════════════ -->
    <section id="api" class="hm-band">
      <div class="hm-wrap hm-api-grid">
        <div class="reveal">
          <p class="hm-kicker">The API</p>
          <h2 class="hm-h2">If you can call OpenAI,<br>you can call this.</h2>
          <p class="hm-sub">
            Same request shape, same response shape. Change the base URL and the key, keep your
            client library. Every response carries what it cost you, so metering is not a
            month-end surprise.
          </p>
          <ul class="hm-ticks">
            <li>Drop-in <code>/v1/chat/completions</code>, streaming supported</li>
            <li>Per-call credit cost returned inline on every response</li>
            <li>Scoped API keys, revocable per environment</li>
            <li>Paper mode by default — real and paper balances never mix</li>
          </ul>
        </div>
        <div class="hm-api-side reveal">
          <div class="hm-kv">
            <div class="hm-kv-i"><span class="hm-kv-k">Endpoint</span><span class="hm-kv-v">/v1/chat/completions</span></div>
            <div class="hm-kv-i"><span class="hm-kv-k">Auth</span><span class="hm-kv-v">Bearer key</span></div>
            <div class="hm-kv-i"><span class="hm-kv-k">Metering</span><span class="hm-kv-v">NUMERIC(20,6)</span></div>
            <div class="hm-kv-i"><span class="hm-kv-k">Settlement</span><span class="hm-kv-v">async, ~1s</span></div>
          </div>
          <p class="hm-note">
            Usage is published on <code>inference.usage.v1</code> and settled by the ledger a
            moment after the call returns — balances move on their own, without a reconciliation job.
          </p>
        </div>
      </div>
    </section>

    <!-- ══════════════ INDEX ══════════════ -->
    <section class="hm-band hm-band-alt">
      <div class="hm-wrap">
        <header class="hm-shead reveal">
          <p class="hm-kicker">The index</p>
          <h2 class="hm-h2">One published price<br>for AI compute.</h2>
          <p class="hm-sub">
            Credits are priced off a daily index rather than a private rate card. The methodology
            is public and every print is hash-chained, so a historical value can be re-derived
            instead of taken on trust.
          </p>
        </header>

        <div class="hm-idx reveal">
          <div class="hm-idx-head">
            <div class="hm-ticker">
              <span class="hm-live" />
              <div>
                <div class="hm-t-label">1TRADE AI Index</div>
                <div class="hm-t-line">
                  <span class="hm-t-val" :class="tkFlash">$1 = <span>{{ tkNum }}</span> credits</span>
                  <span class="hm-t-delta" :class="{ neg: tkDeltaNeg }">{{ tkDelta }}</span>
                </div>
              </div>
            </div>
            <div class="hm-idx-meta">
              <svg class="hm-spark" viewBox="0 0 240 64" preserveAspectRatio="none" aria-hidden="true">
                <path class="hm-spark-f" :d="sparkFillD" />
                <path class="hm-spark-l" :d="sparkLineD" />
              </svg>
              <span class="hm-note-sm">Next print in {{ tkNext }} · 16:00 UTC</span>
            </div>
          </div>

          <div class="hm-chart-wrap">
            <canvas ref="indexCanvas" class="hm-chart" aria-label="1TRADE AI Index, trailing 365 days" />
            <div
              class="hm-tip"
              :style="{ left: chartTipPos.x + 'px', top: chartTipPos.y + 'px', opacity: chartTipPos.opacity }"
            >
              <strong>{{ chartTipVal }}</strong><span>{{ chartTipDate }}</span>
            </div>
          </div>

          <p class="hm-disclaimer">
            <strong>Illustrative.</strong> The index service is not yet in production — the series
            above is simulated for demonstration. Live prints begin with the index launch; the
            methodology is published at
            <NuxtLink to="/benchmark">/benchmark</NuxtLink>.
          </p>
        </div>
      </div>
    </section>

    <!-- ══════════════ TRUST ══════════════ -->
    <section class="hm-trust">
      <div class="hm-wrap hm-trust-grid">
        <div class="hm-trust-i"><span class="hm-trust-k">Ledger</span><span class="hm-trust-v">Append-only, hash-chained</span></div>
        <div class="hm-trust-i"><span class="hm-trust-k">Money math</span><span class="hm-trust-v">Fixed-point, 6 dp</span></div>
        <div class="hm-trust-i"><span class="hm-trust-k">Isolation</span><span class="hm-trust-v">Paper never touches real</span></div>
        <div class="hm-trust-i"><span class="hm-trust-k">Index</span><span class="hm-trust-v">Public methodology</span></div>
        <div class="hm-trust-i"><span class="hm-trust-k">Status</span><span class="hm-trust-v"><NuxtLink to="/status">Live uptime →</NuxtLink></span></div>
      </div>
    </section>

    <!-- ══════════════ CTA ══════════════ -->
    <section class="hm-cta">
      <div class="hm-wrap hm-cta-in reveal">
        <h2 class="hm-h2 hm-h2-lg">Start with a test key<br>and $0 committed.</h2>
        <p class="hm-sub hm-sub-c">
          Sign up, mint paper credits, and make a metered call in under five minutes. Move to real
          money when you're ready — the API doesn't change.
        </p>
        <div class="hm-cta-row hm-cta-row-c">
          <BaseButton as="a" :href="'/signup'" variant="primary" size="lg">Open an account</BaseButton>
          <a href="#platform" class="hm-link-cta">See what's included →</a>
        </div>
        <p class="hm-note-sm hm-note-c">Index now: $1 = {{ footerIdx }} credits</p>
      </div>
    </section>
  </article>
</template>

<style scoped>
/* All landing styles are `hm-` prefixed so they cannot collide with the global
   marketing classes the previous version shared with other pages. */
.hm {
  --hm-gutter: clamp(20px, 5vw, 64px);
  --hm-max: 1240px;
  background: var(--canvas);
  color: var(--text);
  overflow-x: clip;
}

.hm-wrap {
  max-width: var(--hm-max);
  margin: 0 auto;
  padding: 0 var(--hm-gutter);
}

/* ── shared type ── */
.hm-eyebrow,
.hm-kicker {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
  margin: 0 0 var(--sp-4);
}
.hm-kicker { color: var(--brand); }
.hm-kicker-sm { margin-bottom: var(--sp-2); }

.hm-h1 {
  font-family: var(--font-display);
  font-size: clamp(40px, 6.4vw, 74px);
  font-weight: 700;
  line-height: var(--lh-tight);
  letter-spacing: var(--ls-tight);
  margin: 0 0 var(--sp-5);
  text-wrap: balance;
}
.hm-h2 {
  font-family: var(--font-display);
  font-size: clamp(27px, 3.4vw, 40px);
  font-weight: 600;
  line-height: var(--lh-snug);
  letter-spacing: var(--ls-snug);
  margin: 0 0 var(--sp-4);
  text-wrap: balance;
}
.hm-h2-lg { font-size: clamp(31px, 4.4vw, 52px); }
.hm-gold { color: var(--brand); }

.hm-lede {
  font-size: var(--fs-lg);
  line-height: var(--lh-relax);
  color: var(--text-2);
  max-width: 56ch;
  margin: 0 0 var(--sp-6);
}
.hm-sub {
  font-size: var(--fs-md);
  line-height: var(--lh-relax);
  color: var(--text-2);
  max-width: 62ch;
  margin: 0 0 var(--sp-4);
}
.hm-sub-c { margin-left: auto; margin-right: auto; }

.hm-live {
  width: 7px; height: 7px; border-radius: 50%;
  background: var(--pos); flex: none;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--pos) 22%, transparent);
  animation: hm-pulse 2.4s ease-in-out infinite;
}
@keyframes hm-pulse { 50% { opacity: 0.45; } }

/* ── hero ── */
.hm-hero {
  padding: clamp(56px, 9vh, 108px) 0 clamp(48px, 8vh, 96px);
  border-bottom: 1px solid var(--border);
  /* The ambient wash belongs to whichever section opens the page. That is now the
     exchange lead, so the hero carries none — two stacked glows read as noise. */
}
.hm-hero-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
  gap: clamp(32px, 5vw, 72px);
  align-items: center;
}
.hm-hero-note {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  margin: var(--sp-5) 0 0;
}
.hm-cta-row { display: flex; align-items: center; gap: var(--sp-5); flex-wrap: wrap; }
.hm-cta-row-c { justify-content: center; }

.hm-link-cta {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--brand);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color var(--dur) var(--ease);
}
.hm-link-cta:hover { border-bottom-color: var(--brand); }
.hm-link-sm { font-size: var(--fs-sm); }

/* ── hero terminal ── */
.hm-term {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-2);
  overflow: hidden;
  min-width: 0;
}
.hm-term-bar {
  display: flex; align-items: center; gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--border);
  background: var(--overlay);
}
.hm-term-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--brand); flex: none; }
.hm-term-t {
  font-family: var(--font-mono); font-size: var(--fs-xs);
  letter-spacing: var(--ls-tab); color: var(--text-3); flex: 1;
}
.hm-copy {
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-tab); text-transform: uppercase;
  color: var(--text-2); background: transparent;
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  padding: 3px 8px; cursor: pointer;
  transition: color var(--dur) var(--ease), border-color var(--dur) var(--ease);
}
.hm-copy:hover { color: var(--brand); border-color: var(--brand); }
.hm-term-body {
  margin: 0; padding: var(--sp-4);
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.7;
  color: var(--text);
  tab-size: 2;
}
.hm-term-split {
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide); text-transform: uppercase;
  color: var(--text-3);
  padding: var(--sp-2) var(--sp-4);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: var(--overlay);
}
.hm-term-res { color: var(--text-2); }

/* ── bands ── */
.hm-band { padding: clamp(56px, 9vh, 108px) 0; border-bottom: 1px solid var(--border); }
.hm-band-alt { background: var(--elevated); }
.hm-shead { max-width: 62ch; margin-bottom: clamp(32px, 4vw, 56px); }

/* ── layers ── */
.hm-layers { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1px; background: var(--border); border: 1px solid var(--border); }
.hm-layer { background: var(--canvas); padding: clamp(20px, 2.4vw, 32px); display: flex; flex-direction: column; }
.hm-band-alt .hm-layer { background: var(--elevated); }
.hm-layer-t { font-size: var(--fs-xl); font-weight: 600; letter-spacing: var(--ls-near); margin: 0 0 var(--sp-3); }
.hm-stat { display: flex; align-items: baseline; gap: var(--sp-2); margin: 0 0 var(--sp-4); padding-bottom: var(--sp-4); border-bottom: 1px solid var(--border); }
.hm-stat-n { font-family: var(--font-mono); font-size: 34px; font-weight: 600; color: var(--brand); font-variant-numeric: tabular-nums; line-height: 1; }
.hm-stat-l { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); }
.hm-layer-b { font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2); margin: 0 0 var(--sp-4); }
.hm-chips { list-style: none; display: flex; flex-wrap: wrap; gap: 6px; padding: 0; margin: 0 0 var(--sp-5); }
.hm-chips-tight { margin-top: var(--sp-3); margin-bottom: 0; }
.hm-chips li {
  font-family: var(--font-mono); font-size: 10.5px;
  color: var(--text-2); background: var(--overlay);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  padding: 3px 7px;
}
.hm-layer .hm-link-cta { margin-top: auto; align-self: flex-start; }

/* ── credit flow ── */
.hm-flow { list-style: none; padding: 0; margin: 0; display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: clamp(20px, 2.6vw, 36px); }
.hm-flow-i { border-top: 2px solid var(--brand); padding-top: var(--sp-4); }
.hm-flow-n { font-family: var(--font-mono); font-size: var(--fs-tiny); letter-spacing: var(--ls-wide); color: var(--text-3); }
.hm-flow-t { font-size: var(--fs-lg); font-weight: 600; letter-spacing: var(--ls-near); margin: var(--sp-2) 0 var(--sp-2); }
.hm-flow-b { font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2); margin: 0; }
.hm-sub code,
.hm-ticks code,
.hm-note code {
  font-family: var(--font-mono); font-size: 0.9em;
  background: var(--overlay); border: 1px solid var(--border);
  border-radius: 2px; padding: 1px 4px; color: var(--text);
}

/* ── api ── */
.hm-api-grid { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.9fr); gap: clamp(32px, 5vw, 72px); align-items: start; }
.hm-ticks { list-style: none; padding: 0; margin: var(--sp-5) 0 0; display: grid; gap: var(--sp-3); }
.hm-ticks li {
  position: relative; padding-left: 26px;
  font-size: var(--fs-sm); color: var(--text-2); line-height: var(--lh-relax);
}
.hm-ticks li::before {
  content: ""; position: absolute; left: 0; top: 9px;
  width: 12px; height: 1px; background: var(--brand);
}
.hm-kv { display: grid; gap: 1px; background: var(--border); border: 1px solid var(--border); }
.hm-kv-i { background: var(--canvas); padding: var(--sp-4); display: flex; justify-content: space-between; align-items: baseline; gap: var(--sp-3); }
.hm-kv-k { font-family: var(--font-mono); font-size: var(--fs-tiny); letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3); }
.hm-kv-v { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text); text-align: right; }
.hm-note { font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2); margin: var(--sp-4) 0 0; }
.hm-note-sm { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); }
.hm-note-c { display: block; text-align: center; margin-top: var(--sp-5); }

/* ── index ── */
.hm-idx { border: 1px solid var(--border); background: var(--canvas); }
.hm-idx-head {
  display: flex; justify-content: space-between; align-items: center;
  gap: var(--sp-5); flex-wrap: wrap;
  padding: var(--sp-5); border-bottom: 1px solid var(--border);
}
.hm-ticker { display: flex; align-items: center; gap: var(--sp-3); }
.hm-t-label { font-family: var(--font-mono); font-size: var(--fs-tiny); letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3); }
.hm-t-line { display: flex; align-items: baseline; gap: var(--sp-3); margin-top: 3px; }
.hm-t-val { font-family: var(--font-mono); font-size: var(--fs-xl); font-variant-numeric: tabular-nums; transition: color 240ms var(--ease); }
.hm-t-val.flash-up { color: var(--pos); }
.hm-t-val.flash-down { color: var(--neg); }
.hm-t-delta { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--pos); font-variant-numeric: tabular-nums; }
.hm-t-delta.neg { color: var(--neg); }
.hm-idx-meta { display: flex; align-items: center; gap: var(--sp-4); flex-wrap: wrap; }
.hm-spark { width: 180px; height: 44px; display: block; }
.hm-spark-l { fill: none; stroke: var(--brand); stroke-width: 1.5; }
.hm-spark-f { fill: color-mix(in srgb, var(--brand) 16%, transparent); stroke: none; }
.hm-chart-wrap { position: relative; padding: var(--sp-5); height: 320px; }
.hm-chart { width: 100%; height: 100%; }
.hm-tip {
  position: absolute; transform: translate(-50%, -140%);
  pointer-events: none; background: var(--overlay);
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
  padding: 5px 9px; display: grid; gap: 1px;
  transition: opacity 120ms linear;
}
.hm-tip strong { font-family: var(--font-mono); font-size: var(--fs-sm); font-variant-numeric: tabular-nums; }
.hm-tip span { font-family: var(--font-mono); font-size: 10px; color: var(--text-3); }
.hm-disclaimer {
  margin: 0; padding: var(--sp-4) var(--sp-5);
  border-top: 1px solid var(--border);
  font-size: var(--fs-xs); line-height: var(--lh-relax); color: var(--text-3);
}
.hm-disclaimer strong { color: var(--text-2); }
.hm-disclaimer a { color: var(--brand); }

/* ── exchange ── */
.hm-exch {
  background:
    radial-gradient(90% 120% at 100% 50%, color-mix(in srgb, var(--brand) 9%, transparent), transparent 60%),
    var(--canvas);
}

/* The exchange section opens the page, so it takes the opening treatment: room to
   breathe under the sticky nav, the page's single ambient wash, and display-scale
   type. It is the thesis the whole page argues from. */
.hm-exch-lead {
  padding-top: clamp(52px, 9vh, 104px);
  padding-bottom: clamp(56px, 9vh, 104px);
  background:
    radial-gradient(120% 90% at 88% -20%, color-mix(in srgb, var(--brand) 13%, transparent), transparent 62%),
    radial-gradient(90% 120% at 100% 55%, color-mix(in srgb, var(--brand) 8%, transparent), transparent 60%),
    var(--canvas);
}
.hm-exch-lead .hm-h2 {
  font-size: clamp(34px, 5.4vw, 62px);
  letter-spacing: var(--ls-tight);
}
.hm-exch-lead .hm-sub { font-size: var(--fs-lg); }

/* The definition sentence. Set above the supporting prose and marked with a gold rule so
   it reads as the answer to "what is this", not as another paragraph. */
.hm-def {
  font-size: clamp(17px, 1.5vw, 21px);
  line-height: var(--lh-base);
  color: var(--text);
  max-width: 56ch;
  margin: 0 0 var(--sp-5);
  padding-left: var(--sp-4);
  border-left: 2px solid var(--brand);
}
.hm-exch-grid { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(0, 0.85fr); gap: clamp(32px, 5vw, 72px); align-items: center; }
.hm-status {
  display: grid; gap: var(--sp-2);
  font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2);
  border-left: 2px solid var(--brand); padding: 0 0 0 var(--sp-4);
  margin: var(--sp-5) 0 0;
}
.hm-status-pill {
  justify-self: start;
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide); text-transform: uppercase;
  color: var(--brand); border: 1px solid color-mix(in srgb, var(--brand) 42%, transparent);
  border-radius: var(--radius-sm); padding: 2px 7px;
}

/* ── order book preview ── */
.hm-ob { border: 1px solid var(--border); background: var(--elevated); }
.hm-ob-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border);
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3);
}
.hm-ob-tag { color: var(--warn); border: 1px solid color-mix(in srgb, var(--warn) 40%, transparent); border-radius: 2px; padding: 1px 5px; }
.hm-ob-rows { padding: var(--sp-2) 0; }
.hm-ob-r { position: relative; display: flex; justify-content: space-between; padding: 4px var(--sp-4); font-family: var(--font-mono); font-size: var(--fs-xs); font-variant-numeric: tabular-nums; }
.hm-ob-bar { position: absolute; inset: 0 auto 0 0; }
.hm-ob-bar-a { background: var(--neg-bar); }
.hm-ob-bar-b { background: var(--pos-bar); }
.hm-ob-px, .hm-ob-sz { position: relative; }
.hm-ob-sz { color: var(--text-3); }
.hm-pos { color: var(--pos); }
.hm-neg { color: var(--neg); }
.hm-ob-mid {
  display: flex; justify-content: space-between; align-items: baseline;
  padding: var(--sp-2) var(--sp-4); margin: var(--sp-2) 0;
  border-top: 1px solid var(--border); border-bottom: 1px solid var(--border);
  font-family: var(--font-mono); font-size: var(--fs-sm); font-variant-numeric: tabular-nums;
}
.hm-ob-mid-l { font-size: var(--fs-tiny); letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3); }

/* ── audiences ── */
.hm-aud { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: clamp(20px, 2.6vw, 32px); }
.hm-aud-i { border-top: 1px solid var(--border-strong); padding-top: var(--sp-4); display: flex; flex-direction: column; }
.hm-aud-top { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); margin-bottom: var(--sp-3); }
.hm-aud-t { font-size: var(--fs-lg); font-weight: 600; letter-spacing: var(--ls-near); margin: 0; }
.hm-badge {
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-tab); text-transform: uppercase;
  border-radius: var(--radius-sm); padding: 2px 7px; border: 1px solid;
}
.hm-badge.is-live { color: var(--pos); border-color: color-mix(in srgb, var(--pos) 40%, transparent); }
.hm-badge.is-soon { color: var(--text-3); border-color: var(--border-strong); }
.hm-aud-b { font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2); margin: 0 0 var(--sp-4); }
.hm-aud-i .hm-link-cta { margin-top: auto; align-self: flex-start; }

/* ── trust strip ── */
.hm-trust { background: var(--elevated); border-bottom: 1px solid var(--border); padding: var(--sp-6) 0; }
.hm-trust-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: var(--sp-5); }
.hm-trust-i { display: grid; gap: 3px; }
.hm-trust-k { font-family: var(--font-mono); font-size: var(--fs-tiny); letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3); }
.hm-trust-v { font-size: var(--fs-sm); color: var(--text); }
.hm-trust-v a { color: var(--brand); }

/* ── closing cta ── */
.hm-cta { padding: clamp(64px, 11vh, 128px) 0; }
.hm-cta-in { text-align: center; display: flex; flex-direction: column; align-items: center; }

/* ── scroll reveal (opt-in via .reveal-on, so no-JS shows everything) ── */
.hm.reveal-on .reveal { opacity: 0; transform: translateY(14px); }
.hm.reveal-on .reveal.in-view {
  opacity: 1; transform: none;
  transition: opacity 620ms var(--ease), transform 620ms var(--ease);
}
@media (prefers-reduced-motion: reduce) {
  .hm.reveal-on .reveal { opacity: 1 !important; transform: none !important; transition: none !important; }
  .hm-live { animation: none; }
}

/* ── responsive ── */
@media (max-width: 900px) {
  .hm-hero-grid,
  .hm-api-grid,
  .hm-exch-grid { grid-template-columns: minmax(0, 1fr); }
  .hm-hero { padding-top: clamp(44px, 8vh, 72px); }
  .hm-chart-wrap { height: 240px; }
}
@media (max-width: 560px) {
  .hm-idx-head { padding: var(--sp-4); }
  .hm-spark { width: 120px; }
  .hm-term-body { font-size: 11px; }
}
</style>
