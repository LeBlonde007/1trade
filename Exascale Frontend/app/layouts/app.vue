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
// Mobile drawer state — the backdrop below dismisses the off-canvas sidebar.
const sidebar = useSidebar()
</script>

<template>
  <div class="a-shell" data-theme="dark">
    <AppTopbar />
    <AppSidebar />
    <!-- Backdrop: only shown/interactive on mobile while the drawer is open. -->
    <div class="a-overlay" :class="{ show: sidebar.open.value }" @click="sidebar.close()" />
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
  min-width: 0; /* let grid children shrink instead of forcing horizontal overflow */
}

/* Backdrop behind the mobile drawer (desktop: never shown). */
.a-overlay {
  position: fixed;
  inset: var(--topbar-h) 0 0 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 220ms ease;
  z-index: 60;
}

/* ── Mobile: collapse to a single column; the sidebar overlays as a drawer. ── */
@media (max-width: 768px) {
  .a-shell {
    grid-template-columns: 1fr;
    grid-template-areas:
      "topbar"
      "main";
  }
  .a-overlay.show { opacity: 1; pointer-events: auto; }
}
</style>
