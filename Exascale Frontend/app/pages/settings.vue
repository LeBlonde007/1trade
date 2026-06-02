<script setup lang="ts">
/**
 * /settings — Account settings shell.
 *
 * Renders inside the `app` layout (sidebar + topbar are shared).
 * Left nav lists every section; the right pane swaps content.
 * Profile is shown by default and fully populated; other sections
 * have header stubs so the shell is complete-but-minimal.
 */

definePageMeta({ layout: 'app' })
useHead({ title: 'Settings · Profile — Exascale' })

type SectionKey =
  | 'profile'
  | 'security'
  | 'api'
  | 'notifications'
  | 'display'
  | 'kyc'
  | 'billing'
  | 'team'
  | 'compliance'
  | 'danger'

interface SectionDef {
  key: SectionKey
  label: string
  group: string
  badge?: string
  danger?: boolean
}
const SECTIONS: SectionDef[] = [
  { key: 'profile',       label: 'Profile',                group: 'Personal' },
  { key: 'security',      label: 'Account & Security',     group: 'Personal' },
  { key: 'api',           label: 'API Keys',               group: 'Personal' },
  { key: 'notifications', label: 'Notifications',          group: 'Personal' },
  { key: 'display',       label: 'Display preferences',    group: 'Personal' },
  { key: 'kyc',           label: 'KYC status',             group: 'Account',  badge: 'Light · verified' },
  { key: 'billing',       label: 'Billing & Payment',      group: 'Account' },
  { key: 'team',          label: 'Team & Permissions',     group: 'Account',  badge: 'Enterprise' },
  { key: 'compliance',    label: 'Compliance & Audit',     group: 'Account' },
  { key: 'danger',        label: 'Danger zone',            group: 'Account',  danger: true },
]

const active = ref<SectionKey>('profile')
const current = computed(() => SECTIONS.find(s => s.key === active.value)!)

// =====================================================
// Profile form state (pre-populated)
// =====================================================
const profile = reactive({
  firstName: 'Marcus',
  lastName: 'Chen',
  displayName: 'Marcus Chen',
  email: 'marcus.chen@frontier.lab',
  emailVerified: true,
  phone: '+1 (212) 555-0142',
  phoneVerified: false,
  timezone: 'America/New_York',
  currency: 'USD',
  language: 'en-US',
})

const TIMEZONES = [
  { value: 'America/New_York',    label: 'America / New York · UTC−5' },
  { value: 'America/Los_Angeles', label: 'America / Los Angeles · UTC−8' },
  { value: 'America/Chicago',     label: 'America / Chicago · UTC−6' },
  { value: 'Europe/London',       label: 'Europe / London · UTC±0' },
  { value: 'Europe/Frankfurt',    label: 'Europe / Frankfurt · UTC+1' },
  { value: 'Europe/Helsinki',     label: 'Europe / Helsinki · UTC+2' },
  { value: 'Asia/Tokyo',          label: 'Asia / Tokyo · UTC+9' },
  { value: 'Asia/Singapore',      label: 'Asia / Singapore · UTC+8' },
  { value: 'Asia/Hong_Kong',      label: 'Asia / Hong Kong · UTC+8' },
  { value: 'UTC',                 label: 'UTC · Coordinated Universal Time' },
]
const CURRENCIES = [
  { value: 'USD', label: 'USD · US Dollar' },
  { value: 'EUR', label: 'EUR · Euro' },
  { value: 'GBP', label: 'GBP · British Pound' },
  { value: 'JPY', label: 'JPY · Japanese Yen' },
  { value: 'SGD', label: 'SGD · Singapore Dollar' },
  { value: 'CHF', label: 'CHF · Swiss Franc' },
]
const LANGUAGES = [
  { value: 'en-US', label: 'English (US)' },
  { value: 'en-GB', label: 'English (UK)' },
  { value: 'de-DE', label: 'Deutsch' },
  { value: 'fr-FR', label: 'Français' },
  { value: 'ja-JP', label: '日本語' },
  { value: 'zh-CN', label: '中文 (简体)' },
]

const initials = computed(() => (profile.firstName[0] ?? 'M') + (profile.lastName[0] ?? 'C'))

// =====================================================
// Account & Security minimal data
// =====================================================
const SESSIONS = [
  { id: 'this',  device: 'MacBook Pro · Chrome 124',  location: 'New York, US',  ip: '74.125.224.12', last: 'Active now',     current: true },
  { id: 's2',    device: 'iPhone 15 Pro · iOS 17',     location: 'New York, US',  ip: '172.58.232.40', last: '2 hours ago',    current: false },
  { id: 's3',    device: 'Windows 11 · Edge 124',      location: 'Tokyo, JP',     ip: '210.140.92.18', last: '3 days ago',     current: false },
]
const TWOFA = { method: 'TOTP · authenticator app', enrolled: '2026-03-14', backupCodes: 8 }

// =====================================================
// API Keys
// =====================================================
type Scope = 'trade' | 'read' | 'market-data' | 'billing' | 'admin'

interface ApiKey {
  id: string
  name: string
  prefix: string
  scopes: Scope[]
  created: string
  lastUsed: string
  createdBy: string
  status: 'active' | 'revoked'
}

const SCOPE_META: Record<Scope, { label: string; cls: string; help: string }> = {
  trade:         { label: 'trade',        cls: 'sc-trade',  help: 'Place / cancel orders · open / close positions' },
  read:          { label: 'read',         cls: 'sc-read',   help: 'Account balances · positions · order history' },
  'market-data': { label: 'market-data',  cls: 'sc-data',   help: 'Live prices · order book · trade tape · candles' },
  billing:       { label: 'billing',      cls: 'sc-bill',   help: 'Invoices · payment methods · payout settings' },
  admin:         { label: 'admin',        cls: 'sc-admin',  help: 'Full team management · audit access · settings' },
}
const ALL_SCOPES: Scope[] = ['trade', 'read', 'market-data', 'billing', 'admin']

