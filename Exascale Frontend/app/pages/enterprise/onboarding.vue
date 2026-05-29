<script setup lang="ts">
/**
 * /enterprise/onboarding — Enterprise customer onboarding workspace.
 *
 * Post-contract surface: customer has signed an MSA and is now being
 * walked through technical onboarding by Exascale's CSM team (or is
 * self-onboarding with sales support). Light, B2B-procurement
 * aesthetic — not a consumer signup flow.
 */

definePageMeta({ layout: false })
useHead({
  title: 'Onboarding · Walmart Inc. — Exascale Enterprise',
  htmlAttrs: { 'data-theme': 'light' },
})

// =====================================================
// Customer + commercial summary (sticky sidebar)
// =====================================================
const CUSTOMER = {
  name: 'Walmart Inc.',
  shortName: 'Walmart',
  enterpriseId: 'ENT-WMT-001',
  industry: 'Retail',
  region: 'Bentonville, AR · US',
  domain: 'walmart.com',
  contract: {
    type: '3-year enterprise agreement',
    effective: '2026-05-15',
    expires: '2029-05-14',
    commitment: '$5,000,000',
    cadence: 'annual',
    supportTier: 'Enterprise Premium',
    sla: '99.95%',
  },
  csm: {
    name: 'Sarah Lin',
    title: 'Customer Success · Enterprise',
    email: 'sarah.lin@exascale.com',
    phone: '+1 (212) 555-0199',
    slack: '#walmart-exascale',
    tz: 'New York · UTC−5',
  },
}

// =====================================================
// Checklist
// =====================================================
type Status = 'done' | 'in-progress' | 'pending'
interface ChecklistItem {
  id: string
  n: number
  status: Status
  title: string
  desc: string
  meta?: string         // e.g., completion timestamp
  actionLabel: string
  expanded?: boolean    // only one expanded for the mockup
}

const items = reactive<ChecklistItem[]>([
  {
    id: 'msa', n: 1, status: 'done',
    title: 'Master Services Agreement signed',
    desc: 'Counter-signed by Walmart Treasury and Exascale Legal. PDF and DocuSign envelope retained for audit.',
    meta: 'Completed 2026-05-15 · 14:42 ET',
    actionLabel: 'View document',
  },
  {
    id: 'sso', n: 2, status: 'in-progress',
    title: 'SAML SSO configuration',
    desc: 'Federate the venue against your identity provider. Domain auto-routing turns on once activated.',
    meta: 'Started 2026-05-19 · Sarah Lin assisting',
    actionLabel: 'Configure now',
    expanded: true,
  },
  {
    id: 'team', n: 3, status: 'pending',
    title: 'Add team members',
    desc: 'Invite the procurement, FinOps and ML-platform leads. Role assignments can be revised at any time.',
    actionLabel: 'Add team',
  },
  {
    id: 'subs', n: 4, status: 'pending',
    title: 'Set sub-account structure',
    desc: 'Organize the account by team, project or business unit. Spend and reports roll up at each level.',
    actionLabel: 'Configure',
  },
  {
    id: 'buy', n: 5, status: 'pending',
    title: 'Make first credit purchase',
    desc: 'Minimum first purchase $50,000 USD (or JPY equivalent). Wire is settled T+1; ACH is T+3.',
    actionLabel: 'Initiate wire',
  },
  {
    id: 'limits', n: 6, status: 'pending',
    title: 'Set spending limits and alerts',
    desc: 'Hard caps per sub-account, daily / monthly thresholds, and notification routing to your FinOps team.',
    actionLabel: 'Configure',
  },
  {
    id: 'compliance', n: 7, status: 'pending',
    title: 'Review compliance settings',
    desc: 'Audit log retention window, data residency region, and the export schedule for your SIEM pipeline.',
    actionLabel: 'Review',
  },
])

const doneCount = computed(() => items.filter(i => i.status === 'done').length)
const inProgressCount = computed(() => items.filter(i => i.status === 'in-progress').length)
const progressPct = computed(() => Math.round((doneCount.value / items.length) * 100))

function toggleExpand(id: string) {
  const it = items.find(x => x.id === id)
  if (!it) return
  it.expanded = !it.expanded
}

// =====================================================
// SSO config state (expanded under item 2)
// =====================================================
type SsoTab = 'upload' | 'manual'
const ssoTab = ref<SsoTab>('manual')

