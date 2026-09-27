/**
 * GET /api/wallet/cash-transactions — the tenant's paper cash movements (grant / trade / fee), newest
 * first. Proxies credit-ledger /v1/credits/cash/transactions (session JWT); forwards ?limit (1–200).
 */
import { defineEventHandler, getQuery } from 'h3'
import { proxyJson } from '../../utils/api'
import { boundedLimit } from '../../utils/query'

export default defineEventHandler((event) =>
  proxyJson(event, 'ledger', '/v1/credits/cash/transactions' + boundedLimit(getQuery(event).limit, 200)))