// F20×F02 — live API keys: list + create + revoke run against platform-core (real one-time secret).
// Real keys carry arbitrary scope strings, so rendering tolerates scopes not in SCOPE_META.
const liveKeys = useKeys()
const keyError = ref('')

/** scopeClass returns a pill class for a scope, falling back to a neutral style for live scopes. */
function scopeClass(s: string): string {
  return SCOPE_META[s as Scope]?.cls ?? 'sc-read'
}
/** scopeLabel returns a scope's display label (the raw string for live scopes not in SCOPE_META). */
function scopeLabel(s: string): string {
  return SCOPE_META[s as Scope]?.label ?? s
}

/** displayKeys is the rendered key list — the tenant's live platform keys. */
const displayKeys = computed<ApiKey[]>(() => {
  return liveKeys.keys.value.map((k) => ({
    id: k.id,
    name: k.name,
    prefix: k.prefix,
    scopes: (k.scopes ?? []) as Scope[],
    created: k.created_at ? k.created_at.slice(0, 10) : '—',
    lastUsed: '—',
    createdBy: profile.email,
    status: k.revoked ? 'revoked' : 'active',
  }))
})

type Expiration = 'never' | '30d' | '90d' | '1y' | 'custom'
const EXP_OPTIONS: Array<{ value: Expiration; label: string; sub: string }> = [
  { value: '30d',    label: '30 days',  sub: 'Recommended for short-lived bots' },
  { value: '90d',    label: '90 days',  sub: 'Standard rotation cadence' },
  { value: '1y',     label: '1 year',   sub: 'Production · auto-rotates' },
  { value: 'never',  label: 'Never',    sub: 'Discouraged · audit-flagged' },
]

const showApiModal = ref(false)
const apiModalStep = ref<'form' | 'generated'>('form')
const newKeyName = ref('Backfill replay · staging')
const newKeyScopes = reactive<Record<Scope, boolean>>({
  trade: false,
  read: true,
  'market-data': true,
  billing: false,
  admin: false,
})
const newKeyExp = ref<Expiration>('90d')
const generatedKeyFull = ref('')
const generatedKeyId = ref('')
const copyToast = ref<'' | 'copied'>('')

function openCreateKeyModal() {
  showApiModal.value = true
  apiModalStep.value = 'form'
  newKeyName.value = 'Backfill replay · staging'
  newKeyScopes.trade = false
  newKeyScopes.read = true
  newKeyScopes['market-data'] = true
  newKeyScopes.billing = false
  newKeyScopes.admin = false
  newKeyExp.value = '90d'
}
function closeApiModal() {
  showApiModal.value = false
  keyError.value = ''
  liveKeys.dismissSecret() // clear the one-time secret from memory
}

function selectedScopes(): Scope[] {
  return ALL_SCOPES.filter(s => newKeyScopes[s])
}

const canGenerate = computed(() => newKeyName.value.trim().length > 0 && selectedScopes().length > 0)

async function generateKey() {
  if (!canGenerate.value) return
  const scopes = selectedScopes()
  keyError.value = ''
  // Mint through platform-core; the real one-time secret comes back in the response.
  try {
    const r = await liveKeys.create(newKeyName.value.trim(), scopes)
    generatedKeyFull.value = r.secret
    generatedKeyId.value = r.id
    apiModalStep.value = 'generated'
  } catch (e: unknown) {
    const ex = e as { data?: { message?: string }; statusMessage?: string }
    keyError.value = ex?.data?.message || ex?.statusMessage || 'Could not create key.'
  }
}

async function copyGeneratedKey() {
  try {
    await navigator.clipboard.writeText(generatedKeyFull.value)
    copyToast.value = 'copied'
    setTimeout(() => { copyToast.value = '' }, 1600)
  } catch {
    // clipboard unavailable
  }
}

const expLabel = computed(() => EXP_OPTIONS.find(o => o.value === newKeyExp.value)?.label ?? 'Never')

async function revokeKey(id: string) {
  try { await liveKeys.revoke(id) } catch { keyError.value = 'Could not revoke key.' }
}

