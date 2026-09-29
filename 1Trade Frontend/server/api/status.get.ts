/**
 * GET /api/status — a live health check of every platform service, for the public status page.
 *
 * Each service's /readyz is called with a short timeout; the answer is `operational` (2xx) or
 * `outage` (error, timeout, non-2xx), with the round-trip time. Nothing is remembered between calls:
 * there is no uptime history or incident log yet, so none is claimed. Public — it reveals only
 * whether each service answers.
 */
import { defineEventHandler, type H3Event } from 'h3'
import { upstreamBase, type Service } from '../utils/api'

const CHECKS: { svc: Service; group: string; name: string }[] = [
  { svc: 'trading', group: 'Trading', name: 'Paper exchange (matching engine)' },
  { svc: 'ledger', group: 'Account', name: 'Wallet & credit ledger' },
  { svc: 'platform', group: 'Account', name: 'Sign-in, billing & accounts' },
  { svc: 'gateway', group: 'Developer', name: 'Inference API' },
  { svc: 'compute', group: 'Compute', name: 'GPU compute & datacenter supply' },
]

/** check calls one service's /readyz and times it. */
async function check(event: H3Event, c: (typeof CHECKS)[number]) {
  const started = Date.now()
  try {
    const res = await fetch(upstreamBase(event, c.svc) + '/readyz', { signal: AbortSignal.timeout(3000) })
    return { group: c.group, name: c.name, status: res.ok ? 'operational' : 'outage', latency_ms: Date.now() - started }
  } catch {
    return { group: c.group, name: c.name, status: 'outage', latency_ms: null }
  }
}

export default defineEventHandler(async (event) => {
  const components = await Promise.all(CHECKS.map(c => check(event, c)))
  return { checked_at: new Date().toISOString(), components }
})
