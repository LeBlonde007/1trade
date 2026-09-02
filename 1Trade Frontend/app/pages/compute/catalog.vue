<script setup lang="ts">
/**
 * GPU rental catalog — the published lineup (Hopper → Ada → Blackwell → Grace-Blackwell racks) with
 * datasheet specs, per-GPU-hour pricing, and availability. Same selection UX as the inference model
 * catalog: a search box, status filter chips, and selectable cards. Live from compute-control
 * /v1/compute/types via the BFF; billed in per-tier GPU credits.
 */
import type { GpuType } from '~/composables/useCompute'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Compute · GPU catalog — 1Trade' })

const compute = useCompute()
const types = ref<GpuType[]>([])
const loading = ref(true)
const query = ref('')
const filter = ref<'all' | 'available' | 'coming_soon' | 'sold_out'>('all')

onMounted(async () => {
  try { types.value = await compute.loadTypes() } catch { /* empty state */ } finally { loading.value = false }
})

const STATUS_META: Record<string, { label: string; cls: string }> = {
  available: { label: 'Available', cls: 'st-available' },
  sold_out: { label: 'Sold out', cls: 'st-soldout' },
  coming_soon: { label: 'Coming soon', cls: 'st-coming' },
}
function statusMeta(s?: string) { return STATUS_META[s ?? 'available'] ?? STATUS_META.available! }
function fmtPrice(p: string) { return Number(p).toFixed(2) }

const FILTERS: Array<{ k: typeof filter.value; label: string }> = [
  { k: 'all', label: 'All' },
  { k: 'available', label: 'Available' },
  { k: 'coming_soon', label: 'Coming soon' },
  { k: 'sold_out', label: 'Sold out' },
]

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return types.value.filter((t) => {
    if (filter.value !== 'all' && (t.status ?? 'available') !== filter.value) return false
    if (q && !`${t.name} ${t.gpu} ${t.specs?.architecture ?? ''}`.toLowerCase().includes(q)) return false
    return true
  })
})

const SPEC_ROWS: Array<{ label: string; key: keyof NonNullable<GpuType['specs']> }> = [
  { label: 'VRAM', key: 'vram' },
  { label: 'Bandwidth', key: 'mem_bandwidth' },
  { label: 'FP8', key: 'fp8_tflops' },
  { label: 'FP4', key: 'fp4_tflops' },
  { label: 'NVLink', key: 'nvlink' },
  { label: 'TDP', key: 'tdp' },
  { label: 'vCPUs', key: 'vcpus' },
  { label: 'Host RAM', key: 'host_ram' },
]
function specVal(t: GpuType, key: keyof NonNullable<GpuType['specs']>): string {
  const v = t.specs?.[key]
  return v === undefined || v === null || v === '' ? '—' : String(v)
}
</script>

<template>
  <div class="cat-page">
    <div class="subbar">
      <nav class="crumbs">
        <NuxtLink to="/compute" class="crumb">Compute</NuxtLink>
        <span class="sep">›</span>
        <span class="strong">GPU catalog</span>
      </nav>
      <NuxtLink to="/compute" class="btn ghost">Instances</NuxtLink>
    </div>

    <main class="cat-body">
      <header class="cat-head-row">
        <div>
          <div class="eyebrow"><span class="dot" /> Compute · GPU rentals</div>
          <h1 class="page-title">GPU rental</h1>
          <p class="page-sub">NVIDIA accelerators billed by the GPU-hour in per-tier GPU credits — Hopper, Ada, Blackwell, and Grace-Blackwell NVL72 racks. Reserve on-demand; capacity is drawn from owned + partner datacenters.</p>
        </div>
      </header>

      <!-- Selection header — mirrors the inference model catalog: search + filter chips. -->
      <div class="catalog-head">
        <div class="search">
          <span class="search-ic" aria-hidden="true">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square"><circle cx="7" cy="7" r="4.5" /><path d="M10.4 10.4L14 14" /></svg>
          </span>
          <input v-model="query" type="text" class="search-input" :placeholder="`Search ${types.length} GPUs…`" autocomplete="off" />
        </div>
        <div class="filter-row">
          <button v-for="f in FILTERS" :key="f.k" type="button" class="filter-chip" :class="{ active: filter === f.k }" @click="filter = f.k">{{ f.label }}</button>
        </div>
      </div>

      <p v-if="loading" class="cat-empty">Loading catalog…</p>
      <p v-else-if="!filtered.length" class="cat-empty"><span class="mono">No GPUs match "{{ query }}"</span></p>

      <div v-else class="catalog-list">
        <article v-for="t in filtered" :key="t.id" class="model-card" :class="{ dim: t.status === 'sold_out' }">
          <div class="mc-head">
            <h4 class="mc-name">{{ t.name }}</h4>
            <span class="cat-tag" :class="statusMeta(t.status).cls">{{ statusMeta(t.status).label }}</span>
          </div>
          <div class="mc-provider">
            <span>{{ t.specs?.architecture ?? '—' }}</span>
            <span v-if="t.status === 'available'" class="mc-version mono">{{ t.available }} ready</span>
          </div>
          <div class="mc-price mono">${{ fmtPrice(t.price_per_hour) }} <span class="dim">/ GPU-hour</span></div>

          <dl class="mc-specs">
            <div v-for="row in SPEC_ROWS" :key="row.key" class="mc-spec">
              <dt>{{ row.label }}</dt>
              <dd class="mono">{{ specVal(t, row.key) }}</dd>
            </div>
          </dl>

          <NuxtLink v-if="t.status === 'available'" :to="`/compute/new?type=${t.id}`" class="mc-cta">
            Rent now
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
          </NuxtLink>
          <span v-else-if="t.status === 'coming_soon'" class="mc-disabled mono">Coming soon</span>
          <span v-else class="mc-disabled mono">Sold out</span>
        </article>
      </div>
    </main>
  </div>
