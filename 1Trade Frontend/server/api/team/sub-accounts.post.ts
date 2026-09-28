/** POST /api/team/sub-accounts — create a sub-account (admin). */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/account/sub-accounts', { method: 'POST', body: { name: b.name } })
})
