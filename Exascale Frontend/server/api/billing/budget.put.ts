/** PUT /api/billing/budget — set the tenant's monthly budget (billing/admin). Proxies platform-core. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ credit_type?: string; monthly_limit?: string }>(event)
  return proxyJson(event, 'platform', '/v1/billing/budget', { method: 'PUT', body })
})
