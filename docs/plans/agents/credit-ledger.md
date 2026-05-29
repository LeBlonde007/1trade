# Agent plan — `credit-ledger`

> Active, simplified under the GTM pivot. Prepaid balance + consumption debit + conversion +
> hash chain. **No trading settlement** in Phase 1 — that's added in Phase 2 when the matching
> engine switches on.

## 1. Scope under the GTM pivot

Trimmed but **not** softened. The append-only invariant, hash chain, atomic balance+tx writes,
fixed-point math, idempotency, `is_paper` enforcement, reconciliation — all stay. What's
deferred:

- Trade settlement on fill (Phase 2 — interface designed and stubbed, not wired).
- Locked-amount handling for resting orders (Phase 2 — schema column kept, unused in Phase 1).

What's emphasized now:
- Prepaid purchase booking (cash → AI credit / sub-credit / GPU credit).
- Consumption debit from `inference-gateway` and `compute-control`.
- Conversion (AI ↔ sub-credit, GPU credit tiers) at the published rate.

## 2. Features owned

| Feature | Status |
|---|---|
| [F05 — Prepaid credit ledger (schema, balances, append-only tx, hash chain)](../features/F05-prepaid-credit-ledger.md) | active, M1 |
| [F07 — Credit conversion](../features/F07-credit-conversion.md) | active, M2 (AI↔text) + M3 (other modalities + GPU tiers) |
| [F06 — Booking the purchase side](../features/F06-credit-purchase-billing.md) (co-owner with `platform-core`) | active, M2/M3 |
| Mint/burn integration with `settlement-trust` | active, M4 |

## 3. Milestone-by-milestone

### Milestone 1
- `services/credit-ledger/` scaffolded.
- Schema: `credit_balances`, `credit_transactions`, `credit_chain` (per `phase6_v2.md` §8.2 +
  the GTM-pivot adjustments).
- `chain_hash = hash(prev_chain_hash || canonical_json(row))` implemented and tested.
- Atomic transactional invariant: `BEGIN; UPDATE balance; INSERT tx; COMMIT` — verified with
  property-based tests + a fault-injection test (kill mid-transaction; verify nothing leaked).
- API: `GET /v1/credits/balances`, `GET /v1/credits/transactions`, `POST /v1/credits/purchase`
  (provisional — final shape settles M2 with Stripe webhook flow).
- `is_paper` on every row.

### Milestone 2
- Purchase flow wired to `platform-core` + Stripe webhook. Idempotency on Stripe `event_id`.
- Conversion: AI credit → text sub-credit (one direction first; the spread / bidirectional
  decision is open — see Phase 6 §17 q1; default to bidirectional with a 1% spread, awaiting
  `tech-lead` final).
- Consumption debit API consumed by `inference-gateway`. Idempotency on `usage_event_id`.

### Milestone 3
- Conversion: all sub-credits (speech, image, video, embeddings).
- GPU credit tiers (`gpu_h100`, `gpu_h200`); reserved-capacity purchases lock a balance.
- Multi-currency: purchases land in the ledger in the credit unit, not the currency — but record
  source currency in the transaction metadata for accounting.
- Consumption debit consumed by `compute-control` (GPU-hour debits).

### Milestone 4
- Mint/burn API for `settlement-trust`. Burning happens on GPU-credit redemption (consumed
  capacity); minting happens on attested capacity onboarding (within backing-ratio policy).
- Backing-ratio invariant: outstanding GPU credits per class ≤ attested reserved capacity for
  that class.

### Milestone 5
- Reconciliation tooling: scheduled job replays transactions and asserts the balance matches.
  Diverges = page.
- Hash-chain integrity audit endpoint (`GET /v1/credits/audit/chain-verify`).

### Milestone 6
- Performance hardening (purchase throughput, concurrent debits).
- Documented audit-export format for SOC 2 evidence.

### Phase 2 (post-license switch-on)
- Settle-on-fill API for `matching-engine` (already designed; flag flip + tests).
- Locked-amount handling for resting orders.
- Trade settlement events.

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/credit.yaml`.
- `docs/contracts/schemas/credit_*.sql` (table definitions — shared types in `schemas/types.sql`).
- `docs/contracts/events/credit.tx.v1.yaml`.
- The conversion-rate publication contract (lives in `credit.yaml` for now; may move to its own
  doc if it grows).

### Consumed
- `credit-types.md` — never adds a credit type without `tech-lead` updating this.
- Stripe webhook payload shape (external contract; pinned in `platform-core`).
- `events/inference.usage.v1.yaml` (drives sub-credit debits).
- `events/compute.usage.v1.yaml` (drives GPU-credit debits).
- `events/partner.capacity.v1.yaml` (drives mint/burn with `settlement-trust`).

## 5. Local dev

- `services/credit-ledger/` runs on `:8002`.
- Postgres database `credit_ledger` on `localhost:5432`.
- `make seed` creates demo balances: $10,000 AI credits + sample sub-credit + GPU credit balances
  per seeded tenant.
- Verification: `make test-ledger-chain` replays all dev transactions and asserts hash chain integrity.

## 6. Dockerfile

`deploy/docker/Dockerfile.go-service` with `--build-arg SERVICE=credit-ledger`. Distroless, static,
non-root. Exposes `/healthz`, `/readyz`, `/metrics`, `:8000`.

## 7. Deploy

- K8s Deployment, **5 replicas minimum** (ledger is the most read-hot service; reads scale
  horizontally, writes serialize on row locks).
- Postgres connection pool tuned (pgBouncer in transaction mode).
- HPA on CPU + custom metric (tx-per-second).
- Network policy: only `platform-core`, `inference-gateway`, `compute-control`, `supply-service`,
  `matching-engine` (paused but present) may reach the ledger.
- Backups: WAL archive every minute; logical dump hourly; restore drill monthly.
- Hash-chain integrity alert: page on any mismatch. Zero tolerance.

## 8. Conventions

- **Append-only.** Never `UPDATE` or `DELETE` a transaction row. Corrections are compensating entries.
- **Atomic balance + tx in one DB transaction.** Tested under fault injection.
- **Fixed-point** `NUMERIC(20,6)` everywhere; no float arithmetic on money.
- **Idempotency keys** on every mutating call (`purchase`, `convert`, `debit`, `mint`, `burn`).
- **`is_paper`** on every balance and tx.
- **No conversion-rate policy hardcoding** — read from a config table or contract source.

## 9. Hard boundaries

- Don't match orders.
- Don't quote.
- Don't set conversion policy (you execute movements others request).
- Don't schedule GPUs.

## 10. Definition of done

- Atomic, append-only, hash-chained, reconcilable.
- Idempotent under retries (proven with chaos testing).
- Paper/real isolated (separate DBs in prod).
- Every movement audited; hash-chain audit endpoint live.
- Stripe purchases book correctly under webhook replay.
- Inference + compute consumption debits work without race.
- Mint/burn keeps backing ratio honest.
