/** POST /api/conversations — create a conversation. Proxies platform-core. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  return proxyJson(event, 'platform', '/v1/conversations', { method: 'POST', body })
})
