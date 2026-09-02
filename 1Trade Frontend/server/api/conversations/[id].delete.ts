/** DELETE /api/conversations/:id — delete a conversation. Proxies platform-core (204 → {ok:true}). */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id') || ''
  await proxyJson(event, 'platform', `/v1/conversations/${encodeURIComponent(id)}`, { method: 'DELETE' })
  return { ok: true }
})
