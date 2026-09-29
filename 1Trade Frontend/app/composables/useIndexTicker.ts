/**
 * useIndexTicker — the AI Index as the public pages show it (signup, verify): the latest print, its
 * change over the loaded history, a normalised series for a mini chart, and a countdown to the next
 * publication. Everything comes from the index API via the BFF; nothing is generated in the browser.
 * Prints are simulated until the index is in production, and `simulated` says so.
 */
interface IndexPrint { value: string; published_at: string; source: 'mock' | 'computed' }

export function useIndexTicker(days = 30) {
  const value = ref<number | null>(null)
  const publishedAt = ref<string | null>(null)
  const nextAt = ref<string | null>(null)
  const series = ref<number[]>([])
  const simulated = ref(true)
  const countdown = ref('—')
  let timer: ReturnType<typeof setInterval> | null = null

  /** changePct is the move over the loaded history, first print to latest. */
  const changePct = computed(() => {
    const s = series.value
    return s.length > 1 && s[0]! > 0 ? ((s[s.length - 1]! - s[0]!) / s[0]!) * 100 : null
  })
  /** creditsPerUsd is the inverted quote the site uses: AI credits per $1. */
  const creditsPerUsd = computed(() => (value.value ? 1 / value.value : null))

  /** tick updates the countdown to the next publication. */
  function tick() {
    if (!nextAt.value) return
    const diff = Math.max(0, new Date(nextAt.value).getTime() - Date.now())
    const h = Math.floor(diff / 3.6e6), m = Math.floor((diff % 3.6e6) / 6e4)
    countdown.value = `${h}h ${String(m).padStart(2, '0')}m`
  }

  /** load fetches the latest print and the history; a failure leaves the dashes in place. */
  async function load() {
    try {
      const [l, h] = await Promise.all([
        $fetch<{ print: IndexPrint; next_publication_at: string; is_mock: boolean }>('/api/index-service/latest'),
        $fetch<{ prints: IndexPrint[] }>('/api/index-service/history', { query: { days } }),
      ])
      value.value = Number(l.print.value)
      publishedAt.value = l.print.published_at
      nextAt.value = l.next_publication_at
      simulated.value = l.is_mock || l.print.source === 'mock'
      series.value = h.prints.map(p => Number(p.value))
      tick()
    } catch {
      // keep the placeholders; the page stays usable without the index
    }
  }

  onMounted(() => {
    load()
    timer = setInterval(tick, 30_000)
  })
  onBeforeUnmount(() => { if (timer) clearInterval(timer) })

  return { value, creditsPerUsd, publishedAt, nextAt, series, simulated, countdown, changePct }
}

/** sparkPath turns a series into an SVG path (line, and the filled area under it) for a W×H box. */
export function sparkPath(pts: number[], W: number, H: number, pad = 6) {
  if (pts.length < 2) return { line: '', fill: '' }
  const min = Math.min(...pts), max = Math.max(...pts)
  const range = max - min || 1
  const stepX = W / (pts.length - 1)
  const line = pts.map((p, i) => {
    const y = pad + (H - pad * 2) - ((p - min) / range) * (H - pad * 2)
    return `${i === 0 ? 'M' : 'L'}${(i * stepX).toFixed(1)} ${y.toFixed(1)}`
  }).join(' ')
  return { line, fill: `${line} L ${W} ${H} L 0 ${H} Z` }
}
