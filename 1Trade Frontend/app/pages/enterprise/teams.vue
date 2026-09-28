<script setup lang="ts">
/**
 * /enterprise/teams — Team & access, live on platform-core v1.6: members (roles, sub-account, remove),
 * invitations (invite by email with a role and sub-account, revoke), and sub-accounts (create; fund
 * from the main balance or return credits through the ledger). Admin-only actions are shown to admins;
 * everything else is read-only. SCIM is not built and says so.
 */
import type { Invite, Member, SubAccount } from '~/composables/useTeam'

definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Team & access — 1Trade' })

const { user } = useAuth()
const team = useTeam()

const roles = computed(() => user.value?.roles ?? [])
const isAdmin = computed(() => roles.value.includes('admin'))
const canMoveBudget = computed(() => isAdmin.value || roles.value.includes('billing'))
const ROLE_DESC: Record<string, string> = {
  admin: 'Full control — billing, members, keys, settings, audit.',
  billing: 'Manage credits, budgets, sub-account funding and purchases.',
  engineer: 'Run inference + GPU compute; manage API keys.',
  viewer: 'Read-only access to usage and balances.',
  trader: 'Trade credits on the exchange (Phase 2).',
}
const CREDIT_TYPES = ['ai_index', 'text', 'speech', 'image', 'video', 'embeddings', 'gpu_h100', 'gpu_h200']

const members = ref<Member[]>([])
const invites = ref<Invite[]>([])
const subs = ref<SubAccount[]>([])
const error = ref('')
const notice = ref('')
const busy = ref('')

/** load fetches members, sub-accounts and (for admins) pending invitations. */
async function load() {
  error.value = ''
  try {
    const [m, s] = await Promise.all([team.members(), team.subAccounts()])
    members.value = m
    subs.value = s
    invites.value = isAdmin.value ? await team.invites() : []
  } catch (e: any) {
    error.value = e?.data?.message || e?.statusMessage || 'Could not load the team.'
  }
}
onMounted(load)

/** subName names a sub-account (or the main balance). */
const subName = (id: string | null) => (id ? subs.value.find(s => s.id === id)?.name ?? 'unknown' : 'Main balance')

/** act runs one mutation with shared busy / error handling, then reloads. */
async function act(label: string, fn: () => Promise<unknown>, done = '') {
  busy.value = label
  error.value = ''
  notice.value = ''
  try {
    await fn()
    notice.value = done
    await load()
    return true
  } catch (e: any) {
    error.value = e?.data?.message || e?.statusMessage || 'That did not work.'
    return false
  } finally {
    busy.value = ''
  }
}

/** removeMember removes a member after confirmation. */
function removeMember(m: Member) {
  if (globalThis.confirm(`Remove ${m.email} from the team?`)) act('remove', () => team.remove(m.id), `${m.email} removed.`)
}

// Invite form
const inv = reactive({ email: '', role: 'engineer', sub: '' as string })
const lastLink = ref('')
/** sendInvite invites the email; in dev without mail the accept link is shown. */
async function sendInvite() {
  const email = inv.email.trim()
  lastLink.value = ''
  let token = ''
  const ok = await act('invite', async () => {
    token = (await team.invite(email, [inv.role], inv.sub || null)).dev_token ?? ''
  }, `Invitation sent to ${email}.`)
  if (ok) {
    lastLink.value = token ? `${location.origin}/invite?token=${token}` : ''
    inv.email = ''
  }
}

