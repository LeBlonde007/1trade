<script setup lang="ts">
/**
 * TradeTape — streaming trades, newest on top.
 * Log-normal trade sizes, slight buy/sell skew.
 */
interface Trade { id: number; time: string; price: number; size: number; side: 'buy' | 'sell' }

const trades = ref<Trade[]>([])
const MAX_TRADES = 40
let nextId = 0

const genTrade = (basePrice: number): Trade => {
  const isBuy = Math.random() > 0.48
  const sz = Math.round(Math.exp(Math.log(500) + (Math.random() - 0.5) * 2.5))
  const price = basePrice + (Math.random() - 0.5) * 0.000002
  const d = new Date()
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return {
    id: ++nextId,
    time: `${hh}:${mm}:${ss}.${ms}`,
    price,
    size: sz,
    side: isBuy ? 'buy' : 'sell',
  }
}

let interval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  // Seed initial trades
  trades.value = Array.from({ length: 25 }, () => genTrade(0.001005))

  // Live tape — 1–3 new trades every couple seconds
  interval = setInterval(() => {
    const n = 1 + Math.floor(Math.random() * 3)
    for (let i = 0; i < n; i++) {
      trades.value.unshift(genTrade(0.001005))
    }
    if (trades.value.length > MAX_TRADES) trades.value.length = MAX_TRADES
  }, 1800)
})
onUnmounted(() => { if (interval) clearInterval(interval) })
</script>

<template>
  <div class="tape">
    <div class="head">
      <span>Time</span>
      <span class="num">Price</span>
      <span class="num">Size</span>
    </div>
    <div class="rows">
      <TransitionGroup name="tape">
        <div v-for="t in trades" :key="t.id" class="row" :class="`side-${t.side}`">
          <span class="time">{{ t.time }}</span>
          <span class="px num">{{ t.price.toFixed(6) }}</span>
          <span class="sz num">{{ t.size.toLocaleString() }}</span>
          <span class="dir">{{ t.side === 'buy' ? '▲' : '▼' }}</span>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.tape {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.head {
  display: grid;
  grid-template-columns: 1.1fr 1fr 1fr 16px;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
  border-bottom: 1px solid var(--border);
}

.head .num { text-align: right; }

.rows {
  flex: 1;
  overflow: auto;
}

.row {
  display: grid;
  grid-template-columns: 1.1fr 1fr 1fr 16px;
  gap: var(--sp-2);
  padding: 3px var(--sp-3);
  align-items: center;
  font-variant-numeric: tabular-nums;
  border-bottom: 1px solid var(--border);
}

.row .time { color: var(--text-3); }
.row .num { text-align: right; }
.row .dir { text-align: center; font-size: 9px; }
.side-buy .px,
.side-buy .dir { color: var(--pos); }
.side-sell .px,
.side-sell .dir { color: var(--neg); }

.tape-enter-active { transition: all 200ms var(--ease); }
.tape-enter-from { opacity: 0; transform: translateY(-6px); }
</style>
