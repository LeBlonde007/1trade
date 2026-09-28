/**
 * Compute BFF helpers (F14/F15). Ids and idempotency keys are checked here, so a crafted value can
 * never steer the upstream path or smuggle a header.
 */
import { createError, getHeader, getRouterParam, type H3Event } from 'h3'

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const CLUSTER = /^c-[0-9a-f]{8}$/
const KEY = /^[A-Za-z0-9_-]{8,64}$/

/** reservationId returns the validated :id param (a UUID), or throws 404. */
export function reservationId(event: H3Event): string {
  const id = getRouterParam(event, 'id') || ''
  if (!UUID.test(id)) throw createError({ statusCode: 404, statusMessage: 'no such reservation' })
  return id
}

/** clusterId returns the validated :id param (c-xxxxxxxx), or throws 404. */
export function clusterId(event: H3Event): string {
  const id = getRouterParam(event, 'id') || ''
  if (!CLUSTER.test(id)) throw createError({ statusCode: 404, statusMessage: 'no such cluster' })
  return id
}

/**
 * idempotencyKey is the caller's Idempotency-Key when well-formed (so a retry after "payment pending"
 * reuses it), else a fresh one.
 */
export function idempotencyKey(event: H3Event): string {
  const k = getHeader(event, 'idempotency-key') || ''
  return KEY.test(k) ? k : (globalThis.crypto?.randomUUID?.() ?? `web_${Date.now()}`)
}
