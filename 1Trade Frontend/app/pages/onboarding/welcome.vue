<script setup lang="ts">
/**
 * /onboarding/welcome — first screen after email verification.
 *
 * Light, layout-less. Two paths off one persona check:
 *   - AI company / datacenter → ACTIVATION. Starter credits are already granted (platform-core
 *     does it on verify, or on signup when no verification gate is configured), the API key is
 *     minted here, and the first request is shown ready to run.
 *     The goal is that this screen is the last one before a real metered call, not a menu of
 *     places to go next.
 *   - trader → the paper-account + live-index showcase, then the KYC gate (exchange is paused).
 *
 * The guided tour stays an optional link; it is not in the path to first action.
 * Brief warmth, no consumer-cheer. Trust-building via real numbers.
 */
definePageMeta({ layout: false })
useHead({ title: 'Welcome to 1Trade — 1Trade', htmlAttrs: { 'data-theme': 'light' } })

const route = useRoute()
const firstName = ref<string>(typeof route.query.name === 'string' ? route.query.name : 'Jane')

// Persona-aware primary CTA — AI company → console, datacenter → dashboard, trader → KYC gate (the
// paused exchange). Never hardcode /trade: a verified AI company must not land on the exchange.
const personaCx = usePersona()
const isTrader = computed(() => personaCx.persona.value === 'trader')
const ctaTo = computed(() => personaCx.postOnboard.value)
const ctaLabel = computed(() => ({
  trader: 'Continue', enterprise: 'Go to your console', partner: 'Go to your dashboard',
}[personaCx.persona.value]))

// ─── AI-company activation ───────────────────────────────────────────────────────────────
// This screen used to be three links to other screens. Signup creates a tenant with NO key and
// NO credits, so an AI company had to visit Settings for a key and then Wallet to pay before the
// first API call — four screens and a card, against the sub-5-minute time-to-first-action
// commitment. It now provisions instead: platform-core grants the starter paper balance at the
// point the account becomes usable, and the key is minted right here, so the request below runs
// immediately. Step 1 reports the balance it actually read back — it never asserts a grant.
const { create: createApiKey } = useKeys()
const creating = ref(false)
const keyError = ref('')
const secret = ref('')
const copied = ref(false)
const trialCredits = ref<string | null>(null)

/** True only once a positive balance has actually been read back from the ledger. */
const hasCredits = computed(() => {
  const v = Number(trialCredits.value ?? 0)
  return Number.isFinite(v) && v > 0
})

/** The starter balance granted on verification. Shown as proof they can spend before paying. */
async function loadTrialBalance() {
  try {
    const r = await $fetch<{ balances: { credit_type: string; balance: string }[] }>('/api/wallet/balances')
    trialCredits.value = r.balances?.find((b) => b.credit_type === 'text')?.balance ?? null
  } catch { /* console shows the real balance either way — never block onboarding on this */ }
}

/** Mint the tenant's first key. The secret is returned once, so it is held only in memory here. */
async function mintFirstKey() {
  if (creating.value || secret.value) return
  creating.value = true
  keyError.value = ''
  try {
    const r = await createApiKey('Default key', [])
    secret.value = r.secret
  } catch {
    keyError.value = 'Could not create the key. Open the console and try again from Settings.'
  } finally {
    creating.value = false
  }
}

/** Copy the ready-to-run request, secret included — the point is that it works unedited. */
async function copyCurl() {
  try {
    await navigator.clipboard.writeText(curlSnippet.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1800)
  } catch { /* clipboard blocked — the block is selectable */ }
}

const curlSnippet = computed(() => `curl "$TRADE1_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer ${secret.value || '<your-key>'}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"llama-3.1-8b","messages":[{"role":"user","content":"Say hi in three words"}]}'`)

onMounted(() => { if (!isTrader.value) loadTrialBalance() })

// Live AI Index (mean reversion to 1.0024)
const indexValue = ref<number>(1.0024)
const indexPct   = ref<number>(0.18)
let tickInterval: ReturnType<typeof setInterval> | null = null

interface MarketRow { sym: string; px: number; deltaPct: number; precision: number }
const markets = ref<MarketRow[]>([
  { sym: 'EAI-IDX',   px: 0.001005, deltaPct: 0.18, precision: 6 },
  { sym: 'TEXT-SPOT', px: 0.001210, deltaPct: 1.84, precision: 6 },
  { sym: 'H100-SPOT', px: 2.99,     deltaPct: 0.18, precision: 2 },
])

