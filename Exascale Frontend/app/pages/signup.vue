<script setup lang="ts">
/**
 * /signup — Open an Exascale account
 *
 * Faithful port of uploads/Exascale Signup.html from the design bundle.
 * Split-shell layout: 60% form (light) / 40% reassurance panel (dark inverse).
 *
 * Behaviors:
 *   - 3 account-type cards (Trader / AI Co / Enterprise) — single-select
 *   - Password show/hide + live strength meter (s1–s4)
 *   - Right panel: rotating value prop (3 messages, 6s)
 *   - Live AI Index ticker (Brownian motion + mean reversion, 5s)
 *   - 30-day mini chart with draw-on animation
 *   - "Next print" countdown
 *
 * Layout set to false — this is a full-bleed page with its own chrome.
 */
import {
  CandlestickChart, Terminal, Building2,
  Eye, EyeOff, ArrowRight, ShieldCheck, BadgeCheck, Banknote,
} from 'lucide-vue-next'

definePageMeta({ layout: false })
useHead({ title: 'Open an account — Exascale', htmlAttrs: { 'data-theme': 'light' } })

// ─── Account type ───────────────────────────────────────────
type AccountKey = 'trader' | 'ai' | 'ent'
const acctType = ref<AccountKey>('trader')
const acctTypes: { key: AccountKey; icon: any; nm: string; l1: string; l2: string }[] = [
  { key: 'trader', icon: CandlestickChart, nm: 'Trader',     l1: 'Trade credits',  l2: 'Paper or real money' },
  { key: 'ai',     icon: Terminal,         nm: 'AI Company', l1: 'Use credits',     l2: 'Inference & compute' },
  { key: 'ent',    icon: Building2,        nm: 'Enterprise', l1: 'Multi-user SSO', l2: 'Procurement-friendly' },
]

// ─── Email + password ──────────────────────────────────────
const email = ref('')
const password = ref('')
const submitting = ref(false)
const signupError = ref('')
const passwordShown = ref(false)
const pwInputType = computed(() => passwordShown.value ? 'text' : 'password')

const scorePw = (v: string): number => {
  if (!v) return 0
  if (v.length < 6) return 1
  if (v.length < 10) return 2
  let s = 2
  if (/[A-Z]/.test(v)) s++
  if (/[0-9]/.test(v) && /[^A-Za-z0-9]/.test(v)) s++
  return Math.min(4, s)
}
const STR_LABELS = ['', 'Weak', 'Fair', 'Strong', 'Excellent']
const strength = computed(() => scorePw(password.value))
const strengthLabel = computed(() => strength.value ? STR_LABELS[strength.value].toUpperCase() : '')

// ─── Terms ─────────────────────────────────────────────────
const agreed = ref(true)

// ─── Right panel: rotating value prop ──────────────────────
const valueProps = [
  { plain: 'Start trading in ',         em: 'paper mode', tail: ' immediately — no funding needed.' },
  { plain: 'Real-time ',                em: 'AI Index',         tail: ', published daily.' },
  { plain: '',                          em: 'Free egress',      tail: ' on all compute. Always.' },
]
const vpIdx = ref(0)
const vpIn = ref(true)
let vpInterval: ReturnType<typeof setInterval> | null = null

const rotateProp = () => {
  vpIn.value = false
  setTimeout(() => {
    vpIdx.value = (vpIdx.value + 1) % valueProps.length
    requestAnimationFrame(() => { vpIn.value = true })
  }, 320)
}

// ─── Live AI Index ticker ──────────────────────────────────
const tkPrice = ref(0.001005)
const tkPriceText = computed(() =>
  `$${tkPrice.value.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 })}`,
)
const tkDirection = ref<'' | 'flash-up' | 'flash-down'>('')
const tkDeltaPct = ref(0.0018)
const tkDeltaNeg = computed(() => tkDeltaPct.value < 0)
const tkDeltaText = computed(() => {
  const pct = tkDeltaPct.value * 100
  const sign = pct >= 0 ? '▲' : '▼'
  return `${sign} ${Math.abs(pct).toFixed(2)}% (24h)`
})

let tickInterval: ReturnType<typeof setInterval> | null = null
const baseline24h = 0.001003

