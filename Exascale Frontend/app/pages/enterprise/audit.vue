<script setup lang="ts">
/**
 * /enterprise/audit — Audit Log (D3)
 *
 * Light-mode enterprise admin surface. Immutable, hash-chained event log.
 * Compliance officers + auditors view + export.
 */
definePageMeta({ layout: false })
useHead({ title: 'Audit Log · Walmart Inc. — Exascale', htmlAttrs: { 'data-theme': 'light' } })

interface AuditRow {
  id: string
  ts: string                     // ISO with ms
  actor: string
  actorRole?: string
  subAccount: string             // '—' for org-level
  action: string                 // e.g. 'ORDER_PLACED'
  category: Category
  resource: string
  result: 'success' | 'fail'
  ip: string
  block: number
  hash: string                   // 11-char display
  prevHash: string
  userAgent?: string
  geo?: string
}

type Category = 'auth' | 'trading' | 'compute' | 'account' | 'security' | 'system'

const ORG = {
  name: 'Walmart Inc.',
  enterpriseId: 'ent_wmt_8412',
  schemaVersion: 'v1.2',
}

const rows: AuditRow[] = [
  { id: 'e_14287', ts: '2026-05-24T14:32:08.412Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'AI-Research-Team', action: 'ORDER_PLACED',                 category: 'trading',  resource: 'EAI-IDX · market buy · 5,000 credits @ ~$0.001005', result: 'success', ip: '198.51.100.42', block: 14287, hash: '7a3f9c1e021', prevHash: '6d29a40e7e0', userAgent: 'ExascaleDesktop/1.4 · macOS 14.5', geo: 'Bentonville, AR · US' },
  { id: 'e_14286', ts: '2026-05-24T14:31:55.108Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'AI-Research-Team', action: 'ORDER_FILLED',                 category: 'trading',  resource: 'ord_8c2a48f1 · 5,000 @ avg $0.001005 · fee $0.05',  result: 'success', ip: '198.51.100.42', block: 14286, hash: '9e0d8a1ac11', prevHash: '7a3f9c1e021', userAgent: 'system · matching-engine', geo: 'us-east-1 · matching' },
  { id: 'e_14285', ts: '2026-05-24T14:28:43.901Z', actor: 'jane.doe@walmart.com',       actorRole: 'Owner',                         subAccount: '—',                action: 'USER_INVITED',                 category: 'account',  resource: 'raj.patel@walmart.com → Trader · AI-Research-Team', result: 'success', ip: '203.0.113.18',  block: 14285, hash: '4b22f5ac910', prevHash: '9e0d8a1ac11', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'Bentonville, AR · US' },
  { id: 'e_14284', ts: '2026-05-24T14:14:02.337Z', actor: 'okta-sync@walmart.com',      actorRole: 'Service · IdP',                 subAccount: '—',                action: 'SSO_CONFIG_UPDATED',           category: 'auth',     resource: 'SAML metadata v3 · sha256:8a7f… · tenant walmart-prod', result: 'success', ip: '10.142.0.4',   block: 14284, hash: '8f8123c7d92', prevHash: '4b22f5ac910', userAgent: 'okta-scim/2.0',               geo: 'okta · us-prod' },
  { id: 'e_14283', ts: '2026-05-24T13:58:21.044Z', actor: 'priya.shah@walmart.com',     actorRole: 'Compliance Officer',            subAccount: '—',                action: 'AUDIT_EXPORTED',               category: 'security', resource: 'CSV · 90-day range · 8,412 rows · signed bundle',   result: 'success', ip: '198.51.100.74', block: 14283, hash: '2a900f6b03f', prevHash: '8f8123c7d92', userAgent: 'Safari 17 · macOS 14.5',      geo: 'New York, NY · US' },
  { id: 'e_14282', ts: '2026-05-24T13:45:11.226Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'Inference-Prod',   action: 'API_KEY_CREATED',              category: 'security', resource: 'sk_live_…f3a2 · scopes inference:read · inference:write', result: 'success', ip: '198.51.100.42', block: 14282, hash: '6d1402c4e88', prevHash: '2a900f6b03f', userAgent: 'ExascaleDesktop/1.4',         geo: 'Bentonville, AR · US' },
  { id: 'e_14281', ts: '2026-05-24T13:22:09.711Z', actor: 'patrick.obrien@walmart.com', actorRole: 'CTO · Owner',                   subAccount: '—',                action: 'BUDGET_LIMIT_UPDATED',         category: 'account',  resource: 'AI-Research-Team · $50,000 → $75,000 / month',      result: 'success', ip: '192.0.2.55',   block: 14281, hash: '1c77b1aa229', prevHash: '6d1402c4e88', userAgent: 'Chrome 124 · Windows 11',     geo: 'Bentonville, AR · US' },
  { id: 'e_14280', ts: '2026-05-24T13:09:47.612Z', actor: 'linda.martinez@walmart.com', actorRole: 'Procurement',                   subAccount: '—',                action: 'CREDIT_PURCHASE',              category: 'trading',  resource: '$100,000.00 USD wire → 99,503,000 AI credits @ $0.001005', result: 'success', ip: '192.0.2.91',  block: 14280, hash: 'b341a2ce0a4', prevHash: '1c77b1aa229', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'Bentonville, AR · US' },
  { id: 'e_14279', ts: '2026-05-24T12:55:38.220Z', actor: 'derek.thompson@walmart.com', actorRole: 'Trader · Inference-Dev',        subAccount: '—',                action: 'LOGIN_FAILED',                 category: 'auth',     resource: 'invalid password · attempt 2 of 3 (1 before lockout)', result: 'fail',    ip: '203.0.113.214', block: 14279, hash: '5e2b9fcc8f1', prevHash: 'b341a2ce0a4', userAgent: 'Chrome 124 · Windows 11',     geo: 'San Bruno, CA · US' },
  { id: 'e_14278', ts: '2026-05-24T12:54:11.099Z', actor: 'derek.thompson@walmart.com', actorRole: 'Trader · Inference-Dev',        subAccount: '—',                action: 'LOGIN_SUCCESS',                category: 'auth',     resource: 'WebAuthn · Touch ID · MacBook Pro 14"',             result: 'success', ip: '203.0.113.214', block: 14278, hash: '9aa2b1ec017', prevHash: '5e2b9fcc8f1', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'San Bruno, CA · US' },
  { id: 'e_14277', ts: '2026-05-24T12:38:54.408Z', actor: 'jane.doe@walmart.com',       actorRole: 'Owner',                         subAccount: '—',                action: 'ROLE_ASSIGNED',                category: 'account',  resource: 'marcus.chen@walmart.com · Trader → Admin',          result: 'success', ip: '203.0.113.18',  block: 14277, hash: '3f4ed5d88dd', prevHash: '9aa2b1ec017', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'Bentonville, AR · US' },
  { id: 'e_14276', ts: '2026-05-24T11:47:22.881Z', actor: 'system@exascale.com',       actorRole: 'Service · ledger',              subAccount: '—',                action: 'COMPLIANCE_SCAN',              category: 'system',   resource: 'daily policy sweep · 0 violations · 14,277 blocks scanned', result: 'success', ip: '—',        block: 14276, hash: 'cc20faa91be', prevHash: '3f4ed5d88dd', userAgent: 'ledger-scanner/3.1',         geo: 'us-east-1 · audit-svc' },
  { id: 'e_14275', ts: '2026-05-24T11:31:15.504Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'Inference-Prod',   action: 'COMPUTE_INSTANCE_PROVISIONED', category: 'compute',  resource: 'inst_5a91d2 · H200 × 4 · us-east-1 · $13.96/hr',    result: 'success', ip: '198.51.100.42', block: 14275, hash: '7708e2bb22a', prevHash: 'cc20faa91be', userAgent: 'ExascaleDesktop/1.4',         geo: 'Bentonville, AR · US' },
  { id: 'e_14274', ts: '2026-05-24T10:54:09.220Z', actor: 'jane.doe@walmart.com',       actorRole: 'Owner',                         subAccount: '—',                action: 'IP_ALLOWLIST_UPDATED',         category: 'security', resource: 'added 203.0.113.0/24 · HQ Bentonville · justification on file', result: 'success', ip: '203.0.113.18', block: 14274, hash: '4109e4d3c7a', prevHash: '7708e2bb22a', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'Bentonville, AR · US' },
  { id: 'e_14273', ts: '2026-05-24T10:18:44.331Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'AI-Research-Team', action: 'ORDER_CANCELLED',              category: 'trading',  resource: 'ord_2c8e91a · TEXT-SPOT limit $0.001190 · 8,000 credits', result: 'success', ip: '198.51.100.42', block: 14273, hash: 'ad08c46e2f0', prevHash: '4109e4d3c7a', userAgent: 'ExascaleDesktop/1.4',         geo: 'Bentonville, AR · US' },
  { id: 'e_14272', ts: '2026-05-24T09:55:08.197Z', actor: 'jane.doe@walmart.com',       actorRole: 'Owner',                         subAccount: '—',                action: 'WEBHOOK_CONFIGURED',           category: 'security', resource: 'trade.filled → api.walmart.com/exas/hook · HMAC sha256', result: 'success', ip: '203.0.113.18',  block: 14272, hash: '6321bf80aa9', prevHash: 'ad08c46e2f0', userAgent: 'Chrome 124 · macOS 14.5',     geo: 'Bentonville, AR · US' },
  { id: 'e_14271', ts: '2026-05-24T09:40:01.000Z', actor: 'system@exascale.com',       actorRole: 'Service · scheduler',           subAccount: '—',                action: 'MAINTENANCE_WINDOW_NOTICE',    category: 'system',   resource: 'weekly read-only window · 2026-05-26 04:00 UTC · 30s expected', result: 'success', ip: '—',     block: 14271, hash: '92cb04f77f0', prevHash: '6321bf80aa9', userAgent: 'scheduler/2.4',              geo: 'us-east-1 · ops' },
  { id: 'e_14270', ts: '2026-05-24T08:58:33.412Z', actor: 'priya.shah@walmart.com',     actorRole: 'Compliance Officer',            subAccount: '—',                action: 'LOGIN_SUCCESS',                category: 'auth',     resource: 'SSO · Okta · tenant walmart-prod · session 8h',     result: 'success', ip: '198.51.100.74', block: 14270, hash: '5bb1e0f2208', prevHash: '92cb04f77f0', userAgent: 'Safari 17 · macOS 14.5',      geo: 'New York, NY · US' },
  { id: 'e_14269', ts: '2026-05-24T08:14:11.755Z', actor: 'patrick.obrien@walmart.com', actorRole: 'CTO · Owner',                   subAccount: '—',                action: 'API_KEY_REVOKED',              category: 'security', resource: 'sk_live_…77ee · reason: leaked in GitHub PR #2841', result: 'success', ip: '192.0.2.55',   block: 14269, hash: 'af337cdc041', prevHash: '5bb1e0f2208', userAgent: 'Chrome 124 · Windows 11',     geo: 'Bentonville, AR · US' },
  { id: 'e_14268', ts: '2026-05-24T07:02:55.198Z', actor: 'marcus.chen@walmart.com',    actorRole: 'Admin · AI-Research-Team',     subAccount: 'AI-Research-Team', action: 'ORDER_PLACED',                 category: 'trading',  resource: 'H100-SPOT · market buy · 8 GPU-hours @ ~$2.99',     result: 'success', ip: '198.51.100.42', block: 14268, hash: '80f4ad81ba8', prevHash: 'af337cdc041', userAgent: 'ExascaleDesktop/1.4',         geo: 'Bentonville, AR · US' },
]

// ---------- Filters ----------
const ACTIONS_GROUPED: { group: string; items: string[] }[] = [
  { group: 'Auth',     items: ['LOGIN_SUCCESS','LOGIN_FAILED','LOGOUT','MFA_ENABLED','SSO_CONFIG_UPDATED','WEBAUTHN_REGISTERED'] },
  { group: 'Trading',  items: ['ORDER_PLACED','ORDER_FILLED','ORDER_CANCELLED','POSITION_CLOSED','CREDIT_PURCHASE','CREDIT_CONVERSION'] },
  { group: 'Compute',  items: ['COMPUTE_INSTANCE_PROVISIONED','COMPUTE_INSTANCE_TERMINATED','COMPUTE_INSTANCE_RESIZED','INFERENCE_JOB_SUBMITTED'] },
  { group: 'Account',  items: ['USER_INVITED','USER_DEACTIVATED','ROLE_ASSIGNED','ROLE_REVOKED','BUDGET_LIMIT_UPDATED','PAYMENT_METHOD_ADDED'] },
  { group: 'Security', items: ['API_KEY_CREATED','API_KEY_REVOKED','IP_ALLOWLIST_UPDATED','WEBHOOK_CONFIGURED','KEY_ROTATED','AUDIT_EXPORTED','SECURITY_POLICY_CHANGED'] },
  { group: 'System',   items: ['COMPLIANCE_SCAN','MAINTENANCE_WINDOW_NOTICE','CHAIN_INTEGRITY_CHECK'] },
]

const ACTORS = [
  'all',
  'jane.doe@walmart.com',
  'marcus.chen@walmart.com',
  'priya.shah@walmart.com',
  'patrick.obrien@walmart.com',
  'linda.martinez@walmart.com',
  'derek.thompson@walmart.com',
  'okta-sync@walmart.com',
  'system@exascale.com',
]

const SUB_ACCOUNTS = ['all', '—', 'AI-Research-Team', 'Inference-Prod', 'Inference-Dev', 'Marketing-Image-Gen']
const DATE_RANGES = ['Last 24 hours', 'Today', 'Yesterday', 'Last 7 days', 'Last 30 days', 'Custom…']
const RESULT_FILTERS: { key: 'all' | 'success' | 'fail'; label: string }[] = [
  { key: 'all',     label: 'All' },
  { key: 'success', label: 'Success' },
  { key: 'fail',    label: 'Fail' },
]

const fDate     = ref<string>('Last 24 hours')
const fActor    = ref<string>('all')
const fAction   = ref<string>('all')
const fResource = ref<string>('')
const fSub      = ref<string>('all')
const fResult   = ref<'all' | 'success' | 'fail'>('all')

const filteredRows = computed(() => {
  const q = fResource.value.trim().toLowerCase()
  return rows.filter((r) => {
    if (fActor.value !== 'all' && r.actor !== fActor.value) return false
    if (fAction.value !== 'all' && r.action !== fAction.value) return false
    if (fSub.value !== 'all' && r.subAccount !== fSub.value) return false
    if (fResult.value !== 'all' && r.result !== fResult.value) return false
    if (q && !(r.resource.toLowerCase().includes(q) || r.action.toLowerCase().includes(q) || r.actor.toLowerCase().includes(q))) return false
    return true
  })
})

const visibleCount = computed(() => filteredRows.value.length)

// ---------- Stats ----------
const stats = computed(() => {
  const failed = rows.filter((r) => r.result === 'fail').length
  const actors = new Set(rows.map((r) => r.actor)).size
  return {
    events24h: '1,847',
    failed: failed.toString(),
    actors: actors.toString(),
    blocks: '14,287',
  }
})

// ---------- Expand row ----------
const expandedId = ref<string | null>(null)
function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

// ---------- Chain integrity (live "last check") ----------
const lastCheckAt = ref<Date>(new Date(Date.now() - 9 * 60 * 1000)) // 9 min ago
const nowTick = ref<number>(Date.now())
let nowTimer: ReturnType<typeof setInterval> | null = null

const lastCheckRelative = computed(() => {
  const diffSec = Math.max(1, Math.floor((nowTick.value - lastCheckAt.value.getTime()) / 1000))
  if (diffSec < 60) return diffSec + 's ago'
  const m = Math.floor(diffSec / 60)
  if (m < 60) return m + ' min ago'
  const h = Math.floor(m / 60)
  return h + 'h ago'
})

const lastCheckAbs = computed(() => {
  const d = lastCheckAt.value
  const h = String(d.getUTCHours()).padStart(2, '0')
  const m = String(d.getUTCMinutes()).padStart(2, '0')
  return h + ':' + m + ' UTC'
})

onMounted(() => {
  nowTimer = setInterval(() => { nowTick.value = Date.now() }, 30_000)
})
onBeforeUnmount(() => {
  if (nowTimer) clearInterval(nowTimer)
})

// ---------- Helpers ----------
function timeOnly(iso: string): string {
  // returns HH:MM:SS.mmm in UTC
  return iso.slice(11, 23)
}
function dateOnly(iso: string): string {
  return iso.slice(0, 10)
}
function shortActor(email: string): string {
  return email.replace('@walmart.com', '@walmart').replace('@exascale.com', '@exascale')
}

const copyState = ref<string | null>(null)
function copyHash(hash: string, id: string, ev: Event) {
  ev.stopPropagation()
  if (navigator.clipboard) navigator.clipboard.writeText(hash).catch(() => {})
  copyState.value = id
  setTimeout(() => { if (copyState.value === id) copyState.value = null }, 1400)
}

function categoryLabel(c: Category): string {
  return c.charAt(0).toUpperCase() + c.slice(1)
}
</script>

<template>
  <div class="audit-page" data-theme="light">
    <!-- Admin chrome -->
    <header class="admin-chrome">
      <NuxtLink to="/" class="brand"><span class="mark" />Exascale</NuxtLink>
      <NuxtLink to="/enterprise/onboarding" class="org" :title="ORG.enterpriseId">
        <span class="org-name">{{ ORG.name }}</span>
        <span class="org-pill">ENTERPRISE</span>
      </NuxtLink>
      <nav class="chrome-nav">
        <NuxtLink to="/enterprise/onboarding" class="ch-link">Onboarding</NuxtLink>
        <NuxtLink to="/enterprise/teams" class="ch-link">Team</NuxtLink>
        <a href="#" class="ch-link active" aria-current="page">Audit Log</a>
        <NuxtLink to="/enterprise/billing" class="ch-link">Billing</NuxtLink>
      </nav>
      <div class="who">
        <span class="who-email mono">jane.doe@walmart.com</span>
        <span class="avatar">J</span>
      </div>
    </header>

    <!-- Breadcrumb -->
    <div class="crumb">
      <NuxtLink to="/enterprise/onboarding">Admin</NuxtLink>
      <span class="sep">›</span>
      <span class="cur">Audit Log</span>
    </div>

    <!-- Page head -->
    <section class="page-head">
      <div class="ph-left">
        <h1>Audit Log</h1>
        <p class="ph-sub">
          <span class="mono">{{ ORG.name }}</span>
          <span class="dot-sep">·</span>
          <span class="mono">14,287 events on the chain</span>
          <span class="dot-sep">·</span>
          <span class="mono">schema {{ ORG.schemaVersion }}</span>
        </p>
      </div>
      <div class="ph-right">
        <div class="integrity" :title="'Last full re-verification at ' + lastCheckAbs">
          <span class="int-label">Chain integrity</span>
          <span class="int-dot" />
          <span class="int-status">Verified</span>
          <span class="int-meta mono">· last check {{ lastCheckRelative }} ({{ lastCheckAbs }})</span>
        </div>
        <div class="exports">
          <button class="btn ghost" type="button">
            <svg viewBox="0 0 16 16" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M8 2v8m0 0 3-3m-3 3-3-3M2 13h12" /></svg>
            Export CSV
          </button>
          <button class="btn ghost" type="button">
            <svg viewBox="0 0 16 16" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M3 3h7l3 3v7H3z" /><path d="M10 3v3h3" /></svg>
            Export JSON
          </button>
          <button class="btn primary" type="button">
            <svg viewBox="0 0 16 16" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><circle cx="8" cy="8" r="6" /><path d="M8 5v3l2 1" /></svg>
            Schedule SIEM export
          </button>
        </div>
      </div>
    </section>

    <!-- Stats strip -->
    <section class="stats">
      <div class="stat">
        <div class="stat-label">Events · 24h</div>
        <div class="stat-val mono">{{ stats.events24h }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Failed · 24h</div>
        <div class="stat-val mono" :class="{ neg: parseInt(stats.failed) > 0 }">{{ stats.failed }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Unique actors · 24h</div>
        <div class="stat-val mono">{{ stats.actors }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Chain blocks (total)</div>
        <div class="stat-val mono">{{ stats.blocks }}</div>
      </div>
    </section>

    <!-- Filter bar -->
    <section class="filters">
      <div class="f">
        <label class="f-label">Date range</label>
        <select v-model="fDate" class="f-input">
          <option v-for="d in DATE_RANGES" :key="d" :value="d">{{ d }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-label">Actor</label>
        <select v-model="fActor" class="f-input mono">
          <option value="all">All actors</option>
          <option v-for="a in ACTORS.slice(1)" :key="a" :value="a">{{ a }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-label">Action</label>
        <select v-model="fAction" class="f-input mono">
          <option value="all">All actions</option>
          <optgroup v-for="g in ACTIONS_GROUPED" :key="g.group" :label="g.group">
            <option v-for="a in g.items" :key="a" :value="a">{{ a }}</option>
          </optgroup>
        </select>
      </div>
      <div class="f f-grow">
        <label class="f-label">Resource search</label>
        <input v-model="fResource" type="text" class="f-input" placeholder="email, order id, instance id, hash…" />
      </div>
      <div class="f">
        <label class="f-label">Sub-account</label>
        <select v-model="fSub" class="f-input">
          <option v-for="s in SUB_ACCOUNTS" :key="s" :value="s">{{ s === 'all' ? 'All sub-accounts' : s }}</option>
        </select>
      </div>
      <div class="f">
        <label class="f-label">Result</label>
        <div class="chips">
          <button v-for="r in RESULT_FILTERS" :key="r.key" type="button" class="chip" :class="{ on: fResult === r.key }" @click="fResult = r.key">{{ r.label }}</button>
        </div>
      </div>
    </section>

    <!-- Table -->
    <section class="tbl-wrap">
      <table class="audit-tbl">
        <thead>
          <tr>
            <th class="chev"></th>
            <th class="th-time">Time (UTC)</th>
            <th>Actor</th>
            <th>Sub-account</th>
            <th>Action</th>
            <th>Resource</th>
            <th>Result</th>
            <th>IP</th>
            <th>Block · Hash</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="row in filteredRows" :key="row.id">
            <tr class="row" :class="{ open: expandedId === row.id, fail: row.result === 'fail' }" @click="toggleExpand(row.id)">
              <td class="chev">
                <svg viewBox="0 0 12 12" width="10" height="10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="m4 2 4 4-4 4" /></svg>
              </td>
              <td class="mono ts">
                {{ timeOnly(row.ts) }}
                <span class="ts-date">{{ dateOnly(row.ts) }}</span>
              </td>
              <td class="actor">
                <span class="actor-email mono">{{ shortActor(row.actor) }}</span>
                <span v-if="row.actorRole" class="actor-role">{{ row.actorRole }}</span>
              </td>
              <td>
                <span class="sub" :class="{ none: row.subAccount === '—' }">{{ row.subAccount }}</span>
              </td>
              <td>
                <span class="act-chip mono" :class="'cat-' + row.category">{{ row.action }}</span>
              </td>
              <td class="resource">{{ row.resource }}</td>
              <td>
                <span class="result" :class="row.result">
                  <span class="r-dot" />{{ row.result }}
                </span>
              </td>
              <td class="mono ip">{{ row.ip }}</td>
              <td class="hash-cell">
                <span class="block mono">#{{ row.block }}</span>
                <span class="hash mono" :title="row.hash + '… (full hash on detail panel)'">{{ row.hash }}…</span>
                <button class="copy" type="button" :title="'Copy hash ' + row.hash" @click="copyHash(row.hash, row.id, $event)">
                  <svg v-if="copyState !== row.id" viewBox="0 0 12 12" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" /><path d="M3 3V2h6v6h-1" /></svg>
                  <svg v-else viewBox="0 0 12 12" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="m2.5 6 2.5 2.5L10 4" /></svg>
                </button>
              </td>
            </tr>

            <!-- Expanded detail row -->
            <tr v-if="expandedId === row.id" class="detail-row">
              <td colspan="9">
                <div class="detail">
                  <div class="d-left">
                    <div class="d-section">
                      <div class="d-head">Event context</div>
                      <dl class="kv">
                        <dt>Block</dt><dd class="mono">#{{ row.block }} of 14,287</dd>
                        <dt>Event ID</dt><dd class="mono">{{ row.id }}</dd>
                        <dt>Timestamp</dt><dd class="mono">{{ row.ts }}</dd>
                        <dt>Actor</dt><dd class="mono">{{ row.actor }} <span class="muted">· {{ row.actorRole }}</span></dd>
                        <dt>Sub-account</dt><dd>{{ row.subAccount }}</dd>
                        <dt>Action</dt><dd><span class="act-chip mono" :class="'cat-' + row.category">{{ row.action }}</span> <span class="muted">· {{ categoryLabel(row.category) }}</span></dd>
                        <dt>Resource</dt><dd>{{ row.resource }}</dd>
                        <dt>Result</dt><dd><span class="result" :class="row.result"><span class="r-dot" />{{ row.result }}</span></dd>
                        <dt>IP</dt><dd class="mono">{{ row.ip }} <span v-if="row.geo" class="muted">· {{ row.geo }}</span></dd>
                        <dt>User agent</dt><dd class="mono small">{{ row.userAgent }}</dd>
                      </dl>
                    </div>
                  </div>

                  <div class="d-right">
                    <div class="d-section proof">
                      <div class="d-head">Cryptographic proof</div>
                      <dl class="kv">
                        <dt>Previous hash</dt>
                        <dd class="mono small">{{ row.prevHash }}3e4a7c12d9f0… <span class="muted">(block #{{ row.block - 1 }})</span></dd>
                        <dt>This hash</dt>
                        <dd class="mono small strong">{{ row.hash }}8c3201ba9e4f… <span class="muted">(block #{{ row.block }})</span></dd>
                        <dt>Algorithm</dt>
                        <dd class="mono small">SHA-256 over canonical-JSON · ed25519 signature</dd>
                        <dt>Signed by</dt>
                        <dd class="mono small">exascale-audit-key-2026q2 · key id <span class="strong">kid_8a91…</span></dd>
                        <dt>Merkle root (epoch 14250-14299)</dt>
                        <dd class="mono small">7c81 4b2e 0a93 9c54 e102 33ad 8f9b 1c20</dd>
                      </dl>
                      <div class="proof-actions">
                        <button class="btn ghost sm" type="button">
                          <svg viewBox="0 0 14 14" width="12" height="12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M3 7a4 4 0 0 1 7-2.7L11 4M11 7a4 4 0 0 1-7 2.7L3 10" /></svg>
                          Re-verify against ledger
                        </button>
                        <a href="#" class="proof-link">View block #{{ row.block }} on public audit chain →</a>
                      </div>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </section>

    <!-- Footer / pager -->
    <footer class="pager">
      <div class="pg-left">
        Showing <span class="mono">{{ visibleCount }}</span> of <span class="mono">14,287</span> events
        <span v-if="fActor !== 'all' || fAction !== 'all' || fSub !== 'all' || fResult !== 'all' || fResource" class="muted">
          · filters active
        </span>
      </div>
      <div class="pg-right">
        <button class="pg-btn" type="button" disabled>← Newer</button>
        <span class="pg-page mono">Page 1 / 715</span>
        <button class="pg-btn" type="button">Older →</button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
/* ============================================================
   Page shell (light admin)
   ============================================================ */
.audit-page {
  min-height: 100vh;
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}

.audit-page .mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.audit-page .muted { color: var(--text-3); }
.audit-page .strong { color: var(--text); font-weight: 600; }
.audit-page .small { font-size: 11px; }

/* ============================================================
   Admin chrome
   ============================================================ */
.admin-chrome {
  display: grid;
  grid-template-columns: auto auto 1fr auto;
  align-items: center;
  gap: 28px;
  padding: 18px 32px;
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
}
.brand {
  display: inline-flex; align-items: center; gap: 10px;
  font-family: var(--font-display); font-weight: 700; font-size: 17px;
  letter-spacing: -0.02em; color: var(--text); text-decoration: none;
}
.brand .mark { display:inline-block; width:11px; height:11px; background: var(--brand); }
.org {
  display: inline-flex; align-items: center; gap: 10px;
  padding: 6px 12px; border: 1px solid var(--border); border-radius: 2px;
  color: var(--text); text-decoration: none; background: var(--canvas);
}
.org-name { font-weight: 600; font-size: 13px; }
.org-pill {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; padding: 2px 6px; border-radius: 2px;
  background: rgba(74,144,226,0.10); color: #2563B0;
}
.chrome-nav { display: inline-flex; gap: 22px; align-items: center; }
.ch-link {
  color: var(--text-2); text-decoration: none; font-size: 13px;
  transition: color 160ms ease; position: relative; padding-bottom: 16px; margin-bottom: -16px;
}
.ch-link:hover { color: var(--text); }
.ch-link.active { color: var(--text); font-weight: 600; }
.ch-link.active::after {
  content: ''; position: absolute; left: 0; right: 0; bottom: -1px;
  height: 2px; background: var(--text);
}
.who { display: inline-flex; align-items: center; gap: 10px; }
.who-email { font-size: 12px; color: var(--text-2); }
.avatar {
  width: 28px; height: 28px; border-radius: 50%;
  background: var(--brand); color: var(--text);
  display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 12px;
}

/* ============================================================
   Breadcrumb
   ============================================================ */
.crumb {
  padding: 14px 32px 0;
  font-size: 12px; color: var(--text-3);
  display: flex; align-items: center; gap: 8px;
}
.crumb a { color: var(--text-2); text-decoration: none; }
.crumb a:hover { color: var(--text); }
.crumb .sep { color: var(--text-3); }
.crumb .cur { color: var(--text); font-weight: 500; }

/* ============================================================
   Page head
   ============================================================ */
.page-head {
  display: grid; grid-template-columns: 1fr auto; gap: 24px;
  padding: 16px 32px 22px;
  align-items: flex-end;
}
.ph-left h1 {
  font-family: var(--font-display); font-weight: 600;
  font-size: 28px; letter-spacing: -0.015em; margin: 0 0 6px;
}
.ph-sub {
  color: var(--text-2); font-size: 13px; margin: 0;
  display: inline-flex; gap: 6px; align-items: center; flex-wrap: wrap;
}
.dot-sep { color: var(--text-3); }

.ph-right { display: flex; flex-direction: column; gap: 12px; align-items: flex-end; }
.integrity {
  display: inline-flex; align-items: center; gap: 10px;
  padding: 8px 12px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
  font-size: 12px;
  cursor: default;
}
.int-label { font-family: var(--font-mono); font-size: 10px; font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3); }
.int-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: var(--pos);
  box-shadow: 0 0 0 3px rgba(25,195,125,0.16);
  animation: int-pulse 2.6s ease-in-out infinite;
}
@keyframes int-pulse {
  0%, 100% { box-shadow: 0 0 0 3px rgba(25,195,125,0.16); }
  50%      { box-shadow: 0 0 0 6px rgba(25,195,125,0.05); }
}
.int-status { color: var(--pos); font-weight: 600; }
.int-meta { color: var(--text-3); font-size: 11px; }

.exports { display: inline-flex; gap: 8px; }
.btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 12px;
  font-family: var(--font-sans); font-size: 12.5px; font-weight: 500;
  border: 1px solid var(--border-strong);
  background: var(--elevated); color: var(--text);
  border-radius: 2px; cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease;
}
.btn:hover { background: rgba(0,0,0,0.03); border-color: var(--text); }
.btn.ghost { background: var(--elevated); }
.btn.primary { background: var(--brand); border-color: var(--brand); color: var(--text); font-weight: 600; }
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.sm { padding: 5px 10px; font-size: 11.5px; }

/* ============================================================
   Stats strip
   ============================================================ */
.stats {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--border); border-block: 1px solid var(--border);
  margin: 0 32px;
}
.stat {
  background: var(--elevated);
  padding: 14px 18px;
}
.stat-label {
  font-family: var(--font-mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
  margin-bottom: 6px;
}
.stat-val { font-size: 22px; font-weight: 600; color: var(--text); letter-spacing: -0.01em; }
.stat-val.neg { color: var(--neg); }

/* ============================================================
   Filter bar
   ============================================================ */
.filters {
  display: flex; gap: 12px; align-items: flex-end;
  padding: 16px 32px;
  border-bottom: 1px solid var(--border);
  background: var(--elevated);
  flex-wrap: wrap;
}
.f { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.f-grow { flex: 1; min-width: 220px; }
.f-label {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-3);
}
.f-input {
  height: 32px;
  padding: 0 10px;
  font-family: var(--font-sans); font-size: 12.5px; color: var(--text);
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: 2px;
  outline: none;
  transition: border-color 120ms ease;
}
.f-input:focus { border-color: var(--accent); }
.f-input.mono { font-family: var(--font-mono); font-size: 12px; }

.chips { display: inline-flex; gap: 4px; }
.chip {
  height: 32px; padding: 0 12px;
  font-family: var(--font-sans); font-size: 12px; font-weight: 500;
  background: var(--canvas); color: var(--text-2);
  border: 1px solid var(--border-strong); border-radius: 2px;
  cursor: pointer;
  transition: background-color 120ms ease, color 120ms ease, border-color 120ms ease;
}
.chip:hover { color: var(--text); border-color: var(--text); }
.chip.on { background: var(--text); color: var(--elevated); border-color: var(--text); }

/* ============================================================
   Table
   ============================================================ */
.tbl-wrap {
  overflow-x: auto;
  background: var(--elevated);
}
.audit-tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}
.audit-tbl thead th {
  text-align: left;
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 600;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3);
  padding: 12px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
  white-space: nowrap;
}
.audit-tbl th.chev { width: 24px; padding-left: 18px; }
.audit-tbl th.th-time { width: 132px; }
.audit-tbl tbody td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
  white-space: nowrap;
}
.audit-tbl tbody td.resource { white-space: normal; max-width: 360px; }
.audit-tbl tbody tr.row {
  cursor: pointer;
  transition: background-color 120ms ease;
}
.audit-tbl tbody tr.row:hover { background: var(--canvas); }
.audit-tbl tbody tr.row.open {
  background: rgba(74,144,226,0.045);
}
.audit-tbl tbody tr.row.fail .ts { color: var(--neg); }
.audit-tbl td.chev {
  width: 24px; padding-left: 18px; color: var(--text-3);
}
.audit-tbl tr.row.open td.chev svg { transform: rotate(90deg); }
.audit-tbl td.chev svg { transition: transform 160ms ease; }

.ts { font-size: 12px; color: var(--text); display: flex; flex-direction: column; gap: 2px; }
.ts-date { font-size: 10px; color: var(--text-3); }

.actor { display: flex; flex-direction: column; gap: 2px; max-width: 220px; }
.actor-email { font-size: 12px; color: var(--text); }
.actor-role { font-size: 10.5px; color: var(--text-3); }

.sub { font-size: 12px; color: var(--text); }
.sub.none { color: var(--text-3); }

.act-chip {
  display: inline-block;
  padding: 3px 7px;
  border-radius: 2px;
  font-size: 10.5px; font-weight: 600;
  letter-spacing: 0.04em;
  white-space: nowrap;
  border: 1px solid transparent;
}
.act-chip.cat-auth     { background: rgba(74,144,226,0.10);  color: #1F5BA0; }
.act-chip.cat-trading  { background: rgba(25,195,125,0.12);  color: #117A4D; }
.act-chip.cat-compute  { background: rgba(107,91,149,0.12);  color: #4C3F73; }
.act-chip.cat-account  { background: rgba(0,0,0,0.05);       color: var(--text); }
.act-chip.cat-security { background: rgba(245,158,11,0.14);  color: #8C5A05; }
.act-chip.cat-system   { background: rgba(0,0,0,0.04);       color: var(--text-2); }

.resource { color: var(--text); font-size: 12.5px; }

.result {
  display: inline-flex; align-items: center; gap: 6px;
  font-family: var(--font-mono); font-size: 10.5px; font-weight: 600;
  letter-spacing: 0.06em; text-transform: uppercase;
}
.result.success { color: var(--pos); }
.result.fail    { color: var(--neg); }
.result .r-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

.ip { font-size: 12px; color: var(--text); }

.hash-cell {
  display: flex; align-items: center; gap: 8px;
  font-size: 11.5px;
}
.block { color: var(--text-3); font-size: 11px; }
.hash  { color: var(--text); }
.copy {
  width: 22px; height: 22px;
  display: inline-flex; align-items: center; justify-content: center;
  background: var(--canvas); color: var(--text-2);
  border: 1px solid var(--border); border-radius: 2px;
  cursor: pointer;
  transition: background-color 120ms ease, color 120ms ease, border-color 120ms ease;
}
.copy:hover { color: var(--text); border-color: var(--text); }

/* Expanded detail row */
.detail-row td { padding: 0 !important; background: var(--canvas); border-bottom: 1px solid var(--border); }
.detail {
  display: grid; grid-template-columns: 1.1fr 1fr; gap: 24px;
  padding: 20px 32px 22px;
  border-top: 1px dashed var(--border);
}
.d-section { background: var(--elevated); border: 1px solid var(--border); border-radius: 2px; padding: 16px 18px; }
.d-section.proof { border-color: rgba(74,144,226,0.30); }
.d-head {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-2); margin-bottom: 12px;
}
.kv {
  display: grid; grid-template-columns: 140px 1fr;
  row-gap: 6px; column-gap: 12px; margin: 0;
}
.kv dt { font-family: var(--font-mono); font-size: 10.5px; color: var(--text-3); letter-spacing: 0.04em; text-transform: uppercase; padding-top: 1px; }
.kv dd { margin: 0; font-size: 12px; color: var(--text); word-break: break-all; }

.proof-actions { display: flex; align-items: center; gap: 14px; margin-top: 14px; }
.proof-link { font-size: 12px; color: var(--accent); text-decoration: none; }
.proof-link:hover { text-decoration: underline; }

/* ============================================================
   Pager
   ============================================================ */
.pager {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 32px;
  font-size: 12.5px;
  border-top: 1px solid var(--border);
  background: var(--elevated);
}
.pg-left { color: var(--text-2); }
.pg-right { display: inline-flex; align-items: center; gap: 12px; }
.pg-btn {
  padding: 6px 12px; font-family: var(--font-sans); font-size: 12px;
  background: var(--canvas); color: var(--text);
  border: 1px solid var(--border-strong); border-radius: 2px; cursor: pointer;
}
.pg-btn:disabled { color: var(--text-3); cursor: default; }
.pg-page { color: var(--text-2); }

@media (max-width: 1180px) {
  .page-head { grid-template-columns: 1fr; }
  .ph-right { align-items: flex-start; }
  .stats { grid-template-columns: repeat(2, 1fr); }
  .detail { grid-template-columns: 1fr; }
}
</style>