function tickIndex() {
  let v = indexValue.value
  v = v + (1.0024 - v) * 0.04 + (Math.random() - 0.5) * 0.0006
  v = Math.max(0.992, Math.min(1.015, v))
  indexValue.value = v
  indexPct.value = (v - 1.0) * 100

  // Markets drift around the same regime
  const ratio = v / 1.0024
  markets.value = markets.value.map((m, i) => {
    const base = i === 0 ? 0.001005 : i === 1 ? 0.001210 : 2.99
    const jitter = i === 2 ? (Math.random() - 0.5) * 0.004 : (Math.random() - 0.5) * 0.0000008
    return { ...m, px: base * ratio + jitter }
  })
}

// The live index ticker is trading-only — don't run it on the AI-company welcome.
onMounted(() => { if (isTrader.value) tickInterval = setInterval(tickIndex, 2400) })
onBeforeUnmount(() => { if (tickInterval) clearInterval(tickInterval) })

const indexDisplay = computed(() => (indexValue.value * 1000).toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 }))
const indexDeltaStr = computed(() => {
  const arrow = indexPct.value >= 0 ? '▲' : '▼'
  return arrow + ' ' + Math.abs(indexPct.value).toFixed(2) + '%'
})
const indexDeltaPos = computed(() => indexPct.value >= 0)

function fmtPx(px: number, precision: number): string {
  return '$' + px.toFixed(precision)
}
function fmtDelta(pct: number): string {
  const arrow = pct >= 0 ? '▲' : '▼'
  return arrow + ' ' + Math.abs(pct).toFixed(2) + '%'
}
</script>

<template>
  <div class="welcome" data-theme="light">
    <header class="chrome">
      <NuxtLink to="/" class="brand"><span class="mark" />1Trade</NuxtLink>
      <NuxtLink to="/login" class="chrome-link">Sign out</NuxtLink>
    </header>

    <div class="center">
      <div class="hero">
        <h1>Welcome to 1Trade, {{ firstName }}.</h1>
        <p class="sub">{{ isTrader ? 'Your paper trading account is ready.' : 'Inference and GPU compute on prepaid credits. Here\'s where to start.' }}</p>

        <!-- AI company: activation, not a menu. Everything needed for the first call is here. -->
        <div v-if="!isTrader" class="act">
          <ol class="steps">
            <!-- Reports the REAL balance. Never claim credits were added without having read
                 them back — if the grant failed, saying otherwise sends the user to a 402. -->
            <li class="step" :class="{ done: hasCredits }">
              <span class="s-n">{{ hasCredits ? '✓' : '1' }}</span>
              <div class="s-body">
                <template v-if="hasCredits">
                  <h2 class="s-t">Starter credits added</h2>
                  <p class="s-d">
                    <span class="mono s-amt">{{ trialCredits }}</span> text credits, on the house —
                    enough to run the request below. Paper balance, so nothing is charged.
                  </p>
                </template>
                <template v-else>
                  <h2 class="s-t">Credits</h2>
                  <p class="s-d">
                    Your starter balance isn't showing yet. It usually lands within a moment of
                    verifying — the console has the live balance, and you can top up any time.
                  </p>
                  <NuxtLink to="/wallet/buy" class="s-link">Buy credits →</NuxtLink>
                </template>
              </div>
            </li>

            <li class="step" :class="{ done: !!secret }">
              <span class="s-n">{{ secret ? '✓' : '2' }}</span>
              <div class="s-body">
                <h2 class="s-t">Your API key</h2>
                <template v-if="!secret">
                  <p class="s-d">One key, scoped to this account. The secret is shown once.</p>
                  <button class="btn-key" type="button" :disabled="creating" @click="mintFirstKey">
                    {{ creating ? 'Creating…' : 'Create my API key' }}
                  </button>
                  <p v-if="keyError" class="s-err">{{ keyError }}</p>
                </template>
                <template v-else>
                  <p class="s-d">Copy it now — it is not retrievable again.</p>
                  <code class="secret mono">{{ secret }}</code>
                </template>
              </div>
            </li>

            <li class="step">
              <span class="s-n">3</span>
              <div class="s-body">
                <h2 class="s-t">Make your first call</h2>
                <p class="s-d">
                  OpenAI-compatible. Set <code class="ic mono">TRADE1_BASE</code> to your gateway URL and run:
                </p>
                <div class="code">
                  <button class="code-copy" type="button" @click="copyCurl">{{ copied ? 'Copied' : 'Copy' }}</button>
                  <pre class="mono"><code>{{ curlSnippet }}</code></pre>
                </div>
              </div>
            </li>
          </ol>
        </div>

        <!-- Trader (paused exchange): the live index + markets showcase -->
        <div v-else class="cards">
          <div class="card c-1">
            <div class="cap">Paper balance</div>
            <div class="big mono">$10,000.00</div>
            <div class="sub-line">USD ready · risk-free practice</div>
          </div>

          <div class="card c-2">
            <div class="cap">AI Index live</div>
            <div class="big mono">$1 = {{ indexDisplay }}</div>
            <div class="sub-line">
              credits per dollar ·
              <span class="mono" :class="{ pos: indexDeltaPos, neg: !indexDeltaPos }">{{ indexDeltaStr }}</span>
            </div>
          </div>

          <div class="card c-3">
            <div class="cap">Markets open</div>
            <ul class="markets-list">
              <li v-for="m in markets" :key="m.sym" class="market-row">
                <span class="sym mono">{{ m.sym }}</span>
                <span class="px mono">{{ fmtPx(m.px, m.precision) }}</span>
                <span class="delta mono pos">{{ fmtDelta(m.deltaPct) }}</span>
              </li>
            </ul>
          </div>
        </div>

        <div class="cta-row">
          <NuxtLink :to="ctaTo" class="btn-primary">
            {{ ctaLabel }}
            <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
              <path d="M3 8h10M9 4l4 4-4 4" />
            </svg>
          </NuxtLink>
          <NuxtLink to="/onboarding/tour" class="tour-link">Or take the guided product tour</NuxtLink>
        </div>
      </div>
    </div>

    <footer class="page-foot">
      <span class="mono muted">
        {{ isTrader
          ? 'Account EX-PT-7A3C91 · paper-trading mode · capital trading requires further verification'
          : 'Sandbox mode · prepaid credits · real-money purchases require verification (KYC)' }}
      </span>
      <a href="#" class="foot-link">{{ isTrader ? "What's paper trading? →" : 'How credits work →' }}</a>
    </footer>
  </div>
