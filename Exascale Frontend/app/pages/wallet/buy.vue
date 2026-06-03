<script setup lang="ts">
/**
 * /wallet/buy — Buy credits flow (3 steps: Amount → Payment → Confirm).
 *
 * Standalone chrome (no app sidebar / no marketing nav) to keep the buyer
 * focused on the order. Light theme. Renders inside the project's light
 * tokens by virtue of `layout: false`.
 *
 * The standalone design ships a bottom-right "Interactive / Stacked" tweak —
 * skipped here, that's a design-tool affordance, not product UX.
 */

definePageMeta({ layout: false })
useHead({ title: 'Buy Credits — Exascale', htmlAttrs: { 'data-theme': 'dark' } })

const route = useRoute()

// ── Real-money KYC gate (F22) ───────────────────────────────────────────────────────────────
// KYC is NOT an onboarding step — it's a one-time gate at the first REAL-MONEY purchase. Sandbox
// (paper) accounts and all credit usage (inference, GPU compute) need none. Server-side enforcement
// lands with F22 (M3); this is the client gate that blocks a real-money order until identity is
// verified. Preview the gate in dev with ?kyc=1.
const { user } = useAuth()
const realMoney = computed(() => user.value?.is_paper === false)
// No verification backend yet → real-money accounts are treated as unverified until F22 marks them.
const kycVerified = computed(() => (user.value as { kyc_status?: string } | null)?.kyc_status === 'verified')
const showKycGate = computed(() => (realMoney.value && !kycVerified.value) || route.query.kyc === '1')

type StepKey = 'amount' | 'payment' | 'confirm'
type CreditType = 'AI_INDEX' | 'CASH' | 'SUB'
type Payment = 'card' | 'wire' | 'ach'

interface SubCredit { id: string; label: string; pxUsd: number }

const STEPS: Array<{ key: StepKey; label: string; title: string; eyebrow: string; lede?: string }> = [
  { key: 'amount',  label: 'Amount',  title: 'Choose amount & credit type', eyebrow: 'Step 01 · Order',         lede: "Pick the amount and what you'd like to hold the balance as." },
  { key: 'payment', label: 'Payment', title: 'How will you pay?',           eyebrow: 'Step 02 · Settlement',    lede: 'Choose how to settle. Wire is recommended for orders over $25,000.' },
  { key: 'confirm', label: 'Confirm', title: 'Order placed',                eyebrow: 'Step 03 · Confirmation' },
]

const QUICK_AMOUNTS = [100, 500, 1000, 5000, 10000, 50000]

const SUB_CREDITS: SubCredit[] = [
  { id: 'TEXT',   label: 'TEXT-INDEX  ·  text generation',     pxUsd: 0.001210 },
  { id: 'SPEECH', label: 'SPEECH-INDEX  ·  speech synthesis',  pxUsd: 0.001200 },
  { id: 'IMAGE',  label: 'IMAGE-INDEX  ·  image generation',   pxUsd: 0.008000 },
  { id: 'VIDEO',  label: 'VIDEO-INDEX  ·  video generation',   pxUsd: 0.250000 },
  { id: 'NICHE',  label: 'NICHE-INDEX  ·  long-tail tasks',    pxUsd: 0.000400 },
]

const CREDIT_DEFS: Record<CreditType, {
  short: string
  badge: string
  name: string
  desc: string
  pxLabel: string
  featured?: boolean
}> = {
  AI_INDEX: {
    short: 'AI', badge: 'AI-IDX',
    name: 'AI Index Credits',
    desc: 'Composite reference credit. Redeemable across all inference categories.',
    pxLabel: '$0.001005 / credit',
    featured: true,
  },
  CASH: {
    short: '$', badge: 'CASH',
    name: 'Cash Balance',
    desc: 'Hold as USD on the venue. Convert on demand.',
    pxLabel: 'Holds USD · convert anytime',
  },
  SUB: {
    short: 'SC', badge: 'SUB',
    name: 'Specific Sub-Credit',
    desc: 'Single inference category. Higher precision, thinner book.',
    pxLabel: '$0.00121 / credit',
  },
}

const PAYMENT_METHODS: Array<{ key: Payment; name: string; sub: string; icons: string[] }> = [
  { key: 'card', name: 'Credit / debit card',  sub: 'Instant · $25 limit for first purchase',         icons: ['VISA', 'MASTERCARD', 'AMEX'] },
  { key: 'wire', name: 'Bank transfer / wire', sub: '1–3 business days · no limit · recommended',     icons: ['WIRE', 'SEPA', 'SWIFT'] },
  { key: 'ach',  name: 'ACH transfer',         sub: '3–5 business days · US only · ≤ $250K',          icons: ['ACH'] },
]

const ORDER_ID = 'PO-7A3C91'
const WIRE_REF = `EX-${ORDER_ID.replace(/-/g, '')}`

// =====================================================
// State
// =====================================================
const stepIdx = ref(0)
const amount = ref(1000)
const amountStr = ref('1,000')
const creditType = ref<CreditType>('AI_INDEX')
const subCredit = ref('TEXT')
const payment = ref<Payment>('wire')
const termsAccepted = ref(false)
const livePx = ref(0.001005)

// The active step definition. stepIdx is always a valid index (clamped by next/back/goTo).
const step = computed(() => STEPS[stepIdx.value]!)

// Allow ?credit=ai|sub|cash for pre-selection from Wallet links
onMounted(() => {
  const q = route.query.credit
  if (q === 'cash') creditType.value = 'CASH'
  else if (q === 'sub') creditType.value = 'SUB'
  else creditType.value = 'AI_INDEX'
})

