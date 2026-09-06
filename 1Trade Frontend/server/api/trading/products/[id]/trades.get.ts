/**
 * GET /api/trading/products/:id/trades — recent prints (the tape).
 * Each carries aggressor_side and is_paper, so the tape shows real direction rather than a coin
 * flip. Proxies matching-engine /v1/trading/products/{id}/trades.
 */
import { defineEventHandler, getRouterParam, getQuery, createError } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) throw createError({ statusCode: 400, statusMessage: 'product id is required' })
  const limit = getQuery(event).limit
  const qs = limit ? `?limit=${encodeURIComponent(String(limit))}` : ''
  return proxyJson(event, 'trading', `/v1/trading/products/${encodeURIComponent(id)}/trades${qs}`)
})
