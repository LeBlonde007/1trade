/** GET /api/keys — list the tenant's API keys (metadata only). Proxies platform-core /v1/auth/keys. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/auth/keys'))
