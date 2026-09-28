/** DELETE /api/supply/sources/:id — retire a source (it drains, then leaves the pool). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/supply/sources/${sourceId(event)}`, { method: 'DELETE' }))
