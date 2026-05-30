/** GET /api/account/audit — the tenant's audit log (admin). Live: platform-core. */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return { entries: [
      { id: 'a1', action: 'apikey.create', target_type: 'api_key', is_paper: true, created_at: '2026-05-30T12:00:00Z' },
      { id: 'a2', action: 'tenant.signup', target_type: 'tenant', is_paper: true, created_at: '2026-05-30T11:30:00Z' },
    ] }
  }
  return proxyJson(event, 'platform', '/v1/account/audit')
})
