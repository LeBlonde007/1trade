/**
 * POST /api/inference/chat — run a chat completion. Live: inference-gateway /v1/chat/completions
 * (using the session JWT; a 402 INSUFFICIENT_CREDIT is surfaced so the UI can prompt to buy credits).
 * Mock: a deterministic echo with token usage.
 */
import { defineEventHandler, readBody } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

interface ChatIn {
  model?: string
  messages?: Array<{ role: string; content: string }>
  max_tokens?: number
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ChatIn>(event)
  if (isMock(event)) {
    const last = (body?.messages || []).filter((m) => m.role === 'user').pop()?.content || ''
    const content = `This is a mock completion from ${body?.model}. You said: "${last.slice(0, 200)}"`
    return {
      id: 'chatcmpl_mock', object: 'chat.completion', model: body?.model,
      choices: [{ index: 0, message: { role: 'assistant', content }, finish_reason: 'stop' }],
      usage: { prompt_tokens: 12, completion_tokens: 18, total_tokens: 30 },
    }
  }
  return proxyJson(event, 'gateway', '/v1/chat/completions', { method: 'POST', body })
})
