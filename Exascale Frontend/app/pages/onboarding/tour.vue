<script setup lang="ts">
/**
 * /onboarding/tour — Persona picker that kicks off the guided product tour.
 *
 * Light, layout-less landing. User picks one of four personas → tour starts
 * and we router.push() the first step's route. The GuidedTour runner mounted
 * in the layout takes over from there.
 */
import { personas, type PersonaId } from '~/data/tour-scripts'
import { ChevronRight, Play, SkipForward, MousePointerClick, Film } from 'lucide-vue-next'
import type { Persona as VPersona } from '~/composables/useTour'

definePageMeta({ layout: false })
useHead({ title: 'Product tour — Exascale', htmlAttrs: { 'data-theme': 'light' } })

const guided = useGuidedTour()  // old engine — click-through, per-step feedback
const video  = useTour()        // new engine — autoplay, end-of-tour feedback
const personaCx = usePersona()
const router = useRouter()

// Default-select the active persona so first-time visitors land on Trader
// (the default) and returning users land on whatever they last picked.
const initialId = ((): PersonaId => {
  const active = personaCx.persona.value
  if (active === 'enterprise') return 'enterprise'
  if (active === 'partner')    return 'datacenter'
  return 'trader'
})()

const selected = ref<PersonaId | null>(initialId)

/**
 * Tour mode:
 *   'click'  — you click "Next" yourself, leave feedback after relevant steps.
 *   'video'  — sit back; the tour navigates and narrates itself on a timer.
 */
type Mode = 'click' | 'video'
const mode = ref<Mode>('click')

/** Map legacy PersonaId → active-persona ids used app-wide. */
function toActivePersona(id: PersonaId): 'trader' | 'enterprise' | 'partner' {
  if (id === 'datacenter') return 'partner'
  if (id === 'lab')        return 'enterprise'
  if (id === 'enterprise') return 'enterprise'
  return 'trader'
}

function pick(id: PersonaId) {
  selected.value = id
  // Selecting a persona immediately updates the global active-persona so
  // the rest of the app (sidebar, future nav hints) starts filtering.
  personaCx.set(toActivePersona(id))
}

/**
 * Map legacy PersonaId (data/tour-scripts.ts: trader|enterprise|datacenter|lab)
 * to the autoplay engine's Persona type (trader|enterprise|partner).
 */
function toVideoPersona(id: PersonaId): VPersona {
  if (id === 'datacenter') return 'partner'
  if (id === 'lab')        return 'enterprise'
  if (id === 'enterprise') return 'enterprise'
  return 'trader'
}

function start() {
  if (!selected.value) return
  if (mode.value === 'video') {
    // <AppTour /> in app/app.vue catches this and takes over the screen.
    video.openAs(toVideoPersona(selected.value))
    return
  }
  // Click-through path — old engine, AppGuidedTour driver.js runner picks it up.
  guided.start(selected.value)
  const first = guided.script.value[0]
  if (first && first.route !== router.currentRoute.value.path) {
    router.push(first.route)
  }
}

// Skip the tour → land on the persona's product (AI company → console, datacenter → dashboard,
// trader → KYC gate), never the hardcoded exchange.
function skip() {
  router.push(personaCx.postOnboard.value)
}
</script>

