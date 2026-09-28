/** DELETE /api/team/members/:id — remove a member (admin). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'
import { uuidParam } from '../../../utils/ids'

export default defineEventHandler(event => proxyJson(event, 'platform', `/v1/account/members/${uuidParam(event)}`, { method: 'DELETE' }))
