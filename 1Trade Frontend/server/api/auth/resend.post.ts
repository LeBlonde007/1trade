/**
 * POST /api/auth/resend — resend the email-verification link for an (unverified) address. Used by the
 * login screen when sign-in is blocked by the verification gate (the user has no session yet). Proxies
 * platform-core /v1/auth/verify/resend in its unauthenticated by-email mode, which always returns 200
 * regardless of whether the email exists (no account enumeration).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ email?: string }>(event)
  return proxyJson(event, 'platform', '/v1/auth/verify/resend', { method: 'POST', body: { email: body?.email } })
})
