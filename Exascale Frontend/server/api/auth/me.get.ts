/**
 * GET /api/auth/me — the current identity from the session cookie. 401 when logged out. Proxies
 * platform-core /v1/auth/me.
 */
import { defineEventHandler, createError } from 'h3'
import { sessionToken, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (!sessionToken(event)) throw createError({ statusCode: 401, statusMessage: 'not authenticated' })
  return proxyJson(event, 'platform', '/v1/auth/me')
})
