/**
 * usePersona — global "active persona" state.
 *
 * The user picks a persona on /onboarding/tour. Their choice is then used
 * to filter the trading-app sidebar, hint marketing nav, and pre-select
 * the tour engine. Persists to localStorage so it survives reloads.
 *
 * Default = 'enterprise' (AI company) — the platform-first audience post-GTM-pivot.
 *
 *   const p = usePersona()
 *   p.persona.value          → 'trader' | 'enterprise' | 'partner'
 *   p.set('enterprise')
 *   p.belongsTo(['trader','enterprise'])  → boolean for the current persona
 */
export type ActivePersona = 'trader' | 'enterprise' | 'partner'

const STORAGE_KEY = 'exa:active-persona'
const VALID: ActivePersona[] = ['trader', 'enterprise', 'partner']

export interface PersonaMeta {
  id: ActivePersona
  name: string
  short: string
  who: string
}

export const PERSONA_META: Record<ActivePersona, PersonaMeta> = {
  trader:     { id: 'trader',     name: 'Jordan Park',    short: 'Trader',     who: 'Independent quant' },
  enterprise: { id: 'enterprise', name: 'Maya Chen',      short: 'AI Company', who: 'VP Engineering' },
  partner:    { id: 'partner',    name: 'Tom Reyes',      short: 'Datacenter', who: 'Capacity ops' },
}

export function usePersona() {
  // Default = 'enterprise' (AI company) — the platform-first audience post-GTM-pivot. Traders +
  // datacenter partners switch via the sidebar persona pill; the choice persists to localStorage.
  const persona = useState<ActivePersona>('active-persona', () => 'enterprise')

  // Hydrate from localStorage once on the client.
  if (import.meta.client) {
    const saved = (() => {
      try { return localStorage.getItem(STORAGE_KEY) as ActivePersona | null }
      catch { return null }
    })()
    if (saved && VALID.includes(saved) && saved !== persona.value) {
      persona.value = saved
    }
  }

  function set(p: ActivePersona) {
    if (!VALID.includes(p)) return
    persona.value = p
    if (import.meta.client) {
      try { localStorage.setItem(STORAGE_KEY, p) } catch { /* quota */ }
    }
  }

  function belongsTo(personas: ActivePersona[] | 'all'): boolean {
    if (personas === 'all') return true
    return personas.includes(persona.value)
  }

  const meta = computed<PersonaMeta>(() => PERSONA_META[persona.value])

  return { persona, meta, set, belongsTo }
}
