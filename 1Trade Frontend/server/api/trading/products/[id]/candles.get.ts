/**
 * GET /api/trading/products/:id/candles — OHLCV series for the chart.
 * Proxies matching-engine /v1/trading/products/{id}/candles; `interval` and `limit` pass through.
 */
import { defineEventHandler, getRouterParam, getQuery, createError } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) throw createError({ statusCode: 400, statusMessage: 'product id is required' })
  const q = getQuery(event)
  const parts: string[] = []
  if (q.interval) parts.push(`interval=${encodeURIComponent(String(q.interval))}`)
  if (q.limit) parts.push(`limit=${encodeURIComponent(String(q.limit))}`)
  const qs = parts.length ? `?${parts.join('&')}` : ''
  return proxyJson(event, 'trading', `/v1/trading/products/${encodeURIComponent(id)}/candles${qs}`)
})
