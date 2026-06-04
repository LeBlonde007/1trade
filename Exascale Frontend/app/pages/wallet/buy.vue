<script setup lang="ts">
/**
 * /wallet/buy — Buy credits (live, Stripe-only).
 *
 * Wired to F06 billing: pick a credit type + amount → useBilling().checkout() → redirect to the
 * Stripe-hosted checkout; the webhook books the credits to the ledger on settlement. Stripe is the
 * only rail right now (MockStripe in dev); ACH/wire + JPY are M3, so they're not offered here.
 *
 * `amount` is the number of CREDITS to buy (fixed-point), per platform-core /v1/billing/checkout.
 * The USD figure is INDICATIVE (published reference prices) — Stripe shows the exact charge.
 * Real-money purchases pass the KYC gate first; sandbox + credits need none (preview with ?kyc=1).
 */
import { compact, full } from '~/utils/format'

definePageMeta({ layout: false, middleware: 'auth' })
useHead({ title: 'Buy credits — Exascale', htmlAttrs: { 'data-theme': 'dark' } })

const route = useRoute()
const { user, refresh } = useAuth()
const { checkout, loading } = useBilling()
const { balances, loadBalances } = useWallet()
const { submitting: kycSubmitting, submit: submitKyc } = useKyc()

// ── KYC gate (real-money only; sandbox + credits exempt). Server enforcement = F22. ─────────────
const realMoney = computed(() => user.value?.is_paper === false)
const kycStatus = computed(() => user.value?.kyc_status ?? 'unverified')
const kycVerified = computed(() => kycStatus.value === 'verified')
const kycPending = computed(() => kycStatus.value === 'pending')
const showKycGate = computed(() => (realMoney.value && !kycVerified.value) || route.query.kyc === '1')

// Minimal KYC submission — the IDV vendor handles documents; we capture entity basics. POSTing this
// flips the tenant to verified (dev auto-approve) or pending (prod review); the gate then re-reads
// the status off the refreshed session user.
const kycForm = reactive({ legal_name: '', country: 'US', entity_type: 'business' as 'individual' | 'business' })
const kycError = ref('')
const KYC_COUNTRIES = ['US', 'GB', 'CA', 'DE', 'FR', 'NL', 'CH', 'IE', 'JP', 'SG', 'AU', 'AE']

async function onSubmitKyc() {
  kycError.value = ''
  if (!kycForm.legal_name.trim() || kycForm.country.length !== 2) {
    kycError.value = 'Enter your legal name and country.'
    return
  }
  try {
    await submitKyc({ legal_name: kycForm.legal_name.trim(), country: kycForm.country, entity_type: kycForm.entity_type })
    await refresh() // pull the new kyc_status onto the session user so the gate updates in place
  } catch (e: unknown) {
    const ex = e as { data?: { message?: string } }
    kycError.value = ex?.data?.message || 'Could not submit verification. Please try again.'
  }
}

// ── Purchasable credit types + published per-credit reference price (indicative USD only). ───────
interface CreditDef { id: string; label: string; desc: string; refUsd: number }
const CREDITS: CreditDef[] = [
  { id: 'text',      label: 'Text',      desc: 'text generation', refUsd: 0.001210 },
  { id: 'ai_index',  label: 'AI Index',  desc: 'composite · redeemable across categories', refUsd: 0.001005 },
  { id: 'speech',    label: 'Speech',    desc: 'speech synthesis', refUsd: 0.001200 },
  { id: 'image',     label: 'Image',     desc: 'image generation', refUsd: 0.008000 },
  { id: 'video',     label: 'Video',     desc: 'video generation', refUsd: 0.250000 },
  { id: 'gpu_h100',  label: 'H100 GPU',  desc: 'GPU-hours · on-demand compute', refUsd: 2.990000 },
  { id: 'gpu_h200',  label: 'H200 GPU',  desc: 'GPU-hours · on-demand compute', refUsd: 3.490000 },
]
const QUICK_AMOUNTS = [10_000, 50_000, 100_000, 500_000, 1_000_000]
const GPU_QUICK = [1, 8, 24, 100, 500]

const creditType = ref('text')
const amount = ref(100_000)
const amountStr = ref('100,000')
const error = ref('')
const done = ref(false)
const settled = ref(false)

const active = computed<CreditDef>(() => CREDITS.find((c) => c.id === creditType.value) ?? CREDITS[0]!)
const isGpu = computed(() => creditType.value.startsWith('gpu_'))
const quickAmounts = computed(() => (isGpu.value ? GPU_QUICK : QUICK_AMOUNTS))
const indicativeUsd = computed(() => amount.value * active.value.refUsd)
const currentBalance = computed(() =>
  Number(balances.value.find((b) => b.credit_type === creditType.value)?.balance ?? 0),
)

