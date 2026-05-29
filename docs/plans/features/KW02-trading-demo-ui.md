# KW02 — Trading dashboard kept warm

> **Continuous** — already exists, stays demo-able. Owner: `trading-frontend`.

## Spec

Keep the existing trading dashboard alive as the **investor demo** and the Phase 2 surface.
Concretely:

- `/trade` route remains, on mock data (driven by `matching-engine`'s mock adapter — KW03).
- Prominent "**Demo — exchange paused. See methodology.**" ribbon on every trading route.
- All trading screens (Trading Dashboard, Market Detail, Wallet trading sections, Portfolio,
  History, Trader KYC) remain functional in demo mode.
- The mock data + UI quality is good enough to show in investor meetings.
- Methodology page is real (linked from the demo ribbon).

When Phase 2 switch-on happens, this is a **flag flip** + remove the ribbon + cut over to real
matching engine + market maker + index.

## Owning agent

`trading-frontend`.

## Contracts consumed / produced

### Consumes
- `openapi/trading.yaml` (read endpoints from the mock adapter in Phase 1).
- `openapi/index.yaml` (read; uses mock data in Phase 1, switches to live in Phase 2).
- `credit-types.md`.

## Dependencies

- KW03 (mock adapter).

## Sync points

- M1 — ribbon added; copy reviewed by `security-compliance`.
- Continuous — keep alive; iterate when mock-data quality is criticized in demos.

## Acceptance criteria

- [ ] All trading screens render with believable mock data.
- [ ] Ribbon present on every trading route, with link to methodology.
- [ ] No banned regulatory words in customer-facing copy (per F22).
- [ ] Mock candles look Brownian (visual smoke test); depth power-law; trades log-normal.

## Milestone

- M1 (Gate 1): ribbon + mock data quality verified.
