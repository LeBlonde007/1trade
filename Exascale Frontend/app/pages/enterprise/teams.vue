<script setup lang="ts">
/**
 * /enterprise/teams — Team / sub-accounts management.
 *
 * Light, B2B admin surface. 4-stat org summary, sub-account card grid
 * with budget progress bars, an expanded detail block (Members /
 * Budget / Permissions / Activity), and a collapsible recent-activity
 * feed across all teams.
 */

definePageMeta({ layout: false })
useHead({
  title: 'Team management · Walmart Inc. — Exascale Enterprise',
  htmlAttrs: { 'data-theme': 'light' },
})

// =====================================================
// Org context
// =====================================================
const ORG = {
  name: 'Walmart Inc.',
  enterpriseId: 'ENT-WMT-001',
  csmName: 'Sarah Lin',
  totalBudget: 5_000_000,
  totalUsed: 1_847_210,
  totalUsers: 73,
  active7d: 51,
}

// =====================================================
// Sub-accounts
// =====================================================
type SubStatus = 'active' | 'paused' | 'archived'

interface Subaccount {
  id: string
  name: string
  sub: string                // 1-line descriptor
  initial: string            // letter avatar
  accent: string             // top stripe color
  users: number
  budget: number             // annual USD
  used: number               // YTD USD
  todaySpend: number
  weekSpend: number
  status: SubStatus
}

const subaccounts = reactive<Subaccount[]>([
  {
    id: 'ai-research',
    name: 'AI Research',
    sub: 'Foundation-model R&D · Bentonville HQ',
    initial: 'A',
    accent: 'var(--brand)',
    users: 50,
    budget: 2_000_000,
    used: 847_210,
    todaySpend: 3_240,
    weekSpend: 22_180,
    status: 'active',
  },
  {
    id: 'cs-ai',
    name: 'Customer Service AI',
    sub: 'Voice + chat agents · multi-region',
    initial: 'C',
    accent: 'var(--accent)',
    users: 12,
    budget: 500_000,
    used: 485_000,
    todaySpend: 1_840,
    weekSpend: 12_910,
    status: 'active',
  },
  {
    id: 'supply-ml',
    name: 'Supply Chain ML',
    sub: 'Forecasting + routing · ops backbone',
    initial: 'S',
    accent: '#F5A623',
    users: 8,
    budget: 300_000,
    used: 245_000,
    todaySpend: 720,
    weekSpend: 5_900,
    status: 'active',
  },
  {
    id: 'frontier-exp',
    name: 'Frontier Experiments',
    sub: 'Long-tail R&D · sandboxed',
    initial: 'F',
    accent: '#9A9A95',
    users: 3,
    budget: 2_200_000,
    used: 270_000,
    todaySpend: 1_100,
    weekSpend: 8_400,
    status: 'active',
  },
])

// =====================================================
// View mode + expanded sub-account
// =====================================================
type View = 'cards' | 'table'
const view = ref<View>('cards')

const expandedId = ref<string>('ai-research')
const expanded = computed(() => subaccounts.find(s => s.id === expandedId.value)!)

type DetailTab = 'members' | 'budget' | 'permissions' | 'activity'
const DETAIL_TABS: DetailTab[] = ['members', 'budget', 'permissions', 'activity']
const detailTab = ref<DetailTab>('members')

