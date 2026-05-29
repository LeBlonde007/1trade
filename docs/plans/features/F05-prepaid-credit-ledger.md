# F05 — Prepaid credit ledger

> Ship in **Milestone 1** (schema + balances) → **Milestone 2** (debits) → **Milestone 4** (mint/burn).
> Owner: `credit-ledger`.

## Spec

Append-only ledger. Cryptographically auditable. Atomic balance + transaction writes. Fixed-point
math. Idempotent. `is_paper` everywhere.

**Schema** (per Phase 6 §8.2, lightly extended for the GTM pivot):

```sql
CREATE TABLE credit_balances (
    balance_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sub_account_id UUID,                          -- nullable; resolved against orgs
    credit_type TEXT NOT NULL,                    -- per credit-types.md
    balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    locked_amount NUMERIC(20, 6) NOT NULL DEFAULT 0,   -- Phase 2 — for resting orders
    is_paper BOOLEAN NOT NULL,
    UNIQUE (tenant_id, sub_account_id, credit_type, is_paper)
);

CREATE TABLE credit_transactions (
    tx_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sub_account_id UUID,
    credit_type TEXT NOT NULL,
    operation TEXT NOT NULL,                      -- purchase | consumption | conversion | mint | burn | refund | trade (Phase 2)
    amount NUMERIC(20, 6) NOT NULL,
    reference_id TEXT,
    idempotency_key TEXT,                         -- unique with op
    balance_before NUMERIC(20, 6) NOT NULL,
    balance_after NUMERIC(20, 6) NOT NULL,
    is_paper BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    chain_hash TEXT NOT NULL,
    UNIQUE (operation, idempotency_key) WHERE idempotency_key IS NOT NULL
);

CREATE INDEX idx_credit_tx_tenant_time ON credit_transactions (tenant_id, created_at DESC);
CREATE INDEX idx_credit_tx_ref ON credit_transactions (reference_id);
```

**Invariants:**

- Append-only on `credit_transactions`.
- Atomicity: `BEGIN; SELECT balance FOR UPDATE; UPDATE balance; INSERT tx; COMMIT`.
- Hash chain: `chain_hash = sha256(prev_chain_hash || canonical_json(row))`.
- Reconciliation: `sum(amount) by (tenant_id, sub_account_id, credit_type, is_paper)` equals
  `balance` exactly.
- Idempotency: every mutating call carries an `Idempotency-Key`; retrying yields the same result.

## Owning agent

`credit-ledger`.

## Contracts consumed / produced

### Produces
- `openapi/credit.yaml` (balances, transactions, purchase, convert).
- `schemas/credit_balances.sql`, `schemas/credit_transactions.sql`.
- `events/credit.tx.v1.yaml`.

### Consumes
- `credit-types.md`.
- `schemas/types.sql` (tenant_id, audit columns).

## Dependencies

- F01, F03 (tenants/orgs/sub-accounts).

## Sync points

- M1 day 10 — schema + balance/tx APIs published; downstream (`platform-core`, `inference-ml`,
  `compute-platform`) start integrating.
- M2 — purchase + debit wired; Stripe webhook idempotency tested.
- M4 — mint/burn surface published; `settlement-trust` integrates.

## Acceptance criteria

- [ ] Atomic balance + tx insert; fault-injection test passes.
- [ ] Hash chain verifiable; `GET /v1/credits/audit/chain-verify` returns OK.
- [ ] Reconciliation job replays tx and matches balance.
- [ ] Idempotent purchase under webhook replay storm (CI test).
- [ ] Idempotent debit under retry storm (CI test).
- [ ] `is_paper` carried correctly through every API + event.
- [ ] Mint refused if it would over-issue against backing ratio.
- [ ] Performance: 1000 reads/sec, 100 writes/sec sustained without contention.

## Milestone

- M1 (Gate 1): schema + balances + transactions visible.
- M2 (Gate 2): purchase + first debit working.
- M4 (Gate 4): mint/burn live for partner-DC payout cycle.
