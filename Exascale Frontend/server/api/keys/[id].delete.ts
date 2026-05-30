/** DELETE /api/keys/:id — revoke an API key. Live: platform-core DELETE /v1/auth/keys/{id}. */
import { defineEventHandler, getRouterParam } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (isMock(event)) return { ok: true }
  await proxyJson(event, 'platform', `/v1/auth/keys/${id}`, { method: 'DELETE' })
  return { ok: true }
})
