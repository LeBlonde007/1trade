/** POST /api/supply/payouts/:id/dispute — the partner disputes a statement inside its window. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(async (event) => {
  const b = await readBody<{ reason?: unknown }>(event)
  const reason = typeof b?.reason === 'string' ? b.reason : ''
  return proxyJson(event, 'compute', `/v1/supply/payouts/${sourceId(event)}/dispute`, { method: 'POST', body: { reason } })
})
