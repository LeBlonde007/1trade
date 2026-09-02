/** GET /api/conversations/:id — load one conversation (full transcript). Proxies platform-core. */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  const id = getRouterParam(event, 'id') || ''
  return proxyJson(event, 'platform', `/v1/conversations/${encodeURIComponent(id)}`, { method: 'GET' })
})
