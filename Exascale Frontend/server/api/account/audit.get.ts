/** GET /api/account/audit — the tenant's audit log (admin). Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/account/audit'))
