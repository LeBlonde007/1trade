/**
 * GET /api/compute/types — the GPU-tier catalog (type, price/hour, current availability). Proxies
 * compute-control /v1/compute/types.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'compute', '/v1/compute/types'))
