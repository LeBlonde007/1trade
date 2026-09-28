/**
 * POST /api/team/sub-accounts/:id/transfer — fund a sub-account from the main balance or return
 * credits (admin or billing). The caller's Idempotency-Key is reused so a retry never moves twice.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { idempotencyKey } from '../../../../utils/compute'
import { uuidParam } from '../../../../utils/ids'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', `/v1/account/sub-accounts/${uuidParam(event)}/transfer`, {
    method: 'POST',
    body: { credit_type: b.credit_type, amount: b.amount, direction: b.direction },
    headers: { 'Idempotency-Key': idempotencyKey(event) },
  })
})
