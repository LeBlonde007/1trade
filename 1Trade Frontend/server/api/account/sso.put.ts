/** PUT /api/account/sso — save the IdP metadata, email domains and policy (admin). Contract fields only. */
import { defineEventHandler, readBody } from 'h3'
import { proxyJson } from '../../utils/api'

export default defineEventHandler(async (event) => {
  const b = (await readBody(event)) || {}
  return proxyJson(event, 'platform', '/v1/account/sso', {
    method: 'PUT',
    body: { idp_metadata_xml: b.idp_metadata_xml, email_domains: b.email_domains, default_role: b.default_role, jit: b.jit, enforce: !!b.enforce },
  })
})