// =====================================================
// Formatters
// =====================================================
function parseAmount(str: string): number {
  const cleaned = String(str).replace(/[^0-9.]/g, '')
  const n = parseFloat(cleaned)
  return isNaN(n) ? 0 : n
}
function fmtAmount(n: number): string {
  return n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
function fmtUSD(n: number, dec = 2): string {
  return n.toLocaleString('en-US', { minimumFractionDigits: dec, maximumFractionDigits: dec })
}

const activeCreditPx = computed(() => {
  if (creditType.value === 'AI_INDEX') return livePx.value
  if (creditType.value === 'CASH') return 1
  return SUB_CREDITS.find(s => s.id === subCredit.value)?.pxUsd ?? 0.00121
})

const creditsReceived = computed(() => {
  if (creditType.value === 'CASH') return amount.value
  if (activeCreditPx.value === 0) return 0
  return Math.floor(amount.value / activeCreditPx.value)
})

const creditLabel = computed(() => {
  if (creditType.value === 'CASH') return 'USD cash'
  if (creditType.value === 'SUB') return subCredit.value + ' credits'
  return 'AI Index credits'
})
const creditLabelShort = computed(() => {
  if (creditType.value === 'CASH') return 'USD'
  if (creditType.value === 'SUB') return subCredit.value + ' credits'
  return 'AI Index credits'
})

const canAdvance = computed(() => {
  if (stepIdx.value === 0) return amount.value > 0
  if (stepIdx.value === 1) return termsAccepted.value
  return false
})

// =====================================================
// Navigation
// =====================================================
function next() {
  if (!canAdvance.value) return
  if (stepIdx.value < STEPS.length - 1) {
    stepIdx.value++
    if (typeof window !== 'undefined') window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}
function back() {
  if (stepIdx.value > 0) stepIdx.value--
}
function goTo(i: number) {
  if (i < 0 || i >= STEPS.length) return
  stepIdx.value = i
}

function onAmountInput(e: Event) {
  const v = (e.target as HTMLInputElement).value
  amountStr.value = v
  amount.value = parseAmount(v)
}
function onAmountBlur() {
  amountStr.value = fmtAmount(amount.value)
}
function setQuickAmount(v: number) {
  amount.value = v
  amountStr.value = fmtAmount(v)
}

function reset() {
  stepIdx.value = 0
  termsAccepted.value = false
  amount.value = 1000
  amountStr.value = '1,000'
}

// =====================================================
// Wire copy-to-clipboard
// =====================================================
const copiedKey = ref<string | null>(null)
async function copy(text: string, key: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    setTimeout(() => { if (copiedKey.value === key) copiedKey.value = null }, 1400)
  } catch {
    // clipboard unavailable — no-op
  }
}

// =====================================================
// Live price tick (Step 1 only)
// =====================================================
let priceTimer: ReturnType<typeof setInterval> | null = null
watch(stepIdx, (idx) => {
  if (idx === 0 && !priceTimer) {
    priceTimer = setInterval(() => {
      const delta = (Math.random() - 0.5) * 0.0000015
      livePx.value = Math.max(0.000990, Math.min(0.001020, livePx.value + delta))
    }, 5000)
  } else if (idx !== 0 && priceTimer) {
    clearInterval(priceTimer)
    priceTimer = null
  }
}, { immediate: true })
onBeforeUnmount(() => { if (priceTimer) clearInterval(priceTimer) })

// =====================================================
// Wallet balance reflects the buy amount post-confirm (display only)
// =====================================================
const walletBalance = computed(() => 10118.46)

// =====================================================
// Confirmation derived
// =====================================================
const statusLabel = computed(() => {
  if (payment.value === 'card') return 'PROCESSING'
  if (payment.value === 'wire') return 'AWAITING WIRE'
  return 'AWAITING ACH'
})
const settlementText = computed(() => {
  if (payment.value === 'card') return 'Credits will appear in your wallet within 5 minutes.'
  if (payment.value === 'wire') return 'Credits will appear in your wallet within 1–3 business days after we receive your wire.'
  return 'Credits will appear in your wallet within 3–5 business days after ACH clears.'
})
const methodLabel = computed(() => {
  if (payment.value === 'card') return 'Card · VISA ••4242'
  if (payment.value === 'wire') return 'Bank wire · JPMorgan Chase'
  return 'ACH · Chase ••8821'
})
const methodTiming = computed(() => {
  if (payment.value === 'card') return 'Card · ~10 s'
  if (payment.value === 'wire') return 'Bank wire · 1–3 business days'
  return 'ACH · 3–5 business days'
})

// Routing for confirmation buttons
const router = useRouter()
function goWallet() { router.push('/wallet') }
function goTrade()  { router.push('/trade') }
</script>

<template>
  <div class="buy-shell">
    <!-- ===== Top chrome ===== -->
    <header class="topbar">
      <NuxtLink to="/wallet" class="brand">
        <span class="brand-mark" />
        EXASCALE
      </NuxtLink>
      <div class="topbar-right">
        <span class="env-pill" :class="{ live: realMoney }">{{ realMoney ? 'LIVE' : 'PAPER' }}</span>
        <span class="balance">
          Wallet balance
          <strong>${{ fmtUSD(walletBalance) }}</strong>
        </span>
        <NuxtLink to="/wallet" class="exit">Save &amp; exit</NuxtLink>
      </div>
    </header>

    <!-- ===== Page ===== -->
    <div class="page" data-theme="dark">
      <div class="page-header">
        <div class="eyebrow">Buy credits · funding</div>
        <h1 class="page-title">Add funds to your account.</h1>
        <p class="page-subtitle">
          Convert USD into AI Index credits or hold as cash. No conversion fee on bulk purchase.
          Settlement instant on card, 1–3 business days on wire.
        </p>
      </div>

      <!-- ===== KYC gate — a real-money purchase needs a one-time identity check (F22). Sandbox skips it. -->
      <div v-if="showKycGate" class="kyc-gate">
        <div class="kyc-eyebrow">Compliance · Identity verification</div>
        <h2 class="kyc-title">Verify your identity to buy with real money.</h2>
        <p class="kyc-lede">
          Real-money purchases require a one-time identity check (KYC/AML) — credits are prepaid
          service units, so we verify the buyer before the first real-money order. Your
          <strong>sandbox</strong> account and all credit usage (inference, GPU compute) need none.
        </p>
        <ul class="kyc-list">
          <li><span class="k-ix">01</span><div><strong>Government ID</strong><span>Passport or driver's licence — verified in minutes.</span></div></li>
          <li><span class="k-ix">02</span><div><strong>Business details</strong><span>Legal entity + registered address for the account.</span></div></li>
          <li><span class="k-ix">03</span><div><strong>Source of funds</strong><span>A short declaration for AML compliance.</span></div></li>
        </ul>
        <div class="kyc-actions">
          <NuxtLink to="/onboarding/kyc" class="btn primary lg">Begin verification →</NuxtLink>
          <NuxtLink to="/console" class="btn secondary lg">Keep using sandbox</NuxtLink>
        </div>
        <div class="kyc-foot">Identity data is handled under our KYC/AML policy · SOC 2 controls · encrypted at rest.</div>
      </div>

      <template v-else>
      <!-- Sandbox reassurance — paper credits need no verification (the policy, surfaced up front) -->
      <div v-if="!realMoney" class="sandbox-note">
        <span class="sn-dot" />
        Sandbox mode — buying paper credits, no identity verification needed. Real-money purchases will require a one-time KYC check.
      </div>

      <!-- Stepper -->
      <div class="steps" role="tablist" aria-label="Purchase steps">
        <button
          v-for="(s, i) in STEPS"
          :key="s.key"
          role="tab"
          type="button"
          class="step"
          :class="{ active: i === stepIdx, done: i < stepIdx }"
          :aria-selected="i === stepIdx"
          @click="goTo(i)"
        >
          <span class="step-num">
            <svg v-if="i < stepIdx" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="square" class="check-svg">
              <path d="M3 8.5l3.2 3.2L13 5" />
            </svg>
            <template v-else>{{ String(i + 1).padStart(2, '0') }}</template>
          </span>
          <span class="step-info">
            <span class="step-label">{{ s.label }}</span>
            <span class="step-title">{{ s.title }}</span>
          </span>
        </button>
      </div>

      <!-- ===== Card ===== -->
      <div class="card">
        <div class="card-head">
          <div>
            <div class="eyebrow">{{ step.eyebrow }}</div>
            <h2 class="card-ttl">{{ step.title }}</h2>
            <p v-if="step.lede" class="card-lede">{{ step.lede }}</p>
          </div>
          <div class="card-meta">
            <strong>{{ String(stepIdx + 1).padStart(2, '0') }}</strong> / 03 ·
            <span class="dim">{{ stepIdx === 2 ? 'Order ' + ORDER_ID : 'In progress' }}</span>
          </div>
        </div>

        <div class="card-body">
          <div class="step-content">
            <!-- =========================================
                 STEP 1 — Amount
                 ========================================= -->
            <template v-if="stepIdx === 0">
              <div class="field-label">
                <span>— Amount
                  <BaseHelpDot
                    title="Amount (USD)"
                    what="How much US dollars you want to convert into credits in this transaction."
                    why="The fee is a flat 1% regardless of amount, so larger top-ups are slightly more efficient. Buying $10K of credits costs $100 in fee; buying $100 costs $1."
                    field="wallet.buy.amount"
                    placement="right"
                  />
                </span>
                <span class="req">USD</span>
              </div>
              <label class="amount-wrap">
                <span class="amount-prefix">$</span>
                <input
                  class="amount-input"
                  type="text"
                  inputmode="decimal"
                  :value="amountStr"
                  placeholder="0"
                  autocomplete="off"
                  @input="onAmountInput"
                  @blur="onAmountBlur"
                />
                <span class="amount-suffix">USD</span>
              </label>

              <div class="qa-row">
                <button
                  v-for="v in QUICK_AMOUNTS"
                  :key="v"
                  type="button"
                  class="qa-btn"
                  :class="{ active: amount === v }"
                  @click="setQuickAmount(v)"
                >${{ v.toLocaleString('en-US') }}</button>
              </div>

              <div class="field-label"><span>— Receive credits as</span></div>
              <div class="credit-row">
                <button
                  v-for="(c, k) in CREDIT_DEFS"
                  :key="k"
                  type="button"
                  class="credit-card"
                  :class="{ active: creditType === k }"
                  @click="creditType = k"
                >
                  <span v-if="c.featured" class="ribbon">Recommended</span>
                  <div class="top-line">
                    <span
                      class="symbol"
                      :class="['sym-' + k.toLowerCase()]"
                    >{{ c.short }}</span>
                    <span>{{ c.badge }}</span>
                  </div>
                  <div class="name">{{ c.name }}</div>
                  <div class="desc">{{ c.desc }}</div>
                  <div class="px">{{ c.pxLabel }}</div>
                </button>
              </div>

              <select v-if="creditType === 'SUB'" v-model="subCredit" class="sub-select">
                <option v-for="s in SUB_CREDITS" :key="s.id" :value="s.id">
                  {{ s.label }} · ${{ s.pxUsd.toFixed(6) }}
                </option>
              </select>

              <!-- Live preview -->
              <div class="preview">
                <div class="preview-head">
                  <span>— Order preview</span>
                  <span class="live"><span class="pulse" /> Live · refreshes ~5s</span>
                </div>
                <div class="preview-body">
                  <div class="preview-row major">
                    <span class="lbl">You pay</span>
                    <span class="val">
                      ${{ fmtAmount(amount) }}<span class="unit">USD</span>
                    </span>
                  </div>
                  <div class="preview-row major receive">
                    <span class="lbl">You receive</span>
                    <span class="val">
                      {{ creditType === 'CASH' ? '$' + fmtUSD(creditsReceived, 2) : creditsReceived.toLocaleString('en-US') }}
                      <span class="unit">{{ creditLabelShort }}</span>
                    </span>
                  </div>
                  <div class="preview-divider" />
                  <div class="preview-row">
                    <span class="lbl">Effective rate</span>
                    <span class="val small">${{ activeCreditPx.toFixed(6) }} per credit</span>
                  </div>
                  <div class="preview-row">
                    <span class="lbl">Conversion fee</span>
                    <span class="val small">
                      $0.00
                      <span class="unit">{{ amount >= 100 ? 'waived · bulk' : 'standard' }}</span>
                    </span>
                  </div>
                </div>
              </div>
            </template>

            <!-- =========================================
                 STEP 2 — Payment method
                 ========================================= -->
            <template v-else-if="stepIdx === 1">
              <div class="field-label"><span>— Payment method</span></div>
              <div class="pm-row">
                <div
                  v-for="m in PAYMENT_METHODS"
                  :key="m.key"
                  class="pm-card"
                  :class="{ active: payment === m.key }"
                  @click="payment = m.key"
                >
                  <div class="pm-head">
                    <span class="pm-radio" />
                    <span class="pm-info">
                      <div class="pm-name">{{ m.name }}</div>
                      <div class="pm-sub">{{ m.sub }}</div>
                    </span>
                    <span class="pm-icons">
                      <span v-for="ic in m.icons" :key="ic" class="ic">{{ ic }}</span>
                    </span>
                  </div>

                  <!-- Card form -->
                  <div v-if="payment === m.key && m.key === 'card'" class="pm-body" @click.stop>
                    <div class="stripe-grid">
                      <div class="f-input">
                        <span class="f-label">Card</span>
                        <input type="text" value="4242 4242 4242 4242" autocomplete="cc-number" maxlength="19" />
                        <span class="brand-mark-pill">VISA</span>
                      </div>
                      <div class="col2">
                        <div class="f-input">
                          <span class="f-label">Exp</span>
                          <input type="text" value="12 / 28" autocomplete="cc-exp" />
                        </div>
                        <div class="f-input">
                          <span class="f-label">CVC</span>
                          <input type="text" value="•••" autocomplete="cc-csc" maxlength="4" />
                        </div>
                      </div>
                      <div class="f-input">
                        <span class="f-label">Postal</span>
                        <input type="text" value="10004" autocomplete="postal-code" />
                        <span class="brand-mark-pill">US</span>
                      </div>
                    </div>
                  </div>

                  <!-- Wire body -->
                  <div v-if="payment === m.key && m.key === 'wire'" class="pm-body" @click.stop>
                    <p class="wire-intro">
                      Send funds from your firm account to the details below. Include the reference number
                      in the wire memo so we can credit your account automatically.
                    </p>
                    <dl class="wire-table">
                      <dt>Bank</dt>
                      <dd>
                        <span>JPMorgan Chase, N.A. — New York</span>
                        <button type="button" class="copy-btn" @click="copy('JPMorgan Chase, N.A. — New York', 'bank')">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                          {{ copiedKey === 'bank' ? 'Copied' : 'Copy' }}
                        </button>
                      </dd>
                      <dt>SWIFT / BIC</dt>
                      <dd>
                        CHASUS33XXX
                        <button type="button" class="copy-btn" @click="copy('CHASUS33XXX', 'swift')">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                          {{ copiedKey === 'swift' ? 'Copied' : 'Copy' }}
                        </button>
                      </dd>
                      <dt>ABA / routing</dt>
                      <dd>
                        021000021
                        <button type="button" class="copy-btn" @click="copy('021000021', 'aba')">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                          {{ copiedKey === 'aba' ? 'Copied' : 'Copy' }}
                        </button>
                      </dd>
                      <dt>Account number</dt>
                      <dd>
                        5491 0428 7741
                        <button type="button" class="copy-btn" @click="copy('5491 0428 7741', 'acct')">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                          {{ copiedKey === 'acct' ? 'Copied' : 'Copy' }}
                        </button>
                      </dd>
                      <dt>Beneficiary</dt>
                      <dd>
                        Exascale Markets, Inc.
                        <button type="button" class="copy-btn" @click="copy('Exascale Markets, Inc.', 'bene')">
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="5" width="8" height="8" /><path d="M3 11V3h8" /></svg>
                          {{ copiedKey === 'bene' ? 'Copied' : 'Copy' }}
                        </button>
                      </dd>
                      <dt>Address</dt>
                      <dd>200 Pine Street, San Francisco, CA 94104, US</dd>
                    </dl>
                    <div class="wire-ref">
                      <span class="ref-icon">!</span>
                      <div>
                        <div class="ref-intro">Use this reference in your wire memo — required:</div>
                        <strong class="ref-code">{{ WIRE_REF }}</strong>
                      </div>
                    </div>
                  </div>

                  <!-- ACH body -->
                  <div v-if="payment === m.key && m.key === 'ach'" class="pm-body" @click.stop>
                    <p class="wire-intro">
                      ACH debit from your US bank account. Settlement in 3–5 business days. Credits
                      appear in your wallet once funds clear.
                    </p>
                    <div class="stripe-grid">
                      <div class="f-input">
                        <span class="f-label">Bank</span>
                        <input type="text" value="Chase · Personal Checking ••8821" readonly />
                        <span class="brand-mark-pill">VERIFIED</span>
                      </div>
                      <div class="col2">
                        <div class="f-input">
                          <span class="f-label">Routing</span>
                          <input type="text" value="021 000 021" autocomplete="off" />
                        </div>
                        <div class="f-input">
                          <span class="f-label">Account</span>
                          <input type="text" value="•••• •••• 8821" autocomplete="off" />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="field-label"><span>— Order summary</span></div>
              <div class="order-summary">
                <div class="os-row">
                  <span class="os-lbl">Pay</span>
                  <span class="os-val">${{ fmtAmount(amount) }} USD</span>
                </div>
                <div class="os-row">
                  <span class="os-lbl">Receive</span>
                  <span class="os-val">
                    {{ creditType === 'CASH' ? '$' + fmtUSD(creditsReceived, 2) : creditsReceived.toLocaleString('en-US') }}
                    {{ creditLabel }}
                  </span>
                </div>
                <div class="os-row">
                  <span class="os-lbl">Method</span>
                  <span class="os-val">{{ methodTiming }}</span>
                </div>
                <div class="os-row">
                  <span class="os-lbl">Fee</span>
                  <span class="os-val">$0.00</span>
                </div>
                <div class="os-divider" />
                <div class="os-row total">
                  <span class="os-total-lbl">Total</span>
                  <span class="os-val">${{ fmtAmount(amount) }} USD</span>
                </div>
              </div>

              <label class="terms-row" :class="{ checked: termsAccepted }">
                <input v-model="termsAccepted" type="checkbox" />
                <span class="check-box">
                  <svg v-if="termsAccepted" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="square">
                    <path d="M3 8.5l3.2 3.2L13 5" />
                  </svg>
                </span>
                <span class="terms-label">
                  I authorize Exascale to settle this order under the
                  <a href="#">Customer Agreement</a> and acknowledge that credit prices are quoted
                  live and may move between confirmation and settlement. Conversions are final.
                </span>
              </label>
            </template>

            <!-- =========================================
                 STEP 3 — Confirmation
                 ========================================= -->
            <template v-else>
              <div class="confirm-card">
                <div class="confirm-mark" aria-hidden="true">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="square">
                    <path d="M3 8.5l3.2 3.2L13 5" />
                  </svg>
                </div>
                <h2 class="confirm-title">Order placed.</h2>
                <p class="confirm-sub">
                  Your purchase has been queued for settlement. A receipt has been sent to
                  <strong>marcus.chen@frontier.lab</strong>.
                </p>

                <div class="confirm-summary">
                  <div class="cs-section">
                    <div class="cs-row">
                      <span class="cs-lbl">Order ID</span>
                      <span class="cs-val">{{ ORDER_ID }}</span>
                    </div>
                    <div class="cs-row">
                      <span class="cs-lbl">Placed</span>
                      <span class="cs-val">May 20, 2026 · 14:32:18 ET</span>
                    </div>
                    <div class="cs-row">
                      <span class="cs-lbl">Status</span>
                      <span class="cs-val warn">{{ statusLabel }}</span>
                    </div>
                  </div>
                  <div class="cs-section">
                    <div class="cs-row">
                      <span class="cs-lbl">Pay</span>
                      <span class="cs-val big">${{ fmtAmount(amount) }} <span class="dim sm">USD</span></span>
                    </div>
                    <div class="cs-row">
                      <span class="cs-lbl">Receive</span>
                      <span class="cs-val big">
                        {{ creditType === 'CASH' ? '$' + fmtUSD(creditsReceived, 2) : creditsReceived.toLocaleString('en-US') }}
                        <span class="dim sm">{{ creditLabel }}</span>
                      </span>
                    </div>
                    <div class="cs-row">
                      <span class="cs-lbl">Rate</span>
                      <span class="cs-val">
                        {{ creditType === 'CASH' ? '1.000000' : '$' + activeCreditPx.toFixed(6) }}
                        per credit · locked at print
                      </span>
                    </div>
                    <div class="cs-row">
                      <span class="cs-lbl">Method</span>
                      <span class="cs-val">{{ methodLabel }}</span>
                    </div>
                    <div class="cs-row total">
                      <span class="cs-total-lbl">Total</span>
                      <span class="cs-val big">${{ fmtAmount(amount) }} USD</span>
                    </div>
                  </div>
                </div>

                <div class="confirm-status">
                  <span class="ix">i</span>
                  <div class="txt">
                    {{ settlementText }}
                    <span class="meta">
                      Track this order at
                      <a href="#">exascale.com/orders/{{ ORDER_ID }}</a>
                    </span>
                  </div>
                </div>

                <div class="confirm-actions">
                  <button type="button" class="btn primary lg" @click="goWallet">
                    Go to wallet
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                      <path d="M3 8h10M9 4l4 4-4 4" />
                    </svg>
                  </button>
                  <button type="button" class="btn secondary lg" @click="goTrade">
                    Place first trade
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                      <path d="M3 8h10M9 4l4 4-4 4" />
                    </svg>
                  </button>
                  <button type="button" class="btn secondary lg" @click="reset">Done</button>
                </div>

                <div class="confirm-meta">
                  <span>Receipt ref · <strong>{{ ORDER_ID }}</strong> · cryptographic audit chain</span>
                  <a href="#">Download receipt (PDF) →</a>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- Card footer -->
        <div v-if="stepIdx < 2" class="card-footer">
          <span class="footer-hint">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
              <rect x="3.5" y="7" width="9" height="6.5" />
              <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
            </svg>
            <template v-if="stepIdx === 0">Live rate · refreshed every 5 s · TLS 1.3 settlement</template>
            <template v-else>Encrypted in transit · PCI-DSS Level 1</template>
          </span>
          <div class="footer-actions">
            <button v-if="stepIdx > 0" type="button" class="btn secondary" @click="back">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M13 8H3M7 4L3 8l4 4" />
              </svg>
              Back
            </button>
            <button type="button" class="btn primary" :disabled="!canAdvance" @click="next">
              {{ stepIdx === 1 ? 'Submit payment' : 'Continue' }}
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M3 8h10M9 4l4 4-4 4" />
              </svg>
            </button>
          </div>
        </div>
      </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.buy-shell {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  font-feature-settings: 'ss01';
}

/* ===== Top chrome ===== */
.topbar {
  height: 56px;
  background: var(--elevated);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 32px;
  justify-content: space-between;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 15px;
  letter-spacing: -0.005em;
  color: var(--text);
  text-decoration: none;
}
.brand-mark {
  width: 12px;
  height: 12px;
  background: var(--brand);
  display: inline-block;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 24px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--text-3);
}
.balance {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text-2);
}
.balance strong {
  color: var(--text);
  font-weight: 500;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}
