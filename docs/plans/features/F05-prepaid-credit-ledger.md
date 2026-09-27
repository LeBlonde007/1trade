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

---

## Contract change — paper cash + trade settlement (credit.yaml v1.1.0, 2026-09-27)

ADR-0004 (Accepted, option 2) adds the exchange's quote leg to the ledger. It is **authored in the
contract, not yet implemented here**.

- **Paper-only USD cash balance.** Kept in its own append-only, hash-chained table beside credit
  balances. It is not a credit type (`credit-types.md` §6).
- **Endpoints:**
  - `GET /v1/credits/cash/balances` and `GET /v1/credits/cash/transactions` (customer).
  - `POST /v1/credits/paper-cash/grant` (internal, platform-core at activation; idempotent per
    tenant). This is the "$10,000 paper" allocation.
  - `POST /v1/credits/settle-trade` (internal, matching-engine only; Idempotency-Key = `trade_id`).
    It moves the credit leg, the cash leg and both fees in **one** DB transaction. Seller-short is
    402 `INSUFFICIENT_CREDIT` and buyer-short is 402 `INSUFFICIENT_CASH`; in both cases nothing is
    written. `is_paper: false` is 422 `REAL_MONEY_DISABLED`.
- **Events.** Credit legs emit `credit.tx.v1` (operation `trade`, already in the enum). Cash legs
  emit the new `cash.tx.v1`.
- **Acceptance for the implementation:**
  - [ ] Migration for the cash balances and transactions, with append-only triggers.
  - [ ] Settlement is atomic: a property test shows credits and cash are conserved across buyer and
        seller, net of fees.
  - [ ] Replaying the same `trade_id` returns the original result; a different body is 409.
  - [ ] Real money is refused; a same-tenant trade and an internal account on a paper trade are
        refused.
  - [ ] Chain-verify covers the cash chains.
  - [ ] `settle-trade` accepts only the matching-engine service identity; any other caller is 403,
        including other internal services.
  - [ ] `/security-review` is clean.
