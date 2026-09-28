/**
 * Route-param validation for BFF proxies: an id that is not a UUID is refused before it can reach an
 * upstream path.
 */
import { createError, getRouterParam, type H3Event } from 'h3'

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** uuidParam returns the named route param when it is a UUID, or throws 404. */
export function uuidParam(event: H3Event, name = 'id'): string {
  const id = getRouterParam(event, name) || ''
  if (!UUID.test(id)) throw createError({ statusCode: 404, statusMessage: 'not found' })
  return id
}