// =====================================================
// Routing-style hash sync (so a link to /settings#api opens API)
// =====================================================
const route = useRoute()
const router = useRouter()
function goTo(k: SectionKey) {
  active.value = k
  if (typeof window !== 'undefined') {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
  router.replace({ path: route.path, hash: '#' + k })
}
onMounted(() => {
  const h = route.hash?.replace('#', '') as SectionKey
  if (h && SECTIONS.find(s => s.key === h)) active.value = h
  liveKeys.load().catch(() => { keyError.value = 'Could not load keys.' })
})
</script>

<template>
  <div class="settings-page">
    <!-- Sub-topbar with breadcrumbs -->
    <div class="subbar">
      <div class="crumbs">
        <span>Account</span>
        <span class="sep">›</span>
        <span class="cur">Settings</span>
        <span class="sep">›</span>
        <span class="cur strong">{{ current.label }}</span>
      </div>
      <div class="subbar-right">
        <span class="env-pill">PAPER</span>
      </div>
    </div>

    <div class="settings-shell">
      <!-- ============ LEFT NAV ============ -->
      <aside class="settings-nav" aria-label="Settings sections">
        <div class="nav-head">
          <h1 class="nav-title">Settings</h1>
          <p class="nav-sub">Personalize your account, security and venue display.</p>
        </div>

        <div class="nav-group">
          <div class="nav-group-head">Personal</div>
          <ul>
            <li v-for="s in SECTIONS.filter(x => x.group === 'Personal')" :key="s.key">
              <button
                type="button"
                class="nav-item"
                :class="{ active: active === s.key, danger: s.danger }"
                @click="goTo(s.key)"
              >
                <span class="nav-label">{{ s.label }}</span>
                <span v-if="s.badge" class="nav-badge">{{ s.badge }}</span>
              </button>
            </li>
          </ul>
        </div>

        <div class="nav-group">
          <div class="nav-group-head">Account</div>
          <ul>
            <li v-for="s in SECTIONS.filter(x => x.group === 'Account')" :key="s.key">
              <button
                type="button"
                class="nav-item"
                :class="{ active: active === s.key, danger: s.danger }"
                @click="goTo(s.key)"
              >
                <span class="nav-label">{{ s.label }}</span>
                <span v-if="s.badge" class="nav-badge">{{ s.badge }}</span>
              </button>
            </li>
          </ul>
        </div>

        <div class="nav-foot">
          <span class="nf-k">Build</span>
          <span class="nf-v">venue v2.4.1</span>
        </div>
      </aside>

      <!-- ============ CONTENT ============ -->
      <main class="settings-content">
        <!-- ===== PROFILE ===== -->
        <template v-if="active === 'profile'">
          <header class="section-head">
            <div>
              <div class="eyebrow"><span class="dot" /> Settings · Personal</div>
              <h2 class="section-title">Profile</h2>
              <p class="section-sub">
                How you appear on the venue, how we contact you, and your default formatting.
              </p>
            </div>
          </header>

          <!-- Avatar card -->
          <div class="card">
            <div class="card-head"><span class="eyebrow"><span class="dot" /> Avatar</span></div>
            <div class="avatar-row">
              <div class="avatar-big" aria-hidden="true">{{ initials }}</div>
              <div class="avatar-body">
                <div class="avatar-name">{{ profile.displayName }}</div>
                <div class="avatar-meta">PNG / JPG / SVG · square crop · ≤ 2MB · 256×256 recommended</div>
                <div class="avatar-actions">
                  <button type="button" class="btn secondary">
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                      <path d="M3 11l2-2 3 3 3-4 2 2v3H3z" />
                      <path d="M2 13.5h12" />
                    </svg>
                    Upload new
                  </button>
                  <button type="button" class="btn ghost">Remove</button>
                </div>
              </div>
            </div>
          </div>

          <!-- Identity card -->
          <div class="card">
            <div class="card-head"><span class="eyebrow"><span class="dot" /> Identity</span></div>
            <div class="card-body">
              <div class="field-grid">
                <div class="field col-6">
                  <label class="field-label">— First name</label>
                  <input v-model="profile.firstName" class="input" type="text" autocomplete="given-name" />
                </div>
                <div class="field col-6">
                  <label class="field-label">— Last name</label>
                  <input v-model="profile.lastName" class="input" type="text" autocomplete="family-name" />
                </div>
                <div class="field col-12">
                  <label class="field-label">
                    — Display name
                    <span class="hint">Shown in chat, leaderboards and trade tape</span>
                  </label>
                  <input v-model="profile.displayName" class="input" type="text" />
                </div>
              </div>
            </div>
          </div>

          <!-- Contact card -->
          <div class="card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Contact</span>
              <span class="card-meta">2FA &amp; recovery contact</span>
            </div>
            <div class="card-body">
              <div class="field-grid">
                <div class="field col-12">
                  <label class="field-label">— Email</label>
                  <div class="input-affix">
                    <input v-model="profile.email" class="input mono-input" type="email" autocomplete="email" />
                    <span class="affix-right">
                      <span class="status-tag verified">
                        <span class="dot" />
                        Verified
                      </span>
                      <button type="button" class="affix-link">Change</button>
                    </span>
                  </div>
                  <span class="field-hint">Primary login. Used for receipts, 2FA backup and security alerts.</span>
                </div>

                <div class="field col-12">
                  <label class="field-label">— Phone</label>
                  <div class="input-affix">
                    <input v-model="profile.phone" class="input mono-input" type="tel" autocomplete="tel" />
                    <span class="affix-right">
                      <span class="status-tag pending">
                        <span class="dot" />
                        Verify
                      </span>
                      <button type="button" class="affix-link primary">Send code →</button>
                    </span>
                  </div>
                  <span class="field-hint">Optional · enables SMS recovery if you lose your authenticator.</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Preferences card -->
          <div class="card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Locale &amp; formatting</span>
            </div>
            <div class="card-body">
              <div class="field-grid">
                <div class="field col-12">
                  <label class="field-label">— Timezone</label>
                  <select v-model="profile.timezone" class="select">
                    <option v-for="tz in TIMEZONES" :key="tz.value" :value="tz.value">{{ tz.label }}</option>
                  </select>
                  <span class="field-hint">Used for chart axes, daily prints and scheduled reports.</span>
                </div>
                <div class="field col-6">
                  <label class="field-label">— Default currency</label>
                  <select v-model="profile.currency" class="select">
                    <option v-for="c in CURRENCIES" :key="c.value" :value="c.value">{{ c.label }}</option>
                  </select>
                </div>
                <div class="field col-6">
                  <label class="field-label">— Language</label>
                  <select v-model="profile.language" class="select">
                    <option v-for="l in LANGUAGES" :key="l.value" :value="l.value">{{ l.label }}</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

          <!-- Save bar -->
          <div class="save-bar">
            <span class="save-hint">Changes apply immediately on save · email change requires re-verification.</span>
            <div class="save-actions">
              <button type="button" class="btn secondary">Discard</button>
              <button type="button" class="btn primary">Save changes</button>
            </div>
          </div>
        </template>

        <!-- ===== ACCOUNT & SECURITY ===== -->
        <template v-else-if="active === 'security'">
          <header class="section-head">
            <div>
              <div class="eyebrow"><span class="dot" /> Settings · Personal</div>
              <h2 class="section-title">Account &amp; Security</h2>
              <p class="section-sub">Password, two-factor authentication and active sessions.</p>
            </div>
          </header>

          <div class="card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Password</span>
              <span class="card-meta">Last changed 2026-04-02</span>
            </div>
            <div class="card-body">
              <div class="row-action">
                <div class="row-text">
                  <div class="row-title">Set a new password</div>
                  <div class="row-sub">Minimum 12 characters · must include a number and a symbol</div>
                </div>
                <button type="button" class="btn secondary">Change password</button>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Two-factor authentication</span>
              <span class="status-tag verified">
                <span class="dot" />
                Enabled
              </span>
            </div>
            <div class="card-body">
              <dl class="kv-grid">
                <dt>Method</dt><dd>{{ TWOFA.method }}</dd>
                <dt>Enrolled</dt><dd>{{ TWOFA.enrolled }}</dd>
                <dt>Backup codes</dt><dd>{{ TWOFA.backupCodes }} of 10 unused</dd>
              </dl>
              <div class="row-actions-end">
                <button type="button" class="btn ghost">Regenerate backup codes</button>
                <button type="button" class="btn secondary">Reset 2FA</button>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Active sessions</span>
              <button type="button" class="head-link">Sign out everywhere</button>
            </div>
            <div class="card-body">
              <table class="table">
                <thead>
                  <tr>
                    <th class="left">Device</th>
                    <th class="left">Location</th>
                    <th class="left">IP</th>
                    <th class="left">Last activity</th>
                    <th class="right-th"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="s in SESSIONS" :key="s.id">
                    <td class="left">
                      {{ s.device }}
                      <span v-if="s.current" class="status-tag verified inline">This device</span>
                    </td>
                    <td class="left">{{ s.location }}</td>
                    <td class="left mono">{{ s.ip }}</td>
                    <td class="left">{{ s.last }}</td>
                    <td class="right-td">
                      <button type="button" class="btn-mini" :disabled="s.current">Revoke</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>

        <!-- ===== API KEYS ===== -->
        <template v-else-if="active === 'api'">
          <header class="section-head api-head">
            <div>
              <div class="eyebrow"><span class="dot" /> Settings · Personal</div>
              <h2 class="section-title">API Keys</h2>
              <p class="section-sub">Programmatic access to Exascale APIs. Scope each key to the minimum permissions it needs and rotate often.</p>
            </div>
            <button type="button" class="btn primary" @click="openCreateKeyModal">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                <path d="M8 3v10M3 8h10" />
              </svg>
              Create new key
            </button>
          </header>

          <!-- Active keys table -->
          <div class="card api-card">
            <div class="card-head">
              <span class="eyebrow"><span class="dot" /> Active keys</span>
              <span class="card-meta">
                {{ displayKeys.length }} {{ displayKeys.length === 1 ? 'key' : 'keys' }} · <NuxtLink to="/enterprise/audit">Audit log →</NuxtLink>
              </span>
            </div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr>
                    <th class="left">Name</th>
                    <th class="left">Key prefix</th>
                    <th class="left">Scopes</th>
                    <th class="left">Created</th>
                    <th class="left">Last used</th>
                    <th class="left">Created by</th>
                    <th class="left">Status</th>
                    <th class="right-th">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="displayKeys.length === 0">
                    <td class="left dim" colspan="8">No API keys yet — create one to call the inference API.</td>
                  </tr>
                  <tr v-for="k in displayKeys" :key="k.id">
                    <td class="left name-cell">
                      <span class="key-name">{{ k.name }}</span>
                    </td>
                    <td class="left">
                      <span class="key-prefix mono">{{ k.prefix }}…</span>
                    </td>
                    <td class="left">
                      <span class="scope-row">
                        <span
                          v-for="sc in k.scopes"
                          :key="sc"
                          class="scope-badge"
                          :class="scopeClass(sc)"
                        >{{ scopeLabel(sc) }}</span>
                      </span>
                    </td>
                    <td class="left mono dim">{{ k.created }}</td>
                    <td class="left mono">{{ k.lastUsed }}</td>
                    <td class="left mono dim">{{ k.createdBy }}</td>
                    <td class="left">
                      <span v-if="k.status === 'revoked'" class="status-tag revoked">
                        <span class="dot" />
                        Revoked
                      </span>
                      <span v-else class="status-tag verified">
                        <span class="dot" />
                        Active
                      </span>
                    </td>
                    <td class="right-td">
                      <div class="row-actions-inline">
                        <button type="button" class="btn-mini ghost">Edit</button>
                        <button v-if="k.status !== 'revoked'" type="button" class="btn-mini" @click="revokeKey(k.id)">Revoke</button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Security best-practices help -->
          <div class="card help-card">
            <div class="help-grid">
              <div class="help-cell">
                <div class="help-icon">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="3.5" y="7" width="9" height="6.5" />
                    <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
                  </svg>
                </div>
                <div class="help-text">
                  <div class="help-title">Treat keys like passwords</div>
                  <p>Never commit them to git, never paste into chat. Use a secret manager.</p>
                </div>
              </div>
              <div class="help-cell">
                <div class="help-icon">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="8" cy="8" r="6" />
                    <path d="M8 4v4l2.5 2" />
                  </svg>
                </div>
                <div class="help-text">
                  <div class="help-title">Rotate on a schedule</div>
                  <p>Set expiration to 90 days for production. CI/CD keys to 30. Audit before rotation.</p>
                </div>
              </div>
              <div class="help-cell">
                <div class="help-icon">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M2 8h12M5 8a3 3 0 016 0M5 8a3 3 0 003 3M5 8a3 3 0 013-3" />
                  </svg>
                </div>
                <div class="help-text">
                  <div class="help-title">Scope to minimum</div>
                  <p>A market-data dashboard does not need trade. CI does not need admin during deploys.</p>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- ===== STUB SECTIONS ===== -->
        <template v-else>
          <header class="section-head">
            <div>
              <div class="eyebrow"><span class="dot" /> Settings · {{ current.group }}</div>
              <h2 class="section-title">{{ current.label }}</h2>
              <p class="section-sub">
                <template v-if="active === 'notifications'">
                  Choose what triggers an email, push notification or webhook.
                </template>
                <template v-else-if="active === 'display'">
                  Theme, density and table formatting defaults.
                </template>
                <template v-else-if="active === 'kyc'">
                  Light KYC is in effect. Upgrade to Full KYC to enable real-money trading.
                </template>
                <template v-else-if="active === 'billing'">
                  Cards, bank accounts, paid plans and invoice history.
                </template>
                <template v-else-if="active === 'team'">
                  Invite teammates, scope their permissions, view audit trails.
                </template>
                <template v-else-if="active === 'compliance'">
                  Annual attestations, source-of-funds review, sanctions screening.
                </template>
                <template v-else-if="active === 'danger'">
                  Irreversible operations on the account.
                </template>
              </p>
            </div>
          </header>

          <div class="card stub-card">
            <div class="stub-mark">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <rect x="3" y="3" width="10" height="10" />
                <path d="M3 7h10M7 3v10" />
              </svg>
            </div>
            <div class="stub-title">{{ current.label }} · shell only</div>
            <p class="stub-text">
              This section is wired into the navigation but its content is not part of the
              current milestone. The shell preserves layout, breadcrumbs and section meta
              so the surface can be filled in without re-touching the chrome.
            </p>
            <div class="stub-actions">
              <button type="button" class="btn secondary" @click="goTo('profile')">← Back to Profile</button>
            </div>
          </div>
        </template>
      </main>
    </div>

    <!-- ============ CREATE API KEY MODAL ============ -->
    <div
      v-if="showApiModal"
      class="modal-scrim"
      role="dialog"
      aria-modal="true"
      :aria-label="apiModalStep === 'form' ? 'Create API key' : 'New API key generated'"
      @click.self="apiModalStep === 'form' ? closeApiModal() : null"
    >
      <div class="modal">
        <!-- ===== FORM STATE ===== -->
        <template v-if="apiModalStep === 'form'">
          <header class="modal-head">
            <div>
              <div class="eyebrow"><span class="dot" /> Settings · API</div>
              <h3 class="modal-title">Create new API key</h3>
            </div>
            <button type="button" class="modal-x" aria-label="Close" @click="closeApiModal">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M4 4l8 8M12 4l-8 8" />
              </svg>
            </button>
          </header>

          <div class="modal-body">
            <div class="field">
              <label class="field-label">— Key name</label>
              <input v-model="newKeyName" class="input" type="text" placeholder="e.g. Backfill replay · staging" />
              <span class="field-hint">A label for your team. Not used by the API.</span>
            </div>

            <div class="field">
              <label class="field-label">
                — Scopes
                <span class="hint">{{ selectedScopes().length }} of {{ ALL_SCOPES.length }} selected</span>
              </label>
              <div class="scope-list">
                <label
                  v-for="sc in ALL_SCOPES"
                  :key="sc"
                  class="scope-opt"
                  :class="{ checked: newKeyScopes[sc] }"
                >
                  <input v-model="newKeyScopes[sc]" type="checkbox" />
                  <span class="scope-cb">
                    <svg v-if="newKeyScopes[sc]" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="square">
                      <path d="M3 8.5l3.2 3.2L13 5" />
                    </svg>
                  </span>
                  <span class="scope-body">
                    <span class="scope-label-row">
                      <span class="scope-badge" :class="SCOPE_META[sc].cls">{{ SCOPE_META[sc].label }}</span>
                      <span v-if="sc === 'admin'" class="scope-warn">privileged · audit-logged</span>
                    </span>
                    <span class="scope-help">{{ SCOPE_META[sc].help }}</span>
                  </span>
                </label>
              </div>
            </div>

            <div class="field">
              <label class="field-label">— Expiration</label>
              <div class="exp-list">
                <label
                  v-for="o in EXP_OPTIONS"
                  :key="o.value"
                  class="exp-opt"
                  :class="{ checked: newKeyExp === o.value }"
                >
                  <input v-model="newKeyExp" type="radio" :value="o.value" />
                  <span class="exp-radio" />
                  <span class="exp-body">
                    <span class="exp-label">{{ o.label }}</span>
                    <span class="exp-sub">{{ o.sub }}</span>
                  </span>
                </label>
              </div>
              <span class="field-hint">
                You'll get a reminder email 7 days before expiration. Keys can be rotated any time.
              </span>
            </div>
          </div>

          <footer class="modal-foot">
            <span v-if="keyError" class="foot-hint err">{{ keyError }}</span>
            <span v-else class="foot-hint">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect x="3.5" y="7" width="9" height="6.5" />
                <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
              </svg>
              Key is shown once at generation. Save it immediately.
            </span>
            <div class="foot-actions">
              <button type="button" class="btn secondary" @click="closeApiModal">Cancel</button>
              <button type="button" class="btn primary" :disabled="!canGenerate" @click="generateKey">
                Generate key
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                  <path d="M3 8h10M9 4l4 4-4 4" />
                </svg>
              </button>
            </div>
          </footer>
        </template>

        <!-- ===== GENERATED STATE ===== -->
        <template v-else>
          <header class="modal-head">
            <div>
              <div class="eyebrow warn"><span class="dot" /> One-time reveal</div>
              <h3 class="modal-title">Save your new API key</h3>
            </div>
          </header>

          <div class="modal-body">
            <div class="warn-banner">
              <span class="warn-icon">!</span>
              <div class="warn-text">
                <strong>This key won't be shown again.</strong>
                Copy it now into your secret manager. If you lose it, you'll need to revoke and create a new one.
              </div>
            </div>

            <div class="key-reveal">
              <code class="key-string">{{ generatedKeyFull }}</code>
              <button type="button" class="copy-btn" @click="copyGeneratedKey">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                  <rect x="5" y="5" width="8" height="8" />
                  <path d="M3 11V3h8" />
                </svg>
                <span>{{ copyToast === 'copied' ? 'Copied' : 'Copy' }}</span>
              </button>
            </div>

            <dl class="key-meta">
              <dt>Name</dt><dd>{{ newKeyName }}</dd>
              <dt>Scopes</dt>
              <dd class="meta-tags">
                <span
                  v-for="sc in selectedScopes()"
                  :key="sc"
                  class="scope-badge"
                  :class="SCOPE_META[sc].cls"
                >{{ SCOPE_META[sc].label }}</span>
              </dd>
              <dt>Expires</dt><dd>{{ expLabel }}</dd>
              <dt>Created by</dt><dd>{{ profile.email }}</dd>
            </dl>
          </div>

          <footer class="modal-foot">
            <span class="foot-hint check">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square">
                <path d="M3 8.5l3.2 3.2L13 5" />
              </svg>
              Key added to the active list as <strong>{{ newKeyName }}</strong>
            </span>
            <div class="foot-actions">
              <button type="button" class="btn primary" @click="closeApiModal">
                I have saved this key
              </button>
            </div>
          </footer>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.55;
  color: var(--text);
  font-feature-settings: 'tnum' on, 'ss01' on;
  background: var(--canvas);
  min-height: 100%;
}

