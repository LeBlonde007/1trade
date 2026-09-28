/** POST /api/auth/2fa/setup — a fresh authenticator secret (platform-core v1.7). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/auth/2fa/setup', { method: 'POST' }))
