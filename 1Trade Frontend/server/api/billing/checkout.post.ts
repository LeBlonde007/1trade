/**
 * POST /api/billing/checkout — start a credit purchase. Proxies platform-core /v1/billing/checkout
 * (returns a Stripe checkout URL).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface CheckoutIn { amount?: string; credit_type?: string; method?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<CheckoutIn>(event)
  // US dollars only; method is card or ach (wires have their own route).
  return proxyJson(event, 'platform', '/v1/billing/checkout', {
    method: 'POST', body: { amount: body?.amount, credit_type: body?.credit_type, currency: 'usd', method: body?.method || 'card' },
  })
})