/* Sub-topbar */
.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: var(--canvas);
  gap: 16px;
}
.crumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .cur { color: var(--text-2); }
.crumbs .cur.strong { color: var(--text); }
.subbar-right { margin-left: auto; display: flex; align-items: center; gap: 12px; }
.env-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  padding: 4px 8px;
  background: var(--info);
  color: #fff;
  border-radius: var(--radius-sm);
  font-weight: 600;
  opacity: 0.9;
}

/* Page shell */
.settings-shell {
  max-width: 1280px;
  margin: 0 auto;
  padding: 32px 24px 80px;
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 40px;
  align-items: start;
}

/* ============================================================
   Left settings nav
   ============================================================ */
.settings-nav {
  position: sticky;
  top: 16px;
  align-self: start;
}
.nav-head {
  padding-bottom: 18px;
  margin-bottom: 18px;
  border-bottom: 1px solid var(--border);
}
.nav-title {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0 0 6px;
  color: var(--text);
}
.nav-sub {
  font-size: 12px;
  color: var(--text-3);
  line-height: 1.5;
  margin: 0;
  max-width: 220px;
}

.nav-group { margin-bottom: 18px; }
.nav-group-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.settings-nav ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.nav-item {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  align-items: center;
  width: 100%;
  background: transparent;
  border: 0;
  color: var(--text-2);
  font-family: inherit;
  font-size: 13px;
  text-align: left;
  padding: 7px 12px 7px 14px;
  border-left: 2px solid transparent;
  cursor: pointer;
  transition: color 120ms, background 120ms, border-color 120ms;
  letter-spacing: -0.005em;
}
.nav-item:hover { color: var(--text); background: var(--hover); }
.nav-item.active {
  color: var(--text);
  background: var(--elevated);
  border-left-color: var(--brand);
  font-weight: 500;
}
.nav-item.danger { color: var(--neg); }
.nav-item.danger:hover { color: var(--neg); background: var(--neg-soft); }
.nav-item.danger.active { background: var(--neg-soft); border-left-color: var(--neg); }
.nav-label { line-height: 1.3; }
.nav-badge {
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.10em;
  text-transform: uppercase;
  color: var(--text-3);
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 600;
  white-space: nowrap;
}
.nav-item.active .nav-badge { color: var(--text-2); border-color: var(--border-strong); }

