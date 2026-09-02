<script setup lang="ts">
/**
 * BrandLogo — the canonical 1TRADE mark + wordmark.
 * Auto-imported as <BrandLogo />.
 *
 * The mark is inlined (not an <img>) so it inherits `currentColor` and can be
 * recoloured per surface — gold on midnight, bronze on cream — from tokens alone.
 * `faceted` swaps in the brand guide's bevelled gold treatment for hero use.
 */
const props = withDefaults(
  defineProps<{
    /** 'full' = mark + wordmark, 'mark' = mark only, 'wordmark' = text only. */
    variant?: 'full' | 'mark' | 'wordmark'
    /** Size scalar. */
    size?: 'sm' | 'md' | 'lg' | 'xl'
    /** Use the bevelled gold facets instead of a flat currentColor fill. */
    faceted?: boolean
  }>(),
  {
    variant: 'full',
    size: 'md',
    faceted: false,
  },
)

/** The mark's intrinsic aspect (viewBox 431.54 x 1000) — width per unit of height. */
const ASPECT = 0.43154

const sizeMap = {
  sm: { mark: 20, font: 12, gap: 9 },
  md: { mark: 26, font: 14, gap: 11 },
  lg: { mark: 34, font: 18, gap: 14 },
  xl: { mark: 56, font: 28, gap: 20 },
} as const

const s = computed(() => sizeMap[props.size])

/** Mark width derived from its height so the glyph is never distorted. */
const markW = computed(() => +(s.value.mark * ASPECT).toFixed(2))

/** Unique gradient ids — several logos can share a page without id collisions. */
const uid = useId()

/** The mark as three subpaths of one path: nonzero fill unions the bevel facets
 *  without the hairline seams separate elements would show. */
const FLAT
  = 'M3.49 1000L3.49 471.67C69 441.18 134.92 402.71 169.17 342.89L169.17 764.06L249.44 '
  + '709.93L262.22 676.81L296.26 707.21L431.54 707.19ZM262.61 235.76L428.2 235.76L428.2 '
  + '535.66C363.03 563.5 298.6 599.74 262.61 657.06ZM428.2 0L428.2 235.76L262.61 '
  + '235.76L182.43 290.45L169.89 323.36L136.2 292.95L0 292.93Z'

const BASE = 'M3.49 1000L3.49 471.67C69 441.18 134.92 402.71 169.17 342.89L169.17 764.06L249.44 '
  + '709.93L262.22 676.81L296.26 707.21L431.54 707.19Z'
const STEM = 'M262.61 235.76L428.2 235.76L428.2 535.66C363.03 563.5 298.6 599.74 262.61 657.06Z'
const FLAG = 'M428.2 0L428.2 235.76L262.61 235.76L182.43 290.45L169.89 323.36L136.2 292.95L0 292.93Z'
</script>

<template>
  <span class="brand-logo" :data-variant="variant">
    <svg
      v-if="variant !== 'wordmark'"
      class="mark"
      :width="markW"
      :height="s.mark"
      viewBox="0 0 431.54 1000"
      role="img"
      :aria-label="variant === 'mark' ? '1TRADE' : undefined"
      :aria-hidden="variant === 'mark' ? undefined : 'true'"
      focusable="false"
    >
      <template v-if="faceted">
        <defs>
          <linearGradient :id="`${uid}-fl`" gradientUnits="userSpaceOnUse" x1="215" y1="0" x2="215" y2="1000">
            <stop offset="0" stop-color="#E8CD75" /><stop offset="1" stop-color="#C9A233" />
          </linearGradient>
          <linearGradient :id="`${uid}-st`" gradientUnits="userSpaceOnUse" x1="215" y1="0" x2="215" y2="1000">
            <stop offset="0" stop-color="#C9A544" /><stop offset="1" stop-color="#A87833" />
          </linearGradient>
          <linearGradient :id="`${uid}-ba`" gradientUnits="userSpaceOnUse" x1="215" y1="0" x2="215" y2="1000">
            <stop offset="0" stop-color="#A67433" /><stop offset="1" stop-color="#6E4419" />
          </linearGradient>
        </defs>
        <path :fill="`url(#${uid}-ba)`" :d="BASE" />
        <path :fill="`url(#${uid}-st)`" :d="STEM" />
        <path :fill="`url(#${uid}-fl)`" :d="FLAG" />
      </template>
      <path v-else fill="currentColor" fill-rule="nonzero" :d="FLAT" />
    </svg>

    <span
      v-if="variant !== 'mark'"
      class="wordmark"
      :style="{
        fontSize: `${s.font}px`,
        marginLeft: variant === 'full' ? `${s.gap}px` : '0',
      }"
    >1TRADE</span>
  </span>
</template>

<style scoped>
.brand-logo {
  display: inline-flex;
  align-items: center;
  white-space: nowrap;
  font-family: var(--font-brand);
  font-weight: 700;
  text-transform: uppercase;
}

/* Gold on dark, bronze on light — the themed brand token does the switching. */
.mark {
  display: block;
  flex: none;
  color: var(--brand);
}

.wordmark {
  line-height: 1;
  letter-spacing: var(--ls-brand);
  color: var(--text);
}
</style>
