<script setup lang="ts">
/**
 * App/Tour — persona-driven autoplay product tour (L6, v2).
 *
 * Three phases:
 *   1. picker    — choose Trader / Enterprise / Partner.
 *   2. playing   — auto-advance through that persona's scripted journey
 *                  with spotlight + tooltip + video-style player.
 *   3. feedback  — rate 1-5 + leave a comment.
 *
 * Driven by useTour().
 */
import {
  X, Play, Pause, SkipForward, SkipBack, ChevronRight, Star, Briefcase, User as UserIcon, Building2,
} from 'lucide-vue-next'
import { TOUR_PERSONAS } from '~/composables/useTour'

const tour = useTour()
const personas = TOUR_PERSONAS
const router = useRouter()
const route  = useRoute()

// ─── Spotlight rect ──────────────────────────────────────────
interface Rect { top: number; left: number; width: number; height: number }
const rect = ref<Rect | null>(null)

async function locate() {
  if (tour.phase.value !== 'playing' || !tour.currentStep.value) {
    rect.value = null
    return
  }
  // Wait a frame so route nav can settle
  await nextTick()
  const sels = tour.currentStep.value.target.split(',').map((s) => s.trim()).filter(Boolean)
  let el: HTMLElement | null = null
  for (const s of sels) {
    el = document.querySelector(s) as HTMLElement | null
    if (el) break
  }
  if (!el) {
    rect.value = null
    return
  }
  const r = el.getBoundingClientRect()
  rect.value = {
    top: r.top,
    left: r.left,
    width: r.width,
    height: r.height,
  }
}

// ─── Auto-advance progress ───────────────────────────────────
const TICK_MS = 60
let interval: ReturnType<typeof setInterval> | null = null

function startTicker() {
  if (interval) return
  interval = setInterval(() => {
    if (tour.phase.value !== 'playing' || !tour.playing.value || !tour.currentStep.value) return
    const dur = tour.currentStep.value.ms
    tour.progress.value = Math.min(1, tour.progress.value + TICK_MS / dur)
    if (tour.progress.value >= 1) {
      tour.next()
    }
  }, TICK_MS)
}
function stopTicker() {
  if (interval) { clearInterval(interval); interval = null }
}

// ─── Route navigation on step change ─────────────────────────
async function navigateForStep() {
  const s = tour.currentStep.value
  if (!s) return
  if (route.path !== s.route) {
    await router.push(s.route)
  }
  await locate()
}

watch(() => tour.stepIdx.value, navigateForStep)
watch(() => tour.phase.value, async (p) => {
  if (p === 'playing') {
    await navigateForStep()
  } else {
    rect.value = null
  }
})

// Re-locate on resize / scroll while a step is showing
function onResize() { locate() }
function onScroll() { locate() }

// ─── Keyboard ────────────────────────────────────────────────
function onKey(e: KeyboardEvent) {
  if (!tour.isOpen.value) return
  if (e.key === 'Escape')     tour.close()
  if (tour.phase.value !== 'playing') return
  if (e.key === ' ')          { e.preventDefault(); tour.toggle() }
  if (e.key === 'ArrowRight') tour.next()
  if (e.key === 'ArrowLeft')  tour.prev()
}

