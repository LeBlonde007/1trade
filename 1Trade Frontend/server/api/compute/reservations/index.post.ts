/**
 * POST /api/compute/reservations — buy reserved capacity, prepaid in GPU credits (F14). Only the
 * three contract fields are forwarded; the caller's Idempotency-Key is reused so a retry never
 * charges twice.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'
import { idempotencyKey } from '../../../utils/compute'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'compute', '/v1/compute/reservations', {
    method: 'POST',
    body: { gpu_type: b.gpu_type, gpus: b.gpus, term: b.term },
    headers: { 'Idempotency-Key': idempotencyKey(event) },
  })
})
