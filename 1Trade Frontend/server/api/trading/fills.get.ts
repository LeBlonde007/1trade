/**
 * GET /api/trading/fills — the caller's executed paper fills, with fee and maker/taker liquidity.
 * Proxies matching-engine /v1/trading/fills (tenant-scoped by the session JWT); forwards ?limit (1–200).
 */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../utils/api'
import { boundedLimit } from '../../utils/query'

export default defineEventHandler(async (event) =>
  proxyJson(event, 'trading', '/v1/trading/fills' + boundedLimit(getQuery(event).limit, 200)))