<template>
  <div class="page">
    <header class="head">
      <NuxtLink to="/" class="brand mono">EXASCALE</NuxtLink>
      <button class="skip-btn" @click="skip">
        Skip tour <SkipForward :size="13" :stroke-width="1.7" />
      </button>
    </header>

    <main class="main">
      <section class="intro">
        <span class="caps eyebrow">Onboarding · Product tour</span>
        <h1>Tell us who you are.</h1>
        <p class="lede">
          The product surface is large. We'll show you the corner of it that matters to you —
          about <strong>90 seconds</strong>, walking through real pages with live mock data.
          Drop a quick rating on any step so we know what to keep, sharpen, or cut.
        </p>
      </section>

      <section class="picker">
        <ul class="persona-list">
          <li
            v-for="p in personas"
            :key="p.id"
            class="persona"
            :class="{ selected: selected === p.id }"
            tabindex="0"
            @click="pick(p.id)"
            @keydown.enter="pick(p.id)"
            @keydown.space.prevent="pick(p.id)"
          >
            <div class="glyph mono">{{ p.glyph }}</div>
            <div class="text">
              <div class="row">
                <h3>{{ p.name }}</h3>
                <span class="caps est mono">{{ p.estimate }}</span>
              </div>
              <p class="pitch">{{ p.pitch }}</p>
              <p class="blurb">{{ p.blurb }}</p>
            </div>
            <div class="check">
              <span class="radio" :class="{ on: selected === p.id }">
                <span v-if="selected === p.id" class="dot" />
              </span>
            </div>
          </li>
        </ul>
      </section>

      <!-- Mode toggle — click-through vs autoplay -->
      <section class="modes" aria-label="Tour mode">
        <span class="caps mono modes-label">How do you want to watch?</span>
        <div class="mode-grid">
          <button
            type="button"
            class="mode"
            :class="{ on: mode === 'click' }"
            :aria-pressed="mode === 'click'"
            @click="mode = 'click'"
          >
            <span class="mode-icon"><MousePointerClick :size="16" :stroke-width="1.7" /></span>
            <span class="mode-text">
              <span class="mode-name">Click-through</span>
              <span class="mode-blurb">You control the pace. Press Next after each step and leave a quick rating on the way.</span>
            </span>
            <span class="mode-radio" :class="{ on: mode === 'click' }">
              <span v-if="mode === 'click'" class="dot" />
            </span>
          </button>
          <button
            type="button"
            class="mode"
            :class="{ on: mode === 'video' }"
            :aria-pressed="mode === 'video'"
            @click="mode = 'video'"
          >
            <span class="mode-icon"><Film :size="16" :stroke-width="1.7" /></span>
            <span class="mode-text">
              <span class="mode-name">Auto-play <span class="caps mono mode-tag">video</span></span>
              <span class="mode-blurb">Sit back. The tour navigates + narrates itself on a timer. Pause / scrub / skip from a bottom bar. Rate it once at the end.</span>
            </span>
            <span class="mode-radio" :class="{ on: mode === 'video' }">
              <span v-if="mode === 'video'" class="dot" />
            </span>
          </button>
        </div>
      </section>

      <footer class="cta-bar">
        <div class="cta-meta">
          <span v-if="selected" class="caps mono">
            Selected: {{ personas.find(p => p.id === selected)?.name }}
            <span class="cta-mode">· {{ mode === 'video' ? 'auto-play' : 'click-through' }}</span>
          </span>
          <span v-else class="caps mono empty">Pick a persona to continue</span>
        </div>
        <button class="btn-primary" :disabled="!selected" @click="start">
          <Play :size="14" :stroke-width="1.7" :fill="selected ? 'currentColor' : 'none'" />
          Start tour <ChevronRight :size="14" :stroke-width="1.7" />
        </button>
      </footer>

      <p class="fine">
        Tip: you can re-run the tour any time from <NuxtLink to="/settings">Settings → Display</NuxtLink>.
        Press <kbd>Esc</kbd> mid-tour to exit.
      </p>
    </main>
  </div>
</template>

