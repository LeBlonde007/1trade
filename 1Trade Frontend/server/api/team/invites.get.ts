/** GET /api/team/invites — pending invitations (admin). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/account/invites'))
