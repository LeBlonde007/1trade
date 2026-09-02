<script setup lang="ts">
/**
 * PositionPanel — Open positions table for the bottom of the trade dashboard.
 */
interface Pos { market: string; side: 'long' | 'short'; size: number; avgEntry: number; mark: number }

const positions: Pos[] = [
  { market: 'EAI-IDX',    side: 'long',  size: 50_000, avgEntry: 0.000980, mark: 0.001005 },
  { market: 'TEXT-SPOT',  side: 'long',  size: 12_000, avgEntry: 0.00118,  mark: 0.00120 },
  { market: 'IMAGE-SPOT', side: 'short', size: 5_000,  avgEntry: 0.00810,  mark: 0.00798 },
  { market: 'H100-SPOT',  side: 'long',  size: 8,      avgEntry: 2.95,     mark: 2.99 },
]

const pnl = (p: Pos): number => {
  const diff = p.mark - p.avgEntry
  const mult = p.side === 'long' ? 1 : -1
  return diff * p.size * mult
}

const pnlPct = (p: Pos): number => {
  const diff = p.mark - p.avgEntry
  const mult = p.side === 'long' ? 1 : -1
  return (diff / p.avgEntry) * mult
}

const totalPnL = computed(() => positions.reduce((s, p) => s + pnl(p), 0))
const totalValue = computed(() => positions.reduce((s, p) => s + p.mark * p.size, 0))
</script>

<template>
  <BaseCard density="tight">
    <template #header>
      <div class="head-row">
        <span>Positions · {{ positions.length }} open</span>
        <span class="head-meta">
          P&L today
          <PriceDelta :value="totalPnL" :pct="totalPnL / totalValue" format="usd" />
          · Total
          <span class="mono">{{ formatUSD(totalValue) }}</span>
        </span>
      </div>
    </template>

    <BaseTable compact>
      <thead>
        <tr>
          <th>Market</th>
          <th>Side</th>
          <th class="num">Size</th>
          <th class="num">Avg Entry</th>
          <th class="num">Mark</th>
          <th class="num">P&L $</th>
          <th class="num">P&L %</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in positions" :key="p.market">
          <td class="sym">{{ p.market }}</td>
          <td>
            <BaseBadge :tone="p.side === 'long' ? 'positive' : 'negative'" variant="filled">
              {{ p.side }}
            </BaseBadge>
          </td>
          <td class="num">{{ formatCredits(p.size) }}</td>
          <td class="num">{{ p.avgEntry.toFixed(6) }}</td>
          <td class="num">{{ p.mark.toFixed(6) }}</td>
          <td class="num">
            <PriceDelta :value="pnl(p)" format="usd" />
          </td>
          <td class="num">
            <PriceDelta :pct="pnlPct(p)" />
          </td>
          <td class="actions">
            <BaseButton variant="tertiary" size="sm">Close</BaseButton>
          </td>
        </tr>
      </tbody>
    </BaseTable>
  </BaseCard>
</template>

<style scoped>
.head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
}

.head-meta {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-3);
  font-family: var(--font-mono);
  text-transform: none;
  letter-spacing: 0;
  font-size: var(--fs-sm);
  color: var(--text-2);
}

.head-meta .mono { color: var(--text); }

.sym {
  font-family: var(--font-mono);
  font-weight: 600;
}

.actions {
  text-align: right;
}
</style>
