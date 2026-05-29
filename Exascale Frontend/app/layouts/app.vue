<script setup lang="ts">
/**
 * App layout — dark theme, authenticated trading screens.
 * The trading product is ALWAYS dark regardless of any other surface theme.
 * We lock both <html> and the shell <div> to data-theme="dark" so that:
 *   - Body / scrollbar / html bg use dark tokens (no light flash)
 *   - Any descendant page or component that reads var(--canvas) / var(--text)
 *     gets the dark palette, even if it tries to override at the page level.
 */
useHead({ htmlAttrs: { 'data-theme': 'dark' } })
</script>

<template>
  <div class="a-shell" data-theme="dark">
    <AppTopbar />
    <AppSidebar />
    <main class="a-main">
      <slot />
    </main>
    <AppCommandPalette />
    <AppNotifications />
    <TradeOrderDetail />
    <TradePositionDetail />
    <AppTransactionDetail />
  </div>
</template>

<style scoped>
.a-shell {
  display: grid;
  grid-template-columns: var(--app-sb-w) 1fr;
  grid-template-rows: var(--topbar-h) 1fr;
  grid-template-areas:
    "topbar topbar"
    "sidebar main";
  width: 100vw;
  height: 100vh;
  background: var(--canvas);
  color: var(--text);
  overflow: hidden;
}

.a-main {
  grid-area: main;
  overflow: auto;
  background: var(--canvas);
  color: var(--text);
}
</style>
