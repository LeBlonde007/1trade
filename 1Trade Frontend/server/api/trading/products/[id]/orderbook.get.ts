/**
 * GET /api/trading/products/:id/orderbook — aggregated depth for one product.
 * Levels carry price / size / cumulative, so the depth bars are the engine's numbers rather than a
 * client-side accumulation. Proxies matching-engine /v1/trading/products/{id}/orderbook.
 */
import { defineEventHandler, getRouterParam, getQuery, createError } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) throw createError({ statusCode: 400, statusMessage: 'product id is required' })
  const depth = getQuery(event).depth
  const qs = depth ? `?depth=${encodeURIComponent(String(depth))}` : ''
  return proxyJson(event, 'trading', `/v1/trading/products/${encodeURIComponent(id)}/orderbook${qs}`)
})
