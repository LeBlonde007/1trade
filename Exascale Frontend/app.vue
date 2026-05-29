<template>
  <!--
    Canonical root shell.
    GuidedTour + Toasts mount here (outside <NuxtLayout>) so they survive
    pages with `layout: false` (signup, kyc, wallet/buy, etc.).
    `App/GuidedTour.client.vue` is already client-only by file suffix;
    `App/Toasts.vue` is small and safe to SSR.
  -->
  <NuxtLoadingIndicator color="var(--brand)" />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <AppGuidedTour />
  <AppToasts />
</template>
