/** POST /api/compute/clusters — create a cluster, all nodes at once on one fabric (F15). */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'
import { idempotencyKey } from '../../../utils/compute'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'compute', '/v1/compute/clusters', {
    method: 'POST',
    body: { gpu_type: b.gpu_type, gpus: b.gpus, network: b.network, topology: b.topology },
    headers: { 'Idempotency-Key': idempotencyKey(event) },
  })
})
