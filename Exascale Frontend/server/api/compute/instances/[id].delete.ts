/** DELETE /api/compute/instances/:id — terminate an instance (frees its GPUs). Proxies compute-control. */
import { defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => {
  const id = getRouterParam(event, 'id')
  return proxyJson(event, 'compute', `/v1/compute/instances/${id}`, { method: 'DELETE' })
})
