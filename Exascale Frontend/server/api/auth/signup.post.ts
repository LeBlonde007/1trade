/**
 * POST /api/auth/signup — create a tenant + admin user, set the httpOnly session cookie, return the
 * identity. Proxies platform-core /v1/auth/signup → /v1/auth/me.
 */
import { defineEventHandler, readBody } from 'h3'
import { setSession, proxyJson } from '../../utils/api'

interface Creds { email?: string; password?: string; tenant_name?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<Creds>(event)
  const out = await proxyJson<{ token: string; expires_at: string }>(event, 'platform', '/v1/auth/signup', {
    method: 'POST',
    body: { email: body?.email, password: body?.password, tenant_name: body?.tenant_name },
  })
  setSession(event, out.token)
  return proxyJson(event, 'platform', '/v1/auth/me', { token: out.token })
})
