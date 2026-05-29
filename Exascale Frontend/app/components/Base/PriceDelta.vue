<script setup lang="ts">
/**
 * PriceDelta — colored ±value with directional glyph.
 * Used dozens of times across the app — always consistent.
 *
 * <PriceDelta :value="23.41" :pct="0.0023" format="usd" />
 * <PriceDelta :pct="-0.0042" />
 *
 * Per the brand book: always pair color WITH shape (▲/▼) for color-blindness.
 */
const props = withDefaults(
  defineProps<{
    /** Numeric change (e.g. dollars). If omitted, only pct is shown. */
    value?: number
    /** Percentage change as decimal: 0.0235 = 2.35%. */
    pct?: number
    /** How to format the numeric value. */
    format?: 'usd' | 'price' | 'credits' | 'plain'
    /** Show the directional glyph (▲ / ▼). */
    glyph?: boolean
    /** Show the explicit + sign for positives. */
    showSign?: boolean
    /** Treat zero as neutral text (default true). */
    neutralZero?: boolean
    /** Number of decimals if format=price. */
    priceDecimals?: number
  }>(),
  {
    format: 'plain',
    glyph: true,
    showSign: true,
    neutralZero: true,
    priceDecimals: 6,
  },
)

// Decide the direction sign from either value or pct
const signSource = computed(() => {
  if (props.value !== undefined) return props.value
  if (props.pct !== undefined) return props.pct
  return 0
})

const dir = computed<'up' | 'down' | 'flat'>(() => {
  if (signSource.value > 0) return 'up'
  if (signSource.value < 0) return 'down'
  return 'flat'
})

const formattedValue = computed(() => {
  if (props.value === undefined) return ''
  switch (props.format) {
    case 'usd':     return formatUSD(props.value, { showSign: props.showSign })
    case 'price':   return formatPrice(props.value, props.priceDecimals)
    case 'credits': return formatCredits(props.value)
    case 'plain':
    default:        return props.value.toString()
  }
})

const formattedPct = computed(() => {
  if (props.pct === undefined) return ''
  return formatPct(props.pct, { showSign: props.showSign })
})

const glyphChar = computed(() => {
  if (!props.glyph) return ''
  if (dir.value === 'up') return '▲'
  if (dir.value === 'down') return '▼'
  return '•'
})
</script>

<template>
  <span
    class="delta"
    :class="[`dir-${dir}`, { neutral: dir === 'flat' && neutralZero }]"
  >
    <span v-if="glyphChar" class="glyph">{{ glyphChar }}</span>
    <span v-if="formattedValue" class="val">{{ formattedValue }}</span>
    <span v-if="formattedValue && formattedPct" class="sep"> · </span>
    <span v-if="formattedPct" class="pct">{{ formattedPct }}</span>
  </span>
</template>

<style scoped>
.delta {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-weight: 500;
  white-space: nowrap;
}

.dir-up   { color: var(--pos); }
.dir-down { color: var(--neg); }
.neutral  { color: var(--text-2); }

.glyph {
  font-size: 0.85em;
  line-height: 1;
}

.sep {
  color: currentColor;
  opacity: 0.5;
}
</style>