function fmtInt(n: number): string { return n.toLocaleString('en-US') }
function fmtUsd(n: number): string { return n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }
function parseAmount(s: string): number { const n = parseFloat(String(s).replace(/[^0-9.]/g, '')); return Number.isNaN(n) ? 0 : Math.floor(n) }

function onAmountInput(e: Event) { amountStr.value = (e.target as HTMLInputElement).value; amount.value = parseAmount(amountStr.value) }
function onAmountBlur() { amountStr.value = fmtInt(amount.value) }
function setQuick(v: number) { amount.value = v; amountStr.value = fmtInt(v) }
function pickType(id: string) {
  creditType.value = id
  // GPU credits are whole GPU-hours; reset to a sensible default when switching kinds.
  if (id.startsWith('gpu_') && amount.value > 10_000) setQuick(8)
  else if (!id.startsWith('gpu_') && amount.value < 10_000) setQuick(100_000)
}

// ── Live Stripe checkout ────────────────────────────────────────────────────────────────────
async function onCheckout() {
  error.value = ''; done.value = false; settled.value = false
  if (amount.value <= 0) { error.value = 'Enter how many credits to buy.'; return }
  try {
    const res = await checkout(amount.value.toFixed(6), creditType.value, 'usd')
    if (res.settled) {
      // Sandbox: MockStripe booked the credits inline — refresh the balance and confirm in place.
      await loadBalances()
      settled.value = true
    } else {
      // Real Stripe: redirect to the hosted checkout; the webhook books the credits on settlement.
      window.open(res.checkout_url, '_blank')
      done.value = true
    }
  } catch (e: unknown) {
    const ex = e as { data?: { message?: string }; statusCode?: number }
    error.value = ex?.data?.message || 'Could not start checkout. Please try again.'
  }
}

onMounted(() => {
  const q = typeof route.query.credit === 'string' ? route.query.credit : ''
  const legacy: Record<string, string> = { ai: 'ai_index', sub: 'text', cash: 'text' }
  const want = legacy[q] ?? q
  if (want && CREDITS.some((c) => c.id === want)) pickType(want)
  loadBalances()
})
</script>