</template>

<style scoped>
.cat-page { display: flex; flex-direction: column; height: 100%; min-height: 0; }
.subbar { display: flex; align-items: center; justify-content: space-between; padding: 10px 20px; border-bottom: 1px solid var(--border); flex-shrink: 0; }
.crumbs { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.crumb { color: var(--text-3); text-decoration: none; }
.crumb:hover { color: var(--text); }
.sep { color: var(--text-3); }
.strong { color: var(--text); font-weight: 600; }
.btn.ghost { font-size: 12px; padding: 5px 12px; border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text); text-decoration: none; }
.btn.ghost:hover { border-color: var(--accent); }

.cat-body { padding: 22px 20px 48px; overflow-y: auto; }
.eyebrow { display: inline-flex; align-items: center; gap: 7px; font-size: 11px; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-3); margin-bottom: 8px; }
.eyebrow .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--brand); }
.page-title { font-size: 22px; font-weight: 650; margin: 0 0 6px; color: var(--text); letter-spacing: -0.01em; }
.page-sub { font-size: 13px; color: var(--text-3); max-width: 660px; line-height: 1.55; margin: 0 0 20px; }

/* Selection header — copied from the inference model catalog so the two read identically. */
.catalog-head { display: flex; flex-direction: column; gap: 10px; margin-bottom: 16px; max-width: 720px; }
.search { display: grid; grid-template-columns: 28px 1fr; align-items: center; height: 34px; background: var(--elevated); border: 1px solid var(--border-strong); border-radius: var(--radius-sm); }
.search:focus-within { border-color: var(--accent); }
.search-ic { color: var(--text-3); display: grid; place-items: center; }
.search-ic svg { width: 14px; height: 14px; }
.search-input { background: transparent; border: 0; outline: 0; color: var(--text); font-family: var(--font-sans); font-size: 13px; width: 100%; padding: 0 12px 0 0; }
.search-input::placeholder { color: var(--text-3); }
.filter-row { display: flex; flex-wrap: wrap; gap: 4px; }
.filter-chip { background: transparent; border: 1px solid var(--border); color: var(--text-3); font-family: var(--font-mono); font-size: 10px; letter-spacing: 0.06em; text-transform: uppercase; font-weight: 600; padding: 3px 8px; border-radius: var(--radius-sm); cursor: pointer; }
.filter-chip:hover { color: var(--text); border-color: var(--border-strong); }
.filter-chip.active { background: var(--text); color: var(--canvas); border-color: var(--text); }

.cat-empty { color: var(--text-3); font-size: 13px; padding: 20px 0; }

/* Card grid — same .model-card aesthetic as the inference catalog. */
.catalog-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(296px, 1fr)); gap: 10px; }
.model-card { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 14px; display: flex; flex-direction: column; gap: 8px; transition: border-color 120ms; }
.model-card:hover { border-color: var(--border-strong); }
.model-card.dim { opacity: 0.6; }
.mc-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; }
.mc-name { font-size: 15px; font-weight: 650; margin: 0; color: var(--text); }
.cat-tag { font-size: 9.5px; font-weight: 700; letter-spacing: 0.04em; text-transform: uppercase; padding: 3px 7px; border-radius: 4px; white-space: nowrap; border: 1px solid transparent; }
.st-available { color: var(--success, var(--brand)); border-color: var(--success, var(--brand)); }
.st-coming { color: var(--warn); border-color: var(--warn); }
.st-soldout { color: var(--text-3); border-color: var(--border); }
.mc-provider { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; font-size: 12px; color: var(--text-3); }
.mc-version { font-size: 11px; color: var(--success, var(--brand)); }
.mc-price { font-size: 16px; font-weight: 650; color: var(--text); font-variant-numeric: tabular-nums; }
.mc-price .dim { font-size: 11px; font-weight: 400; color: var(--text-3); }
.mc-specs { margin: 2px 0 0; display: grid; grid-template-columns: 1fr 1fr; gap: 5px 14px; padding-top: 8px; border-top: 1px solid var(--border); }
.mc-spec { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; min-width: 0; }
.mc-spec dt { font-size: 10.5px; color: var(--text-3); white-space: nowrap; }
.mc-spec dd { font-size: 11px; color: var(--text); margin: 0; text-align: right; font-variant-numeric: tabular-nums; }
.mc-cta { margin-top: 4px; display: inline-flex; align-items: center; justify-content: center; gap: 6px; font-size: 13px; font-weight: 600; padding: 8px; border-radius: var(--radius-sm); background: var(--brand); border: 1px solid var(--brand); color: var(--canvas); text-decoration: none; }
.mc-cta svg { width: 14px; height: 14px; }
.mc-disabled { margin-top: 4px; text-align: center; font-size: 12px; padding: 8px; border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-3); }
</style>
