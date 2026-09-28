<script setup lang="ts">
/**
 * /invite?token= — accept a team invitation (platform-core v1.6). Public: the emailed token is the
 * credential. Shows which team and role the invitation is for, takes a new password, creates the
 * member (email already verified) and signs them in.
 */
definePageMeta({ layout: false })
useHead({ title: 'Join your team — 1Trade' })

const route = useRoute()
const token = typeof route.query.token === 'string' ? route.query.token : ''
const team = useTeam()
const { refresh } = useAuth()
const info = ref<{ email: string; tenant_name: string; roles: string[]; expires_at: string } | null>(null)
const invalid = ref(false)
const password = ref('')
const confirmPw = ref('')
const busy = ref(false)
const error = ref('')

onMounted(async () => {
  if (!token) { invalid.value = true; return }
  try { info.value = await team.lookupInvite(token) } catch { invalid.value = true }
})

/** accept creates the account and continues to the console. */
async function accept() {
  error.value = ''
  if (password.value.length < 8) { error.value = 'Use at least 8 characters.'; return }
  if (password.value !== confirmPw.value) { error.value = 'The passwords do not match.'; return }
  busy.value = true
  try {
    await team.acceptInvite(token, password.value)
    await refresh()
    await navigateTo('/console')
  } catch (e: any) {
    error.value = e?.data?.code === 'invite_invalid'
      ? 'This invitation is no longer valid. Ask your admin for a new one.'
      : (e?.data?.message || e?.statusMessage || 'Could not accept the invitation.')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="inv">
    <div class="card">
      <div class="brand">1TRADE</div>
      <template v-if="invalid">
        <h1>Invitation not valid</h1>
        <p class="muted">This link is invalid, already used, revoked or expired. Ask your team admin to invite you again.</p>
        <NuxtLink to="/login" class="link">Sign in instead →</NuxtLink>
      </template>
      <template v-else-if="info">
        <h1>Join {{ info.tenant_name }}</h1>
        <p class="muted">
          You were invited as <strong>{{ info.roles.join(', ') }}</strong>. Choose a password for
          <span class="mono">{{ info.email }}</span>.
        </p>
        <form @submit.prevent="accept">
          <label>Password<input v-model="password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required></label>
          <label>Confirm password<input v-model="confirmPw" type="password" autocomplete="new-password" required></label>
          <p v-if="error" class="err" role="alert">{{ error }}</p>
          <button type="submit" :disabled="busy">{{ busy ? 'Joining…' : 'Join team' }}</button>
        </form>
        <p class="muted small">This invitation expires {{ info.expires_at.slice(0, 10) }}.</p>
      </template>
      <p v-else class="muted">Checking your invitation…</p>
    </div>
  </main>
</template>

<style scoped>
.inv { min-height: 100vh; display: grid; place-items: center; background: var(--canvas); color: var(--text); padding: 16px; }
.card { width: 100%; max-width: 420px; background: var(--elevated); border: 1px solid var(--border); border-radius: 10px; padding: 28px; display: flex; flex-direction: column; gap: 14px; }
.brand { font-family: var(--font-mono); letter-spacing: 0.2em; font-size: 13px; color: var(--brand); }
h1 { margin: 0; font-size: 22px; letter-spacing: -0.01em; }
.muted { color: var(--text-2); font-size: 14px; margin: 0; }
.small { font-size: 12px; }
.mono { font-family: var(--font-mono); }
form { display: flex; flex-direction: column; gap: 12px; }
label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-2); }
input { background: var(--canvas); border: 1px solid var(--border); border-radius: 6px; padding: 9px 10px; color: var(--text); font-size: 14px; }
button { background: var(--brand); color: var(--text-on-accent); border: 0; border-radius: 6px; padding: 10px; font-weight: 600; cursor: pointer; }
button:disabled { opacity: 0.6; }
.err { color: var(--neg); font-size: 13px; margin: 0; }
.link { color: var(--brand); font-size: 14px; }
</style>
