<script setup lang="ts">
/**
 * Sparkline — tiny inline trend chart drawn with pure SVG.
 * Auto-detects up/down direction and colors accordingly.
 *
 * <Sparkline :data="[1, 1.02, 1.01, 1.05, 1.08]" :width="200" :height="40" />
 */
const props = withDefaults(
  defineProps<{
    /** Time-ordered numeric series. */
    data: number[]
    width?: number
    height?: number
    /** Override line color. Defaults to direction-based semantic color. */
    color?: string
    /** Fill area below the line (subtle). */
    filled?: boolean
  }>(),
  {
    width: 160,
    height: 40,
    filled: true,
  },
)

const path = computed(() => {
  if (!props.data || props.data.length < 2) return ''
  const min = Math.min(...props.data)
  const max = Math.max(...props.data)
  const range = max - min || 1
  const stepX = props.width / (props.data.length - 1)

  return props.data
    .map((v, i) => {
      const x = i * stepX
      const y = props.height - ((v - min) / range) * props.height
      return `${i === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
    })
    .join(' ')
})

const fillPath = computed(() => {
  if (!path.value) return ''
  return `${path.value} L ${props.width} ${props.height} L 0 ${props.height} Z`
})

const direction = computed<'up' | 'down' | 'flat'>(() => {
  if (props.data.length < 2) return 'flat'
  const first = props.data[0]
  const last = props.data[props.data.length - 1]
  if (last > first) return 'up'
  if (last < first) return 'down'
  return 'flat'
})

const strokeColor = computed(() => {
  if (props.color) return props.color
  if (direction.value === 'up') return 'var(--pos)'
  if (direction.value === 'down') return 'var(--neg)'
  return 'var(--text-2)'
})
</script>

<template>
  <svg
    :width="width"
    :height="height"
    :viewBox="`0 0 ${width} ${height}`"
    class="sparkline"
    preserveAspectRatio="none"
    aria-hidden="true"
  >
    <path
      v-if="filled"
      :d="fillPath"
      :fill="strokeColor"
      opacity="0.12"
    />
    <path
      :d="path"
      :stroke="strokeColor"
      stroke-width="1.5"
      fill="none"
      stroke-linecap="round"
      stroke-linejoin="round"
      vector-effect="non-scaling-stroke"
    />
  </svg>
</template>

<style scoped>
.sparkline {
  display: block;
}
</style>