.env-pill {
  padding: 4px 8px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--accent);
  background: rgba(74, 144, 226, 0.10);
  border: 1px solid rgba(74, 144, 226, 0.25);
  border-radius: var(--radius-sm);
  text-transform: uppercase;
  font-weight: 600;
}
.exit {
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 13px;
  letter-spacing: 0;
  text-decoration: none;
}
.exit:hover { color: var(--text); }

/* ===== Page ===== */
.page {
  max-width: 720px;
  margin: 0 auto;
  padding: 40px 24px 80px;
}
.page-header { margin-bottom: 32px; }

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
.eyebrow::before {
  content: '';
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
.page-subtitle {
  color: var(--text-2);
  font-size: 14px;
  margin: 0;
  max-width: 540px;
}

/* ===== KYC gate (real-money) ===== */
.env-pill.live { color: var(--pos); background: rgba(22, 163, 74, 0.10); border-color: rgba(22, 163, 74, 0.25); }
.sandbox-note {
  display: flex; align-items: center; gap: 10px;
  font-size: 12.5px; color: var(--text-2);
  background: var(--canvas); border: 1px solid var(--border); border-radius: var(--radius-sm);
  padding: 10px 14px; margin-bottom: 20px;
}
.sn-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); flex-shrink: 0; }
.kyc-gate {
  background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm);
  padding: 32px; max-width: 640px;
}
.kyc-eyebrow {
  font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.18em;
  text-transform: uppercase; color: var(--text-3); margin-bottom: 14px;
  display: inline-flex; align-items: center; gap: 8px;
}
.kyc-eyebrow::before { content: ''; width: 5px; height: 5px; background: var(--warn); display: inline-block; }
.kyc-title {
  font-family: var(--font-display); font-weight: 600; font-size: 24px; letter-spacing: -0.02em;
  line-height: 1.15; margin: 0 0 10px; color: var(--text);
}
.kyc-lede { color: var(--text-2); font-size: 14px; line-height: 1.55; margin: 0 0 22px; max-width: 560px; }
.kyc-lede strong { color: var(--text); font-weight: 600; }
.kyc-list { list-style: none; margin: 0 0 22px; padding: 0; border: 1px solid var(--border); border-radius: var(--radius-sm); }
.kyc-list li { display: flex; gap: 14px; align-items: flex-start; padding: 14px 18px; border-bottom: 1px solid var(--border); }
.kyc-list li:last-child { border-bottom: 0; }
.k-ix { font-family: var(--font-mono); font-size: 11px; font-weight: 600; color: var(--text-3); padding-top: 2px; font-variant-numeric: tabular-nums; }
.kyc-list strong { display: block; font-size: 14px; font-weight: 500; color: var(--text); margin-bottom: 2px; }
.kyc-list li span:last-child { font-size: 12.5px; color: var(--text-2); }
.kyc-actions { display: flex; gap: 8px; margin-bottom: 18px; }
.kyc-actions .btn { text-decoration: none; }
.kyc-foot { font-family: var(--font-mono); font-size: 11px; color: var(--text-3); letter-spacing: 0.04em; border-top: 1px dashed var(--border); padding-top: 14px; }

