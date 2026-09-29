# KW04 — Market-maker spec

> **Spec-only.** Owner: `market-maker`. Implementation ships in Phase 2.

## Spec

`services/market-maker/SPEC.md` documents the eventual implementation. Phase 1 ships nothing
beyond the spec.

### Pricing logic

- **Base spread**: 1% bid-ask (matches taker fee tier).
- **Fair-value anchor**: midpoint = current private reference index (KW01) ± skew based on
  inventory imbalance.
- **Skew**: if long inventory > 0, lower midpoint; if short, raise it. Formula tunable.
- **Quote refresh cadence**: every 250ms minimum; faster under fast markets.

### Risk controls

- **Max inventory per product** — hard limit; once hit, only one-sided quotes (e.g., long-only
  shrink position).
- **Daily P&L stop-loss** — once daily realized P&L drops below threshold, withdraw all quotes
  until manual override.
- **Volatility-triggered quote withdrawal** — if realized vol over last N minutes exceeds
  threshold, widen or withdraw.
- **Anomalous book detection** — if order book depth drops below floor, withdraw.
- **Manual override** — operator can pause/resume per product.

### Audit

- Every quote update logged with `(product_id, side, price, size, timestamp, reason_for_change)`.
- Operator pause/resume events logged.

### Insider-risk constraint

- MM service runs against a separate internal tenant.
- **Internal accounts cannot trade against customer paper accounts** (insider risk; reviewed by
  `security-compliance`). This is enforced at the `matching-engine` level by checking tenant
  classes on cross.

## Owning agent

`market-maker`.

## Contracts consumed / produced

### Owned (Phase 2)
- None (MM submits to others' APIs).

### Consumes (Phase 2)
- `openapi/trading.yaml`.
- `openapi/index.yaml` — fair-value anchor.
- `events/trades.executed.v1.yaml`.

## Dependencies

- KW03 (matching engine), KW01 (index), KW05 (surveillance interplay).

## Sync points

- M6 — SPEC.md complete; reviewed.

## Acceptance criteria

### Phase 1
- [ ] SPEC.md covers pricing, risk, audit, insider-risk constraint.
- [ ] Test plan written (no tests run yet — no engine to test against).

### Phase 2 (separate scope)
- [ ] Continuous two-sided quotes within target spread.
- [ ] Risk limits enforced and tested.
- [ ] Withdraws cleanly under volatility.
- [ ] All quotes audited.

## Milestone

- M6 (Gate 6): spec finalized.

## Status — paper liquidity account built (2026-09-29)

The internal market maker still may not quote customer paper books (insider-risk rule). Paper
liquidity instead comes from a **separate, non-internal paper liquidity account**
(`matching-engine/internal/liquidity`, owner decision 2026-09-29):

- **Quoting:** a symmetric ladder (3 levels, first at ±0.5% = the 1% base spread, then 0.25% apart,
  at least a tick apart) around the reference price, $250 of paper notional per level. It re-quotes
  on a 0.5% move or when a level is taken.
- **Paper only:**
  - the engine refuses its orders on real books;
  - its value is paper grants (a one-time seed plus hourly-bounded top-ups on refusal);
  - it can't use the HTTP API.
- **Surveillance:** on paper books it is exempt from the quote-pattern rules (layering, spoofing,
  excessive cancellation) only; every trade-based rule still applies (KW05).

The real market maker — skew, inventory limits, stop-loss, volatility withdrawal, quoting on real
books — remains Phase 2 and licence-gated.
