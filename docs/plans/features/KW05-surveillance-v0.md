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
