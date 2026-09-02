<script setup lang="ts">
/**
 * /enterprise/sso — Single sign-on (SAML). Honest M4 state: SAML/SCIM/2FA are M4 additions (the
 * contract says so), so this shows the current sign-in method (live) + what enterprise SSO will bring,
 * rather than a fake configuration wizard. No mock IdP connections.
 */
definePageMeta({ layout: 'app', middleware: 'auth' })
useHead({ title: 'Single sign-on — 1Trade' })

const { user } = useAuth()

const IDPS = ['Okta', 'Microsoft Entra ID', 'Google Workspace', 'OneLogin', 'JumpCloud', 'Ping Identity']
const M4 = [
  { t: 'SAML 2.0 SSO', d: 'Connect your IdP; enforce SSO for every member of the org.' },
  { t: 'SCIM provisioning', d: 'Auto-provision + de-provision users and groups from your directory.' },
  { t: 'Enforced 2FA', d: 'Require TOTP / WebAuthn for all members, or fall back to IdP MFA.' },
  { t: 'Just-in-time users', d: 'New members are created on first SSO login with mapped roles.' },
]
</script>

<template>
  <div class="sso">
    <header class="head">
      <div>
        <div class="eyebrow">Security · Single sign-on</div>
        <h1 class="title">SAML single sign-on</h1>
        <p class="sub">Enterprise SSO lets your team sign in through your identity provider. Available in M4.</p>
      </div>
      <span class="tag-m4">M4</span>
    </header>

    <!-- Current sign-in (live) -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Current sign-in</span></header>
      <div class="cur">
        <div class="cur-row">
          <span class="cur-l">Method</span>
          <span class="cur-v">Email &amp; password <span class="muted">· OAuth (Google / GitHub) scaffolded</span></span>
        </div>
        <div class="cur-row">
          <span class="cur-l">Signed in as</span>
          <span class="cur-v mono">{{ user?.email || '—' }}</span>
        </div>
        <div class="cur-row">
          <span class="cur-l">SSO enforced</span>
          <span class="cur-v"><span class="dot-off" />Not enforced — available in M4</span>
        </div>
      </div>
    </section>

    <!-- What M4 brings -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Coming in M4 — Enterprise SSO</span><span class="tag-m4 sm">M4</span></header>
      <ul class="feat">
        <li v-for="m in M4" :key="m.t">
          <div class="feat-t">{{ m.t }}</div>
          <div class="feat-d">{{ m.d }}</div>
        </li>
      </ul>
    </section>

    <!-- IdP support -->
    <section class="panel">
      <header class="panel-h"><span class="panel-title">Identity providers (planned)</span></header>
      <div class="idps">
        <span v-for="i in IDPS" :key="i" class="idp mono">{{ i }}</span>
      </div>
      <footer class="panel-foot muted">
        Any SAML 2.0 IdP will be supported. Need this sooner? <a href="mailto:enterprise@exascale.io" class="link">Talk to us →</a>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.sso { background: var(--canvas); color: var(--text); font-family: var(--font-sans); padding: var(--sp-5); display: flex; flex-direction: column; gap: var(--sp-4); max-width: 900px; margin: 0 auto; }
.mono { font-family: var(--font-mono); }
.muted { color: var(--text-3); }

.head { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border); padding-bottom: var(--sp-4); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.16em; color: var(--text-3); }
.title { font-size: var(--fs-3xl); font-weight: 600; letter-spacing: -0.02em; margin: 4px 0 6px; line-height: 1; }
.sub { font-size: var(--fs-sm); color: var(--text-2); margin: 0; max-width: 560px; }
.tag-m4 { font-family: var(--font-mono); font-size: var(--fs-tiny); font-weight: 600; letter-spacing: 0.1em; color: var(--warn); border: 1px solid var(--warn); border-radius: var(--radius-sm); padding: 2px var(--sp-2); }
.tag-m4.sm { font-size: 9px; }

.panel { background: var(--elevated); border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden; }
.panel-h { display: flex; justify-content: space-between; align-items: center; padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.panel-title { font-size: var(--fs-sm); font-weight: 600; }
.panel-foot { padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border); font-size: var(--fs-xs); }
.link { color: var(--brand); text-decoration: none; }

.cur { padding: var(--sp-4); display: flex; flex-direction: column; gap: var(--sp-3); }
.cur-row { display: flex; gap: var(--sp-4); font-size: var(--fs-sm); }
.cur-l { font-family: var(--font-mono); font-size: var(--fs-tiny); text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-3); width: 120px; flex-shrink: 0; padding-top: 2px; }
.cur-v { color: var(--text); }
.dot-off { display: inline-block; width: 6px; height: 6px; border-radius: var(--radius-full); background: var(--text-3); margin-right: 6px; }

.feat { list-style: none; margin: 0; padding: 0; }
.feat li { padding: var(--sp-3) var(--sp-4); border-bottom: 1px solid var(--border); }
.feat li:last-child { border-bottom: 0; }
.feat-t { font-size: var(--fs-sm); font-weight: 600; }
.feat-d { font-size: var(--fs-xs); color: var(--text-3); margin-top: 3px; }

.idps { display: flex; flex-wrap: wrap; gap: var(--sp-2); padding: var(--sp-4); }
.idp { font-size: var(--fs-xs); background: var(--overlay); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 4px var(--sp-3); color: var(--text-2); }
</style>
