<script setup lang="ts">
/**
 * /login — Sign in to Exascale (split screen, light left + dark right).
 * 2FA challenge state replaces the form when credentials advance.
 */

definePageMeta({ layout: false })
useHead({ title: 'Sign in — Exascale', htmlAttrs: { 'data-theme': 'light' } })

const step = ref<'creds' | '2fa'>('creds')
const email = ref('')
const password = ref('')
const authError = ref('')
const submitting = ref(false)
// Shown after the user confirms their email and is redirected here (verify.vue → /login?verified=1).
const justVerified = computed(() => useRoute().query.verified === '1')
const showPw = ref(false)
const code = ref('428')
const rememberDevice = ref(false)
const codeInputs = ref<Array<HTMLInputElement | null>>([])

// Real login via the BFF. (2FA is an M4 step; not reached until the backend supports it.)
async function onCreds(e: Event) {
  e.preventDefault()
  authError.value = ''
  if (!email.value || !password.value) return
  submitting.value = true
  try {
    await useAuth().login(email.value, password.value)
    await navigateTo('/console')
  } catch (err: unknown) {
    const ex = err as { data?: { message?: string } }
    authError.value = ex?.data?.message || 'Invalid email or password'
  } finally {
    submitting.value = false
  }
}

function setCodeDigit(i: number, raw: string) {
  const digit = raw.replace(/[^0-9]/g, '').slice(-1)
  const arr = code.value.split('')
  while (arr.length < 6) arr.push('')
  arr[i] = digit
  code.value = arr.join('').slice(0, 6)
  if (digit && i < 5) codeInputs.value[i + 1]?.focus()
}
function onCodeKey(i: number, e: KeyboardEvent) {
  if (e.key === 'Backspace' && !code.value[i] && i > 0) codeInputs.value[i - 1]?.focus()
  if (e.key === 'ArrowLeft' && i > 0) codeInputs.value[i - 1]?.focus()
  if (e.key === 'ArrowRight' && i < 5) codeInputs.value[i + 1]?.focus()
}
function onPasteCode(e: ClipboardEvent) {
  e.preventDefault()
  const txt = (e.clipboardData?.getData('text') ?? '').replace(/[^0-9]/g, '').slice(0, 6)
  if (txt) {
    code.value = txt
    codeInputs.value[Math.min(txt.length, 5)]?.focus()
  }
}

// =====================================================
// Right-pane live ticker
// =====================================================
interface MarketRow {
  sym: string
  px: number
  dec: number
  chg: number
  dir: 'up' | 'down'
  flash: '' | 'tick-flash-up' | 'tick-flash-dn'
}
const tickerRows = ref<MarketRow[]>([
  { sym: 'AI-INDEX',    px: 0.001005, dec: 6, chg:  0.41, dir: 'up',   flash: '' },
  { sym: 'H100-USD',    px: 2.9912,   dec: 4, chg:  0.18, dir: 'up',   flash: '' },
  { sym: 'H200-USD',    px: 3.8450,   dec: 4, chg: -0.27, dir: 'down', flash: '' },
  { sym: 'TEXT-INDEX',  px: 0.001210, dec: 6, chg:  1.84, dir: 'up',   flash: '' },
  { sym: 'IMAGE-INDEX', px: 0.007980, dec: 6, chg: -0.62, dir: 'down', flash: '' },
  { sym: 'VIDEO-INDEX', px: 0.249820, dec: 6, chg:  0.14, dir: 'up',   flash: '' },
])

function fmtPx(n: number, dec = 6) {
  return n.toLocaleString('en-US', { minimumFractionDigits: dec, maximumFractionDigits: dec })
}
function sparkPoints(dir: 'up' | 'down') {
  return dir === 'up'
    ? '0,18 8,16 16,17 24,14 32,15 40,12 48,11 56,9 64,10 72,7 80,6 88,4'
    : '0,4 8,6 16,5 24,8 32,9 40,7 48,11 56,12 64,10 72,14 80,15 88,17'
}