// Sub-accounts
const newSub = ref('')
/** createSub adds a sub-account and clears the field. */
async function createSub() {
  const name = newSub.value.trim()
  if (await act('create', () => team.createSubAccount(name), `Sub-account ${name} created.`)) newSub.value = ''
}
const move = reactive({ sub: '', credit: 'text', amount: '', direction: 'fund' as 'fund' | 'return', key: '' })
watch(() => [move.sub, move.credit, move.amount, move.direction], () => { move.key = '' })
/** moveBudget funds or drains a sub-account; the key is kept until the form changes, so a retry is safe. */
async function moveBudget() {
  if (!move.key) move.key = globalThis.crypto?.randomUUID?.() ?? `mv${Date.now()}`
  const ok = await act('move', () => team.transfer(move.sub, move.credit, move.amount.trim(), move.direction, move.key),
    `${move.direction === 'fund' ? 'Funded' : 'Returned'} ${move.amount.trim()} ${move.credit} ${move.direction === 'fund' ? 'to' : 'from'} ${subName(move.sub)}.`)
  if (ok) move.amount = ''
}
</script>

<template>
  <div class="teams">
    <header class="head">
      <div>
        <div class="eyebrow">Account · Access</div>
        <h1 class="title">Team &amp; access</h1>
        <p class="sub">Members, invitations and sub-accounts. A member placed in a sub-account spends from its balances, which you fund from the main balance.</p>
      </div>
      <span v-if="user?.is_paper" class="chip paper">sandbox</span>
      <span v-else class="chip live">live</span>
    </header>

    <p v-if="error" class="banner neg" role="alert">{{ error }}</p>
    <p v-if="notice" class="banner pos" aria-live="polite">{{ notice }}</p>

    <section class="panel">
      <header class="panel-h"><span class="panel-title">Members</span><span class="panel-meta mono">{{ members.length }}</span></header>
      <table class="tbl">
        <thead><tr><th>Email</th><th>Role</th><th>Sub-account</th><th>Joined</th><th v-if="isAdmin" /></tr></thead>
        <tbody>
          <tr v-for="m in members" :key="m.id">
            <td class="mono">{{ m.email }} <span v-if="m.id === user?.user_id" class="you">you</span></td>
            <td><span v-for="r in m.roles" :key="r" class="role sm">{{ r }}</span></td>
            <td>
              <select v-if="isAdmin" :value="m.sub_account_id ?? ''" :disabled="!!busy" :aria-label="`Sub-account for ${m.email}`"
                @change="act('place', () => team.place(m.id, ($event.target as HTMLSelectElement).value || null), `${m.email} moved (takes effect at their next sign-in).`)">
                <option value="">Main balance</option>
                <option v-for="s in subs" :key="s.id" :value="s.id">{{ s.name }}</option>
              </select>
              <span v-else>{{ subName(m.sub_account_id) }}</span>
            </td>
            <td class="mono muted">{{ m.created_at.slice(0, 10) }}</td>
            <td v-if="isAdmin" class="right">
              <button v-if="m.id !== user?.user_id" type="button" class="link danger" :disabled="!!busy"
                @click="removeMember(m)">Remove</button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <div class="grid">
      <section v-if="isAdmin" class="panel">
        <header class="panel-h"><span class="panel-title">Invite a teammate</span></header>
        <form class="form" @submit.prevent="sendInvite">
          <label>Email<input v-model="inv.email" type="email" required placeholder="name@company.com"></label>
          <div class="row2">
            <label>Role
              <select v-model="inv.role"><option v-for="(_, r) in ROLE_DESC" :key="r" :value="r">{{ r }}</option></select>
            </label>
            <label>Sub-account
              <select v-model="inv.sub"><option value="">Main balance</option><option v-for="s in subs" :key="s.id" :value="s.id">{{ s.name }}</option></select>
            </label>
          </div>
          <button type="submit" class="btn" :disabled="!!busy || !inv.email.includes('@')">Send invitation</button>
          <p v-if="lastLink" class="small muted">No email transport in this environment — the invitation link: <code class="mono">{{ lastLink }}</code></p>
        </form>
        <table v-if="invites.length" class="tbl">
          <thead><tr><th>Pending</th><th>Role</th><th>Expires</th><th /></tr></thead>
          <tbody>
            <tr v-for="i in invites" :key="i.id">
              <td class="mono">{{ i.email }}</td>
              <td><span v-for="r in i.roles" :key="r" class="role sm">{{ r }}</span></td>
              <td class="mono muted">{{ i.expires_at.slice(0, 10) }}</td>
              <td class="right"><button type="button" class="link danger" :disabled="!!busy" @click="act('revoke', () => team.revoke(i.id), `Invitation to ${i.email} revoked.`)">Revoke</button></td>
            </tr>
          </tbody>
        </table>
      </section>

      <section class="panel">
        <header class="panel-h"><span class="panel-title">Sub-accounts</span><span class="panel-meta mono">{{ subs.length }}</span></header>
        <table v-if="subs.length" class="tbl">
          <thead><tr><th>Name</th><th class="right">Members</th></tr></thead>
          <tbody><tr v-for="s in subs" :key="s.id"><td>{{ s.name }}</td><td class="right mono">{{ s.members }}</td></tr></tbody>
        </table>
        <p v-else class="panel-foot muted">No sub-accounts. Everyone spends from the main balance.</p>
        <form v-if="isAdmin" class="form inline" @submit.prevent="createSub">
          <input v-model="newSub" maxlength="80" placeholder="New sub-account, e.g. Research" aria-label="New sub-account name">
          <button type="submit" class="btn" :disabled="!!busy || !newSub.trim()">Create</button>
        </form>
        <form v-if="canMoveBudget && subs.length" class="form" @submit.prevent="moveBudget">
          <div class="row2">
            <label>Direction
              <select v-model="move.direction"><option value="fund">Fund from main balance</option><option value="return">Return to main balance</option></select>
            </label>
            <label>Sub-account
              <select v-model="move.sub" required><option value="" disabled>Choose</option><option v-for="s in subs" :key="s.id" :value="s.id">{{ s.name }}</option></select>
            </label>
          </div>
          <div class="row2">
            <label>Credit<select v-model="move.credit"><option v-for="c in CREDIT_TYPES" :key="c" :value="c">{{ c }}</option></select></label>
            <label>Amount<input v-model="move.amount" inputmode="decimal" class="mono" placeholder="100.00" required></label>
          </div>
          <button type="submit" class="btn" :disabled="!!busy || !move.sub || !move.amount">{{ move.direction === 'fund' ? 'Fund' : 'Return' }}</button>
        </form>
      </section>
    </div>

    <div class="grid">
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Roles</span><span class="panel-meta mono">RBAC</span></header>
        <table class="tbl">
          <tbody>
            <tr v-for="(d, r) in ROLE_DESC" :key="r">
              <td class="role-cell"><span class="role">{{ r }}</span></td>
              <td class="muted rdesc">{{ d }}</td>
            </tr>
          </tbody>
        </table>
      </section>
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Sign-in security</span></header>
        <ul class="m4">
          <li><div class="m4-t">Two-factor authentication</div><div class="m4-d">Authenticator-app codes for your account. <NuxtLink to="/settings" class="m4-link">Settings →</NuxtLink></div></li>
          <li><div class="m4-t">SAML single sign-on</div><div class="m4-d">Sign in through your identity provider. <NuxtLink to="/enterprise/sso" class="m4-link">Configure →</NuxtLink></div></li>
          <li><div class="m4-t">SCIM provisioning<span class="tag-m4 sm">not built</span></div><div class="m4-d">Automatic provisioning from your IdP is not available yet; invite members here.</div></li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.teams { background: var(--canvas); color: var(--text); font-family: var(--font-sans); padding: var(--sp-5); display: flex; flex-direction: column; gap: var(--sp-4); max-width: 1100px; margin: 0 auto; }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.muted { color: var(--text-3); }
