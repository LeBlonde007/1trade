/** GET /api/billing/budget — the tenant's monthly budget. Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/billing/budget'))