const doTick = () => {
  const drift = (0.001005 - tkPrice.value) * 0.05
  const noise = (Math.random() - 0.5) * 0.0000028
  const next = Math.max(0.000990, Math.min(0.001020, tkPrice.value + drift + noise))
  const up = next > tkPrice.value
  tkPrice.value = next
  tkDeltaPct.value = (tkPrice.value - baseline24h) / baseline24h
  tkDirection.value = up ? 'flash-up' : 'flash-down'
  setTimeout(() => { tkDirection.value = '' }, 700)
}

// ─── Countdown to next print ──────────────────────────────
const tkNext = ref('3h 24m')
let nextSec = 3 * 3600 + 24 * 60
let nextInterval: ReturnType<typeof setInterval> | null = null

// ─── 30-day mini chart path ───────────────────────────────
const miniLineD = ref('')
const miniFillD = ref('')

const buildMini = () => {
  const W = 320, H = 90
  const days = 30
  let v = 0.000982
  const pts: number[] = []
  for (let i = 0; i < days; i++) {
    const drift = 0.0000003
    const dd = Math.random() < 0.05 ? -(0.000008 + Math.random() * 0.000010) : 0
    v += drift + (Math.random() - 0.48) * 0.0000060 + dd
    v = Math.max(0.000960, Math.min(0.001020, v))
    pts.push(v)
  }
  pts[pts.length - 1] = 0.001005
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
  miniLineD.value = d
  miniFillD.value = `${d} L ${W} ${H} L 0 ${H} Z`
}

onMounted(() => {
  buildMini()
  vpInterval = setInterval(rotateProp, 6000)
  tickInterval = setInterval(doTick, 5000)
  nextInterval = setInterval(() => {
    nextSec = Math.max(0, nextSec - 60)
    const h = Math.floor(nextSec / 3600)
    const m = Math.floor((nextSec % 3600) / 60)
    tkNext.value = `${h}h ${String(m).padStart(2, '0')}m`
  }, 60_000)
})

onUnmounted(() => {
  if (vpInterval) clearInterval(vpInterval)
  if (tickInterval) clearInterval(tickInterval)
  if (nextInterval) clearInterval(nextInterval)
})

const togglePw = () => { passwordShown.value = !passwordShown.value }

/**
 * Submit handler — route based on the selected account type.
 * All flows pass through email verification first (D2), then their
 * type-specific onboarding.
 *   trader → /onboarding/verify → /onboarding/kyc → /onboarding/welcome → /trade
 *   ent    → /onboarding/verify → /onboarding/kyc → ...
 *   ai     → /onboarding/verify → /trade (placeholder)
 */
