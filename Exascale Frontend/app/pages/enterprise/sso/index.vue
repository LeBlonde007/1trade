<script setup lang="ts">
/**
 * /enterprise/sso — SSO / SAML configuration wizard (H3).
 * Light enterprise admin. 3-step funnel + active-connection state.
 */
import {
  Check, ChevronRight, ChevronLeft, Download, Copy, ShieldCheck,
  AlertTriangle, ExternalLink, RefreshCw, Trash2, Loader2, XCircle
} from 'lucide-vue-next'

definePageMeta({ layout: 'app' })
useHead({ title: 'SSO & SAML — Enterprise — Exascale' })

interface IdP {
  id: string
  name: string
  vendor: string
  doc: string
  hint: string
}

const idps: IdP[] = [
  { id: 'okta',   name: 'Okta',              vendor: 'okta.com',          doc: 'docs/sso/okta',   hint: 'Most common — guided template available' },
  { id: 'azure',  name: 'Microsoft Entra ID', vendor: 'microsoft.com',     doc: 'docs/sso/entra',  hint: 'Formerly Azure AD' },
  { id: 'google', name: 'Google Workspace',  vendor: 'workspace.google.com', doc: 'docs/sso/google', hint: 'Provisioning via SCIM-lite' },
  { id: 'custom', name: 'Custom SAML 2.0',   vendor: 'Any SAML 2.0 IdP',  doc: 'docs/sso/saml',   hint: 'JumpCloud, OneLogin, Auth0, PingFederate, ADFS, etc.' },
]

// Wizard state ----------------------------------------
type WizardStep = 0 | 1 | 2 | 3
const step = ref<WizardStep>(0)
const selectedIdp = ref<IdP | null>(null)
const metadataMode = ref<'url' | 'xml'>('url')
const metadataUrl = ref('')
const metadataXml = ref('')

const testState = ref<'idle' | 'running' | 'success' | 'failure'>('idle')
const testEmail = ref('admin@deepminds.me')

// Mock attribute mapping returned after handshake
const attributeMap = ref([
  { saml: 'NameID',                       claude: 'user.email',     value: 'admin@deepminds.me',  status: 'matched' as const },
  { saml: 'http://.../claims/givenname',  claude: 'user.firstName', value: 'Mark',                status: 'matched' as const },
  { saml: 'http://.../claims/surname',    claude: 'user.lastName',  value: 'Sandoval',            status: 'matched' as const },
  { saml: 'http://.../claims/groups',     claude: 'user.groups',    value: '["Engineers","Finance-Admin"]', status: 'matched' as const },
  { saml: 'department',                   claude: 'user.department', value: 'Research',           status: 'matched' as const },
  { saml: 'employeeNumber',               claude: '— (unmapped)',   value: 'EMP-204871',          status: 'unmapped' as const },
])

// Active connection (set true to render the "configured" state) -------
const connectionActive = ref(false)
const activeConnection = ref({
  idp: 'Okta',
  domain: 'deepminds.me',
  signOnUrl: 'https://deepminds.okta.com/app/exa_prod/exkqq3of7Z82Gns0F297/sso/saml',
  configuredBy: 'mark.s@deepminds.me',
  configuredAt: '2026-04-12 09:21 UTC',
  scimEnabled: true,
  lastSync: '2026-05-24 02:00 UTC',
  usersProvisioned: 47,
  groupsProvisioned: 8,
})

// Helpers ---------------------------------------------
const spMetadata = {
  entityId: 'https://app.exascale.com/saml/metadata/org_8c4f2a1e',
  acsUrl:   'https://app.exascale.com/saml/acs/org_8c4f2a1e',
  sloUrl:   'https://app.exascale.com/saml/slo/org_8c4f2a1e',
  cert:     'MIIDpDCCAoygAwIBAgIBATANBgkqhkiG9w0BAQsFADBjMQsw…',
}

function pickIdp(idp: IdP) {
  selectedIdp.value = idp
}

function next() {
  if (step.value === 0 && !selectedIdp.value) return
  if (step.value === 1) {
    if (metadataMode.value === 'url' && !metadataUrl.value.trim()) return
    if (metadataMode.value === 'xml' && !metadataXml.value.trim()) return
  }
  if (step.value < 3) step.value = (step.value + 1) as WizardStep
}
function back() {
  if (step.value > 0) step.value = (step.value - 1) as WizardStep
}