<template>
  <div class="buy-shell">
    <!-- Minimal focused chrome -->
    <header class="topbar">
      <NuxtLink to="/console" class="brand"><span class="brand-mark" />EXASCALE</NuxtLink>
      <div class="topbar-right">
        <span class="env-pill" :class="{ live: realMoney }">{{ realMoney ? 'LIVE' : 'PAPER' }}</span>
        <NuxtLink to="/wallet" class="exit">Back to wallet</NuxtLink>
      </div>
    </header>

    <div class="page" data-theme="dark">
      <div class="page-header">
        <div class="eyebrow">Buy credits · funding</div>
        <h1 class="page-title">Add credits to your account.</h1>
        <p class="page-subtitle">
          Pick a credit type and amount, then pay with card via Stripe. Credits are booked to your
          wallet on settlement. Sandbox purchases use test mode — no real charge.
        </p>
      </div>

      <!-- ===== KYC gate — a real-money purchase needs a one-time identity check (F22). Sandbox skips it. -->
      <div v-if="showKycGate" class="kyc-gate">
        <div class="kyc-eyebrow">Compliance · Identity verification</div>

        <!-- Submitted, awaiting a compliance decision (prod review path) -->
        <template v-if="kycPending">
          <h2 class="kyc-title">Verification in review.</h2>
          <p class="kyc-lede">
            Thanks — your identity submission is being reviewed. Real-money purchases unlock as soon as
            it's approved. Your <strong>sandbox</strong> account and all credit usage keep working now.
          </p>
          <div class="kyc-actions">
            <NuxtLink to="/console" class="btn secondary lg">Back to console</NuxtLink>
          </div>
        </template>

        <!-- Submit identity verification (unverified / rejected) -->
        <template v-else>
          <h2 class="kyc-title">Verify your identity to buy with real money.</h2>
          <p class="kyc-lede">
            Real-money purchases require a one-time identity check (KYC/AML) — credits are prepaid
            service units, so we verify the buyer first. Your <strong>sandbox</strong> account and all
            credit usage (inference, GPU compute) need none.
          </p>
          <form class="kyc-form" @submit.prevent="onSubmitKyc">
            <label class="kf-field">
              <span class="kf-label">Legal name</span>
              <input v-model="kycForm.legal_name" class="kf-input" type="text" placeholder="Registered legal entity or full name" autocomplete="organization" />
            </label>
            <div class="kf-row">
              <label class="kf-field">
                <span class="kf-label">Country</span>
                <select v-model="kycForm.country" class="kf-input">
                  <option v-for="c in KYC_COUNTRIES" :key="c" :value="c">{{ c }}</option>
                </select>
              </label>
              <label class="kf-field">
                <span class="kf-label">Account type</span>
                <select v-model="kycForm.entity_type" class="kf-input">
                  <option value="business">Business</option>
                  <option value="individual">Individual</option>
                </select>
              </label>
            </div>
            <p v-if="kycError" class="kf-error">{{ kycError }}</p>
            <div class="kyc-actions">
              <button type="submit" class="btn primary lg" :disabled="kycSubmitting">{{ kycSubmitting ? 'Submitting…' : 'Submit verification →' }}</button>
              <NuxtLink to="/console" class="btn secondary lg">Keep using sandbox</NuxtLink>
            </div>
          </form>
          <div class="kyc-foot">Minimal details only — documents are handled by our IDV vendor under our KYC/AML policy · SOC 2 controls · encrypted at rest.</div>
        </template>
      </div>

      <!-- ===== Live Stripe checkout ===== -->
      <div v-else class="card">
        <div v-if="!realMoney" class="sandbox-note">
          <span class="sn-dot" />
          Sandbox mode — test-mode Stripe, no real charge. Credits arrive in your wallet on settlement.
        </div>

        <!-- Credit type -->
        <div class="field-label">— Credit type</div>
        <div class="type-grid">
          <button
            v-for="c in CREDITS"
            :key="c.id"
            type="button"
            class="type-chip"
            :class="{ active: creditType === c.id }"
            @click="pickType(c.id)"
          >
            <span class="t-id mono">{{ c.label }}</span>
            <span class="t-desc">{{ c.desc }}</span>
            <span class="t-px mono">${{ c.refUsd.toFixed(6) }}<span class="per">/credit</span></span>
          </button>
        </div>

        <!-- Amount -->
        <div class="field-label">— Amount <span class="req">{{ isGpu ? 'GPU-hours' : 'credits' }}</span></div>
        <label class="amount-wrap">
          <input class="amount-input mono" type="text" inputmode="numeric" :value="amountStr" placeholder="0" autocomplete="off" @input="onAmountInput" @blur="onAmountBlur" />
          <span class="amount-suffix mono">{{ active.label.toUpperCase() }}</span>
        </label>
        <div class="qa-row" :style="{ gridTemplateColumns: `repeat(${quickAmounts.length}, 1fr)` }">
          <button v-for="v in quickAmounts" :key="v" type="button" class="qa-btn mono" :class="{ active: amount === v }" @click="setQuick(v)">{{ compact(v) }}</button>
        </div>

        <!-- Preview -->
        <div class="preview">
          <div class="preview-row major">
            <span class="lbl">You buy</span>
            <span class="val mono" :title="full(amount)">{{ compact(amount) }}<span class="unit">{{ active.label }} credits</span></span>
          </div>
          <div class="preview-row">
            <span class="lbl">Indicative cost</span>
            <span class="val small mono">${{ fmtUsd(indicativeUsd) }}<span class="unit">USD · exact at checkout</span></span>
          </div>
          <div class="preview-row">
            <span class="lbl">Current balance</span>
            <span class="val small mono" :title="full(currentBalance)">{{ compact(currentBalance) }}<span class="unit">{{ active.label }}</span></span>
          </div>
        </div>

        <div v-if="error" class="banner neg">{{ error }}</div>
        <div v-else-if="settled" class="banner ok">
          ✓ Purchased — {{ compact(amount) }} {{ active.label }} credits added to your wallet (test mode).
          <NuxtLink to="/wallet" class="bk">View wallet →</NuxtLink>
        </div>
        <div v-else-if="done" class="banner ok">
          Secure checkout opened in a new tab — complete payment there. Credits arrive in your wallet on settlement.
        </div>

        <button type="button" class="btn primary lg full" :disabled="loading || amount <= 0" @click="onCheckout">
          {{ loading ? 'Starting checkout…' : 'Continue to secure checkout →' }}
        </button>
        <div class="foot-note">
          Stripe-hosted · PCI-DSS · we never see your card. ACH / wire + JPY settlement arrive in a later release.
        </div>
      </div>
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
  -webkit-font-smoothing: antialiased;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }

