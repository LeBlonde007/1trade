# Agent plan — `matching-engine`

> **Paused for Phase 1.** Ships a **mock data adapter** behind the trading API contract so the
> trading-dashboard demo stays alive, and writes the **real engine spec** so Phase 2 switch-on
> is a flag flip, not a redesign.

## 1. Scope under the pivot

The original agent definition stays — the correctness invariants, event sourcing, single-threaded
per-product matching, paper/real isolation, performance targets — but no real engine ships in
Phase 1. What ships:

- A **mock data adapter** that returns simulated quotes / fills / book state behind the
  `openapi/trading.yaml` contract. The trading-frontend demo runs against this.
- The **engine specification** (in `services/matching-engine/SPEC.md`) — full design, tested
  pseudocode, performance budget — so Phase 2 implementation begins from a known-good baseline.
- The **mock→real cutover** plan — exactly which flags flip, which DB tables fill, which event
  subjects start firing.

## 2. Features owned

| Feature | Status |
|---|---|
| [KW03 — Matching engine mock + spec](../features/KW03-matching-engine-mock-and-spec.md) | keep-warm, M1 (mock) + M6 (spec finalized) |
| Real engine implementation | Phase 2 (post-license) |

## 3. Milestone-by-milestone

### Milestone 1 — Mock data adapter
- `services/matching-engine/internal/mock/` ships a generator producing:
  - Plausible candles (Brownian motion w/ mean reversion).
  - Order book depth with power-law size distribution.
  - Trade prints with log-normal trade sizes.
- Adapter implements every `openapi/trading.yaml` read endpoint (`quote`, `orderbook`, `trades`,
  `candles`); write endpoints (`POST /orders`, `cancel`) accept and return a stubbed response
  with a clear "exchange paused" code.
- `is_paper=true` hardcoded for every mock object.

### Milestones 2–5 — Spec maintenance
- `services/matching-engine/SPEC.md` updated as related contracts evolve:
  - Order types (market / limit / IOC / FOK) data shape.
  - Atomic ledger settlement (calls `credit-ledger`).
  - Event emission (trade executed, order state changed).
  - Snapshot + replay strategy (5-minute Postgres snapshots from Redis sorted sets).
- Performance budget worked out: order acceptance P99 <10ms, match P99 <5ms, end-to-end <100ms.
- Property-based test suite drafted (not run yet).

### Milestone 6 — Spec finalized + handoff-ready
- SPEC.md reviewed by `tech-lead` and `security-compliance`.
- Cutover plan documented: which flags flip, what data backfills, what telemetry to watch during
  the first hour of real matching.
- Mock adapter feature-frozen — only bug fixes.

### Phase 2 — Implement
- The actual Go engine: order book in Redis sorted sets, single-threaded per product, price-time
  priority, event-sourced, replayable from event log.
- Wire to `credit-ledger` for atomic settlement.
- Wire to `index-service` (trades become index inputs).
- Wire to `surveillance` (every order/trade triggers evaluation).
- Wire to `market-maker` (no privileged path — MM is just another order source).
- Performance targets met under load tests.

## 4. Contracts owned / consumed

### Owned (Phase 1 — specced; Phase 2 — produced)
- `docs/contracts/openapi/trading.yaml` (consumes; produces for write side).
- `docs/contracts/events/trades.executed.v1.yaml` (specced; produced in Phase 2).
- `docs/contracts/events/orders.state.v1.yaml` (specced; produced in Phase 2).

### Consumed
- `schemas/orders.sql`, `schemas/trades.sql`, `schemas/products.sql` (specced; tables exist for
  mock adapter to write demo state).
- `openapi/credit.yaml` (settlement on fill, Phase 2).

## 5. Local dev

- `services/matching-engine/` runs on `:8007` in mock mode.
- `make seed` populates a mock product catalog (so the trading dashboard has products to render).
- `make trade-demo` runs a synthetic order/trade tape to drive the demo.

## 6. Dockerfile

`deploy/docker/Dockerfile.go-service` with `--build-arg SERVICE=matching-engine`. Same image
serves mock (Phase 1) and real (Phase 2) modes — the difference is config + DB state.

## 7. Deploy

- Phase 1: K8s Deployment, 1 replica (it's serving mock data — no need to scale).
- Phase 2: 1 replica per product (single-threaded matching), with hot standby + leader election.
- Network policy: Phase 2 routing to credit-ledger, index-service, surveillance — all configured
  during keep-warm so the switch-on doesn't need a network-policy review.

## 8. Conventions

- **Every order/trade carries `is_paper`** even in the mock (it's always `true`).
- **Paper and real-money books strictly separate** (Phase 2 — design preserved in mock).
- **Fixed-point math only.**
- **Deterministic ordering** — no map iteration nondeterminism in matching.
- **Mock data must look believable** — investors / traders look at it; if the candles look
  Brownian, fix them.

## 9. Hard boundaries

- Don't own balances (that's `credit-ledger`).
- Don't quote (that's `market-maker`).
- Don't surveil (that's `surveillance`).
- **Don't ship a real engine in Phase 1.** Off-plan.

## 10. Definition of done

### Phase 1
- Mock adapter passes every `openapi/trading.yaml` read endpoint with believable data.
- Write endpoints return a clear "exchange paused" code (HTTP 503 + error code).
- Spec is comprehensive enough that a fresh engineer could implement Phase 2 from it.
- Cutover plan documented and reviewed.

### Phase 2
- Deterministic, replayable, race-free under concurrent load.
- Paper/real isolation enforced.
- Matches the trading contract.
- Property-based tests for matching invariants pass.
- Performance targets met.
