/** POST /api/compute/instances/:id/start — restart a stopped instance (re-reserves GPUs). */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../../utils/api'

export default defineEventHandler((event) => {
  const id = getRouterParam(event, 'id')
  return proxyJson(event, 'compute', `/v1/compute/instances/${id}/start`, { method: 'POST' })
})
