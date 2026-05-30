/** GET /api/billing/budget — the tenant's monthly budget. Live: platform-core. */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) return { budget: { credit_type: 'text', monthly_limit: '1000.000000' } }
  return proxyJson(event, 'platform', '/v1/billing/budget')
})
