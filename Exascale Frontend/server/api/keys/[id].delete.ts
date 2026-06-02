/** DELETE /api/keys/:id — revoke an API key. Proxies platform-core DELETE /v1/auth/keys/{id}. */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  await proxyJson(event, 'platform', `/v1/auth/keys/${id}`, { method: 'DELETE' })
  return { ok: true }
})
