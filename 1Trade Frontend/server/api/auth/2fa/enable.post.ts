/** POST /api/auth/2fa/enable — confirm a code from the app; returns the recovery codes once. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/auth/2fa/enable', { method: 'POST', body: { code: String(b.code ?? '') } })
})
