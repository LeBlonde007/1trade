/**
 * useInference — run a chat completion via the BFF (`/api/inference/chat`) and expose the result +
 * token usage + a 402 (insufficient credit) flag so the playground can prompt the user to buy.
 */
export interface ChatUsage { prompt_tokens: number; completion_tokens: number; total_tokens: number }
export interface ChatResult { content: string; usage: ChatUsage; model: string }

interface ChatResponse {
  model: string
  choices: Array<{ message: { content: string } }>
  usage: ChatUsage
}

export function useInference() {
  const running = useState('inf:running', () => false)
  const error = useState('inf:error', () => '')
  const insufficientCredit = useState('inf:402', () => false)
  const result = useState<ChatResult | null>('inf:result', () => null)

  /**
   * runStream sends a single-turn prompt and streams the completion token-by-token (SSE via the BFF).
   * `onToken` fires for each delta so the UI can render live; the resolved value carries the full text
   * plus usage when the gateway includes it. A 402 is reflected on `insufficientCredit` (and thrown)
   * BEFORE streaming starts. Pass an AbortSignal to let the user stop generation mid-flight.
   */
  async function runStream(
    model: string,
    prompt: string,
    maxTokens: number | undefined,
    handlers: { onToken: (delta: string) => void; signal?: AbortSignal },
  ): Promise<{ content: string; usage?: ChatUsage; aborted: boolean }> {
    running.value = true
    error.value = ''
    insufficientCredit.value = false
    try {
      const resp = await fetch('/api/inference/stream', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ model, messages: [{ role: 'user', content: prompt }], max_tokens: maxTokens }),
        signal: handlers.signal,
      })
      if (!resp.ok || !resp.body) {
        let data: { code?: string; message?: string } | null = null
        try { data = await resp.json() } catch { /* non-JSON */ }
        if (resp.status === 402 || data?.code === 'INSUFFICIENT_CREDIT') insufficientCredit.value = true
        error.value = data?.message || 'Inference failed'
        const err = new Error(error.value) as Error & { statusCode?: number; data?: unknown }
        err.statusCode = resp.status
        err.data = data
        throw err
      }
      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let content = ''
      let usage: ChatUsage | undefined
      let aborted = false
      try {
        for (;;) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          // SSE frames are separated by a blank line; keep the trailing partial in the buffer.
          const frames = buffer.split('\n\n')
          buffer = frames.pop() ?? ''
          for (const frame of frames) {
            const dataLine = frame.split('\n').find((l) => l.startsWith('data:'))
            if (!dataLine) continue
            const payload = dataLine.slice(5).trim()
            if (!payload || payload === '[DONE]') continue
            try {
              const j = JSON.parse(payload) as {
                choices?: Array<{ delta?: { content?: string } }>
                usage?: ChatUsage
              }
              const delta = j.choices?.[0]?.delta?.content
              if (delta) { content += delta; handlers.onToken(delta) }
              if (j.usage) usage = j.usage
            } catch { /* keep-alive comment or split frame — ignore */ }
          }
        }
      } catch (streamErr) {
        // An abort is an expected user action: keep whatever streamed so far.
        if ((streamErr as Error)?.name === 'AbortError' || handlers.signal?.aborted) aborted = true
        else throw streamErr
      }
      result.value = { content, usage: usage ?? { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 }, model }
      return { content, usage, aborted }
    } finally {
      running.value = false
    }
  }

  /** run sends a single-turn prompt to a model and stores the completion + usage. */
  async function run(model: string, prompt: string, maxTokens?: number) {
    running.value = true
    error.value = ''
    insufficientCredit.value = false
    try {
      const r = await $fetch<ChatResponse>('/api/inference/chat', {
        method: 'POST',
        body: { model, messages: [{ role: 'user', content: prompt }], max_tokens: maxTokens },
      })
      result.value = { content: r.choices?.[0]?.message?.content || '', usage: r.usage, model: r.model }
      return result.value
    } catch (err: unknown) {
      const ex = err as { statusCode?: number; data?: { code?: string; message?: string } }
      if (ex?.statusCode === 402 || ex?.data?.code === 'INSUFFICIENT_CREDIT') insufficientCredit.value = true
      error.value = ex?.data?.message || 'Inference failed'
      throw err
    } finally {
      running.value = false
    }
  }

  return { running, error, insufficientCredit, result, run, runStream }
}