const toasts = useToasts?.() ?? null
function copy(s: string, label = 'Copied') {
  navigator.clipboard?.writeText(s)
  toasts?.push({ tone: 'info', title: label, body: s.length > 60 ? s.slice(0, 60) + '…' : s })
}

function runTest() {
  testState.value = 'running'
  setTimeout(() => {
    testState.value = 'success'
  }, 2400)
}

function finish() {
  connectionActive.value = true
  step.value = 0
  toasts?.push({ tone: 'pos', title: 'SSO connection active', body: `${selectedIdp.value?.name ?? 'IdP'} now configured for deepminds.me` })
}

function disconnect() {
  connectionActive.value = false
  selectedIdp.value = null
  step.value = 0
}
</script>

<template>
  <div class="page">
    <!-- Page head ----------------------------------- -->
    <header class="page-head">
      <div>
        <div class="caps crumb">Enterprise · Security</div>
        <h1>SSO & SAML</h1>
        <p class="subtitle">
          Federate sign-in with your identity provider. Members in <span class="mono">deepminds.me</span> will be required to sign in via SSO once enabled.
        </p>
      </div>
      <div class="head-side">
        <a class="link-out" href="/docs/sso" target="_blank" rel="noopener">
          Setup docs <ExternalLink :size="12" :stroke-width="1.7" />
        </a>
      </div>
    </header>

    <!-- ACTIVE CONNECTION STATE --------------------- -->
    <section v-if="connectionActive" class="active-state">
      <div class="active-card">
        <div class="active-head">
          <div class="active-title">
            <ShieldCheck :size="18" :stroke-width="1.7" />
            <h2>SSO is active</h2>
            <span class="badge ok">connected</span>
          </div>
          <div class="active-actions">
            <button class="btn-ghost"><RefreshCw :size="13" :stroke-width="1.7" /> Re-test</button>
            <button class="btn-ghost danger" @click="disconnect"><Trash2 :size="13" :stroke-width="1.7" /> Disconnect</button>
          </div>
        </div>

        <dl class="conn-grid">
          <div>
            <dt class="caps">Identity provider</dt>
            <dd>{{ activeConnection.idp }}</dd>
          </div>
          <div>
            <dt class="caps">Email domain</dt>
            <dd class="mono">{{ activeConnection.domain }}</dd>
          </div>
          <div>
            <dt class="caps">Configured by</dt>
            <dd class="mono">{{ activeConnection.configuredBy }}</dd>
          </div>
          <div>
            <dt class="caps">Configured on</dt>
            <dd class="mono">{{ activeConnection.configuredAt }}</dd>
          </div>
          <div class="span-2">
            <dt class="caps">Sign-on URL</dt>
            <dd class="mono trunc">
              {{ activeConnection.signOnUrl }}
              <button class="copy-btn" @click="copy(activeConnection.signOnUrl, 'Sign-on URL copied')"><Copy :size="11" :stroke-width="1.7" /></button>
            </dd>
          </div>
        </dl>
      </div>

      <div class="scim-card">
        <div class="scim-head">
          <div>
            <h3>SCIM provisioning</h3>
            <p class="subtitle">Sync user and group membership nightly from your IdP. Removing a user in {{ activeConnection.idp }} revokes access within 60 seconds.</p>
          </div>
          <label class="toggle">
            <input type="checkbox" :checked="activeConnection.scimEnabled" @change="activeConnection.scimEnabled = ($event.target as HTMLInputElement).checked">
            <span class="track"><span class="knob" /></span>
          </label>
        </div>

        <div v-if="activeConnection.scimEnabled" class="scim-grid">
          <div>
            <span class="caps">Last sync</span>
            <span class="mono">{{ activeConnection.lastSync }}</span>
            <span class="badge ok small">success</span>
          </div>
          <div>
            <span class="caps">Users provisioned</span>
            <span class="mono">{{ activeConnection.usersProvisioned }}</span>
          </div>
          <div>
            <span class="caps">Groups provisioned</span>
            <span class="mono">{{ activeConnection.groupsProvisioned }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- WIZARD -------------------------------------- -->
    <section v-else class="wizard">
      <!-- Step indicator -->
      <ol class="stepper">
        <li :class="{ active: step === 0, done: step > 0 }">
          <span class="step-num">
            <Check v-if="step > 0" :size="13" :stroke-width="2" />
            <span v-else>1</span>
          </span>
          <div>
            <div class="step-label">Choose IdP</div>
            <div class="step-sub">{{ selectedIdp ? selectedIdp.name : 'Pick your identity provider' }}</div>
          </div>
        </li>
        <li class="step-connector" :class="{ done: step > 0 }" />
        <li :class="{ active: step === 1, done: step > 1 }">
          <span class="step-num">
            <Check v-if="step > 1" :size="13" :stroke-width="2" />
            <span v-else>2</span>
          </span>
          <div>
            <div class="step-label">Exchange metadata</div>
            <div class="step-sub">Download Exascale SP · upload IdP</div>
          </div>
        </li>
        <li class="step-connector" :class="{ done: step > 1 }" />
        <li :class="{ active: step === 2, done: step > 2 }">
          <span class="step-num">
            <Check v-if="step > 2" :size="13" :stroke-width="2" />
            <span v-else>3</span>
          </span>
          <div>
            <div class="step-label">Test sign-in</div>
            <div class="step-sub">Verify with a real handshake</div>
          </div>
        </li>
      </ol>

      <!-- Step content -->
      <div class="step-pane">

        <!-- STEP 0 — Pick IdP ----------------------- -->
        <div v-if="step === 0" class="pane">
          <h2 class="pane-title">Choose your identity provider</h2>
          <p class="pane-sub">
            We support Okta, Microsoft Entra ID, Google Workspace, and any standards-compliant SAML 2.0 IdP.
            Pick the one your organization uses for sign-in today.
          </p>

          <ul class="idp-list">
            <li
              v-for="idp in idps"
              :key="idp.id"
              :class="{ selected: selectedIdp?.id === idp.id }"
              @click="pickIdp(idp)"
            >
              <div class="idp-logo" :data-vendor="idp.id">
                <span>{{ idp.name[0] }}</span>
              </div>
              <div class="idp-text">
                <div class="idp-name">{{ idp.name }}</div>
                <div class="idp-hint">{{ idp.hint }}</div>
                <div class="idp-doc mono">{{ idp.vendor }}</div>
              </div>
              <div class="idp-check">
                <span class="radio" :class="{ on: selectedIdp?.id === idp.id }">
                  <span v-if="selectedIdp?.id === idp.id" class="dot" />
                </span>
              </div>
            </li>
          </ul>
        </div>

        <!-- STEP 1 — Exchange metadata -------------- -->
        <div v-if="step === 1" class="pane">
          <h2 class="pane-title">Exchange metadata with {{ selectedIdp?.name }}</h2>
          <p class="pane-sub">
            Two-way trust. First, configure {{ selectedIdp?.name }} using Exascale's Service Provider metadata.
            Then paste {{ selectedIdp?.name }}'s metadata back here so we can verify signed assertions.
          </p>

          <!-- SP metadata download -->
          <section class="meta-section">
            <header class="meta-head">
              <div>
                <span class="caps">Step 2a</span>
                <h3>Configure {{ selectedIdp?.name }} with our SP metadata</h3>
              </div>
              <button class="btn-primary"><Download :size="13" :stroke-width="1.7" /> Download XML</button>
            </header>

            <dl class="kv">
              <div>
                <dt>SP Entity ID</dt>
                <dd class="mono">
                  {{ spMetadata.entityId }}
                  <button class="copy-btn" @click="copy(spMetadata.entityId, 'Entity ID copied')"><Copy :size="11" :stroke-width="1.7" /></button>
                </dd>
              </div>
              <div>
                <dt>ACS (Reply) URL</dt>
                <dd class="mono">
                  {{ spMetadata.acsUrl }}
                  <button class="copy-btn" @click="copy(spMetadata.acsUrl, 'ACS URL copied')"><Copy :size="11" :stroke-width="1.7" /></button>
                </dd>
              </div>
              <div>
                <dt>Single Logout URL</dt>
                <dd class="mono">
                  {{ spMetadata.sloUrl }}
                  <button class="copy-btn" @click="copy(spMetadata.sloUrl, 'SLO URL copied')"><Copy :size="11" :stroke-width="1.7" /></button>
                </dd>
              </div>
              <div>
                <dt>Signing certificate</dt>
                <dd class="mono cert">{{ spMetadata.cert }}</dd>
              </div>
            </dl>
          </section>

          <!-- IdP metadata upload -->
          <section class="meta-section">
            <header class="meta-head">
              <div>
                <span class="caps">Step 2b</span>
                <h3>Provide {{ selectedIdp?.name }}'s metadata</h3>
              </div>
              <div class="seg">
                <button :class="{ active: metadataMode === 'url' }" @click="metadataMode = 'url'">Metadata URL</button>
                <button :class="{ active: metadataMode === 'xml' }" @click="metadataMode = 'xml'">Paste XML</button>
              </div>
            </header>

            <div v-if="metadataMode === 'url'" class="field">
              <label class="caps">IdP metadata URL</label>
              <input
                v-model="metadataUrl"
                type="url"
                placeholder="https://your-tenant.okta.com/app/exkqq3of7Z82Gns0F297/sso/saml/metadata"
              >
              <p class="field-hint">We'll fetch this URL daily to pick up signing-key rotations.</p>
            </div>

            <div v-else class="field">
              <label class="caps">IdP metadata XML</label>
              <textarea
                v-model="metadataXml"
                rows="8"
                placeholder="<?xml version='1.0' encoding='UTF-8'?>&#10;<EntityDescriptor ..."
              />
              <p class="field-hint">Or <a href="#">upload a file</a>. We will validate signature, entityID, and SSO endpoints before you can continue.</p>
            </div>
          </section>
        </div>

        <!-- STEP 2 — Test sign-in -------------------- -->
        <div v-if="step === 2" class="pane">
          <h2 class="pane-title">Test the SAML handshake</h2>
          <p class="pane-sub">
            We'll run a real sign-in against {{ selectedIdp?.name }} using the configuration above
            and show you exactly which attributes come back. No real users are provisioned until you confirm.
          </p>

          <section class="test-section">
            <div class="test-row">
              <label>
                <span class="caps">Test as user</span>
                <input v-model="testEmail" type="email" placeholder="someone@deepminds.me">
              </label>
              <button
                class="btn-primary lg"
                :disabled="testState === 'running'"
                @click="runTest"
              >
                <Loader2 v-if="testState === 'running'" :size="14" :stroke-width="1.7" class="spin" />
                <ExternalLink v-else :size="14" :stroke-width="1.7" />
                {{ testState === 'running' ? 'Running handshake…' : testState === 'success' ? 'Run again' : 'Open IdP sign-in' }}
              </button>
            </div>

            <!-- Handshake stages -->
            <ol v-if="testState !== 'idle'" class="stages">
              <li :class="{ done: testState !== 'running' || true, active: testState === 'running' }">
                <span class="stage-dot" />
                <span>SP-initiated AuthnRequest sent to {{ selectedIdp?.name }}</span>
                <Check :size="13" :stroke-width="2" />
              </li>
              <li :class="{ done: testState === 'success' }">
                <span class="stage-dot" />
                <span>SAMLResponse received, signature verified</span>
                <Check v-if="testState === 'success'" :size="13" :stroke-width="2" />
                <Loader2 v-else-if="testState === 'running'" :size="13" :stroke-width="1.7" class="spin" />
              </li>
              <li :class="{ done: testState === 'success' }">
                <span class="stage-dot" />
                <span>Assertion conditions valid, audience and recipient match</span>
                <Check v-if="testState === 'success'" :size="13" :stroke-width="2" />
              </li>
              <li :class="{ done: testState === 'success' }">
                <span class="stage-dot" />
                <span>Attribute mapping preview</span>
                <Check v-if="testState === 'success'" :size="13" :stroke-width="2" />
              </li>
            </ol>

            <!-- Attribute mapping table -->
            <div v-if="testState === 'success'" class="attr-result">
              <header class="result-head">
                <div class="result-status ok">
                  <ShieldCheck :size="16" :stroke-width="1.7" />
                  <span>Handshake successful</span>
                </div>
                <span class="mono ts">2026-05-24 14:08:22 UTC · response 312 ms</span>
              </header>

              <table class="attr-table">
                <thead>
                  <tr>
                    <th>SAML attribute</th>
                    <th>Maps to</th>
                    <th>Value returned</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="a in attributeMap" :key="a.saml">
                    <td class="mono">{{ a.saml }}</td>
                    <td class="mono">{{ a.claude }}</td>
                    <td class="mono trunc">{{ a.value }}</td>
                    <td>
                      <span class="badge" :class="a.status">
                        {{ a.status === 'matched' ? 'matched' : 'unmapped' }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>

              <div class="unmapped-note">
                <AlertTriangle :size="13" :stroke-width="1.7" />
                <span>1 attribute is not mapped to an Exascale field. You can <a href="#">add a custom mapping</a> after enabling.</span>
              </div>
            </div>

            <div v-if="testState === 'failure'" class="failure-box">
              <XCircle :size="16" :stroke-width="1.7" />
              <div>
                <strong>Handshake failed</strong>
                <p class="mono err">SAMLResponse signature invalid — signing certificate fingerprint did not match IdP metadata.</p>
                <a href="#">View full SAML trace</a>
              </div>
            </div>
          </section>
        </div>

        <!-- STEP 3 — Final confirm ------------------ -->
        <div v-if="step === 3" class="pane">
          <h2 class="pane-title">Enable SSO for deepminds.me</h2>
          <p class="pane-sub">
            Once you enable, all 47 members of <span class="mono">deepminds.me</span> will be required to sign in via {{ selectedIdp?.name }}.
            Password sign-in will be disabled. Make sure you can complete an end-to-end SSO sign-in <em>before</em> you sign out of your current session.
          </p>

          <div class="warn-box">
            <AlertTriangle :size="16" :stroke-width="1.7" />
            <div>
              <strong>Break-glass account</strong>
              <p>We'll generate a one-time recovery code for <span class="mono">mark.s@deepminds.me</span> that bypasses SSO. Store it in a password manager — you'll only see it once.</p>
            </div>
          </div>

          <ul class="confirm-list">
            <li>
              <span>Identity provider</span>
              <span class="mono">{{ selectedIdp?.name }}</span>
            </li>
            <li>
              <span>Email domain enforced</span>
              <span class="mono">deepminds.me</span>
            </li>
            <li>
              <span>Members impacted</span>
              <span class="mono">47</span>
            </li>
            <li>
              <span>SCIM provisioning</span>
              <span class="mono">Enabled (daily sync 02:00 UTC)</span>
            </li>
            <li>
              <span>Effective</span>
              <span class="mono">Immediately on confirm</span>
            </li>
          </ul>
        </div>
      </div>

      <!-- Wizard footer -->
      <footer class="wizard-foot">
        <button class="btn-ghost" :disabled="step === 0" @click="back">
          <ChevronLeft :size="13" :stroke-width="1.7" /> Back
        </button>
        <div class="foot-meta">
          Step {{ step + 1 }} of 4
        </div>
        <button
          v-if="step < 2"
          class="btn-primary"
          :disabled="(step === 0 && !selectedIdp) || (step === 1 && metadataMode === 'url' && !metadataUrl) || (step === 1 && metadataMode === 'xml' && !metadataXml)"
          @click="next"
        >
          Continue <ChevronRight :size="13" :stroke-width="1.7" />
        </button>
        <button v-else-if="step === 2" class="btn-primary" :disabled="testState !== 'success'" @click="next">
          Review & enable <ChevronRight :size="13" :stroke-width="1.7" />
        </button>
        <button v-else class="btn-primary danger" @click="finish">
          <ShieldCheck :size="13" :stroke-width="1.7" /> Enable SSO for organization
        </button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.page {
  background: var(--surface-canvas, #F8F7F4);
  color: var(--text-primary, #0A0B0E);
  min-height: 100vh;
  font-family: 'Inter', system-ui, sans-serif;
  font-feature-settings: 'tnum';
  padding: 32px 32px 96px;
}

.mono { font-family: 'JetBrains Mono', ui-monospace, monospace; font-variant-numeric: tabular-nums; }

.caps {
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 600;
  color: var(--text-tertiary, #9A9A95);
}

/* Page head ---------------------------------------- */
.page-head {
  max-width: 880px;
  margin: 0 auto 28px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
}
.crumb { margin-bottom: 8px; }
.page-head h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.subtitle {
  margin: 8px 0 0;
  color: var(--text-secondary, #5F5F5C);
  font-size: 14px;
  line-height: 1.55;
  max-width: 620px;
}
.head-side .link-out {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-secondary);
  text-decoration: none;
}
.head-side .link-out:hover { color: var(--text-primary); }

/* Active state ------------------------------------- */
.active-state {
  max-width: 880px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.active-card,
.scim-card {
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
  padding: 20px 24px;
}
.active-card { border-left: 3px solid #19C37D; }

.active-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}
.active-title {
  display: flex;
  align-items: center;
  gap: 10px;
}
.active-title svg { color: #19C37D; }
.active-title h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.badge {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: 2px;
  background: rgba(0,0,0,0.06);
  color: var(--text-secondary);
}
.badge.ok       { background: rgba(25,195,125,0.12); color: #128050; }
.badge.matched  { background: rgba(25,195,125,0.12); color: #128050; }
.badge.unmapped { background: rgba(245,158,11,0.12); color: #B97A06; }
.badge.small    { font-size: 9px; padding: 1px 6px; }

.active-actions { display: flex; gap: 8px; }

.conn-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px 32px;
  margin: 0;
}
.conn-grid > div > dt {
  margin: 0 0 4px;
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 600;
  color: var(--text-tertiary);
}
.conn-grid > div > dd {
  margin: 0;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.span-2 { grid-column: span 2; }
.trunc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.scim-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  margin-bottom: 12px;
}
.scim-head h3 { margin: 0; font-size: 16px; font-weight: 600; }
.scim-head p { margin: 4px 0 0; font-size: 13px; }
.scim-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-top: 12px;
  padding-top: 16px;
  border-top: 1px solid rgba(0,0,0,0.06);
}
.scim-grid > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.scim-grid .mono { font-size: 16px; }

/* Toggle */
.toggle { display: inline-flex; cursor: pointer; }
.toggle input { display: none; }
.toggle .track {
  width: 36px;
  height: 20px;
  background: #D5D2CC;
  border-radius: 10px;
  position: relative;
  transition: background 120ms ease;
}
.toggle .knob {
  position: absolute;
  width: 16px;
  height: 16px;
  background: #FFFFFF;
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: left 120ms ease;
  box-shadow: 0 1px 2px rgba(0,0,0,0.2);
}
.toggle input:checked + .track { background: #19C37D; }
.toggle input:checked + .track .knob { left: 18px; }

/* Wizard ------------------------------------------- */
.wizard {
  max-width: 880px;
  margin: 0 auto;
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
  overflow: hidden;
}

.stepper {
  display: flex;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid rgba(0,0,0,0.08);
  list-style: none;
  margin: 0;
  gap: 0;
}
.stepper > li:not(.step-connector) {
  display: flex;
  align-items: center;
  gap: 10px;
}
.stepper .step-num {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(0,0,0,0.06);
  color: var(--text-tertiary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}
.stepper > li.active .step-num {
  background: var(--text-primary);
  color: #F8F7F4;
}
.stepper > li.done .step-num {
  background: #19C37D;
  color: #FFFFFF;
}
.step-label {
  font-size: 13px;
  font-weight: 500;
}
.step-sub {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 1px;
}
.step-connector {
  flex: 1;
  height: 1px;
  background: rgba(0,0,0,0.08);
  margin: 0 16px;
}
.step-connector.done { background: #19C37D; }

.step-pane { padding: 28px 32px; }
.pane-title { margin: 0 0 6px; font-size: 20px; font-weight: 600; letter-spacing: -0.01em; }
.pane-sub { margin: 0 0 24px; color: var(--text-secondary); font-size: 14px; line-height: 1.6; max-width: 640px; }

/* IdP list */
.idp-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
@media (max-width: 720px) { .idp-list { grid-template-columns: 1fr; } }
.idp-list li {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 4px;
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
  background: #FFFFFF;
}
.idp-list li:hover { border-color: rgba(0,0,0,0.16); }
.idp-list li.selected {
  border-color: var(--text-primary);
  background: rgba(74,144,226,0.04);
}
.idp-logo {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  background: #F0EFEC;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-weight: 700;
  font-size: 18px;
  color: var(--text-primary);
}
.idp-logo[data-vendor='okta']   { background: #007DC1; color: #FFF; }
.idp-logo[data-vendor='azure']  { background: #00A4EF; color: #FFF; }
.idp-logo[data-vendor='google'] { background: #4285F4; color: #FFF; }
.idp-logo[data-vendor='custom'] { background: #0A0B0E; color: #C8F25C; }
.idp-text { flex: 1; min-width: 0; }
.idp-name { font-size: 15px; font-weight: 600; margin-bottom: 2px; }
.idp-hint { font-size: 12px; color: var(--text-secondary); }
.idp-doc  { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }
.radio {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 1.5px solid rgba(0,0,0,0.24);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.radio.on { border-color: var(--text-primary); }
.radio .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-primary); }

/* Meta section */
.meta-section {
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 4px;
  padding: 20px;
  margin-bottom: 16px;
  background: #FCFBF8;
}
.meta-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 16px;
  gap: 16px;
  flex-wrap: wrap;
}
.meta-head h3 { margin: 4px 0 0; font-size: 15px; font-weight: 600; }

.kv { margin: 0; }
.kv > div { padding: 8px 0; border-bottom: 1px dashed rgba(0,0,0,0.06); display: grid; grid-template-columns: 200px 1fr; gap: 16px; align-items: center; }
.kv > div:last-child { border-bottom: none; }
.kv dt { margin: 0; font-size: 11px; color: var(--text-secondary); font-weight: 500; }
.kv dd { margin: 0; font-size: 12px; display: flex; align-items: center; gap: 6px; word-break: break-all; }
.kv dd.cert { font-size: 11px; color: var(--text-tertiary); }

.copy-btn {
  background: none;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 2px;
  padding: 2px 4px;
  color: var(--text-tertiary);
  cursor: pointer;
  display: inline-flex;
}
.copy-btn:hover { color: var(--text-primary); border-color: rgba(0,0,0,0.16); }

/* Seg toggle */
.seg {
  display: inline-flex;
  border: 1px solid rgba(0,0,0,0.16);
  border-radius: 4px;
  overflow: hidden;
}
.seg button {
  background: #FFFFFF;
  border: none;
  height: 28px;
  padding: 0 12px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  font-family: inherit;
}
.seg button.active { background: var(--text-primary); color: #F8F7F4; }

.field { display: flex; flex-direction: column; gap: 6px; }
.field label { font-size: 10px; letter-spacing: 0.08em; text-transform: uppercase; font-weight: 600; color: var(--text-tertiary); }
.field input,
.field textarea {
  border: 1px solid rgba(0,0,0,0.16);
  border-radius: 4px;
  padding: 10px 12px;
  font-size: 13px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  background: #FFFFFF;
  color: var(--text-primary);
  width: 100%;
}
.field input:focus,
.field textarea:focus {
  outline: none;
  border-color: #4A90E2;
  box-shadow: 0 0 0 3px rgba(74,144,226,0.15);
}
.field textarea { resize: vertical; min-height: 140px; line-height: 1.5; }
.field-hint { margin: 0; font-size: 12px; color: var(--text-secondary); }
.field-hint a { color: var(--text-primary); }

/* Test section ------------------------------------- */
.test-section { display: flex; flex-direction: column; gap: 20px; }
.test-row {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}
.test-row label { flex: 1; min-width: 280px; }
.test-row label span { display: block; margin-bottom: 6px; }
.test-row input {
  width: 100%;
  border: 1px solid rgba(0,0,0,0.16);
  border-radius: 4px;
  padding: 0 12px;
  height: 36px;
  font-size: 13px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

.stages {
  list-style: none;
  padding: 16px 20px;
  margin: 0;
  background: #FCFBF8;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 4px;
}
.stages li {
  display: grid;
  grid-template-columns: 20px 1fr 16px;
  gap: 12px;
  align-items: center;
  padding: 6px 0;
  font-size: 13px;
  color: var(--text-secondary);
}
.stages li.done { color: var(--text-primary); }
.stage-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(0,0,0,0.16);
  margin: 0 6px;
}
.stages li.done .stage-dot { background: #19C37D; }
.stages li.active .stage-dot { background: #4A90E2; animation: pulse 1.4s infinite; }
@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(74,144,226,0.4); }
  50%      { box-shadow: 0 0 0 6px rgba(74,144,226,0); }
}
.stages svg { color: #19C37D; justify-self: end; }

.attr-result {
  border: 1px solid rgba(25,195,125,0.32);
  border-radius: 4px;
  overflow: hidden;
}
.result-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(25,195,125,0.08);
  border-bottom: 1px solid rgba(25,195,125,0.16);
}
.result-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 13px;
  color: #128050;
}
.ts { font-size: 11px; color: var(--text-tertiary); }

.attr-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.attr-table th {
  text-align: left;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  padding: 8px 12px;
  background: #FFFFFF;
  border-bottom: 1px solid rgba(0,0,0,0.06);
}
.attr-table td {
  padding: 10px 12px;
  border-bottom: 1px solid rgba(0,0,0,0.04);
  background: #FFFFFF;
}
.attr-table tr:last-child td { border-bottom: none; }
.attr-table td.trunc { max-width: 280px; }

.unmapped-note {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: rgba(245,158,11,0.08);
  border-top: 1px solid rgba(245,158,11,0.16);
  font-size: 12px;
  color: var(--text-secondary);
}
.unmapped-note svg { color: #F59E0B; flex-shrink: 0; }
.unmapped-note a { color: var(--text-primary); }

.failure-box {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  background: rgba(239,68,68,0.06);
  border: 1px solid rgba(239,68,68,0.32);
  border-radius: 4px;
}
.failure-box svg { color: #EF4444; flex-shrink: 0; margin-top: 2px; }
.failure-box strong { display: block; margin-bottom: 4px; font-size: 13px; }
.failure-box p { margin: 0; font-size: 12px; color: var(--text-secondary); }
.failure-box .err { color: #B82929; }
.failure-box a { font-size: 12px; color: var(--text-primary); }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Confirm list */
.confirm-list {
  list-style: none;
  padding: 0;
  margin: 16px 0 0;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 4px;
}
.confirm-list li {
  display: flex;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.04);
  font-size: 13px;
}
.confirm-list li:last-child { border-bottom: none; }
.confirm-list li > :first-child { color: var(--text-secondary); }

.warn-box {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  background: rgba(245,158,11,0.08);
  border: 1px solid rgba(245,158,11,0.24);
  border-radius: 4px;
  margin: 16px 0 24px;
}
.warn-box svg { color: #F59E0B; flex-shrink: 0; margin-top: 2px; }
.warn-box strong { display: block; margin-bottom: 4px; font-size: 13px; }
.warn-box p { margin: 0; font-size: 13px; color: var(--text-secondary); line-height: 1.5; }

/* Wizard footer ------------------------------------ */
.wizard-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-top: 1px solid rgba(0,0,0,0.08);
  background: #FCFBF8;
}
.foot-meta { font-size: 12px; color: var(--text-tertiary); }

.btn-primary,
.btn-ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 500;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  border: 1px solid rgba(0,0,0,0.08);
  background: #FFFFFF;
  color: var(--text-primary);
  transition: background 100ms ease, border-color 100ms ease;
}
.btn-primary {
  background: var(--text-primary);
  color: #F8F7F4;
  border-color: var(--text-primary);
}
.btn-primary:hover { background: #1C1F26; }
.btn-primary:disabled {
  background: rgba(0,0,0,0.12);
  border-color: transparent;
  color: var(--text-tertiary);
  cursor: not-allowed;
}
.btn-primary.danger {
  background: #EF4444;
  border-color: #EF4444;
}
.btn-primary.danger:hover { background: #DC2A2A; }
.btn-primary.lg { height: 38px; padding: 0 18px; font-size: 13px; }

.btn-ghost:hover { border-color: rgba(0,0,0,0.16); }
.btn-ghost:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-ghost.danger { color: #EF4444; }
.btn-ghost.danger:hover { border-color: rgba(239,68,68,0.32); background: rgba(239,68,68,0.06); }
</style>
