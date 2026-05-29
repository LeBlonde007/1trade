# Help-dot rollout guide

Universal "what is this?" + free-text feedback affordance for every
input, select, chart, and KPI in Exascale.

> Implementation: `app/components/Base/HelpDot.vue` + `app/composables/useFieldHelp.ts`.
> Auto-imports as `<BaseHelpDot />`. Feedback is collected via `useFieldHelp().submit()`
> and surfaced as a global toast on submit.

## What's already wired

- **Trade order form** (`components/Trade/OrderForm.vue`) — Side · Order
  type · Limit price · Quantity.
- **Wallet · Buy credits** (`pages/wallet/buy.vue`) — Amount.
- **Portfolio chart** (`pages/portfolio.vue`) — Compare-to-AI-INDEX
  toggle · time-range tab bar.
- **Billing KPI** (`pages/enterprise/billing.vue`) — Monthly budget cap.

These are reference implementations — copy the pattern.

## How it appears

A 14px circular outlined `!` icon sits next to a label, switch, range
tab, or stat tile. Hover changes color. Click opens a 300px popover
with three sections:

```
┌─ Title ─────────────────────────  X ┐
│                                     │
│ WHAT IT DOES                        │
│ One short paragraph in plain English│
│                                     │
│ WHY IT MATTERS                      │
│ When to use it / when to skip it    │
│                                     │
│ 💬 LEAVE FEEDBACK                   │
│ ┌─────────────────────────────────┐ │
│ │ (textarea, no rating)           │ │
│ │                                 │ │
│ └─────────────────────────────────┘ │
│                ⌘↵ to send · [Submit]│
└─────────────────────────────────────┘
```

Submit → submission stored in localStorage + global toast fires.

## Where to add it (pattern by surface)

### A) On any `<BaseInput>`

Easy mode — just pass three new props:

```vue
<BaseInput
  v-model="value"
  label="Limit price"
  mono
  help-what="The exact USD price per credit at which your limit order will work."
  help-why="Tighter than mid = unlikely to fill. Looser than mid = fills quickly but you pay more."
  help-field="trade.order.limit-px"
/>
```

The dot renders next to the label automatically.

### B) Next to a label / heading / switch / chart control

```vue
<div class="field-head">
  <span class="field-lbl">Order type</span>
  <BaseHelpDot
    title="Order type"
    what="Market = take the best available price right now. Limit = only fill at your specified price or better."
    why="Market is fast but pays the spread + slippage. Limit is patient but may never fill."
    field="trade.order.type"
    placement="right"
  />
</div>
```

### C) Over a KPI tile / chart corner

For charts and dense stat tiles, attach the dot to the tile's label so
it doesn't sit over the data:

```vue
<div class="kpi">
  <div class="kpi-label">
    Budget · monthly cap
    <BaseHelpDot
      title="Monthly budget cap"
      what="The hard ceiling on combined spend across all sub-accounts this month."
      why="At 100% of cap, all non-paying-customer workloads auto-pause unless 'soft cap' is set."
      field="billing.kpi.budget"
      placement="bottom"
    />
  </div>
  <div class="kpi-val">$80,000</div>
  …
</div>
```

### D) On a custom select / segmented control (no <BaseInput>)

```vue
<div class="seg-row">
  <div class="seg">
    <button>Daily</button><button>Weekly</button><button>Cumulative</button>
  </div>
  <BaseHelpDot
    title="Chart aggregation"
    what="Daily = one bar per day. Weekly = bars summed within an ISO week. Cumulative = running total of spend."
    why="Use Daily to spot anomalies, Cumulative to track burn-rate against the budget bar."
    field="billing.chart.agg"
    placement="left"
  />
</div>
```

## Props reference (`<BaseHelpDot>`)

| Prop | Required | Type | Notes |
|---|---|---|---|
| `title` | yes | string | Popover heading + toast subject |
| `what` | yes | string | One-sentence "what it does" in plain English |
| `why` | no | string | One-sentence "why you care / when to use it" |
| `field` | no | string | Stable id (recommended). Persisted with the submission |
| `placement` | no | `'bottom'`*\|`'top'`\|`'left'`\|`'right'` | Popover anchor |
| `tone` | no | `'mute'`*\|`'on-dark'`\|`'on-accent'` | Visual tone of the dot |