const clock = ref('14:32:41 ET')
function tickClock() {
  const start = new Date('2026-05-19T14:32:41')
  const now = new Date(start.getTime() + (Date.now() % 60_000))
  clock.value = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')} ET`
}

let tickerInterval: ReturnType<typeof setInterval> | null = null
let clockInterval:  ReturnType<typeof setInterval> | null = null

onMounted(() => {
  tickClock()
  clockInterval = setInterval(tickClock, 1000)
  tickerInterval = setInterval(() => {
    const which = Math.floor(Math.random() * tickerRows.value.length)
    tickerRows.value = tickerRows.value.map((m, i) => {
      if (i !== which) return { ...m, flash: '' }
      const delta = (Math.random() - 0.48) * m.px * 0.0008
      const newPx = Math.max(0, m.px + delta)
      const newChg = m.chg + delta / m.px * 100
      const dir: 'up' | 'down' = delta >= 0 ? 'up' : 'down'
      return {
        ...m,
        px: newPx,
        chg: newChg,
        dir,
        flash: delta >= 0 ? 'tick-flash-up' : 'tick-flash-dn',
      }
    })
  }, 1200)
})
onBeforeUnmount(() => {
  if (tickerInterval) clearInterval(tickerInterval)
  if (clockInterval)  clearInterval(clockInterval)
})
</script>

