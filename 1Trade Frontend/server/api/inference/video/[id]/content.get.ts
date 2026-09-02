/**
 * GET /api/inference/video/:id/content — stream the finished mp4. The provider keeps the bytes behind
 * the API key, so the browser can't fetch them directly: this proxies inference-gateway
 * GET /v1/videos/{id}/content (with the session JWT) and pipes the body straight through, so a
 * <video src> can point here. Only call it once the job's status is 'completed'.
 */
import { defineEventHandler, getRouterParam, setResponseHeader, createError } from 'h3'
import { sessionToken } from '../../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id') || ''
  const cfg = useRuntimeConfig(event)
  const token = sessionToken(event)
  const upstream = await fetch(`${cfg.gatewayUrl as string}/v1/videos/${encodeURIComponent(id)}/content`, {
    headers: { ...(token ? { authorization: `Bearer ${token}` } : {}) },
  })
  if (!upstream.ok || !upstream.body) {
    throw createError({ statusCode: upstream.status || 502, statusMessage: 'video content unavailable' })
  }
  setResponseHeader(event, 'content-type', upstream.headers.get('content-type') || 'video/mp4')
  setResponseHeader(event, 'cache-control', 'private, max-age=600')
  return upstream.body
})
