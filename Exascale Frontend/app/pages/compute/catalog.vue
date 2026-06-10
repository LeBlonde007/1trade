<script setup lang="ts">
/**
 * GPU rental catalog — the published lineup (Hopper → Ada → Blackwell → Grace-Blackwell racks) with
 * datasheet specs, per-GPU-hour pricing, and marketplace availability (available / sold out / coming
 * soon). Live from compute-control /v1/compute/types via the BFF; billed in per-tier GPU credits.
 */
import type { GpuType } from '~/composables/useCompute'

definePageMeta({ layout: 'app' })
useHead({ title: 'Compute · GPU catalog — Exascale' })

const compute = useCompute()
const types = ref<GpuType[]>([])
const loading = ref(true)

onMounted(async () => {
  try { types.value = await compute.loadTypes() } catch { /* keep empty → empty state */ } finally { loading.value = false }
})

const STATUS_META: Record<string, { label: string; cls: string }> = {
  available: { label: 'Available now', cls: 'st-available' },
  sold_out: { label: 'Sold out', cls: 'st-soldout' },
  coming_soon: { label: 'Coming soon', cls: 'st-coming' },
}
function statusMeta(s?: string) { return STATUS_META[s ?? 'available'] ?? STATUS_META.available! }
function fmtPrice(p: string) { return Number(p).toFixed(2) }

const counts = computed(() => {
  const c = { available: 0, coming_soon: 0, sold_out: 0 }
  for (const t of types.value) {
    const k = (t.status ?? 'available') as keyof typeof c
    if (k in c) c[k]++
  }
  return c
})

// The ordered datasheet rows shown per card (label → spec key).
const SPEC_ROWS: Array<{ label: string; key: keyof NonNullable<GpuType['specs']> }> = [
  { label: 'VRAM', key: 'vram' },
  { label: 'Mem bandwidth', key: 'mem_bandwidth' },
  { label: 'FP16 dense', key: 'fp16_tflops' },
  { label: 'FP8 dense', key: 'fp8_tflops' },
  { label: 'FP4 dense', key: 'fp4_tflops' },
  { label: 'NVLink', key: 'nvlink' },
  { label: 'Interconnect', key: 'interconnect' },
  { label: 'TDP', key: 'tdp' },
  { label: 'Form factor', key: 'form_factor' },
  { label: 'vCPUs', key: 'vcpus' },
  { label: 'Host RAM', key: 'host_ram' },
  { label: 'Released', key: 'released' },
]
function specVal(t: GpuType, key: keyof NonNullable<GpuType['specs']>): string {
  const v = t.specs?.[key]
  return v === undefined || v === null || v === '' ? '—' : String(v)
}
</script>

<template>
  <div class="cat-page">
    <div class="subbar">
      <div class="subbar-left">
        <NuxtLink to="/compute" class="crumb">Compute</NuxtLink>
        <span class="sep">/</span>
        <span class="strong">GPU catalog</span>
      </div>
      <div class="subbar-right">
        <NuxtLink to="/compute" class="btn ghost">Instances</NuxtLink>
      </div>
    </div>

    <div class="cat-body">
      <header class="cat-head">
        <div>
          <h1 class="page-title">GPU rental</h1>
          <p class="page-sub">
            NVIDIA accelerators billed by the GPU-hour in per-tier GPU credits — Hopper, Ada, Blackwell,
            and Grace-Blackwell NVL72 racks. Reserve on-demand; capacity is drawn from owned + partner
            datacenters.
          </p>
        </div>
        <div v-if="!loading" class="cat-counts mono">
          <span class="cc cc-av">{{ counts.available }} available</span>
          <span class="cc cc-cs">{{ counts.coming_soon }} coming soon</span>
          <span class="cc cc-so">{{ counts.sold_out }} sold out</span>
        </div>
      </header>

      <div v-if="loading" class="cat-loading">Loading catalog…</div>
      <p v-else-if="!types.length" class="cat-empty">Catalog unavailable right now.</p>

      <div v-else class="cat-grid">
        <article v-for="t in types" :key="t.id" class="gpu-card" :class="{ dim: t.status === 'sold_out' }">
          <div class="gc-top">
            <div class="gc-id">
              <h2 class="gc-name">{{ t.name }}</h2>
              <span class="gc-arch">{{ t.specs?.architecture ?? '—' }}</span>
            </div>
            <span class="st-pill" :class="statusMeta(t.status).cls">{{ statusMeta(t.status).label }}</span>
          </div>
          <p class="gc-full">{{ t.gpu }}</p>

          <div class="gc-price">
            <span class="gc-amt mono">${{ fmtPrice(t.price_per_hour) }}</span>
            <span class="gc-unit">/ GPU-hour</span>
            <span v-if="t.status === 'available'" class="gc-avail mono">· {{ t.available }} ready</span>
          </div>

          <dl class="gc-specs">
            <div v-for="row in SPEC_ROWS" :key="row.key" class="gc-row">
              <dt>{{ row.label }}</dt>
              <dd class="mono">{{ specVal(t, row.key) }}</dd>
            </div>
          </dl>

          <div class="gc-cta">
            <NuxtLink v-if="t.status === 'available'" :to="`/compute/new?type=${t.id}`" class="btn primary block">
              Rent now
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square"><path d="M3 8h10M9 4l4 4-4 4" /></svg>
            </NuxtLink>
            <button v-else-if="t.status === 'coming_soon'" type="button" class="btn block" disabled>Coming soon</button>
            <button v-else type="button" class="btn block" disabled>Sold out</button>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cat-page { display: flex; flex-direction: column; height: 100%; min-height: 0; }
.subbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 20px; border-bottom: 1px solid var(--border); flex-shrink: 0;
}
.subbar-left { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.crumb { color: var(--muted, var(--text)); text-decoration: none; }
.crumb:hover { color: var(--text); }
.sep { color: var(--muted, var(--text)); }
.strong { color: var(--text); font-weight: 600; }

.cat-body { padding: 24px 20px 48px; overflow-y: auto; }
.cat-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; margin-bottom: 24px; flex-wrap: wrap; }
.page-title { font-size: 22px; font-weight: 650; margin: 0 0 6px; color: var(--text); letter-spacing: -0.01em; }
.page-sub { font-size: 13px; color: var(--muted, var(--text)); max-width: 640px; line-height: 1.55; margin: 0; }
.cat-counts { display: flex; gap: 14px; font-size: 12px; font-variant-numeric: tabular-nums; }
.cc { display: inline-flex; align-items: center; gap: 6px; }
.cc::before { content: ''; width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
.cc-av { color: var(--success, var(--accent)); }
.cc-cs { color: var(--warn); }
.cc-so { color: var(--muted, var(--text)); }

.cat-loading, .cat-empty { color: var(--muted, var(--text)); font-size: 13px; padding: 24px 0; }

.cat-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.gpu-card {
  display: flex; flex-direction: column; gap: 12px;
  border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--surface, var(--canvas));
  padding: 16px;
}
.gpu-card.dim { opacity: 0.62; }
.gc-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.gc-id { display: flex; flex-direction: column; gap: 3px; }
.gc-name { font-size: 16px; font-weight: 650; margin: 0; color: var(--text); }
.gc-arch { font-size: 11px; letter-spacing: 0.03em; color: var(--muted, var(--text)); text-transform: uppercase; }
.gc-full { font-size: 12px; color: var(--muted, var(--text)); margin: -4px 0 0; }

.st-pill { font-size: 10.5px; font-weight: 600; letter-spacing: 0.02em; padding: 3px 8px; border-radius: 999px; white-space: nowrap; border: 1px solid transparent; }
.st-available { color: var(--success, var(--accent)); border-color: var(--success, var(--accent)); }
.st-coming { color: var(--warn); border-color: var(--warn); }
.st-soldout { color: var(--muted, var(--text)); border-color: var(--border); }

.gc-price { display: flex; align-items: baseline; gap: 6px; padding-bottom: 4px; border-bottom: 1px solid var(--border); }
.gc-amt { font-size: 20px; font-weight: 650; color: var(--text); font-variant-numeric: tabular-nums; }
.gc-unit { font-size: 12px; color: var(--muted, var(--text)); }
.gc-avail { font-size: 11px; color: var(--success, var(--accent)); margin-left: auto; }

.gc-specs { margin: 0; display: grid; grid-template-columns: 1fr 1fr; gap: 6px 14px; }
.gc-row { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; min-width: 0; }
.gc-row dt { font-size: 11px; color: var(--muted, var(--text)); white-space: nowrap; }
.gc-row dd { font-size: 11.5px; color: var(--text); margin: 0; text-align: right; font-variant-numeric: tabular-nums; }

.gc-cta { margin-top: auto; padding-top: 4px; }
.btn { display: inline-flex; align-items: center; justify-content: center; gap: 6px; font-size: 13px; padding: 8px 14px; border-radius: var(--radius-sm); border: 1px solid var(--border); background: none; color: var(--text); cursor: pointer; text-decoration: none; }
.btn.block { width: 100%; }
.btn.ghost { font-size: 12px; padding: 5px 12px; }
.btn.primary { background: var(--accent); border-color: var(--accent); color: var(--accent-contrast, var(--canvas)); font-weight: 600; }
.btn.primary svg { width: 14px; height: 14px; }
.btn:disabled { cursor: not-allowed; color: var(--muted, var(--text)); }
.btn:not(:disabled):hover { border-color: var(--accent); }
</style>
