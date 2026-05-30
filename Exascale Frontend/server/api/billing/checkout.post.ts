/**
 * POST /api/billing/checkout — start a credit purchase. Live: platform-core /v1/billing/checkout
 * (returns a Stripe checkout URL). Mock: a fake checkout URL.
 */
import { defineEventHandler, readBody } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

interface CheckoutIn { amount?: string; credit_type?: string; currency?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<CheckoutIn>(event)
  if (isMock(event)) {
    return { purchase_id: 'mock-purchase', checkout_url: 'https://checkout.stripe.test/pay/mock' }
  }
  return proxyJson(event, 'platform', '/v1/billing/checkout', { method: 'POST', body })
})
