/**
 * POST /api/inference/video — submit an async text-to-video job. Proxies inference-gateway
 * POST /v1/videos (session JWT; a 402 INSUFFICIENT_CREDIT surfaces so the UI can prompt to buy video
 * credits). Returns { id, status } — the client then polls GET /api/inference/video/{id}.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface VideoIn {
  model?: string
  prompt?: string
  size?: string
}

export default defineEventHandler(async (event) => {
  const body = await readBody<VideoIn>(event)
  return proxyJson(event, 'gateway', '/v1/videos', { method: 'POST', body })
})
