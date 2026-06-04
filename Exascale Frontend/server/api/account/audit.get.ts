/** GET /api/account/audit — the tenant's audit log (admin). Proxies platform-core; forwards ?limit. */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  const { limit } = getQuery(event)
  const n = Number(limit)
  const qs = Number.isFinite(n) && n >= 1 && n <= 500 ? `?limit=${Math.floor(n)}` : ''
  return proxyJson(event, 'platform', '/v1/account/audit' + qs)
})
