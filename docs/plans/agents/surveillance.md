# Agent plan — `surveillance`

> **Mostly paused for Phase 1.** Trade surveillance is dormant (the exchange is paused). Only
> **basic abuse / rate-limit monitoring** runs on the platform itself. Full 6-pattern spec is
> kept warm.

## 1. Scope under the pivot

- **Active in Phase 1 (minimal):** auth-abuse signals, rate-limit floors per tenant, anomalous
  consumption alerting (sudden spike = stolen credentials?), excessive-cancellation patterns on
  reservation/booking ops. Practical platform-safety surveillance.
- **Paused in Phase 1:** wash trade, spoofing, layering, marking-the-close, cross-product
  manipulation — these need trade data, and trades don't exist in Phase 1.
- **Spec-only:** full 6-pattern v1 detection (per Phase 6 §8.5).

## 2. Features owned

| Feature | Status |
|---|---|
| [KW05 — Surveillance spec + basic abuse monitoring](../features/KW05-surveillance-v0.md) | active (minimal) + spec, continuous |
| Full 6-pattern trade surveillance | Phase 2 (post-license) |

## 3. Milestone-by-milestone (Phase 1)

### Milestones 1–6 — Basic platform abuse
- Rate limits per tenant per endpoint (consumed from `platform-core` config).
- Anomalous consumption alerting: if a tenant's GPU-hours-per-hour jumps 10×, page.
- Excessive cancellation on reservation/booking ops: order-to-trade-equivalent ratios on
  reservation create/cancel.
- Alert format: `surveillance_alerts` table (per Phase 6 §9.3).
- Spec updates to `services/surveillance/SPEC.md` for the 6 trade patterns kept current alongside
  the matching-engine and market-maker specs.

### Phase 2 — Trade surveillance switch-on
- Implement: wash trade (same beneficial owner both sides), spoofing (large frequently-canceled
  orders), layering (staggered false depth), marking-the-close (concentration at index window),
  cross-product manipulation, excessive cancellation (order-to-trade ratio).
- Feed marking-the-close flags to `index-service` so manipulated observations are excluded.
- Position limits enforced.
- Actions: flag / alert / rate-limit / exclude-from-index / suspension recommendation.

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/events/surveillance.alert.v1.yaml` (specced now; produced in Phase 2 fully).
- `services/surveillance/SPEC.md` (the 6-pattern detection spec).

### Consumed
- `events/credit.tx.v1.yaml` (in Phase 1 — for anomalous-consumption detection).
- `events/trades.executed.v1.yaml` (Phase 2).
- `events/orders.state.v1.yaml` (Phase 2).
- `schemas/positions.sql` (Phase 2).

## 5. Local dev

- `services/surveillance/` runs on `:8008` in Phase 1 minimal mode.
- `make seed` creates a sample tenant whose seed transactions trigger the anomalous-consumption
  rule (to verify the alert path works end-to-end).

## 6. Dockerfile + deploy

- `deploy/docker/Dockerfile.go-service` with `--build-arg SERVICE=surveillance`.
- K8s Deployment, 2 replicas (active-active; alerts deduplicated by event_id).
- Alert sink: in Phase 1, `surveillance_alerts` table + PagerDuty for sev1 (anomalous consumption).
  In Phase 2, expand to operations console + customer-facing risk-control screens.

## 7. Conventions

- **Evaluate on every relevant event** (consumption in Phase 1; trades/orders in Phase 2).
- **Keep evidence JSON for each alert** (auditable; required by SOC 2 + future regulator).
- **Detection thresholds configurable**, not hardcoded magic numbers.
- **Phase 1 alerts must not generate false positives** at high volume — `infra-sre` owns the
  on-call rotation; surveillance noise = on-call burden.

## 8. Hard boundaries

- Detect and flag only — don't match, quote, or move credits.
- Enforcement actions are recommended / emitted, not executed by mutating other services' state.
- **Don't ship full trade surveillance in Phase 1.** Off-plan.

## 9. Definition of done

### Phase 1
- Basic abuse / anomalous-consumption / rate-limit floor live without false-positive storms.
- Spec for the 6 trade patterns kept current.
- `surveillance_alerts` schema exists; alerts queryable.

### Phase 2
- All six v1 trade patterns implemented with tests + tunable thresholds.
- Alerts carry evidence JSON.
- Marking-the-close flags reach `index-service` in time to exclude tainted observations.
