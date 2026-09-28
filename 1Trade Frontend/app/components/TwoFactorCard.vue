<script setup lang="ts">
/**
 * TwoFactorCard — the live two-factor section of Settings (platform-core v1.7): status, enrolment
 * (secret + otpauth link, confirm a code, recovery codes shown once) and turning it off with a code.
 */
const status = ref<{ enabled: boolean; recovery_codes_left: number } | null>(null)
const setup = ref<{ secret: string; otpauth_uri: string } | null>(null)
const recovery = ref<string[]>([])
const code = ref('')
const disableCode = ref('')
const busy = ref(false)
const error = ref('')

/** load reads the current status. */
async function load() {
  try { status.value = await $fetch<{ enabled: boolean; recovery_codes_left: number }>('/api/auth/2fa') } catch { status.value = null }
}
onMounted(load)

/** run performs one call with shared busy / error handling. */
async function run(fn: () => Promise<void>) {
  busy.value = true
  error.value = ''
  try { await fn() } catch (e: any) {
    error.value = e?.data?.code === 'mfa_locked' ? 'Too many wrong codes — try again in 15 minutes.' : (e?.data?.message || 'That did not work.')
  } finally { busy.value = false }
}

const start = () => run(async () => { setup.value = await $fetch<{ secret: string; otpauth_uri: string }>('/api/auth/2fa/setup', { method: 'POST' }) })
const enable = () => run(async () => {
  const r = await $fetch<{ recovery_codes: string[] }>('/api/auth/2fa/enable', { method: 'POST', body: { code: code.value.trim() } })
  recovery.value = r.recovery_codes
  setup.value = null
  code.value = ''
  await load()
})
const disable = () => run(async () => {
  const c = disableCode.value.trim()
  await $fetch<unknown>('/api/auth/2fa/disable', { method: 'POST', body: c.includes('-') ? { recovery_code: c } : { code: c } })
  disableCode.value = ''
  recovery.value = []
  await load()
})
</script>

<template>
  <div class="card tfa">
    <div class="card-head">
      <span class="eyebrow"><span class="dot" /> Two-factor authentication</span>
      <span v-if="status?.enabled" class="status-tag verified"><span class="dot" /> Enabled</span>
      <span v-else-if="status" class="status-tag">Off</span>
    </div>
    <div class="card-body">
      <p v-if="error" class="tfa-err" role="alert">{{ error }}</p>

      <div v-if="recovery.length" class="tfa-box" aria-live="polite">
        <strong>Save these recovery codes now.</strong> Each works once if you lose your authenticator. They will not be shown again.
        <ul class="tfa-codes mono"><li v-for="c in recovery" :key="c">{{ c }}</li></ul>
      </div>

      <template v-if="status?.enabled">
        <dl class="kv-grid">
          <dt>Method</dt><dd>Authenticator app (TOTP)</dd>
          <dt>Recovery codes</dt><dd>{{ status.recovery_codes_left }} of 10 unused</dd>
        </dl>
        <form class="tfa-row" @submit.prevent="disable">
          <input v-model="disableCode" class="tfa-input mono" placeholder="Code or recovery code" aria-label="Code to turn off two-factor" autocomplete="one-time-code">
          <button type="submit" class="btn secondary" :disabled="busy || !disableCode.trim()">Turn off</button>
        </form>
      </template>

      <template v-else-if="setup">
        <p class="tfa-p">Add this key to your authenticator app (Google Authenticator, 1Password, Authy…), or open the link on your phone:</p>
        <p class="mono tfa-secret">{{ setup.secret.match(/.{1,4}/g)?.join(' ') }}</p>
        <a class="tfa-link" :href="setup.otpauth_uri">Open in authenticator app</a>
        <form class="tfa-row" @submit.prevent="enable">
          <input v-model="code" class="tfa-input mono" inputmode="numeric" maxlength="6" placeholder="6-digit code" aria-label="Code from your authenticator" autocomplete="one-time-code">
          <button type="submit" class="btn" :disabled="busy || code.trim().length !== 6">Turn on</button>
        </form>
      </template>

      <div v-else-if="status" class="row-action">
        <div class="row-text">
          <div class="row-title">Protect your sign-in</div>
          <div class="row-sub">After your password, sign-in asks for a code from your authenticator app.</div>
        </div>
        <button type="button" class="btn secondary" :disabled="busy" @click="start">Set up</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tfa-err { color: var(--neg); font-size: 13px; margin: 0 0 10px; }
.tfa-box { border: 1px solid var(--warn); border-radius: 6px; padding: 10px 12px; font-size: 13px; margin-bottom: 12px; }
.tfa-codes { list-style: none; padding: 0; margin: 8px 0 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 4px; }
.tfa-p { font-size: 13px; color: var(--text-2); margin: 0 0 8px; }
.tfa-secret { font-size: 15px; letter-spacing: 0.08em; margin: 0 0 6px; word-break: break-all; }
.tfa-link { font-size: 13px; color: var(--brand); }
.tfa-row { display: flex; gap: 8px; margin-top: 12px; }
.tfa-input { flex: 0 1 220px; background: var(--canvas); border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px; color: var(--text); }
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
</style>
