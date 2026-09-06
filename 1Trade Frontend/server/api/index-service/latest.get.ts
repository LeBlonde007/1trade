/**
 * GET /api/index-service/latest — the current index print.
 * Proxies matching-engine /v1/index/latest. Named `index-service` because a route file called
 * `index.get.ts` would resolve as the directory's index route, not `/api/index`.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => proxyJson(event, 'trading', '/v1/index/latest'))