/* ===== Stepper ===== */
.steps {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 24px;
}
.step {
  position: relative;
  padding: 14px 18px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 0;
  border-right: 1px solid var(--border);
  background: transparent;
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition: background 120ms;
}
.step:last-child { border-right: 0; }
.step:hover { background: var(--sunken); }
.step.active { background: var(--elevated); }
.step.active::after {
  content: '';
  position: absolute;
  left: 0; right: 0; bottom: -1px;
  height: 2px;
  background: var(--text);
}
.step-num {
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  border: 1px solid var(--border-strong);
  color: var(--text-3);
}
.step.active .step-num { background: var(--text); color: var(--elevated); border-color: var(--text); }
.step.done .step-num { background: var(--text); color: var(--brand); border-color: var(--text); }
.check-svg { width: 11px; height: 11px; }
.step-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.step-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.14em;
  color: var(--text-3);
  text-transform: uppercase;
}
.step.active .step-label { color: var(--text); }
.step-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}

/* ===== Card ===== */
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 32px;
}
.card-head {
  padding: 24px 28px 18px;
  border-bottom: 1px solid rgba(14, 14, 14, 0.06);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
}
.card-ttl {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 20px;
  letter-spacing: -0.01em;
  margin: 6px 0 0;
  line-height: 1.25;
}
.card-lede {
  color: var(--text-2);
  font-size: 13.5px;
  margin: 4px 0 0;
}
.card-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
  white-space: nowrap;
  padding-top: 4px;
}
.card-meta strong { color: var(--text); font-weight: 500; }
.dim { color: var(--text-3); }