onMounted(() => {
  startTicker()
  document.addEventListener('keydown', onKey)
  window.addEventListener('resize', onResize)
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => {
  stopTicker()
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', onResize)
  window.removeEventListener('scroll', onScroll)
})

// ─── Tooltip position relative to spotlight rect ─────────────
const tooltipStyle = computed(() => {
  if (!rect.value || !tour.currentStep.value) return { display: 'none' as const }
  const r = rect.value
  const gap = 16
  const tipW = 360, tipH = 160
  const vw = typeof window !== 'undefined' ? window.innerWidth  : 1440
  const vh = typeof window !== 'undefined' ? window.innerHeight : 900
  const placement = tour.currentStep.value.placement
  // Smart default: if no placement, prefer right if room, else bottom
  let p: 'right' | 'left' | 'top' | 'bottom' = placement ?? 'right'
  if (!placement) {
    if (r.left + r.width + gap + tipW > vw - 20) p = 'left'
    if (p === 'left' && r.left - gap - tipW < 20) p = 'bottom'
  }

  let top = 0, left = 0
  if (p === 'right')  { top = r.top + r.height / 2 - tipH / 2; left = r.left + r.width + gap }
  if (p === 'left')   { top = r.top + r.height / 2 - tipH / 2; left = r.left - tipW - gap   }
  if (p === 'top')    { top = r.top - tipH - gap;              left = r.left + r.width / 2 - tipW / 2 }
  if (p === 'bottom') { top = r.top + r.height + gap;          left = r.left + r.width / 2 - tipW / 2 }

  top  = Math.max(12, Math.min(vh - tipH - 110, top))
  left = Math.max(12, Math.min(vw - tipW - 12, left))

  return { top: top + 'px', left: left + 'px' }
})

const spotStyle = computed(() => {
  if (!rect.value) return { display: 'none' as const }
  const r = rect.value
  const pad = 6
  return {
    top:    (r.top - pad) + 'px',
    left:   (r.left - pad) + 'px',
    width:  (r.width  + pad * 2) + 'px',
    height: (r.height + pad * 2) + 'px',
  }
})

// ─── Step + journey UI helpers ───────────────────────────────
const journeyProgressPct = computed(() => {
  if (!tour.totalSteps.value) return 0
  return ((tour.stepIdx.value + tour.progress.value) / tour.totalSteps.value) * 100
})

const personaIcon = {
  trader:     UserIcon,
  enterprise: Briefcase,
  partner:    Building2,
}

// Feedback star hover
const hoverStar = ref(0)
function setRating(n: number)  { tour.fbRating.value = n }
function hoverRating(n: number){ hoverStar.value = n }
</script>

<template>
  <Teleport to="body">
    <Transition name="tour-fade">
      <div v-if="tour.isOpen.value" class="tour-root" role="dialog" aria-modal="true" aria-label="Product tour">

        <!-- ============================================================
             PHASE 1 — Persona picker
             ============================================================ -->
        <div v-if="tour.phase.value === 'picker'" class="picker-shell">
          <div class="picker-card">
            <header class="pk-head">
              <div class="pk-eyebrow">1TRADE · GUIDED TOUR</div>
              <h2 class="pk-title">Pick the experience you want to see.</h2>
              <p class="pk-sub">Three pre-scripted journeys. The tour drives itself — sit back, watch, leave feedback at the end.</p>
              <button class="pk-close" aria-label="Close tour" type="button" @click="tour.close()">
                <X :size="18" />
              </button>
            </header>

            <div class="pk-grid">
              <button
                v-for="p in personas"
                :key="p.id"
                type="button"
                class="pk-tile"
                @click="tour.selectPersona(p.id)"
              >
                <div class="pk-tile-icon" :style="{ color: p.color }">
                  <component :is="personaIcon[p.id]" :size="18" :stroke-width="1.6" />
                </div>
                <div class="pk-tile-body">
                  <div class="pk-tile-name">{{ p.name }}</div>
                  <div class="pk-tile-who">{{ p.who }}</div>
                  <div class="pk-tile-meta">
                    <span class="mono">{{ p.steps.length }} stops</span>
                    <span class="dot-sep">·</span>
                    <span class="mono">{{ p.runtime }}</span>
                  </div>
                </div>
                <ChevronRight :size="14" class="pk-tile-arrow" />
              </button>
            </div>

            <footer class="pk-foot">
              <button type="button" class="pk-link" @click="tour.close()">No thanks — I'll explore on my own →</button>
            </footer>
          </div>
        </div>

        <!-- ============================================================
             PHASE 2 — Playing
             ============================================================ -->
        <template v-if="tour.phase.value === 'playing'">
          <!-- Spotlight backdrop (4-piece cutout) -->
          <template v-if="rect">
            <div class="bd top"    :style="{ height: Math.max(0, rect.top - 6) + 'px' }" />
            <div class="bd bottom" :style="{ top: (rect.top + rect.height + 6) + 'px' }" />
            <div class="bd left"   :style="{ top: (rect.top - 6) + 'px', height: (rect.height + 12) + 'px', width: (rect.left - 6) + 'px' }" />
            <div class="bd right"  :style="{ top: (rect.top - 6) + 'px', height: (rect.height + 12) + 'px', left: (rect.left + rect.width + 6) + 'px' }" />
            <div class="spot" :style="spotStyle" />
          </template>
          <div v-else class="bd full" />

          <!-- Tooltip narration -->
          <div v-if="tour.currentStep.value" class="step-card" :style="tooltipStyle">
            <div class="sc-head">
              <span class="sc-step mono">{{ tour.stepIdx.value + 1 }} / {{ tour.totalSteps.value }}</span>
              <span v-if="tour.persona.value" class="sc-persona mono">{{ tour.persona.value.name.toUpperCase() }}</span>
            </div>
            <h3 class="sc-title">{{ tour.currentStep.value.title }}</h3>
            <p class="sc-body">{{ tour.currentStep.value.body }}</p>
            <div class="sc-progress">
              <div class="sc-progress-fill" :style="{ width: (tour.progress.value * 100) + '%' }" />
            </div>
          </div>

          <!-- Bottom video-player bar -->
          <div class="player">
            <div class="player-bg" />
            <div class="player-inner">
              <div class="player-meta">
                <div class="player-persona">
                  <span class="pp-name">{{ tour.persona.value?.name }}</span>
                  <span class="pp-who">{{ tour.persona.value?.who }}</span>
                </div>
                <div class="player-step mono">
                  {{ String(tour.stepIdx.value + 1).padStart(2, '0') }} · {{ tour.currentStep.value?.title }}
                </div>
              </div>

              <div class="player-bar-row">
                <button class="pb-btn" aria-label="Previous step" :disabled="tour.stepIdx.value === 0" type="button" @click="tour.prev()">
                  <SkipBack :size="14" />
                </button>
                <button class="pb-btn primary" :aria-label="tour.playing.value ? 'Pause' : 'Play'" type="button" @click="tour.toggle()">
                  <Pause v-if="tour.playing.value" :size="14" />
                  <Play  v-else :size="14" />
                </button>
                <button class="pb-btn" aria-label="Next step" type="button" @click="tour.next()">
                  <SkipForward :size="14" />
                </button>

                <div class="pb-progress" @click.self="(e) => {
                  const t = e.currentTarget
                  if (!(t instanceof HTMLElement)) return
                  const x = e.offsetX
                  const pct = Math.max(0, Math.min(1, x / t.offsetWidth))
                  tour.goto(Math.floor(pct * tour.totalSteps.value))
                }">
                  <div class="pb-fill" :style="{ width: journeyProgressPct + '%' }" />
                  <div v-for="(_, i) in tour.persona.value?.steps ?? []" :key="i" class="pb-tick" :style="{ left: ((i + 1) / tour.totalSteps.value * 100) + '%' }" />
                </div>

                <button class="pb-link" type="button" @click="tour.finish()">Skip to end</button>
                <button class="pb-link" type="button" @click="tour.close()" aria-label="Close tour">
                  <X :size="14" />
                </button>
              </div>
            </div>
          </div>
        </template>

        <!-- ============================================================
             PHASE 3 — Feedback
             ============================================================ -->
        <div v-if="tour.phase.value === 'feedback'" class="picker-shell">
          <div class="picker-card fb-card">
            <header class="pk-head">
              <div class="pk-eyebrow">TOUR COMPLETE</div>
              <h2 class="pk-title">How did this feel?</h2>
              <p class="pk-sub">Your rating goes to the product team. Be honest — this is a demo, we're not investors yet.</p>
              <button class="pk-close" aria-label="Close" type="button" @click="tour.close()">
                <X :size="18" />
              </button>
            </header>

            <div class="fb-stars" @mouseleave="hoverStar = 0">
              <button
                v-for="n in 5"
                :key="n"
                type="button"
                class="fb-star"
                :class="{ on: (hoverStar || tour.fbRating.value) >= n }"
                :aria-label="n + ' star' + (n > 1 ? 's' : '')"
                @click="setRating(n)"
                @mouseenter="hoverRating(n)"
              >
                <Star :size="28" :fill="(hoverStar || tour.fbRating.value) >= n ? 'currentColor' : 'transparent'" :stroke-width="1.5" />
              </button>
            </div>
            <div class="fb-rating-label">
              <span v-if="tour.fbRating.value === 0">Tap a star</span>
              <span v-else>
                <span class="mono">{{ tour.fbRating.value }} / 5</span>
                — {{ ['Painful', 'Meh', 'Okay', 'Solid', 'Exceptional'][tour.fbRating.value - 1] }}
              </span>
            </div>

            <textarea
              v-model="tour.fbComment.value"
              class="fb-comment"
              rows="3"
              placeholder="What was confusing? What landed? (optional)"
            />

            <footer class="fb-foot">
              <button type="button" class="pk-link" @click="tour.open()">↺ Try another persona</button>
              <button type="button" class="btn primary" :disabled="tour.fbRating.value === 0" @click="tour.submitFeedback()">
                Submit &amp; close
              </button>
            </footer>
          </div>
        </div>

      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.tour-root {
  position: fixed; inset: 0;
  z-index: 2000;
  font-family: var(--font-sans);
  color: var(--text);
}
.tour-root .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.tour-root .dot-sep { color: var(--text-3); }