.nav-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.nf-k { font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; }
.nf-v { color: var(--text-2); font-variant-numeric: tabular-nums; }

/* ============================================================
   Content
   ============================================================ */
.settings-content { min-width: 0; }

.section-head { margin-bottom: 24px; }
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
  margin-bottom: 10px;
}
.eyebrow .dot {
  width: 5px;
  height: 5px;
  background: var(--brand);
  display: inline-block;
}
.section-title {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.022em;
  line-height: 1.1;
  margin: 0 0 8px;
  color: var(--text);
}
.section-sub {
  color: var(--text-2);
  font-size: 14px;
  margin: 0;
  max-width: 640px;
  line-height: 1.55;
}

/* ============================================================
   Card
   ============================================================ */
.card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 16px;
}
.card-head {
  padding: 14px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.card-head .eyebrow { margin: 0; }
.card-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.head-link {
  background: transparent;
  border: 0;
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 12px;
  cursor: pointer;
}
.head-link:hover { color: var(--text); text-decoration: underline; }
.card-body { padding: 18px 20px 20px; }

/* Avatar card */
.avatar-row {
  display: grid;
  grid-template-columns: 96px 1fr;
  gap: 24px;
  align-items: center;
  padding: 22px 20px;
}
.avatar-big {
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--brand);
  color: var(--canvas);
  display: grid;
  place-items: center;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 32px;
  letter-spacing: -0.02em;
}
.avatar-body { min-width: 0; }
.avatar-name {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
  margin-bottom: 4px;
}
.avatar-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-bottom: 14px;
}
.avatar-actions { display: flex; gap: 8px; }

