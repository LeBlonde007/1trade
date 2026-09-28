/** GET /api/supply/sources — the partner's supply sources with live GPUs in use (compute-control). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'compute', '/v1/supply/sources'))
