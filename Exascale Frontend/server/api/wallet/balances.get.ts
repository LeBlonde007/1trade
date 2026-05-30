/**
 * GET /api/wallet/balances — the tenant's credit balances. Live: credit-ledger /v1/credits/balances
 * (resolved from the session JWT). Mock: canned balances. Amounts are fixed-point decimal strings.
 */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return {
      balances: [
        { credit_type: 'text', balance: '482.500000', locked_amount: '0.000000', is_paper: true },
        { credit_type: 'speech', balance: '60.000000', locked_amount: '0.000000', is_paper: true },
      ],
    }
  }
  return proxyJson(event, 'ledger', '/v1/credits/balances')
})