const sso = reactive({
  entityId: 'https://walmart.okta.com/exk7r4u9p1zL3aB0d2c8',
  ssoUrl:   'https://walmart.okta.com/app/exascale_prod/sso/saml',
  certificate:
    `-----BEGIN CERTIFICATE-----
MIIDqzCCApOgAwIBAgIGAYmvT8h2MA0GCSqGSIb3DQEBCwUAMIGSMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExDzANBgNVBAcMBkR1YmxpbjESMBAGA1UECgwJV0FMTUFSVDEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDFdhbG1hcnQgT2t0YTEcMBoGCSqG
SIb3DQEJARYNYWRtaW5Ad21ldC5jb20wHhcNMjYwMzE0MTQyMjA4WhcNMjcwMzE0MTQy
MzA4WjCBkjELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMQ8wDQYDVQQHDAZEdWJsaW4x
EjAQBgNVBAoMCVdBTE1BUlQxFDASBgNVBAsMC1NTT1Byb3ZpZGVyMRUwEwYDVQQDDAxX
−−−  (truncated for display · 3,072 bytes total) −−−`,
  emailDomain: 'walmart.com',
  attrs: {
    email:      'NameID',
    firstName:  'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname',
    lastName:   'http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname',
    department: 'http://schemas.xmlsoap.org/exascale/department',
    role:       'http://schemas.xmlsoap.org/exascale/role',
  },
})

type SsoTestState = 'idle' | 'testing' | 'passed' | 'failed'
const ssoTestState = ref<SsoTestState>('idle')
const ssoTestDetail = ref('')
let ssoTimer: ReturnType<typeof setTimeout> | null = null

function runSsoTest() {
  if (ssoTestState.value === 'testing') return
  ssoTestState.value = 'testing'
  ssoTestDetail.value = 'Posting test assertion to IdP…'
  if (ssoTimer) clearTimeout(ssoTimer)
  ssoTimer = setTimeout(() => {
    ssoTestState.value = 'passed'
    ssoTestDetail.value = 'sarah.lin@walmart.com · 248ms · NameID, givenname, surname, department received'
  }, 1800)
}
function activateSso() {
  if (ssoTestState.value !== 'passed') return
  // Flip item 2 to done; trigger 3 to in-progress
  const sItem = items.find(x => x.id === 'sso')
  if (sItem) {
    sItem.status = 'done'
    sItem.meta = 'Activated ' + new Date().toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: '2-digit' })
    sItem.expanded = false
  }
  const tItem = items.find(x => x.id === 'team')
  if (tItem && tItem.status === 'pending') tItem.status = 'in-progress'
}

onBeforeUnmount(() => { if (ssoTimer) clearTimeout(ssoTimer) })

// =====================================================
// Formatters
// =====================================================
function statusLabel(s: Status) {
  if (s === 'done')        return 'Complete'
  if (s === 'in-progress') return 'In progress'
  return 'Pending'
}
</script>

