/** GET /api/compute/reservations — the tenant's reservations and reserved-vs-occupied GPUs (F14). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'compute', '/v1/compute/reservations'))
