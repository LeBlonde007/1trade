<script setup lang="ts">
/**
 * BaseTable — dense data table.
 * Use it via slots:
 *
 * <BaseTable>
 *   <thead><tr><th>Market</th><th class="num">Price</th></tr></thead>
 *   <tbody><tr><td>EAI-IDX</td><td class="num">$0.001005</td></tr></tbody>
 * </BaseTable>
 *
 * Add class="num" on <th> / <td> for right-aligned mono cells.
 */
withDefaults(
  defineProps<{
    /** Compact row height (for Bloomberg-tier density). */
    compact?: boolean
    /** Add hover state to rows. */
    hover?: boolean
  }>(),
  {
    compact: false,
    hover: true,
  },
)
</script>

<template>
  <div class="table-wrap">
    <table :class="{ compact, hover }">
      <slot />
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  width: 100%;
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-base);
}

:deep(th),
:deep(td) {
  text-align: left;
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--border);
  font-weight: 400;
  vertical-align: middle;
}

:deep(thead th) {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: var(--ls-tab);
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 500;
  border-bottom: 1px solid var(--border-strong);
  background: var(--canvas);
}

:deep(td.num),
:deep(th.num) {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
}

:deep(td.mono),
:deep(th.mono) {
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
}

.compact :deep(th),
.compact :deep(td) {
  padding: var(--sp-2) var(--sp-3);
  font-size: var(--fs-sm);
}

.hover :deep(tbody tr):hover {
  background: var(--hover, var(--sunken, transparent));
}
</style>
