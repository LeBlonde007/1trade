/**
 * POST /api/billing/checkout — start a credit purchase. Proxies platform-core /v1/billing/checkout
 * (returns a Stripe checkout URL).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface CheckoutIn { amount?: string; credit_type?: string; currency?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<CheckoutIn>(event)
  return proxyJson(event, 'platform', '/v1/billing/checkout', { method: 'POST', body })
})