.card-body { padding: 24px 28px 24px; }

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 28px;
  border-top: 1px solid rgba(14, 14, 14, 0.06);
  background: var(--canvas);
}
.footer-hint {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: flex;
  align-items: center;
  gap: 8px;
}
.footer-hint svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  fill: none;
  stroke-width: 1.5;
}
.footer-actions { display: flex; gap: 8px; }

.step-content {
  animation: step-enter 200ms cubic-bezier(0, 0, 0.2, 1);
}
@keyframes step-enter {
  from { opacity: 0; transform: translateY(2px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== Field label ===== */
.field-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.field-label .req {
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: none;
  font-size: 11px;
  font-family: var(--font-sans);
}

/* ===== Amount input ===== */
.amount-wrap {
  display: flex;
  align-items: center;
  border: 1px solid var(--border-strong);
  background: var(--elevated);
  border-radius: var(--radius-sm);
  margin-bottom: 12px;
  transition: border-color 120ms, box-shadow 120ms;
}
.amount-wrap:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.amount-prefix {
  font-family: var(--font-mono);
  font-size: 32px;
  color: var(--text-3);
  padding: 0 12px 0 20px;
  font-variant-numeric: tabular-nums;
  font-weight: 500;
}
.amount-input {
  flex: 1;
  height: 72px;
  border: 0;
  outline: 0;
  background: transparent;
  font-family: var(--font-mono);
  font-size: 32px;
  font-weight: 500;
  letter-spacing: -0.01em;
  color: var(--text);
  padding: 0 20px 0 0;
  font-variant-numeric: tabular-nums;
  width: 100%;
}
.amount-input::placeholder { color: #B8B8B0; }
.amount-suffix {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.14em;
  color: var(--text-3);
  padding: 0 20px;
  font-weight: 600;
}

/* Quick amounts */
.qa-row {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 6px;
  margin-bottom: 24px;
}
.qa-btn {
  height: 32px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 500;
  color: var(--text-2);
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
  transition: background 120ms, border-color 120ms, color 120ms;
}
.qa-btn:hover { border-color: var(--border-strong); color: var(--text); }
.qa-btn.active {
  background: var(--text);
  border-color: var(--text);
  color: var(--elevated);
}

/* ===== Credit type cards ===== */
.credit-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 8px;
}
.credit-card {
  position: relative;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 14px;
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  color: var(--text);
  transition: background 120ms, border-color 120ms;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 100px;
}
.credit-card:hover { background: var(--canvas); }
.credit-card.active {
  border-color: var(--text);
  background: var(--elevated);
}
.credit-card.active::after {
  content: '';
  position: absolute;
  inset: 0;
  border: 1px solid var(--text);
  border-radius: var(--radius-sm);
  pointer-events: none;
}
.credit-card .ribbon {
  position: absolute;
  top: -1px;
  right: -1px;
  background: var(--brand);
  color: var(--text);
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.14em;
  padding: 2px 7px;
  text-transform: uppercase;
}
.credit-card .top-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.credit-card .symbol {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-family: var(--font-mono);
  font-weight: 600;
}
.credit-card .sym-ai_index { background: var(--brand); color: var(--text); }
.credit-card .sym-cash     { background: var(--sunken); color: var(--text); }
.credit-card .sym-sub      { background: rgba(74, 144, 226, 0.18); color: var(--accent); }
.credit-card .name {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: var(--text);
  line-height: 1.2;
}
.credit-card .desc {
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.4;
}
.credit-card .px {
  margin-top: auto;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  padding-top: 4px;
  border-top: 1px dashed var(--border);
}

.sub-select {
  width: 100%;
  height: 36px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  background: var(--elevated);
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  margin-top: 10px;
  outline: none;
}
.sub-select:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}

/* ===== Live preview ===== */
.preview {
  margin-top: 24px;
  border: 1px solid var(--border-strong);
  background: var(--elevated);
  border-radius: var(--radius-sm);
  position: relative;
  overflow: hidden;
}
.preview::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--brand);
}
.preview-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 18px;
  border-bottom: 1px solid rgba(14, 14, 14, 0.06);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
}
.preview-head .live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-2);
}
.preview-head .pulse {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse-ring 2.4s infinite;
}
@keyframes pulse-ring {
  0%   { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0.6); }
  70%  { box-shadow: 0 0 0 6px rgba(22, 163, 74, 0); }
  100% { box-shadow: 0 0 0 0 rgba(22, 163, 74, 0); }
}
.preview-body { padding: 18px 22px; }
.preview-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 8px 0;
}
.preview-row .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.preview-row .val {
  font-family: var(--font-mono);
  font-size: 18px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  color: var(--text);
  letter-spacing: -0.005em;
}
.preview-row .val.small { font-size: 13px; }
.preview-row .val .unit {
  font-size: 12px;
  color: var(--text-3);
  margin-left: 4px;
  font-weight: 400;
  letter-spacing: 0.04em;
}
.preview-row.major .val { font-size: 22px; }
.preview-divider {
  height: 1px;
  background: rgba(14, 14, 14, 0.06);
  margin: 8px 0;
}

