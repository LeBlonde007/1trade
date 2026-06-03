/**
 * GET /api/compute/instances — the tenant's GPU instances (optionally ?state=running). Proxies
 * compute-control /v1/compute/instances (resolved from the session JWT).
 */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => {
  const state = getQuery(event).state
  const qs = state ? `?state=${encodeURIComponent(String(state))}` : ''
  return proxyJson(event, 'compute', `/v1/compute/instances${qs}`)
})
