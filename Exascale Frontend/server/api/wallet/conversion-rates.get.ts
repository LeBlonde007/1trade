/**
 * GET /api/wallet/conversion-rates — current per-pair credit conversion rates + the house spread.
 * Live: credit-ledger /v1/credits/conversion-rates (public; rates aren't secret). Mock: the seeded
 * AI-index ↔ text pair so the convert drawer shows a realistic rate offline. Rates are decimal
 * strings ('to' units per 1 'from' unit, pre-spread); spread is a decimal fraction.
 */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return {
      spread: '0.010000',
      rates: [
        { from: 'ai_index', to: 'text', rate: '0.830579' },
        { from: 'text', to: 'ai_index', rate: '1.203980' },
      ],
    }
  }
  return proxyJson(event, 'ledger', '/v1/credits/conversion-rates')
})
