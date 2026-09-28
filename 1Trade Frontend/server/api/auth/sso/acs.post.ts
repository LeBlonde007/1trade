/**
 * POST /api/auth/sso/acs — the SAML Assertion Consumer Service. The identity provider form-posts
 * SAMLResponse + RelayState here; platform-core verifies it, and on success we set the session cookie
 * and send the browser to the console. Failures land on /login with a reason (never a stack).
 */
import { defineEventHandler, readBody, sendRedirect } from 'h3'
import { proxyJson, setSession } from '../../../utils/api'

export default defineEventHandler(async (event) => {
  const form = (await readBody<Record<string, string>>(event)) || {}
  try {
    const out = await proxyJson<{ token: string }>(event, 'platform', '/v1/auth/sso/acs', {
      method: 'POST', body: { saml_response: String(form.SAMLResponse ?? ''), relay_state: String(form.RelayState ?? '') },
    })
    setSession(event, out.token)
    return sendRedirect(event, '/console', 303)
  } catch (e: any) {
    const code = encodeURIComponent(String(e?.data?.code ?? 'sso_failed'))
    return sendRedirect(event, `/login?sso_error=${code}`, 303)
  }
})