</template>

<style scoped>
.welcome {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.55;
  display: flex;
  flex-direction: column;
}
.welcome .mono   { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.welcome .muted  { color: var(--text-3); }
.welcome .pos    { color: var(--pos); font-weight: 600; }
.welcome .neg    { color: var(--neg); font-weight: 600; }

/* Chrome */
.chrome {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28px 48px;
  flex-shrink: 0;
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

/* Hero */
.center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 32px;
}
.hero {
  width: 100%;
  max-width: 1100px;
  text-align: center;
}
.hero h1 {
  font-family: var(--font-display);
  font-size: 52px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.08;
  margin: 0 0 12px;
  color: var(--text);
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 80ms forwards;
}
.hero .sub {
  font-size: 17px;
  color: var(--text-2);
  margin: 0 0 48px;
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 200ms forwards;
}
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* Cards */
.cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin: 0 0 40px;
  text-align: left;
}
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: 24px;
  min-height: 168px;
  display: flex;
  flex-direction: column;
  opacity: 0;
  transform: translateY(8px);
  animation: cardIn 600ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
  transition: border-color 160ms ease;
}
.card.c-1 { animation-delay: 340ms; }
.card.c-2 { animation-delay: 420ms; }
.card.c-3 { animation-delay: 500ms; }
.card:hover { border-color: var(--border-strong); }
@keyframes cardIn { to { opacity: 1; transform: translateY(0); } }

/* Quick-start cards (AI company) are clickable — whole card is a link to a first action */
.card.qs {
  text-decoration: none;
  color: var(--text);
}
.card.qs:hover { border-color: var(--brand); }
.card.qs .qs-go {
  margin-top: auto;
  padding-top: 14px;
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--accent);
  transition: color 160ms ease;
}
.card.qs:hover .qs-go { color: var(--text); }

.cap {
  font-family: var(--font-mono);
  font-size: 10px; font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 18px;
}
.big {
  font-size: 28px; font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text);
  line-height: 1.1;
  margin-bottom: 6px;
}
.sub-line {
  font-size: 13px;
  color: var(--text-2);
}

/* Markets list (card 3) */
.markets-list { list-style: none; margin: 0; padding: 0; }
.market-row {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 12px;
  align-items: center;
  padding: 9px 0;
  font-family: var(--font-mono);
}
.market-row + .market-row { border-top: 1px solid var(--border); }
.market-row .sym {
  color: var(--text); font-weight: 600;
  font-size: 11px; letter-spacing: 0.06em;
}
.market-row .px {
  color: var(--text); font-size: 13px; font-weight: 500;
}
.market-row .delta {
  font-size: 11px; font-weight: 600;
}

