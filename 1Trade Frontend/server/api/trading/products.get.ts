/**
 * GET /api/trading/products — the exchange's tradeable products plus its own status.
 *
 * The matching engine reports `exchange_status` (state / reason / methodology_url) alongside the
 * products. Screens must render that verbatim rather than assuming the venue is open: order entry
 * is refused with 503 EXCHANGE_PAUSED until the F22 licence clears, and the market data it returns
 * is simulated. Proxies matching-engine /v1/trading/products.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/trading/products'))
