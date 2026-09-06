/**
 * GET /api/trading/orders — the caller's open orders.
 *
 * Legitimately empty while the venue is paused: order entry returns 503 EXCHANGE_PAUSED, so no
 * orders can exist. The panel must render that emptiness honestly rather than inventing rows.
 * Proxies matching-engine /v1/trading/orders (tenant-scoped by the session JWT).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/trading/orders'))
