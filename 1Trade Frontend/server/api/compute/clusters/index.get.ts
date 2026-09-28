/** GET /api/compute/clusters — the tenant's multi-node clusters (F15). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'compute', '/v1/compute/clusters'))
