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

  return { sources, loading, error, load, usage, act, register }
}
