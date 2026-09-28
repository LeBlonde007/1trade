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

---

## Status — real engine core built; order entry still closed (2026-09-27)

`services/matching-engine/internal/engine/` is the real matching engine. It is pure, deterministic
Go with no clock, network or randomness inside. It covers:

- one book per `(product, is_paper)`, with price-time priority at the maker's price;
- market, limit (gtc/day), IOC and FOK orders;
- self-trade prevention (cancel newest) and the insider-risk rule (no internal orders on paper
  books);
- fixed-point `int64` micro-units, with overflow-checked notional;
- taker/maker fees, a pre-trade risk seam, idempotent submits, and tenant-scoped cancel;
- day expiry and a hash-chained trade log;
- a command journal that `Replay` reproduces exactly.

`services/matching-engine/SPEC.md` is written.

**Evidence.** Scenario tests cover each order type and rule, and every validation error. Property
tests run 200 random streams of 400 commands each; they check no crossed book, no self or
paper/internal trades, prices within limits, filled-quantity conservation, strictly increasing
sequences, and valid trade chains. Replay determinism is checked after a JSON round trip, and there
is a concurrent `-race` run. The suite was mutation-checked: breaking self-trade prevention or the
execution price fails it.

**Durable journal — done (2026-09-27).** `internal/journal` plus `migrations/0001_journal.sql` give a
write-ahead, append-only, hash-chained Postgres journal. `journal.Recover` verifies it and rebuilds
the engine on start. Integration-tested against real Postgres: crash and recover, tamper and gap
detection, second-writer refusal, and refusing commands while the DB is down. Cost is about 0.4 ms
per order locally. It is not deployed yet: the service opens no database until order entry is wired.

**Event encoding — done (2026-09-27).** `internal/events` encodes `orders.state.v1` and
`trades.executed.v1`. It is validated against the contract YAML schemas, and the trade chain can be
re-verified from the payloads alone.

**Settlement — ledger live, engine client done (2026-09-27).**
- The ledger implements `credit.yaml` v1.1 (F05).
- `internal/settle` settles engine trades with the engine's own token.
- The request body is pinned to the contract schema, and outcomes are split into settled /
  unsettleable / conflict / transient.
- A cross-service test settles a real engine trade on the real ledger binary. Balances are exact,
  a replay is a no-op, and the shared token is refused.

**Reservations: done end to end (2026-09-27).**
- The engine computes each order's hold and reserves it through the ledger as its risk check, which
  fails closed.
- The in-order worker settles each trade, then releases leftovers.
- Trade ids carry a per-journal epoch.
- The cross-service lifecycle test against the real ledger shows:
  - oversized and double-spend orders are rejected;
  - a price-improved fill settles out of its reservation;
  - every leftover is released, ending with nothing locked.

**Event publishing: done (2026-09-27).** The journal is the outbox. A relay re-derives events on a
shadow engine and publishes them to JetStream in order: at-least-once, with dedupe ids and a Postgres
cursor. Proven end to end on real Postgres and JetStream.

**Orphaned reservations: closed on the engine side (2026-09-28, SPEC §7.3).** A submit whose journal
write fails after the ledger reserved its hold used to leave that reservation locked, with no order.
- The engine now reports such submits, and a reconciler voids the order_id after a grace period.
  The void is a journaled tombstone, so no retry can claim the reservation. The reconciler then
  releases it.
- The reserve client refuses a replayed reservation that is no longer open.
- Proven with a stress run (every live order funded, nothing orphaned) and against the real ledger.
- Crash-safe (credit.yaml v1.3): the reconciler also sweeps the ledger's open reservations, so an
  orphan whose suspect died with the process is still found. Releases carry an audit reason.
  Invariant: one engine journal per ledger (SPEC §7.3).

Remaining: only the licence-gated cutover wiring (SPEC §7.3):
- DATABASE_URL and the migrations;
- `journal.Recover` with `ReserveRisk`, plus the reservation reconciler;
- running the relay and the settlement worker;
- serving order entry from the engine.

**Not done** (SPEC.md §7): snapshots, settlement plus NATS publishing (contract now authored), the
real risk hook (can now check the buyer's paper cash), and API wiring. The API wiring is the
licence-gated cutover.

**Open for tech-lead** (SPEC.md §8):
- sequence scope per `(product, is_paper)`;
- how paper liquidity works given the insider-risk rule (blocks KW04 paper quoting);
- enumerating cancel and reject reasons;
- ~~the credit.yaml v1.3 reservation listing~~ (authored and built 2026-09-28).

Acceptance (Phase 2): ✅ deterministic, replayable, race-free under concurrent load · ✅ paper/real
isolation enforced · ✅ property-based tests pass · ⬜ matches the trading contract end to end (API
not wired) · ⬜ performance targets measured.
