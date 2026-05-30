/** GET /api/keys — list the tenant's API keys (metadata only). Live: platform-core /v1/auth/keys. */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return { keys: [{ id: 'key_demo', name: 'production', prefix: 'exk_1a2b3c4d', scopes: ['inference:read'], created_at: '2026-05-20T10:00:00Z', revoked: false }] }
  }
  return proxyJson(event, 'platform', '/v1/auth/keys')
})
