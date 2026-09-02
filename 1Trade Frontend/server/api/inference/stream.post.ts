/**
 * POST /api/inference/stream — streaming chat completion over Server-Sent Events. Proxies
 * inference-gateway /v1/chat/completions with `stream: true` and pipes the SSE body straight to the
 * browser, so tokens render as the model generates them. A pre-stream non-2xx (e.g. 402
 * INSUFFICIENT_CREDIT) is surfaced as a normal JSON error BEFORE any streaming begins, so the client
 * can show the buy-credits prompt instead of a broken stream.
 */
import { defineEventHandler, readBody, setResponseHeader, createError } from 'h3'
import { sessionToken } from '../../utils/api'

interface ChatIn {
  model?: string
  messages?: Array<{ role: string; content: string }>
  max_tokens?: number
}

interface UpstreamError {
  message?: string
  code?: string
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ChatIn>(event)
  const cfg = useRuntimeConfig(event)
  const token = sessionToken(event)

  const upstream = await fetch((cfg.gatewayUrl as string) + '/v1/chat/completions', {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      accept: 'text/event-stream',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
    },
    // Force streaming; keep the caller's model/messages/max_tokens untouched.
    body: JSON.stringify({ ...body, stream: true }),
  })

  // Surface auth/credit/upstream failures as a JSON error before we commit to an SSE response.
  if (!upstream.ok || !upstream.body) {
    const text = await upstream.text()
    let data: UpstreamError | null = null
    try { data = text ? (JSON.parse(text) as UpstreamError) : null } catch { /* non-JSON upstream */ }
    throw createError({ statusCode: upstream.status, statusMessage: data?.message || 'upstream error', data })
  }

  // Stream the SSE body straight through. `x-accel-buffering: no` + `no-transform` ask any proxy in
  // front of us not to buffer, so tokens arrive incrementally rather than in one flush.
  setResponseHeader(event, 'content-type', 'text/event-stream; charset=utf-8')
  setResponseHeader(event, 'cache-control', 'no-cache, no-transform')
  setResponseHeader(event, 'connection', 'keep-alive')
  setResponseHeader(event, 'x-accel-buffering', 'no')
  return upstream.body
})
