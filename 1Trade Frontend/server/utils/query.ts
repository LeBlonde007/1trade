/**
 * boundedLimit turns an untrusted `?limit` into a `?limit=N` suffix for an upstream call, or '' when it is
 * absent or outside [1, max] — the upstream default then applies. Keeps the BFF from forwarding arbitrary
 * query text to a platform service.
 */
export function boundedLimit(raw: unknown, max: number): string {
  const n = Number(raw)
  return Number.isInteger(n) && n >= 1 && n <= max ? `?limit=${n}` : ''
}