function selectSubaccount(id: string) {
  expandedId.value = id
  detailTab.value = 'members'
  if (typeof window !== 'undefined') {
    nextTick(() => {
      const el = document.getElementById('sub-detail')
      el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }
}

// =====================================================
// Members (for the expanded sub-account)
// =====================================================
interface Member {
  name: string
  email: string
  initials: string
  role: 'Owner' | 'Admin' | 'Trader' | 'Analyst' | 'Viewer'
  lastActive: string
  status: 'online' | 'idle' | 'offline'
}

const MEMBERS_BY_SUB: Record<string, Member[]> = {
  'ai-research': [
    { name: 'Sarah Lin',          email: 'sarah.lin@walmart.com',     initials: 'SL', role: 'Owner',   lastActive: 'Active now',     status: 'online' },
    { name: 'Marcus Chen',        email: 'marcus.chen@walmart.com',   initials: 'MC', role: 'Admin',   lastActive: '12 minutes ago', status: 'online' },
    { name: 'Jane Doe',           email: 'jane.doe@walmart.com',      initials: 'JD', role: 'Trader',  lastActive: '1 hour ago',     status: 'idle'   },
    { name: 'Yui Tanaka',         email: 'yui.tanaka@walmart.com',    initials: 'YT', role: 'Trader',  lastActive: '3 hours ago',    status: 'idle'   },
    { name: 'Léon Beaumont',      email: 'leon.b@walmart.com',        initials: 'LB', role: 'Analyst', lastActive: '1 day ago',      status: 'offline'},
    { name: 'Priya Rao',          email: 'priya.rao@walmart.com',     initials: 'PR', role: 'Analyst', lastActive: '2 days ago',     status: 'offline'},
    { name: "Aiden O'Connell",    email: 'aiden.oc@walmart.com',      initials: 'AO', role: 'Viewer',  lastActive: '5 days ago',     status: 'offline'},
    { name: 'Daniel Martínez',    email: 'd.martinez@walmart.com',    initials: 'DM', role: 'Trader',  lastActive: '24 minutes ago', status: 'online' },
  ],
  'cs-ai': [
    { name: 'Anna Korhonen',      email: 'a.korhonen@walmart.com',    initials: 'AK', role: 'Owner',   lastActive: '40 minutes ago', status: 'idle'   },
    { name: 'Ravi Subramanian',   email: 'ravi.s@walmart.com',        initials: 'RS', role: 'Admin',   lastActive: '2 hours ago',    status: 'idle'   },
    { name: 'Greta Müller',       email: 'g.mueller@walmart.com',     initials: 'GM', role: 'Trader',  lastActive: '6 hours ago',    status: 'offline'},
  ],
  'supply-ml': [
    { name: 'Eleanor Park',       email: 'e.park@walmart.com',        initials: 'EP', role: 'Owner',   lastActive: 'Active now',     status: 'online' },
    { name: 'Tom Wilson',         email: 't.wilson@walmart.com',      initials: 'TW', role: 'Trader',  lastActive: '45 minutes ago', status: 'idle'   },
  ],
  'frontier-exp': [
    { name: 'Dr. Rina Halpern',   email: 'r.halpern@walmart.com',     initials: 'RH', role: 'Owner',   lastActive: '8 hours ago',    status: 'offline'},
    { name: 'Hiroshi Ito',        email: 'h.ito@walmart.com',         initials: 'HI', role: 'Analyst', lastActive: '2 days ago',     status: 'offline'},
  ],
}
const expandedMembers = computed(() => MEMBERS_BY_SUB[expandedId.value] ?? [])

// =====================================================
// Permissions (for expanded sub-account)
// =====================================================
interface PermissionRow {
  capability: string
  detail: string
  owner: 'allow' | 'deny'
  admin: 'allow' | 'deny'
  trader: 'allow' | 'deny'
  analyst: 'allow' | 'deny'
  viewer: 'allow' | 'deny'
}
const PERMISSIONS: PermissionRow[] = [
  { capability: 'Place orders',         detail: 'Buy/sell against live markets',         owner: 'allow', admin: 'allow', trader: 'allow', analyst: 'deny',  viewer: 'deny'  },
  { capability: 'Manage positions',     detail: 'Close, hedge, partial fills',           owner: 'allow', admin: 'allow', trader: 'allow', analyst: 'deny',  viewer: 'deny'  },
  { capability: 'Purchase credits',     detail: 'Wire-funded bulk purchase',             owner: 'allow', admin: 'allow', trader: 'deny',  analyst: 'deny',  viewer: 'deny'  },
  { capability: 'View positions',       detail: 'Read-only access to portfolio',         owner: 'allow', admin: 'allow', trader: 'allow', analyst: 'allow', viewer: 'allow' },
  { capability: 'Edit budget',          detail: 'Set sub-account budget envelope',       owner: 'allow', admin: 'allow', trader: 'deny',  analyst: 'deny',  viewer: 'deny'  },
  { capability: 'Invite members',       detail: 'Send invites + assign roles',           owner: 'allow', admin: 'allow', trader: 'deny',  analyst: 'deny',  viewer: 'deny'  },
  { capability: 'Export audit log',     detail: 'CSV / JSON for SIEM pipelines',         owner: 'allow', admin: 'allow', trader: 'deny',  analyst: 'allow', viewer: 'deny'  },
  { capability: 'Manage API keys',      detail: 'Create, scope and revoke tokens',       owner: 'allow', admin: 'allow', trader: 'deny',  analyst: 'deny',  viewer: 'deny'  },
]

// =====================================================
// Recent activity (collapsible)
// =====================================================
interface Activity {
  ts: string
  who: string
  action: string
  sub: string
  badge: 'order' | 'budget' | 'member' | 'key' | 'auto'
}
const ACTIVITY: Activity[] = [
  { ts: '12 min ago',   who: 'marcus.chen@walmart.com',  action: 'placed bulk credit purchase · $25,000 USD',                  sub: 'AI Research',          badge: 'order' },
  { ts: '38 min ago',   who: 'sarah.lin@walmart.com',    action: 'updated annual budget allocation to $500,000',               sub: 'Customer Service AI',  badge: 'budget' },
  { ts: '1 h ago',      who: 'jane.doe@walmart.com',     action: 'invited yui.tanaka@walmart.com as Trader',                   sub: 'AI Research',          badge: 'member' },
  { ts: '2 h ago',      who: 'sarah.lin@walmart.com',    action: 'rotated API key esx_admin_b91c08…',                          sub: 'Frontier Experiments', badge: 'key' },
  { ts: '4 h ago',      who: 'automation@exascale',      action: 'auto-paused trading · daily budget 95% reached',             sub: 'Customer Service AI',  badge: 'auto' },
  { ts: '8 h ago',      who: 'e.park@walmart.com',       action: 'added member d.martinez@walmart.com as Trader',              sub: 'Supply Chain ML',      badge: 'member' },
  { ts: '1 day ago',    who: 'priya.rao@walmart.com',    action: 'exported audit log · 24h window · 3,412 rows',               sub: 'AI Research',          badge: 'key' },
  { ts: '2 days ago',   who: 'r.halpern@walmart.com',    action: 'placed bulk credit purchase · $100,000 USD · wire pending',  sub: 'Frontier Experiments', badge: 'order' },
]
const activityOpen = ref(true)

// =====================================================
// Formatters + budget helpers
// =====================================================
function fmtUsd(n: number, dp = 0) {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function pctOf(used: number, budget: number) {
  return budget === 0 ? 0 : (used / budget) * 100
}
function pctTone(pct: number): 'pos' | 'warn' | 'neg' {
  if (pct >= 95) return 'neg'
  if (pct >= 80) return 'warn'
  return 'pos'
}
function pctLabel(pct: number) {
  if (pct >= 95) return 'At cap'
  if (pct >= 80) return 'Approaching cap'
  return 'Healthy'
}

const orgPct = computed(() => pctOf(ORG.totalUsed, ORG.totalBudget))
</script>

<template>
  <div class="teams-shell" data-theme="light">
    <!-- ============ Top bar ============ -->
    <header class="topbar">
      <div class="brand-row">
        <NuxtLink to="/" class="brand">
          <span class="brand-mark" />
          EXASCALE
        </NuxtLink>
        <span class="ent-pill">ENTERPRISE</span>
        <span class="brand-sep">·</span>
        <NuxtLink to="/enterprise/onboarding" class="cust" :title="ORG.enterpriseId">
          <span class="cust-mark" aria-hidden="true">W</span>
          <span class="cust-name">{{ ORG.name }}</span>
          <span class="cust-id">{{ ORG.enterpriseId }}</span>
        </NuxtLink>
      </div>
      <nav class="crumbs">
        <NuxtLink to="/enterprise/onboarding">Enterprise</NuxtLink>
        <span class="sep">›</span>
        <span class="cur">Team management</span>
      </nav>
      <div class="top-right">
        <span class="csm-tag">CSM · {{ ORG.csmName }}</span>
        <button type="button" class="btn ghost">Audit log</button>
      </div>
    </header>

    <main class="page">
      <!-- ============ Page header ============ -->
      <section class="page-head">
        <div>
          <div class="eyebrow"><span class="dot" /> Enterprise · administration</div>
          <h1 class="page-title">Team management</h1>
          <p class="page-sub">
            <strong>{{ ORG.name }}</strong>
            · {{ subaccounts.length }} sub-accounts
            · {{ ORG.totalUsers }} total users
            · {{ ORG.active7d }} active in the last 7 days
          </p>
        </div>
        <div class="head-actions">
          <button type="button" class="btn secondary">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M8 3v10M3 8l5 5 5-5" />
            </svg>
            Export CSV
          </button>
          <button type="button" class="btn primary">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
              <path d="M8 3v10M3 8h10" />
            </svg>
            Add sub-account
          </button>
        </div>
      </section>

      <!-- ============ Org-level summary stats ============ -->
      <section class="stats-grid">
        <div class="stat featured">
          <div class="stat-lbl">— Total budget allocated · {{ new Date().getFullYear() }}</div>
          <div class="stat-val">{{ fmtUsd(ORG.totalBudget) }}</div>
          <div class="stat-sub">Across {{ subaccounts.length }} sub-accounts · annual rolling</div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Total used YTD</div>
          <div class="stat-val">{{ fmtUsd(ORG.totalUsed) }}</div>
          <div class="stat-sub">
            <span class="pos-text">{{ orgPct.toFixed(1) }}%</span>
            of envelope · {{ fmtUsd(ORG.totalBudget - ORG.totalUsed) }} remaining
          </div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Total users</div>
          <div class="stat-val">{{ ORG.totalUsers }}</div>
          <div class="stat-sub">{{ ORG.active7d }} active in last 7d</div>
        </div>
        <div class="stat">
          <div class="stat-lbl">— Active in last 7d</div>
          <div class="stat-val">{{ ORG.active7d }}</div>
          <div class="stat-sub">
            <span class="pos-text">+12.3%</span>
            vs prior week
          </div>
        </div>
      </section>

      <!-- ============ Sub-accounts header + view toggle ============ -->
      <section class="sub-head">
        <h2 class="sub-title">Sub-accounts</h2>
        <div class="sub-head-right">
          <div class="view-seg">
            <button type="button" :class="{ active: view === 'cards' }" @click="view = 'cards'">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <rect x="2.5" y="2.5" width="5" height="5" />
                <rect x="8.5" y="2.5" width="5" height="5" />
                <rect x="2.5" y="8.5" width="5" height="5" />
                <rect x="8.5" y="8.5" width="5" height="5" />
              </svg>
              Cards
            </button>
            <button type="button" :class="{ active: view === 'table' }" @click="view = 'table'">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <path d="M2.5 4h11M2.5 8h11M2.5 12h11" />
              </svg>
              Table
            </button>
          </div>
          <button type="button" class="btn secondary sm">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
              <path d="M3 4h10M5 8h6M7 12h2" />
            </svg>
            Filter
          </button>
        </div>
      </section>

      <!-- ============ Sub-account cards ============ -->
      <section v-if="view === 'cards'" class="sub-grid">
        <article
          v-for="s in subaccounts"
          :key="s.id"
          class="sub-card"
          :class="{ selected: s.id === expandedId }"
          @click="selectSubaccount(s.id)"
        >
          <span class="sub-accent" :style="{ background: s.accent }" />
          <header class="sub-card-head">
            <div class="sub-card-id">
              <span class="sub-avatar" :style="{ background: s.accent }">{{ s.initial }}</span>
              <div>
                <h3 class="sub-card-name">{{ s.name }}</h3>
                <span class="sub-card-sub">{{ s.sub }}</span>
              </div>
            </div>
            <div class="sub-card-meta">
              <span class="status-tag" :class="s.status">
                <span class="dot" /> {{ s.status.charAt(0).toUpperCase() + s.status.slice(1) }}
              </span>
              <button type="button" class="kebab" aria-label="Actions" @click.stop>
                <svg viewBox="0 0 16 16" fill="currentColor"><circle cx="3" cy="8" r="1.3" /><circle cx="8" cy="8" r="1.3" /><circle cx="13" cy="8" r="1.3" /></svg>
              </button>
            </div>
          </header>

          <div class="sub-card-row">
            <div class="kv">
              <span class="kv-k">Users</span>
              <span class="kv-v">{{ s.users }}</span>
            </div>
            <div class="kv">
              <span class="kv-k">Budget · annual</span>
              <span class="kv-v">{{ fmtUsd(s.budget) }}</span>
            </div>
          </div>

          <div class="budget">
            <div class="budget-row">
              <span class="budget-k">Used YTD</span>
              <span class="budget-v">
                {{ fmtUsd(s.used) }}
                <span class="budget-pct" :class="pctTone(pctOf(s.used, s.budget))">
                  ({{ pctOf(s.used, s.budget).toFixed(0) }}%)
                </span>
              </span>
            </div>
            <div class="budget-bar">
              <div
                class="budget-fill"
                :class="pctTone(pctOf(s.used, s.budget))"
                :style="{ width: Math.min(100, pctOf(s.used, s.budget)) + '%' }"
              />
            </div>
            <div class="budget-foot">
              <span class="budget-state" :class="pctTone(pctOf(s.used, s.budget))">
                {{ pctLabel(pctOf(s.used, s.budget)) }}
              </span>
              <span class="budget-remaining">
                {{ fmtUsd(s.budget - s.used) }} remaining
              </span>
            </div>
          </div>

          <div class="sub-card-row mini">
            <div class="kv">
              <span class="kv-k">Today</span>
              <span class="kv-v mono">{{ fmtUsd(s.todaySpend) }}</span>
            </div>
            <div class="kv">
              <span class="kv-k">7d spend</span>
              <span class="kv-v mono">{{ fmtUsd(s.weekSpend) }}</span>
            </div>
          </div>

          <footer class="sub-card-foot">
            <button type="button" class="btn-mini primary" @click.stop="selectSubaccount(s.id)">
              View
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M3 8h10M9 4l4 4-4 4" />
              </svg>
            </button>
            <button type="button" class="btn-mini" @click.stop>Edit</button>
            <button type="button" class="btn-mini danger" @click.stop>Delete</button>
          </footer>
        </article>
      </section>

      <!-- ============ Sub-account table (alt view) ============ -->
      <section v-else class="sub-table-wrap">
        <table class="sub-table">
          <thead>
            <tr>
              <th class="left">Sub-account</th>
              <th>Users</th>
              <th>Budget</th>
              <th>Used YTD</th>
              <th class="left">Burn</th>
              <th>Today</th>
              <th>7d</th>
              <th class="left">Status</th>
              <th class="right-th">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="s in subaccounts"
              :key="s.id"
              :class="{ selected: s.id === expandedId }"
              @click="selectSubaccount(s.id)"
            >
              <td class="left">
                <div class="row-id">
                  <span class="sub-avatar small" :style="{ background: s.accent }">{{ s.initial }}</span>
                  <div>
                    <div class="row-name">{{ s.name }}</div>
                    <div class="row-sub">{{ s.sub }}</div>
                  </div>
                </div>
              </td>
              <td>{{ s.users }}</td>
              <td class="mono">{{ fmtUsd(s.budget) }}</td>
              <td class="mono">{{ fmtUsd(s.used) }}</td>
              <td class="left">
                <div class="row-burn">
                  <div class="row-bar">
                    <div class="budget-fill" :class="pctTone(pctOf(s.used, s.budget))" :style="{ width: Math.min(100, pctOf(s.used, s.budget)) + '%' }" />
                  </div>
                  <span class="row-pct mono" :class="pctTone(pctOf(s.used, s.budget))">
                    {{ pctOf(s.used, s.budget).toFixed(0) }}%
                  </span>
                </div>
              </td>
              <td class="mono">{{ fmtUsd(s.todaySpend) }}</td>
              <td class="mono">{{ fmtUsd(s.weekSpend) }}</td>
              <td class="left">
                <span class="status-tag" :class="s.status">
                  <span class="dot" /> {{ s.status.charAt(0).toUpperCase() + s.status.slice(1) }}
                </span>
              </td>
              <td class="right-td">
                <div class="row-actions">
                  <button type="button" class="btn-mini primary" @click.stop="selectSubaccount(s.id)">View</button>
                  <button type="button" class="btn-mini" @click.stop>Edit</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- ============ Expanded sub-account detail ============ -->
      <section id="sub-detail" class="detail">
        <div class="detail-head">
          <div class="detail-head-left">
            <span class="sub-avatar lg" :style="{ background: expanded.accent }">{{ expanded.initial }}</span>
            <div>
              <h2 class="detail-title">{{ expanded.name }}</h2>
              <p class="detail-sub">{{ expanded.sub }}</p>
            </div>
          </div>
          <div class="detail-meta">
            <div class="meta-kv">
              <span class="meta-k">Users</span>
              <span class="meta-v">{{ expanded.users }}</span>
            </div>
            <div class="meta-kv">
              <span class="meta-k">Budget</span>
              <span class="meta-v mono">{{ fmtUsd(expanded.budget) }}</span>
            </div>
            <div class="meta-kv">
              <span class="meta-k">Used YTD</span>
              <span class="meta-v mono">
                {{ fmtUsd(expanded.used) }}
                <span class="budget-pct" :class="pctTone(pctOf(expanded.used, expanded.budget))">
                  ({{ pctOf(expanded.used, expanded.budget).toFixed(0) }}%)
                </span>
              </span>
            </div>
          </div>
        </div>

        <!-- Tabs -->
        <div class="tabs">
          <button
            v-for="t in DETAIL_TABS"
            :key="t"
            type="button"
            class="tab"
            :class="{ active: detailTab === t }"
            @click="detailTab = t"
          >
            {{ t.charAt(0).toUpperCase() + t.slice(1) }}
            <span v-if="t === 'members'" class="tab-count">{{ expandedMembers.length }}</span>
          </button>
          <div class="tab-spacer" />
          <button type="button" class="btn secondary sm">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
              <path d="M8 3v10M3 8h10" />
            </svg>
            Invite member
          </button>
        </div>

        <!-- ===== MEMBERS ===== -->
        <div v-if="detailTab === 'members'" class="detail-body">
          <table class="members-table">
            <thead>
              <tr>
                <th class="left">Name</th>
                <th class="left">Email</th>
                <th class="left">Role</th>
                <th class="left">Last active</th>
                <th class="right-th"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in expandedMembers" :key="m.email">
                <td class="left">
                  <div class="m-id">
                    <span class="m-avatar" :class="m.status">{{ m.initials }}</span>
                    <span class="m-name">{{ m.name }}</span>
                  </div>
                </td>
                <td class="left mono">{{ m.email }}</td>
                <td class="left">
                  <span class="role-tag" :class="'r-' + m.role.toLowerCase()">{{ m.role }}</span>
                </td>
                <td class="left dim">{{ m.lastActive }}</td>
                <td class="right-td">
                  <button type="button" class="btn-mini">Manage</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div class="see-all">
            <span class="dim">Showing {{ expandedMembers.length }} of {{ expanded.users }} members</span>
            <a href="#">View all {{ expanded.users }} members →</a>
          </div>
        </div>

        <!-- ===== BUDGET ===== -->
        <div v-else-if="detailTab === 'budget'" class="detail-body">
          <div class="bd-grid">
            <div class="bd-card">
              <div class="bd-k">Annual envelope</div>
              <div class="bd-v">{{ fmtUsd(expanded.budget) }}</div>
              <div class="bd-sub">Set 2026-01-01 · auto-renews</div>
            </div>
            <div class="bd-card">
              <div class="bd-k">Used YTD</div>
              <div class="bd-v">{{ fmtUsd(expanded.used) }}</div>
              <div class="bd-sub" :class="pctTone(pctOf(expanded.used, expanded.budget))">
                {{ pctOf(expanded.used, expanded.budget).toFixed(0) }}% of envelope
              </div>
            </div>
            <div class="bd-card">
              <div class="bd-k">Remaining</div>
              <div class="bd-v">{{ fmtUsd(expanded.budget - expanded.used) }}</div>
              <div class="bd-sub">At current burn, lasts ~9 months</div>
            </div>
            <div class="bd-card">
              <div class="bd-k">Avg daily spend · 30d</div>
              <div class="bd-v">{{ fmtUsd(Math.round(expanded.used / 140)) }}</div>
              <div class="bd-sub pos-text">▼ 4.2% vs prior 30d</div>
            </div>
          </div>

          <div class="bd-bar">
            <div class="bd-bar-track">
              <div
                class="budget-fill big"
                :class="pctTone(pctOf(expanded.used, expanded.budget))"
                :style="{ width: Math.min(100, pctOf(expanded.used, expanded.budget)) + '%' }"
              />
            </div>
            <div class="bd-bar-ticks">
              <span>0</span>
              <span>25%</span>
              <span>50%</span>
              <span class="warn-tick">75%</span>
              <span class="cap-tick">100%</span>
            </div>
          </div>

          <div class="bd-rules">
            <div class="bd-rule">
              <span class="rule-k">Hard cap</span>
              <span class="rule-v">100% of envelope · trading auto-paused at cap</span>
            </div>
            <div class="bd-rule">
              <span class="rule-k">Daily limit</span>
              <span class="rule-v">$25,000 USD · per sub-account</span>
            </div>
            <div class="bd-rule">
              <span class="rule-k">Alert thresholds</span>
              <span class="rule-v">Email FinOps at 75% · Slack #finops at 90%</span>
            </div>
          </div>
        </div>

        <!-- ===== PERMISSIONS ===== -->
        <div v-else-if="detailTab === 'permissions'" class="detail-body">
          <div class="perms-intro">
            Role-based access control for <strong>{{ expanded.name }}</strong>. Changes propagate immediately to all members of the matching role.
          </div>
          <table class="perms-table">
            <thead>
              <tr>
                <th class="left">Capability</th>
                <th class="center">Owner</th>
                <th class="center">Admin</th>
                <th class="center">Trader</th>
                <th class="center">Analyst</th>
                <th class="center">Viewer</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in PERMISSIONS" :key="p.capability">
                <td class="left">
                  <div class="cap-name">{{ p.capability }}</div>
                  <div class="cap-detail">{{ p.detail }}</div>
                </td>
                <td class="center"><span class="perm" :class="p.owner">{{ p.owner === 'allow' ? '✓' : '×' }}</span></td>
                <td class="center"><span class="perm" :class="p.admin">{{ p.admin === 'allow' ? '✓' : '×' }}</span></td>
                <td class="center"><span class="perm" :class="p.trader">{{ p.trader === 'allow' ? '✓' : '×' }}</span></td>
                <td class="center"><span class="perm" :class="p.analyst">{{ p.analyst === 'allow' ? '✓' : '×' }}</span></td>
                <td class="center"><span class="perm" :class="p.viewer">{{ p.viewer === 'allow' ? '✓' : '×' }}</span></td>
              </tr>
            </tbody>
          </table>
          <div class="perms-foot">
            <span class="dim">All changes are written to the audit log with the actor's email + IP.</span>
            <button type="button" class="btn secondary sm">Edit role matrix</button>
          </div>
        </div>

        <!-- ===== ACTIVITY ===== -->
        <div v-else class="detail-body">
          <div class="activity-list">
            <div v-for="(a, i) in ACTIVITY.filter(x => x.sub === expanded.name).concat(ACTIVITY.filter(x => x.sub !== expanded.name)).slice(0, 6)" :key="i" class="act-row">
              <span class="act-ts mono">{{ a.ts }}</span>
              <span class="act-badge" :class="'b-' + a.badge">{{ a.badge }}</span>
              <span class="act-text">
                <span class="act-who">{{ a.who }}</span>
                <span class="act-action">{{ a.action }}</span>
                <span class="act-sub">· {{ a.sub }}</span>
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ Recent activity (collapsible, cross-team) ============ -->
      <section class="activity-card" :class="{ collapsed: !activityOpen }">
        <header class="activity-head" @click="activityOpen = !activityOpen">
          <span class="eyebrow"><span class="dot" /> Recent user activity · all sub-accounts</span>
          <div class="activity-head-right">
            <a href="#" @click.stop>View full audit log →</a>
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="caret">
              <path d="M4 6l4 4 4-4" />
            </svg>
          </div>
        </header>
        <div class="activity-body">
          <div class="activity-list">
            <div v-for="(a, i) in ACTIVITY" :key="i" class="act-row">
              <span class="act-ts mono">{{ a.ts }}</span>
              <span class="act-badge" :class="'b-' + a.badge">{{ a.badge }}</span>
              <span class="act-text">
                <span class="act-who">{{ a.who }}</span>
                <span class="act-action">{{ a.action }}</span>
                <span class="act-sub">· {{ a.sub }}</span>
              </span>
            </div>
          </div>
        </div>
      </section>

      <footer class="page-foot">
        <span>ORG <strong>{{ ORG.enterpriseId }}</strong> · {{ ORG.name }} · {{ subaccounts.length }} sub-accounts · last updated 2 minutes ago</span>
        <NuxtLink to="/enterprise/onboarding">← Back to onboarding</NuxtLink>
      </footer>
    </main>
  </div>
</template>

<style scoped>
.teams-shell {
  --bd-soft: rgba(14, 14, 14, 0.06);
  background: var(--canvas);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  line-height: 1.5;
  font-feature-settings: 'ss01';
  -webkit-font-smoothing: antialiased;
  min-height: 100vh;
}

.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.dim { color: var(--text-3); }
.pos-text { color: var(--pos); font-weight: 500; }

/* ============================================================
   Top bar
   ============================================================ */
.topbar {
  height: 56px;
  background: var(--elevated);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 28px;
  gap: 20px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.brand-row {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 15px;
  letter-spacing: -0.005em;
  color: var(--text);
  text-decoration: none;
}
.brand-mark {
  width: 12px;
  height: 12px;
  background: var(--brand);
  display: inline-block;
}
.ent-pill {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.20em;
  color: var(--text);
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}
.brand-sep { color: var(--text-3); }
.cust {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: var(--text);
}
.cust:hover .cust-name { color: var(--text); }
.cust-mark {
  width: 22px;
  height: 22px;
  background: var(--text);
  color: var(--brand);
  display: grid;
  place-items: center;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 13px;
  border-radius: var(--radius-sm);
}
.cust-name {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 14px;
  letter-spacing: -0.01em;
  color: var(--text);
}
.cust-id {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  padding-left: 10px;
  border-left: 1px solid var(--border);
}

.crumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
  margin-left: 8px;
}
.crumbs a { color: var(--text-2); text-decoration: none; }
.crumbs a:hover { color: var(--text); }
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .cur { color: var(--text); }

.top-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10px;
}
.csm-tag {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
  padding-right: 12px;
  border-right: 1px solid var(--border);
}

/* ============================================================
   Page shell
   ============================================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 32px 28px 96px;
}

.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}
.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.eyebrow .dot {
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.page-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 34px;
  letter-spacing: -0.025em;
  line-height: 1.05;
  margin: 0 0 8px;
  color: var(--text);
}
.page-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0;
  font-family: var(--font-sans);
}
.page-sub strong { color: var(--text); font-weight: 500; }

.head-actions { display: flex; gap: 8px; }

/* ============================================================
   Buttons
   ============================================================ */
.btn {
  height: 38px;
  padding: 0 16px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: -0.005em;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: background 120ms, border-color 120ms, color 120ms;
  text-decoration: none;
}
.btn svg { width: 14px; height: 14px; }
.btn.sm { height: 30px; padding: 0 12px; font-size: 12px; }
.btn.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
  font-weight: 600;
}
.btn.primary:hover { background: #222; border-color: #222; }
.btn.secondary {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover {
  background: var(--canvas);
  border-color: rgba(14, 14, 14, 0.32);
}
.btn.ghost {
  background: transparent;
  color: var(--text-2);
  height: 30px;
  padding: 0 12px;
  font-size: 12px;
}
.btn.ghost:hover { color: var(--text); background: var(--canvas); }

.btn-mini {
  height: 28px;
  padding: 0 10px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.005em;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.btn-mini:hover {
  background: var(--canvas);
  border-color: rgba(14, 14, 14, 0.32);
}
.btn-mini.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
}
.btn-mini.primary:hover { background: #222; border-color: #222; }
.btn-mini.danger:hover {
  background: rgba(220, 38, 38, 0.06);
  color: var(--neg);
  border-color: rgba(220, 38, 38, 0.30);
}
.btn-mini svg { width: 12px; height: 12px; }

/* ============================================================
   Stats grid
   ============================================================ */
.stats-grid {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 1fr;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 32px;
  overflow: hidden;
}
.stat {
  padding: 20px 22px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-right: 1px solid var(--border);
  position: relative;
}
.stat:last-child { border-right: 0; }
.stat.featured::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--text);
}
.stat-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}
.stat-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--text);
}
.stat-sub {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}

