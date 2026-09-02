/**
 * BFF helpers (F20). Every screen calls the Nitro BFF (`/api/**`), never a platform service
 * directly — that keeps requests same-origin (no CORS), hides internal URLs, and lets the session
 * JWT live in an httpOnly cookie the browser can't read. The BFF is always live: every route proxies
 * to the real platform service (no mock mode — see docs).
 */
import { getCookie, setCookie, deleteCookie, createError, type H3Event } from 'h3'

/** Name of the httpOnly cookie holding the platform JWT. */
export const SESSION_COOKIE = 'ex_session'

type Service = 'platform' | 'gateway' | 'ledger' | 'compute'

/** upstreamBase returns the base URL for a platform service from runtime config. */
function upstreamBase(event: H3Event, svc: Service): string {
  const c = useRuntimeConfig(event)
  return { platform: c.platformCoreUrl, gateway: c.gatewayUrl, ledger: c.ledgerUrl, compute: c.computeUrl }[svc] as string
}

/** sessionToken reads the platform JWT from the request's session cookie (empty when logged out). */
export function sessionToken(event: H3Event): string {
  return getCookie(event, SESSION_COOKIE) || ''
}

/** setSession stores the JWT in a secure httpOnly cookie. */
export function setSession(event: H3Event, token: string): void {
  setCookie(event, SESSION_COOKIE, token, {
    httpOnly: true,
    sameSite: 'lax',
    path: '/',
    secure: !import.meta.dev,
    maxAge: 60 * 60 * 24, // 24h, matches the platform token TTL
  })
}

/** clearSessionCookie removes the session cookie (logout). Named to avoid h3's `clearSession`. */
export function clearSessionCookie(event: H3Event): void {
  deleteCookie(event, SESSION_COOKIE, { path: '/' })
}

interface ProxyOpts {
  method?: string
  body?: unknown
  /** Override the bearer token (e.g. right after login, before the cookie is on the next request). */
  token?: string
  headers?: Record<string, string>
}

/**
 * proxyJson forwards a JSON request to an upstream platform service and returns the parsed body. It
 * attaches the session JWT (or an explicit `token`) as a bearer, and re-throws upstream non-2xx as an
 * h3 error so the client sees the real status (e.g. 401, 402 INSUFFICIENT_CREDIT).
 */
export async function proxyJson<T = unknown>(event: H3Event, svc: Service, path: string, opts: ProxyOpts = {}): Promise<T> {
  const headers: Record<string, string> = { 'content-type': 'application/json', ...(opts.headers || {}) }
  const token = opts.token || sessionToken(event)
  if (token && !headers.authorization) headers.authorization = `Bearer ${token}`

  const res = await fetch(upstreamBase(event, svc) + path, {
    method: opts.method || 'GET',
    headers,
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    throw createError({ statusCode: res.status, statusMessage: (data && data.message) || 'upstream error', data })
  }
  return data as T
}
