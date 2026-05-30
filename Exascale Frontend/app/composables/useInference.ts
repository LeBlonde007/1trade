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

  return { running, error, insufficientCredit, result, run }
}
