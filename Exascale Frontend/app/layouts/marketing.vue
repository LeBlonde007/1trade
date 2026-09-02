<script setup lang="ts">
/**
 * Marketing layout — public pages (landing, index methodology, status, 404).
 *
 * Theme is per-page, not fixed by the layout. 1TRADE is a dark-first brand, but
 * `benchmark` and `status` were built against light surfaces and carry light-specific
 * styling, so flipping the whole layout would break them. A page opts in with:
 *
 *   definePageMeta({ layout: 'marketing', theme: 'dark' })
 *
 * Default stays 'light' so existing pages are untouched. We lock the choice on <html>
 * as well as the shell so body/scrollbar/html background flip together and we never
 * inherit the previous route's theme.
 */
const route = useRoute()
const theme = computed(() => (route.meta.theme === 'dark' ? 'dark' : 'light'))

useHead({ htmlAttrs: { 'data-theme': theme } })
</script>

<template>
  <div class="m-shell" :data-theme="theme">
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
