/**
 * GET /api/wallet/transactions — the tenant's recent ledger transactions. Live: credit-ledger
 * /v1/credits/transactions (session JWT). Mock: a small canned list.
 */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return {
      transactions: [
        { tx_id: 'tx_demo_3', credit_type: 'text', operation: 'consumption', amount: '-0.135000', balance_after: '482.500000', is_paper: true, created_at: '2026-05-30T12:02:00Z' },
        { tx_id: 'tx_demo_2', credit_type: 'text', operation: 'consumption', amount: '-0.100000', balance_after: '482.635000', is_paper: true, created_at: '2026-05-30T12:00:00Z' },
        { tx_id: 'tx_demo_1', credit_type: 'text', operation: 'purchase', amount: '+500.000000', balance_after: '500.000000', is_paper: true, created_at: '2026-05-30T11:30:00Z' },
      ],
    }
  }
  return proxyJson(event, 'ledger', '/v1/credits/transactions')
})
