/** POST /api/auth/verify — verify an email with the one-time token. Proxies platform-core. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ token?: string }>(event)
  return proxyJson(event, 'platform', '/v1/auth/verify', { method: 'POST', body: { token: body?.token } })
})
