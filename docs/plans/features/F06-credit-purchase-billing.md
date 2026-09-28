# F06 — Credit purchase / billing

> Ship in **Milestone 2** (Stripe cards) → **Milestone 3** (ACH/wire + multi-currency).
> Co-owners: `platform-core` (orchestration) + `credit-ledger` (booking).

## Spec

Customers buy AI / sub-credit / GPU credits with cash:

- **Stripe (cards)** for self-serve smaller customers.
- **ACH / wire** for $10K+ purchases (enterprise).
- **Purchase Order (PO)** flow for enterprise procurement.
- **Multi-currency**: USD + JPY in v1 (EUR/GBP v1.5).
- Real-time visibility: balance, MTD consumption, projected month-end, alerts at 50/80/100% of
  budget, auto-stop policies.

Flow:

```
Customer → POST /v1/credits/purchase { amount, credit_type, currency, idempotency_key }
  → platform-core: validate, decide rail (Stripe vs. wire)
  → Stripe checkout / wire instructions
  → Stripe webhook OR manual wire confirmation
  → credit-ledger: atomic mint into balance (with Stripe event_id / wire reference as
    idempotency key)
  → event: credit.tx.v1 { operation: "purchase", ... }
```

## Owning agents

- `platform-core`: Stripe integration, wire instructions, PO flow, multi-currency, auto-stop.
- `credit-ledger`: booking the purchase atomically with the right idempotency.

## Contracts consumed / produced

### Produces
- `openapi/platform-core.yaml` — `/billing/checkout`, `/billing/wire-instructions`,
  `/billing/po`, webhook receiver.
- `openapi/credit.yaml` — `/credits/purchase` (ledger side).
- `events/billing.cycle.v1.yaml` — monthly billing close events (for accounting).

### Consumes
- Stripe webhook payloads.
- `credit-types.md`.

## Dependencies

- F02 (auth), F03 (orgs/sub-accounts for PO + budgets), F05 (ledger), F22 (regulatory framing
  sign-off so the prepaid framing is clean from M2).

## Sync points

- M2 — Stripe cards live; sub-5-min purchase flow tested.
- M3 — ACH/wire live; JPY supported; PO flow live.
- M3 — `trading-frontend` wires the buy-credits UI; CLI `credits purchase` works.

## Acceptance criteria

- [ ] Stripe checkout works for USD card payments.
- [ ] Stripe webhook signature verified; idempotent under replay.
- [ ] ACH/wire instructions surfaced in UI + CLI; manual confirmation flow logged + audited.
- [ ] JPY purchases land in the ledger as JPY-priced AI credits.
- [ ] Cost alerts fire at 50/80/100% of configured budget.
- [ ] Auto-stop policies (idle resource, budget-hit) configurable per tenant.
- [ ] `security-compliance`: no path skips KYC for real-money purchases.
- [ ] PO flow tested with one enterprise prospect during M3.

## Milestone

- M2 (Gate 2): first real Stripe purchase books to ledger.
- M3 (Gate 3): first real ACH/wire enterprise purchase.

---

## Status — starter grant added (2026-09-06)

**New, previously unplanned scope.** A one-time **trial grant** now mints credits without a
purchase. It appeared in no feature doc (checked F02/F05/F06) but books to the ledger, so it is
recorded here as F06's concern.

- **What:** 25.000000 `text` credits, once per tenant (`TrialCreditAmount` /
  `TrialCreditType` in `platform-core/internal/api/gaps.go`).
- **When:** at the point the account becomes usable — on email verification where the gate is on,
  on signup where it is not. Granting only on verify silently did nothing in any environment with
  `RequireEmailVerification=false`, leaving a zero balance while onboarding announced credits.
- **Always `is_paper: true`**, regardless of the tenant's own flag — a free grant must never reach a
  real-money balance.
- **Idempotent** on tenant id via the ledger's `Idempotency-Key`, so a replayed verification link
  cannot mint twice.
- **Best-effort:** a ledger outage logs and continues rather than failing an otherwise-valid
  verification. Audited as `tenant.trial_credits.grant`.
- Tests cover both grant paths, paper-only, and no-double-grant-on-replay.

**Open**
- [ ] Amount (25) is a placeholder, not a business decision — ~5,000 tokens on `llama-3.2-1b`.
- [ ] No expiry, and no abuse control: one free grant per *tenant*, and signup creates a tenant.

---

## Status — ACH and wire, US dollars only (2026-09-28)

**Decision (founder, 2026-09-28): payments are in US dollars only.** JPY is removed from checkout, the
contract, and the console. Multi-currency is out of scope.

**Built** (`platform-core.yaml` v1.9, migration `0011_payments.sql`):
- **Card:** unchanged; settles at checkout.
- **ACH bank debit:** Stripe Checkout with `us_bank_account`. The purchase goes `pending` →
  `processing` when the debit starts (`checkout.session.completed` with `payment_status=unpaid`), then
  `paid` and booked on `async_payment_succeeded`, or `failed` on `async_payment_failed`. A failed
  purchase is never revived, and a paid one never flips to failed.
- **Fix, found while adding ACH:** the webhook used to book on any `checkout.session.completed`, which
  for ACH would have minted credits before the money arrived. It now books only when payment has
  settled.
- **One booking per purchase:** the ledger idempotency key is `purchase:<id>` (it was the Stripe
  event id), so a webhook, its retries, the second ACH event and the dev auto-settle can never
  double-mint.
- **Wire:**
  - `POST /v1/billing/wires` (admin or billing, KYC for real money, from $1,000) returns 1Trade's USD
    bank details (`WIRE_*` from a Secret), the exact dollar amount and a unique `1T-…` reference;
  - treasury records the receipt with the service token (`/wires/{id}/received`); the amount must
    match to the cent, or it is 409 `amount_mismatch` and reconciled by hand;
  - a replay of the same receipt re-books idempotently, which also recovers a failed booking;
  - audited.
- **Console:** `/wallet/buy` has a USD payment-method choice (card / ACH / wire), shows wire
  instructions with the reference, and explains ACH timing. Purchase history shows the method,
  dollars, wire reference and failure reason.

**Verified:**
- **Tests:**
  - the ACH lifecycle (nothing booked at checkout or when the debit starts; booked once on success,
    retries included; a returned debit books nothing and is not revived; no booking without
    `payment_status=paid`);
  - USD-only and method validation; a paid purchase never marked failed;
  - wire: minimum, invoice, amount mismatches, sub-cent input, a tenant cannot record its own wire,
    the receipt, a replay, a second receipt, roles, KYC.
- **Mutation-checked:** 13 of 13 caught.
- **Live stack** (real ledger): an ACH purchase of 24 GPU-hours reaches the balance only on the
  cleared event (a retry does not double). The wire invoice is $2,990.00 with its reference; a wrong
  amount is refused; the correct receipt books 1,000 GPU-hours. Screenshot:
  `docs/screenshots/buy-wire.png`.

**Not built:**
- automatic wire matching from a bank feed (treasury records receipts today);
- ACH return handling after `paid` (a late R-code would need a clawback/debit flow);
- Stripe live keys and live bank details (configuration, not code).
