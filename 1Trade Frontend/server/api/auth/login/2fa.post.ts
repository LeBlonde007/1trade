/**
 * POST /api/auth/login/2fa — the second sign-in step: a TOTP code or a recovery code redeems the
 * challenge cookie from /api/auth/login for a session.
 */
import { createError, defineEventHandler, deleteCookie, getCookie, readBody } from 'h3'
import { setSession, proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const challenge = getCookie(event, 'ex_mfa')
  if (!challenge) throw createError({ statusCode: 401, statusMessage: 'sign in again', data: { code: 'invalid_challenge', message: 'Sign in again — the two-factor step expired.' } })
  const b = (await readBody(event)) || {}
  const out = await proxyJson<{ token: string }>(event, 'platform', '/v1/auth/login/2fa', {
    method: 'POST', body: { mfa_token: challenge, code: b.code || '', recovery_code: b.recovery_code || '' },
  })
  deleteCookie(event, 'ex_mfa', { path: '/api/auth/login' })
  setSession(event, out.token)
  return proxyJson(event, 'platform', '/v1/auth/me', { token: out.token })
})