<template>
  <div class="login-screen">
    <!-- ============ LEFT: form ============ -->
    <section class="lf-pane">
      <div class="lf-top">
        <NuxtLink class="lf-brand" to="/">
          <span class="brand-mark" />
          EXASCALE
        </NuxtLink>
        <span class="lf-status">
          <span class="status-dot" />
          All systems operational
        </span>
      </div>

      <div class="lf-body">
        <!-- ===== Sign-in ===== -->
        <template v-if="step === 'creds'">
          <div class="lf-eyebrow">Account · sign in</div>
          <h1 class="lf-title">Sign in</h1>
          <p class="lf-sub">Access your paper or real-money account on the Exascale venue.</p>

          <p v-if="justVerified" class="form-ok" role="status">Email verified — sign in to continue.</p>
          <p v-if="authError" class="form-error" role="alert">{{ authError }}</p>

          <form @submit="onCreds">
            <div class="field">
              <div class="field-label"><span>— Work email</span></div>
              <input
                v-model="email"
                class="input"
                type="email"
                placeholder="you@firm.com"
                autocomplete="username"
              />
            </div>

            <div class="field">
              <div class="field-label">
                <span>— Password</span>
                <a href="#">Forgot password?</a>
              </div>
              <div class="field-pw">
                <input
                  v-model="password"
                  class="input mono-input"
                  :type="showPw ? 'text' : 'password'"
                  placeholder="••••••••"
                  autocomplete="current-password"
                />
                <button class="reveal" type="button" @click="showPw = !showPw">
                  <svg v-if="showPw" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M2 2l12 12M6.2 6.2a2.4 2.4 0 003.6 3.6M4.5 4.7C2.6 6 1.5 8 1.5 8s2.5 4.5 6.5 4.5c1.2 0 2.3-.3 3.2-.8M9.5 3.6A6.6 6.6 0 008 3.5C4 3.5 1.5 8 1.5 8c.6 1 1.4 1.9 2.2 2.6m3.2-7C12.5 3.5 14.5 8 14.5 8c-.3.5-.7 1.1-1.2 1.6" />
                  </svg>
                  <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M1.5 8S4 3.5 8 3.5 14.5 8 14.5 8 12 12.5 8 12.5 1.5 8 1.5 8z" />
                    <circle cx="8" cy="8" r="2.2" />
                  </svg>
                  <span>{{ showPw ? 'Hide' : 'Show' }}</span>
                </button>
              </div>
            </div>

            <button class="btn" type="submit" :disabled="submitting">
              <template v-if="submitting">
                <span class="btn-spinner" aria-hidden="true" />
                Signing in…
              </template>
              <template v-else>
                Sign in
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                  <path d="M3 8h10M9 4l4 4-4 4" />
                </svg>
              </template>
            </button>
          </form>

          <div class="divider-or"><span>OR CONTINUE WITH</span></div>

          <div class="oauth-row">
            <button class="oauth-btn" type="button">
              <svg viewBox="0 0 18 18" width="14" height="14">
                <path d="M17.64 9.2c0-.64-.06-1.25-.16-1.84H9v3.48h4.84c-.21 1.13-.84 2.08-1.79 2.72v2.26h2.9c1.7-1.56 2.69-3.87 2.69-6.62z" fill="#4285F4" />
                <path d="M9 18c2.43 0 4.47-.8 5.96-2.18l-2.9-2.26c-.81.54-1.83.86-3.06.86-2.35 0-4.34-1.59-5.05-3.72H.96v2.34A9 9 0 0 0 9 18z" fill="#34A853" />
                <path d="M3.95 10.7A5.4 5.4 0 0 1 3.66 9c0-.59.1-1.16.29-1.7V4.96H.96A9 9 0 0 0 0 9c0 1.45.35 2.83.96 4.04l2.99-2.34z" fill="#FBBC05" />
                <path d="M9 3.58c1.32 0 2.51.46 3.44 1.35l2.58-2.58A9 9 0 0 0 9 0 9 9 0 0 0 .96 4.96L3.95 7.3C4.66 5.17 6.65 3.58 9 3.58z" fill="#EA4335" />
              </svg>
              <span class="name">Google</span>
            </button>
            <button class="oauth-btn" type="button">
              <svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.69-.01-1.36-2.22.48-2.69-1.07-2.69-1.07-.36-.92-.89-1.17-.89-1.17-.73-.5.06-.49.06-.49.81.06 1.23.83 1.23.83.72 1.22 1.87.87 2.33.66.07-.52.28-.87.5-1.07-1.77-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.01.08-2.12 0 0 .67-.21 2.2.82a7.6 7.6 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.11.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38C13.71 14.53 16 11.54 16 8c0-4.42-3.58-8-8-8z" />
              </svg>
              <span class="name">GitHub</span>
            </button>
            <button class="oauth-btn" type="button">
              <svg viewBox="0 0 16 16" width="14" height="14">
                <rect x="1" y="1" width="6.5" height="6.5" fill="#F25022" />
                <rect x="8.5" y="1" width="6.5" height="6.5" fill="#7FBA00" />
                <rect x="1" y="8.5" width="6.5" height="6.5" fill="#00A4EF" />
                <rect x="8.5" y="8.5" width="6.5" height="6.5" fill="#FFB900" />
              </svg>
              <span class="name">Microsoft</span>
            </button>
          </div>

          <div class="lf-meta">
            <NuxtLink to="/signup" class="lf-link primary">New to Exascale? Open an account →</NuxtLink>
            <a href="#" class="lf-link">Sign in with SSO (firm accounts)</a>
          </div>
        </template>

        <!-- ===== 2FA ===== -->
        <template v-else>
          <div class="lf-eyebrow">Account · two-factor authentication</div>
          <h1 class="lf-title">Verify it's you</h1>
          <p class="lf-sub">Enter the 6-digit code from your authenticator app to finish signing in.</p>

          <div class="email-pill">
            <span class="email-avatar">MC</span>
            <span class="email-em">{{ email }}</span>
            <span class="email-switch" @click="step = 'creds'">Not you?</span>
          </div>

          <div class="field">
            <div class="field-label">
              <span>— Authenticator code</span>
              <a href="#">Use a backup code instead →</a>
            </div>
            <div class="code-input" @paste="onPasteCode">
              <input
                v-for="i in 6"
                :key="i"
                :ref="el => { codeInputs[i - 1] = el }"
                class="code-cell"
                :class="{ filled: !!code[i - 1] }"
                maxlength="1"
                inputmode="numeric"
                :autocomplete="i === 1 ? 'one-time-code' : 'off'"
                :value="code[i - 1] || ''"
                :aria-label="`Digit ${i}`"
                @input="e => setCodeDigit(i - 1, e.target.value)"
                @keydown="e => onCodeKey(i - 1, e)"
              />
            </div>
            <div class="code-meta">
              <span class="resend">Didn't get a code? Re-sync authenticator →</span>
              <span class="timer">Code refreshes in 00:18</span>
            </div>
          </div>

          <label class="remember-row">
            <input v-model="rememberDevice" type="checkbox" />
            Remember this device for 30 days
          </label>

          <button class="btn" type="button" @click="navigateTo('/trade')">
            Verify and continue
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </button>

          <div class="lf-meta">
            <a href="#" class="lf-link" @click.prevent="step = 'creds'">← Back to sign in</a>
            <a href="#" class="lf-link">Lost your authenticator? Contact support →</a>
          </div>
        </template>
      </div>

      <footer class="lf-foot">
        <div class="legal">
          By signing in you agree to Exascale's <a href="#">Customer Agreement</a> and <a href="#">Risk Disclosure</a>.
          Paper accounts carry no monetary value.
        </div>
        <div class="lang">English (US)</div>
      </footer>
    </section>

    <!-- ============ RIGHT: dark reassurance ============ -->
    <aside class="rp-pane">
      <div class="rp-top">
        <span class="rp-eyebrow">
          <span>— EXASCALE</span>
          <span class="rp-live"><span class="pulse" />Live · NYSE {{ clock.slice(0, 5) }} ET</span>
        </span>
        <span class="rp-version">venue v2.4.1 · TLS 1.3 · region us-east-1</span>
      </div>

      <h2 class="rp-headline">
        The <em>commodity market</em><br />for AI compute.
      </h2>
      <p class="rp-lede">
        AI credits and GPU credits trade on a single venue. Daily reference index,
        deep order book, owned underlying — settlement in cash or physical compute.
      </p>

      <!-- Live ticker -->
      <div class="terminal">
        <div class="terminal-head">
          <span>— Live · top markets</span>
          <span class="terminal-meta">
            <span class="clock">{{ clock }}</span>
            <span class="dots"><span /><span /><span /></span>
          </span>
        </div>
        <table class="ticker-table">
          <thead>
            <tr>
              <th>Market</th>
              <th class="num-h">Last (USD)</th>
              <th class="num-h">24h Δ</th>
              <th class="num-h">Sparkline · 1H</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in tickerRows" :key="m.sym" :class="m.flash">
              <td class="sym">{{ m.sym }}</td>
              <td class="num">{{ fmtPx(m.px, m.dec) }}</td>
              <td class="num" :class="m.chg >= 0 ? 'pos' : 'neg'">
                <span class="chg">
                  <span class="arr">{{ m.chg >= 0 ? '▲' : '▼' }}</span>{{ Math.abs(m.chg).toFixed(2) }}%
                </span>
              </td>
              <td class="num">
                <svg class="sparkline" :class="m.dir" viewBox="0 0 92 22" preserveAspectRatio="none">
                  <polyline :points="sparkPoints(m.dir)" fill="none" stroke-width="1.2" stroke-linecap="square" stroke-linejoin="miter" />
                </svg>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Mini index card -->
      <div class="idx-card">
        <div class="idx-cell">
          <div class="lbl">24h volume</div>
          <div class="val">$12.4M</div>
          <div class="sub">across 6 markets</div>
        </div>
        <div class="idx-cell">
          <div class="lbl">AI-INDEX</div>
          <div class="val">1.0024</div>
          <div class="sub pos">▲ +0.18% · 24h</div>
        </div>
        <div class="idx-cell">
          <div class="lbl">Typical spread</div>
          <div class="val">0.10%</div>
          <div class="sub">10 bps · L1</div>
        </div>
        <div class="idx-cell">
          <div class="lbl">Settlement</div>
          <div class="val">T+0</div>
          <div class="sub">cash · physical</div>
        </div>
      </div>

      <!-- Footer citation -->
      <div class="rp-foot">
        <div class="rp-quote">
          “The first venue where a CFO and a GPU buyer hedge on the same screen.
          We use it for the same reason we use CME — because the underlying is real.”
          <span class="rp-quote-name"><strong>R. Sato</strong> · Treasury · Frontier Lab</span>
        </div>
        <div class="rp-links">
          <a href="#">Market data</a>
          <a href="#">Documentation</a>
          <a href="#">Status →</a>
        </div>
      </div>
    </aside>
  </div>