`* = default`

## Props reference (`<BaseInput>` additions)

| Prop | Type | Notes |
|---|---|---|
| `helpWhat` | string | Passing this enables the help dot |
| `helpWhy` | string | Optional |
| `helpField` | string | Optional stable id |
| `helpTitle` | string | Optional override (defaults to `label`) |

## Writing good help copy

The dot is the one place we get to explain ourselves. Treat it like
microcopy:

1. **"What it does" in one sentence.** No marketing. No
   "Empowers users to…". Just: *"How much US dollars you want to
   convert into credits in this transaction."*
2. **"Why it matters" gives a heuristic or a tradeoff.** *"Larger orders
   walk the book and pay worse average prices (slippage). Split big
   orders into slices or use a TWAP from Advanced order types."*
3. **Use real numbers when you can.** *"Fee is a flat 1%, so $10K
   top-up costs $100; $100 top-up costs $1."*
4. **Plain English over jargon.** If you can't avoid jargon (Limit, TWAP,
   spread), link to a future glossary.
5. **Honest about limitations.** *"Stop is for risk management. Not
   guaranteed to fill at the stop price if the market gaps."*

## Field-id convention

Use dotted paths that mirror the URL where the field lives. Makes
filtering feedback by surface trivial:

```
trade.order.side
trade.order.type
trade.order.limit-px
trade.order.qty
trade.book.depth-display
trade.tape.filter
portfolio.chart.range
portfolio.chart.compare-idx
portfolio.positions.sort
wallet.buy.amount
wallet.buy.payment-method
wallet.withdraw.destination
billing.kpi.budget
billing.kpi.projected
billing.alert.budget-80
enterprise.audit.export
enterprise.audit.chain-verify
datacenter.register.tier
datacenter.register.pue
inference.playground.model
inference.playground.temperature
compute.new.gpu-type
compute.new.region
```

## Where submissions go

Right now: `localStorage['exa:help-feedback']` (capped to last 200) +
the global toast region.

When you wire a backend, the only change is inside
`composables/useFieldHelp.ts → submit()`:

```ts
function submit(title, text, field) {
  …
  // Replace this:
  savePersisted(submissions.value)
  // With this:
  $fetch('/api/feedback', { method: 'POST', body: entry })
}
```

No component changes needed.

## Surfaces still to wire (suggested priority order)

1. **Trade chart** — timeframe selector (1m/5m/15m/1h/1d), indicator toggle.
2. **Order book** — group-by-price selector, depth slider.
3. **Trade tape** — filter chips.
4. **Wallet buy** — payment-method radio buttons.
5. **Wallet withdraw** — destination picker, fee disclosure.
6. **Inference playground** — model picker, temperature slider, max-tokens.
7. **Compute · new instance** — GPU type, count, region, image, budget cap.
8. **Settings · Notifications** — every channel toggle.
9. **Settings · Trading defaults** — default order type, default size,
   confirm-trade toggle.
10. **Enterprise · onboarding** — each checklist row.
11. **Enterprise · teams** — role picker, budget input.
12. **Enterprise · audit** — date-range filter, export buttons.
13. **Enterprise · billing** — projected number, status pill, all charts.
14. **Datacenter · register** — every form field (Tier rating, PUE,
   certifications, capacity, pricing floor, throttle).
15. **Datacenter · live capacity** — region filter, color thresholds.

That's roughly 90-110 dots across the app. Average per surface: ~5
dots. Each takes about 90 seconds to write good copy for.

## Reviewing collected feedback

To see what users submitted in a session, in the dev console:

```js
JSON.parse(localStorage.getItem('exa:help-feedback'))
```

Returns an array of `{ id, title, field, route, text, ts }` entries.
A `/settings#feedback` view (M1 in v1.5 catalog) can render this as
a table when you build that section.
