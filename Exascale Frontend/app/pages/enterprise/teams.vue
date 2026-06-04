<script setup lang="ts">
/**
 * /enterprise/teams — Team & access. Honest, live-where-real: shows the tenant's actual account +
 * the signed-in member and their roles (from the session), the role reference (F03 RBAC), and a clear
 * roadmap of the team features that land in M4 (sub-accounts, invites, SAML SSO, SCIM, 2FA). No mock
 * members/budgets — the backend for those is M4 (F02/F03), so we don't fabricate them.
 */
definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Team & access — Exascale' })

const { user } = useAuth()

const roles = computed(() => user.value?.roles ?? [])
const ROLE_DESC: Record<string, string> = {
  admin: 'Full control — billing, members, keys, settings, audit.',
  billing: 'Manage credits, budgets, and purchases.',
  engineer: 'Run inference + GPU compute; manage API keys.',
  viewer: 'Read-only access to usage and balances.',
  trader: 'Trade credits on the exchange (Phase 2).',
}
function short(id?: string): string { return id ? id.slice(0, 8) : '—' }
function memberName(email?: string): string {
  const local = (email ?? '').split('@')[0] || 'you'
  return local.split(/[._-]/).filter(Boolean).map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ') || 'You'
}

// Team capabilities that arrive in M4 (no backend yet — shown as roadmap, not faked).
const M4 = [
  { t: 'Sub-accounts', d: 'Isolated per-team accounts with their own balances + budgets, rolled up to the org.' },
  { t: 'Invite teammates', d: 'Add members by email with a role; pending-invite + revoke flow.' },
  { t: 'SAML single sign-on', d: 'Okta, Microsoft Entra ID, Google Workspace — enforce SSO org-wide.', to: '/enterprise/sso' },
  { t: 'SCIM provisioning', d: 'Auto-provision + de-provision users from your IdP.' },
  { t: 'Two-factor auth', d: 'Enforce 2FA for all members (TOTP + WebAuthn).' },
]
</script>

<template>
  <div class="teams">
    <header class="head">
      <div>
        <div class="eyebrow">Account · Access</div>
        <h1 class="title">Team &amp; access</h1>
        <p class="sub">Your organization, members, and roles. Sub-accounts, invites, and SSO arrive in M4.</p>
      </div>
      <span v-if="user?.is_paper" class="chip paper">sandbox</span>
      <span v-else class="chip live">live</span>
    </header>

    <!-- Your account (live) -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Your account</span></header>
      <dl class="kv">
        <div><dt>Signed in as</dt><dd>{{ user?.email || '—' }}</dd></div>
        <div><dt>Roles</dt><dd class="rolewrap"><span v-for="r in roles" :key="r" class="role">{{ r }}</span><span v-if="!roles.length" class="muted">—</span></dd></div>
        <div><dt>Tenant</dt><dd class="mono">{{ short(user?.tenant_id) }}</dd></div>
        <div><dt>Organization</dt><dd class="mono">{{ user?.org_id ? short(user.org_id) : 'default' }}</dd></div>
        <div><dt>Mode</dt><dd>{{ user?.is_paper ? 'Sandbox (paper credits)' : 'Live (real money)' }}</dd></div>
      </dl>
    </section>

    <div class="grid">
      <!-- Members (live: just you for now) -->
      <section class="panel">
        <header class="panel-h"><span class="panel-title">Members</span><span class="panel-meta mono">1</span></header>
        <table class="tbl">
          <thead><tr><th>Member</th><th>Email</th><th>Role</th><th>Status</th></tr></thead>
          <tbody>
            <tr>
              <td class="strong">{{ memberName(user?.email) }} <span class="you">you</span></td>
              <td class="mono muted">{{ user?.email }}</td>
              <td><span v-for="r in roles" :key="r" class="role sm">{{ r }}</span></td>
              <td><span class="dot-ok" />active</td>
            </tr>
          </tbody>
        </table>
        <footer class="panel-foot muted">
          You're the only member. Inviting teammates + assigning roles lands in M4.
        </footer>
      </section>

      <!-- Roles reference (F03 RBAC) -->
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
    </div>

    <!-- Coming in M4 -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Team management</span><span class="tag-m4">M4</span></header>
      <ul class="m4">
        <li v-for="m in M4" :key="m.t">
          <div class="m4-t">{{ m.t }}<span class="tag-m4 sm">M4</span></div>
          <div class="m4-d">{{ m.d }} <NuxtLink v-if="m.to" :to="m.to" class="m4-link">Preview →</NuxtLink></div>
        </li>
      </ul>
    </section>
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

@media (max-width: 860px) { .grid { grid-template-columns: 1fr; } }
</style>
