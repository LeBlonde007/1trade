/**
 * POST /api/keys — mint a scoped API key. The `secret` is returned ONCE. Proxies platform-core
 * /v1/auth/keys.
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface KeyIn { name?: string; scopes?: string[] }

export default defineEventHandler(async (event) => {
  const body = await readBody<KeyIn>(event)
  return proxyJson(event, 'platform', '/v1/auth/keys', { method: 'POST', body: { name: body?.name, scopes: body?.scopes || [] } })
})
