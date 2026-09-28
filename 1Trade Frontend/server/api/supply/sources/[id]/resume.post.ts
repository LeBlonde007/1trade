/** POST /api/supply/sources/:id/resume — take work again after a suspension. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/supply/sources/${sourceId(event)}/resume`, { method: 'POST' }))
