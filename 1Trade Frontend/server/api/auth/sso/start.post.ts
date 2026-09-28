/** POST /api/auth/sso/start {email} — the identity-provider URL to send the browser to. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/auth/sso/start', { method: 'POST', body: { email: String(b.email ?? '') } })
})
