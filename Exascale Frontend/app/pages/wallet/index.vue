<script setup lang="ts">
/**
 * /wallet — All credit and cash balances + conversion drawer.
 * Dark theme. Renders inside the `app` layout (sidebar + topbar already provided).
 *
 * The standalone design ships its own crumbs/topbar pill row; we keep the
 * project's shared chrome and port the page body (header + allocation bar +
 * balance cards + recent movements + Convert drawer).
 */

definePageMeta({ layout: 'app' })
useHead({ title: 'Wallet — Exascale' })

// =====================================================
// Asset list — matches the design's ASSETS array
// =====================================================
type ColorKey = 'cash' | 'ai' | 'text' | 'speech' | 'image' | 'video' | 'niche' | 'h100' | 'h200'

interface Asset {
  key: string
  name: string
  sym: string
  qty: number
  usdPrice: number
  color: ColorKey
  change: number
  locked: number
  featured?: boolean
  empty?: boolean
  description?: string
  secondary?: string
  cls?: 'cash'
  spark: number[]
}

const COLOR: Record<ColorKey, string> = {
  cash:   '#9A9A95',
  ai:     '#C8F25C',
  text:   '#4A90E2',
  speech: '#6BA4E8',
  image:  '#8FB8E6',
  video:  '#5B9BE0',
  niche:  '#3A7FCC',
  h100:   '#F5A623',
  h200:   '#E89818',
}

function genSpark(): number[] {
  let v = 1.0
  const out: number[] = []
  const trend = Math.random() - 0.4
  for (let i = 0; i < 24; i++) {
    v += (Math.random() - 0.5 + trend * 0.02) * 0.022
    out.push(v)
  }
  return out
}

const assets = reactive<Asset[]>([
  { key: 'cash',   name: 'Cash · USD',       sym: 'USD',    qty: 3891.42, usdPrice: 1,        color: 'cash',   change: 0,     locked: 0,     secondary: '¥0', cls: 'cash', spark: [] },
  { key: 'ai',     name: 'AI Credits',       sym: 'EAI',    qty: 2425000, usdPrice: 0.001005, color: 'ai',     change: 1.05,  locked: 50000, featured: true, description: 'Unified inference index', spark: genSpark() },
  { key: 'text',   name: 'Text Credits',     sym: 'TEXT',   qty: 1200000, usdPrice: 0.001210, color: 'text',   change: 0.42,  locked: 0,     spark: genSpark() },
  { key: 'speech', name: 'Speech Credits',   sym: 'SPEECH', qty: 480000,  usdPrice: 0.001200, color: 'speech', change: -0.18, locked: 0,     spark: genSpark() },
  { key: 'image',  name: 'Image Credits',    sym: 'IMAGE',  qty: 152000,  usdPrice: 0.008000, color: 'image',  change: 1.24,  locked: 12000, spark: genSpark() },
  { key: 'video',  name: 'Video Credits',    sym: 'VIDEO',  qty: 7240,    usdPrice: 0.250000, color: 'video',  change: -0.65, locked: 0,     spark: genSpark() },
  { key: 'niche',  name: 'Niche Credits',    sym: 'NICHE',  qty: 320000,  usdPrice: 0.000400, color: 'niche',  change: 0.31,  locked: 0,     spark: genSpark() },
  { key: 'h100',   name: 'H100 GPU Credits', sym: 'H100',   qty: 245,     usdPrice: 2.99,     color: 'h100',   change: 0.41,  locked: 8,     spark: genSpark() },
  { key: 'h200',   name: 'H200 GPU Credits', sym: 'H200',   qty: 0,       usdPrice: 3.49,     color: 'h200',   change: 0,     locked: 0,     empty: true, spark: [] },
])

