/**
 * usePersona — the product surface the app shows (sidebar, landing page, trader chrome).
 *
 * When signed in it is the account's type from /v1/auth/me (platform-core v1.11) — chosen at signup
 * and stored server-side, so it is the same on every device: trader → trader, AI company and
 * enterprise → enterprise, datacenter → partner. Signed out (marketing pages, the tour) it falls back
 * to a local choice kept in localStorage.
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

/** The persona each account type sees. */
const ACCOUNT_PERSONA: Record<string, ActivePersona> = {
  trader: 'trader', ai_company: 'enterprise', enterprise: 'enterprise', datacenter: 'partner',
}
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
  // The local choice (signed-out fallback). Default = 'enterprise' (AI company), the platform-first
  // audience.
  const local = useState<ActivePersona>('active-persona', () => 'enterprise')
  const auth = useAuth()
  // The signed-in account's type wins; it cannot be switched from the UI.
  const persona = computed<ActivePersona>({
    get: () => ACCOUNT_PERSONA[auth.user.value?.account_type ?? ''] ?? local.value,
    set: (p) => { local.value = p },
  })

  // Hydrate from localStorage once on the client.
  if (import.meta.client) {
    const saved = (() => {
      try { return localStorage.getItem(STORAGE_KEY) as ActivePersona | null }
      catch { return null }
    })()
    if (saved && VALID.includes(saved) && saved !== local.value) {
      local.value = saved
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

  // home — where this persona's product lives. AI company → the console (live product); datacenter
  // → the supply dashboard; trader → the paper exchange. Used by onboarding + nav to land users
  // on the right surface instead of hardcoding `/trade` everywhere.
  const home = computed(() => ({ trader: '/trade', enterprise: '/console', partner: '/datacenter' }[persona.value]))

  // postOnboard — where to go after the welcome + tour: every persona's home. Paper trading needs no
  // KYC; the KYC gate applies to real-money purchases, which ask for it when they happen.
  const postOnboard = computed(() => home.value)

  return { persona, meta, set, belongsTo, home, postOnboard }
}