<template>
  <div class="ent-shell" data-theme="light">
    <!-- ============ Top bar ============ -->
    <header class="topbar">
      <div class="brand-row">
        <a href="#" class="brand">
          <span class="brand-mark" />
          EXASCALE
        </a>
        <span class="ent-pill">ENTERPRISE</span>
        <span class="brand-sep">·</span>
        <span class="cust">
          <span class="cust-mark" aria-hidden="true">W</span>
          <span class="cust-name">{{ CUSTOMER.name }}</span>
          <span class="cust-id">{{ CUSTOMER.enterpriseId }}</span>
        </span>
      </div>
      <div class="top-right">
        <div class="csm-widget">
          <div class="csm-meta">
            <span class="csm-label">Your CSM</span>
            <span class="csm-name">{{ CUSTOMER.csm.name }}</span>
          </div>
          <span class="csm-avatar" aria-hidden="true">SL</span>
          <div class="csm-actions">
            <a :href="`mailto:${CUSTOMER.csm.email}`" class="csm-btn" title="Email">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                <rect x="2" y="3.5" width="12" height="9" />
                <path d="M2 4l6 5 6-5" />
              </svg>
            </a>
            <a href="#" class="csm-btn" title="Slack">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round">
                <rect x="2.5" y="6" width="4" height="4" rx="1" />
                <rect x="9.5" y="6" width="4" height="4" rx="1" />
                <path d="M6 4.5L7 3 8 4.5 9 3 10 4.5M6 11.5L7 13 8 11.5 9 13 10 11.5" />
              </svg>
            </a>
            <a :href="`tel:${CUSTOMER.csm.phone}`" class="csm-btn primary">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                <path d="M3 3l2.5-.5L7 5 5.5 6c1 2 2.5 3.5 4.5 4.5L11 9l2.5 1.5L13 13c-5 .5-10-4.5-10-10z" />
              </svg>
              <span>Schedule call</span>
            </a>
          </div>
        </div>
      </div>
    </header>

    <!-- ============ Page ============ -->
    <main class="page">
      <div class="shell">
        <!-- ============ LEFT: Welcome + checklist ============ -->
        <div class="left">
          <!-- Welcome -->
          <section class="welcome">
            <div class="welcome-head">
              <div class="welcome-left">
                <div class="eyebrow">Enterprise onboarding · {{ CUSTOMER.enterpriseId }}</div>
                <h1 class="welcome-title">Welcome, Walmart team.</h1>
                <p class="welcome-sub">
                  Let's get your Exascale enterprise account set up. The seven steps below
                  are required before your first credit purchase clears. Your CSM can drive any
                  of them — or you can self-serve.
                </p>
              </div>
              <div class="welcome-right">
                <div class="progress-card">
                  <div class="pc-row">
                    <span class="pc-k">Onboarding progress</span>
                    <span class="pc-v">{{ doneCount }} / {{ items.length }}</span>
                  </div>
                  <div class="pc-bar">
                    <div class="pc-fill" :style="{ width: progressPct + '%' }" />
                  </div>
                  <div class="pc-foot">
                    <span class="pc-pct">{{ progressPct }}% complete</span>
                    <span class="pc-meta">{{ inProgressCount }} in progress</span>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <!-- Checklist -->
          <section class="checklist" aria-label="Onboarding checklist">
            <div
              v-for="it in items"
              :key="it.id"
              class="ck-item"
              :class="[it.status, { expanded: it.expanded }]"
            >
              <div class="ck-rail">
                <span class="ck-icon" aria-hidden="true">
                  <svg v-if="it.status === 'done'" viewBox="0 0 24 24" fill="none">
                    <circle cx="12" cy="12" r="11" fill="currentColor" />
                    <path d="M7 12.5l3.4 3.4L17 8.5" stroke="white" stroke-width="2.4" stroke-linecap="square" fill="none" />
                  </svg>
                  <svg v-else-if="it.status === 'in-progress'" viewBox="0 0 24 24" fill="none">
                    <circle cx="12" cy="12" r="10.5" stroke="currentColor" stroke-width="1.5" />
                    <path d="M12 1.5 A10.5 10.5 0 0 1 12 22.5 Z" fill="currentColor" />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" fill="none">
                    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 3" />
                  </svg>
                </span>
                <span v-if="it.n < items.length" class="ck-line" />
              </div>

              <div class="ck-body">
                <header class="ck-head">
                  <div class="ck-head-left">
                    <span class="ck-num">{{ String(it.n).padStart(2, '0') }}</span>
                    <h3 class="ck-title">{{ it.title }}</h3>
                    <span class="ck-status" :class="it.status">
                      <span class="dot" /> {{ statusLabel(it.status) }}
                    </span>
                  </div>
                  <div class="ck-head-right">
                    <button
                      v-if="it.id === 'sso'"
                      type="button"
                      class="btn secondary"
                      @click="toggleExpand(it.id)"
                    >
                      {{ it.expanded ? 'Hide configuration' : it.actionLabel }}
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                        <path d="M4 6l4 4 4-4" />
                      </svg>
                    </button>
                    <button
                      v-else
                      type="button"
                      :class="['btn', it.status === 'done' ? 'ghost' : it.status === 'in-progress' ? 'primary' : 'secondary']"
                    >
                      {{ it.status === 'done' ? it.actionLabel : it.actionLabel }}
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                        <path d="M3 8h10M9 4l4 4-4 4" />
                      </svg>
                    </button>
                  </div>
                </header>

                <p class="ck-desc">{{ it.desc }}</p>
                <div v-if="it.meta" class="ck-meta">— {{ it.meta }}</div>

                <!-- =================================================
                     SAML SSO expanded panel
                     ================================================= -->
                <div v-if="it.id === 'sso' && it.expanded" class="sso-panel">
                  <div class="sso-head">
                    <div class="seg">
                      <button
                        type="button"
                        :class="{ active: ssoTab === 'upload' }"
                        @click="ssoTab = 'upload'"
                      >Upload SAML metadata</button>
                      <button
                        type="button"
                        :class="{ active: ssoTab === 'manual' }"
                        @click="ssoTab = 'manual'"
                      >Enter manually</button>
                    </div>
                    <span class="sso-status" :class="ssoTestState">
                      <span class="dot" />
                      <template v-if="ssoTestState === 'idle'">Not yet tested</template>
                      <template v-else-if="ssoTestState === 'testing'">Testing connection…</template>
                      <template v-else-if="ssoTestState === 'passed'">Test passed</template>
                      <template v-else>Test failed</template>
                    </span>
                  </div>

                  <!-- ===== Upload state ===== -->
                  <div v-if="ssoTab === 'upload'" class="sso-upload">
                    <div class="upload-dropzone">
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square" class="up-ic">
                        <path d="M8 2v8M4 7l4-4 4 4M3 13h10" />
                      </svg>
                      <div class="up-title">Drop your <code>SAML metadata.xml</code> here</div>
                      <div class="up-sub">
                        Or <button type="button" class="up-link">browse files</button> · up to 256 KB · we will parse Entity ID, SSO URL and certificate
                      </div>
                    </div>
                    <div class="up-hint">
                      Most IdPs (Okta, Azure AD, Google Workspace, Ping) ship metadata XML. Paste a URL instead if your IdP exposes a hosted endpoint.
                    </div>
                  </div>

                  <!-- ===== Manual state ===== -->
                  <div v-else class="sso-manual">
                    <div class="sso-grid">
                      <div class="field col-12">
                        <label class="field-label">— IdP Entity ID</label>
                        <input v-model="sso.entityId" class="input mono-input" type="text" />
                      </div>
                      <div class="field col-12">
                        <label class="field-label">— SSO URL · SAML 2.0 endpoint</label>
                        <input v-model="sso.ssoUrl" class="input mono-input" type="text" />
                      </div>
                      <div class="field col-12">
                        <label class="field-label">
                          — X.509 certificate
                          <span class="hint">PEM-encoded · public key</span>
                        </label>
                        <textarea v-model="sso.certificate" class="textarea mono-input" rows="6" />
                      </div>

                      <div class="field col-6">
                        <label class="field-label">
                          — Email domain to auto-route
                          <span class="hint">Anyone signing in with this domain is sent through SAML</span>
                        </label>
                        <div class="input-affix">
                          <span class="affix-left">@</span>
                          <input v-model="sso.emailDomain" class="input mono-input pad-l" type="text" />
                        </div>
                      </div>
                      <div class="field col-6">
                        <label class="field-label">— Just-in-time provisioning</label>
                        <div class="seg-flat">
                          <button type="button" class="active">On</button>
                          <button type="button">Off</button>
                        </div>
                        <span class="field-hint">Auto-create new Exascale users on first sign-in.</span>
                      </div>
                    </div>

                    <div class="attr-section">
                      <div class="attr-head">— Attribute mapping</div>
                      <table class="attr-table">
                        <thead>
                          <tr>
                            <th class="left">Exascale field</th>
                            <th class="left">SAML attribute</th>
                            <th class="center">Required</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr>
                            <td class="left k">Email</td>
                            <td class="left mono">{{ sso.attrs.email }}</td>
                            <td class="center"><span class="req">Required</span></td>
                          </tr>
                          <tr>
                            <td class="left k">First name</td>
                            <td class="left mono">{{ sso.attrs.firstName }}</td>
                            <td class="center"><span class="req">Required</span></td>
                          </tr>
                          <tr>
                            <td class="left k">Last name</td>
                            <td class="left mono">{{ sso.attrs.lastName }}</td>
                            <td class="center"><span class="req">Required</span></td>
                          </tr>
                          <tr>
                            <td class="left k">Department</td>
                            <td class="left mono">{{ sso.attrs.department }}</td>
                            <td class="center"><span class="optional">Optional</span></td>
                          </tr>
                          <tr>
                            <td class="left k">Role</td>
                            <td class="left mono">{{ sso.attrs.role }}</td>
                            <td class="center"><span class="optional">Optional</span></td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <!-- ===== Footer ===== -->
                  <footer class="sso-foot">
                    <div class="sso-foot-left">
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                        <rect x="3.5" y="7" width="9" height="6.5" />
                        <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
                      </svg>
                      <span class="sso-foot-text">
                        ACS URL <code class="mono-pill">https://venue.exascale.com/sso/saml/ent-wmt-001/acs</code>
                        · <a href="#">Copy</a>
                      </span>
                    </div>
                    <div class="sso-foot-actions">
                      <button type="button" class="btn secondary" :disabled="ssoTestState === 'testing'" @click="runSsoTest">
                        <template v-if="ssoTestState === 'testing'">
                          <span class="spinner" />
                          Testing…
                        </template>
                        <template v-else>
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                            <path d="M3 8l3.2 3.2L13 5" />
                          </svg>
                          Test connection
                        </template>
                      </button>
                      <button
                        type="button"
                        class="btn primary"
                        :disabled="ssoTestState !== 'passed'"
                        @click="activateSso"
                      >
                        Activate SSO
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </footer>

                  <div v-if="ssoTestDetail" class="sso-detail" :class="ssoTestState">
                    {{ ssoTestDetail }}
                  </div>
                </div>
              </div>
            </div>
          </section>

          <!-- Footer note -->
          <footer class="page-foot">
            <span>Onboarding tracker · session ONB-WMT-2026Q2 · Sarah Lin (Exascale)</span>
            <a href="#">Download checklist as PDF →</a>
          </footer>
        </div>

        <!-- ============ RIGHT: Account summary ============ -->
        <aside class="right">
          <div class="summary">
            <div class="summary-head">
              <span class="cust-mark big" aria-hidden="true">W</span>
              <div>
                <div class="summary-cust">{{ CUSTOMER.name }}</div>
                <div class="summary-meta">{{ CUSTOMER.industry }} · {{ CUSTOMER.region }}</div>
              </div>
            </div>

            <dl class="summary-list">
              <dt>Enterprise ID</dt>
              <dd class="mono">{{ CUSTOMER.enterpriseId }}</dd>

              <dt>Contract type</dt>
              <dd>{{ CUSTOMER.contract.type }}</dd>

              <dt>Effective · expires</dt>
              <dd class="mono">{{ CUSTOMER.contract.effective }} · {{ CUSTOMER.contract.expires }}</dd>

              <dt>Annual commitment</dt>
              <dd class="big">{{ CUSTOMER.contract.commitment }}<span class="dim"> / {{ CUSTOMER.contract.cadence }}</span></dd>

              <dt>Support tier</dt>
              <dd>{{ CUSTOMER.contract.supportTier }}</dd>

              <dt>SLA</dt>
              <dd class="mono">{{ CUSTOMER.contract.sla }} uptime</dd>
            </dl>

            <div class="summary-divider" />

            <div class="summary-csm">
              <div class="csm-head">— Your Customer Success</div>
              <div class="csm-card">
                <span class="csm-avatar lg" aria-hidden="true">SL</span>
                <div class="csm-card-meta">
                  <span class="csm-card-name">{{ CUSTOMER.csm.name }}</span>
                  <span class="csm-card-title">{{ CUSTOMER.csm.title }}</span>
                </div>
              </div>
              <div class="csm-contacts">
                <a :href="`mailto:${CUSTOMER.csm.email}`" class="csm-contact">
                  <span class="cc-k">Email</span>
                  <span class="cc-v">{{ CUSTOMER.csm.email }}</span>
                </a>
                <a :href="`tel:${CUSTOMER.csm.phone}`" class="csm-contact">
                  <span class="cc-k">Direct</span>
                  <span class="cc-v">{{ CUSTOMER.csm.phone }}</span>
                </a>
                <a href="#" class="csm-contact">
                  <span class="cc-k">Slack</span>
                  <span class="cc-v">{{ CUSTOMER.csm.slack }}</span>
                </a>
                <div class="csm-contact static">
                  <span class="cc-k">Working hours</span>
                  <span class="cc-v">{{ CUSTOMER.csm.tz }}</span>
                </div>
              </div>
              <button type="button" class="btn primary csm-cta">
                Schedule onboarding session
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                  <path d="M3 8h10M9 4l4 4-4 4" />
                </svg>
              </button>
            </div>

            <div class="summary-divider" />

            <div class="summary-legal">
              <span class="ll-k">Documents</span>
              <a href="#" class="ll-link">Master Services Agreement</a>
              <a href="#" class="ll-link">Pricing Schedule v2.4</a>
              <a href="#" class="ll-link">Data Processing Addendum</a>
              <a href="#" class="ll-link">Order Form 2026-1</a>
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<style scoped>
.ent-shell {
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

/* ============================================================
   Top bar
   ============================================================ */
.topbar {
  height: 64px;
  background: var(--elevated);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 32px;
  gap: 24px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.brand-row {
  display: flex;
  align-items: center;
  gap: 16px;
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
  min-width: 0;
}
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
  letter-spacing: -0.02em;
  border-radius: var(--radius-sm);
}
.cust-mark.big {
  width: 32px;
  height: 32px;
  font-size: 17px;
}
.cust-name {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.015em;
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

.top-right { margin-left: auto; }
.csm-widget {
  display: grid;
  grid-template-columns: auto auto auto;
  align-items: center;
  gap: 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 6px 8px 6px 14px;
}
.csm-meta { display: flex; flex-direction: column; line-height: 1.1; }
.csm-label {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.csm-name {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  margin-top: 2px;
  letter-spacing: -0.005em;
}
.csm-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--text);
  color: var(--brand);
  display: grid;
  place-items: center;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 11px;
  letter-spacing: -0.005em;
}
.csm-avatar.lg {
  width: 40px;
  height: 40px;
  font-size: 14px;
}
.csm-actions {
  display: flex;
  gap: 4px;
  padding-left: 4px;
  border-left: 1px solid var(--border);
}
.csm-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 8px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-2);
  text-decoration: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 500;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.csm-btn:hover {
  background: var(--canvas);
  color: var(--text);
}
.csm-btn svg { width: 14px; height: 14px; }
.csm-btn.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
  padding: 0 12px;
  font-weight: 500;
}
.csm-btn.primary:hover { background: #222; border-color: #222; color: var(--elevated); }

/* ============================================================
   Page shell
   ============================================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 36px 32px 96px;
}
.shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 36px;
  align-items: start;
}

/* ============================================================
   Welcome
   ============================================================ */
.welcome {
  margin-bottom: 36px;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--border);
}
.welcome-head {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 32px;
  align-items: end;
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
.welcome-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 38px;
  letter-spacing: -0.025em;
  line-height: 1.05;
  margin: 0 0 8px;
  color: var(--text);
}
.welcome-sub {
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
  max-width: 620px;
}