// =====================================================
// Formatters
// =====================================================
function fmt(n: number, dp = 2) {
  return Number(n).toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function fmtInt(n: number) {
  return Math.round(n).toLocaleString('en-US')
}
function fmtUsd(n: number, dp = 2) {
  return '$' + Number(n).toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}

// =====================================================
// Totals + allocation
// =====================================================
const totalUsd = computed(() => assets.reduce((s, a) => s + a.qty * a.usdPrice, 0))

// F20: in live mode (EXASCALE_API_MODE=local), overlay real credit balances from the ledger onto the
// matching assets. Mock mode keeps the showcase numbers untouched; assets with no live balance are
// left as-is (non-destructive).
const walletApiMode = useRuntimeConfig().public.apiMode
onMounted(async () => {
  if (walletApiMode !== 'local') return
  const creditKeyFor: Record<string, string> = {
    ai: 'ai_index', text: 'text', speech: 'speech', image: 'image', video: 'video', h100: 'gpu_h100', h200: 'gpu_h200',
  }
  try {
    const live = await useWallet().loadBalances()
    for (const a of assets) {
      const ct = creditKeyFor[a.key]
      if (!ct) continue
      const b = live.find((x) => x.credit_type === ct)
      if (b) { a.qty = Number(b.balance); a.locked = Number(b.locked_amount); a.empty = Number(b.balance) === 0 }
    }
  } catch { /* leave the showcase values on error */ }
})

interface AllocGroup { key: string; name: string; color: string; usd: number }
const allocationGroups = computed<AllocGroup[]>(() => {
  const get = (k: string) => assets.find(a => a.key === k)!
  return [
    { key: 'cash',  name: 'Cash',       color: COLOR.cash,  usd: get('cash').qty },
    { key: 'ai',    name: 'AI Credits', color: COLOR.ai,    usd: get('ai').qty * get('ai').usdPrice },
    { key: 'text',  name: 'Text',       color: COLOR.text,  usd: get('text').qty * get('text').usdPrice },
    { key: 'image', name: 'Image',      color: COLOR.image, usd: get('image').qty * get('image').usdPrice },
    { key: 'h100',  name: 'H100',       color: COLOR.h100,  usd: get('h100').qty * get('h100').usdPrice },
    {
      key: 'other', name: 'Other', color: '#5F5F5C',
      usd: get('speech').qty * get('speech').usdPrice
         + get('video').qty * get('video').usdPrice
         + get('niche').qty * get('niche').usdPrice,
    },
  ]
})

// =====================================================
// Sparkline path generator
// =====================================================
function sparkPaths(pts: number[], w = 240, h = 36) {
  if (!pts.length) return { line: '', fill: '' }
  const min = Math.min(...pts), max = Math.max(...pts)
  const range = max - min || 1
  const pad = 3
  const usableH = h - pad * 2
  const stepX = w / (pts.length - 1)
  const line = pts.map((p, i) => {
    const x = i * stepX
    const y = pad + usableH - ((p - min) / range) * usableH
    return (i === 0 ? 'M' : 'L') + x.toFixed(1) + ' ' + y.toFixed(1)
  }).join(' ')
  const fill = line + ` L ${w} ${h} L 0 ${h} Z`
  return { line, fill }
}

// =====================================================
// Recent movements
// =====================================================
interface Movement {
  time: string
  type: string
  typeKey: 'buy' | 'sell' | 'conv' | 'dep'
  asset: string
  amount: number
  price: number | string
  balance: number
}
const MOVEMENTS: Movement[] = [
  { time: '14:23:47', type: 'Trade buy',  typeKey: 'buy',  asset: 'AI Credits',    amount:  5000,    price: 0.000998, balance: 2425000 },
  { time: '14:18:23', type: 'Trade sell', typeKey: 'sell', asset: 'Text Credits',  amount: -250,     price: 0.001212, balance: 1200000 },
  { time: '13:55:12', type: 'Conversion', typeKey: 'conv', asset: 'AI → Speech',   amount: -1500,    price: '1.193',  balance: 2420000 },
  { time: '13:41:08', type: 'Trade buy',  typeKey: 'buy',  asset: 'H100 GPU',      amount:  2,       price: 2.948,    balance: 245 },
  { time: '12:18:55', type: 'Deposit',    typeKey: 'dep',  asset: 'USD',           amount:  500.00,  price: 1,        balance: 3891.42 },
  { time: '11:02:14', type: 'Trade sell', typeKey: 'sell', asset: 'Image Credits', amount: -1200,    price: 0.007981, balance: 152000 },
  { time: '10:47:31', type: 'Conversion', typeKey: 'conv', asset: 'AI → Image',    amount: -8500,    price: '0.126',  balance: 2421500 },
  { time: '09:38:22', type: 'Trade buy',  typeKey: 'buy',  asset: 'AI Credits',    amount:  20000,   price: 0.000994, balance: 2430000 },
]

function fmtMovementAmount(m: Movement) {
  const sign = m.amount >= 0 ? '+' : '−'
  const abs = Math.abs(m.amount)
  if (m.asset === 'USD') return sign + '$' + fmt(abs)
  return sign + fmtInt(abs)
}
function fmtMovementPrice(m: Movement) {
  if (typeof m.price === 'string') return m.price
  if (m.price >= 1) return '$' + fmt(m.price)
  return '$' + fmt(m.price, 6)
}
function fmtMovementBalance(m: Movement) {
  return (m.asset === 'USD' ? '$' : '') + fmtInt(m.balance)
}

// =====================================================
// Conversion drawer state (open by default per design)
// =====================================================
const drawerOpen = ref(true)
const convFromKey = ref('ai')
const convToKey = ref('text')
const convAmountText = ref('1,000')

const convFromAsset = computed(() => assets.find(a => a.key === convFromKey.value)!)
const convToAsset = computed(() => assets.find(a => a.key === convToKey.value)!)

const convAmount = computed(() => {
  const cleaned = convAmountText.value.replace(/[^\d.]/g, '')
  return parseFloat(cleaned) || 0
})

// Rate: from→to based on USD prices (simple cross-rate)
const convRate = computed(() => {
  if (convToAsset.value.usdPrice === 0) return 0
  return convFromAsset.value.usdPrice / convToAsset.value.usdPrice
})
const SPREAD = 0.005
const convEffRate = computed(() => convRate.value * (1 - SPREAD))
const convOut = computed(() => convAmount.value * convEffRate.value)

function setAmountPct(pct: number) {
  const v = Math.round(convFromAsset.value.qty * pct)
  convAmountText.value = fmtInt(v)
}
function openDrawer(fromKey: string) {
  convFromKey.value = fromKey
  // Default destination: if user selects AI, send to text; otherwise default to AI
  if (fromKey === 'ai') convToKey.value = 'text'
  else if (fromKey === 'cash') convToKey.value = 'ai'
  else convToKey.value = 'ai'
  drawerOpen.value = true
}
function closeDrawer() { drawerOpen.value = false }
function swapDirection() {
  const a = convFromKey.value
  convFromKey.value = convToKey.value
  convToKey.value = a
  // clamp amount to new from-asset balance
  if (convAmount.value > convFromAsset.value.qty) {
    convAmountText.value = fmtInt(convFromAsset.value.qty)
  }
}

// =====================================================
// Live drift simulation (subtle qty/price updates)
// =====================================================
const flashKeys = reactive<Record<string, 'up' | 'down' | undefined>>({})
let tickInterval: ReturnType<typeof setInterval> | null = null

function tick() {
  for (const a of assets) {
    if (a.empty) continue
    const noise = (Math.random() - 0.5) * (a.usdPrice * 0.002)
    a.usdPrice = Math.max(a.usdPrice * 0.98, a.usdPrice + noise)
    if (Math.random() < 0.25) {
      const dq = (Math.random() - 0.5) * (a.qty * 0.0008)
      const next = Math.max(0, a.qty + dq)
      if (Math.abs(next - a.qty) > 1) {
        flashKeys[a.key] = next > a.qty ? 'up' : 'down'
        setTimeout(() => { flashKeys[a.key] = undefined }, 700)
      }
      a.qty = next
    }
  }
}
const totalFlash = ref<'up' | 'down' | undefined>(undefined)
let lastTotal = 0
watch(totalUsd, (v) => {
  if (lastTotal && Math.abs(v - lastTotal) > 0.01) {
    totalFlash.value = v > lastTotal ? 'up' : 'down'
    setTimeout(() => { totalFlash.value = undefined }, 700)
  }
  lastTotal = v
})

const startTotal = ref(0)
onMounted(() => {
  lastTotal = totalUsd.value
  startTotal.value = totalUsd.value - 23.41
  tickInterval = setInterval(tick, 3000)
})
onBeforeUnmount(() => { if (tickInterval) clearInterval(tickInterval) })

const todayPnl = computed(() => totalUsd.value - startTotal.value)
const todayPct = computed(() => startTotal.value === 0 ? 0 : todayPnl.value / startTotal.value * 100)
</script>

<template>
  <div class="wallet-page">
    <!-- Sub-topbar: crumbs only (live data already in app topbar) -->
    <div class="subbar">
      <nav class="breadcrumbs">
        <a href="#">Account</a>
        <span class="sep">›</span>
        <span class="cur">Wallet</span>
      </nav>
    </div>

    <main class="main" :class="{ 'drawer-open': drawerOpen }">
      <div class="page">
        <!-- ================= HEADER ================= -->
        <section class="wallet-header">
          <div>
            <h1>Wallet</h1>
            <p class="sub">Across all credit types and cash balances</p>
          </div>
          <div class="wh-right">
            <span class="lbl">— Total value · USD</span>
            <div class="wh-total" :class="{ 'flash-up': totalFlash === 'up', 'flash-down': totalFlash === 'down' }">
              {{ fmtUsd(totalUsd) }}
            </div>
            <div class="wh-today" :class="todayPnl >= 0 ? 'pos' : 'neg'">
              {{ todayPnl >= 0 ? '▲ +$' : '▼ −$' }}{{ fmt(Math.abs(todayPnl)) }} ({{ todayPnl >= 0 ? '+' : '−' }}{{ Math.abs(todayPct).toFixed(2) }}%) today
            </div>
          </div>
        </section>

        <!-- ================= ALLOCATION ================= -->
        <section class="allocation">
          <div class="alloc-labels">
            <div
              v-for="g in allocationGroups"
              :key="g.key"
              class="alloc-label"
              :style="{ flex: (g.usd / totalUsd) }"
            >
              <div class="swatch" :style="{ background: g.color }" />
              <span class="nm">{{ g.name }}</span>
              <span class="pct">{{ (g.usd / totalUsd * 100).toFixed(1) }}%</span>
            </div>
          </div>
          <div class="alloc-bar">
            <div
              v-for="g in allocationGroups"
              :key="g.key"
              :style="{ background: g.color, width: (g.usd / totalUsd * 100) + '%' }"
            />
          </div>
        </section>

        <!-- ================= BALANCES ================= -->
        <div class="sec-head">
          <h2>Balances</h2>
          <span class="right">9 ASSETS · LIVE</span>
        </div>

        <div class="balance-grid" :class="{ 'two-col': drawerOpen }">
          <!-- CASH -->
          <div v-for="a in assets" :key="a.key" class="bcard" :class="{ featured: a.featured, empty: a.empty, cash: a.cls === 'cash' }" :data-key="a.key">
            <div class="accent" :style="{ background: COLOR[a.color], opacity: a.empty ? 0.3 : 1 }" />

            <div class="head-row">
              <span class="ttl">— {{ a.name }}</span>
              <span
                v-if="!a.empty && a.cls !== 'cash'"
                class="badge"
                :class="a.change >= 0 ? 'pos' : 'neg'"
              >
                {{ a.change >= 0 ? '▲' : '▼' }} {{ Math.abs(a.change).toFixed(2) }}%
              </span>
            </div>

            <!-- Cash special -->
            <template v-if="a.cls === 'cash'">
              <div class="qty-row">
                <span class="qty">{{ fmtUsd(a.qty) }}</span>
                <span class="secondary-qty">{{ a.secondary }}</span>
              </div>
              <span class="usd">USD primary · JPY available</span>
              <div class="actions">
                <NuxtLink to="/wallet/buy?credit=cash" class="act primary">
                  <svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" /></svg>
                  Add cash
                </NuxtLink>
                <button class="act">
                  <svg viewBox="0 0 16 16"><path d="M4 12L12 4M6 4h6v6" /></svg>
                  Send
                </button>
              </div>
            </template>

            <!-- Empty (H200) -->
            <template v-else-if="a.empty">
              <span class="qty">0</span>
              <span class="usd">No balance · Acquire to begin</span>
              <div class="actions">
                <NuxtLink to="/wallet/buy" class="act primary">
                  <svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" /></svg>
                  Acquire
                </NuxtLink>
                <button class="act">
                  <svg viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" /><path d="M8 7v4M8 5h.01" /></svg>
                  About
                </button>
              </div>
            </template>

            <!-- Standard credit -->
            <template v-else>
              <span
                class="qty"
                :class="{ 'flash-up': flashKeys[a.key] === 'up', 'flash-down': flashKeys[a.key] === 'down' }"
              >
                {{ fmtInt(a.qty) }}
                <span v-if="a.key === 'h100'" class="qty-unit">credits</span>
              </span>
              <span class="usd">
                {{ fmtUsd(a.qty * a.usdPrice, (a.qty * a.usdPrice) < 1 ? 4 : 2) }}
                <span class="secondary-tag">USD equivalent</span>
              </span>
              <span v-if="a.locked" class="lock">
                <svg viewBox="0 0 16 16"><rect x="3.5" y="7" width="9" height="6.5" /><path d="M5.5 7V5a2.5 2.5 0 015 0v2" /></svg>
                {{ fmtInt(a.locked) }} locked in open orders
              </span>
              <svg class="spark" viewBox="0 0 240 36" preserveAspectRatio="none">
                <path :d="sparkPaths(a.spark).fill" :fill="COLOR[a.color]" fill-opacity="0.10" />
                <path :d="sparkPaths(a.spark).line" fill="none" :stroke="COLOR[a.color]" stroke-width="1.4" />
              </svg>
              <div class="actions">
                <button class="act primary" @click="openDrawer(a.key)">
                  <svg viewBox="0 0 16 16"><path d="M3 5h8l-2-2M13 11H5l2 2" /></svg>
                  Convert
                </button>
                <button class="act">
                  <svg viewBox="0 0 16 16"><path d="M2 12l4-6 3 4 5-7" /></svg>
                  Trade
                </button>
                <button class="act">
                  <svg viewBox="0 0 16 16"><path d="M4 12L12 4M6 4h6v6" /></svg>
                  Send
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- ================= RECENT MOVEMENTS ================= -->
        <div class="sec-head movements-head">
          <h2>Recent movements</h2>
          <span class="right">LAST 8 TRANSACTIONS</span>
        </div>
        <table class="movements-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Type</th>
              <th>Asset</th>
              <th>Amount</th>
              <th>Price</th>
              <th>Balance after</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(m, i) in MOVEMENTS" :key="i">
              <td>{{ m.time }} UTC</td>
              <td><span class="type-pill" :class="`type-${m.typeKey}`">{{ m.type }}</span></td>
              <td>{{ m.asset }}</td>
              <td class="amt" :class="m.amount >= 0 ? 'pos' : 'neg'">{{ fmtMovementAmount(m) }}</td>
              <td>{{ fmtMovementPrice(m) }}</td>
              <td class="balance">{{ fmtMovementBalance(m) }}</td>
            </tr>
          </tbody>
        </table>
        <a href="#" class="full-link">
          View full history
          <svg viewBox="0 0 16 16"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
        </a>
      </div>

      <!-- ================= CONVERSION DRAWER ================= -->
      <aside class="drawer" :class="{ closed: !drawerOpen }">
        <div class="drawer-head">
          <h2>Convert credits</h2>
          <button class="drawer-close" type="button" @click="closeDrawer">
            <svg viewBox="0 0 16 16"><path d="M4 4l8 8M12 4l-8 8" /></svg>
          </button>
        </div>

        <div class="field-block">
          <span class="lbl">
            — From
            <span class="avail">{{ fmtInt(convFromAsset.qty) }} available</span>
          </span>
          <div class="select">
            <div class="sym-swatch" :style="{ background: COLOR[convFromAsset.color] }" />
            <div class="meta">
              <span class="nm">{{ convFromAsset.name }}</span>
              <span class="bal">{{ convFromAsset.sym }} · {{ convFromAsset.description ?? 'Available to convert' }}</span>
            </div>
            <span class="chev">
              <svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" /></svg>
            </span>
          </div>
        </div>

        <div class="swap-arrow">
          <span class="ring" title="Swap direction" @click="swapDirection">
            <svg viewBox="0 0 16 16"><path d="M8 3v10M4 9l4 4 4-4" /></svg>
          </span>
        </div>

        <div class="field-block">
          <span class="lbl">— To</span>
          <div class="select">
            <div class="sym-swatch" :style="{ background: COLOR[convToAsset.color] }" />
            <div class="meta">
              <span class="nm">{{ convToAsset.name }}</span>
              <span class="bal">{{ convToAsset.sym }} · destination</span>
            </div>
            <span class="chev">
              <svg viewBox="0 0 16 16"><path d="M4 6l4 4 4-4" /></svg>
            </span>
          </div>
        </div>

        <div class="field-block">
          <span class="lbl">— Amount</span>
          <div class="amount-input">
            <input v-model="convAmountText" type="text" />
            <span class="unit">{{ convFromAsset.name }}</span>
          </div>
          <div class="amount-row">
            <button type="button" @click="setAmountPct(0.25)">25%</button>
            <button type="button" @click="setAmountPct(0.5)">50%</button>
            <button type="button" @click="setAmountPct(0.75)">75%</button>
            <button type="button" @click="setAmountPct(1)">Max</button>
          </div>
        </div>

        <div class="preview-card">
          <div class="pv-out">
            <span>{{ fmtInt(convAmount) }}</span> {{ convFromAsset.sym }}
            <span class="arr">→</span>
            <span>{{ fmtInt(convOut) }}</span> {{ convToAsset.sym }}
          </div>
          <div class="pv-rows">
            <div class="pv-row live">
              <span class="k">RATE · LIVE</span>
              <span class="v">1 {{ convFromAsset.sym }} = {{ convRate.toFixed(3) }} {{ convToAsset.sym }}</span>
            </div>
            <div class="pv-row">
              <span class="k">HOUSE SPREAD</span>
              <span class="v">0.50%</span>
            </div>
            <div class="pv-row">
              <span class="k">EFFECTIVE RATE</span>
              <span class="v">{{ convEffRate.toFixed(3) }}</span>
            </div>
            <div class="pv-row">
              <span class="k">EST. SETTLEMENT</span>
              <span class="v">Instant</span>
            </div>
          </div>
        </div>

        <div class="drawer-submit">
          <button type="button">
            Convert {{ fmtInt(convAmount) }} {{ convFromAsset.sym }} → {{ fmtInt(convOut) }} {{ convToAsset.sym }}
            <svg viewBox="0 0 16 16"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
          </button>
          <div class="submit-note">Rate locked for 8 seconds at submit</div>
        </div>
      </aside>
    </main>
  </div>
</template>

<style scoped>
.wallet-page {
  --border-2: rgba(255, 255, 255, 0.14);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.55;
  font-feature-settings: 'tnum' on, 'ss01' on;
  color: var(--text);
}

.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: var(--canvas);
}
.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.breadcrumbs a { color: var(--text-2); text-decoration: none; }
.breadcrumbs a:hover { color: var(--text); }
.breadcrumbs .sep { color: var(--text-3); }
.breadcrumbs .cur { color: var(--text); }