/* ===== Buttons ===== */
.btn {
  height: 44px;
  padding: 0 20px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 120ms, border-color 120ms, color 120ms, transform 80ms;
}
.btn:active { transform: translateY(1px); }
.btn.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
}
.btn.primary:hover { background: #222; border-color: #222; }
.btn.primary:disabled {
  background: var(--sunken);
  color: #B8B8B0;
  border-color: var(--border);
  cursor: not-allowed;
  transform: none;
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
.btn svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }
.btn.lg { height: 48px; padding: 0 24px; font-size: 15px; }

/* ===== Payment method cards ===== */
.pm-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 24px;
}
.pm-card {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  overflow: hidden;
  transition: border-color 120ms;
}
.pm-card.active { border-color: var(--text); }
.pm-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
}
.pm-radio {
  width: 16px;
  height: 16px;
  border: 1.5px solid var(--border-strong);
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  position: relative;
}
.pm-card.active .pm-radio { border-color: var(--text); }
.pm-card.active .pm-radio::after {
  content: '';
  width: 8px;
  height: 8px;
  background: var(--text);
  border-radius: 50%;
}
.pm-info { flex: 1; min-width: 0; }
.pm-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}
.pm-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 2px;
}
.pm-icons { display: flex; gap: 6px; }
.pm-icons .ic {
  height: 22px;
  padding: 0 7px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
}
.pm-body {
  padding: 18px 18px 18px 48px;
  border-top: 1px solid rgba(14, 14, 14, 0.06);
  background: var(--canvas);
}

