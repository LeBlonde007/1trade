/** GET /api/trading/products/:id — one product with its 24h summary (public market data). */
import { createError, defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler((event) => {
  const id = getRouterParam(event, 'id') || ''
  if (!/^[A-Z0-9-]{2,32}$/.test(id)) throw createError({ statusCode: 404, statusMessage: 'no such product' })
  return proxyJson(event, 'trading', `/v1/trading/products/${id}`)
})