</template>

<style scoped>
/* ============================================================
   The login screen uses both light tokens (left pane) and inverse
   tokens (right pane). We pin the right pane's colors as locals so
   it stays dark regardless of the visiting theme.
   ============================================================ */
.login-screen {
  width: 100vw;
  min-height: 100vh;
  display: grid;
  grid-template-columns: 560px 1fr;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  font-feature-settings: 'ss01';
  -webkit-font-smoothing: antialiased;
  overflow: hidden;
}

.mono-input { font-family: var(--font-mono); font-size: 13.5px; letter-spacing: 0.04em; }

/* ============================================================
   LEFT — form
   ============================================================ */
.lf-pane {
  background: var(--elevated);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 32px 56px;
  position: relative;
  min-height: 100vh;
}
.lf-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.lf-brand {
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
  width: 14px;
  height: 14px;
  background: var(--brand);
  display: inline-block;
}
.lf-status {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.status-dot {
  width: 6px;
  height: 6px;
  background: var(--pos);
  border-radius: 50%;
}

.lf-body {
  margin: auto 0;
  width: 100%;
  max-width: 380px;
}

.lf-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.lf-eyebrow::before {
  content: '';
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.lf-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 32px;
  letter-spacing: -0.02em;
  line-height: 1.1;
  margin: 0 0 8px;
}
.lf-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0 0 28px;
  line-height: 1.5;
}

