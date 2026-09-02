/** POST /api/auth/logout — clear the session cookie. */
import { defineEventHandler } from 'h3'
import { clearSessionCookie } from '../../utils/api'

export default defineEventHandler((event) => {
  clearSessionCookie(event)
  return { ok: true }
})
