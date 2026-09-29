/**
 * POST /api/trading/orders — place a paper order (trading.yaml v1.1).
 *
 * The Idempotency-Key is forwarded so a retried click returns the same order instead of placing a
 * second one. is_paper is never sent: the engine takes it from the session's token. A real-money
 * account gets 503 EXCHANGE_PAUSED, and proxyJson re-throws it with the real status so the ticket shows
 * the refusal — do not swallow it to make the UI feel responsive.
 */
import { createError, defineEventHandler, getHeader, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const key = getHeader(event, 'idempotency-key') || ''
  if (!key || key.length > 128) throw createError({ statusCode: 400, statusMessage: 'Idempotency-Key required' })
  const b = await readBody(event)
  const body = {
    product_id: b?.product_id, side: b?.side, order_type: b?.order_type, quantity: b?.quantity,
    limit_price: b?.limit_price ?? null, time_in_force: b?.time_in_force,
  }
  return proxyJson(event, 'trading', '/v1/trading/orders', { method: 'POST', body, headers: { 'idempotency-key': key } })
})
