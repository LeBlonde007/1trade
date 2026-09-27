/**
 * GET /api/wallet/cash-balances — the tenant's PAPER cash balances (USD), with locked_amount held by
 * open orders. Proxies credit-ledger /v1/credits/cash/balances (credit.yaml v1.1+; session JWT).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'ledger', '/v1/credits/cash/balances'))