.progress-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px 18px;
}
.pc-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8px;
}
.pc-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.pc-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.01em;
}
.pc-bar {
  height: 4px;
  background: var(--sunken);
  border-radius: 2px;
  overflow: hidden;
}
.pc-fill {
  height: 100%;
  background: var(--text);
  transition: width 400ms ease-out;
}
.pc-foot {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 8px;
}
.pc-pct { color: var(--text); font-weight: 500; }

/* ============================================================
   Checklist
   ============================================================ */
.checklist {
  display: flex;
  flex-direction: column;
}
.ck-item {
  display: grid;
  grid-template-columns: 32px 1fr;
  gap: 16px;
  padding-bottom: 20px;
}
.ck-item:last-child { padding-bottom: 0; }
.ck-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 2px;
}
.ck-icon {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.ck-icon svg { width: 24px; height: 24px; }
.ck-item.done .ck-icon         { color: var(--pos); }
.ck-item.in-progress .ck-icon  { color: var(--accent); }
.ck-item.pending .ck-icon      { color: var(--text-3); }
.ck-line {
  flex: 1;
  width: 1px;
  background: var(--border);
  margin: 6px 0 -4px;
  min-height: 16px;
}
.ck-item.done .ck-line { background: rgba(25, 195, 125, 0.30); }
.ck-item.in-progress .ck-line { background: rgba(74, 144, 226, 0.25); }

.ck-body {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px 20px;
  min-width: 0;
}
.ck-item.in-progress .ck-body {
  border-color: rgba(74, 144, 226, 0.30);
  box-shadow: 0 0 0 3px rgba(74, 144, 226, 0.06);
}

.ck-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 6px;
}
.ck-head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}
.ck-num {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}
.ck-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 17px;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
}
.ck-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.ck-status .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.ck-status.done {
  background: rgba(22, 163, 74, 0.10);
  color: var(--pos);
  border: 1px solid rgba(22, 163, 74, 0.25);
}
.ck-status.in-progress {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border: 1px solid rgba(74, 144, 226, 0.25);
}
.ck-status.pending {
  background: transparent;
  color: var(--text-3);
  border: 1px dashed var(--border-strong);
}

