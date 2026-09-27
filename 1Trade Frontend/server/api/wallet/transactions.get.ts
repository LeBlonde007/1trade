/**
 * GET /api/wallet/transactions — the tenant's recent ledger transactions. Proxies credit-ledger
 * /v1/credits/transactions (session JWT); forwards ?limit (1–200, credit.yaml).
 */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../utils/api'
import { boundedLimit } from '../../utils/query'

export default defineEventHandler((event) =>
  proxyJson(event, 'ledger', '/v1/credits/transactions' + boundedLimit(getQuery(event).limit, 200)))
