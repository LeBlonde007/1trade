<script setup lang="ts">
/**
 * /onboarding/kyc — Trader Light KYC flow (4 steps).
 *
 * Standalone chrome (no marketing nav / no app sidebar) to keep the focus
 * on the form. The user's design tweaks landed on: lime primary button,
 * filled-by-default data, flow view. The stacked design-tool view and
 * the Tweaks panel are not ported — they are design affordances, not
 * product UX.
 */
definePageMeta({ layout: false })
useHead({ title: 'Identity Verification — Exascale', htmlAttrs: { 'data-theme': 'light' } })

// Persona-aware. KYC (identity verification) is a *trading* requirement — only the trader persona
// goes through the Light-KYC flow below. AI companies don't trade (verified → straight to the
// console); datacenter partners onboard capacity instead. See the persona branch in the template.
const personaCx = usePersona()
const isTrader = computed(() => personaCx.persona.value === 'trader')
const personaName = computed(() => personaCx.meta.value.short)

type StepKey = 'personal' | 'background' | 'intentions' | 'confirm'

interface StepDef {
  key: StepKey
  label: string
  title: string
  eyebrow: string
}

const STEPS: StepDef[] = [
  { key: 'personal',   label: 'Personal',    title: 'Personal info',       eyebrow: 'Step 01 · Identity' },
  { key: 'background', label: 'Background',  title: 'Trading background',  eyebrow: 'Step 02 · Experience' },
  { key: 'intentions', label: 'Intentions',  title: 'Trading intentions',  eyebrow: 'Step 03 · Account use' },
  { key: 'confirm',    label: 'Confirm',     title: 'Review & confirm',    eyebrow: 'Step 04 · Submit' },
]

// Countries with sub-region requirement match the design rule (US / CA / AU).
const COUNTRIES: Array<[string, string]> = [
  ['US', 'United States'],
  ['GB', 'United Kingdom'],
  ['CA', 'Canada'],
  ['DE', 'Germany'],
  ['FR', 'France'],
  ['NL', 'Netherlands'],
  ['CH', 'Switzerland'],
  ['IE', 'Ireland'],
  ['JP', 'Japan'],
  ['SG', 'Singapore'],
  ['HK', 'Hong Kong SAR'],
  ['AU', 'Australia'],
  ['AE', 'United Arab Emirates'],
  ['BR', 'Brazil'],
  ['IN', 'India'],
  ['MX', 'Mexico'],
  ['SE', 'Sweden'],
  ['NO', 'Norway'],
  ['KR', 'South Korea'],
  ['IL', 'Israel'],
]

const US_STATES = [
  'Alabama','Alaska','Arizona','Arkansas','California','Colorado','Connecticut','Delaware','Florida','Georgia',
  'Hawaii','Idaho','Illinois','Indiana','Iowa','Kansas','Kentucky','Louisiana','Maine','Maryland',
  'Massachusetts','Michigan','Minnesota','Mississippi','Missouri','Montana','Nebraska','Nevada','New Hampshire','New Jersey',
  'New Mexico','New York','North Carolina','North Dakota','Ohio','Oklahoma','Oregon','Pennsylvania','Rhode Island','South Carolina',
  'South Dakota','Tennessee','Texas','Utah','Vermont','Virginia','Washington','West Virginia','Wisconsin','Wyoming',
]

const INTENTIONS = [
  { id: 'explore',  title: 'Explore AI compute as a new asset class',           sub: 'Most common — paper traders exploring the venue' },
  { id: 'hedge',    title: "Hedge my company's AI compute costs",               sub: 'CFOs / FinOps offsetting cloud-spend volatility' },
  { id: 'test',     title: 'Test the venue before institutional deployment',    sub: 'Trading desks evaluating execution quality' },
  { id: 'personal', title: 'Personal investing',                                sub: 'Individual portfolios, retirement, side capital' },
  { id: 'other',    title: 'Other',                                             sub: 'Tell us in your own words below' },
]

const YEARS_OPTIONS = [
  { value: '<1',   label: 'Less than 1 year' },
  { value: '1-3',  label: '1 – 3 years' },
  { value: '3-10', label: '3 – 10 years' },
  { value: '10+',  label: 'More than 10 years' },
]

const ASSET_OPTIONS = [
  'Equities only',
  'Equities & options',
  'Futures / derivatives',
  'FX',
  'Crypto',
  'Multiple of the above',
  'None',
]

const ROLE_OPTIONS = ['Trader', 'Portfolio manager', 'Quant researcher', 'Engineer', 'Other']

const VOLUME_OPTIONS = [
  { value: '<10k',    label: 'Under $10,000' },
  { value: '10-50k',  label: '$10,000 – $50,000' },
  { value: '50-250k', label: '$50,000 – $250,000' },
  { value: '250k-1m', label: '$250,000 – $1,000,000' },
  { value: '1-10m',   label: '$1,000,000 – $10,000,000' },
  { value: '10m+',    label: 'Over $10,000,000' },
]

const FOCUS_OPTIONS = [
  'AI-INDEX',
  'GPU credits (H100/H200)',
  'Sub-indices (TEXT, IMAGE, VIDEO)',
  'All markets',
  'Not sure yet',
]

const TRADED_LABELS: Record<string, string> = {
  yes: 'Yes, actively', some: 'Some experience', no: 'No',
}
const YEARS_LABELS = Object.fromEntries(YEARS_OPTIONS.map(o => [o.value, o.label]))
const VOLUME_LABELS = Object.fromEntries(VOLUME_OPTIONS.map(o => [o.value, o.label]))

// Realistic prefilled defaults (matches the design's INITIAL_DATA).
const data = reactive({
  first: 'Marcus',
  last:  'Chen',
  dob:   '1989-03-14',
  country: 'US',
  state: 'New York',
  tradedBefore: 'yes',
  years: '3-10',
  assets: 'Equities & options',
  isPro: 'yes',
  firmName: 'Citadel Securities',
  firmRole: 'Quant researcher',
  intentions: ['explore', 'test'] as string[],
  otherReason: '',
  volume: '50-250k',
  focus: 'AI-INDEX',
  confirmed: false,
})

