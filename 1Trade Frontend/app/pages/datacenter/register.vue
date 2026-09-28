<script setup lang="ts">
/**
 * /datacenter/register — register GPU capacity with 1Trade (F17), live on compute-control's supply API.
 * The form holds one idempotency key for its lifetime, so a double submit or a retry after a network
 * error registers the source once. A new source is pending until 1Trade activates it.
 */
import type { Registration, SupplySource } from '~/composables/useSupply'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Datacenter · Register capacity — 1Trade' })

const { register } = useSupply()
const form = reactive<Registration>({ name: '', gpu_type: 'gpu_h100', gpu_count: 8, region: '', sla_tier: 'silver' })
const key = ref(globalThis.crypto?.randomUUID?.().replace(/-/g, '') ?? `reg${Date.now()}`)
const submitting = ref(false)
const error = ref('')
const created = ref<SupplySource | null>(null)

const valid = computed(() =>
  form.name.trim().length >= 1 && form.name.trim().length <= 80 &&
  form.region.trim().length >= 1 && form.region.trim().length <= 64 &&
  Number.isInteger(form.gpu_count) && form.gpu_count >= 1 && form.gpu_count <= 4096)

/** submit registers the source; a validation or permission error is shown inline. */
async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    created.value = await register({ ...form, name: form.name.trim(), region: form.region.trim() }, key.value)
  } catch (e: any) {
    error.value = e?.statusCode === 403
      ? 'Registering capacity needs the admin or engineer role on this account.'
      : (e?.data?.message || e?.statusMessage || 'Registration failed. Check the details and try again.')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="dc-page">
    <div class="subbar">
      <nav class="crumbs">
        <span>Account</span><span class="sep">›</span><NuxtLink to="/datacenter">Datacenter</NuxtLink>
        <span class="sep">›</span><span class="cur">Register capacity</span>
      </nav>
    </div>

    <main class="page">
      <section class="head">
        <h1>Register capacity</h1>
        <p class="sub">Bring a block of GPUs into the 1Trade pool. One source is one GPU tier in one location.</p>
      </section>

      <section v-if="created" class="card done" aria-live="polite">
        <h2>Registered — pending activation</h2>
        <p class="small">
          <strong>{{ created.name }}</strong> ({{ created.gpu_count }} × {{ created.gpu_type === 'gpu_h100' ? 'H100' : 'H200' }},
          {{ created.region }}) is registered. 1Trade activates it after review; it takes work once it is active and
          your agent is heartbeating.
        </p>
        <p class="small">Source id <span class="mono">{{ created.id }}</span> — your agent heartbeats to it:</p>
        <pre class="mono code">curl -X POST https://api.1trade.io/v1/supply/sources/{{ created.id }}/heartbeat \
  -H "Authorization: Bearer $ONETRADE_TOKEN" \
  -d '{"gpus_healthy": {{ created.gpu_count }}, "utilization_pct": 0, "ecc_errors": 0}'</pre>
        <NuxtLink to="/datacenter" class="btn-primary">Go to your supply dashboard</NuxtLink>
      </section>

      <form v-else class="card form" @submit.prevent="submit">
        <label>
          <span>Name</span>
          <input v-model="form.name" maxlength="80" required placeholder="e.g. Frankfurt row 7">
        </label>
        <div class="row">
          <label>
            <span>GPU</span>
            <select v-model="form.gpu_type">
              <option value="gpu_h100">NVIDIA H100 80GB</option>
              <option value="gpu_h200">NVIDIA H200 141GB</option>
            </select>
          </label>
          <label>
            <span>Number of GPUs</span>
            <input v-model.number="form.gpu_count" type="number" min="1" max="4096" step="1" required class="mono">
          </label>
        </div>
        <div class="row">
          <label>
            <span>Region / location</span>
            <input v-model="form.region" maxlength="64" required placeholder="e.g. eu-central-1a">
          </label>
          <label>
            <span>SLA tier</span>
            <select v-model="form.sla_tier">
              <option value="bronze">Bronze</option>
              <option value="silver">Silver</option>
              <option value="gold">Gold</option>
            </select>
          </label>
        </div>
        <p v-if="error" class="banner neg" role="alert">{{ error }}</p>
        <div class="foot">
          <span class="small dim">New sources start pending until 1Trade activates them.</span>
          <button type="submit" class="btn-primary" :disabled="!valid || submitting">{{ submitting ? 'Registering…' : 'Register' }}</button>
        </div>
      </form>
    </main>
  </div>
</template>

<style scoped>
.dc-page { min-height: 100%; background: var(--canvas); color: var(--text); }
.subbar { padding: 10px 24px; border-bottom: 1px solid var(--border); font-size: 12px; color: var(--text-2); }
.crumbs a { color: var(--text-2); }
.crumbs .sep { margin: 0 6px; color: var(--text-3); }
.crumbs .cur { color: var(--text); }
.page { max-width: 720px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 16px; }
h1 { font-family: var(--font-display); font-size: 26px; margin: 0; letter-spacing: -0.02em; }
h2 { font-size: 15px; margin: 0 0 8px; }
.sub { margin: 4px 0 0; color: var(--text-2); font-size: 13px; }
.card { background: var(--elevated); border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
.form { display: flex; flex-direction: column; gap: 14px; }
.row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-2); }
input, select { background: var(--canvas); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; color: var(--text); font-size: 14px; }
input:focus, select:focus { outline: none; border-color: var(--border-focus); }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.foot { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.btn-primary { background: var(--brand); color: var(--text-on-accent); border: 0; padding: 8px 14px; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; text-decoration: none; display: inline-block; }
.btn-primary:disabled { opacity: 0.5; cursor: default; }
.banner { margin: 0; padding: 8px 12px; border-radius: 6px; font-size: 13px; }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.small { font-size: 13px; }
.dim { color: var(--text-2); }
.code { background: var(--sunken); border: 1px solid var(--border); border-radius: 6px; padding: 10px; font-size: 12px; overflow-x: auto; white-space: pre; }
.done { display: flex; flex-direction: column; gap: 8px; align-items: flex-start; }
@media (max-width: 600px) { .row { grid-template-columns: 1fr; } }
</style>
