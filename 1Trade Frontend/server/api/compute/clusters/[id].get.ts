/** GET /api/compute/clusters/:id — one cluster with its nodes (F15). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { clusterId } from '../../../utils/compute'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/compute/clusters/${clusterId(event)}`))
