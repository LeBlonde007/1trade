<script setup lang="ts">
/**
 * /onboarding/verify — Email verification waiting (D2-A)
 *
 * Light, layout-less standalone screen. Sits between /signup and /onboarding/kyc.
 * The "magic link" in the real flow lands the user on this same URL with a
 * verified=1 query param; for the demo we expose an "I've verified" link.
 */
definePageMeta({ layout: false })
useHead({ title: 'Verify your email — 1Trade', htmlAttrs: { 'data-theme': 'light' } })

const route = useRoute()
const email = ref<string>(typeof route.query.email === 'string' ? route.query.email : 'jane.doe@walmart.com')

const COOLDOWN_SECS = 60
const remaining = ref<number>(47) // start mid-cooldown so demo shows the countdown state
let timer: ReturnType<typeof setInterval> | null = null

function startCooldown(secs: number) {
  remaining.value = secs
  if (timer) clearInterval(timer)
  timer = setInterval(() => {
    remaining.value = Math.max(0, remaining.value - 1)
    if (remaining.value <= 0 && timer) { clearInterval(timer); timer = null }
  }, 1000)
}

const verified = ref(false)
const verifyError = ref('')

// Where to go after the email is verified. If the deployment gated login on verification the user has
// NO session yet (signup issued none) → send them to /login to sign in. Otherwise (dev/immediate-login)
// they're already authed → continue onboarding to the persona welcome + tour (journey step 3). KYC is a
// later, trading-only gate handled after the tour, not here.
async function continueAfterVerify() {
  const { user, refresh } = useAuth()
  await refresh()
  await navigateTo(user.value ? '/onboarding/welcome' : '/login?verified=1')
}

async function onResend() {
  if (remaining.value > 0) return
  startCooldown(COOLDOWN_SECS)
  try {
    await $fetch('/api/auth/verify/resend', { method: 'POST' })
  } catch { /* cooldown still applies; surfaced on next attempt */ }
}

// A magic-link lands here with ?token=…; consume it, then continue onboarding.
onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) return
  try {
    await $fetch('/api/auth/verify', { method: 'POST', body: { token } })
    verified.value = true
    setTimeout(continueAfterVerify, 800)
  } catch {
    verifyError.value = 'This verification link is invalid or has already been used.'
  }
})

const cooldownLabel = computed(() => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return m + ':' + String(s).padStart(2, '0')
})

// Live AI Index ticker (Brownian motion, mean reversion to 1.0024)
const indexValue = ref<number>(1.0024)
const indexPct   = ref<number>(0.18)
const utcTime    = ref<string>('14:32:08 UTC')
let tickInterval: ReturnType<typeof setInterval> | null = null
let clockInterval: ReturnType<typeof setInterval> | null = null

function tickIndex() {
  let v = indexValue.value
  v = v + (1.0024 - v) * 0.04 + (Math.random() - 0.5) * 0.0006
  v = Math.max(0.992, Math.min(1.015, v))
  indexValue.value = v
  indexPct.value = (v - 1.0) * 100
}
function tickClock() {
  const d = new Date()
  const h = String(d.getUTCHours()).padStart(2, '0')
  const m = String(d.getUTCMinutes()).padStart(2, '0')
  const s = String(d.getUTCSeconds()).padStart(2, '0')
  utcTime.value = h + ':' + m + ':' + s + ' UTC'
}

const indexDisplay = computed(() => (indexValue.value * 1000).toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 }))
const indexDeltaStr = computed(() => {
  const arrow = indexPct.value >= 0 ? '▲' : '▼'
  return arrow + ' ' + Math.abs(indexPct.value).toFixed(2) + '% (24h)'
})
const indexDeltaPos = computed(() => indexPct.value >= 0)

onMounted(() => {
  startCooldown(remaining.value)
  tickInterval = setInterval(tickIndex, 2400)
  clockInterval = setInterval(tickClock, 1000)
  tickClock()
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (tickInterval) clearInterval(tickInterval)
  if (clockInterval) clearInterval(clockInterval)
})

const showChange = ref(false)
const draftEmail = ref('')
function openChange() {
  draftEmail.value = email.value
  showChange.value = true
}
function commitChange() {
  if (draftEmail.value && draftEmail.value.includes('@')) {
    email.value = draftEmail.value
    showChange.value = false
    startCooldown(COOLDOWN_SECS)
  }
}
</script>

