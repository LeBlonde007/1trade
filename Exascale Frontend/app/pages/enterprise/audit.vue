<script setup lang="ts">
/**
 * /enterprise/audit — the rich audit-trail screen (F03). The append-only, tenant-scoped record of
 * sensitive admin actions (role changes, budgets, org edits, key + purchase events), live from
 * platform-core `/v1/account/audit` (admin-only, a SOC 2 control). Dense, filterable, with the
 * before→after diff for each change. No mock data — the audit log is append-only (not hash-chained;
 * the hash chain is the credit ledger), so this shows the real fields the backend records.
 */
import { compact } from '~/utils/format'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Audit log — Exascale' })

interface AuditEntry {
  id: string
  action: string
  target_type?: string
  target_id?: string
  actor_id?: string
  is_paper: boolean
  created_at: string
  before?: Record<string, unknown>
  after?: Record<string, unknown>
}

const entries = useState<AuditEntry[]>('audit:entries', () => [])
const loading = ref(false)
const forbidden = ref(false)
const error = ref('')
const limit = ref(100)

// ── Filters ─────────────────────────────────────────────────────────────────────────────────
const q = ref('')
const actionFilter = ref('all')
const targetFilter = ref('all')

const actions = computed(() => [...new Set(entries.value.map((e) => e.action))].sort())
const targets = computed(() => [...new Set(entries.value.map((e) => e.target_type).filter(Boolean) as string[])].sort())

const filtered = computed(() => {
  const term = q.value.trim().toLowerCase()
  return entries.value.filter((e) => {
    if (actionFilter.value !== 'all' && e.action !== actionFilter.value) return false
    if (targetFilter.value !== 'all' && e.target_type !== targetFilter.value) return false
    if (term) {
      const hay = `${e.action} ${e.target_type ?? ''} ${e.target_id ?? ''} ${e.actor_id ?? ''}`.toLowerCase()
      if (!hay.includes(term)) return false
    }
    return true
  })
})
const lastEvent = computed(() => entries.value[0]?.created_at ?? '')

// ── Load ────────────────────────────────────────────────────────────────────────────────────
async function load() {
  loading.value = true
  error.value = ''
  forbidden.value = false
  try {
    const res = await $fetch<{ entries: AuditEntry[] }>(`/api/account/audit?limit=${limit.value}`)
    entries.value = res.entries || []
  } catch (e: unknown) {
    const ex = e as { statusCode?: number; data?: { message?: string } }
    if (ex?.statusCode === 403) forbidden.value = true
    else error.value = ex?.data?.message || 'Could not load the audit log.'
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ── Expand diffs ────────────────────────────────────────────────────────────────────────────
const open = ref<Set<string>>(new Set())
function toggle(id: string) {
  const n = new Set(open.value)
  if (n.has(id)) n.delete(id)
  else n.add(id)
  open.value = n
}
function hasDiff(e: AuditEntry): boolean {
  return !!(e.before || e.after)
}
interface DiffPair { key: string; from: string; to: string }
function diffPairs(e: AuditEntry): DiffPair[] {
  const before = (e.before ?? {}) as Record<string, unknown>
  const after = (e.after ?? {}) as Record<string, unknown>
  const keys = [...new Set([...Object.keys(before), ...Object.keys(after)])].sort()
  const out: DiffPair[] = []
  for (const k of keys) {
    const f = JSON.stringify(before[k])
    const t = JSON.stringify(after[k])
    if (f !== t) out.push({ key: k, from: f ?? '—', to: t ?? '—' })
  }
  return out
}

// ── Format ──────────────────────────────────────────────────────────────────────────────────
function ts(t: string): string {
  const d = new Date(t)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
function actor(e: AuditEntry): string { return e.actor_id ? e.actor_id.slice(0, 8) : 'self-serve' }
function target(e: AuditEntry): string {
  if (!e.target_type) return '—'
  return e.target_id ? `${e.target_type}/${e.target_id.slice(0, 8)}` : e.target_type
}
</script>

<template>
  <div class="audit">
    <!-- Header -->
    <header class="head">
      <div class="titles">
        <div class="eyebrow">Compliance · Audit trail</div>
        <h1 class="title">Audit log</h1>
        <p class="sub">Append-only record of sensitive actions in this account. Newest first · admin-only · SOC 2 control.</p>
      </div>
      <div class="meta mono">
        <span class="stat"><span class="n">{{ compact(filtered.length) }}</span><span class="l">shown</span></span>
        <span class="stat"><span class="n">{{ compact(actions.length) }}</span><span class="l">action types</span></span>
        <span v-if="lastEvent" class="stat"><span class="n">{{ ts(lastEvent).slice(11) }}</span><span class="l">last event</span></span>
      </div>
    </header>

    <!-- Admin-required / error states -->
    <div v-if="forbidden" class="panel state">
      <div class="state-ic">🔒</div>
      <div>
        <div class="state-h">Administrator access required</div>
        <p class="state-p">The audit trail is restricted to account administrators. Ask an admin on your team for access.</p>
      </div>
    </div>
    <div v-else-if="error" class="panel state neg">{{ error }}</div>

    <template v-else>
      <!-- Filters -->
      <div class="filters">
        <input v-model="q" class="input search" placeholder="Search action, target, actor…" />
        <select v-model="actionFilter" class="input">
          <option value="all">All actions</option>
          <option v-for="a in actions" :key="a" :value="a">{{ a }}</option>
        </select>
        <select v-model="targetFilter" class="input">
          <option value="all">All targets</option>
          <option v-for="t in targets" :key="t" :value="t">{{ t }}</option>
        </select>
        <div class="spacer" />
        <select v-model.number="limit" class="input" @change="load">
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="250">250</option>
          <option :value="500">500</option>
        </select>
        <button class="btn ghost" :disabled="loading" @click="load">{{ loading ? 'Loading…' : 'Refresh' }}</button>
      </div>

      <!-- Table -->
      <div class="panel">
        <table class="tbl">
          <thead>
            <tr>
              <th class="w-time">Time (local)</th>
              <th>Actor</th>
              <th>Action</th>
              <th>Target</th>
              <th>Mode</th>
              <th class="w-diff" />
            </tr>
          </thead>
          <tbody>
            <template v-for="e in filtered" :key="e.id">
              <tr class="row" :class="{ exp: open.has(e.id), can: hasDiff(e) }" @click="hasDiff(e) && toggle(e.id)">
                <td class="mono muted nowrap">{{ ts(e.created_at) }}</td>
                <td class="mono">{{ actor(e) }}</td>
                <td><span class="action mono">{{ e.action }}</span></td>
                <td class="mono muted">{{ target(e) }}</td>
                <td><span class="chip" :class="e.is_paper ? 'paper' : 'live'">{{ e.is_paper ? 'sandbox' : 'live' }}</span></td>
                <td class="w-diff"><span v-if="hasDiff(e)" class="chev" :class="{ o: open.has(e.id) }">›</span></td>
              </tr>
              <tr v-if="open.has(e.id)" :key="e.id + '-d'" class="diffrow">
                <td colspan="6">
                  <div class="diff">
                    <div v-for="d in diffPairs(e)" :key="d.key" class="dpair">
                      <span class="dkey mono">{{ d.key }}</span>
                      <span class="dfrom mono">{{ d.from }}</span>
                      <span class="darr">→</span>
                      <span class="dto mono">{{ d.to }}</span>
                    </div>
                    <div v-if="!diffPairs(e).length" class="muted">No field-level changes recorded.</div>
                  </div>
                </td>
              </tr>
            </template>
            <tr v-if="!filtered.length && !loading"><td colspan="6" class="empty">No audit entries{{ entries.length ? ' match the filters' : ' yet' }}.</td></tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<style scoped>
.audit {
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  padding: var(--sp-5);
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  max-width: 1280px;
  margin: 0 auto;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.muted { color: var(--text-3); }
.nowrap { white-space: nowrap; }

/* Header */
.head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); gap: var(--sp-5); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 6px; line-height: 1; }
.sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0; max-width: 560px; }
.meta { display: flex; gap: var(--sp-5); }
.stat { display: flex; flex-direction: column; align-items: flex-end; }
.stat .n { font-size: var(--fs-xl); font-weight: 600; }
.stat .l { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); }