/* Positive note after email verification — sharp, mono, positive semantic. */
.form-ok {
  margin: 0 0 18px;
  padding: 10px 12px;
  border: 1px solid var(--pos);
  border-radius: 2px;
  color: var(--pos);
  font-size: 13px;
  font-family: var(--font-mono);
}
.form-error {
  margin: 0 0 18px;
  padding: 10px 12px;
  border: 1px solid var(--neg);
  border-radius: 2px;
  color: var(--neg);
  font-size: 13px;
  font-family: var(--font-mono);
}

.field { margin-bottom: 16px; position: relative; }
.field-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  margin-bottom: 6px;
}
.field-label a {
  font-family: var(--font-sans);
  font-size: 12px;
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-2);
  text-decoration: none;
  font-weight: 500;
}
.field-label a:hover { color: var(--text); text-decoration: underline; }

.input {
  width: 100%;
  height: 44px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 14px;
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text);
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
  font-feature-settings: 'ss01';
}
.input::placeholder { color: #B8B8B0; }
.input:hover { border-color: rgba(14, 14, 14, 0.30); }
.input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}

.field-pw { position: relative; }
.field-pw .input { padding-right: 70px; }
.field-pw .reveal {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  background: transparent;
  border: 0;
  padding: 6px 8px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-2);
  cursor: pointer;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.field-pw .reveal:hover { color: var(--text); background: var(--sunken); }
.field-pw .reveal svg { width: 12px; height: 12px; stroke: currentColor; fill: none; stroke-width: 1.5; }

.btn {
  height: 44px;
  width: 100%;
  border-radius: var(--radius-sm);
  border: 1px solid var(--text);
  background: var(--text);
  color: var(--elevated);
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  transition: background 120ms, border-color 120ms;
}
.btn:hover { background: #222; border-color: #222; }
.btn:disabled { opacity: 0.6; cursor: default; }
.btn:disabled:hover { background: var(--text); border-color: var(--text); }
.btn svg { width: 14px; height: 14px; }
.btn-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: btn-spin 700ms linear infinite;
}
@keyframes btn-spin { to { transform: rotate(360deg); } }

