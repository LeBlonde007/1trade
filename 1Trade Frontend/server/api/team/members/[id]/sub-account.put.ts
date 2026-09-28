/** PUT /api/team/members/:id/sub-account — place a member in a sub-account, or back on the main balance (admin). */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { uuidParam } from '../../../../utils/ids'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', `/v1/account/members/${uuidParam(event)}/sub-account`, {
    method: 'PUT', body: { sub_account_id: b.sub_account_id ?? null },
  })
})
