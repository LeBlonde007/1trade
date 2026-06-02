/**
 * GET /api/wallet/conversion-rates — current per-pair credit conversion rates + the house spread.
 * Proxies credit-ledger /v1/credits/conversion-rates (public; rates aren't secret). Rates are decimal
 * strings ('to' units per 1 'from' unit, pre-spread); spread is a decimal fraction.
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'ledger', '/v1/credits/conversion-rates'))