.divider-or {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 12px;
  margin: 22px 0 18px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--text-3);
  text-transform: uppercase;
}
.divider-or::before,
.divider-or::after {
  content: '';
  height: 1px;
  background: var(--border);
}

.oauth-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
}
.oauth-btn {
  height: 40px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  transition: background 120ms, border-color 120ms;
}
.oauth-btn:hover { background: var(--canvas); border-color: rgba(14, 14, 14, 0.30); }
.oauth-btn .name { letter-spacing: -0.005em; }

.lf-meta {
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.lf-link {
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text-2);
  text-decoration: none;
}
.lf-link:hover { color: var(--text); text-decoration: underline; }
.lf-link.primary { color: var(--text); font-weight: 500; }

.lf-foot {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
  margin-top: auto;
  padding-top: 24px;
}
.lf-foot .legal {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--text-3);
  line-height: 1.7;
  max-width: 380px;
}
.lf-foot .legal a { color: var(--text-2); text-decoration: none; }
.lf-foot .legal a:hover { color: var(--text); text-decoration: underline; }
.lf-foot .lang {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--text-2);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}
.lf-foot .lang::after {
  content: '▾';
  font-size: 8px;
  color: var(--text-3);
}

/* 2FA segmented code input — fixed cell width so the row doesn't stretch
   across the form pane on wider viewports. 44px cells × 6 + 5×8px gaps
   = 304px total, which sits comfortably inside .lf-body (380px max). */
.code-input {
  display: grid;
  grid-template-columns: repeat(6, 44px);
  gap: 8px;
  justify-content: flex-start;
  max-width: 320px;
}
.code-cell {
  width: 44px;
  height: 52px;
  padding: 0;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  text-align: center;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 20px;
  font-weight: 500;
  color: var(--text);
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
}
.code-cell:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}

.code-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.code-meta .resend {
  color: var(--text-2);
  cursor: pointer;
}
.code-meta .resend:hover { color: var(--text); }

.email-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px 6px 8px;
  border: 1px solid var(--border);
  background: var(--canvas);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-2);
  letter-spacing: 0.02em;
  margin-bottom: 18px;
}
.email-avatar {
  width: 18px;
  height: 18px;
  background: var(--text);
  color: var(--brand);
  display: inline-grid;
  place-items: center;
  font-size: 9px;
  letter-spacing: 0.02em;
  font-weight: 600;
}
.email-em { color: var(--text); }
.email-switch {
  margin-left: 4px;
  padding-left: 10px;
  border-left: 1px solid var(--border);
  color: var(--text-3);
  cursor: pointer;
  font-family: var(--font-sans);
  font-size: 12px;
  letter-spacing: 0;
}
.email-switch:hover { color: var(--text); }

.remember-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 20px 0;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-2);
}
.remember-row input {
  accent-color: #0E0E0E;
  width: 14px;
  height: 14px;
  margin: 0;
}

/* ============================================================
   RIGHT — dark reassurance panel
   ============================================================ */
.rp-pane {
  --ti-1: #E8E6E0;
  --ti-2: #9A9A95;
  --ti-3: #5F5F5C;
  --bd-inv: rgba(255, 255, 255, 0.08);
  --bd-inv-s: rgba(255, 255, 255, 0.16);
  --inverse:    #0E0E0E;
  --inverse-2:  #14161B;

  background: var(--inverse);
  color: var(--ti-1);
  padding: 32px 56px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 100vh;
}
.rp-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.rp-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--ti-3);
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.rp-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--ti-2);
}
.rp-live .pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--pos);
  box-shadow: 0 0 0 0 rgba(25, 195, 125, 0.6);
  animation: rp-pulse 2.4s infinite;
}
@keyframes rp-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0.5); }
  70%  { box-shadow: 0 0 0 6px rgba(25, 195, 125, 0); }
  100% { box-shadow: 0 0 0 0 rgba(25, 195, 125, 0); }
}
.rp-version {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--ti-3);
}

