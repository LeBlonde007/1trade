<script setup lang="ts">
/**
 * BaseCard — sharp institutional card.
 * Has optional header slot. Default radius is sm (2px) per brand book.
 *
 * <BaseCard>
 *   <template #header>Order Book · 10 levels</template>
 *   …body…
 * </BaseCard>
 */
withDefaults(
  defineProps<{
    /** Padding density. 'tight' for trading panels, 'comfortable' for marketing cards. */
    density?: 'tight' | 'comfortable'
    /** Surface variant. 'elevated' (default) or 'sunken' (subtle/recessed). */
    surface?: 'elevated' | 'sunken'
    /** Highlight border (e.g. for active/featured card). */
    highlight?: boolean
  }>(),
  {
    density: 'comfortable',
    surface: 'elevated',
    highlight: false,
  },
)
</script>

<template>
  <article
    class="card"
    :class="[`density-${density}`, `surface-${surface}`, { highlight }]"
  >
    <header v-if="$slots.header" class="card-head">
      <slot name="header" />
    </header>
    <div class="card-body">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="card-foot">
      <slot name="footer" />
    </footer>
  </article>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  transition: border-color var(--dur) var(--ease);
}

.surface-sunken {
  background: var(--sunken, var(--canvas));
}

.highlight {
  border-color: var(--brand);
}

.card-head {
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
}

.card-body {
  flex: 1;
}

.density-tight  .card-body { padding: var(--sp-3) var(--sp-4); }
.density-comfortable .card-body { padding: var(--sp-5) var(--sp-6); }

.card-foot {
  padding: var(--sp-3) var(--sp-4);
  border-top: 1px solid var(--border);
}
</style>