// Default to step 2 (index 1) — matches the brief's example state.
const stepIdx = ref(1)
const submitting = ref(false)
const submitted = ref(false)

function fmtClock(d = new Date()): string {
  const hh = ((d.getHours() + 11) % 12) + 1
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ap = d.getHours() >= 12 ? 'PM' : 'AM'
  return `${hh}:${mm} ${ap}`
}
const savedAt = ref(fmtClock())

const showStateField = computed(() => ['US', 'CA', 'AU'].includes(data.country))
const stateLabel = computed(() => {
  if (data.country === 'US') return 'State of residence'
  if (data.country === 'CA') return 'Province'
  return 'State / territory'
})
const stateOptions = computed(() => (data.country === 'US' ? US_STATES : ['—']))

function validate(idx: number) {
  const e: Record<string, boolean> = {}
  if (idx === 0) {
    if (!data.first.trim()) e.first = true
    if (!data.last.trim()) e.last = true
    if (!data.dob) e.dob = true
    if (!data.country) e.country = true
    if (showStateField.value && !data.state) e.state = true
  } else if (idx === 1) {
    if (!data.tradedBefore) e.tradedBefore = true
    if (!data.years) e.years = true
    if (!data.isPro) e.isPro = true
    if (data.isPro === 'yes' && !data.firmName.trim()) e.firmName = true
  } else if (idx === 3) {
    if (!data.confirmed) e.confirmed = true
  }
  return e
}

const errors = computed(() => validate(stepIdx.value))
const canContinue = computed(() => Object.keys(errors.value).length === 0)
const currentStep = computed(() => STEPS[stepIdx.value]!)

// Re-mark autosave whenever the form actually changes (not the submit gate).
watch(
  () => ({
    a: data.first, b: data.last, c: data.dob, d: data.country, e: data.state,
    f: data.tradedBefore, g: data.years, h: data.assets, i: data.isPro,
    j: data.firmName, k: data.firmRole, l: [...data.intentions].join(','),
    m: data.otherReason, n: data.volume, o: data.focus,
  }),
  () => { savedAt.value = fmtClock() },
)

function goTo(i: number, push = true) {
  if (i < 0 || i >= STEPS.length) return
  stepIdx.value = i
  if (push && typeof window !== 'undefined') {
    history.pushState({ step: i }, '', '')
  }
  if (typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

function next() {
  if (!canContinue.value) return
  if (stepIdx.value === STEPS.length - 1) {
    submitting.value = true
    setTimeout(() => {
      submitting.value = false
      submitted.value = true
    }, 1400)
    return
  }
  goTo(stepIdx.value + 1)
}
const back = () => goTo(Math.max(0, stepIdx.value - 1))

function reset() {
  submitted.value = false
  stepIdx.value = 0
  if (typeof window !== 'undefined') {
    history.replaceState({ step: 0 }, '', '')
    window.scrollTo({ top: 0 })
  }
}

onMounted(() => {
  history.replaceState({ step: stepIdx.value }, '', '')
  const onPop = (e: PopStateEvent) => {
    const s = e.state && typeof e.state.step === 'number' ? e.state.step : 0
    stepIdx.value = s
  }
  window.addEventListener('popstate', onPop)
  onBeforeUnmount(() => window.removeEventListener('popstate', onPop))
})

function toggleIntention(id: string) {
  const idx = data.intentions.indexOf(id)
  if (idx === -1) data.intentions.push(id)
  else data.intentions.splice(idx, 1)
}

// Country combobox state.
const comboOpen = ref(false)
const comboQuery = ref('')
const comboRef = ref<HTMLDivElement | null>(null)
const filteredCountries = computed(() => {
  const q = comboQuery.value.toLowerCase()
  if (!q) return COUNTRIES
  return COUNTRIES.filter(([code, name]) =>
    name.toLowerCase().includes(q) || code.toLowerCase().includes(q),
  )
})
const selectedCountryName = computed(() => COUNTRIES.find(c => c[0] === data.country)?.[1] ?? '')

function openCombo() {
  comboOpen.value = true
  comboQuery.value = ''
}
function pickCountry(code: string) {
  data.country = code
  if (!['US', 'CA', 'AU'].includes(code)) data.state = ''
  comboOpen.value = false
  comboQuery.value = ''
}
function onDocClick(e: MouseEvent) {
  if (comboRef.value && !comboRef.value.contains(e.target as Node)) {
    comboOpen.value = false
  }
}
onMounted(() => {
  document.addEventListener('mousedown', onDocClick)
  onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
})

function fmtDate(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: '2-digit' })
}

const intentionTitles = computed(() =>
  data.intentions
    .map(id => INTENTIONS.find(o => o.id === id)?.title)
    .filter((t): t is string => Boolean(t)),
)
</script>

