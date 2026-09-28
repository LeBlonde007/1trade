/**
 * GET /api/supply/agreement — the signed-in partner's payout terms. The tenant id comes from the
 * session (never the client), so a partner can only ever read its own.
 */
import { defineEventHandler, createError } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const me = await proxyJson<{ tenant_id?: string; tenant?: { id?: string } }>(event, 'platform', '/v1/auth/me')
  const tenant = me?.tenant_id || me?.tenant?.id
  if (!tenant || !/^[0-9a-f-]{36}$/i.test(tenant)) throw createError({ statusCode: 404, statusMessage: 'no agreement' })
  return proxyJson(event, 'compute', `/v1/supply/partners/${tenant}/agreement`)
})