.ck-desc {
  color: var(--text-2);
  font-size: 13.5px;
  margin: 0 0 4px;
  line-height: 1.55;
  max-width: 60ch;
}
.ck-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 4px;
}

/* ============================================================
   Buttons
   ============================================================ */
.btn {
  height: 36px;
  padding: 0 14px;
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
  white-space: nowrap;
}
.btn svg { width: 14px; height: 14px; }
.btn.primary {
  background: var(--text);
  color: var(--elevated);
  border-color: var(--text);
}
.btn.primary:hover { background: #222; border-color: #222; }
.btn.primary:disabled {
  background: var(--sunken);
  color: var(--text-3);
  border-color: var(--border);
  cursor: not-allowed;
}
.btn.secondary {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover {
  background: var(--canvas);
  border-color: rgba(14, 14, 14, 0.32);
}
.btn.secondary:disabled {
  color: var(--text-3);
  border-color: var(--border);
  cursor: not-allowed;
}
.btn.ghost {
  background: transparent;
  color: var(--text-2);
  border-color: transparent;
}
.btn.ghost:hover { color: var(--text); background: var(--canvas); }

/* ============================================================
   SAML SSO expanded panel
   ============================================================ */
.sso-panel {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px dashed var(--bd-soft);
}
.sso-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
  gap: 16px;
  flex-wrap: wrap;
}
.seg {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.seg button {
  background: transparent;
  color: var(--text-2);
  border: 0;
  border-right: 1px solid var(--border);
  font-family: var(--font-sans);
  font-size: 12px;
  height: 30px;
  padding: 0 14px;
  cursor: pointer;
  font-weight: 500;
  letter-spacing: -0.005em;
  transition: background 120ms, color 120ms;
}
.seg button:last-child { border-right: 0; }
.seg button:hover { background: var(--canvas); color: var(--text); }
.seg button.active {
  background: var(--text);
  color: var(--elevated);
}
.seg-flat {
  display: inline-flex;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
  height: 38px;
}
.seg-flat button {
  background: var(--elevated);
  border: 0;
  border-right: 1px solid var(--border-strong);
  color: var(--text-2);
  padding: 0 14px;
  font-family: var(--font-sans);
  font-size: 13px;
  cursor: pointer;
  flex: 1;
}
.seg-flat button:last-child { border-right: 0; }
.seg-flat button.active { background: var(--text); color: var(--elevated); }

.sso-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}
.sso-status .dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.sso-status.idle {
  background: transparent;
  color: var(--text-3);
  border: 1px dashed var(--border-strong);
}
.sso-status.testing {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border: 1px solid rgba(74, 144, 226, 0.25);
}
.sso-status.passed {
  background: rgba(22, 163, 74, 0.10);
  color: var(--pos);
  border: 1px solid rgba(22, 163, 74, 0.25);
}
.sso-status.failed {
  background: rgba(220, 38, 38, 0.10);
  color: var(--neg);
  border: 1px solid rgba(220, 38, 38, 0.25);
}

.upload-dropzone {
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--canvas);
  padding: 36px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.up-ic { width: 28px; height: 28px; color: var(--text-3); }
.up-title {
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.up-title code {
  font-family: var(--font-mono);
  font-size: 12.5px;
  background: var(--elevated);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: 2px;
}
.up-sub {
  font-size: 12.5px;
  color: var(--text-2);
}
.up-link {
  background: transparent;
  border: 0;
  font: inherit;
  color: var(--text);
  text-decoration: underline;
  text-decoration-color: var(--text-3);
  cursor: pointer;
  padding: 0;
}
.up-link:hover { color: var(--text); text-decoration-color: var(--text); }
.up-hint {
  margin-top: 10px;
  font-size: 12px;
  color: var(--text-3);
  line-height: 1.5;
}

/* SSO Form grid */
.sso-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 14px;
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
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}
.field-label .hint {
  font-family: var(--font-sans);
  font-size: 11px;
  text-transform: none;
  letter-spacing: 0;
  color: var(--text-3);
  font-weight: 500;
}
.field-hint {
  font-size: 11.5px;
  color: var(--text-3);
}

.input, .textarea {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 0 12px;
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
  width: 100%;
  height: 38px;
  outline: none;
  transition: border-color 120ms, box-shadow 120ms;
}
.textarea {
  height: auto;
  padding: 10px 12px;
  resize: vertical;
  min-height: 120px;
  line-height: 1.45;
  white-space: pre;
}
.input:focus, .textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.mono-input {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12.5px;
  letter-spacing: 0.02em;
}
.input-affix {
  position: relative;
  display: flex;
  align-items: center;
}
.affix-left {
  position: absolute;
  left: 12px;
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-3);
}
.pad-l { padding-left: 26px; }

