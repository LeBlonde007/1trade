<script setup lang="ts">
/**
 * StatTile — label + big mono number + optional delta.
 * Used in hero stats, account header, market detail stats grid, billing summary.
 *
 * <StatTile label="24h Volume" value="$1.24M" :delta="0.123" />
 * <StatTile label="Total Value" value="$10,247.83" :delta="0.0023" size="lg" />
 */
withDefaults(
  defineProps<{
    label: string
    /** Pre-formatted display value (use format utils to prepare it). */
    value: string
    /** Optional change as decimal: 0.0235 = +2.35%. */
    delta?: number
    /** Optional caption below the value. */
    caption?: string
    /** Size scale. */
    size?: 'sm' | 'md' | 'lg' | 'xl'
  }>(),
  {
    size: 'md',
  },
)
</script>

<template>
  <div class="stat-tile" :class="`size-${size}`">
    <span class="label">{{ label }}</span>
    <span class="value">{{ value }}</span>
    <PriceDelta v-if="delta !== undefined" :pct="delta" class="delta" />
    <span v-if="caption" class="caption">{{ caption }}</span>
  </div>
</template>

<style scoped>
.stat-tile {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.label {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
}

.value {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--text);
  line-height: var(--lh-tight);
}

.size-sm .value { font-size: var(--fs-lg); }
.size-md .value { font-size: var(--fs-2xl); }
.size-lg .value { font-size: var(--fs-3xl); }
.size-xl .value { font-size: var(--fs-5xl); letter-spacing: var(--ls-snug); }

.delta {
  font-size: var(--fs-sm);
}

.caption {
  font-size: var(--fs-sm);
  color: var(--text-2);
}
</style>