.rp-headline {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 48px;
  letter-spacing: -0.025em;
  line-height: 1.0;
  margin: 64px 0 0;
  color: var(--ti-1);
  max-width: 560px;
}
.rp-headline em {
  font-style: normal;
  background: var(--brand);
  color: var(--inverse);
  padding: 0 0.08em;
}
.rp-lede {
  font-family: var(--font-sans);
  font-size: 15px;
  line-height: 1.55;
  color: var(--ti-2);
  margin: 16px 0 0;
  max-width: 520px;
}

/* Terminal */
.terminal {
  margin-top: 36px;
  background: var(--inverse-2);
  border: 1px solid var(--bd-inv);
  border-radius: var(--radius-sm);
}
.terminal-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--bd-inv);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--ti-3);
}
.terminal-meta {
  display: inline-flex;
  gap: 14px;
  align-items: center;
}
.terminal-head .clock { color: var(--ti-2); letter-spacing: 0.06em; }
.terminal-head .dots { display: inline-flex; gap: 4px; }
.terminal-head .dots span {
  width: 8px;
  height: 8px;
  background: transparent;
  border: 1px solid var(--bd-inv-s);
  border-radius: 50%;
}

.ticker-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}
.ticker-table thead th {
  text-align: left;
  padding: 8px 16px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--ti-3);
  font-weight: 600;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--bd-inv);
}
.ticker-table thead th.num-h { text-align: right; }
.ticker-table tbody td {
  padding: 9px 16px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12.5px;
  color: var(--ti-1);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}
.ticker-table tbody tr:last-child td { border-bottom: 0; }
.ticker-table tbody td.num { text-align: right; }
.ticker-table tbody td.sym { font-weight: 500; letter-spacing: 0.04em; }
.ticker-table .pos { color: var(--pos); }
.ticker-table .neg { color: var(--neg); }
.ticker-table .arr { font-size: 9px; margin-right: 4px; }
.ticker-table .chg { display: inline-flex; align-items: center; }

.tick-flash-up { animation: flash-up 800ms ease-out; }
.tick-flash-dn { animation: flash-dn 800ms ease-out; }
@keyframes flash-up {
  0%   { background: rgba(25, 195, 125, 0.18); }
  100% { background: transparent; }
}
@keyframes flash-dn {
  0%   { background: rgba(239, 68, 68, 0.18); }
  100% { background: transparent; }
}

.sparkline { width: 92px; height: 22px; vertical-align: middle; }
.sparkline.up polyline { stroke: var(--pos); }
.sparkline.down polyline { stroke: var(--neg); }

/* Mini index card */
.idx-card {
  margin-top: 24px;
  background: var(--inverse-2);
  border: 1px solid var(--bd-inv);
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  border-radius: var(--radius-sm);
}
.idx-cell {
  padding: 16px 18px;
  border-right: 1px solid var(--bd-inv);
}
.idx-cell:last-child { border-right: 0; }
.idx-cell .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--ti-3);
  font-weight: 600;
  margin-bottom: 6px;
}
.idx-cell .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  color: var(--ti-1);
  font-weight: 500;
}
.idx-cell .sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--ti-2);
  margin-top: 4px;
}
.idx-cell .sub.pos { color: var(--pos); }

/* Bottom citation */
.rp-foot {
  margin-top: auto;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--bd-inv);
}
.rp-quote {
  max-width: 480px;
  color: var(--ti-2);
  font-size: 13.5px;
  line-height: 1.55;
}
.rp-quote-name {
  display: block;
  margin-top: 10px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ti-3);
}
.rp-quote-name strong { color: var(--ti-1); font-weight: 500; }
.rp-links {
  display: flex;
  gap: 18px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--ti-3);
}
.rp-links a { color: var(--ti-3); text-decoration: none; }
.rp-links a:hover { color: var(--ti-1); }

/* ============================================================
   Responsive — stack panes
   ============================================================ */
@media (max-width: 1100px) {
  .login-screen { grid-template-columns: 1fr; }
  .rp-pane { display: none; }
  .lf-pane { border-right: 0; max-width: 560px; margin: 0 auto; }
}
</style>