/* ============================================================
   Sub-accounts header + view toggle
   ============================================================ */
.sub-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.sub-title {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--text);
}
.sub-head-right { display: flex; align-items: center; gap: 8px; }

.view-seg {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.view-seg button {
  background: transparent;
  color: var(--text-2);
  border: 0;
  border-right: 1px solid var(--border);
  font-family: var(--font-sans);
  font-size: 12px;
  height: 30px;
  padding: 0 12px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  cursor: pointer;
}
.view-seg button:last-child { border-right: 0; }
.view-seg button:hover { background: var(--canvas); color: var(--text); }
.view-seg button.active {
  background: var(--text);
  color: var(--elevated);
}
.view-seg button svg { width: 12px; height: 12px; }

/* ============================================================
   Sub-account cards
   ============================================================ */
.sub-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 36px;
}
.sub-card {
  position: relative;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 20px 22px 16px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: border-color 120ms, box-shadow 120ms, transform 80ms;
}
.sub-card:hover {
  border-color: var(--border-strong);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
}
.sub-card.selected {
  border-color: var(--text);
  box-shadow: 0 0 0 3px rgba(14, 14, 14, 0.04);
}
.sub-accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
}

.sub-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.sub-card-id {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.sub-avatar {
  width: 32px;
  height: 32px;
  background: var(--text);
  color: var(--canvas);
  display: grid;
  place-items: center;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 15px;
  letter-spacing: -0.02em;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.sub-avatar.small { width: 26px; height: 26px; font-size: 12px; }
.sub-avatar.lg { width: 44px; height: 44px; font-size: 18px; }
.sub-card-name {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
}
.sub-card-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.sub-card-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.status-tag .dot { width: 4px; height: 4px; border-radius: 50%; background: currentColor; }
.status-tag.active {
  background: rgba(22, 163, 74, 0.10);
  color: var(--pos);
  border: 1px solid rgba(22, 163, 74, 0.25);
}
.status-tag.paused {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border: 1px solid rgba(245, 158, 11, 0.25);
}
.status-tag.archived {
  background: transparent;
  color: var(--text-3);
  border: 1px solid var(--border);
}
.kebab {
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-3);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.kebab:hover { background: var(--canvas); color: var(--text); }
.kebab svg { width: 14px; height: 14px; }

.sub-card-row {
  display: flex;
  gap: 24px;
}
.sub-card-row.mini {
  padding: 10px 0 0;
  border-top: 1px dashed var(--bd-soft);
}
.kv {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.kv-k {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.kv-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.01em;
}
.kv-v.mono { font-family: var(--font-mono); }

/* Budget row */
.budget {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.budget-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.budget-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.budget-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.budget-pct {
  font-size: 12px;
  font-weight: 500;
  margin-left: 4px;
}
.budget-pct.pos  { color: var(--pos); }
.budget-pct.warn { color: var(--warn); }
.budget-pct.neg  { color: var(--neg); }

.budget-bar {
  height: 6px;
  background: var(--sunken);
  border-radius: 999px;
  overflow: hidden;
}
.budget-fill {
  height: 100%;
  border-radius: 999px;
  transition: width 400ms ease-out;
}
.budget-fill.pos   { background: linear-gradient(90deg, var(--pos), rgba(22, 163, 74, 0.65)); }
.budget-fill.warn  { background: linear-gradient(90deg, var(--warn), rgba(245, 158, 11, 0.65)); }
.budget-fill.neg   { background: linear-gradient(90deg, var(--neg), rgba(220, 38, 38, 0.65)); }
.budget-fill.big   { height: 100%; }

.budget-foot {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.budget-state.pos  { color: var(--pos); }
.budget-state.warn { color: var(--warn); }
.budget-state.neg  { color: var(--neg); }
.budget-remaining  { color: var(--text-3); }

.sub-card-foot {
  display: flex;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}

/* ============================================================
   Sub-account table (alt view)
   ============================================================ */
.sub-table-wrap {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  margin-bottom: 36px;
}
.sub-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.sub-table thead th {
  text-align: right;
  padding: 10px 14px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  background: var(--canvas);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.sub-table th.left, .sub-table td.left { text-align: left; }
.sub-table th.right-th, .sub-table td.right-td { text-align: right; }
.sub-table tbody td {
  padding: 14px;
  border-bottom: 1px solid var(--bd-soft);
  text-align: right;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text);
  cursor: pointer;
}
.sub-table tbody td.left { font-family: var(--font-sans); }
.sub-table tbody tr:hover td { background: var(--canvas); }
.sub-table tbody tr.selected td { background: var(--sunken); }
.sub-table tbody tr:last-child td { border-bottom: 0; }
.row-id {
  display: flex;
  align-items: center;
  gap: 12px;
}
.row-name {
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 13.5px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.row-sub {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 2px;
}
.row-burn {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 140px;
}
.row-bar {
  flex: 1;
  height: 4px;
  background: var(--sunken);
  border-radius: 999px;
  overflow: hidden;
}
.row-pct {
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 600;
  min-width: 38px;
}
.row-pct.pos  { color: var(--pos); }
.row-pct.warn { color: var(--warn); }
.row-pct.neg  { color: var(--neg); }
.row-actions { display: inline-flex; gap: 6px; justify-content: flex-end; }

/* ============================================================
   Expanded detail
   ============================================================ */
.detail {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  margin-bottom: 24px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.03);
  scroll-margin-top: 80px;
}
.detail-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
  gap: 24px;
}
.detail-head-left {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}
.detail-title {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
  margin: 0;
}
.detail-sub {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin: 4px 0 0;
}
.detail-meta {
  display: flex;
  gap: 28px;
}
.meta-kv {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}
.meta-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.meta-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.01em;
}

/* Tabs */
.tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 24px;
  border-bottom: 1px solid var(--border);
}
.tab {
  background: transparent;
  border: 0;
  border-bottom: 2px solid transparent;
  color: var(--text-3);
  padding: 12px 14px;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  letter-spacing: -0.005em;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: color 120ms, border-color 120ms;
}
.tab:hover { color: var(--text); }
.tab.active {
  color: var(--text);
  border-bottom-color: var(--text);
  font-weight: 600;
}
.tab-count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 1px 5px;
  border-radius: var(--radius-sm);
}
.tab.active .tab-count { color: var(--text); border-color: var(--border-strong); }
.tab-spacer { flex: 1; }
.tabs .btn { margin: 6px 0; }

.detail-body { padding: 18px 24px 24px; }

/* ===== Members table ===== */
.members-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.members-table thead th {
  text-align: left;
  padding: 8px 12px;
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  border-bottom: 1px solid var(--border);
}
.members-table th.right-th { text-align: right; }
.members-table tbody td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--bd-soft);
  color: var(--text);
}
.members-table tbody td.left { text-align: left; }
.members-table tbody td.right-td { text-align: right; }
.members-table tbody td.mono { font-family: var(--font-mono); font-size: 12px; color: var(--text-2); }
.members-table tbody td.dim { color: var(--text-3); font-size: 12px; }
.members-table tbody tr:last-child td { border-bottom: 0; }
.members-table tbody tr:hover { background: var(--canvas); }

