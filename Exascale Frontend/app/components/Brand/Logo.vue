<script setup lang="ts">
/**
 * BrandLogo — the canonical Exascale wordmark + mark.
 * Auto-imported as <BrandLogo />.
 *
 * Pulls colors from the current theme tokens — no hardcoded color.
 */
const props = withDefaults(
  defineProps<{
    /** Visual variant. 'full' = mark + wordmark, 'mark' = lime square only, 'wordmark' = text only. */
    variant?: 'full' | 'mark' | 'wordmark'
    /** Optional size scalar. Defaults to base. */
    size?: 'sm' | 'md' | 'lg'
  }>(),
  {
    variant: 'full',
    size: 'md',
  },
)

const sizeMap = {
  sm: { mark: 14, font: 12, gap: 8, ls: '0.14em' },
  md: { mark: 18, font: 14, gap: 10, ls: '0.16em' },
  lg: { mark: 24, font: 18, gap: 14, ls: '0.18em' },
} as const

const s = computed(() => sizeMap[props.size])
</script>

<template>
  <span class="brand-logo" :data-variant="variant">
    <span
      v-if="variant !== 'wordmark'"
      class="mark"
      :style="{ width: `${s.mark}px`, height: `${s.mark}px` }"
      aria-hidden="true"
    />
    <span
      v-if="variant !== 'mark'"
      class="wordmark"
      :style="{ fontSize: `${s.font}px`, letterSpacing: s.ls, marginLeft: variant === 'full' ? `${s.gap}px` : '0' }"
    >EXASCALE</span>
  </span>
</template>

<style scoped>
.brand-logo {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
  color: var(--text);
  font-family: var(--font-sans);
  font-weight: 700;
  text-transform: uppercase;
}

.mark {
  display: inline-block;
  background: var(--brand);
  border-radius: 1px;
}

.wordmark {
  line-height: 1;
}
</style>
