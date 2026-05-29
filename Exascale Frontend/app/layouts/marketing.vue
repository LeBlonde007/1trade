<script setup lang="ts">
/**
 * Marketing layout — light theme, public pages.
 * Locks <html data-theme="light"> so body/scrollbar/html bg flip to the
 * light palette and we don't inherit dark from a prior page.
 */
useHead({ htmlAttrs: { 'data-theme': 'light' } })
</script>

<template>
  <div class="m-shell" data-theme="light">
    <MarketingNav />
    <main>
      <slot />
    </main>
    <MarketingFooter />
  </div>
</template>

<style scoped>
.m-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--canvas);
  color: var(--text);
}

main {
  flex: 1;
}
</style>
