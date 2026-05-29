# KW03 — Matching engine mock adapter + real-engine spec

> **Mock adapter ships M1; spec finalized M6.** Owner: `matching-engine`. Real engine ships in
> Phase 2.

## Spec

### Mock adapter (Phase 1)

`services/matching-engine/` exists, but only the mock adapter is live:

- Implements every `openapi/trading.yaml` **read** endpoint with believable simulated data:
  - `GET /v1/trading/products` — fixed product set (AI index, sub-credit spots, GPU spots).
  - `GET /v1/trading/products/{id}/quote` — 2-sided mock quote with 1% spread.
  - `GET /v1/trading/products/{id}/orderbook` — power-law depth.
  - `GET /v1/trading/products/{id}/trades` — log-normal sizes, Brownian price walk.
  - `GET /v1/trading/products/{id}/candles?interval=1m` — multi-interval candle generator.
- Implements **write** endpoints (`POST /orders`, `cancel`) as stubs returning HTTP 503 with
  error code `EXCHANGE_PAUSED` + message linking to the methodology page.
- Every mock object has `is_paper=true`.

### Real-engine spec (`SPEC.md`)

Lives at `services/matching-engine/SPEC.md`. Comprehensive enough for a fresh engineer to
implement in Phase 2:

- Order book in Redis sorted sets (bid/ask, sorted by price; secondary by time).
- Single-threaded matching per product (deterministic; v1).
- Price-time priority.
- Order types: market, limit, IOC, FOK.
- Event sourcing — every state change written to an event log; engine replayable from log + last
  snapshot.
- 5-minute Postgres snapshots from Redis state.
- Atomic settlement: on match, call `credit-ledger` to deduct/add atomically; emit
  `events/trades.executed.v1.yaml`.
- Surveillance hook: emit order/trade events to `events/orders.state.v1.yaml` and
  `events/trades.executed.v1.yaml`.
- Performance budget:
  - Order acceptance P99 <10ms.
  - Match P99 <5ms.
  - End-to-end confirmation P99 <100ms.

### Cutover plan

- All Phase 2 contracts pre-published in M1; mock adapter consumes them.
- Switch-on procedure: deploy real engine alongside mock; redirect Kong routes to real engine;
  remove `EXCHANGE_PAUSED` stub; remove ribbon in `apps/web/`.
- Surveillance + market-maker must be deployed *before* the switch-on.

## Owning agent

`matching-engine`.

## Contracts consumed / produced

### Produces (Phase 1: specced; Phase 2: produced)
- `openapi/trading.yaml`.
- `events/trades.executed.v1.yaml`.
- `events/orders.state.v1.yaml`.

### Consumes
- `openapi/credit.yaml` (Phase 2 — settle on fill).
- `schemas/products.sql`, `schemas/orders.sql`, `schemas/trades.sql`.

## Dependencies

- F01.
- Phase 2 — F05 (ledger settle-on-fill), KW05 (surveillance), KW04 (market-maker).

## Sync points

- M1 — mock adapter live; trading-frontend `/trade` demos against it.
- M6 — SPEC.md finalized; reviewed by `tech-lead` + `security-compliance`.

## Acceptance criteria

### Phase 1
- [ ] Mock read endpoints return believable data (Brownian candles, power-law depth, log-normal trades).
- [ ] Write endpoints return 503 with `EXCHANGE_PAUSED`.
- [ ] Spec covers every Phase 2 invariant.
- [ ] Cutover plan documented and reviewable.

### Phase 2 (separate scope)
- [ ] Deterministic, replayable, race-free under concurrent load.
- [ ] Paper/real isolation enforced.
- [ ] Matches the trading contract; property-based tests pass.
- [ ] Performance targets met.

## Milestone

- M1 (Gate 1): mock adapter live.
- M6 (Gate 6): spec finalized.
