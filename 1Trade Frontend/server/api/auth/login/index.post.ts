/**
 * POST /api/auth/login — verify credentials, set the httpOnly session cookie, return the identity.
 * Proxies platform-core /v1/auth/login → /v1/auth/me. When the account has two-factor on, the
 * password earns only a 5-minute challenge: it is kept in its own httpOnly cookie (never exposed to
 * page scripts) and the answer is { mfa_required: true } — /api/auth/login/2fa finishes the sign-in.
 */
import { defineEventHandler, readBody, setCookie } from 'h3'
import { setSession, proxyJson } from '../../../utils/api'

interface Creds { email?: string; password?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<Creds>(event)
  const out = await proxyJson<{ token?: string; mfa_required?: boolean; mfa_token?: string }>(event, 'platform', '/v1/auth/login', {
    method: 'POST',
    body: { email: body?.email, password: body?.password },
  })
  if (out.mfa_required && out.mfa_token) {
    setCookie(event, 'ex_mfa', out.mfa_token, { httpOnly: true, sameSite: 'strict', path: '/api/auth/login', secure: !import.meta.dev, maxAge: 300 })
    return { mfa_required: true }
  }
  setSession(event, out.token as string)
  return proxyJson(event, 'platform', '/v1/auth/me', { token: out.token })
})
