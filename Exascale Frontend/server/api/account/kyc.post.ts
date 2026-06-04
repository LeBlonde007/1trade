/** POST /api/account/kyc — submit identity verification (KYC/AML). Proxies platform-core. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ legal_name?: string; country?: string; entity_type?: string }>(event)
  return proxyJson(event, 'platform', '/v1/account/kyc', { method: 'POST', body })
})
