/**
 * POST /api/supply/sources — register capacity (it starts pending until 1Trade activates it). The
 * client sends its own Idempotency-Key so a retried submit cannot register the source twice.
 */
import { defineEventHandler, readBody, getHeader, createError } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const key = getHeader(event, 'idempotency-key') || ''
  if (!/^[A-Za-z0-9_-]{8,128}$/.test(key)) throw createError({ statusCode: 422, statusMessage: 'Idempotency-Key required' })
  const b = await readBody<Record<string, unknown>>(event)
  // Forward only the contract's fields.
  const body = { name: b?.name, gpu_type: b?.gpu_type, gpu_count: b?.gpu_count, region: b?.region, sla_tier: b?.sla_tier }
  return proxyJson(event, 'compute', '/v1/supply/sources', { method: 'POST', body, headers: { 'Idempotency-Key': key } })
})
