/**
 * GET /api/supply/agreement — the signed-in partner's payout terms, or null until 1Trade sets them
 * (a normal state for a new partner, so it is answered 200 rather than surfacing as a 404 error). The
 * tenant id comes from the session (never the client), so a partner can only ever read its own.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const me = await proxyJson<{ tenant_id?: string; tenant?: { id?: string } }>(event, 'platform', '/v1/auth/me')
  const tenant = me?.tenant_id || me?.tenant?.id
  if (!tenant || !/^[0-9a-f-]{36}$/i.test(tenant)) return null
  try {
    return await proxyJson(event, 'compute', `/v1/supply/partners/${tenant}/agreement`)
  } catch (e: any) {
    if (e?.statusCode === 404) return null
    throw e
  }
})
