/** GET /api/auth/2fa — whether two-factor is on and how many recovery codes are left. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/auth/2fa'))