.m-id {
  display: flex;
  align-items: center;
  gap: 10px;
}
.m-avatar {
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  background: var(--text);
  color: var(--elevated);
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 10.5px;
  letter-spacing: 0.02em;
  border-radius: 50%;
  position: relative;
}
.m-avatar::after {
  content: '';
  position: absolute;
  bottom: -1px;
  right: -1px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  border: 2px solid var(--elevated);
  background: var(--text-3);
}
.m-avatar.online::after { background: var(--pos); }
.m-avatar.idle::after   { background: var(--warn); }
.m-name {
  font-family: var(--font-sans);
  font-weight: 500;
  font-size: 13.5px;
  color: var(--text);
}

.role-tag {
  display: inline-flex;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.10em;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  text-transform: uppercase;
}
.role-tag.r-owner   { background: rgba(14, 14, 14, 0.06);  color: var(--text);   border-color: var(--border-strong); }
.role-tag.r-admin   { background: rgba(74, 144, 226, 0.10); color: var(--accent); border-color: rgba(74, 144, 226, 0.25); }
.role-tag.r-trader  { background: rgba(200, 242, 92, 0.20); color: #4a5300;       border-color: rgba(14, 14, 14, 0.20); }
.role-tag.r-analyst { background: rgba(255, 255, 255, 0.0); color: var(--text-2); border-color: var(--border); }
.role-tag.r-viewer  { background: rgba(0, 0, 0, 0.02);      color: var(--text-3); border-color: var(--border); }

.see-all {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.see-all a {
  color: var(--text);
  text-decoration: none;
  font-weight: 600;
  font-family: var(--font-sans);
  font-size: 13px;
}
.see-all a:hover { text-decoration: underline; }

/* ===== Budget detail ===== */
.bd-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 22px;
}
.bd-card {
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bd-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.bd-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--text);
}
.bd-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}
.bd-sub.pos { color: var(--pos); }
.bd-sub.warn { color: var(--warn); }
.bd-sub.neg { color: var(--neg); }

