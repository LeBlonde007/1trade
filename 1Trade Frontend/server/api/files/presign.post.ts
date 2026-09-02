/**
 * POST /api/files/presign — get a short-lived presigned PUT URL for a user upload (object storage on
 * DigitalOcean Spaces). The browser then PUTs the file bytes DIRECTLY to `upload_url`; nothing flows
 * through the BFF. Proxies platform-core /v1/files/presign (authed via the session cookie).
 */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

interface Body { filename?: string; content_type?: string }

export default defineEventHandler(async (event) => {
  const body = await readBody<Body>(event)
  return proxyJson(event, 'platform', '/v1/files/presign', {
    method: 'POST',
    body: { filename: body?.filename, content_type: body?.content_type },
  })
})
