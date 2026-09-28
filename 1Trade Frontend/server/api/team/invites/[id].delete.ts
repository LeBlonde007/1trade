/** DELETE /api/team/invites/:id — revoke a pending invitation (admin). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { uuidParam } from '../../../utils/ids'

export default defineEventHandler(event => proxyJson(event, 'platform', `/v1/account/invites/${uuidParam(event)}`, { method: 'DELETE' }))