<template>
  <div class="kyc-shell">
    <!-- ============ Top chrome ============ -->
    <header class="topbar">
      <div class="brand-lockup">
        <span class="brand-mark" />
        EXASCALE
      </div>
      <div class="topbar-right">
        <span class="save-state">
          <span class="save-dot" />
          <span>Autosaved · {{ savedAt }}</span>
        </span>
        <span class="email">marcus.chen@frontier.lab</span>
        <a href="#" class="exit">Save &amp; exit</a>
      </div>
    </header>

    <!-- ============ Success state ============ -->
    <div v-if="submitted" class="page">
      <div class="success-wrap">
        <div class="success-mark" aria-hidden="true">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="square"
               style="width:24px;height:24px"><path d="M3 8.5l3.2 3.2L13 5" /></svg>
        </div>
        <h1 class="success-title">Paper account opened.</h1>
        <p class="success-sub">Identity verified. Your demo trading account is funded and ready.</p>
        <div class="balance-card">
          <div class="balance-row">
            <span class="lbl">Starting balance</span>
            <span class="val">$10,000.00</span>
          </div>
          <div class="balance-row">
            <span class="lbl">Default market</span>
            <span class="val sm">AI-INDEX</span>
          </div>
          <div class="balance-row">
            <span class="lbl">Account ID</span>
            <span class="val xs">EX-PT-7A3C91</span>
          </div>
        </div>
        <div class="success-actions">
          <NuxtLink to="/onboarding/welcome" class="btn primary accent lg">
            Enter the venue
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"
                 style="width:14px;height:14px"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
          </NuxtLink>
          <button class="btn ghost" type="button" @click="reset">← Restart flow</button>
        </div>
      </div>
    </div>

    <!-- ============ Non-trader personas: KYC doesn't apply ============ -->
    <div v-else-if="!isTrader" class="page">
      <div class="page-header">
        <div class="eyebrow"><span class="dot" /> {{ personaName }} onboarding</div>
        <template v-if="personaCx.persona.value === 'enterprise'">
          <h1 class="page-title">You're verified — welcome to Exascale.</h1>
          <p class="page-subtitle">
            Identity verification (KYC) is only required for real-money <em>trading</em>. As an AI
            company you run inference &amp; compute on prepaid credits — no KYC needed. You're ready to go.
          </p>
        </template>
        <template v-else>
          <h1 class="page-title">Let's onboard your datacenter.</h1>
          <p class="page-subtitle">
            Register your capacity to start supplying GPUs to the Exascale market. Next we'll collect
            your cluster details, location, and payout account — not personal identity.
          </p>
        </template>
      </div>
      <div class="success-actions">
        <NuxtLink v-if="personaCx.persona.value === 'enterprise'" to="/console" class="btn primary accent lg">
          Go to your console
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"
               style="width:14px;height:14px"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
        </NuxtLink>
        <NuxtLink v-else to="/datacenter/register" class="btn primary accent lg">
          Register a datacenter
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"
               style="width:14px;height:14px"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
        </NuxtLink>
        <NuxtLink to="/onboarding/welcome" class="btn ghost">Take the product tour first</NuxtLink>
      </div>
    </div>

    <!-- ============ Interactive flow (trader) ============ -->
    <div v-else class="page">
      <div class="page-header">
        <div class="eyebrow"><span class="dot" /> Trader onboarding · Light KYC</div>
        <h1 class="page-title">Verify your identity to start paper trading.</h1>
        <p class="page-subtitle">
          Light KYC takes about three minutes and unlocks a $10,000 paper-trading account
          against live order-book data. Real-money trading requires a separate Full KYC upgrade.
        </p>
      </div>

      <!-- Stepper -->
      <div class="stepper" role="tablist" aria-label="Onboarding steps">
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
          <div class="step-row">
            <span class="step-num">
              <svg v-if="i < stepIdx" viewBox="0 0 16 16" fill="none" stroke="currentColor"
                   stroke-width="2.2" stroke-linecap="square" style="width:11px;height:11px">
                <path d="M3 8.5l3.2 3.2L13 5" />
              </svg>
              <template v-else>{{ String(i + 1).padStart(2, '0') }}</template>
            </span>
            <span class="step-label">{{ s.label }}</span>
          </div>
          <span class="step-title">{{ s.title }}</span>
        </button>
      </div>

      <!-- Card -->
      <div class="card">
        <div class="card-head">
          <div>
            <div class="eyebrow"><span class="dot" /> {{ currentStep.eyebrow }}</div>
            <h2 class="card-ttl">{{ currentStep.title }}</h2>
          </div>
          <div class="card-meta">
            <strong>{{ String(stepIdx + 1).padStart(2, '0') }}</strong> / 04 ·
            <span class="dim">Light KYC</span>
          </div>
        </div>

        <div class="card-body">
          <!-- ===== Form pane ===== -->
          <div class="form-pane">
            <!-- Step 1 — Personal -->
            <div v-if="stepIdx === 0" class="step-content">
              <div class="field-grid">
                <div class="field col-6">
                  <div class="field-label"><span>— Legal first name</span></div>
                  <input
                    class="input"
                    :class="{ error: errors.first }"
                    v-model="data.first"
                    placeholder="As shown on government ID"
                    autocomplete="given-name"
                  />
                </div>
                <div class="field col-6">
                  <div class="field-label"><span>— Legal last name</span></div>
                  <input
                    class="input"
                    :class="{ error: errors.last }"
                    v-model="data.last"
                    placeholder="As shown on government ID"
                    autocomplete="family-name"
                  />
                </div>

                <div class="field col-6">
                  <div class="field-label"><span>— Date of birth</span></div>
                  <input
                    class="input mono-input"
                    :class="{ error: errors.dob }"
                    type="date"
                    v-model="data.dob"
                    min="1920-01-01"
                    max="2010-01-01"
                  />
                  <div class="field-help">You must be 18 or older.</div>
                </div>

                <div class="field col-6" ref="comboRef">
                  <div class="field-label"><span>— Country of residence</span></div>
                  <div class="combo">
                    <div class="input-affix">
                      <input
                        class="input combo-input"
                        type="text"
                        :value="comboOpen ? comboQuery : selectedCountryName"
                        :placeholder="'Search country'"
                        :style="{ cursor: comboOpen ? 'text' : 'pointer' }"
                        @focus="openCombo"
                        @input="(e) => { comboQuery = (e.target as HTMLInputElement).value; comboOpen = true }"
                      />
                      <span class="affix affix-right">
                        <svg viewBox="0 0 10 6" fill="none" stroke="currentColor" stroke-width="1.4"
                             stroke-linecap="square" style="width:10px;height:6px">
                          <path d="M1 1L5 5L9 1" />
                        </svg>
                      </span>
                    </div>
                    <div v-if="comboOpen" class="combo-menu" role="listbox">
                      <div v-if="filteredCountries.length === 0" class="combo-opt dim">
                        <span>No matches</span>
                      </div>
                      <div
                        v-for="[code, name] in filteredCountries"
                        :key="code"
                        class="combo-opt"
                        :class="{ selected: code === data.country }"
                        @mousedown.prevent="pickCountry(code)"
                      >
                        <span>{{ name }}</span>
                        <span class="combo-right">
                          <span class="iso">{{ code }}</span>
                          <svg v-if="code === data.country" viewBox="0 0 16 16" fill="none"
                               stroke="currentColor" stroke-width="2.2" stroke-linecap="square"
                               class="check" style="width:12px;height:12px">
                            <path d="M3 8.5l3.2 3.2L13 5" />
                          </svg>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="showStateField" class="field col-12">
                  <div class="field-label"><span>— {{ stateLabel }}</span></div>
                  <select
                    class="select"
                    :class="{ error: errors.state }"
                    v-model="data.state"
                  >
                    <option value="" disabled>Select</option>
                    <option v-for="s in stateOptions" :key="s" :value="s">{{ s }}</option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Step 2 — Background -->
            <div v-else-if="stepIdx === 1" class="step-content">
              <div class="field-grid">
                <div class="field col-12">
                  <div class="field-label"><span>— Have you traded financial markets before?</span></div>
                  <div class="radio-group" :style="{ '--cols': 3 }">
                    <label
                      v-for="o in [
                        { value: 'yes',  label: 'Yes, actively' },
                        { value: 'some', label: 'Some experience' },
                        { value: 'no',   label: 'No' },
                      ]"
                      :key="o.value"
                      class="radio-opt"
                      :class="{ checked: data.tradedBefore === o.value }"
                      tabindex="0"
                      @keydown.space.prevent="data.tradedBefore = o.value"
                      @keydown.enter.prevent="data.tradedBefore = o.value"
                    >
                      <input type="radio" :checked="data.tradedBefore === o.value"
                             @change="data.tradedBefore = o.value" />
                      <span>{{ o.label }}</span>
                    </label>
                  </div>
                </div>

                <div class="field col-6">
                  <div class="field-label"><span>— Years of experience</span></div>
                  <select class="select" :class="{ error: errors.years }" v-model="data.years">
                    <option value="" disabled>Select</option>
                    <option v-for="o in YEARS_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
                  </select>
                </div>

                <div class="field col-6">
                  <div class="field-label">
                    <span>— Asset classes traded</span>
                    <span class="req optional">Optional</span>
                  </div>
                  <select class="select" v-model="data.assets">
                    <option value="" disabled>Select</option>
                    <option v-for="o in ASSET_OPTIONS" :key="o" :value="o">{{ o }}</option>
                  </select>
                </div>

                <div class="field col-12">
                  <div class="field-label"><span>— Are you a professional trader at a firm?</span></div>
                  <div class="radio-group" :style="{ '--cols': 2 }">
                    <label
                      v-for="o in [{ value: 'yes', label: 'Yes' }, { value: 'no', label: 'No' }]"
                      :key="o.value"
                      class="radio-opt"
                      :class="{ checked: data.isPro === o.value }"
                      tabindex="0"
                      @keydown.space.prevent="data.isPro = o.value"
                      @keydown.enter.prevent="data.isPro = o.value"
                    >
                      <input type="radio" :checked="data.isPro === o.value"
                             @change="data.isPro = o.value" />
                      <span>{{ o.label }}</span>
                    </label>
                  </div>
                </div>

                <template v-if="data.isPro === 'yes'">
                  <div class="field col-8">
                    <div class="field-label"><span>— Firm name</span></div>
                    <input
                      class="input"
                      :class="{ error: errors.firmName }"
                      v-model="data.firmName"
                      placeholder="e.g. Jane Street, Citadel Securities, Two Sigma"
                    />
                  </div>
                  <div class="field col-4">
                    <div class="field-label"><span>— Role</span></div>
                    <select class="select" v-model="data.firmRole">
                      <option value="" disabled>Select</option>
                      <option v-for="o in ROLE_OPTIONS" :key="o" :value="o">{{ o }}</option>
                    </select>
                  </div>
                </template>
              </div>
            </div>

            <!-- Step 3 — Intentions -->
            <div v-else-if="stepIdx === 2" class="step-content">
              <div class="field-grid">
                <div class="field col-12">
                  <div class="field-label">
                    <span>— Why are you opening this account?</span>
                    <span class="req optional">Optional</span>
                  </div>
                  <div class="field-help tight">
                    Select all that apply. Helps us prioritize features and route product updates.
                  </div>
                  <div class="check-list">
                    <label
                      v-for="o in INTENTIONS"
                      :key="o.id"
                      class="check-opt"
                      :class="{ checked: data.intentions.includes(o.id) }"
                    >
                      <input
                        type="checkbox"
                        :checked="data.intentions.includes(o.id)"
                        @change="toggleIntention(o.id)"
                      />
                      <span class="check-box">
                        <svg v-if="data.intentions.includes(o.id)" viewBox="0 0 16 16" fill="none"
                             stroke="currentColor" stroke-width="2.2" stroke-linecap="square"
                             style="width:10px;height:10px">
                          <path d="M3 8.5l3.2 3.2L13 5" />
                        </svg>
                      </span>
                      <span class="check-content">
                        <span class="check-label">{{ o.title }}</span>
                        <span class="check-sub">{{ o.sub }}</span>
                      </span>
                    </label>
                  </div>
                </div>

                <div v-if="data.intentions.includes('other')" class="field col-12">
                  <div class="field-label">
                    <span>— Tell us more</span>
                    <span class="req optional">Optional</span>
                  </div>
                  <textarea
                    class="textarea"
                    rows="3"
                    v-model="data.otherReason"
                    placeholder="A short sentence is enough."
                  />
                </div>

                <div class="field col-8">
                  <div class="field-label">
                    <span>— Expected monthly trading volume</span>
                    <span class="req optional">Optional</span>
                  </div>
                  <select class="select" v-model="data.volume">
                    <option value="" disabled>Select</option>
                    <option v-for="o in VOLUME_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
                  </select>
                </div>

                <div class="field col-4">
                  <div class="field-label">
                    <span>— Primary market focus</span>
                    <span class="req optional">Optional</span>
                  </div>
                  <select class="select" v-model="data.focus">
                    <option value="" disabled>Select</option>
                    <option v-for="o in FOCUS_OPTIONS" :key="o" :value="o">{{ o }}</option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Step 4 — Confirm -->
            <div v-else class="step-content">
              <div class="summary">
                <section class="summary-section">
                  <div class="summary-head">
                    <span class="summary-ttl" data-num="01">Personal info</span>
                    <button type="button" class="summary-edit" @click="goTo(0)">Edit →</button>
                  </div>
                  <dl class="summary-kv">
                    <dt>Legal name</dt>
                    <dd>{{ data.first || '—' }} {{ data.last }}</dd>
                    <dt>Date of birth</dt>
                    <dd>{{ fmtDate(data.dob) }}</dd>
                    <dt>Country</dt>
                    <dd>{{ COUNTRIES.find(c => c[0] === data.country)?.[1] || '—' }}</dd>
                    <template v-if="data.state">
                      <dt>State / region</dt>
                      <dd>{{ data.state }}</dd>
                    </template>
                  </dl>
                </section>

                <section class="summary-section">
                  <div class="summary-head">
                    <span class="summary-ttl" data-num="02">Trading background</span>
                    <button type="button" class="summary-edit" @click="goTo(1)">Edit →</button>
                  </div>
                  <dl class="summary-kv">
                    <dt>Traded before</dt>
                    <dd>{{ TRADED_LABELS[data.tradedBefore] || '—' }}</dd>
                    <dt>Years of experience</dt>
                    <dd>{{ YEARS_LABELS[data.years] || '—' }}</dd>
                    <template v-if="data.assets">
                      <dt>Asset classes</dt>
                      <dd>{{ data.assets }}</dd>
                    </template>
                    <dt>Professional trader</dt>
                    <dd>{{ data.isPro === 'yes' ? 'Yes' : data.isPro === 'no' ? 'No' : '—' }}</dd>
                    <template v-if="data.isPro === 'yes'">
                      <dt>Firm</dt>
                      <dd>{{ data.firmName || '—' }}<template v-if="data.firmRole"> · {{ data.firmRole }}</template></dd>
                    </template>
                  </dl>
                </section>

                <section class="summary-section">
                  <div class="summary-head">
                    <span class="summary-ttl" data-num="03">Trading intentions</span>
                    <button type="button" class="summary-edit" @click="goTo(2)">Edit →</button>
                  </div>
                  <dl class="summary-kv">
                    <dt>Reasons</dt>
                    <dd class="tag-list">
                      <span v-if="intentionTitles.length === 0" class="dim">— Not specified</span>
                      <span v-for="(t, i) in intentionTitles" :key="i" class="tag">{{ t }}</span>
                    </dd>
                    <template v-if="data.otherReason">
                      <dt>Other</dt>
                      <dd>{{ data.otherReason }}</dd>
                    </template>
                    <dt>Expected volume</dt>
                    <dd>
                      <template v-if="VOLUME_LABELS[data.volume]">{{ VOLUME_LABELS[data.volume] }}</template>
                      <span v-else class="dim">— Not specified</span>
                    </dd>
                    <template v-if="data.focus">
                      <dt>Market focus</dt>
                      <dd>{{ data.focus }}</dd>
                    </template>
                  </dl>
                </section>
              </div>

              <label class="confirm-row" :class="{ checked: data.confirmed }">
                <input type="checkbox" v-model="data.confirmed" />
                <span class="check-box">
                  <svg v-if="data.confirmed" viewBox="0 0 16 16" fill="none" stroke="currentColor"
                       stroke-width="2.2" stroke-linecap="square" style="width:10px;height:10px">
                    <path d="M3 8.5l3.2 3.2L13 5" />
                  </svg>
                </span>
                <span class="confirm-label">
                  I confirm the information above is accurate and complete.
                  <small>
                    Paper-trading accounts are funded with simulated capital and carry no monetary value.
                    By proceeding you agree to the <a href="#">Customer Agreement</a> and
                    <a href="#">Risk Disclosure</a>.
                  </small>
                </span>
              </label>
            </div>
          </div>

          <!-- ===== Help pane ===== -->
          <aside class="help-pane">
            <!-- Step 1 help -->
            <template v-if="stepIdx === 0">
              <div class="help-section">
                <div class="help-eyebrow">Why we ask</div>
                <h4 class="help-title">Required by financial regulations, even for paper trading.</h4>
                <p class="help-body">
                  Exascale is a regulated venue. We collect identity to comply with KYC, sanctions
                  screening, and tax-reporting obligations — for every account, regardless of whether
                  real money is at stake.
                </p>
                <hr class="help-divider" />
                <div class="help-eyebrow">What's collected</div>
                <ul class="help-list">
                  <li>Legal name &amp; date of birth — for sanctions screening</li>
                  <li>Country &amp; sub-region — for jurisdictional eligibility</li>
                  <li>Nothing else at this stage</li>
                </ul>
              </div>
              <div class="help-section">
                <div class="help-eyebrow">Light KYC</div>
                <p class="help-body">
                  Light verification unlocks paper trading on the full venue. Real-money trading
                  requires a separate identity-document upload during Full KYC.
                </p>
              </div>
            </template>

            <!-- Step 2 help -->
            <template v-else-if="stepIdx === 1">
              <div class="help-section">
                <div class="help-eyebrow">Why we ask</div>
                <h4 class="help-title">Calibrates the onboarding to your experience.</h4>
                <p class="help-body">
                  Experienced traders skip the order-book primer and land straight on the venue.
                  New traders get a guided walk-through of order types, position sizing, and the
                  AI-INDEX methodology.
                </p>
                <hr class="help-divider" />
                <div class="help-eyebrow">Disclosure</div>
                <p class="help-body">
                  If you are employed by a regulated trading firm we may need to coordinate with
                  your compliance team before enabling real-money trading. Paper trading is
                  unaffected.
                </p>
              </div>
            </template>

            <!-- Step 3 help -->
            <template v-else-if="stepIdx === 2">
              <div class="help-section">
                <div class="help-eyebrow">Optional</div>
                <h4 class="help-title">Helps us serve you better.</h4>
                <p class="help-body">
                  Everything in this step is optional and can be skipped. Your answers do not affect
                  eligibility for paper trading.
                </p>
                <hr class="help-divider" />
                <div class="help-eyebrow">How it's used</div>
                <ul class="help-list">
                  <li>Sets relevant default watchlists</li>
                  <li>Calibrates the in-product walk-through</li>
                  <li>Prioritizes product updates we send</li>
                </ul>
              </div>
              <div class="help-section">
                <div class="help-eyebrow">Privacy</div>
                <p class="help-body">
                  Aggregated intent data is reported internally. It is not sold, shared with
                  counterparties, or used to determine pricing.
                </p>
              </div>
            </template>

            <!-- Step 4 help -->
            <template v-else>
              <div class="help-section">
                <div class="help-eyebrow">Next</div>
                <h4 class="help-title">We'll fund your account with $10,000 paper credit.</h4>
                <p class="help-body">
                  Simulated capital settles instantly. You can start placing orders against live
                  order-book data within a few seconds.
                </p>
                <dl class="help-kv">
                  <dt>Starting balance</dt><dd>$10,000.00</dd>
                  <dt>Default market</dt><dd>AI-INDEX</dd>
                  <dt>Maker / taker fee</dt><dd>0 bps (paper)</dd>
                  <dt>Daily reset</dt><dd>Manual</dd>
                </dl>
                <hr class="help-divider" />
                <div class="help-eyebrow">Upgrade later</div>
                <p class="help-body">
                  Real-money trading requires Full KYC — document upload, source-of-funds, and bank
                  linking. You can initiate the upgrade from Settings once you're in.
                </p>
              </div>
            </template>
          </aside>
        </div>

        <!-- Card footer -->
        <div class="card-footer">
          <span class="footer-hint">
            <template v-if="stepIdx === 0">
              Encrypted in transit ·
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"
                   style="width:11px;height:11px;vertical-align:-1px">
                <rect x="3.5" y="7" width="9" height="6.5" />
                <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
              </svg>
              TLS 1.3
            </template>
            <template v-else-if="stepIdx === 1">
              Saved automatically · Press <kbd>Enter</kbd> to continue
            </template>
            <template v-else-if="stepIdx === 2">
              Optional step — skip anything that doesn't apply
            </template>
            <template v-else>
              By submitting you open a paper trading account
            </template>
          </span>
          <div class="footer-actions">
            <button v-if="stepIdx > 0" type="button" class="btn secondary" @click="back">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"
                   stroke-linecap="square" style="width:14px;height:14px">
                <path d="M13 8H3M7 4L3 8l4 4" />
              </svg>
              Back
            </button>
            <button
              v-if="stepIdx < STEPS.length - 1"
              type="button"
              class="btn primary accent"
              :disabled="!canContinue"
              @click="next"
            >
              Continue
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"
                   stroke-linecap="square" style="width:14px;height:14px">
                <path d="M3 8h10M9 4l4 4-4 4" />
              </svg>
            </button>
            <button
              v-else
              type="button"
              class="btn primary accent lg"
              :disabled="!canContinue"
              @click="next"
            >
              Open paper trading account
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ============ Submitting overlay ============ -->
    <div v-if="submitting" class="loading-overlay" role="status" aria-live="polite">
      <div class="spinner" />
      <div>Opening account…</div>
    </div>
  </div>
