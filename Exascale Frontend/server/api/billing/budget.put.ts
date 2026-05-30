/** PUT /api/billing/budget — set the tenant's monthly budget (billing/admin). Live: platform-core. */
import { defineEventHandler, readBody } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ credit_type?: string; monthly_limit?: string }>(event)
  if (isMock(event)) return { credit_type: body?.credit_type, monthly_limit: body?.monthly_limit }
  return proxyJson(event, 'platform', '/v1/billing/budget', { method: 'PUT', body })
})
