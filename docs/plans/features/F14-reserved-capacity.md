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