</template>

<style scoped>
/* ============================================================
   The token names below resolve from /app/assets/css/tokens.css
   (canvas / elevated / sunken / text / text-2 / text-3 / border /
   border-strong / brand / brand-hov / accent / pos / warn / fonts).
   Soft border, quaternary text, and warning-subtle tints don't
   have project tokens — kept inline as rgba.
   ============================================================ */

.kyc-shell {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* ========= Top chrome ========= */
.topbar {
  height: 56px;
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
  display: flex;
  align-items: center;
  padding: 0 32px;
  justify-content: space-between;
}
.brand-lockup {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 15px;
  letter-spacing: -0.01em;
  color: var(--text);
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
  gap: 28px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--text-3);
}
.save-state { display: flex; align-items: center; gap: 7px; }
.save-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--pos);
}
.email { opacity: 0.5; }
.exit {
  color: var(--text-2);
  text-decoration: none;
  font-family: var(--font-sans);
  font-size: 13px;
  letter-spacing: 0;
}
.exit:hover { color: var(--text); }

/* ========= Page shell ========= */
.page {
  max-width: 1040px;
  margin: 0 auto;
  padding: 48px 32px 96px;
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
}
.eyebrow .dot {
  width: 5px; height: 5px;
  background: var(--brand);
  display: inline-block;
}
.page-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 32px;
  letter-spacing: -0.02em;
  line-height: 1.1;
  margin: 12px 0 6px;
}
.page-subtitle {
  color: var(--text-2);
  font-size: 15px;
  max-width: 580px;
  margin: 0;
}

