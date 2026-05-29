# KW01 — Private reference index (keep-warm)

> **Live, internal-only** from **Milestone 5**. Owner: `index-service`. Activates publicly in Phase 2.

## Spec

Compute a daily reference index from **real platform transactions** (consumption + prepaid
purchases as price-discovery signals) so the eventual public index launches with established
methodology and historical track record.

- **Internal-only in Phase 1** — visible to operations + design partners + investors, never
  customer-facing/tradeable.
- **Methodology is published from day one** (`docs/index-methodology.md`) so external
  audit/review can begin.
- **Audit hash chain** runs from day one — prints are immutable.
- **Daily cadence at 16:00 UTC** with the full Phase 6 §11.5 publication pipeline.

Phase 1 constituent observations (no trades available):
- Prepaid purchase prices (per credit class).
- Realized consumption rates (USD-equivalent per consumed sub-credit).
- Reserved-capacity transaction prices.

Phase 2 additions (post-switch-on):
- Live trade prints (from `matching-engine`).
- Marking-the-close flags from `surveillance` exclude tainted observations.

## Owning agent

`index-service`.

## Contracts consumed / produced

### Produces
- `openapi/index.yaml` (the read API; private auth scope in Phase 1).
- `events/index.print.v1.yaml`.
- `docs/index-methodology.md`.

### Consumes
- `events/credit.tx.v1.yaml` (Phase 1).
- Phase 2: `events/trades.executed.v1.yaml`, `events/surveillance.alert.v1.yaml`.

## Dependencies

- F01, F05 (transactions to feed the index).

## Sync points

- M3 — methodology document drafted, reviewed with `tech-lead` + `security-compliance`.
- M5 — first live private print published; daily cadence stable.
- M6 — methodology audit firm engaged.

## Acceptance criteria

- [ ] Daily print at 16:00 UTC; never missed.
- [ ] Audit hash chain verifiable.
- [ ] Constituent transparency endpoint works (anonymized aggregates).
- [ ] Methodology version stamped on every print.
- [ ] Phase 2 switch-on is a flag (`public: true` + add trade constituents). No new code paths.

## Milestone

- M5 (Gate 5).
