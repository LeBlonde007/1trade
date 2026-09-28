# F14 — Reserved capacity

> Ship in **Milestone 3**. Co-owners: `compute-platform` + `credit-ledger`.

## Spec

Customers prepay for guaranteed GPU access over a term:

| Term | Discount vs. on-demand |
|---|---|
| 1 month | 17% |
| 6 months | 27% |
| 12 months | 33% |

Mechanism: reservation is represented as **GPU credits** (e.g., `gpu_h100`) credited to the
customer's wallet on purchase. Consumption burns those credits at the discounted rate. When the
reserved balance is exhausted, on-demand pricing kicks in (configurable per tenant).

Flow:

```
Customer → POST /v1/credits/purchase
  { amount: <GPU-hours>, credit_type: "gpu_h100", term: "12mo", currency: "USD" }
  → platform-core: rail (Stripe/wire)
  → credit-ledger: mint GPU credits (with backing-ratio check against settlement-trust)
  → compute-platform: pre-emption priority bumped for that tenant
```

Backing: `settlement-trust` enforces ≥100% backing — purchases over current attested reserved
capacity for the term are blocked or queued pending capacity sign-up.

## Owning agents

- `compute-platform`: scheduling/pre-emption priority + capacity reservation in the supply pool.
- `credit-ledger`: minting the GPU credits with the right tenor metadata.
- `settlement-trust`: backing-ratio enforcement.

## Contracts consumed / produced

### Produces
- `openapi/compute.yaml` — reservation endpoints.
- Mint API on `openapi/credit.yaml` (used by reservation flow).

### Consumes
- `openapi/supply.yaml` — backing-ratio check.

## Dependencies

- F05, F06, F12, F19 (attestation supports backing).

## Sync points

- M3 — UI flow + CLI `gpu reserve --type h100 --hours 1000 --term 12mo` work.

## Acceptance criteria

- [ ] Reservation flow with discount applied correctly.
- [ ] Mint refused when backing would drop below 100%.
- [ ] Scheduling pre-emption tested under contention.
- [ ] CFO-friendly reporting: mark-to-market value of remaining GPU credits visible in console.

## Milestone

- M3 (Gate 3).

---

## Status — prepaid reservations held in the pool (2026-09-28)

**Built** (`compute.yaml` v1.2, compute-control migration `0004_reservations.sql`, ledger consumer):
- **Terms:** 1 month (730 h, 17% off), 6 months (4,380 h, 27% off), 12 months (8,760 h, 33% off).
  Price = gpus × hours × (100 − discount) / 100 in the tier's GPU credits (one credit is one on-demand
  GPU-hour). Exact `big.Rat` math, and a DB CHECK that the stored price matches the formula.
- **Backing:** a reservation is sold only against GPUs free right now. They become a **hold** in the
  pool for the tenant, side (paper or real) and tier. On-demand placement (instances and internal jobs)
  only uses GPUs beyond the unused holds, so reserved capacity cannot be taken. Holds are pool-wide, so
  draining one datacenter does not void a reservation.
- **Payment:** hold, then record `pending_payment`, then debit the ledger (`/v1/credits/debit`),
  idempotent on `reservation:<id>`, then `active` with its term.
  - A short balance means `failed` (402), and the GPUs go back.
  - A ledger outage leaves it pending (503, GPUs held). A retry with the same Idempotency-Key, or the
    minute Sync, pays it **exactly once**.
  - Only the admin or billing role can buy.
- **Use:** the owner's instances draw from the hold first (`reserved: true`). Their `compute.usage.v1`
  carries `units: 0`, prepaid, and still reports `gpu_seconds` for payouts and audit. The ledger acks a
  zero-unit event without a movement.
- **Expiry:** Sync expires ended terms and rebuilds every hold from the records (also at startup). Work
  still running is billed on-demand from the next interval, and never billed for GPUs a reservation
  still covers.
- **Console:** `/compute/reserve` shows a live quote, buys with a held Idempotency-Key, and lists
  reservations. The `/compute` side card shows active reservations and reserved GPUs in use; the old
  placeholder card is gone.

**Verified:**
- **Unit:**
  - pool holds: sold only against free capacity; on-demand cannot take held GPUs; the owner draws from
    the hold first; paper and real stay apart;
  - an ended hold is overdrawn and its work detached;
  - the ledger consumer does not book zero units.
- **Postgres end to end:**
  - the quote math and validation;
  - buy: prepaid once, and a replay does not charge; the same key with another body is a 409;
  - held GPUs are refused to another tenant's instance and to internal jobs;
  - the owner's instance is reserved and its usage has units 0; beyond the reservation it is billed;
  - roles and isolation;
  - no capacity means nothing recorded or charged; too few credits means failed and the GPUs released;
  - ledger outage, then a retry and a Sync each pay exactly once;
  - holds rebuilt after a restart; expiry moves running work to on-demand.
- **Live stack:** sandbox checkout buys 5,000 H100 credits. The quote is 4,847.2, and the **real ledger**
  goes to 152.8. An 8-GPU instance runs on the reservation and the next one is on-demand. Screenshots:
  `docs/screenshots/compute-reserve.png`, `compute-reserved-card.png`.
- **Mutation-checked:** 16 mutations that change behaviour, all caught.

**Acceptance:** ✅ reservation flow with the discount applied · ✅ refused when capacity is not there
to back it (≥100% backing by construction) · ✅ scheduling priority under contention (held GPUs are
never given to on-demand) · ⬜ evicting running on-demand work (no pre-emption yet: a hold is sold only
from free GPUs) · ⬜ mark-to-market reporting of remaining credits · ⬜ CLI `gpu reserve`.
