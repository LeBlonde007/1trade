/** DELETE /api/account/sso — turn single sign-on off (admin). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/account/sso', { method: 'DELETE' }))
