/**
 * GET /api/auth/me — the current identity from the session cookie. 401 when logged out. Live:
 * platform-core /v1/auth/me. Mock: a canned identity.
 */
import { defineEventHandler, createError } from 'h3'
import { isMock, mockIdentity, sessionToken, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) return mockIdentity()
  if (!sessionToken(event)) throw createError({ statusCode: 401, statusMessage: 'not authenticated' })
  return proxyJson(event, 'platform', '/v1/auth/me')
})