const onSubmit = async () => {
  if (!agreed.value || !email.value || !password.value) return
  signupError.value = ''
  submitting.value = true
  try {
    // Create the real account (tenant + admin user) via the BFF, then continue onboarding.
    await useAuth().signup(email.value, password.value)
    await navigateTo('/onboarding/verify?email=' + encodeURIComponent(email.value))
  } catch (err: unknown) {
    const ex = err as { statusCode?: number; data?: { message?: string } }
    signupError.value = ex?.statusCode === 409
      ? 'That email is already registered'
      : (ex?.data?.message || 'Could not create the account')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="page" data-theme="light">
    <!-- ════════════════════════════════════════════════════════
         LEFT — FORM
         ════════════════════════════════════════════════════════ -->
    <section class="form-side">
      <div class="topbar">
        <NuxtLink to="/" class="brand">
          <span class="mark" />Exascale
        </NuxtLink>
        <span class="step">
          Step <span class="step-num">1</span> <span class="total">of 2</span>
        </span>
      </div>

      <div class="form-card">
        <h1 class="title">Open an Exascale account</h1>
        <p class="subhead">5 minutes. No credit card required for paper trading.</p>

        <span class="lbl section-lbl">— Account type</span>
        <div class="acct-types">
          <button
            v-for="a in acctTypes"
            :key="a.key"
            type="button"
            class="acct"
            :class="{ active: acctType === a.key }"
            @click="acctType = a.key"
          >
            <span class="check" />
            <span class="ic-wrap"><component :is="a.icon" :size="16" :stroke-width="1.6" /></span>
            <span class="nm">{{ a.nm }}</span>
            <span class="l1">{{ a.l1 }}</span>
            <span class="l2">{{ a.l2 }}</span>
          </button>
        </div>

        <div class="field">
          <div class="row"><span class="lbl">— Work email</span></div>
          <input v-model="email" type="email" />
        </div>

        <div class="field">
          <div class="row">
            <span class="lbl">— Password</span>
            <span v-if="strengthLabel" class="hint" :class="`hint-s${strength}`">{{ strengthLabel }}</span>
          </div>
          <div class="input-with-toggle">
            <input v-model="password" :type="pwInputType" />
            <button
              type="button"
              class="toggle"
              :title="passwordShown ? 'Hide password' : 'Show password'"
              @click="togglePw"
            >
              <component :is="passwordShown ? EyeOff : Eye" :size="16" />
            </button>
          </div>
          <div class="strength" :class="strength ? `s${strength}` : ''">
            <div /><div /><div /><div />
          </div>
        </div>

        <div class="or">Or continue with</div>

        <div class="oauth-stack">
          <!-- OAuth brand SVGs use the providers' literal brand colors (Google/GitHub/Microsoft).
               Those are external brand identities, not design-system tokens — left as-is intentionally. -->
          <button type="button" class="oauth">
            <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true">
              <path fill="#4285F4" d="M17.64 9.2c0-.64-.06-1.25-.16-1.84H9v3.48h4.84a4.14 4.14 0 0 1-1.79 2.71v2.26h2.9c1.7-1.56 2.69-3.87 2.69-6.61z" />
              <path fill="#34A853" d="M9 18c2.43 0 4.47-.81 5.96-2.19l-2.9-2.26c-.81.54-1.84.86-3.06.86-2.35 0-4.34-1.59-5.05-3.72H.96v2.33A8.999 8.999 0 0 0 9 18z" />
              <path fill="#FBBC05" d="M3.95 10.69A5.41 5.41 0 0 1 3.66 9c0-.59.1-1.16.29-1.69V4.98H.96A8.997 8.997 0 0 0 0 9c0 1.45.35 2.83.96 4.02l2.99-2.33z" />
              <path fill="#EA4335" d="M9 3.58c1.32 0 2.51.45 3.44 1.35l2.58-2.58A8.998 8.998 0 0 0 9 0C5.48 0 2.44 2.02.96 4.98l2.99 2.33C4.66 5.18 6.65 3.58 9 3.58z" />
            </svg>
            Continue with Google
          </button>
          <button type="button" class="oauth">
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
              <path fill="currentColor" d="M12 .3a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1.1-.8.1-.8.1-.8 1.2.1 1.9 1.3 1.9 1.3 1.1 1.9 2.8 1.3 3.5 1 .1-.8.4-1.3.8-1.6-2.7-.3-5.5-1.3-5.5-6 0-1.3.5-2.4 1.3-3.2-.1-.3-.6-1.6.1-3.2 0 0 1-.3 3.3 1.2a11.4 11.4 0 0 1 6 0c2.3-1.6 3.3-1.2 3.3-1.2.7 1.6.2 2.9.1 3.2.8.8 1.3 1.9 1.3 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .3" />
            </svg>
            Continue with GitHub
          </button>
          <button type="button" class="oauth">
            <svg width="18" height="18" viewBox="0 0 21 21" aria-hidden="true">
              <rect x="1" y="1" width="9" height="9" fill="#F25022" />
              <rect x="11" y="1" width="9" height="9" fill="#7FBA00" />
              <rect x="1" y="11" width="9" height="9" fill="#00A4EF" />
              <rect x="11" y="11" width="9" height="9" fill="#FFB900" />
            </svg>
            Continue with Microsoft
          </button>
        </div>

        <label class="terms">
          <input v-model="agreed" type="checkbox" />
          <span>
            I agree to Exascale's
            <a href="#">Terms of Service</a> and
            <a href="#">Privacy Policy</a>. Real-money trading requires further verification at v1.5.
          </span>
        </label>

        <button type="button" class="submit" :disabled="!agreed" @click="onSubmit">
          Open Account
          <ArrowRight :size="16" />
        </button>

        <p class="signin-link">
          Already have an account?
          <NuxtLink to="/login">Sign in →</NuxtLink>
        </p>
      </div>
    </section>

    <!-- ════════════════════════════════════════════════════════
         RIGHT — REASSURANCE PANEL (dark)
         ════════════════════════════════════════════════════════ -->
    <section class="panel">
      <div class="panel-top">
        <a href="mailto:sales@exascale.com">Need help? Contact sales →</a>
      </div>

      <div class="value-prop-wrap">
        <div class="value-prop" :class="{ in: vpIn }">
          <template v-for="(seg, i) in [valueProps[vpIdx]]" :key="i">
            <span>{{ seg.plain }}</span><em>{{ seg.em }}</em><span>{{ seg.tail }}</span>
          </template>
        </div>
      </div>

      <div class="value-prop-dots">
        <div v-for="(_, i) in valueProps" :key="i" :class="{ active: i === vpIdx }" />
      </div>

      <div class="ticker">
        <div class="head">
          <span class="live-dot" />
          <span class="t-label">Exascale AI Index · Live</span>
          <span class="badge">PUBLISHED DAILY</span>
        </div>
        <div class="row-2">
          <span class="price" :class="tkDirection">{{ tkPriceText }}</span>
          <span class="delta" :class="{ neg: tkDeltaNeg }">{{ tkDeltaText }}</span>
        </div>
        <div class="meta">Last print 16:00 UTC · Next in <span>{{ tkNext }}</span></div>
      </div>

      <div class="mini-chart">
        <div class="mc-head">
          <span class="mc-title">— 30D index history</span>
          <span class="mc-stat">▲ +2.31% (30d)</span>
        </div>
        <svg viewBox="0 0 320 90" preserveAspectRatio="none">
          <path class="fill" :d="miniFillD" />
          <path class="line" :d="miniLineD" />
        </svg>
        <div class="ax">
          <span>30D AGO</span>
          <span>NOW</span>
        </div>
      </div>

      <div class="trust">
        <div class="item">
          <span class="ico"><ShieldCheck :size="18" :stroke-width="1.6" /></span>
          <span class="nm">SOC 2 Type I path</span>
          <span class="sub">AUDIT Q3 2026</span>
        </div>
        <div class="item">
          <span class="ico"><BadgeCheck :size="18" :stroke-width="1.6" /></span>
          <span class="nm">Audited externally</span>
          <span class="sub">INDEX METHODOLOGY</span>
        </div>
        <div class="item">
          <span class="ico"><Banknote :size="18" :stroke-width="1.6" /></span>
          <span class="nm">UBS Japan anchor</span>
          <span class="sub">FINANCIAL SERVICES</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* ─── Page shell ─── */
.page {
  display: grid;
  grid-template-columns: 60fr 40fr;
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 15px;
  line-height: 1.55;
  -webkit-font-smoothing: antialiased;
}

.lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.20em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
}

.section-lbl { display: block; margin-bottom: 10px; }

/* ─── LEFT — Form ─── */
.form-side {
  background: var(--canvas);
  padding: 40px 80px 64px;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 64px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 16px;
  letter-spacing: -0.01em;
  color: var(--text);
  text-decoration: none;
}

.brand .mark {
  display: inline-block;
  width: 14px;
  height: 14px;
  background: var(--brand);
}

.step {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
}

.step .step-num { color: var(--text); }
.step .total    { color: var(--text-3); }

.form-card {
  max-width: 540px;
  width: 100%;
  margin: 0 auto;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.title {
  font-family: var(--font-display);
  font-size: 36px;
  font-weight: 700;
  letter-spacing: -0.025em;
  line-height: 1.05;
  margin: 0 0 12px;
  color: var(--text);
}

.subhead {
  font-size: 16px;
  color: var(--text-2);
  margin: 0 0 36px;
}

/* ─── Account-type cards ─── */
.acct-types {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 32px;
}

.acct {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 18px 16px 16px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 10px;
  position: relative;
  text-align: left;
  transition: background var(--dur) var(--ease), border-color var(--dur) var(--ease);
  color: var(--text);
}

.acct:hover { border-color: var(--border-strong); }

.acct .ic-wrap {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  background: var(--sunken);
  border-radius: var(--radius-sm);
}

.acct .nm {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text);
}

.acct .l1 { font-size: 12.5px; color: var(--text-2); line-height: 1.4; }
.acct .l2 { font-size: 12.5px; color: var(--text-3); line-height: 1.4; }

.acct .check {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1.5px solid var(--border-strong);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}

.acct .check::after {
  content: '';
  width: 6px;
  height: 6px;
  background: var(--text);
  border-radius: 50%;
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease);
}

