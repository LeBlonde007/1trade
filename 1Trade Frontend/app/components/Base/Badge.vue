<script setup lang="ts">
/**
 * BaseBadge — small status/category pill.
 *
 * <BaseBadge tone="positive">Active</BaseBadge>
 * <BaseBadge tone="negative">Pending</BaseBadge>
 * <BaseBadge tone="brand">New</BaseBadge>
 */
withDefaults(
  defineProps<{
    tone?: 'neutral' | 'positive' | 'negative' | 'warning' | 'info' | 'brand'
    /** Display variant: 'filled' (solid soft bg) vs 'outline' (border only). */
    variant?: 'filled' | 'outline'
    size?: 'sm' | 'md'
  }>(),
  {
    tone: 'neutral',
    variant: 'filled',
    size: 'sm',
  },
)
</script>

<template>
  <span class="badge" :class="[`tone-${tone}`, `variant-${variant}`, `size-${size}`]">
    <slot />
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-weight: 600;
  letter-spacing: var(--ls-tab);
  text-transform: uppercase;
  line-height: 1;
}

.size-sm { padding: 3px 6px; font-size: var(--fs-tiny); }
.size-md { padding: 5px 10px; font-size: var(--fs-xs); }

/* Filled (soft semantic background) */
.variant-filled.tone-neutral  { background: var(--sunken, var(--hover)); color: var(--text-2); }
.variant-filled.tone-positive { background: var(--pos-soft); color: var(--pos); }
.variant-filled.tone-negative { background: var(--neg-soft); color: var(--neg); }
.variant-filled.tone-warning  { background: rgba(245, 158, 11, 0.14); color: var(--warn); }
.variant-filled.tone-info     { background: rgba(74, 144, 226, 0.14); color: var(--info, var(--accent)); }
.variant-filled.tone-brand    { background: var(--brand); color: var(--text-on-accent); }

/* Outline */
.variant-outline {
  background: transparent;
  border: 1px solid var(--border-strong);
  color: var(--text-2);
}
.variant-outline.tone-positive { border-color: var(--pos); color: var(--pos); }
.variant-outline.tone-negative { border-color: var(--neg); color: var(--neg); }
.variant-outline.tone-warning  { border-color: var(--warn); color: var(--warn); }
.variant-outline.tone-brand    { border-color: var(--brand); color: var(--brand); }
</style>
