/**
 * GET /api/wallet/transactions — the tenant's recent ledger transactions. Proxies credit-ledger
 * /v1/credits/transactions (session JWT).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'ledger', '/v1/credits/transactions'))