.acct.active {
  background: color-mix(in srgb, var(--brand) 10%, var(--elevated));
  border-color: var(--brand);
}

.acct.active .check {
  background: var(--brand);
  border-color: var(--brand);
}

.acct.active .check::after { opacity: 1; }

.acct.active .ic-wrap {
  color: var(--text);
  background: color-mix(in srgb, white 60%, transparent);
}

/* ─── Form fields ─── */
.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 18px;
}

.field .row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.field .hint {
  font-size: 12px;
  font-family: var(--font-mono);
  letter-spacing: 0.02em;
  color: var(--text);
}

.field input {
  height: 48px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 0 16px;
  font-family: var(--font-sans);
  font-size: 15px;
  color: var(--text);
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
  width: 100%;
}

.field input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 12%, transparent);
}

.input-with-toggle {
  position: relative;
}

.input-with-toggle .toggle {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: transparent;
  border: 0;
  color: var(--text-3);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.input-with-toggle .toggle:hover { color: var(--text); }

/* Strength meter */
.strength {
  display: flex;
  gap: 4px;
  margin-top: 8px;
}

.strength div {
  flex: 1;
  height: 3px;
  background: var(--border);
  border-radius: var(--radius-sm);
  transition: background var(--dur-fast) var(--ease);
}

.strength.s1 div:nth-child(-n+1) { background: var(--neg); }
.strength.s2 div:nth-child(-n+2) { background: var(--warn); }
.strength.s3 div:nth-child(-n+3) { background: var(--brand); }
.strength.s4 div:nth-child(-n+4) { background: var(--pos); }

/* ─── Or-divider ─── */
.or {
  display: flex;
  align-items: center;
  gap: 14px;
  margin: 24px 0 18px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}

.or::before,
.or::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border);
}

