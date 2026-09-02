/**
 * POST /api/inference/vision — ask a VLM about an image (a captured screen frame). Proxies
 * inference-gateway POST /v1/chat/vision (session JWT). Body: { model, prompt, image_url (data URI),
 * max_tokens }. Returns { text, usage }. A 402 INSUFFICIENT_CREDIT surfaces for the buy-credits prompt.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface VisionIn {
  model?: string
  prompt?: string
  image_url?: string
  max_tokens?: number
}

export default defineEventHandler(async (event) => {
  const body = await readBody<VisionIn>(event)
  return proxyJson(event, 'gateway', '/v1/chat/vision', { method: 'POST', body })
})
