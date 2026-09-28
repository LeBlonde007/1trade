/** GET /api/supply/sources/:id/usage — GPU time served on the source over the last 30 days, per tier. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/supply/sources/${sourceId(event)}/usage`))