/* ─── OAuth buttons ─── */
.oauth-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 28px;
}

.oauth {
  height: 48px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  width: 100%;
  transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}

.oauth:hover {
  background: var(--sunken);
  border-color: var(--border-strong);
}

.oauth svg { display: block; }

/* ─── Terms ─── */
.terms {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 18px;
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
}

.terms input[type="checkbox"] {
  appearance: none;
  width: 16px;
  height: 16px;
  margin: 2px 0 0 0;
  border: 1.5px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  cursor: pointer;
  flex-shrink: 0;
  position: relative;
  transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}

.terms input[type="checkbox"]:checked {
  background: var(--text);
  border-color: var(--text);
}

.terms input[type="checkbox"]:checked::after {
  content: '✓';
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  color: var(--brand);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
}

.terms a {
  color: var(--text);
  text-decoration: underline;
  text-decoration-color: var(--border-strong);
  text-underline-offset: 3px;
}

.terms a:hover {
  color: var(--accent);
  text-decoration-color: var(--accent);
}

/* ─── Submit ─── */
.submit {
  height: 52px;
  width: 100%;
  background: var(--brand);
  color: var(--text);
  border: 0;
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
  transition: background var(--dur-fast) var(--ease);
}

.submit:hover:not(:disabled) { background: var(--brand-hov); }

.submit:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.signin-link {
  margin: 18px 0 0;
  text-align: center;
  font-size: 13.5px;
  color: var(--text-2);
}

.signin-link a {
  color: var(--text);
  font-weight: 500;
  text-decoration: none;
  border-bottom: 1px solid var(--text);
  padding-bottom: 1px;
}

.signin-link a:hover {
  color: var(--accent);
  border-bottom-color: var(--accent);
}


/* ═══════════════════════════════════════════════════════════
   RIGHT — Reassurance panel (dark)
   ═══════════════════════════════════════════════════════════ */
.panel {
  background: var(--inverse);
  color: var(--text-inverse);
  padding: 40px 56px 56px;
  display: flex;
  flex-direction: column;
  min-width: 0;
  position: relative;
  overflow: hidden;
}

/* Subtle grid backdrop */
.panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
  background-size: 48px 48px;
  pointer-events: none;
}

.panel > * { position: relative; z-index: 1; }

.panel-top {
  display: flex;
  justify-content: flex-end;
}

.panel-top a {
  color: color-mix(in srgb, var(--text-inverse) 50%, transparent);
  font-size: 13px;
  text-decoration: none;
  font-family: var(--font-sans);
  transition: color var(--dur) var(--ease);
}

.panel-top a:hover { color: var(--text-inverse); }