.bd-bar {
  margin-bottom: 22px;
}
.bd-bar-track {
  height: 12px;
  background: var(--sunken);
  border-radius: 999px;
  overflow: hidden;
  margin-bottom: 6px;
}
.bd-bar-ticks {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.bd-bar-ticks .warn-tick { color: var(--warn); }
.bd-bar-ticks .cap-tick  { color: var(--neg); }

.bd-rules {
  display: flex;
  flex-direction: column;
  gap: 1px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.bd-rule {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 16px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--bd-soft);
  font-size: 13px;
}
.bd-rule:last-child { border-bottom: 0; }
.rule-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  align-self: center;
}
.rule-v { color: var(--text); font-family: var(--font-sans); }

/* ===== Permissions ===== */
.perms-intro {
  font-size: 13px;
  color: var(--text-2);
  margin-bottom: 14px;
  line-height: 1.55;
}
.perms-intro strong { color: var(--text); font-weight: 500; }
.perms-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}
.perms-table thead th {
  text-align: left;
  padding: 9px 14px;
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  border-bottom: 1px solid var(--border);
  background: var(--canvas);
}
.perms-table th.center, .perms-table td.center { text-align: center; }
.perms-table tbody td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--bd-soft);
  color: var(--text);
}
.perms-table tbody tr:last-child td { border-bottom: 0; }
.cap-name {
  font-family: var(--font-sans);
  font-weight: 500;
  font-size: 13.5px;
  color: var(--text);
}
.cap-detail {
  font-size: 11.5px;
  color: var(--text-3);
  font-family: var(--font-sans);
  margin-top: 2px;
}
.perm {
  display: inline-grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: 12px;
}
.perm.allow {
  background: rgba(22, 163, 74, 0.10);
  color: var(--pos);
}
.perm.deny {
  background: var(--canvas);
  color: var(--text-3);
  border: 1px solid var(--border);
}