/* ========= Stepper ========= */
.stepper {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  margin: 32px 0 24px;
  position: relative;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
}
.step {
  position: relative;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  text-align: left;
  font-family: inherit;
  cursor: pointer;
  transition: background 150ms var(--ease);
}
.step:last-child { border-right: 0; }
.step:hover { background: var(--sunken); }
.step:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}
.step.active { background: var(--elevated); }
.step.active::after {
  content: "";
  position: absolute;
  left: 0; right: 0; bottom: -1px;
  height: 2px;
  background: var(--text);
}
.step-row { display: flex; align-items: center; gap: 10px; }
.step-num {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  background: transparent;
  color: var(--text-3);
  border: 1px solid var(--border-strong);
  flex-shrink: 0;
}
.step.active .step-num {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
}
.step.done .step-num {
  background: var(--text);
  color: var(--brand);
  border-color: var(--text);
}
.step-label {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.step.active .step-label { color: var(--text); }
.step-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}

/* ========= Card ========= */
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02);
}
.card-head {
  padding: 24px 32px 20px;
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
.card-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  padding-top: 6px;
}
.card-meta strong { color: var(--text); font-weight: 500; }
.card-meta .dim { color: var(--text-3); }

.card-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 296px;
  gap: 0;
}
.form-pane {
  padding: 28px 32px 32px;
  border-right: 1px solid rgba(14, 14, 14, 0.06);
  min-width: 0;
}
.help-pane {
  padding: 28px 28px 32px;
  background: var(--canvas);
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 32px;
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
  gap: 6px;
}
.footer-hint kbd {
  display: inline-block;
  font-family: var(--font-mono);
  font-size: 10px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: var(--radius-sm);
  padding: 1px 5px;
  color: var(--text-2);
  margin: 0 3px;
}
.footer-actions { display: flex; gap: 8px; }

