/** GET /api/compute/reservations/quote?gpu_type=&gpus=&term= — price a reservation (F14). */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => {
  const q = getQuery(event)
  const qs = new URLSearchParams({ gpu_type: String(q.gpu_type ?? ''), gpus: String(q.gpus ?? ''), term: String(q.term ?? '') })
  return proxyJson(event, 'compute', `/v1/compute/reservations/quote?${qs}`)
})
