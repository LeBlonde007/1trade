/**
 * useFieldHelp — collects "what / why" feedback from inline help dots.
 *
 *   const fh = useFieldHelp()
 *   fh.submit('Limit price', 'wording confusing', 'trade.order.limit-px')
 *
 * Each submission gets a sequential id, the current route, the field id
 * (if provided), and a timestamp. Persisted to localStorage so the team
 * can replay sessions during demos.
 *
 * A global toast fires on submit (via useToasts).
 */
export interface HelpFeedback {
  id: number
  title: string
  field?: string
  route: string
  text: string
  ts: number          // unix ms
}

const STORAGE_KEY = 'exa:help-feedback'

function loadPersisted(): HelpFeedback[] {
  if (typeof localStorage === 'undefined') return []
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function savePersisted(list: HelpFeedback[]) {
  if (typeof localStorage === 'undefined') return
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(-200))) }
  catch { /* quota — drop silently */ }
}

let nextId = 1

export function useFieldHelp() {
  const submissions = useState<HelpFeedback[]>('help-feedback', () => [])

  // Hydrate once on client mount.
  if (import.meta.client && submissions.value.length === 0) {
    const persisted = loadPersisted()
    if (persisted.length) {
      submissions.value = persisted
      nextId = Math.max(...persisted.map((p) => p.id), 0) + 1
    }
  }

  function submit(title: string, text: string, field?: string) {
    if (!text.trim()) return
    const route = (typeof window !== 'undefined' && window.location)
      ? window.location.pathname + window.location.hash
      : ''
    const entry: HelpFeedback = {
      id:    nextId++,
      title,
      field,
      route,
      text:  text.trim(),
      ts:    Date.now(),
    }
    submissions.value = [...submissions.value, entry]
    savePersisted(submissions.value)
    try {
      const t = useToasts()
      t.push({
        tone:  'info',
        title: 'Feedback recorded',
        body:  '"' + entry.text.slice(0, 64) + (entry.text.length > 64 ? '…' : '') + '" — on ' + title,
      })
    } catch {
      // useToasts may not be available in some contexts
    }
  }

  function clear() {
    submissions.value = []
    savePersisted([])
    nextId = 1
  }

  return { submissions, submit, clear }
}