.strong { font-weight: 600; }

.head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 6px; line-height: 1; }
.sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0; max-width: 560px; }
.chip { font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.1em; padding: 3px var(--sp-2); border-radius: var(--radius-sm); }
.chip.live { background: var(--pos-soft); color: var(--pos); }
.chip.paper { color: var(--info); border: 1px solid var(--border); }

.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden; }
.panel-h { display: flex; justify-content: space-between; align-items: center; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.panel-title { font-size: var(--fs-sm); font-weight: 600; }
.panel-meta { font-size: var(--fs-xs); color: var(--text-3); }
.panel-foot { padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border); font-size: var(--fs-xs); }

.grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-4); align-items: start; }

/* key/value (your account) */
.kv { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: var(--sp-4); padding: var(--sp-4); margin: 0; }
.kv dt { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); margin-bottom: 4px; }
.kv dd { margin: 0; font-size: var(--fs-sm); color: var(--text); }
.rolewrap { display: flex; gap: 4px; flex-wrap: wrap; }
.role { font-family: var(--font-mono); font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: 0.06em; background: var(--overlay); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 2px var(--sp-2); color: var(--text); }
.role.sm { font-size: 10px; }

/* tables */
.tbl { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
.tbl th { text-align: left; font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); font-weight: 500; padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); }
.tbl td { padding: var(--sp-2) var(--sp-4); border-bottom: 1px solid var(--border); vertical-align: top; }
.tbl tbody tr:last-child td { border-bottom: 0; }
.you { font-size: 10px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-3); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 0 4px; margin-left: 4px; }
.dot-ok { display: inline-block; width: 6px; height: 6px; border-radius: var(--radius-full); background: var(--pos); margin-right: 6px; }
.role-cell { width: 110px; }
.rdesc { font-size: var(--fs-xs); }