/* ============================================================
   Fields
   ============================================================ */
.field-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 16px 14px;
}
.field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field.col-12 { grid-column: span 12; }
.field.col-6  { grid-column: span 6; }
.field-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-2);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.field-label .hint {
  font-family: var(--font-sans);
  font-size: 11.5px;
  text-transform: none;
  letter-spacing: 0;
  color: var(--text-3);
  font-weight: 500;
}
.field-hint {
  font-size: 12px;
  color: var(--text-3);
  line-height: 1.45;
  font-family: var(--font-sans);
}

.input,
.select {
  height: 38px;
  background: var(--canvas);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text);
  width: 100%;
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
  -webkit-appearance: none;
  appearance: none;
  font-feature-settings: 'ss01';
}
.input:hover,
.select:hover { border-color: rgba(255, 255, 255, 0.24); }
.input:focus,
.select:focus {
  border-color: var(--info);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.20);
}
.mono-input {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13.5px;
  letter-spacing: 0.02em;
}
.select {
  background-image: url("data:image/svg+xml,%3Csvg width='10' height='6' viewBox='0 0 10 6' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1L5 5L9 1' stroke='%239A9A95' stroke-width='1.4' fill='none' stroke-linecap='square'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 32px;
  cursor: pointer;
}

.input-affix {
  position: relative;
  display: flex;
  align-items: center;
}
.input-affix .input { padding-right: 180px; }
.affix-right {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.affix-link {
  background: transparent;
  border: 0;
  cursor: pointer;
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-2);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
}
.affix-link:hover { color: var(--text); background: var(--hover); }
.affix-link.primary { color: var(--brand); }
.affix-link.primary:hover { color: var(--brand-hov); background: var(--hover); }

/* ============================================================
   Status tags
   ============================================================ */
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.status-tag .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}
.status-tag.verified {
  background: var(--pos-soft);
  color: var(--pos);
  border: 1px solid rgba(25, 195, 125, 0.25);
}
.status-tag.revoked {
  background: var(--hover);
  color: var(--text-3);
  border: 1px solid var(--border);
}
.status-tag.pending {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border: 1px solid rgba(245, 158, 11, 0.30);
}
.status-tag.inline {
  margin-left: 8px;
  padding: 2px 6px;
  font-size: 9px;
}