.main { position: relative; transition: padding-right 280ms cubic-bezier(0.2, 0, 0, 1); }
.main.drawer-open { padding-right: 460px; }

/* ============================================
   Page
   ============================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 40px 40px 80px;
}

/* Wallet header */
.wallet-header {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 64px;
  align-items: end;
  padding-bottom: 40px;
  border-bottom: 1px solid var(--border);
}
.wallet-header h1 {
  font-family: var(--font-display);
  font-size: 40px;
  font-weight: 600;
  letter-spacing: -0.025em;
  margin: 0 0 8px;
  line-height: 1.05;
  color: var(--text);
}
.wallet-header .sub {
  color: var(--text-2);
  font-size: 16px;
  margin: 0;
}
.wh-right { text-align: right; }
.wh-right .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.20em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  margin-bottom: 6px;
  display: block;
}
.wh-total {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 60px;
  font-weight: 600;
  letter-spacing: -0.035em;
  line-height: 1;
  color: var(--text);
  transition: color 600ms;
}
.wh-total.flash-up { color: var(--pos); }
.wh-total.flash-down { color: var(--neg); }
.wh-today {
  margin-top: 10px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
}
.wh-today.pos { color: var(--pos); }
.wh-today.neg { color: var(--neg); }

/* ============================================
   Allocation strip
   ============================================ */
