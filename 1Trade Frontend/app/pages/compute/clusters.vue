<script setup lang="ts">
/**
 * /compute/clusters — multi-node InfiniBand clusters (F15), live on compute-control. A cluster is
 * whole 8-GPU nodes on one fabric, placed all at once or not at all. Up to 32 GPUs is self-serve;
 * larger clusters are arranged with sales.
 */
import type { Cluster } from '~/composables/useCompute'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Compute · Clusters — 1Trade' })

const compute = useCompute()
const list = ref<Cluster[]>([])
const loading = ref(true)
const error = ref('')
const form = reactive<{ gpu_type: string; gpus: number; topology: Cluster['topology'] }>({ gpu_type: 'gpu_h100', gpus: 16, topology: 'fat-tree' })
const key = ref('')
const creating = ref(false)
const createError = ref('')
const busy = ref('')
const TIER: Record<string, string> = { gpu_h100: 'H100', gpu_h200: 'H200' }

/** newKey mints the idempotency key for the order in the form. */
const newKey = () => { key.value = globalThis.crypto?.randomUUID?.().replace(/-/g, '') ?? `cl${Date.now()}` }
watch(form, newKey, { deep: true })

/** load fetches the tenant's clusters (terminated ones are hidden). */
async function load() {
  error.value = ''
  try {
    list.value = (await compute.clusters()).filter(c => c.state !== 'terminated')
  } catch (e: any) {
    error.value = e?.data?.message || e?.statusMessage || 'Could not load your clusters.'
  } finally {
    loading.value = false
  }
}

/** create places a cluster; a 402 means no single fabric has that many GPUs free right now. */
async function create() {
  if (creating.value) return
  creating.value = true
  createError.value = ''
  try {
    await compute.createCluster({ gpu_type: form.gpu_type, gpus: form.gpus, network: 'infiniband', topology: form.topology }, key.value)
    newKey()
    await load()
  } catch (e: any) {
    createError.value = e?.statusCode === 402
      ? 'No single datacenter fabric has that many GPUs free right now. A cluster is never split or partly placed — try fewer GPUs or later.'
      : (e?.data?.message || e?.statusMessage || 'The cluster could not be created.')
  } finally {
    creating.value = false
  }
}

/** remove terminates a cluster after confirmation. */
async function remove(c: Cluster) {
  if (!confirm(`Terminate cluster ${c.id}? All ${c.nodes.length} nodes stop together; this cannot be undone.`)) return
  busy.value = c.id
  try {
    await compute.deleteCluster(c.id)
    await load()
  } catch (e: any) {
    error.value = e?.data?.message || e?.statusMessage || 'Could not terminate the cluster.'
  } finally {
    busy.value = ''
  }
}

onMounted(() => {
  newKey()
  load()
})
</script>

