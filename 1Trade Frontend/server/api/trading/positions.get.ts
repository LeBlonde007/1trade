/**
 * GET /api/trading/positions — the caller's paper positions.
 * Every row carries is_paper; the engine only holds paper state while the venue is unlicensed.
 * Proxies matching-engine /v1/trading/positions (tenant-scoped by the session JWT).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/trading/positions'))
