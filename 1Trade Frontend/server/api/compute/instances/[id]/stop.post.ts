/** POST /api/compute/instances/:id/stop — stop an instance (releases GPUs, ends billing). */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler((event) => {
  const id = getRouterParam(event, 'id')
  return proxyJson(event, 'compute', `/v1/compute/instances/${id}/stop`, { method: 'POST' })
})