.perms-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed var(--bd-soft);
  font-size: 12px;
  color: var(--text-3);
}

/* ===== Activity (in detail + standalone card) ===== */
.activity-list {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.act-row {
  display: grid;
  grid-template-columns: 110px 90px 1fr;
  gap: 16px;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--bd-soft);
  font-size: 13px;
}
.act-row:last-child { border-bottom: 0; }
.act-row:hover { background: var(--canvas); }
.act-ts {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.act-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-mono);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  text-align: center;
  width: fit-content;
}
.act-badge.b-order  { background: rgba(74, 144, 226, 0.10); color: var(--accent); border-color: rgba(74, 144, 226, 0.25); }
.act-badge.b-budget { background: rgba(200, 242, 92, 0.20); color: #4a5300;       border-color: rgba(14, 14, 14, 0.20); }
.act-badge.b-member { background: rgba(0, 0, 0, 0.03);      color: var(--text-2); border-color: var(--border); }
.act-badge.b-key    { background: rgba(245, 158, 11, 0.10); color: var(--warn);   border-color: rgba(245, 158, 11, 0.25); }
.act-badge.b-auto   { background: transparent;              color: var(--text-3); border: 1px dashed var(--border-strong); }
.act-text {
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.5;
}
.act-who {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  margin-right: 4px;
}
.act-action { color: var(--text-2); }
.act-sub {
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 11.5px;
  margin-left: 6px;
  letter-spacing: 0.04em;
}

/* ============================================================
   Recent activity card (collapsible)
   ============================================================ */
.activity-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 24px;
}
.activity-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  cursor: pointer;
  user-select: none;
}
.activity-head .eyebrow { margin: 0; }
.activity-head-right {
  display: flex;
  align-items: center;
  gap: 14px;
}
.activity-head-right a {
  color: var(--text-2);
  text-decoration: none;
  font-family: var(--font-sans);
  font-size: 12px;
}
.activity-head-right a:hover { color: var(--text); text-decoration: underline; }
.activity-head .caret {
  width: 14px;
  height: 14px;
  color: var(--text-3);
  transition: transform 200ms;
}
.activity-card.collapsed .caret { transform: rotate(-90deg); }
.activity-body {
  max-height: 600px;
  overflow: hidden;
  transition: max-height 280ms ease-out;
  padding: 0 20px 18px;
}
.activity-card.collapsed .activity-body {
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
.activity-card .activity-list { border: 0; }
.activity-card .activity-list .act-row {
  border-bottom: 1px solid var(--bd-soft);
  padding-left: 0;
  padding-right: 0;
}
.activity-card .activity-list .act-row:last-child { border-bottom: 0; }

/* Page footer */
.page-foot {
  margin-top: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  padding-top: 14px;
  border-top: 1px solid var(--border);
}
.page-foot strong { color: var(--text); font-weight: 500; }
.page-foot a { color: var(--text); text-decoration: none; }
.page-foot a:hover { text-decoration: underline; }

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1100px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .stat:nth-child(2) { border-right: 0; }
  .sub-grid { grid-template-columns: 1fr; }
  .detail-head { flex-direction: column; align-items: stretch; gap: 16px; }
  .detail-meta { justify-content: flex-start; }
  .meta-kv { align-items: flex-start; }
  .bd-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 720px) {
  .page-head { flex-direction: column; align-items: stretch; }
  .head-actions { flex-direction: column; }
  .stats-grid { grid-template-columns: 1fr; }
  .stat { border-right: 0; border-bottom: 1px solid var(--border); }
  .stat:last-child { border-bottom: 0; }
  .bd-grid { grid-template-columns: 1fr; }
  .act-row { grid-template-columns: 1fr; gap: 4px; }
  .perms-table { font-size: 11.5px; }
  .perms-table th, .perms-table td { padding: 8px 6px; }
}
</style>