<style scoped>
.page {
  background: var(--surface-canvas, #F8F7F4);
  color: var(--text-primary, #0A0B0E);
  min-height: 100vh;
  font-family: 'Inter', system-ui, sans-serif;
  font-feature-settings: 'tnum';
  display: flex;
  flex-direction: column;
}

.mono { font-family: 'JetBrains Mono', ui-monospace, monospace; font-variant-numeric: tabular-nums; }
.caps { font-size: 10px; letter-spacing: 0.08em; text-transform: uppercase; font-weight: 600; color: var(--text-tertiary, #9A9A95); }

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 32px;
  border-bottom: 1px solid rgba(0,0,0,0.06);
}
.brand {
  text-decoration: none;
  color: var(--text-primary);
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.04em;
}
.skip-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: 1px solid rgba(0,0,0,0.12);
  border-radius: 4px;
  padding: 6px 12px;
  font-size: 12px;
  color: var(--text-secondary, #5F5F5C);
  cursor: pointer;
  font-family: inherit;
}
.skip-btn:hover { background: rgba(0,0,0,0.04); color: var(--text-primary); }

.main {
  flex: 1;
  width: 100%;
  max-width: 880px;
  margin: 0 auto;
  padding: 48px 32px 64px;
}

.intro { margin-bottom: 32px; }
.eyebrow { display: inline-block; margin-bottom: 12px; }
.intro h1 {
  margin: 0 0 12px;
  font-size: 36px;
  font-weight: 600;
  letter-spacing: -0.015em;
}
.lede {
  margin: 0;
  font-size: 16px;
  line-height: 1.55;
  color: var(--text-secondary, #5F5F5C);
  max-width: 640px;
}
.lede strong { color: var(--text-primary); font-weight: 600; }

.persona-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
@media (max-width: 720px) { .persona-list { grid-template-columns: 1fr; } }

.persona {
  display: grid;
  grid-template-columns: 56px 1fr 24px;
  gap: 16px;
  align-items: center;
  padding: 18px;
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 120ms ease, transform 120ms ease, box-shadow 120ms ease;
}
.persona:hover {
  border-color: rgba(0,0,0,0.18);
  transform: translateY(-1px);
}
.persona.selected {
  border-color: var(--text-primary);
  box-shadow: 0 4px 16px rgba(0,0,0,0.06);
  background: #FCFBF8;
}
.persona:focus-visible {
  outline: 2px solid #4A90E2;
  outline-offset: 2px;
}

.glyph {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #F0EFEC;
  border-radius: 4px;
  font-size: 28px;
  color: var(--text-primary);
  font-weight: 400;
}
.persona.selected .glyph { background: #0A0B0E; color: #C8F25C; }

.text { min-width: 0; }
.row { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; }
.text h3 { margin: 0; font-size: 17px; font-weight: 600; letter-spacing: -0.005em; }
.est { color: var(--text-tertiary); }

.pitch { margin: 6px 0 4px; font-size: 13px; color: var(--text-primary); font-weight: 500; }
.blurb { margin: 0; font-size: 12px; color: var(--text-secondary); line-height: 1.5; }

.check { display: flex; justify-content: flex-end; }
.radio {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 1.5px solid rgba(0,0,0,0.20);
  display: flex;
  align-items: center;
  justify-content: center;
}
.radio.on { border-color: var(--text-primary); }
.radio .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-primary); }

/* Mode toggle (click-through vs autoplay) */
.modes { margin-top: 28px; }
.modes-label { display: block; margin-bottom: 10px; }
.mode-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
@media (max-width: 720px) { .mode-grid { grid-template-columns: 1fr; } }

.mode {
  display: grid;
  grid-template-columns: 24px 1fr 20px;
  gap: 14px;
  align-items: flex-start;
  text-align: left;
  padding: 14px 16px;
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.10);
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: border-color 120ms ease, background-color 120ms ease;
}
.mode:hover { border-color: rgba(0,0,0,0.22); }
.mode.on    { border-color: var(--text-primary, #0A0B0E); background: #FCFBF8; }
.mode-icon  { color: var(--text-secondary, #5F5F5C); padding-top: 2px; }
.mode.on .mode-icon { color: var(--text-primary, #0A0B0E); }

.mode-text { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.mode-name {
  font-size: 13px; font-weight: 600;
  color: var(--text-primary, #0A0B0E);
  letter-spacing: -0.005em;
  display: inline-flex; align-items: center; gap: 6px;
}
.mode-tag {
  background: var(--text-primary, #0A0B0E);
  color: var(--brand-primary, #C8F25C);
  padding: 1px 5px;
  border-radius: 2px;
  font-size: 8.5px;
  letter-spacing: 0.14em;
}
.mode-blurb { font-size: 11.5px; color: var(--text-secondary, #5F5F5C); line-height: 1.5; }

.mode-radio {
  width: 18px; height: 18px;
  border-radius: 50%;
  border: 1.5px solid rgba(0,0,0,0.20);
  display: flex; align-items: center; justify-content: center;
  align-self: center;
}
.mode-radio.on    { border-color: var(--text-primary); }
.mode-radio .dot  { width: 8px; height: 8px; border-radius: 50%; background: var(--text-primary); }

.cta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid rgba(0,0,0,0.08);
}
.cta-meta .empty { color: var(--text-tertiary); }
.cta-mode { color: var(--text-tertiary, #9A9A95); margin-left: 4px; }

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 40px;
  padding: 0 18px;
  font-size: 14px;
  font-weight: 600;
  background: var(--text-primary);
  color: var(--brand-primary, #C8F25C);
  border: 1px solid var(--text-primary);
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: background 100ms ease;
}
.btn-primary:hover:not(:disabled) { background: #1C1F26; }
.btn-primary:disabled {
  background: rgba(0,0,0,0.12);
  color: var(--text-tertiary);
  border-color: transparent;
  cursor: not-allowed;
}

.fine {
  margin-top: 32px;
  font-size: 12px;
  color: var(--text-tertiary);
}
.fine a { color: var(--text-secondary); text-decoration: underline; text-underline-offset: 2px; }
.fine kbd {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(0,0,0,0.04);
  border: 1px solid rgba(0,0,0,0.12);
}
</style>
