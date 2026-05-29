# Agent plan — `index-service`

> **Keep-warm under the GTM pivot.** Phase 1 runs a **private internal reference index** computed
> from real platform transactions. Not customer-facing, not tradeable. **The methodology is
> published and audited from day one** so the Phase 2 public index launches with credibility
> already established.

## 1. Scope under the pivot

The agent definition stays. What changes:

- Phase 1 prints are **internal-only** (not exposed customer-facing).
- The methodology document **is** published from day one (`docs/index-methodology.md`).
- The audit hash chain runs from day one — every print is immutable and verifiable.
- The constituents in Phase 1 are *real platform transactions* (inference + compute consumption,
  prepaid credit purchases at market) — there are no trades yet to feed it.

This makes the eventual exchange's index credible because it inherits real demand-side data,
not a fresh-from-zero benchmark.

## 2. Features owned

| Feature | Status |
|---|---|
| [KW01 — Private reference index](../features/KW01-private-reference-index.md) | keep-warm, M5 (live internal) |
| Index methodology document | active, M5 (publication-ready) |
| Audit hash chain on prints | active, M5 |

## 3. Milestone-by-milestone

### Milestones 1–4 — Specification (no code yet, beyond skeleton)
- Coordinate with `tech-lead` on `openapi/index.yaml` v0.9 (specced but mostly returns mock prints from `trading-frontend`).
- Methodology design document drafted: 95% trimmed mean across constituent observations, volume
  floor for a valid print, manipulation resistance via cross-validation, daily cadence at 16:00 UTC.
- Mock-data dynamics for the trading-frontend `/index` page: Brownian motion w/ mean reversion,
  visible in the frontend without a real backend.

### Milestone 5 — Live private index
- `services/index-service/` (Go) stood up.
- `index_prints` table (TimescaleDB for history).
- Daily publication pipeline:
  - 15:30 UTC — collect inputs from last 24h (real platform tx + capacity reservations + redeemed
    credits as price-discovery signals).
  - 15:45 UTC — filter outliers per the methodology.
  - 15:55 UTC — compute final value.
  - 16:00 UTC — extend audit chain, publish.
  - 16:00:01 UTC — notify subscribers (internal only in Phase 1).
- Audit hash chain: `chain_hash = hash(prev_chain_hash || canonical_json(print_row))`.
- Methodology version stamped on every print.
- **Constituent transparency**: which observations went into this print, anonymized at the
  per-tx level but aggregable.

### Milestone 6 — Auditability hardening
- `GET /v1/index/audit/chain-verify` endpoint.
- Methodology audit firm engaged (per Phase 6 §2 — external audit firm for methodology integrity).
- Methodology versioning tested (quarterly cadence; v1.1 dry-run).

### Phase 2 — Public + tradeable
- Flip the `/index` route from private to public.
- `index-service` consumes trades from `matching-engine` as additional constituent observations.
- `surveillance` flags marking-the-close activity for exclusion — coordinate the flag schema.
- Index becomes tradeable as a spot product (via `matching-engine` + `market-maker`).

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/index.yaml`.
- `docs/contracts/events/index.print.v1.yaml`.
- `docs/index-methodology.md` (the methodology doc — separate from contracts; this is the
  human-readable spec).

### Consumed
- `events/credit.tx.v1.yaml` (platform transactions feed the index in Phase 1).
- `events/trades.executed.v1.yaml` (Phase 2 — adds trade prints to inputs).
- `events/surveillance.alert.v1.yaml` (Phase 2 — marking-the-close flags trigger exclusion).

## 5. Local dev

- `services/index-service/` runs on `:8006`.
- `make seed` populates mock prints for the last 30 days so the frontend `/index` route has
  history.
- A `make index-publish-now` command runs the daily pipeline immediately (for testing).

## 6. Dockerfile

`deploy/docker/Dockerfile.go-service` with `--build-arg SERVICE=index-service`.

## 7. Deploy

- K8s CronJob for the 15:30–16:00 publication pipeline.
- K8s Deployment for the API (read path).
- TimescaleDB for `index_prints` (hypertable; chunked by month).
- **Never miss a print.** Alert if the daily print is more than 5 minutes late.

## 8. Conventions

- **Prints are immutable + hash-chained** like the ledger. No corrections; new prints (errata)
  are issued as additional rows with reference back to the corrected print.
- **Methodology version** with every print; updates are quarterly and versioned.
- **Outliers + flagged activity excluded** before computing (Phase 2 — coordinate with
  `surveillance`).

## 9. Hard boundaries

- Don't detect manipulation (that's `surveillance`).
- Don't price individual trades (that's the market).
- Don't expose the index publicly in Phase 1 (private internal only).

## 10. Definition of done

- Deterministic, auditable, reproducible prints.
- Methodology documented and versioned.
- Never misses a daily print (100% reliability).
- Outliers / flagged activity correctly excluded (Phase 2).
- Audit chain verifiable.
- Phase 2 switch-on is a config flip: public flag + add trade constituents.
