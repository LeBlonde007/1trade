/** GET /api/supply/payouts — the partner's payout statements, newest first (compute-control, F18). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'compute', '/v1/supply/payouts'))