/* ============================================================
   Save bar
   ============================================================ */
.save-bar {
  position: sticky;
  bottom: 12px;
  z-index: 5;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--overlay);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 12px 18px;
  margin-top: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.30);
}
.save-hint {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
}
.save-actions { display: flex; gap: 8px; }

/* ============================================================
   Buttons
   ============================================================ */
.btn {
  height: 36px;
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
}
.btn svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.6; }
.btn.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
  font-weight: 600;
}
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.secondary {
  background: var(--canvas);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover { background: var(--hover); border-color: rgba(255, 255, 255, 0.24); }
.btn.ghost {
  background: transparent;
  color: var(--text-2);
}
.btn.ghost:hover { color: var(--text); background: var(--hover); }

.btn-mini {
  height: 26px;
  padding: 0 10px;
  background: transparent;
  border: 1px solid var(--border-strong);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-transform: uppercase;
}
.btn-mini:hover {
  background: var(--neg-soft);
  color: var(--neg);
  border-color: var(--neg);
}
.btn-mini:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

/* ============================================================
   Row action (security card)
   ============================================================ */
.row-action {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}
.row-text .row-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}
.row-text .row-sub {
  font-size: 12px;
  color: var(--text-3);
  margin-top: 4px;
}
.row-actions-end {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.kv-grid {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 8px 24px;
  margin: 0;
}
.kv-grid dt {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  align-self: center;
}
.kv-grid dd {
  margin: 0;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--text);
}

/* Table (sessions) */
.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.table thead th {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
  text-align: left;
}
.table thead th.right-th { text-align: right; }
.table tbody td {
  padding: 12px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  color: var(--text);
  font-size: 13px;
}
.table tbody tr:last-child td { border-bottom: 0; }
.table tbody td.right-td { text-align: right; }
.table tbody td.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text-2);
  font-size: 12px;
}

/* ============================================================
   Stub
   ============================================================ */
.stub-card {
  text-align: center;
  padding: 56px 24px 48px;
}
.stub-mark {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  display: grid;
  place-items: center;
  color: var(--text-3);
}
.stub-mark svg { width: 20px; height: 20px; }
.stub-title {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
  margin-bottom: 8px;
}
.stub-text {
  font-size: 13px;
  color: var(--text-2);
  max-width: 480px;
  margin: 0 auto 24px;
  line-height: 1.6;
}
.stub-actions {
  display: flex;
  justify-content: center;
}

/* Responsive */
@media (max-width: 1024px) {
  .settings-shell {
    grid-template-columns: 200px minmax(0, 1fr);
    gap: 24px;
  }
}
@media (max-width: 820px) {
  .settings-shell {
    grid-template-columns: 1fr;
  }
  .settings-nav {
    position: static;
    border-bottom: 1px solid var(--border);
    padding-bottom: 12px;
    margin-bottom: 12px;
  }
  .avatar-row { grid-template-columns: 72px 1fr; gap: 16px; }
  .avatar-big { width: 72px; height: 72px; font-size: 24px; }
  .field.col-6 { grid-column: span 12; }
  .save-bar { flex-direction: column; gap: 12px; align-items: stretch; }
  .save-actions { justify-content: flex-end; }
}

/* ============================================================
   API Keys section
   ============================================================ */
.section-head.api-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
}
.section-head.api-head .btn { flex-shrink: 0; }

.api-card .card-meta a {
  color: var(--text-2);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
  padding-bottom: 1px;
  margin-left: 6px;
}
.api-card .card-meta a:hover { color: var(--text); }

.api-table-wrap {
  overflow-x: auto;
}
.api-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}
.api-table thead th {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
  text-align: left;
}
.api-table thead th.right-th { text-align: right; }
.api-table tbody td {
  padding: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 13px;
  vertical-align: middle;
  color: var(--text);
}
.api-table tbody tr:last-child td { border-bottom: 0; }
.api-table tbody tr:hover { background: rgba(255, 255, 255, 0.02); }
.api-table tbody td.dim { color: var(--text-3); }
.api-table tbody td.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}
.api-table .name-cell {
  font-weight: 500;
  letter-spacing: -0.005em;
  min-width: 180px;
}
.api-table .key-name { color: var(--text); }
.api-table .key-prefix {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--canvas);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  color: var(--text);
  letter-spacing: 0.02em;
}
.scope-row {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
}
.scope-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  border: 1px solid transparent;
}
.scope-badge.sc-trade {
  background: rgba(200, 242, 92, 0.10);
  color: var(--brand);
  border-color: rgba(200, 242, 92, 0.25);
}
.scope-badge.sc-read {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-2);
  border-color: var(--border);
}
.scope-badge.sc-data {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border-color: rgba(74, 144, 226, 0.25);
}
.scope-badge.sc-bill {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border-color: rgba(245, 158, 11, 0.25);
}
.scope-badge.sc-admin {
  background: rgba(239, 68, 68, 0.10);
  color: var(--neg);
  border-color: rgba(239, 68, 68, 0.25);
}

.row-actions-inline {
  display: inline-flex;
  gap: 6px;
  justify-content: flex-end;
}
.btn-mini.ghost {
  border-color: var(--border);
  color: var(--text-2);
  letter-spacing: 0.06em;
}
.btn-mini.ghost:hover {
  background: var(--hover);
  color: var(--text);
  border-color: var(--border-strong);
}

