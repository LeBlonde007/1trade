/**
 * POST /api/compute/instances — launch an on-demand GPU instance. Proxies compute-control
 * POST /v1/compute/instances with a fresh Idempotency-Key. is_paper is derived server-side from the
 * session principal, never the body.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  const idem = (globalThis.crypto?.randomUUID?.() ?? `web_${Date.now()}`)
  return proxyJson(event, 'compute', '/v1/compute/instances', {
    method: 'POST',
    body,
    headers: { 'Idempotency-Key': idem },
  })
})
