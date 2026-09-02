/** POST /api/auth/verify/resend — issue a fresh verification token for the caller. Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/auth/verify/resend', { method: 'POST' }))