/* Filters */
.filters { display: flex; gap: var(--sp-2); align-items: center; }
.input { background: var(--elevated); color: var(--text); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--sp-2) var(--sp-3); font-size: var(--fs-sm); font-family: var(--font-sans); height: 34px; }
.input:focus { outline: none; border-color: var(--border-focus); }
.search { flex: 0 1 320px; }
.spacer { flex: 1; }
.btn { height: 34px; padding: 0 var(--sp-4); border-radius: var(--radius-sm); font-size: var(--fs-sm); font-weight: 600; cursor: pointer; border: 1px solid var(--border-strong); background: transparent; color: var(--text); }
.btn.ghost:hover { background: var(--overlay); }
.btn:disabled { opacity: 0.5; cursor: default; }

/* Panel + table */
.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden; }
.tbl { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.tbl th { text-align: left; font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); font-weight: 500; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.tbl td { padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); }
.w-time { width: 180px; }
.w-diff { width: 28px; text-align: center; }
.row { transition: background 120ms; }
.row.can { cursor: pointer; }
.row.can:hover td, .row.exp td { background: var(--overlay); }
.action { font-size: var(--fs-xs); background: var(--overlay); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 2px var(--sp-2); color: var(--text); }
.chip { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.1em; padding: 2px var(--sp-2); border-radius: var(--radius-sm); }
.chip.live { background: var(--pos-soft); color: var(--pos); }
.chip.paper { color: var(--info); border: 1px solid var(--border); }
.chev { display: inline-block; color: var(--text-3); font-size: var(--fs-md); transition: transform 140ms; }
.chev.o { transform: rotate(90deg); color: var(--text); }

/* Diff */
.diffrow td { background: var(--canvas); padding: 0 !important; }
.diff { padding: var(--sp-3) var(--sp-4) var(--sp-3) 36px; display: flex; flex-direction: column; gap: 6px; }
.dpair { display: grid; grid-template-columns: 160px 1fr auto 1fr; gap: var(--sp-3); align-items: baseline; font-size: var(--fs-xs); }
.dkey { color: var(--text-2); }
.dfrom { color: var(--neg); word-break: break-all; }
.dto { color: var(--pos); word-break: break-all; }
.darr { color: var(--text-3); }

.empty { padding: var(--sp-5) !important; text-align: center; color: var(--text-3); }

/* States */
.state { display: flex; gap: var(--sp-4); align-items: center; padding: var(--sp-5); }
.state.neg { color: var(--neg); }
.state-ic { font-size: 22px; }
.state-h { font-size: var(--fs-md); font-weight: 600; }
.state-p { font-size: var(--fs-sm); color: var(--text-2); margin: 4px 0 0; max-width: 460px; }

@media (max-width: 900px) {
  .head { flex-direction: column; align-items: flex-start; gap: var(--sp-3); }
  .meta { gap: var(--sp-4); }
  .filters { flex-wrap: wrap; }
}
</style>
