/** GET /api/account/sso — the tenant's SAML configuration and our service-provider URLs (admin). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(event => proxyJson(event, 'platform', '/v1/account/sso'))
