/** DELETE /api/trading/orders/:id — cancel one of the caller's open paper orders. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { uuidParam } from '../../../utils/ids'

export default defineEventHandler((event) =>
  proxyJson(event, 'trading', `/v1/trading/orders/${uuidParam(event)}`, { method: 'DELETE' }))
