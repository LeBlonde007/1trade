<script setup lang="ts">
/**
 * TradeOrderForm — buy/sell entry with market/limit order types.
 * Live total preview, fee calculation.
 */
const side = ref<'buy' | 'sell'>('buy')
const type = ref<'market' | 'limit'>('limit')
const price = ref('0.001005')
const quantity = ref('1000')

const numPrice = computed(() => parseFloat(price.value) || 0)
const numQty   = computed(() => parseFloat(quantity.value) || 0)
const subtotal = computed(() => numPrice.value * numQty.value)
const fee      = computed(() => subtotal.value * 0.01)
const total    = computed(() => subtotal.value + (side.value === 'buy' ? fee.value : -fee.value))

const balanceUSD = 10247.83
const buyingPower = computed(() => balanceUSD / numPrice.value)

const setQtyPct = (pct: number) => {
  quantity.value = Math.floor(buyingPower.value * pct).toString()
}
</script>

<template>
  <div class="order-form">
    <!-- Buy / Sell tabs -->
    <div class="field-head">
      <span class="field-lbl">Side</span>
      <BaseHelpDot
        title="Side — Buy or Sell"
        what="Pick Buy to acquire credits at the current ask. Pick Sell to offload credits you hold at the current bid."
        why="On a thin book the two sides can have very different effective prices. Sell into the bid is instant; Buy into the ask is instant. A limit order on either side waits for the price to come to you."
        field="trade.order.side"
        placement="right"
      />
    </div>
    <div class="side-tabs">
      <button
        type="button"
        class="tab tab-buy"
        :class="{ active: side === 'buy' }"
        @click="side = 'buy'"
      >Buy</button>
      <button
        type="button"
        class="tab tab-sell"
        :class="{ active: side === 'sell' }"
        @click="side = 'sell'"
      >Sell</button>
    </div>

    <!-- Order type -->
    <div class="field-head">
      <span class="field-lbl">Order type</span>
      <BaseHelpDot
        title="Order type"
        what="Market = take the best available price right now. Limit = only fill at your specified price or better. Stop = trigger a market order once price crosses your stop level."
        why="Market is fast but pays the spread + slippage. Limit is patient but may never fill. Stop is for risk management — set it once and walk away."
        field="trade.order.type"
        placement="right"
      />
    </div>
    <div class="type-seg">
      <button type="button" :class="{ active: type === 'market' }" @click="type = 'market'">Market</button>
      <button type="button" :class="{ active: type === 'limit' }" @click="type = 'limit'">Limit</button>
      <button type="button" disabled>Stop</button>
    </div>

    <!-- Price -->
    <BaseInput
      v-if="type === 'limit'"
      v-model="price"
      label="Price (USD)"
      mono
      suffix="USD"
      help-what="The exact USD price per credit at which your limit order will work. Order only fills at this price or better."
      help-why="Tighter than mid = unlikely to fill. Looser than mid = fills quickly but you pay more. Most traders place limits 1-3 bps inside the spread."
      help-field="trade.order.limit-px"
    />

    <!-- Quantity -->
    <BaseInput
      v-model="quantity"
      label="Quantity"
      mono
      suffix="credits"
      help-what="How many credits to buy or sell in this order. Quick-fill chips below let you spend 25 / 50 / 75 / 100% of your buying power."
      help-why="Larger orders walk the book and pay worse average prices (slippage). On thin markets, split big orders into slices or use a TWAP from Advanced order types."
      help-field="trade.order.qty"
    />

    <!-- Quick fills -->
    <div class="quick">
      <button type="button" @click="setQtyPct(0.25)">25%</button>
      <button type="button" @click="setQtyPct(0.50)">50%</button>
      <button type="button" @click="setQtyPct(0.75)">75%</button>
      <button type="button" @click="setQtyPct(1.00)">100%</button>
    </div>

    <!-- Summary -->
    <dl class="summary">
      <div>
        <dt>{{ type === 'limit' ? 'Price' : 'Est. price' }} × Qty</dt>
        <dd class="mono">{{ formatUSD(subtotal) }}</dd>
      </div>
      <div>
        <dt>Fee 1.00%</dt>
        <dd class="mono">{{ formatUSD(fee) }}</dd>
      </div>
      <div class="total-row">
        <dt><strong>Total</strong></dt>
        <dd class="mono"><strong>{{ formatUSD(total) }}</strong></dd>
      </div>
    </dl>

    <p class="balance">
      Available <span class="mono">{{ formatUSD(balanceUSD) }}</span> · buying power
      <span class="mono">{{ formatCredits(buyingPower) }} credits</span>
    </p>

    <BaseButton
      :variant="side === 'buy' ? 'buy' : 'sell'"
      size="lg"
      full
    >
      {{ side === 'buy' ? 'Buy' : 'Sell' }} {{ formatCredits(numQty) }} credits · {{ formatUSD(total) }}
    </BaseButton>
  </div>
</template>

<style scoped>
.order-form {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-4);
}

.field-head {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: -4px 0 -4px;
}
.field-lbl {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
}

.side-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.tab {
  padding: var(--sp-3);
  font-weight: 600;
  font-size: var(--fs-sm);
  color: var(--text-2);
  background: transparent;
  cursor: pointer;
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
}

.tab-buy.active { background: var(--pos); color: #fff; }
.tab-sell.active { background: var(--neg); color: #fff; }

.type-seg {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.type-seg button {
  padding: var(--sp-2);
  font-size: var(--fs-sm);
  color: var(--text-2);
  cursor: pointer;
  background: transparent;
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
}

.type-seg button:hover:not(:disabled) { color: var(--text); }
.type-seg button.active {
  background: var(--canvas);
  color: var(--brand);
}

.quick {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-1);
}

.quick button {
  padding: 6px 0;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-2);
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: border-color var(--dur) var(--ease);
}

.quick button:hover { border-color: var(--border-strong); color: var(--text); }

.summary {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
  margin: 0;
  padding: var(--sp-3) 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  font-size: var(--fs-sm);
}

.summary > div {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: var(--sp-4);
}

.summary dt { color: var(--text-2); }
.summary dd { margin: 0; color: var(--text); text-align: right; font-variant-numeric: tabular-nums; }

.total-row {
  padding-top: var(--sp-1);
  margin-top: var(--sp-1);
  border-top: 1px solid var(--border);
}

.balance {
  font-size: var(--fs-xs);
  color: var(--text-3);
  margin: 0;
  line-height: var(--lh-relax);
}

.balance .mono { color: var(--text); }
</style>
