/**
 * POST /api/trading/orders — submit an order.
 *
 * While the venue is unlicensed the engine answers 503 EXCHANGE_PAUSED, and proxyJson re-throws
 * upstream non-2xx with the real status — so the ticket surfaces the pause honestly instead of
 * appearing to accept an order. Do not swallow that error to make the UI feel responsive.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  return proxyJson(event, 'trading', '/v1/trading/orders', { method: 'POST', body })
})
