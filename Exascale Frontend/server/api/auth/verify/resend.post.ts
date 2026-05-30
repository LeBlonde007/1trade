/** POST /api/auth/verify/resend — issue a fresh verification token for the caller. */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) return { sent: true }
  return proxyJson(event, 'platform', '/v1/auth/verify/resend', { method: 'POST' })
})
