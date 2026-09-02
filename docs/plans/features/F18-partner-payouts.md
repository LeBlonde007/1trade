# F18 — Partner payouts (escrow + streamed)

> Ship in **Milestone 4** (first cycle) → **Milestone 5** (proof-of-reserves public).
> Owner: `settlement-trust`. Co-owners: `credit-ledger`, `infra-sre` (data pipeline).

## Spec

Payout pipeline:

```
1. Customer prepays 1Trade (F06) → cash sits in custodial escrow account (not the DC).
2. compute-platform serves request on partner capacity → compute.usage.v1 with supply_source_id.
3. settlement-trust aggregates per-partner consumption per cycle (daily streaming, monthly cycle).
4. payout_calc:
     gross_revenue = sum(gpu_hours × agreed_rate_per_partner)
     1trade_fee  = gross_revenue × fee_percent
     gross_payout  = gross_revenue - 1trade_fee
     holdback      = gross_payout × holdback_percent  (10-20%, per agreement)
     released      = gross_payout - holdback
5. wire transfer `released` amount; `holdback` released after dispute window passes.
6. emit payout.cycle.v1 event; record payout_id.
```

Schema (per Phase 6 §9.2):

```sql
CREATE TABLE partner_payouts (
    payout_id UUID PRIMARY KEY,
    partner_id UUID REFERENCES dc_partners(partner_id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    gpu_hours_consumed NUMERIC(20, 4) NOT NULL,
    gross_revenue NUMERIC(20, 4) NOT NULL,
    1trade_fee NUMERIC(20, 4) NOT NULL,
    holdback NUMERIC(20, 4) NOT NULL,
    partner_payout NUMERIC(20, 4) NOT NULL,
    state TEXT NOT NULL, -- 'pending' | 'wired' | 'holdback-released' | 'settled' | 'disputed'
    wired_at TIMESTAMPTZ,
    holdback_released_at TIMESTAMPTZ
);
```

**Proof-of-reserves** (published M5):

```
GET /v1/supply/proof-of-reserves
  → per credit class (gpu_h100, gpu_h200, ...):
    {
      attested_capacity_hours_per_window: 1_000_000,
      outstanding_credit_hours: 850_000,
      backing_ratio: 1.18,
      last_attestation_at: "2026-08-01T12:00:00Z",
      partners_contributing: 2
    }
```

Backing ratio ≥ 1.00 enforced at mint (F05/F14).

## Owning agent

`settlement-trust`. `credit-ledger` consumes mint/burn calls. `infra-sre` ensures the
`compute.usage.v1` event pipeline is reliable enough for monthly aggregation.

## Contracts consumed / produced

### Produces
- `events/payout.cycle.v1.yaml`.
- `openapi/supply.yaml` — `/proof-of-reserves`, `/partners/{id}/payouts`.

### Consumes
- `events/compute.usage.v1.yaml`.
- `openapi/credit.yaml` — mint/burn.

## Dependencies

- F05, F12, F16, F17, F19.

## Sync points

- M4 — first payout cycle wired; reviewed against agreed formula.
- M5 — proof-of-reserves endpoint live; web view published in platform console.

## Acceptance criteria

- [ ] Payout calc matches sum of usage events (reconciled).
- [ ] Holdback released after dispute window with no disputes.
- [ ] Disputed payouts route to operations review queue.
- [ ] Proof-of-reserves accurate against ledger + attestation.
- [ ] Backing ratio alert: any partner credit class dropping below 1.00 pages immediately.

## Milestone

- M4 (Gate 4): first payout cycle.
- M5 (Gate 5): public PoR.
