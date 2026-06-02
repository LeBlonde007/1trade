/**
 * GET /api/wallet/balances — the tenant's credit balances. Proxies credit-ledger /v1/credits/balances
 * (resolved from the session JWT). Amounts are fixed-point decimal strings.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'ledger', '/v1/credits/balances'))