<template>
  <div class="verify" data-theme="light">
    <header class="chrome">
      <NuxtLink to="/" class="brand"><span class="mark" />1Trade</NuxtLink>
      <a href="mailto:support@exascale.com" class="chrome-link">Need help? Contact support →</a>
    </header>

    <div class="center">
      <div class="card">
        <div class="icon-box" aria-hidden="true">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="4" width="20" height="16" />
            <path d="m22 6-10 7L2 6" />
          </svg>
        </div>

        <h2>Verify your email</h2>
        <p class="sent-to">We sent a verification link to <strong class="mono">{{ email }}</strong></p>
        <p class="hint">It can take up to a minute to arrive. Check spam or promotions if it doesn't appear. The link expires in 60 minutes.</p>

        <div class="actions">
          <button
            class="btn resend"
            type="button"
            :disabled="remaining > 0"
            @click="onResend"
          >
            <template v-if="remaining > 0">
              <span class="lbl">Resend in</span>
              <span class="countdown mono">{{ cooldownLabel }}</span>
            </template>
            <template v-else>
              Resend email
            </template>
          </button>

          <div class="below-actions">
            <button v-if="!showChange" type="button" class="link-btn" @click="openChange">Change email →</button>
            <button type="button" class="link-btn primary" @click="continueAfterVerify">I've verified my email →</button>
          </div>

          <div v-if="showChange" class="change-form">
            <input
              v-model="draftEmail"
              type="email"
              class="change-input mono"
              placeholder="new@email.com"
              @keyup.enter="commitChange"
              @keyup.esc="showChange = false"
            />
            <button type="button" class="btn sm" @click="commitChange">Send link</button>
            <button type="button" class="link-btn" @click="showChange = false">Cancel</button>
          </div>
        </div>
      </div>
    </div>

    <div class="ticker" aria-label="Live AI index">
      <div class="t-left">
        <span class="dot" />
        <span class="t-label">AI INDEX</span>
        <span class="sep">·</span>
        <span class="t-meta">$1 =</span>
        <span class="t-num mono">{{ indexDisplay }}</span>
        <span class="t-meta">credits</span>
        <span class="sep">·</span>
        <span class="t-delta mono" :class="{ pos: indexDeltaPos, neg: !indexDeltaPos }">{{ indexDeltaStr }}</span>
      </div>
      <div class="t-right">
        <span>MARKETS OPEN</span>
        <span class="sep">·</span>
        <span class="mono">{{ utcTime }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.verify {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.55;
  display: flex;
  flex-direction: column;
}
.verify .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }

/* Chrome */
.chrome {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28px 48px;
}
.brand {
  display: inline-flex; align-items: center; gap: 10px;
  font-family: var(--font-display); font-weight: 700; font-size: 18px;
  letter-spacing: -0.02em; color: var(--text); text-decoration: none;
}
.brand .mark { display: inline-block; width: 8.6px; height: 20px; flex: none; background: var(--brand); -webkit-mask: url('/brand/mark.svg') center / contain no-repeat; mask: url('/brand/mark.svg') center / contain no-repeat; }
.chrome-link {
  color: var(--text-2); text-decoration: none; font-size: 13px;
  transition: color 160ms ease;
}
.chrome-link:hover { color: var(--text); }

/* Centered card */
.center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.card {
  width: 100%;
  max-width: 460px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: 48px 40px;
  text-align: center;
  box-shadow: 0 1px 0 rgba(0,0,0,0.02);
}
.icon-box {
  width: 56px; height: 56px;
  margin: 0 auto 24px;
  background: rgba(0,0,0,0.035);
  border: 1px solid var(--border);
  border-radius: 2px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text);
}
.card h2 {
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 600;
  letter-spacing: -0.005em;
  margin: 0 0 10px;
}
.sent-to { color: var(--text-2); font-size: 14px; margin: 0 0 6px; }
.sent-to strong { color: var(--text); font-weight: 500; letter-spacing: -0.01em; }
.hint { color: var(--text-3); font-size: 12px; margin: 0 0 28px; line-height: 1.65; }

/* Actions */
.actions { display: flex; flex-direction: column; gap: 14px; }
.btn {
  width: 100%;
  height: 44px;
  background: var(--elevated);
  color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: 2px;
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}
.btn:hover:not(:disabled) {
  background: rgba(0,0,0,0.03);
  border-color: var(--text);
}
.btn:disabled { color: var(--text-2); cursor: default; }
.btn.sm { height: 36px; width: auto; padding: 0 14px; font-size: 13px; }

.countdown { color: var(--text-3); }

.below-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-top: 2px;
}
.link-btn {
  background: none; border: none; padding: 0;
  color: var(--text-2); font-size: 13px;
  font-family: var(--font-sans);
  text-decoration: underline;
  text-underline-offset: 3px;
  text-decoration-thickness: 1px;
  cursor: pointer;
}
.link-btn:hover { color: var(--text); }
.link-btn.primary { color: var(--text); font-weight: 600; text-decoration-color: var(--text); }

/* Change email inline form */
.change-form {
  display: flex; gap: 8px; align-items: center;
  padding: 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.change-input {
  flex: 1;
  height: 36px; padding: 0 10px;
  background: var(--elevated); color: var(--text);
  border: 1px solid var(--border-strong); border-radius: 2px;
  font-size: 13px; outline: none;
}
.change-input:focus { border-color: var(--accent); }

/* Live ticker */
.ticker {
  flex-shrink: 0;
  padding: 14px 48px;
  border-top: 1px solid var(--border);
  background: var(--elevated);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.t-left { display: inline-flex; align-items: center; gap: 10px; color: var(--text-2); }
.t-label { font-weight: 600; color: var(--text); letter-spacing: 0.1em; }
.t-num { color: var(--text); font-weight: 600; }
.t-meta { color: var(--text-2); }
.t-delta.pos { color: var(--pos); font-weight: 600; }
.t-delta.neg { color: var(--neg); font-weight: 600; }
.t-right { display: inline-flex; align-items: center; gap: 10px; color: var(--text-2); }
.sep { color: var(--text-3); }

.dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--pos);
  display: inline-block;
  animation: pulse 2.4s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%      { opacity: 0.55; transform: scale(0.85); }
}

@media (max-width: 600px) {
  .chrome, .ticker { padding-left: 24px; padding-right: 24px; }
  .card { padding: 36px 28px; }
}
</style>