/* CTA row */
.cta-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  opacity: 0;
  animation: fadeUp 700ms cubic-bezier(0.16, 1, 0.3, 1) 620ms forwards;
}
.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 15px 30px;
  background: var(--brand);
  color: var(--text);
  border: 1px solid var(--brand);
  border-radius: 2px;
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 15px;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease, transform 80ms ease;
}
.btn-primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn-primary:active { transform: translateY(1px); }
.tour-link {
  color: var(--text-2);
  font-size: 13px;
  text-decoration: underline;
  text-underline-offset: 3px;
  text-decoration-thickness: 1px;
  font-family: var(--font-sans);
}
.tour-link:hover { color: var(--text); }

/* Footer */
.page-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 48px 22px;
  font-size: 11.5px;
  border-top: 1px solid var(--border);
  background: var(--elevated);
}
.foot-link { color: var(--accent); text-decoration: none; }
.foot-link:hover { text-decoration: underline; }

@media (max-width: 900px) {
  .cards { grid-template-columns: 1fr; }
  .hero h1 { font-size: 36px; }
  .chrome, .page-foot { padding-left: 24px; padding-right: 24px; }
}
</style>

<style scoped>
/* ── AI-company activation ──
   A numbered checklist rather than a card grid: these are steps with an order and a state,
   not destinations to choose between. Step 1 lands already done, so the screen opens on
   something the account HAS rather than something it must go do. */
.act { text-align: left; margin: var(--sp-6) 0 var(--sp-7); }

.steps { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--sp-5); }

.step { display: grid; grid-template-columns: 26px 1fr; gap: var(--sp-4); align-items: start; }

.s-n {
  width: 26px; height: 26px; border-radius: 50%;
  display: grid; place-items: center;
  font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: 600;
  color: var(--text-3);
  border: 1px solid var(--border-strong);
  background: var(--elevated);
}
.step.done .s-n { color: var(--text-on-accent); background: var(--brand); border-color: var(--brand); }

.s-t { font-size: var(--fs-md); font-weight: 600; letter-spacing: var(--ls-near); margin: 2px 0 var(--sp-2); }
.s-d { font-size: var(--fs-sm); line-height: var(--lh-relax); color: var(--text-2); margin: 0 0 var(--sp-3); }
.s-amt { color: var(--text); font-weight: 600; }
.s-err { font-size: var(--fs-sm); color: var(--neg); margin: var(--sp-2) 0 0; }
.s-link { font-size: var(--fs-sm); font-weight: 600; color: var(--brand); text-decoration: none; }
.s-link:hover { text-decoration: underline; }

.btn-key {
  font: inherit; font-size: var(--fs-sm); font-weight: 600;
  padding: 8px 14px; border-radius: var(--radius-sm); cursor: pointer;
  background: var(--brand); color: var(--text-on-accent); border: 1px solid var(--brand);
  transition: background-color var(--dur) var(--ease);
}
.btn-key:hover:not(:disabled) { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn-key:disabled { opacity: 0.6; cursor: default; }

/* The one-time secret: full width, selectable, wrapping — it must never be visually truncated. */
.secret {
  display: block; width: 100%;
  padding: 10px 12px;
  font-size: var(--fs-sm);
  color: var(--text);
  background: var(--sunken);
  border: 1px solid var(--brand);
  border-radius: var(--radius-sm);
  word-break: break-all;
  user-select: all;
}

.ic {
  font-size: 0.92em; padding: 1px 5px;
  background: var(--sunken); border: 1px solid var(--border); border-radius: 2px;
}

.code { position: relative; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--sunken); }
.code pre { margin: 0; padding: 12px; overflow-x: auto; font-size: 12px; line-height: 1.65; color: var(--text); }
.code-copy {
  position: absolute; top: 8px; right: 8px;
  font-family: var(--font-mono); font-size: var(--fs-tiny);
  letter-spacing: var(--ls-tab); text-transform: uppercase;
  padding: 3px 8px; cursor: pointer;
  color: var(--text-2); background: var(--elevated);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
}
.code-copy:hover { color: var(--brand); border-color: var(--brand); }

@media (max-width: 560px) {
  .step { grid-template-columns: 22px 1fr; gap: var(--sp-3); }
  .code pre { font-size: 11px; }
}
</style>
