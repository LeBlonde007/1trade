/** GET /api/billing/purchases — the tenant's purchase history. Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/billing/purchases'))