.allocation { padding: 36px 0 8px; }
.alloc-labels { display: flex; gap: 4px; margin-bottom: 8px; }
.alloc-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-right: 12px;
}
.alloc-label .swatch { width: 100%; height: 2px; margin-bottom: 6px; }
.alloc-label .nm {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
}
.alloc-label .pct {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.alloc-bar { height: 14px; display: flex; gap: 1px; overflow: hidden; }
.alloc-bar > div { height: 100%; transition: width 400ms ease-out; }

/* ============================================
   Section head
   ============================================ */
.sec-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 56px 0 20px;
}
.sec-head h2 {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0;
}
.sec-head .right {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
}
.movements-head { margin-top: 64px; }

/* ============================================
   Balance cards
   ============================================ */
.balance-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.balance-grid.two-col { grid-template-columns: repeat(2, 1fr); }

.bcard {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 20px 22px 18px;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 150ms, background 150ms;
}
.bcard:hover { border-color: var(--border-2); background: rgba(255, 255, 255, 0.012); }
.bcard .accent { position: absolute; top: 0; left: 0; right: 0; height: 2px; }
.bcard.featured {
  border-color: rgba(200, 242, 92, 0.30);
  background: linear-gradient(180deg, rgba(200, 242, 92, 0.04), transparent 60%);
}
.bcard.featured:hover { border-color: rgba(200, 242, 92, 0.55); }
.bcard.empty { background: rgba(255, 255, 255, 0.012); }
.bcard.empty .qty { color: var(--text-3); }

