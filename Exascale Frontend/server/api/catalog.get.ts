/**
 * GET /api/catalog — the model catalog. Proxies inference-gateway /v1/models (OpenAI-shaped, with the
 * Exascale pricing extension).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'gateway', '/v1/models'))
