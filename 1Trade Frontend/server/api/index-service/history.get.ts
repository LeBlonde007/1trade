/**
 * GET /api/index-service/history?days=N — historical index prints, oldest first, hash-chained.
 * Proxies matching-engine /v1/index/history; `days` is clamped to the contract's 1..730.
 */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const n = Number(getQuery(event).days)
  const days = Number.isFinite(n) ? Math.min(730, Math.max(1, Math.floor(n))) : 90
  return proxyJson(event, 'trading', `/v1/index/history?days=${days}`)
})