.head-row { display: flex; align-items: center; gap: 8px; }
.bcard .ttl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-2);
  font-weight: 500;
}
.bcard .badge {
  margin-left: auto;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 10.5px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.04em;
}
.bcard .badge.pos { background: var(--pos-soft); color: var(--pos); }
.bcard .badge.neg { background: var(--neg-soft); color: var(--neg); }

.bcard .qty {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
  transition: color 600ms;
}
.bcard.featured .qty { font-size: 34px; }
.bcard .qty.flash-up { color: var(--pos); }
.bcard .qty.flash-down { color: var(--neg); }
.qty-unit { font-size: 14px; color: var(--text-3); }

.bcard .usd {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text-2);
}
.bcard .secondary-tag { color: var(--text-3); margin-left: 8px; }

.bcard .lock {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--warn);
  background: rgba(245, 158, 11, 0.08);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  align-self: flex-start;
  letter-spacing: 0.02em;
}
.bcard .lock svg { width: 11px; height: 11px; stroke: currentColor; fill: none; stroke-width: 1.4; }

.bcard svg.spark {
  width: 100%;
  height: 36px;
  display: block;
  margin: 2px 0 4px;
}

.bcard .actions {
  display: flex;
  gap: 6px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}
.bcard .act {
  flex: 1;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 11.5px;
  font-weight: 500;
  height: 30px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  text-decoration: none;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.bcard .act:hover { background: var(--hover); color: var(--text); }
.bcard .act.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
  font-weight: 600;
}
.bcard .act.primary:hover { background: var(--brand-hov); }
.bcard .act svg { width: 13px; height: 13px; stroke: currentColor; fill: none; stroke-width: 1.6; }