/* Attribute mapping */
.attr-section { margin-top: 22px; }
.attr-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-2);
  margin-bottom: 8px;
}
.attr-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}
.attr-table thead th {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
}
.attr-table th.left, .attr-table td.left { text-align: left; }
.attr-table th.center, .attr-table td.center { text-align: center; }
.attr-table tbody td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--bd-soft);
  color: var(--text);
}
.attr-table tbody tr:last-child td { border-bottom: 0; }
.attr-table td.k { font-weight: 500; color: var(--text); }
.attr-table td.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--text-2);
  font-size: 11.5px;
  letter-spacing: 0.02em;
  word-break: break-all;
}
.req {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.10em;
  color: var(--text-2);
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: 2px;
  text-transform: uppercase;
}
.optional {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.10em;
  color: var(--text-3);
  text-transform: uppercase;
}

/* SSO foot */
.sso-foot {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px dashed var(--bd-soft);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.sso-foot-left {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-3);
}
.sso-foot-left svg { width: 12px; height: 12px; }
.sso-foot-text {
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-3);
}
.sso-foot-text a { color: var(--text-2); text-decoration: none; border-bottom: 1px solid var(--border-strong); }
.sso-foot-text a:hover { color: var(--text); }
.mono-pill {
  font-family: var(--font-mono);
  font-size: 11.5px;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 2px 6px;
  border-radius: 2px;
  color: var(--text);
  letter-spacing: 0.02em;
  margin: 0 6px;
}
.sso-foot-actions { display: flex; gap: 8px; }

