<script setup lang="ts">
/**
 * TradeOrderBook — 10-level bid/ask ladder with depth bars.
 * Power-law depth, jitters live every 1.5 seconds.
 */
interface Level { price: number; size: number; total: number }

const asks = ref<Level[]>([])
const bids = ref<Level[]>([])
const mid = ref(0.001005)
const spread = ref(0.00002)

const seedLevels = (basePrice: number, side: 'ask' | 'bid'): Level[] => {
  const levels: Level[] = []
  let cumulative = 0
  for (let i = 0; i < 10; i++) {
    const step = (i + 1) * 0.0000005 * (1 + Math.random() * 0.3)
    const price = side === 'ask' ? basePrice + step : basePrice - step
    // Power-law size: small near top, large deeper
    const size = Math.round(2000 + Math.pow(i + 1, 1.6) * (3000 + Math.random() * 4000))
    cumulative += size
    levels.push({ price, size, total: cumulative })
  }
  // Asks ascending from best, but we'll render descending so best is at bottom
  return side === 'ask' ? levels.reverse() : levels
}

const maxTotal = computed(() => {
  const last = (arr: Level[]) => (arr[arr.length - 1]?.total ?? arr[0]?.total ?? 0)
  // For asks (reversed), the largest total is at the top (last in display order)
  return Math.max(last(asks.value), last(bids.value), 1)
})

const updateBook = () => {
  asks.value = seedLevels(mid.value + spread.value / 2, 'ask')
  bids.value = seedLevels(mid.value - spread.value / 2, 'bid')
}

let interval: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  updateBook()
  interval = setInterval(() => {
    // Slight mid drift
    mid.value += (Math.random() - 0.5) * 0.0000008
    updateBook()
  }, 1500)
})
onUnmounted(() => { if (interval) clearInterval(interval) })

const fmt = (n: number) => n.toFixed(6)
const fmtSize = (n: number) => Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 }).format(n)
</script>

<template>
  <div class="ob">
    <div class="ob-head">
      <span>Price</span>
      <span class="num">Size</span>
      <span class="num">Total</span>
    </div>

    <div class="ob-side asks">
      <div v-for="lvl in asks" :key="`a-${lvl.price}`" class="row ask">
        <span class="bar" :style="{ width: `${(lvl.total / maxTotal) * 100}%` }" />
        <span class="px">{{ fmt(lvl.price) }}</span>
        <span class="sz">{{ fmtSize(lvl.size) }}</span>
        <span class="tt">{{ fmtSize(lvl.total) }}</span>
      </div>
    </div>

    <div class="mid">
      <span class="mid-px mono">{{ fmt(mid) }}</span>
      <span class="mid-spread">Spread {{ (spread * 1e6).toFixed(1) }} bp · {{ ((spread / mid) * 100).toFixed(2) }}%</span>
    </div>

    <div class="ob-side bids">
      <div v-for="lvl in bids" :key="`b-${lvl.price}`" class="row bid">
        <span class="bar" :style="{ width: `${(lvl.total / maxTotal) * 100}%` }" />
        <span class="px">{{ fmt(lvl.price) }}</span>
        <span class="sz">{{ fmtSize(lvl.size) }}</span>
        <span class="tt">{{ fmtSize(lvl.total) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ob {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text);
}

.ob-head {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-3);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
  border-bottom: 1px solid var(--border);
}

.ob-head .num { text-align: right; }

.row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: var(--sp-3);
  padding: 3px var(--sp-3);
  position: relative;
  font-variant-numeric: tabular-nums;
}

.bar {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  pointer-events: none;
}

.row.ask .bar { background: var(--neg-bar, rgba(239, 68, 68, 0.14)); }
.row.bid .bar { background: var(--pos-bar, rgba(25, 195, 125, 0.14)); }

.row > *:not(.bar) {
  position: relative;
  z-index: 1;
}

.row.ask .px { color: var(--neg); }
.row.bid .px { color: var(--pos); }

.sz,
.tt { text-align: right; color: var(--text-2); }

.mid {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: var(--sp-3);
  padding: var(--sp-3);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}

.mid-px {
  font-size: var(--fs-md);
  font-weight: 600;
  color: var(--text);
}

.mid-spread {
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-tab);
  color: var(--text-3);
}
</style>
