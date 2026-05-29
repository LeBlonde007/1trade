/**
 * useToasts — global ephemeral toast queue (N3).
 *
 *   const t = useToasts()
 *   t.push({ tone: 'pos', title: 'Order filled', body: '5,000 EAI-IDX @ $0.001005' })
 *
 * Auto-dismiss 4.5s; hover to pause (handled in the AppToasts region).
 */
export type ToastTone = 'pos' | 'neg' | 'warn' | 'info'

export interface Toast {
  id: number
  tone: ToastTone
  title: string
  body?: string
  href?: string         // optional: click toast to navigate
  hrefLabel?: string    // CTA label inside toast
  createdAt: number
}

let nextId = 1

export function useToasts() {
  const toasts = useState<Toast[]>('toasts', () => [])

  function push(t: Omit<Toast, 'id' | 'createdAt'>): number {
    const id = nextId++
    toasts.value = [...toasts.value, { ...t, id, createdAt: Date.now() }]
    return id
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter((x) => x.id !== id)
  }

  function clear() {
    toasts.value = []
  }

  return { toasts, push, dismiss, clear }
}