.sso-detail {
  margin-top: 12px;
  padding: 10px 12px;
  font-family: var(--font-mono);
  font-size: 11.5px;
  letter-spacing: 0.02em;
  border-radius: var(--radius-sm);
}
.sso-detail.testing {
  background: rgba(74, 144, 226, 0.06);
  color: var(--accent);
  border: 1px solid rgba(74, 144, 226, 0.20);
}
.sso-detail.passed {
  background: rgba(22, 163, 74, 0.06);
  color: var(--pos);
  border: 1px solid rgba(22, 163, 74, 0.20);
}

/* Spinner for "Testing…" */
@keyframes spin { to { transform: rotate(360deg); } }
.spinner {
  width: 12px;
  height: 12px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
  display: inline-block;
}

/* ============================================================
   Page footer
   ============================================================ */
.page-foot {
  margin-top: 28px;
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
.page-foot a {
  color: var(--text);
  text-decoration: none;
}
.page-foot a:hover { text-decoration: underline; }

/* ============================================================
   Right summary
   ============================================================ */
.right {
  position: sticky;
  top: 92px;
  align-self: start;
}
.summary {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 22px 22px 20px;
}
.summary-head {
  display: grid;
  grid-template-columns: 32px 1fr;
  gap: 12px;
  align-items: center;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 16px;
}
.summary-cust {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 16px;
  letter-spacing: -0.015em;
  color: var(--text);
}
.summary-meta {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 2px;
}

.summary-list {
  display: grid;
  grid-template-columns: 1fr;
  margin: 0;
  padding: 0;
}
.summary-list dt {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-top: 12px;
}
.summary-list dt:first-child { margin-top: 0; }
.summary-list dd {
  margin: 4px 0 0;
  font-family: var(--font-sans);
  font-size: 13px;
  color: var(--text);
}
.summary-list dd.mono {
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}
.summary-list dd.big {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin-top: 6px;
}

.summary-divider {
  height: 1px;
  background: var(--border);
  margin: 18px 0;
}

/* CSM card */
.summary-csm { display: flex; flex-direction: column; gap: 12px; }
.csm-head {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.csm-card {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: 12px;
  align-items: center;
}
.csm-card-meta { display: flex; flex-direction: column; line-height: 1.2; }
.csm-card-name {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 14px;
  color: var(--text);
  letter-spacing: -0.01em;
}
.csm-card-title {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 2px;
}

.csm-contacts {
  display: flex;
  flex-direction: column;
  gap: 1px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--canvas);
}
.csm-contact {
  display: grid;
  grid-template-columns: 90px 1fr;
  gap: 8px;
  align-items: center;
  padding: 8px 12px;
  text-decoration: none;
  border-bottom: 1px solid var(--bd-soft);
  transition: background 120ms;
}
.csm-contact:last-child { border-bottom: 0; }
.csm-contact:hover { background: var(--sunken); }
.csm-contact.static { cursor: default; }
.csm-contact.static:hover { background: var(--canvas); }
.cc-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.cc-v {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
  word-break: break-all;
}

.csm-cta {
  margin-top: 6px;
  height: 40px;
  width: 100%;
  justify-content: center;
  font-weight: 600;
}

/* Documents */
.summary-legal { display: flex; flex-direction: column; gap: 4px; }
.summary-legal .ll-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 4px;
}
.ll-link {
  font-family: var(--font-sans);
  font-size: 12.5px;
  color: var(--text-2);
  text-decoration: none;
  padding: 4px 0;
  border-bottom: 1px solid var(--bd-soft);
  display: flex;
  align-items: center;
  gap: 6px;
}
.ll-link:last-child { border-bottom: 0; }
.ll-link::after {
  content: '→';
  margin-left: auto;
  color: var(--text-3);
  font-size: 13px;
}
.ll-link:hover { color: var(--text); }
.ll-link:hover::after { color: var(--text); }

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1100px) {
  .shell { grid-template-columns: 1fr; }
  .right { position: static; }
  .welcome-head { grid-template-columns: 1fr; gap: 18px; }
}
@media (max-width: 720px) {
  .csm-widget { grid-template-columns: auto auto; padding-right: 4px; }
  .csm-actions { display: none; }
  .ck-item { grid-template-columns: 24px 1fr; gap: 12px; }
  .sso-foot { flex-direction: column; align-items: stretch; }
  .sso-foot-actions { justify-content: flex-end; }
  .field.col-6 { grid-column: span 12; }
}
</style>
