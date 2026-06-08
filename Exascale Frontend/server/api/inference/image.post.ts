/**
 * POST /api/inference/image — generate an image. Proxies inference-gateway /v1/images/generations
 * (using the session JWT; a 402 INSUFFICIENT_CREDIT is surfaced so the UI can prompt to buy image
 * credits). The gateway returns base64 PNGs in `data[].b64_json`.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface ImageIn {
  model?: string
  prompt?: string
  n?: number
  size?: string
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ImageIn>(event)
  return proxyJson(event, 'gateway', '/v1/images/generations', { method: 'POST', body })
})
