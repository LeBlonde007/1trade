<script setup lang="ts">
/**
 * /enterprise/sso — SAML single sign-on, live on platform-core v1.8. Admins paste their identity
 * provider's metadata, list the email domains that sign in through it, and choose the policy
 * (just-in-time members and their role; enforce SSO for non-admins). The service-provider values the
 * IdP needs (entity id, ACS URL, metadata URL) come from the backend. SCIM is not built.
 */
definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Single sign-on — 1Trade' })

interface SSO {
  configured: boolean
  idp_entity_id?: string
  email_domains?: string[]
  verified_domains?: string[]
  pending_domains?: { domain: string; txt_name: string; txt_value: string }[]
  default_role?: string
  jit?: boolean
  enforce?: boolean
  updated_at?: string
  sp_entity_id: string
  acs_url: string
  sp_metadata_url: string
}

const { user } = useAuth()
const isAdmin = computed(() => (user.value?.roles ?? []).includes('admin'))
const sso = ref<SSO | null>(null)
const form = reactive({ metadata: '', domains: '', role: 'viewer', jit: true, enforce: false })
const busy = ref(false)
const error = ref('')
const notice = ref('')

/** load reads the configuration (admins only; others see the explanation). */
async function load() {
  if (!isAdmin.value) return
  try {
    sso.value = await $fetch<SSO>('/api/account/sso')
    if (sso.value.configured) {
      form.domains = (sso.value.email_domains ?? []).join(', ')
      form.role = sso.value.default_role ?? 'viewer'
      form.jit = sso.value.jit ?? true
      form.enforce = sso.value.enforce ?? false
    }
  } catch (e: any) {
    error.value = e?.data?.message || 'Could not load the configuration.'
  }
}
onMounted(load)

/** save stores the IdP metadata, domains and policy. */
async function save() {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    sso.value = await $fetch<SSO>('/api/account/sso', {
      method: 'PUT',
      body: {
        idp_metadata_xml: form.metadata,
        email_domains: form.domains.split(/[\s,]+/).filter(Boolean),
        default_role: form.role, jit: form.jit, enforce: form.enforce,
      },
    })
    form.metadata = ''
    notice.value = 'Single sign-on saved.'
  } catch (e: any) {
    error.value = e?.data?.message || 'Could not save.'
  } finally {
    busy.value = false
  }
}

/** verify asks the backend to check a domain's DNS TXT record. */
async function verify(domain: string) {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    sso.value = await $fetch<SSO>(`/api/account/sso/domains/${encodeURIComponent(domain)}/verify`, { method: 'POST' })
    notice.value = `${domain} verified — its members can now sign in with SSO.`
  } catch (e: any) {
    error.value = e?.data?.message || 'Could not verify that domain yet.'
  } finally {
    busy.value = false
  }
}

