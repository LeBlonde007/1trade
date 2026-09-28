/**
 * Supply BFF helpers (F17). Source ids are UUIDs; anything else is refused before it reaches the
 * upstream path, so a crafted id can never steer the call to a different compute-control endpoint.
 */
import { createError, getRouterParam, type H3Event } from 'h3'

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** sourceId returns the validated :id route param, or throws 404. */
export function sourceId(event: H3Event): string {
  const id = getRouterParam(event, 'id') || ''
  if (!UUID.test(id)) throw createError({ statusCode: 404, statusMessage: 'no such source' })
  return id
}
