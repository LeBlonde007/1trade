/**
 * GET /api/index-service/methodology — the methodology version, window, trim, volume floor and the
 * live constituent weight schedule. Proxies matching-engine /v1/index/methodology.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/index/methodology'))
