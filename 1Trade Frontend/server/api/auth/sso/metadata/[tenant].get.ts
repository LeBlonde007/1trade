/** GET /api/auth/sso/metadata/:tenant — our SAML service-provider metadata, for the IdP admin. */
import { createError, defineEventHandler, setHeader } from 'h3'
import { uuidParam } from '../../../../utils/ids'

export default defineEventHandler(async (event) => {
  const tenant = uuidParam(event, 'tenant')
  const res = await fetch(useRuntimeConfig(event).platformCoreUrl + '/v1/auth/sso/metadata/' + tenant)
  if (!res.ok) throw createError({ statusCode: res.status, statusMessage: 'single sign-on is not configured' })
  setHeader(event, 'content-type', 'application/samlmetadata+xml')
  return await res.text()
})
