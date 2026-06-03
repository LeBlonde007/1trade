/**
 * useCompute — GPU instance lifecycle from the BFF (`/api/compute/*`, F13). Instances draw from the
 * same GPU pool the scheduler places jobs on; per-second GPU-hour usage debits the gpu_* credit while
 * an instance runs. Render IDs + counts mono + tabular-nums per the design system.
 */
export interface GpuType {
  id: string
  name: string
  gpu: string
  credit_type: string
  price_per_hour: string
  available: number
}

export interface Connect {
  ssh: string | null
  jupyter: string | null
  http: string | null
}

export interface Instance {
  id: string
  gpu_type: string
  count: number
  state: 'provisioning' | 'running' | 'stopping' | 'stopped' | 'starting' | 'terminated'
  image: string
  region: string
  connect: Connect
  supply_source_id: string
  idle_stop_minutes: number | null
  is_paper: boolean
  created_at: string
  started_at: string | null
}

/** CreateInstance is the create-instance request body (is_paper is server-derived, never sent). */
export interface CreateInstance {
  type: string
  count?: number
  image?: string
  region?: string
  idle_stop_minutes?: number | null
}

export function useCompute() {
  const types = useState<GpuType[]>('compute:types', () => [])
  const instances = useState<Instance[]>('compute:instances', () => [])
  const busy = useState<boolean>('compute:busy', () => false)

  /** loadTypes fetches the GPU-tier catalog with live availability. */
  async function loadTypes() {
    types.value = (await $fetch<{ types: GpuType[] }>('/api/compute/types')).types
    return types.value
  }

  /** loadInstances fetches the tenant's instances, optionally filtered by state. */
  async function loadInstances(state?: string) {
    const qs = state ? `?state=${encodeURIComponent(state)}` : ''
    instances.value = (await $fetch<{ instances: Instance[] }>(`/api/compute/instances${qs}`)).instances
    return instances.value
  }

  /** getInstance fetches one instance (with connection info). */
  function getInstance(id: string) {
    return $fetch<Instance>(`/api/compute/instances/${id}`)
  }

  /** create launches a new instance; the BFF mints the Idempotency-Key. */
  async function create(req: CreateInstance): Promise<Instance> {
    busy.value = true
    try {
      return await $fetch<Instance>('/api/compute/instances', { method: 'POST', body: req })
    } finally {
      busy.value = false
    }
  }

  /** stop / start / remove run a lifecycle action and return the updated instance. */
  function stop(id: string) {
    return action(`/api/compute/instances/${id}/stop`, 'POST')
  }
  function start(id: string) {
    return action(`/api/compute/instances/${id}/start`, 'POST')
  }
  function remove(id: string) {
    return action(`/api/compute/instances/${id}`, 'DELETE')
  }

  /** action runs a mutating lifecycle call, toggling the busy flag. */
  async function action(path: string, method: 'POST' | 'DELETE'): Promise<Instance> {
    busy.value = true
    try {
      return await $fetch<Instance>(path, { method })
    } finally {
      busy.value = false
    }
  }

  return { types, instances, busy, loadTypes, loadInstances, getInstance, create, stop, start, remove }
}
