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

---

## Status — methodology published; computation still mock (2026-09-06)

**"Methodology published from day one" is now actually true.** `docs/index-methodology.md` existed
as a requirement and as six references across the repo — including the shared contract
`openapi/index.yaml` — but the file itself was never written. It is now published, at the version
the code actually implements (**v1.2**, effective 2026-04-01), and its claims were checked line by
line against `refindex.go` rather than written from the spec.

**What is real today**
- The **audit chain** (immutable commitment #2). Prints are hash-chained with
  `SHA256(prev || canonical_json)` over an explicit fixed key order and fixed-point values.
  Verified end to end: re-deriving six prints from *only* what §6 of the published document states
  reproduces every `chain_hash`, with links intact. A third party can audit the series from the
  document alone — which is the entire point of publishing it before launch.
- The disclosed **constituent + weight schedule** (6 constituents, summing to exactly 1.0000),
  served live at `GET /v1/index/methodology` and identical to §3 of the document.
- The documented parameters match the code exactly: 120-minute window, 95% trimmed mean,
  25,000 volume floor, 16:00 UTC cadence, genesis 2025-10-15.

**What is not real**
- **Values are simulated.** Every print carries `provisional: true` and `source: "mock"`, and
  collections carry `is_mock: true`. The document leads with that and §9 lists the limitations
  plainly, including the single-operator conflict.
- No observations are drawn from `credit.tx.v1` yet — the ledger publishes the events, nothing
  consumes them for the index.
- `index-service` does not exist; the reads live in `matching-engine/internal/refindex`, which
  labels itself a keep-warm mock.

**Next for KW01**
- [ ] Consume `credit.tx.v1` and compute prints from real consumption + purchase observations.
- [ ] Stand up `index-service` and move the reads; the contract and the document do not change.
- [ ] Daily 16:00 UTC publication job (today's cadence is derived, not scheduled).
- [ ] Independent attestation of the chain.

Note: `docs/plans/SEQUENCING.md` still names the gate "methodology v1.0"; the implemented and
published version is 1.2. Left as-is rather than silently editing a milestone definition.
