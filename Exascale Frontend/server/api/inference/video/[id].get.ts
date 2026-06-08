/**
 * GET /api/inference/video/:id — poll an async video job's status. Proxies inference-gateway
 * GET /v1/videos/{id}. Returns the provider object including `status` (queued|in_progress|completed|
 * failed); the client fetches /content once status === 'completed'.
 */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id') || ''
  return proxyJson(event, 'gateway', `/v1/videos/${encodeURIComponent(id)}`, { method: 'GET' })
})
