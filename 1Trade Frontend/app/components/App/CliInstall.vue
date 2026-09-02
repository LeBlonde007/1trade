<script setup lang="ts">
/**
 * CliInstall — a compact "Install the CLI" snippet with a one-click copy. The URL is derived from the
 * current origin (set after mount to avoid a hydration mismatch), since the web image bundles the CLI
 * binaries + installer at /cli/. So `curl … | sh` always points at whatever host serves the app.
 */
const origin = ref('https://sandbox.trade1.ai')
onMounted(() => { origin.value = window.location.origin })
const cmd = computed(() => `curl -fsSL ${origin.value}/cli/install.sh | sh`)

const copied = ref(false)
async function copy() {
  try {
    await navigator.clipboard.writeText(cmd.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* clipboard unavailable */ }
}
</script>

<template>
  <div class="cli-install">
    <span class="cli-label">Install the CLI</span>
    <code class="cli-cmd">{{ cmd }}</code>
    <button class="cli-copy" :class="{ copied }" type="button" @click="copy">
      {{ copied ? '✓ Copied' : 'Copy' }}
    </button>
  </div>
</template>

<style scoped>
.cli-install {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: 2px;
  font-size: 13px;
}
.cli-label { color: var(--text-2); white-space: nowrap; }
.cli-cmd {
  flex: 1;
  min-width: 0;
  color: var(--text);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  overflow-x: auto;
  white-space: nowrap;
}
.cli-copy {
  background: none;
  border: 1px solid var(--border-strong);
  border-radius: 2px;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--text);
  cursor: pointer;
  white-space: nowrap;
  transition: color 160ms ease, border-color 160ms ease;
}
.cli-copy:hover { background: rgba(0, 0, 0, 0.03); }
.cli-copy.copied { color: var(--pos); border-color: rgba(25, 195, 125, 0.4); }
</style>