/* Value prop rotator */
.value-prop-wrap {
  margin-top: 56px;
  min-height: 200px;
  display: flex;
  align-items: flex-start;
}

.value-prop {
  font-family: var(--font-display);
  font-size: 40px;
  font-weight: 600;
  line-height: 1.08;
  letter-spacing: -0.025em;
  color: var(--text-inverse);
  opacity: 0;
  transform: translateY(8px);
  transition: opacity 600ms cubic-bezier(0.2, 0, 0, 1), transform 600ms cubic-bezier(0.2, 0, 0, 1);
}

.value-prop.in {
  opacity: 1;
  transform: translateY(0);
}

.value-prop em {
  font-style: normal;
  background: var(--brand);
  color: var(--text);
  padding: 0 0.08em;
}

.value-prop-dots {
  display: flex;
  gap: 6px;
  margin-top: 28px;
}

.value-prop-dots div {
  width: 18px;
  height: 2px;
  background: color-mix(in srgb, white 16%, transparent);
  transition: background var(--dur) var(--ease);
}

.value-prop-dots div.active { background: var(--brand); }


/* ─── Live ticker (right panel) ─── */
.ticker {
  margin-top: 48px;
  background: color-mix(in srgb, white 2.5%, transparent);
  border: 1px solid color-mix(in srgb, white 8%, transparent);
  border-radius: var(--radius-sm);
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ticker .head {
  display: flex;
  align-items: center;
  gap: 10px;
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
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: color-mix(in srgb, var(--text-inverse) 55%, transparent);
}

.ticker .badge {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--pos);
  background: var(--pos-soft);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.ticker .row-2 {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-top: 4px;
}

.ticker .price {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.025em;
  transition: color 600ms;
  color: var(--text-inverse);
}

.ticker .price.flash-up { color: var(--pos); }
.ticker .price.flash-down { color: var(--neg); }

.ticker .delta {
  font-family: var(--font-mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  color: var(--pos);
}

.ticker .delta.neg { color: var(--neg); }

.ticker .meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: color-mix(in srgb, var(--text-inverse) 40%, transparent);
  margin-top: 8px;
}

/* ─── 30D mini chart ─── */
.mini-chart {
  margin-top: 14px;
  background: color-mix(in srgb, white 2%, transparent);
  border: 1px solid color-mix(in srgb, white 8%, transparent);
  border-radius: var(--radius-sm);
  padding: 14px 16px 10px;
}

.mc-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8px;
}

.mc-title {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: color-mix(in srgb, var(--text-inverse) 50%, transparent);
}

.mc-stat {
  font-family: var(--font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--pos);
}

.mini-chart svg {
  width: 100%;
  height: 90px;
  display: block;
  overflow: visible;
}

.mini-chart svg .line {
  fill: none;
  stroke: var(--brand);
  stroke-width: 1.5;
  stroke-dasharray: 1400;
  stroke-dashoffset: 1400;
  animation: draw 1.4s cubic-bezier(0.2, 0, 0, 1) 0.3s forwards;
}

.mini-chart svg .fill {
  fill: var(--brand);
  fill-opacity: 0;
  animation: fadein 1.2s ease-out 0.8s forwards;
}

@keyframes draw  { to { stroke-dashoffset: 0; } }
@keyframes fadein{ to { fill-opacity: 0.10; } }

.ax {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: color-mix(in srgb, var(--text-inverse) 35%, transparent);
  letter-spacing: 0.06em;
  margin-top: 6px;
}

/* ─── Trust marks ─── */
.trust {
  margin-top: auto;
  padding-top: 40px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  border-top: 1px solid color-mix(in srgb, white 8%, transparent);
}

.trust .item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.trust .item .ico {
  width: 24px;
  height: 24px;
  color: var(--brand);
  margin-bottom: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
}

.trust .item .nm {
  font-family: var(--font-display);
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text-inverse);
  letter-spacing: -0.005em;
}

.trust .item .sub {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: color-mix(in srgb, var(--text-inverse) 50%, transparent);
}


/* ─── Responsive ─── */
@media (max-width: 1100px) {
  .form-side { padding: 32px 40px 64px; }
  .panel { padding: 32px 40px; }
}

@media (max-width: 800px) {
  .page { grid-template-columns: 1fr; }
  .panel { display: none; }
  .form-side { padding: 24px 24px 64px; }
  .topbar { margin-bottom: 32px; }
}
</style>