<template>
  <div class="cl-page">
    <div class="subbar">
      <nav class="crumbs">
        <span>Account</span><span class="sep">›</span><NuxtLink to="/compute">Compute</NuxtLink>
        <span class="sep">›</span><span class="cur">Clusters</span>
      </nav>
    </div>

    <main class="page">
      <section class="head">
        <h1>Clusters</h1>
        <p class="sub">
          Multi-node GPU clusters on one InfiniBand fabric for distributed training. All nodes start together or the
          request is refused — never a partial cluster. Billed per GPU-second like instances; reserved GPUs are used first.
        </p>
      </section>

      <form class="card form" @submit.prevent="create">
        <h2>New cluster</h2>
        <div class="row">
          <label><span>GPU</span>
            <select v-model="form.gpu_type">
              <option value="gpu_h100">NVIDIA H100 80GB</option>
              <option value="gpu_h200">NVIDIA H200 141GB</option>
            </select>
          </label>
          <label><span>Size</span>
            <select v-model.number="form.gpus" class="mono">
              <option :value="16">16 GPUs · 2 nodes</option>
              <option :value="24">24 GPUs · 3 nodes</option>
              <option :value="32">32 GPUs · 4 nodes</option>
            </select>
          </label>
          <label><span>Topology</span>
            <select v-model="form.topology">
              <option value="fat-tree">Fat-tree</option>
              <option value="rail-optimized">Rail-optimized</option>
            </select>
          </label>
        </div>
        <p v-if="createError" class="banner neg" role="alert">{{ createError }}</p>
        <div class="foot">
          <span class="small dim">Over 32 GPUs (up to 256)? Clusters that size are arranged with our team.</span>
          <button type="submit" class="btn-primary" :disabled="creating">{{ creating ? 'Placing…' : 'Create cluster' }}</button>
        </div>
      </form>

      <section class="card">
        <header class="card-head"><h2>Your clusters</h2></header>
        <p v-if="error" class="banner neg" role="alert">{{ error }}</p>
        <div v-if="loading" class="empty">Loading…</div>
        <div v-else-if="!list.length && !error" class="empty">No clusters running.</div>
        <div v-for="c in list" :key="c.id" class="cluster">
          <div class="c-head">
            <div>
              <div class="mono name">{{ c.id }}</div>
              <div class="small dim">
                {{ c.gpus }} × {{ TIER[c.gpu_type] ?? c.gpu_type }} · {{ c.nodes.length }} nodes · InfiniBand {{ c.topology }} ·
                {{ c.region }}<template v-if="c.reserved"> · reserved (prepaid)</template>
              </div>
            </div>
            <span class="pill pos">{{ c.state }}</span>
            <button type="button" class="danger" :disabled="busy === c.id" @click="remove(c)">Terminate</button>
          </div>
          <ul class="nodes">
            <li v-for="n in c.nodes" :key="n.name"><span class="mono dim">{{ n.name }}</span> <code class="mono">{{ n.ssh }}</code></li>
          </ul>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.cl-page { min-height: 100%; background: var(--canvas); color: var(--text); }
.subbar { padding: 10px 24px; border-bottom: 1px solid var(--border); font-size: 12px; color: var(--text-2); }
.crumbs a { color: var(--text-2); }
.crumbs .sep { margin: 0 6px; color: var(--text-3); }
.crumbs .cur { color: var(--text); }
.page { max-width: 1040px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 16px; }
h1 { font-family: var(--font-display); font-size: 26px; margin: 0; letter-spacing: -0.02em; }
h2 { font-size: 15px; margin: 0 0 10px; }
.sub { margin: 4px 0 0; color: var(--text-2); font-size: 13px; max-width: 760px; }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
.form { display: flex; flex-direction: column; gap: 12px; }
.row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-2); }
select { background: var(--canvas); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; color: var(--text); font-size: 14px; }
.foot { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.btn-primary { background: var(--brand); color: var(--text-on-accent); border: 0; padding: 9px 14px; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; }
.btn-primary:disabled { opacity: 0.5; cursor: default; }
.danger { background: transparent; border: 1px solid var(--neg); color: var(--neg); border-radius: 6px; padding: 5px 10px; font-size: 12px; cursor: pointer; }
.banner { margin: 0; padding: 8px 12px; border-radius: 6px; font-size: 13px; }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.small { font-size: 12px; }
.dim { color: var(--text-2); }
.empty { padding: 16px 0; color: var(--text-2); font-size: 13px; }
.cluster { border-top: 1px solid var(--border); padding: 12px 0; }
.c-head { display: flex; align-items: center; gap: 12px; }
.c-head > div { flex: 1; }
.name { font-size: 14px; }
.nodes { list-style: none; margin: 8px 0 0; padding: 0; display: flex; flex-direction: column; gap: 4px; font-size: 12px; }
.nodes code { background: var(--sunken); border: 1px solid var(--border); border-radius: 4px; padding: 2px 6px; }
.pill { display: inline-block; font-size: 11px; padding: 1px 8px; border-radius: 999px; border: 1px solid currentColor; }
.pill.pos { color: var(--pos); }
@media (max-width: 760px) { .row { grid-template-columns: 1fr; } }
</style>