.bcard.cash .qty-row { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
.bcard.cash .secondary-qty {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 16px;
  color: var(--text-3);
}

/* ============================================
   Recent movements table
   ============================================ */
.movements-table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
}
.movements-table th, .movements-table td {
  text-align: right;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
}
.movements-table th:first-child, .movements-table td:first-child {
  text-align: left;
  color: var(--text-3);
}
.movements-table th:nth-child(2), .movements-table td:nth-child(2),
.movements-table th:nth-child(3), .movements-table td:nth-child(3) {
  text-align: left;
}
.movements-table thead th {
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  padding: 8px 14px;
  border-bottom-color: var(--border-2);
}
.movements-table tbody tr { transition: background 120ms; cursor: pointer; }
.movements-table tbody tr:hover { background: rgba(255, 255, 255, 0.02); }
.movements-table .type-pill {
  display: inline-block;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  letter-spacing: 0.06em;
  font-weight: 500;
}
.movements-table .type-buy  { background: var(--pos-soft); color: var(--pos); }
.movements-table .type-sell { background: var(--neg-soft); color: var(--neg); }
.movements-table .type-conv { background: rgba(200, 242, 92, 0.10); color: var(--brand); }
.movements-table .type-dep  { background: rgba(74, 144, 226, 0.12); color: var(--accent); }
.movements-table .amt.pos { color: var(--pos); }
.movements-table .amt.neg { color: var(--neg); }
.movements-table .balance { color: var(--text); }