/* ========= Form layout ========= */
.field-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px 14px;
}
.field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field.col-12 { grid-column: span 12; }
.field.col-8  { grid-column: span 8; }
.field.col-6  { grid-column: span 6; }
.field.col-4  { grid-column: span 4; }

.field-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.field-label .req {
  color: var(--text-3);
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: none;
  font-size: 11px;
  font-family: var(--font-sans);
}
.field-help {
  font-size: 12px;
  color: var(--text-3);
  font-family: var(--font-sans);
  line-height: 1.4;
}
.field-help.tight { margin-top: -2px; margin-bottom: 4px; }

/* Inputs / selects / textarea ============ */
.input,
.select,
.textarea {
  height: 40px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text);
  width: 100%;
  transition: border-color 150ms var(--ease), box-shadow 150ms var(--ease);
  outline: none;
  -webkit-appearance: none;
  appearance: none;
  font-variant-numeric: tabular-nums;
}
.input::placeholder { color: #B8B8B0; }
.input:hover,
.select:hover { border-color: rgba(14, 14, 14, 0.32); }
.input:focus,
.select:focus,
.textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.input.error,
.select.error { border-color: var(--neg); }
.input.error:focus,
.select.error:focus { box-shadow: 0 0 0 2px rgba(220, 38, 38, 0.18); }
.mono-input { font-family: var(--font-mono); font-size: 13px; }

.select {
  background-image: url("data:image/svg+xml,%3Csvg width='10' height='6' viewBox='0 0 10 6' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1L5 5L9 1' stroke='%234A4A45' stroke-width='1.4' fill='none' stroke-linecap='square'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 32px;
  cursor: pointer;
}

.textarea {
  height: auto;
  padding: 10px 12px;
  resize: vertical;
  min-height: 64px;
  line-height: 1.5;
}

/* Input affix (used by country combobox) */
.input-affix { position: relative; }
.combo-input { padding-right: 32px; }
.affix {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-3);
  pointer-events: none;
  display: flex;
  align-items: center;
}
.affix-right { right: 12px; }

/* ========= Country combobox ========= */
.combo { position: relative; }
.combo-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0; right: 0;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  z-index: 20;
  max-height: 240px;
  overflow-y: auto;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.10);
}
.combo-opt {
  padding: 9px 12px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.combo-opt:hover { background: var(--canvas); }
.combo-opt.dim { color: var(--text-3); cursor: default; }
.combo-right { display: flex; align-items: center; gap: 8px; }
.combo-opt .iso {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.combo-opt .check { color: var(--text); }

/* ========= Radio group ========= */
.radio-group {
  display: grid;
  grid-template-columns: repeat(var(--cols, 3), 1fr);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.radio-opt {
  position: relative;
  cursor: pointer;
  background: var(--elevated);
  padding: 10px 14px;
  border-right: 1px solid var(--border-strong);
  font-size: 14px;
  color: var(--text-2);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 120ms var(--ease), color 120ms var(--ease);
}
.radio-opt:last-child { border-right: 0; }
.radio-opt input { position: absolute; opacity: 0; pointer-events: none; }
.radio-opt:hover { background: var(--sunken); }
.radio-opt.checked {
  background: var(--text);
  color: var(--elevated);
}
.radio-opt.checked::before {
  content: "";
  width: 6px;
  height: 6px;
  background: var(--brand);
  display: inline-block;
}
.radio-opt:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

/* ========= Checkbox stack ========= */
.check-list { display: flex; flex-direction: column; margin-top: 4px; }
.check-opt {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-bottom-width: 0;
  background: var(--elevated);
  cursor: pointer;
  transition: background 120ms var(--ease);
}
.check-opt:first-child { border-radius: var(--radius-sm) var(--radius-sm) 0 0; }
.check-opt:last-child {
  border-bottom-width: 1px;
  border-radius: 0 0 var(--radius-sm) var(--radius-sm);
}
.check-opt:hover,
.check-opt.checked { background: var(--canvas); }
.check-opt input { position: absolute; opacity: 0; pointer-events: none; }
.check-box {
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--elevated);
  flex-shrink: 0;
  margin-top: 1px;
  transition: background 120ms var(--ease), border-color 120ms var(--ease);
  color: var(--brand);
}
.check-opt.checked .check-box,
.confirm-row.checked .check-box {
  background: var(--text);
  border-color: var(--text);
}
.check-content { flex: 1; min-width: 0; }
.check-label {
  display: block;
  font-size: 14px;
  color: var(--text);
  line-height: 1.4;
  letter-spacing: -0.005em;
}
.check-sub {
  display: block;
  font-size: 12px;
  color: var(--text-3);
  margin-top: 2px;
  font-family: var(--font-mono);
  letter-spacing: 0.02em;
}

/* ========= Help pane content ========= */
.help-section { margin-bottom: 24px; }
.help-section:last-child { margin-bottom: 0; }
.help-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-2);
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.help-eyebrow::before {
  content: "—";
  color: var(--text-3);
}
.help-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.005em;
  margin: 0 0 8px;
  color: var(--text);
  line-height: 1.35;
}
.help-body {
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.55;
  margin: 0;
}
.help-body + .help-body { margin-top: 8px; }
.help-list {
  list-style: none;
  padding: 0;
  margin: 8px 0 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.help-list li {
  font-size: 12.5px;
  color: var(--text-2);
  padding-left: 14px;
  position: relative;
  line-height: 1.45;
}
.help-list li::before {
  content: "—";
  position: absolute;
  left: 0;
  color: var(--text-3);
}
.help-divider {
  border: 0;
  border-top: 1px solid var(--border);
  margin: 20px 0;
}
.help-kv {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 6px 12px;
  font-size: 12px;
  margin: 14px 0 0;
}
.help-kv dt {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
  align-self: center;
}
.help-kv dd {
  margin: 0;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
  text-align: right;
}

/* ========= Buttons ========= */
.btn {
  height: 40px;
  padding: 0 18px;
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
  transition:
    background 120ms var(--ease),
    border-color 120ms var(--ease),
    color 120ms var(--ease),
    transform 80ms var(--ease);
  text-decoration: none;
}
.btn:active { transform: translateY(1px); }
.btn:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }

