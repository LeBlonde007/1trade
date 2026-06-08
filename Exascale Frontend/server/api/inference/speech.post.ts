/**
 * POST /api/inference/speech — generate speech audio. Proxies inference-gateway POST /v1/audio/speech
 * (with the session JWT) and pipes the audio bytes (WAV) straight back, so the client can turn the
 * response into a blob and play it. A pre-audio non-2xx (402 INSUFFICIENT_CREDIT, etc.) surfaces as a
 * JSON error before any audio streams.
 */
import { defineEventHandler, readBody, setResponseHeader, createError } from 'h3'
import { sessionToken } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  const cfg = useRuntimeConfig(event)
  const token = sessionToken(event)
  const upstream = await fetch(`${cfg.gatewayUrl as string}/v1/audio/speech`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', ...(token ? { authorization: `Bearer ${token}` } : {}) },
    body: JSON.stringify(body),
  })
  if (!upstream.ok || !upstream.body) {
    let data: unknown = null
    try { data = await upstream.json() } catch { /* non-JSON */ }
    throw createError({ statusCode: upstream.status, statusMessage: 'speech generation failed', data })
  }
  setResponseHeader(event, 'content-type', upstream.headers.get('content-type') || 'audio/wav')
  return upstream.body
})