/* Stripe-ish forms */
.stripe-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}
.col2 { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

.f-input {
  display: flex;
  align-items: center;
  height: 38px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  gap: 8px;
}
.f-input:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.f-input input {
  flex: 1;
  border: 0;
  outline: 0;
  background: transparent;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
  letter-spacing: 0.02em;
  width: 100%;
}
.f-label {
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  flex-shrink: 0;
}
.brand-mark-pill {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--text-3);
}

/* Wire body */
.wire-intro {
  font-size: 13px;
  color: var(--text-2);
  margin: 0 0 10px;
}
.wire-table {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 4px 16px;
  font-size: 13px;
  margin: 4px 0 0;
}
.wire-table dt {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  padding: 6px 0;
}
.wire-table dd {
  margin: 0;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
  padding: 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.copy-btn {
  background: transparent;
  border: 0;
  font-family: var(--font-sans);
  font-size: 11px;
  color: var(--text-3);
  cursor: pointer;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  letter-spacing: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.copy-btn:hover { background: rgba(14, 14, 14, 0.06); color: var(--text); }
.copy-btn svg { width: 11px; height: 11px; }

.wire-ref {
  margin-top: 10px;
  padding: 10px 12px;
  background: rgba(217, 119, 6, 0.10);
  border: 1px solid rgba(217, 119, 6, 0.25);
  border-radius: var(--radius-sm);
  font-size: 12.5px;
  color: var(--text);
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.wire-ref .ref-icon {
  color: var(--warn);
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: 12px;
  margin-top: 1px;
}
.ref-intro { margin-bottom: 4px; }
.ref-code {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 14px;
  letter-spacing: 0.02em;
}

/* ===== Order summary ===== */
.order-summary {
  margin-top: 8px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 14px 18px;
}
.os-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 4px 0;
  font-size: 13px;
}
.os-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.os-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
}
.os-divider { height: 1px; background: var(--border); margin: 8px 0; }
.os-row.total .os-val { font-size: 16px; font-weight: 500; }
.os-total-lbl {
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  letter-spacing: 0;
}

/* ===== Terms ===== */
.terms-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: 18px 0 0;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  cursor: pointer;
  position: relative;
}
.terms-row.checked {
  background: rgba(200, 242, 92, 0.18);
  border-color: rgba(14, 14, 14, 0.30);
}
.terms-row input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}
.terms-row .check-box {
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  flex-shrink: 0;
  margin-top: 1px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.terms-row.checked .check-box {
  background: var(--text);
  border-color: var(--text);
  color: var(--brand);
}
.terms-row.checked .check-box svg { width: 10px; height: 10px; }
.terms-label {
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
}
.terms-label a {
  color: var(--text);
  text-decoration: underline;
  text-decoration-color: var(--text-3);
}

/* ===== Confirmation ===== */
.confirm-card {
  position: relative;
  background: var(--elevated);
  border-radius: var(--radius-sm);
  padding: 8px 0 0;
  text-align: left;
}
.confirm-mark {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--text);
  color: var(--brand);
  margin-bottom: 16px;
}
.confirm-mark svg { width: 18px; height: 18px; }
.confirm-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 26px;
  letter-spacing: -0.022em;
  line-height: 1.15;
  margin: 0 0 6px;
  color: var(--text);
}
.confirm-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0 0 24px;
  line-height: 1.5;
}
.confirm-sub strong {
  color: var(--text);
  font-weight: 500;
}