.btn.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
}
.btn.primary:hover { background: #222; border-color: #222; }
.btn.primary.accent {
  background: var(--brand);
  color: var(--text);
  border-color: var(--brand);
}
.btn.primary.accent:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.primary:disabled,
.btn.primary.accent:disabled {
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

.btn.ghost {
  background: transparent;
  color: var(--text-2);
  border-color: transparent;
  padding: 0 12px;
}
.btn.ghost:hover { color: var(--text); }

.btn.lg { height: 48px; padding: 0 22px; font-size: 15px; }

/* ========= Step 4 summary ========= */
.summary {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
}
.summary-section {
  padding: 18px 20px;
  border-bottom: 1px solid rgba(14, 14, 14, 0.06);
}
.summary-section:last-child { border-bottom: 0; }
.summary-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.summary-ttl {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-2);
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.summary-ttl::before {
  content: attr(data-num);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px; height: 20px;
  border-radius: 50%;
  background: var(--text);
  color: var(--brand);
  font-size: 10px;
}
.summary-edit {
  font-family: var(--font-sans);
  font-size: 12px;
  letter-spacing: 0;
  color: var(--text-2);
  background: none;
  border: 0;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  text-transform: none;
}
.summary-edit:hover { background: var(--canvas); color: var(--text); }

.summary-kv {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 8px 24px;
  font-size: 13px;
  margin: 0;
}
.summary-kv dt {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  color: var(--text-3);
  align-self: center;
  margin: 0;
}
.summary-kv dd {
  margin: 0;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.summary-kv dd.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.summary-kv .tag {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.02em;
  padding: 3px 8px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-2);
}
.dim { color: var(--text-3); }

/* Confirm row */
.confirm-row {
  margin-top: 20px;
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 16px 20px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  cursor: pointer;
  position: relative;
}
.confirm-row.checked {
  background: rgba(200, 242, 92, 0.12);
  border-color: rgba(14, 14, 14, 0.30);
}
.confirm-row input { position: absolute; opacity: 0; pointer-events: none; }
.confirm-row .check-box { margin-top: 2px; }
.confirm-label {
  font-size: 13.5px;
  color: var(--text);
  line-height: 1.5;
}
.confirm-label small {
  display: block;
  margin-top: 4px;
  color: var(--text-3);
  font-size: 12px;
}
.confirm-label a { color: var(--text); }

/* ========= Success state ========= */
.success-wrap {
  text-align: center;
  padding: 64px 32px;
  max-width: 560px;
  margin: 0 auto;
}
.success-mark {
  width: 56px;
  height: 56px;
  background: var(--text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
  color: var(--brand);
}
.success-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 28px;
  letter-spacing: -0.02em;
  margin: 0 0 12px;
}
.success-sub {
  color: var(--text-2);
  font-size: 15px;
  margin: 0 0 32px;
}
.balance-card {
  border: 1px solid var(--border);
  background: var(--elevated);
  padding: 20px;
  text-align: left;
  margin-bottom: 24px;
}
.balance-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 4px 0;
}
.balance-row .lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
}
.balance-row .val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  font-weight: 500;
  color: var(--text);
}
.balance-row .val.sm { font-size: 14px; }
.balance-row .val.xs { font-size: 13px; color: var(--text-2); }
.success-actions { display: flex; gap: 12px; justify-content: center; }

/* ========= Loading overlay ========= */
@keyframes spin { to { transform: rotate(360deg); } }
.spinner {
  width: 14px;
  height: 14px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
  display: inline-block;
}
.loading-overlay {
  position: fixed;
  inset: 0;
  background: rgba(250, 250, 247, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 16px;
  z-index: 50;
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-2);
}
.loading-overlay .spinner {
  width: 24px; height: 24px;
  border-width: 2px;
  color: var(--text);
}

/* Step transition */
.step-content {
  animation: stepEnter 200ms cubic-bezier(0, 0, 0.2, 1);
}
@keyframes stepEnter {
  from { opacity: 0; transform: translateY(2px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ========= Responsive ========= */
@media (max-width: 880px) {
  .card-body { grid-template-columns: 1fr; }
  .help-pane { border-left: 0; border-top: 1px solid rgba(14, 14, 14, 0.06); }
  .form-pane { border-right: 0; }
  .stepper { grid-template-columns: repeat(2, 1fr); }
  .step:nth-child(2) { border-right: 0; }
  .step:nth-child(-n+2) { border-bottom: 1px solid var(--border); }
}
</style>
