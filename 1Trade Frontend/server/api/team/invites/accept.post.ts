/**
 * POST /api/team/invites/accept — accept an invitation with a new password: creates the member,
 * sets the session cookie, and returns the identity (like login).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson, setSession } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  const out = await proxyJson<{ token: string }>(event, 'platform', '/v1/account/invites/accept', {
    method: 'POST', body: { token: b.token, password: b.password },
  })
  setSession(event, out.token)
  return proxyJson(event, 'platform', '/v1/auth/me', { token: out.token })
})
