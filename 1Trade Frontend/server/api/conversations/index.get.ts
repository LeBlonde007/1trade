/** GET /api/conversations — list the caller's saved conversations (metadata). Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/conversations', { method: 'GET' }))
