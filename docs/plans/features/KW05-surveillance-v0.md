# KW05 — Surveillance v0 + spec for the 6 patterns

> **Basic abuse / consumption-anomaly monitoring ships M1.** Full 6-pattern trade surveillance is
> specced only. Owner: `surveillance`.

## Spec

### Phase 1 (active — basic abuse only)

- Rate limits per tenant per endpoint (config from `platform-core`).
- Anomalous consumption alerting: a tenant's GPU-hours-per-hour jumps 10× → page.
- Reservation create/cancel ratios — excessive cancellation on booking ops.
- Auth abuse signals: failed-login storms, API-key burst usage from new IP.
- Schema: `surveillance_alerts` (per Phase 6 §9.3).

### Phase 2 (specced — trade surveillance)

Six patterns implementing per Phase 6 §8.5:

| Pattern | Description | Action |
|---|---|---|
| Wash trade | Same beneficial owner on both sides | Flag; review |
| Spoofing | Large orders frequently canceled | Alert; rate-limit |
| Layering | Multiple staggered orders creating false depth | Alert; investigate |
| Marking the close | Concentrated activity at index calc window | Flag → exclude from index |
| Cross-product manipulation | Spot manipulation affecting forwards | Cross-product surveillance |
| Excessive cancellation | Order-to-trade ratio above threshold | Rate-limit |

### Position limits (Phase 2)

- Per-customer max position per product.
- Per-customer max gross exposure.
- Configurable by tier.

## Owning agent

`surveillance`.

## Contracts consumed / produced

### Produces
- `events/surveillance.alert.v1.yaml` (specced; produced fully in Phase 2).
- Phase 2: marking-the-close flag stream to `index-service`.

### Consumes
- `events/credit.tx.v1.yaml` (Phase 1).
- `events/inference.usage.v1.yaml`, `events/compute.usage.v1.yaml` (Phase 1 — consumption-anomaly).
- Phase 2: `events/trades.executed.v1.yaml`, `events/orders.state.v1.yaml`,
  `schemas/positions.sql`.

## Dependencies

- F05 (transactions stream).
- Phase 2: KW03 (matching engine).

## Sync points

- M1 — basic abuse rules live; alert path tested.
- M2–M5 — keep spec current as trading contracts evolve.
- M6 — full 6-pattern spec finalized.

## Acceptance criteria

### Phase 1
- [ ] Anomalous-consumption alert fires on synthetic test; on-call pages.
- [ ] Rate limits enforced (per `platform-core` config).
- [ ] No false-positive storms over rolling 30-day window.
- [ ] Alert evidence JSON captured.

### Phase 2 (separate scope)
- [ ] All six patterns implemented with tunable thresholds.
- [ ] Marking-the-close flags reach `index-service` in time.
- [ ] Position limits enforced.
- [ ] Alerts queryable + auditable.

## Milestone

- M1 (Gate 1): basic abuse live.
- M6 (Gate 6): full spec ready for Phase 2 build.

---

## Status — trade surveillance detectors built (2026-09-27)

**Contract.** `events/surveillance.alert.v1.yaml` was listed as "specced" but did not exist. It is
now authored (tech-lead; a new additive subject). It carries:
- `rule`, `severity`, and `action` (`review` / `rate_limit` / `exclude_from_index` /
  `suspend_recommended` — surveillance recommends, the owning service acts);
- the tenant, the product, and `is_paper`;
- the window, and evidence JSON;
- the trade and order ids behind the alert.

**Detectors.** `services/surveillance/internal/detect` is a new Go module: pure and deterministic
rules over `orders.state.v1` and `trades.executed.v1`.

| Rule | Fires when | Action |
|---|---|---|
| wash_trade | Same beneficial owner on both sides (owner map); **exact** round trip A→B→A; or a *repeated* near round trip | review |
| spoofing | An order ≥ 10× the product's average size, user-cancelled within 30 s having filled ≤ 10%, while the tenant traded the other side | rate_limit |
| layering | ≥ 4 unfilled orders on one side at ≥ 3 prices pulled within 1 min, while the tenant traded the other side | review |
| marking_the_close | ≥ 50% of a product's volume in the 30 min before the 16:00 UTC print (min volume 1000) | exclude_from_index |
| cross_product | ≥ 60% of a product's aggressor volume, price moved ≥ 100 bps in that direction, while holding a same-direction position in a related product | review |
| excessive_cancellation | ≥ 20 orders and an order-to-trade ratio ≥ 20 within 5 min | rate_limit |
| position_limit | Net position per product, or gross exposure, above the configured limit | suspend_recommended |

**Properties.**
- **Thresholds.** Every threshold is in `Config` (`DefaultConfig` is the v1 start).
- **Event time.** Windows use the events' own time, never the wall clock.
- **Alert ids.** Deterministic, so a replayed stream never re-alerts.
- **Redelivery.** At-least-once redeliveries (the same trade id or order event id) are ignored.
- **Isolation.** Paper and real state never mix.
- **Automatic cancels.** Engine auto-cancels (IOC remainders, FOK misses, self-trade prevention) are
  not treated as the customer's cancels.

**Tests.**
- Every rule has a positive case plus near misses.
- Paper/real isolation.
- Redelivery.
- Deterministic replay, and idempotence when the same stream is fed twice.
- Every rule's alert validates against the contract.
- **No false-positive storm:** a benign 30-tenant market of ~20k orders per seed, across 3 seeds,
  must raise zero alerts.

**What that test caught.** The first thresholds (round trips within 1% / 20 bps / 10 min) raised 15
false wash alerts. Tightening them still left one chance match on a third seed. The rule now alerts
at once only on an exact round trip, and needs a near round trip to repeat between the same pair.
A redelivery bug (a replayed trade paired with its own later neighbours) was also found and fixed.

**Not done.**
- The service binary: the NATS consumer for the two subjects, publishing `surveillance.alert.v1`,
  an alerts table plus a query API for review, and deployment.
- Consuming `exclude_from_index` in index-service.
- Wiring `rate_limit` / `suspend_recommended` into the engine's risk hook.
- Phase 1 abuse monitoring (consumption anomalies, auth abuse).

All of these consume the contract as authored.

Acceptance (Phase 2): ✅ all six patterns with tunable thresholds (plus position limits) ·
⬜ marking-the-close flags reach index-service · ◐ position limits detected (enforcement is the
engine's risk hook) · ◐ alerts auditable (evidence plus ids in every alert; the store and API are
next).

