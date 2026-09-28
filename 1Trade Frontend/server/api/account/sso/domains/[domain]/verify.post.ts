/** POST /api/account/sso/domains/:domain/verify — check the domain's DNS TXT proof (admin). */
import { createError, defineEventHandler, getRouterParam } from 'h3'
import { proxyJson } from '../../../../../utils/api'

const DOMAIN = /^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/

export default defineEventHandler((event) => {
  const d = (getRouterParam(event, 'domain') || '').toLowerCase()
  if (!DOMAIN.test(d)) throw createError({ statusCode: 404, statusMessage: 'not found' })
  return proxyJson(event, 'platform', `/v1/account/sso/domains/${d}/verify`, { method: 'POST' })
})
