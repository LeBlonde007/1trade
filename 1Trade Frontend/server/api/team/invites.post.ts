/** POST /api/team/invites — invite an email with roles (admin). Only the contract fields are forwarded. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/account/invites', {
    method: 'POST', body: { email: b.email, roles: b.roles, sub_account_id: b.sub_account_id ?? null },
  })
})