.confirm-summary {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 16px;
}
.cs-section {
  padding: 14px 18px;
  border-bottom: 1px solid rgba(14, 14, 14, 0.06);
}
.cs-section:last-child { border-bottom: 0; }
.cs-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 4px 0;
}
.cs-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.cs-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
}
.cs-val.big { font-size: 18px; font-weight: 500; }
.cs-val.warn { color: var(--warn); }
.cs-val .sm { font-size: 12px; }
.cs-row.total {
  padding-top: 8px;
  border-top: 1px solid var(--border);
  margin-top: 8px;
}
.cs-total-lbl {
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  letter-spacing: 0;
}

.confirm-status {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 14px 18px;
  margin-bottom: 20px;
}
.confirm-status .ix {
  color: var(--accent);
  font-family: var(--font-mono);
  font-weight: 700;
  margin-top: 1px;
  font-size: 14px;
}
.confirm-status .txt {
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
}
.confirm-status .meta {
  display: block;
  margin-top: 4px;
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.confirm-status a { color: var(--text); }

.confirm-actions {
  display: grid;
  grid-template-columns: 1fr 1fr 0.6fr;
  gap: 8px;
}

.confirm-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px dashed var(--border);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.confirm-meta strong { color: var(--text); font-weight: 500; }
.confirm-meta a { color: var(--text); text-decoration: none; }
.confirm-meta a:hover { text-decoration: underline; }

/* ===== Responsive ===== */
@media (max-width: 760px) {
  .qa-row { grid-template-columns: repeat(3, 1fr); }
  .credit-row { grid-template-columns: 1fr; }
  .confirm-actions { grid-template-columns: 1fr; }
}
</style>
