/** GET /api/billing/purchases — the tenant's purchase history. Live: platform-core. */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return { purchases: [{ id: 'p_demo', amount: '500.000000', credit_type: 'text', currency: 'usd', status: 'paid', created_at: '2026-05-30T11:30:00Z' }] }
  }
  return proxyJson(event, 'platform', '/v1/billing/purchases')
})