/** remove turns single sign-on off after confirmation. */
async function remove() {
  if (!globalThis.confirm('Turn off single sign-on? Members will sign in with passwords again.')) return
  busy.value = true
  try {
    await $fetch('/api/account/sso', { method: 'DELETE' })
    notice.value = 'Single sign-on turned off.'
    await load()
  } catch (e: any) {
    error.value = e?.data?.message || 'Could not turn it off.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="sso">
    <header class="head">
      <div>
        <div class="eyebrow">Security · Single sign-on</div>
        <h1 class="title">SAML single sign-on</h1>
        <p class="sub">Your team signs in through your identity provider (Okta, Microsoft Entra ID, Google Workspace, OneLogin…). Your IdP handles its own MFA.</p>
      </div>
      <span v-if="sso?.configured" class="tag ok">on</span>
      <span v-else class="tag">off</span>
    </header>

    <p v-if="!isAdmin" class="panel pad muted">Only admins can configure single sign-on. Ask an admin on your team.</p>
    <template v-else>
      <p v-if="error" class="banner neg" role="alert">{{ error }}</p>
      <p v-if="notice" class="banner pos" aria-live="polite">{{ notice }}</p>

      <section v-if="sso" class="panel">
        <header class="panel-h"><span class="panel-title">1 · Give your identity provider these values</span></header>
        <dl class="kv">
          <div><dt>Entity ID / audience</dt><dd class="mono">{{ sso.sp_entity_id }}</dd></div>
          <div><dt>ACS URL (HTTP-POST)</dt><dd class="mono">{{ sso.acs_url }}</dd></div>
          <div><dt>Name ID</dt><dd class="mono">emailAddress</dd></div>
          <div><dt>Service-provider metadata</dt><dd class="mono">{{ sso.configured ? sso.sp_metadata_url : 'available once saved' }}</dd></div>
        </dl>
      </section>

      <section class="panel">
        <header class="panel-h">
          <span class="panel-title">2 · Connect it</span>
          <span v-if="sso?.configured" class="panel-meta mono">IdP {{ sso.idp_entity_id }}</span>
        </header>
        <form class="form" @submit.prevent="save">
          <label>Identity-provider metadata (XML)
            <textarea v-model="form.metadata" rows="7" class="mono" :placeholder="sso?.configured ? 'Paste new metadata to update (required to save)' : '<EntityDescriptor …>'" required />
          </label>
          <label>Email domains that sign in with SSO
            <input v-model="form.domains" class="mono" placeholder="example.com, example.co.uk" required>
          </label>
          <div class="row2">
            <label>Role for new members
              <select v-model="form.role"><option value="viewer">viewer</option><option value="engineer">engineer</option><option value="billing">billing</option></select>
            </label>
            <div class="checks">
              <label class="chk"><input v-model="form.jit" type="checkbox"> Create members on first sign-in</label>
              <label class="chk"><input v-model="form.enforce" type="checkbox"> Require SSO (admins keep password sign-in as a fallback)</label>
            </div>
          </div>
          <div class="actions">
            <button type="submit" class="btn" :disabled="busy || !form.metadata.trim() || !form.domains.trim()">Save</button>
            <button v-if="sso?.configured" type="button" class="btn ghost" :disabled="busy" @click="remove">Turn off</button>
          </div>
        </form>
      </section>

      <section v-if="sso?.configured" class="panel">
        <header class="panel-h"><span class="panel-title">3 · Prove you own each domain</span></header>
        <p class="pad muted small">A domain signs people in only after you add this DNS TXT record. Until then nobody is routed to your IdP — and nobody else can take the domain from you.</p>
        <table class="dns">
          <tbody>
            <tr v-for="d in sso.verified_domains" :key="d"><td class="mono">{{ d }}</td><td colspan="2"><span class="tag ok">verified</span></td></tr>
            <tr v-for="d in sso.pending_domains" :key="d.domain">
              <td class="mono">{{ d.domain }}</td>
              <td class="mono small">TXT <strong>{{ d.txt_name }}</strong><br>{{ d.txt_value }}</td>
              <td><button type="button" class="btn" :disabled="busy" @click="verify(d.domain)">Verify</button></td>
            </tr>
          </tbody>
        </table>
      </section>

      <section class="panel pad muted small">
        Accounts are never merged: an email that already belongs to another 1Trade tenant cannot sign in through your IdP.
        SCIM provisioning is not built yet — use invitations or just-in-time creation.
      </section>
    </template>
  </div>
</template>

<style scoped>
.sso { background: var(--canvas); color: var(--text); padding: var(--sp-5); display: flex; flex-direction: column; gap: var(--sp-4); max-width: 1000px; margin: 0 auto; }
.head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 6px; line-height: 1; }
.sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0; max-width: 620px; }
.tag { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.1em; border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 2px 8px; color: var(--text-3); }
.tag.ok { color: var(--pos); border-color: var(--pos); }
.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden; }
.pad { padding: var(--sp-4); margin: 0; }
.panel-h { display: flex; justify-content: space-between; align-items: center; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.panel-title { font-size: var(--fs-sm); font-weight: 600; }
.panel-meta { font-size: var(--fs-xs); color: var(--text-3); }
.kv { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-3) var(--sp-4); padding: var(--sp-4); margin: 0; }
.kv dt { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); margin-bottom: 4px; }
.kv dd { margin: 0; font-size: var(--fs-xs); word-break: break-all; }
.form { display: flex; flex-direction: column; gap: var(--sp-3); padding: var(--sp-4); }
.form label { display: flex; flex-direction: column; gap: 4px; font-size: var(--fs-xs); color: var(--text-2); }
textarea, input, select { background: var(--canvas); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 7px 9px; color: var(--text); font-size: var(--fs-sm); }
.row2 { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-4); }
.checks { display: flex; flex-direction: column; gap: 8px; justify-content: flex-end; }
.form .chk { flex-direction: row; align-items: center; gap: 8px; font-size: var(--fs-sm); color: var(--text); }
.actions { display: flex; gap: var(--sp-2); }
.btn { background: var(--brand); color: var(--text-on-accent); border: 0; border-radius: var(--radius-sm); padding: 7px 14px; font-size: var(--fs-sm); font-weight: 600; cursor: pointer; }
.btn.ghost { background: transparent; color: var(--neg); border: 1px solid var(--neg); }
.btn:disabled { opacity: 0.5; cursor: default; }
.banner { margin: 0; padding: var(--sp-2) var(--sp-3); border-radius: var(--radius-sm); font-size: var(--fs-sm); }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.banner.pos { background: var(--pos-soft); color: var(--pos); }
.mono { font-family: var(--font-mono); }
.muted { color: var(--text-3); }
.small { font-size: var(--fs-xs); }
.dns { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.dns td { padding: var(--sp-2) var(--sp-4); border-top: 1px solid var(--border); vertical-align: middle; word-break: break-all; }
@media (max-width: 760px) { .kv, .row2 { grid-template-columns: 1fr; } }
</style>