.full-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 24px;
  color: var(--text-2);
  font-size: 13px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 1px;
  text-decoration: none;
}
.full-link:hover { color: var(--brand); border-bottom-color: var(--brand); }
.full-link svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }

/* ============================================
   Conversion drawer
   ============================================ */
.drawer {
  position: fixed;
  top: var(--topbar-h);
  right: 0;
  bottom: 0;
  width: 460px;
  background: var(--canvas);
  border-left: 1px solid var(--border-2);
  z-index: 40;
  display: flex;
  flex-direction: column;
  padding: 28px 32px;
  transform: translateX(0);
  transition: transform 280ms cubic-bezier(0.2, 0, 0, 1);
  box-shadow: -24px 0 48px rgba(0, 0, 0, 0.4);
  overflow-y: auto;
}
.drawer.closed { transform: translateX(100%); }

.drawer-head {
  display: flex;
  align-items: center;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}
.drawer-head h2 {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0;
}
.drawer-close {
  margin-left: auto;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  width: 30px;
  height: 30px;
  cursor: pointer;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.drawer-close:hover { color: var(--text); background: var(--hover); }
.drawer-close svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }

.field-block { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.field-block .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.20em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  display: flex;
  justify-content: space-between;
}
.field-block .lbl .avail {
  color: var(--text-2);
  letter-spacing: 0.04em;
  font-size: 10px;
}