/* Top chrome */
.topbar { height: 56px; background: var(--elevated); border-bottom: 1px solid var(--border); display: flex; align-items: center; padding: 0 32px; justify-content: space-between; }
.brand { display: flex; align-items: center; gap: 10px; font-family: var(--font-display); font-weight: 700; font-size: 15px; letter-spacing: -0.005em; color: var(--text); text-decoration: none; }
.brand-mark { width: 12px; height: 12px; background: var(--brand); display: inline-block; }
.topbar-right { display: flex; align-items: center; gap: 18px; }
.env-pill { padding: 4px 8px; font-family: var(--font-mono); font-size: 10px; letter-spacing: 0.18em; color: var(--accent); background: rgba(74,144,226,0.10); border: 1px solid rgba(74,144,226,0.25); border-radius: var(--radius-sm); text-transform: uppercase; font-weight: 600; }
.env-pill.live { color: var(--pos); background: rgba(22,163,74,0.10); border-color: rgba(22,163,74,0.25); }
.exit { color: var(--text-2); font-size: 13px; text-decoration: none; }
.exit:hover { color: var(--text); }

/* Page */
.page { max-width: 640px; margin: 0 auto; padding: 40px 24px 80px; }
.page-header { margin-bottom: 28px; }
.eyebrow { font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.18em; text-transform: uppercase; color: var(--text-3); display: inline-flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.eyebrow::before { content: ''; width: 5px; height: 5px; background: var(--brand); display: inline-block; }
.page-title { font-family: var(--font-display); font-weight: 600; font-size: 30px; letter-spacing: -0.022em; line-height: 1.1; margin: 0 0 8px; color: var(--text); }
.page-subtitle { color: var(--text-2); font-size: 14px; margin: 0; max-width: 560px; line-height: 1.55; }

/* Card */
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 24px 28px 28px; }
.sandbox-note { display: flex; align-items: center; gap: 10px; font-size: 12.5px; color: var(--text-2); background: var(--canvas); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 10px 14px; margin-bottom: 22px; }
.sn-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); flex-shrink: 0; }

.field-label { font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.16em; text-transform: uppercase; color: var(--text-2); display: flex; align-items: center; gap: 8px; margin: 0 0 10px; }
.field-label .req { color: var(--text-3); font-weight: 500; letter-spacing: 0.04em; text-transform: none; font-size: 11px; }

/* Credit type chips */
.type-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; margin-bottom: 22px; }
.type-chip { display: flex; flex-direction: column; gap: 3px; text-align: left; padding: 10px 12px; background: var(--elevated); border: 1px solid var(--border-strong); border-radius: var(--radius-sm); cursor: pointer; color: var(--text); transition: border-color 120ms, background 120ms; min-height: 74px; }
.type-chip:hover { background: var(--canvas); }
.type-chip.active { border-color: var(--text); }
.t-id { font-size: 13px; font-weight: 600; }
.t-desc { font-size: 10.5px; color: var(--text-3); line-height: 1.3; }
.t-px { margin-top: auto; font-size: 10.5px; color: var(--text-2); }
.t-px .per { color: var(--text-3); }

/* Amount */
.amount-wrap { display: flex; align-items: center; border: 1px solid var(--border-strong); background: var(--elevated); border-radius: var(--radius-sm); margin-bottom: 10px; }
.amount-wrap:focus-within { border-color: var(--accent); box-shadow: 0 0 0 2px rgba(74,144,226,0.18); }
.amount-input { flex: 1; height: 64px; border: 0; outline: 0; background: transparent; font-size: 30px; font-weight: 500; letter-spacing: -0.01em; color: var(--text); padding: 0 20px; }
.amount-suffix { font-size: 11px; letter-spacing: 0.14em; color: var(--text-3); padding: 0 20px; font-weight: 600; }
.qa-row { display: grid; gap: 6px; margin-bottom: 22px; }
.qa-btn { height: 32px; background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); cursor: pointer; font-size: 12px; font-weight: 500; color: var(--text-2); transition: background 120ms, border-color 120ms, color 120ms; }
.qa-btn:hover { border-color: var(--border-strong); color: var(--text); }
.qa-btn.active { background: var(--text); border-color: var(--text); color: var(--elevated); }