/* Backdrop pieces */
.bd {
  position: absolute;
  background: rgba(5, 6, 8, 0.78);
  pointer-events: auto;
  transition: top 220ms cubic-bezier(0.4,0,0.2,1), left 220ms cubic-bezier(0.4,0,0.2,1), height 220ms cubic-bezier(0.4,0,0.2,1), width 220ms cubic-bezier(0.4,0,0.2,1);
}
.bd.top    { top: 0; left: 0; right: 0; }
.bd.bottom { left: 0; right: 0; bottom: 0; }
.bd.left   { left: 0; }
.bd.right  { right: 0; }
.bd.full   { inset: 0; }

.spot {
  position: absolute;
  border: 2px solid var(--brand);
  border-radius: var(--radius-sm);
  box-shadow: 0 0 0 6px rgba(200, 242, 92, 0.16), 0 0 28px rgba(200, 242, 92, 0.28);
  pointer-events: none;
  transition: top 240ms cubic-bezier(0.4,0,0.2,1), left 240ms cubic-bezier(0.4,0,0.2,1), width 240ms cubic-bezier(0.4,0,0.2,1), height 240ms cubic-bezier(0.4,0,0.2,1);
}

/* ============================================================
   Step tooltip
   ============================================================ */
.step-card {
  position: absolute;
  width: 360px;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 16px 18px 14px;
  box-shadow: 0 16px 40px rgba(0,0,0,0.55);
  transition: top 240ms cubic-bezier(0.4,0,0.2,1), left 240ms cubic-bezier(0.4,0,0.2,1);
  pointer-events: auto;
}
.sc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.sc-step    { font-size: 10px; letter-spacing: 0.14em; font-weight: 700; color: var(--text-3); }
.sc-persona { font-size: 9.5px; letter-spacing: 0.14em; font-weight: 700; color: var(--text-2); }
.sc-title   { font-family: var(--font-display); font-size: 16px; font-weight: 600; margin: 0 0 6px; letter-spacing: -0.005em; }
.sc-body    { font-size: 12.5px; color: var(--text-2); line-height: 1.55; margin: 0 0 14px; }
.sc-progress {
  width: 100%; height: 2px;
  background: rgba(255,255,255,0.08);
  border-radius: 1px;
  overflow: hidden;
}
.sc-progress-fill {
  height: 100%;
  background: var(--brand);
  transition: width 80ms linear;
}

