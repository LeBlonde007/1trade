/**
 * useGuidedTour — global state machine for the persona-driven tour.
 *
 *   const tour = useGuidedTour()
 *   tour.start('trader')          → kicks off, routes to step 0
 *   tour.next() / tour.prev()
 *   tour.stop()
 *
 * State is held in `useState` (SSR/SPA-safe) AND mirrored to localStorage
 * so a route navigation in the middle of a step survives a full reload.
 *
 * The actual driver.js popover rendering lives in App/GuidedTour.client.vue,
 * which watches `currentStep` and arms / re-arms driver.js as needed.
 */
import { tours, type PersonaId, type TourStep } from '~/data/tour-scripts'

const STORAGE_KEY = 'exa:guided-tour:v2'

interface PersistedState {
  active: boolean
  persona: PersonaId | null
  index: number
  sessionId: string | null
}

const EMPTY: PersistedState = { active: false, persona: null, index: 0, sessionId: null }

function loadPersisted(): PersistedState {
  if (typeof localStorage === 'undefined') return EMPTY
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return EMPTY
    const parsed = JSON.parse(raw) as PersistedState
    // Validate: persona must be a known key, index within bounds.
    if (parsed.active && parsed.persona && tours[parsed.persona]) {
      const max = tours[parsed.persona].length
      if (parsed.index < 0 || parsed.index >= max) return EMPTY
      return parsed
    }
    return EMPTY
  } catch {
    return EMPTY
  }
}

function savePersisted(s: PersistedState) {
  if (typeof localStorage === 'undefined') return
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(s)) } catch { /* quota */ }
}

function newSessionId() {
  return 'tour_' + Math.random().toString(36).slice(2, 10) + Date.now().toString(36)
}

export function useGuidedTour() {
  const active    = useState<boolean>('gt-active',    () => false)
  const persona   = useState<PersonaId | null>('gt-persona', () => null)
  const index     = useState<number>('gt-index',      () => 0)
  const sessionId = useState<string | null>('gt-sessionId', () => null)

  // Hydrate from localStorage on first call (client only)
  if (import.meta.client && !active.value && persona.value === null) {
    const s = loadPersisted()
    if (s.active && s.persona) {
      active.value    = s.active
      persona.value   = s.persona
      index.value     = s.index
      sessionId.value = s.sessionId
    }
  }

  const script = computed<TourStep[]>(() => persona.value ? tours[persona.value] : [])
  const currentStep = computed<TourStep | null>(() => script.value[index.value] ?? null)
  const total = computed(() => script.value.length)
  const isLast = computed(() => index.value >= total.value - 1)

  function persist() {
    savePersisted({
      active:    active.value,
      persona:   persona.value,
      index:     index.value,
      sessionId: sessionId.value,
    })
  }

  function start(p: PersonaId) {
    persona.value   = p
    index.value     = 0
    active.value    = true
    sessionId.value = newSessionId()
    persist()
  }

  function next() {
    if (!active.value) return
    if (isLast.value) { stop(); return }
    index.value = index.value + 1
    persist()
  }

  function prev() {
    if (!active.value) return
    index.value = Math.max(0, index.value - 1)
    persist()
  }

  function goto(n: number) {
    if (!active.value) return
    index.value = Math.max(0, Math.min(total.value - 1, n))
    persist()
  }

  function stop() {
    active.value    = false
    persona.value   = null
    index.value     = 0
    sessionId.value = null
    persist()
  }

  async function submitFeedback(body: string) {
    if (!currentStep.value || !sessionId.value || !persona.value) return false
    try {
      await $fetch('/api/feedback', {
        method: 'POST',
        body: {
          sessionId: sessionId.value,
          persona:   persona.value,
          stepId:    currentStep.value.id,
          screen:    currentStep.value.route,
          section:   currentStep.value.section,
          feedback:  body.trim(),
          ts:        new Date().toISOString(),
          userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
        },
      })
      return true
    } catch (err) {
      console.error('[guided-tour] feedback POST failed', err)
      return false
    }
  }

  return {
    active,
    persona,
    index,
    sessionId,
    script,
    currentStep,
    total,
    isLast,
    start,
    next,
    prev,
    goto,
    stop,
    submitFeedback,
  }
}