/* Preview */
.preview { border: 1px solid var(--border-strong); background: var(--elevated); border-radius: var(--radius-sm); padding: 14px 20px; margin-bottom: 18px; position: relative; overflow: hidden; }
.preview::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; background: var(--brand); }
.preview-row { display: flex; justify-content: space-between; align-items: baseline; padding: 7px 0; }
.preview-row .lbl { font-family: var(--font-mono); font-size: 10px; letter-spacing: 0.16em; text-transform: uppercase; color: var(--text-3); font-weight: 600; }
.preview-row .val { font-size: 18px; font-weight: 500; color: var(--text); }
.preview-row .val.small { font-size: 13px; }
.preview-row.major .val { font-size: 24px; }
.preview-row .val .unit { font-size: 11px; color: var(--text-3); margin-left: 6px; font-weight: 400; letter-spacing: 0.02em; }

/* Banners + button */
.banner { padding: 12px 14px; border-radius: var(--radius-sm); font-size: 13px; margin-bottom: 14px; }
.banner.neg { background: var(--neg-soft, rgba(239,68,68,0.12)); border: 1px solid var(--neg); color: var(--text); }
.banner.ok { background: var(--pos-soft, rgba(22,163,74,0.12)); border: 1px solid var(--pos); color: var(--text); }
.banner .bk { color: var(--text); text-decoration: underline; margin-left: 8px; font-weight: 500; }
.btn { height: 44px; padding: 0 20px; border-radius: var(--radius-sm); border: 1px solid transparent; font-size: 14px; font-weight: 500; cursor: pointer; display: inline-flex; align-items: center; justify-content: center; gap: 8px; text-decoration: none; transition: background 120ms, border-color 120ms; }
.btn.lg { height: 48px; font-size: 15px; }
.btn.full { width: 100%; }
.btn.primary { background: var(--text); color: var(--elevated); border-color: var(--text); }
.btn.primary:hover { background: #222; }
.btn.primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn.secondary { background: var(--elevated); color: var(--text); border-color: var(--border-strong); }
.btn.secondary:hover { background: var(--canvas); }
.foot-note { font-family: var(--font-mono); font-size: 11px; color: var(--text-3); letter-spacing: 0.03em; margin-top: 14px; text-align: center; }

/* KYC gate */
.kyc-gate { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 32px; }
.kyc-eyebrow { font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.18em; text-transform: uppercase; color: var(--text-3); margin-bottom: 14px; display: inline-flex; align-items: center; gap: 8px; }
.kyc-eyebrow::before { content: ''; width: 5px; height: 5px; background: var(--warn); display: inline-block; }
.kyc-title { font-family: var(--font-display); font-weight: 600; font-size: 24px; letter-spacing: -0.02em; line-height: 1.15; margin: 0 0 10px; color: var(--text); }
.kyc-lede { color: var(--text-2); font-size: 14px; line-height: 1.55; margin: 0 0 22px; }
.kyc-lede strong { color: var(--text); font-weight: 600; }
.kyc-list { list-style: none; margin: 0 0 22px; padding: 0; border: 1px solid var(--border); border-radius: var(--radius-sm); }
.kyc-list li { display: flex; gap: 14px; align-items: flex-start; padding: 14px 18px; border-bottom: 1px solid var(--border); }
.kyc-list li:last-child { border-bottom: 0; }
.k-ix { font-family: var(--font-mono); font-size: 11px; font-weight: 600; color: var(--text-3); padding-top: 2px; }
.kyc-list strong { display: block; font-size: 14px; font-weight: 500; color: var(--text); margin-bottom: 2px; }
.kyc-list li span:last-child { font-size: 12.5px; color: var(--text-2); }
.kyc-actions { display: flex; gap: 8px; margin-bottom: 18px; }
/* KYC submit form */
.kyc-form { margin: 0 0 4px; }
.kf-field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 14px; flex: 1; }
.kf-row { display: flex; gap: 14px; }
.kf-label { font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3); }
.kf-input { background: var(--canvas); border: 1px solid var(--border-strong); border-radius: var(--radius-sm); color: var(--text); padding: 0 12px; height: 40px; font-size: 14px; font-family: var(--font-sans); outline: none; width: 100%; }
.kf-input:focus { border-color: var(--brand); }
.kf-error { color: var(--neg); font-size: 12.5px; margin: 0 0 12px; }
.kyc-foot { font-family: var(--font-mono); font-size: 11px; color: var(--text-3); letter-spacing: 0.04em; border-top: 1px dashed var(--border); padding-top: 14px; }

@media (max-width: 640px) {
  .type-grid { grid-template-columns: repeat(2, 1fr); }
  .kyc-actions { flex-direction: column; }
}
</style>