/* ============================================================
   Picker (Phase 1) + Feedback (Phase 3) shared shell
   ============================================================ */
.picker-shell {
  position: absolute; inset: 0;
  background: rgba(5, 6, 8, 0.78);
  display: flex; align-items: center; justify-content: center;
  padding: 32px;
  pointer-events: auto;
}
.picker-card {
  width: 100%;
  max-width: 560px;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 24px 56px rgba(0,0,0,0.6);
  padding: 28px 28px 22px;
}
.pk-head { position: relative; margin-bottom: 20px; }
.pk-eyebrow {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: 0.18em; color: var(--text-3);
  margin-bottom: 8px;
}
.pk-title {
  font-family: var(--font-display); font-size: 22px; font-weight: 600;
  letter-spacing: -0.015em; margin: 0 0 6px;
}
.pk-sub { color: var(--text-2); font-size: 13px; margin: 0; line-height: 1.55; }
.pk-close {
  position: absolute; top: -6px; right: -6px;
  width: 30px; height: 30px;
  background: transparent; color: var(--text-2);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.pk-close:hover { color: var(--text); border-color: var(--text); }

.pk-grid { display: flex; flex-direction: column; gap: 8px; margin-bottom: 18px; }
.pk-tile {
  display: grid;
  grid-template-columns: 36px 1fr 14px;
  gap: 14px;
  width: 100%;
  padding: 14px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color 140ms ease, background-color 140ms ease;
  font-family: var(--font-sans);
}
.pk-tile:hover {
  border-color: var(--border-strong);
  background: rgba(255,255,255,0.025);
}
.pk-tile-icon {
  width: 36px; height: 36px;
  background: rgba(255,255,255,0.04);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
}
.pk-tile-name { font-weight: 600; font-size: 14px; }
.pk-tile-who  { color: var(--text-2); font-size: 11.5px; margin-top: 2px; }
.pk-tile-meta { color: var(--text-3); font-size: 10.5px; margin-top: 6px; display: inline-flex; gap: 6px; }
.pk-tile-arrow { color: var(--text-3); align-self: center; }
.pk-tile:hover .pk-tile-arrow { color: var(--text); }

.pk-foot { display: flex; justify-content: center; }
.pk-link {
  background: transparent; border: none; padding: 0;
  color: var(--text-3); font-size: 12px; cursor: pointer;
  text-decoration: underline; text-underline-offset: 3px;
}
.pk-link:hover { color: var(--text); }

/* ============================================================
   Feedback card
   ============================================================ */
.fb-card { padding-bottom: 18px; }
.fb-stars {
  display: flex; justify-content: center; gap: 10px;
  margin: 14px 0 8px;
  color: var(--text-3);
}
.fb-star {
  background: transparent; border: none; padding: 6px;
  cursor: pointer; color: var(--text-3);
  transition: color 120ms ease, transform 120ms ease;
}
.fb-star.on { color: var(--brand); }
.fb-star:hover { transform: translateY(-1px); }
.fb-rating-label {
  text-align: center; color: var(--text-2); font-size: 12.5px;
  margin-bottom: 14px; min-height: 18px;
}
.fb-comment {
  width: 100%;
  background: var(--canvas, rgba(0,0,0,0.2));
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  color: var(--text); font-family: var(--font-sans); font-size: 12.5px;
  resize: vertical;
  outline: none;
  margin-bottom: 16px;
}
.fb-comment:focus { border-color: var(--accent); }
.fb-foot { display: flex; justify-content: space-between; align-items: center; }
.btn {
  padding: 8px 16px;
  font-family: var(--font-sans); font-size: 13px; font-weight: 600;
  background: transparent; color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.btn.primary { background: var(--brand); color: var(--text-on-accent); border-color: var(--brand); }
.btn.primary:hover:not(:disabled) { background: var(--brand-hov); }
.btn:disabled { opacity: 0.45; cursor: default; }

/* ============================================================
   Video-style player (Phase 2 bottom bar)
   ============================================================ */
.player {
  position: fixed;
  left: 50%; bottom: 24px;
  transform: translateX(-50%);
  width: min(720px, calc(100vw - 48px));
  pointer-events: auto;
}
.player-bg {
  position: absolute; inset: 0;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 20px 48px rgba(0,0,0,0.6);
}
.player-inner {
  position: relative;
  padding: 12px 16px 14px;
}
.player-meta { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 8px; gap: 12px; }
.pp-name { font-weight: 700; font-size: 12.5px; }
.pp-who  { color: var(--text-3); font-size: 11px; margin-left: 8px; }
.player-step { color: var(--text-2); font-size: 11px; letter-spacing: 0.06em; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.player-bar-row {
  display: flex; align-items: center; gap: 8px;
}
.pb-btn {
  width: 30px; height: 30px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-2);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.pb-btn:hover:not(:disabled) { color: var(--text); border-color: var(--text); }
.pb-btn:disabled { opacity: 0.4; cursor: default; }
.pb-btn.primary {
  background: var(--brand);
  color: var(--text-on-accent);
  border-color: var(--brand);
}
.pb-btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }

.pb-progress {
  position: relative;
  flex: 1;
  height: 6px;
  background: rgba(255,255,255,0.08);
  border-radius: 3px;
  cursor: pointer;
  overflow: hidden;
}
.pb-fill {
  height: 100%; background: var(--brand);
  transition: width 80ms linear;
}
.pb-tick {
  position: absolute;
  top: 0; bottom: 0;
  width: 1px;
  background: rgba(0,0,0,0.6);
  transform: translateX(-1px);
}

.pb-link {
  background: transparent; border: none;
  color: var(--text-3); font-size: 11.5px;
  cursor: pointer;
  padding: 0 4px;
  font-family: var(--font-sans);
}
.pb-link:hover { color: var(--text); }

/* Phase fade */
.tour-fade-enter-active, .tour-fade-leave-active { transition: opacity 180ms ease; }
.tour-fade-enter-from, .tour-fade-leave-to { opacity: 0; }
</style>
