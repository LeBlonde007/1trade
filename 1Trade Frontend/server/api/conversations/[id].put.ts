/** PUT /api/conversations/:id — save a conversation's transcript (+ optional title/model). Proxies platform-core. */
import { defineEventHandler, getRouterParam, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id') || ''
  const body = await readBody(event)
  return proxyJson(event, 'platform', `/v1/conversations/${encodeURIComponent(id)}`, { method: 'PUT', body })
})
