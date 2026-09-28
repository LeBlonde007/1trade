/**
 * useSupply — the partner's supply sources (supply.yaml v1.0, F16/F17) through the BFF. Live only:
 * every value comes from compute-control; there is no mock data.
 */
export interface SupplySource {
  id: string
  name: string
  gpu_type: 'gpu_h100' | 'gpu_h200'
  gpu_count: number
  region: string
  sla_tier: 'bronze' | 'silver' | 'gold'
  state: 'pending' | 'active' | 'suspended' | 'retired'
  schedulable: boolean
  gpus_healthy: number | null
  gpus_in_use: number
  utilization_pct: number | null
  last_heartbeat_at: string | null
  created_at: string
}

export interface SourceUsage {
  source_id: string
  from: string
  to: string
  by_tier: { gpu_type: string; gpu_seconds: string; units: string; sessions: number }[]
}

export interface PayoutLine { gpu_type: string; gpu_hours: string; rate: string; gross: string }

export interface Payout {
  id: string
  period_start: string
  period_end: string
  currency: 'USD'
  is_paper: boolean
  gpu_hours: string
  gross: string
  fee: string
  payout: string
  holdback: string
  released: string
  usage_records: number
  state: 'pending' | 'wired' | 'disputed' | 'settled'
  wire_reference: string | null
  dispute_until: string
  dispute_reason: string | null
  resolution: 'release' | 'withhold' | null
  lines: PayoutLine[]
}

export interface Agreement {
  rates: Record<string, string>
  fee_percent: string
  holdback_percent: string
  dispute_days: number
}

export interface Registration {
  name: string
  gpu_type: 'gpu_h100' | 'gpu_h200'
  gpu_count: number
  region: string
  sla_tier: 'bronze' | 'silver' | 'gold'
}

export function useSupply() {
  const sources = useState<SupplySource[]>('supply:sources', () => [])
  const loading = ref(false)
  const error = ref('')

  /** load fetches the partner's sources. */
  async function load() {
    loading.value = true
    error.value = ''
    try {
      const r = await $fetch<{ data: SupplySource[] }>('/api/supply/sources')
      sources.value = r.data
    } catch (e: any) {
      error.value = e?.statusCode === 403
        ? 'Managing supply needs the admin or engineer role on this account.'
        : (e?.data?.message || e?.statusMessage || 'Could not load your supply sources.')
    } finally {
      loading.value = false
    }
  }

  /** usage fetches the last 30 days of GPU time served on one source. */
  function usage(id: string) {
    return $fetch<SourceUsage>(`/api/supply/sources/${id}/usage`)
  }

  /** act runs suspend | resume | retire and reloads. */
  async function act(id: string, action: 'suspend' | 'resume' | 'retire') {
    if (action === 'retire') await $fetch(`/api/supply/sources/${id}`, { method: 'DELETE' })
    else await $fetch(`/api/supply/sources/${id}/${action}`, { method: 'POST' })
    await load()
  }

  /** register submits a new source under a caller-held idempotency key (safe to retry). */
  function register(reg: Registration, key: string) {
    return $fetch<SupplySource>('/api/supply/sources', { method: 'POST', body: reg, headers: { 'Idempotency-Key': key } })
  }

  /** payouts fetches the partner's statements; agreement its terms (null until 1Trade sets them). */
  const payouts = () => $fetch<{ data: Payout[] }>('/api/supply/payouts').then(r => r.data)
  const agreement = () => $fetch<Agreement>('/api/supply/agreement').catch((e: any) => {
    if (e?.statusCode === 404) return null
    throw e
  })
  /** dispute raises a dispute on a statement inside its window. */
  const dispute = (id: string, reason: string) =>
    $fetch<Payout>(`/api/supply/payouts/${id}/dispute`, { method: 'POST', body: { reason } })

  return { sources, loading, error, load, usage, act, register, payouts, agreement, dispute }
}
