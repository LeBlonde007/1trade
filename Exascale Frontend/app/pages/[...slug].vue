<script setup lang="ts">
/**
 * Catch-all 404. Marketing layout, light theme.
 */
definePageMeta({ layout: 'marketing' })
useHead({ title: '404 — 1Trade' })
</script>

<template>
  <section class="nf">
    <div class="bound">
      <p class="eyebrow"><span class="dot" />404</p>
      <h1 class="title">This page doesn't exist.</h1>
      <p class="lede">It may have moved, or never existed at all.</p>
      <div class="cta">
        <BaseButton as="a" :href="'/'" variant="primary" size="lg">Go to homepage →</BaseButton>
        <BaseButton as="a" :href="'/trade'" variant="secondary" size="lg">Open trading dashboard</BaseButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.nf {
  padding: var(--sp-10) var(--sp-6);
  min-height: calc(100vh - 60px);
  display: flex;
  align-items: center;
}

.bound {
  max-width: 600px;
  margin: 0 auto;
  text-align: left;
}

.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-3);
  margin: 0 0 var(--sp-4);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-extra);
  text-transform: uppercase;
  color: var(--text-3);
}

.eyebrow .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--brand); }

.title {
  font-size: var(--fs-5xl);
  letter-spacing: var(--ls-snug);
  line-height: var(--lh-tight);
  margin: 0 0 var(--sp-3);
}

.lede {
  font-size: var(--fs-lg);
  color: var(--text-2);
  margin: 0 0 var(--sp-7);
}

.cta {
  display: flex;
  gap: var(--sp-3);
  flex-wrap: wrap;
}
</style>
