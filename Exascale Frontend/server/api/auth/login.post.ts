/**
 * POST /api/auth/login — verify credentials, set the httpOnly session cookie, return the identity.
 * Live: platform-core /v1/auth/login → /v1/auth/me. Mock: a canned identity.
 */
import { defineEventHandler, readBody } from 'h3'
import { isMock, setSession, mockIdentity, proxyJson } from '../../utils/api'

interface Creds { email?: string; password?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<Creds>(event)
  if (isMock(event)) {
    setSession(event, 'mock-token')
    return mockIdentity(body?.email)
  }
  const out = await proxyJson<{ token: string; expires_at: string }>(event, 'platform', '/v1/auth/login', {
    method: 'POST',
    body: { email: body?.email, password: body?.password },
  })
  setSession(event, out.token)
  return proxyJson(event, 'platform', '/v1/auth/me', { token: out.token })
})
