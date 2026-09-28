/** DELETE /api/compute/clusters/:id — terminate every node together (F15). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { clusterId } from '../../../utils/compute'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/compute/clusters/${clusterId(event)}`, { method: 'DELETE' }))
