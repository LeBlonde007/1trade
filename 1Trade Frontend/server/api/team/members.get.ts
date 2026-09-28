/** GET /api/team/members — the tenant's members (platform-core v1.6). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/account/members'))