.select {
  height: 48px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 0 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: border-color 120ms, background 120ms;
}
.select:hover { border-color: var(--border-2); background: var(--hover); }
.select .sym-swatch { width: 18px; height: 18px; flex-shrink: 0; }
.select .meta { display: flex; flex-direction: column; line-height: 1.15; }
.select .nm { font-family: var(--font-sans); font-weight: 600; font-size: 14px; color: var(--text); }
.select .bal {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}
.select .chev { margin-left: auto; color: var(--text-3); }
.select .chev svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }

.swap-arrow { display: flex; justify-content: center; margin: 6px 0; }
.swap-arrow .ring {
  width: 32px;
  height: 32px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  cursor: pointer;
  transition: transform 200ms, color 120ms;
}
.swap-arrow .ring:hover { color: var(--brand); transform: rotate(180deg); }
.swap-arrow .ring svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }

.amount-input {
  height: 60px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 0 16px;
  display: flex;
  align-items: center;
}
.amount-input input {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
  outline: none;
  text-align: right;
}
.amount-input input:focus { outline: 1px solid var(--accent); outline-offset: -1px; }
.amount-input .unit {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.18em;
  text-transform: uppercase;
  margin-left: 10px;
}

.amount-row { display: flex; gap: 6px; }
.amount-row button {
  flex: 1;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10.5px;
  height: 26px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  letter-spacing: 0.02em;
  transition: background 120ms, color 120ms;
}
.amount-row button:hover { background: var(--hover); color: var(--text); }

.preview-card {
  margin-top: 24px;
  background: rgba(200, 242, 92, 0.04);
  border: 1px solid rgba(200, 242, 92, 0.20);
  border-radius: var(--radius-sm);
  padding: 20px;
}
.preview-card .pv-out {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 22px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.015em;
  margin-bottom: 6px;
}
.preview-card .pv-out .arr { color: var(--brand); margin: 0 8px; }
.preview-card .pv-rows {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid rgba(200, 242, 92, 0.16);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.preview-card .pv-row {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
}
.preview-card .pv-row .k { color: var(--text-3); letter-spacing: 0.04em; }
.preview-card .pv-row .v { color: var(--text); }
.preview-card .pv-row.live .v::after {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  margin-left: 6px;
  vertical-align: 2px;
  animation: pulseDot 2s ease-in-out infinite;
}
@keyframes pulseDot { 50% { opacity: 0.35; } }

.drawer-submit {
  margin-top: auto;
  padding-top: 20px;
  border-top: 1px solid var(--border);
}
.drawer-submit button {
  width: 100%;
  height: 48px;
  background: var(--brand);
  color: var(--canvas);
  border: none;
  border-radius: var(--radius-sm);
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 120ms;
}
.drawer-submit button:hover { background: var(--brand-hov); }
.drawer-submit button svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }
.drawer-submit .submit-note {
  margin-top: 10px;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.06em;
  text-align: center;
  text-transform: uppercase;
}

/* ============================================
   Responsive
   ============================================ */
@media (max-width: 1200px) {
  .balance-grid,
  .balance-grid.two-col { grid-template-columns: repeat(2, 1fr); }
  .wallet-header { grid-template-columns: 1fr; gap: 32px; }
  .wh-right { text-align: left; }
  .drawer { width: 380px; }
  .main.drawer-open { padding-right: 380px; }
}
@media (max-width: 900px) {
  .main.drawer-open { padding-right: 0; }
  .drawer { width: 100%; }
}
</style>
