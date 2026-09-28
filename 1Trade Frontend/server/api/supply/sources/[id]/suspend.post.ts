/** POST /api/supply/sources/:id/suspend — stop taking new work; running work drains. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/supply/sources/${sourceId(event)}/suspend`, { method: 'POST' }))
