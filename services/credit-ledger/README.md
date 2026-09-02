# credit-ledger (F05)

The financial heart — every credit movement in 1Trade goes through here, with cryptographic
auditability. Owner: `credit-ledger`. Contracts: `docs/contracts/openapi/credit.yaml`,
`docs/contracts/schemas/types.sql`, `docs/contracts/events/credit.tx.v1.yaml`,
`docs/contracts/credit-types.md`.

## Status (F05, in progress)
- ✅ **Domain core** (`internal/domain/`): fixed-point `Money` (big.Int micro-units, exact), the
  append-only hash chain (`sha256(prev || canonical_json(row))`), and `Apply` (signed-delta balance
  math + insufficient-credit guard + chain extension). Pure, IO-free, unit-tested (`go test ./...`).
- ✅ **Schema** (`migrations/0001_init.sql`): `credit_balances` + append-only `credit_transactions`
  (DB-level append-only trigger; `(operation, idempotency_key)` unique; `is_paper` everywhere).
- ⬜ **Store** (`internal/store/`): Postgres binding — atomic balance update + tx insert in one DB
  transaction; reconciliation replay.
- ⬜ **API** (`cmd/credit-ledger/`): the `credit.yaml` endpoints (balances, transactions, purchase,
  convert, debit, mint, burn, chain-verify) behind auth; emit `credit.tx.v1`.
- ⬜ Dockerfile + k8s manifests + integration tests against the live Postgres.

## Invariants (enforced)
Append-only · atomic balance+tx · `chain_hash = sha256(prev_chain_hash || canonical_json(row))` ·
fixed-point `NUMERIC(20,6)` (no floats) · idempotency on every mutating call · `is_paper` on every
balance/tx · reconciliation (replay reproduces the balance exactly).

## Dev
```bash
cd services/credit-ledger
go test ./...        # domain unit tests (no DB needed)
```
