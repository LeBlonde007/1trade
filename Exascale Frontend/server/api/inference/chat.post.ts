/**
 * POST /api/inference/chat — run a chat completion. Proxies inference-gateway /v1/chat/completions
 * (using the session JWT; a 402 INSUFFICIENT_CREDIT is surfaced so the UI can prompt to buy credits).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface ChatIn {
  model?: string
  messages?: Array<{ role: string; content: string }>
  max_tokens?: number
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ChatIn>(event)
  return proxyJson(event, 'gateway', '/v1/chat/completions', { method: 'POST', body })
})
