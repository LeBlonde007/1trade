/**
 * GET /api/trading/fills — the caller's executed paper fills, with fee and maker/taker liquidity.
 * Proxies matching-engine /v1/trading/fills (tenant-scoped by the session JWT).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/trading/fills'))
