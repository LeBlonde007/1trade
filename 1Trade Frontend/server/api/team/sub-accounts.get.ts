/** GET /api/team/sub-accounts — the tenant's sub-accounts. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/account/sub-accounts'))