/* Help card with best practices */
.help-card { padding: 0; }
.help-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
}
.help-cell {
  padding: 18px 20px;
  border-right: 1px solid rgba(255, 255, 255, 0.05);
  display: grid;
  grid-template-columns: 28px 1fr;
  gap: 12px;
  align-items: start;
}
.help-cell:last-child { border-right: 0; }
.help-icon {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-2);
}
.help-icon svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.5; }
.help-title {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 4px;
  letter-spacing: -0.005em;
}
.help-text p {
  font-size: 12px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.5;
}

/* ============================================================
   Modal scrim + modal box
   ============================================================ */
.modal-scrim {
  position: fixed;
  inset: 0;
  background: rgba(7, 8, 10, 0.72);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 32px;
  animation: scrimIn 200ms ease-out;
}
@keyframes scrimIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}
.modal {
  width: 100%;
  max-width: 520px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 64px);
  overflow: hidden;
  animation: modalIn 200ms cubic-bezier(0.2, 0, 0, 1);
}
@keyframes modalIn {
  from { opacity: 0; transform: translateY(8px) scale(0.99); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

.modal-head {
  padding: 16px 22px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-shrink: 0;
}
.modal-head .eyebrow { margin: 0 0 6px; }
.modal-head .eyebrow.warn { color: var(--warn); }
.modal-head .eyebrow.warn .dot { background: var(--warn); }
.modal-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 20px;
  letter-spacing: -0.018em;
  color: var(--text);
  margin: 0;
  line-height: 1.2;
}
.modal-x {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.modal-x:hover { background: var(--hover); color: var(--text); }
.modal-x svg { width: 14px; height: 14px; }

.modal-body {
  padding: 18px 22px 6px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.modal-body .field { gap: 8px; }
.modal-body .field-label { color: var(--text-2); }
.modal-body .field-label .hint {
  font-family: var(--font-sans);
  font-size: 11px;
  text-transform: none;
  letter-spacing: 0;
  color: var(--text-3);
}

/* Scope multi-select rows */
.scope-list {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  overflow: hidden;
}
.scope-opt {
  display: grid;
  grid-template-columns: 22px 1fr;
  gap: 10px;
  padding: 10px 14px;
  cursor: pointer;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  position: relative;
  transition: background 120ms;
}
.scope-opt:last-child { border-bottom: 0; }
.scope-opt:hover { background: rgba(255, 255, 255, 0.025); }
.scope-opt.checked { background: rgba(255, 255, 255, 0.03); }
.scope-opt input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}
.scope-cb {
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-top: 1px;
  color: var(--brand);
}
.scope-opt.checked .scope-cb {
  background: var(--text);
  border-color: var(--text);
}
.scope-opt.checked .scope-cb svg { width: 10px; height: 10px; }
.scope-body { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.scope-label-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.scope-warn {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  color: var(--neg);
}
.scope-help {
  font-size: 12px;
  color: var(--text-3);
  line-height: 1.4;
}

/* Expiration radio segments */
.exp-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
}
.exp-opt {
  display: grid;
  grid-template-columns: 16px 1fr;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  cursor: pointer;
  align-items: center;
  position: relative;
  transition: background 120ms, border-color 120ms;
}
.exp-opt:hover { background: rgba(255, 255, 255, 0.025); }
.exp-opt.checked {
  background: rgba(255, 255, 255, 0.03);
  border-color: var(--border-strong);
}
.exp-opt input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}
.exp-radio {
  width: 14px;
  height: 14px;
  border: 1px solid var(--border-strong);
  border-radius: 50%;
  background: var(--elevated);
  position: relative;
}
.exp-opt.checked .exp-radio { border-color: var(--text); }
.exp-opt.checked .exp-radio::after {
  content: '';
  position: absolute;
  inset: 3px;
  background: var(--text);
  border-radius: 50%;
}
.exp-body { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.exp-label {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  letter-spacing: -0.005em;
}
.exp-sub {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}

.modal-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 14px 22px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  background: var(--canvas);
  flex-shrink: 0;
}
.foot-hint {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.foot-hint svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  fill: none;
  stroke-width: 1.5;
}
.foot-hint.check { color: var(--pos); }
.foot-hint.err { color: var(--neg); }
.foot-hint.check strong { color: var(--text); font-family: var(--font-sans); font-weight: 500; }
.foot-actions { display: flex; gap: 8px; }
.modal-foot .btn { height: 36px; }
.modal-foot .btn.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
}

/* Warning banner (one-time reveal) */
.warn-banner {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px 16px;
  background: rgba(245, 158, 11, 0.10);
  border: 1px solid rgba(245, 158, 11, 0.30);
  border-radius: var(--radius-sm);
}
.warn-banner .warn-icon {
  width: 20px;
  height: 20px;
  background: var(--warn);
  color: var(--canvas);
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
  margin-top: 1px;
}
.warn-text {
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
}
.warn-text strong { color: var(--text); font-weight: 600; }

/* Key reveal */
.key-reveal {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: stretch;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  overflow: hidden;
}
.key-string {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text);
  letter-spacing: 0.02em;
  padding: 14px 16px;
  background: transparent;
  overflow-x: auto;
  white-space: nowrap;
  display: block;
}
.copy-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  background: var(--elevated);
  border: 0;
  border-left: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: background 120ms, color 120ms;
}
.copy-btn:hover {
  background: var(--hover);
  color: var(--text);
}
.copy-btn svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  fill: none;
  stroke-width: 1.5;
}

.key-meta {
  display: grid;
  grid-template-columns: 110px 1fr;
  gap: 8px 18px;
  margin: 4px 0 0;
  padding: 0;
}
.key-meta dt {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  align-self: center;
}
.key-meta dd {
  margin: 0;
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  word-break: break-all;
}
.meta-tags { display: flex; flex-wrap: wrap; gap: 4px; }

@media (max-width: 820px) {
  .help-grid { grid-template-columns: 1fr; }
  .help-cell { border-right: 0; border-bottom: 1px solid rgba(255, 255, 255, 0.05); }
  .help-cell:last-child { border-bottom: 0; }
  .section-head.api-head { flex-direction: column; align-items: stretch; }
  .exp-list { grid-template-columns: 1fr; }
  .scope-list { font-size: 13px; }
}
</style>
