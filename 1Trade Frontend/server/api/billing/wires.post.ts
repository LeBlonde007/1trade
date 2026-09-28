/** POST /api/billing/wires — invoice a credit purchase payable by USD wire (admin or billing). */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/billing/wires', { method: 'POST', body: { amount: b.amount, credit_type: b.credit_type } })
})
