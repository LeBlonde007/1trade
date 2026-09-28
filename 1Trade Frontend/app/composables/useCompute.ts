/**
 * useCompute — GPU instance lifecycle from the BFF (`/api/compute/*`, F13). Instances draw from the
 * same GPU pool the scheduler places jobs on; per-second GPU-hour usage debits the gpu_* credit while
 * an instance runs. Render IDs + counts mono + tabular-nums per the design system.
 */
export interface GpuSpecs {
  architecture: string
  vram: string
  mem_bandwidth: string
  fp16_tflops: string
  fp8_tflops: string
  fp4_tflops?: string
  nvlink: string
  interconnect: string
  tdp: string
  form_factor: string
  vcpus: number
  host_ram: string
  released: string
}

export interface GpuType {
  id: string
  name: string
  gpu: string
  credit_type: string
  price_per_hour: string
  available: number
  status?: 'available' | 'sold_out' | 'coming_soon'
  specs?: GpuSpecs
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
  reserved?: boolean
  created_at: string
  started_at: string | null
}

export type Term = '1mo' | '6mo' | '12mo'

/** ReservationQuote is a reservation's price before buying (GPU credits, fixed-point strings). */
export interface ReservationQuote {
  gpu_type: string
  gpus: number
  term: Term
  hours: number
  discount_pct: number
  gpu_hours: string
  on_demand_credits: string
  price_credits: string
  saving_credits: string
}

/** Reservation is prepaid reserved capacity (F14). */
export interface Reservation {
  id: string
  is_paper: boolean
  gpu_type: string
  gpus: number
  term: Term
  discount_pct: number
  gpu_hours: string
  price_credits: string
  state: 'pending_payment' | 'active' | 'expired' | 'failed'
  starts_at: string | null
  ends_at: string | null
  failure: string | null
  created_at: string
}

/** ReservedCapacity is, per tier, the GPUs set aside and how many running instances occupy. */
export interface ReservedCapacity { gpu_type: string; is_paper: boolean; reserved_gpus: number; in_use_gpus: number }

/** Cluster is a multi-node InfiniBand cluster (F15). */
export interface Cluster {
  id: string
  gpu_type: string
  gpus: number
  network: 'infiniband'
  topology: 'fat-tree' | 'rail-optimized'
  state: Instance['state']
  nodes: { name: string; ssh: string }[]
  region: string
  reserved: boolean
  supply_source_id: string
  is_paper: boolean
  created_at: string
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

  /** reservations fetches the tenant's reservations and reserved-vs-occupied GPUs per tier. */
  const reservations = () =>
    $fetch<{ data: Reservation[]; capacity: ReservedCapacity[] }>('/api/compute/reservations')
  /** quote prices a reservation (nothing is held or charged). */
  const quote = (gpuType: string, gpus: number, term: Term) =>
    $fetch<{ quote: ReservationQuote; available: number }>('/api/compute/reservations/quote', { query: { gpu_type: gpuType, gpus, term } })
  /** reserve buys a reservation under a caller-held key, so a retry never charges twice. */
  const reserve = (gpuType: string, gpus: number, term: Term, key: string) =>
    $fetch<Reservation>('/api/compute/reservations', { method: 'POST', body: { gpu_type: gpuType, gpus, term }, headers: { 'Idempotency-Key': key } })

  /** clusters lists the tenant's clusters; createCluster places one (all nodes or none). */
  const clusters = () => $fetch<{ clusters: Cluster[] }>('/api/compute/clusters').then(r => r.clusters)
  const createCluster = (req: { gpu_type: string; gpus: number; network: 'infiniband'; topology: Cluster['topology'] }, key: string) =>
    $fetch<Cluster>('/api/compute/clusters', { method: 'POST', body: req, headers: { 'Idempotency-Key': key } })
  const deleteCluster = (id: string) => $fetch<Cluster>(`/api/compute/clusters/${id}`, { method: 'DELETE' })

  return {
    types, instances, busy, loadTypes, loadInstances, getInstance, create, stop, start, remove,
    reservations, quote, reserve, clusters, createCluster, deleteCluster,
  }
}
