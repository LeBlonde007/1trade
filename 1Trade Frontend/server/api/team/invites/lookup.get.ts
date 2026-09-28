/** GET /api/team/invites/lookup?token= — what an invitation is for (no session: the token is the credential). */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => {
  const token = String(getQuery(event).token ?? '').slice(0, 256)
  return proxyJson(event, 'platform', `/v1/account/invites/lookup?token=${encodeURIComponent(token)}`)
})
