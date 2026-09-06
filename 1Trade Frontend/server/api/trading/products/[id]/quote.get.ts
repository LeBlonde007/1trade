/**
 * GET /api/trading/products/:id/quote — two-sided quote for one product.
 *
 * The engine is the single source of truth for market data, including while the venue is paused:
 * the numbers are simulated, but they are simulated SERVER-side so every screen shows the same
 * book. Screens must not invent their own prices. Proxies matching-engine
 * /v1/trading/products/{id}/quote.
 */
import { defineEventHandler, getRouterParam, createError } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) throw createError({ statusCode: 400, statusMessage: 'product id is required' })
  return proxyJson(event, 'trading', `/v1/trading/products/${encodeURIComponent(id)}/quote`)
})
