/** GET /api/account/kyc — the caller's tenant KYC status (gates real-money buys). Proxies platform-core. */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler((event) => proxyJson(event, 'platform', '/v1/account/kyc'))
