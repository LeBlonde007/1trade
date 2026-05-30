/**
 * POST /api/keys — mint a scoped API key. The `secret` is returned ONCE. Live: platform-core
 * /v1/auth/keys. Mock: a fake one-time secret.
 */
import { defineEventHandler, readBody } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

interface KeyIn { name?: string; scopes?: string[] }

export default defineEventHandler(async (event) => {
  const body = await readBody<KeyIn>(event)
  if (isMock(event)) {
    return { id: 'key_new', name: body?.name, prefix: 'exk_9f8e7d6c', scopes: body?.scopes || [], secret: 'exk_9f8e7d6c' + 'mocksecretshownonce0000000000000000' }
  }
  return proxyJson(event, 'platform', '/v1/auth/keys', { method: 'POST', body: { name: body?.name, scopes: body?.scopes || [] } })
})
