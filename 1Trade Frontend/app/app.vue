<script setup lang="ts">
/**
 * Root app component (Nuxt 4).
 * Holds NuxtLayout + NuxtPage and mounts the truly global overlays:
 *   - <AppTour />    — persona-driven autoplay product tour
 *   - <AppToasts />  — global ephemeral toast queue
 * These need to live above the layout so they survive pages that opt out of
 * layouts entirely with `definePageMeta({ layout: false })`.
 */
</script>

<template>
  <div>
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
    <!-- Two coexisting tour engines (user picks the mode at /onboarding/tour):
         - GuidedTour: driver.js, click-through with per-step feedback (default)
         - AppTour:    autoplay video-style overlay -->
    <AppGuidedTour />
    <AppTour />
    <AppToasts />
  </div>
</template>