/* M4 roadmap */
.tag-m4 { font-family: var(--font-mono); font-size: var(--fs-tiny); font-weight: 600; letter-spacing: 0.1em; color: var(--warn); border: 1px solid var(--warn); border-radius: var(--radius-sm); padding: 1px var(--sp-2); }
.tag-m4.sm { font-size: 9px; padding: 0 5px; margin-left: 8px; }
.m4 { list-style: none; margin: 0; padding: 0; }
.m4 li { padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.m4 li:last-child { border-bottom: 0; }
.m4-t { font-size: var(--fs-sm); font-weight: 600; display: flex; align-items: center; }
.m4-d { font-size: var(--fs-xs); color: var(--text-3); margin-top: 3px; }
.m4-link { color: var(--brand); text-decoration: none; margin-left: 6px; }


.banner { margin: 0; padding: var(--sp-2) var(--sp-3); border-radius: var(--radius-sm); font-size: var(--fs-sm); }
.banner.neg { background: var(--neg-soft); color: var(--neg); }
.banner.pos { background: var(--pos-soft); color: var(--pos); }
.form { display: flex; flex-direction: column; gap: var(--sp-3); padding: var(--sp-4); border-top: 1px solid var(--border); }
.form:first-child { border-top: 0; }
.form.inline { flex-direction: row; }
.form.inline input { flex: 1; }
.form label { display: flex; flex-direction: column; gap: 4px; font-size: var(--fs-xs); color: var(--text-2); }
.form input, .form select, .tbl select { background: var(--canvas); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 6px 8px; color: var(--text); font-size: var(--fs-sm); }
.row2 { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-3); }
.btn { align-self: flex-start; background: var(--brand); color: var(--text-on-accent); border: 0; border-radius: var(--radius-sm); padding: 7px 14px; font-size: var(--fs-sm); font-weight: 600; cursor: pointer; }
.btn:disabled { opacity: 0.5; cursor: default; }
.link { background: none; border: 0; padding: 0; cursor: pointer; font-size: var(--fs-xs); color: var(--brand); }
.link.danger { color: var(--neg); }
.right { text-align: right; }
.small { font-size: var(--fs-xs); margin: 0; word-break: break-all; }

@media (max-width: 860px) { .grid { grid-template-columns: 1fr; } }
</style>
