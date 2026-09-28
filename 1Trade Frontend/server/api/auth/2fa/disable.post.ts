/** POST /api/auth/2fa/disable — turn two-factor off; needs a current code or a recovery code. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/auth/2fa/disable', {
    method: 'POST', body: { code: String(b.code ?? ''), recovery_code: String(b.recovery_code ?? '') },
  })
})
