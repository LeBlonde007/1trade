/** GET /api/compute/reservations/:id — one reservation (F14). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { reservationId } from '../../../utils/compute'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/compute/reservations/${reservationId(event)}`))
