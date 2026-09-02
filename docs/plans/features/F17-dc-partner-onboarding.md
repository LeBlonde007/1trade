# F17 — DC partner onboarding (manual v1)

> Ship in **Milestone 4** (first partner) → **Milestone 5** (second). Owner: `settlement-trust`.

## Spec

v1 is **manual**. Productized self-serve onboarding is v1.5 (not in this plan).

Steps (per Phase 6 §11.6 and the settlement-trust architecture doc):

```
1. Sales conversation with partner DC operator.
2. Mutual NDA + technical scoping (GPU type, count, NIC topology, SLA, region).
3. Capacity assessment (F19 attestation kickoff).
4. Commercial agreement (pricing floor, payout %, term, bond requirement).
5. 1Trade agent deployed at partner DC.
6. Capacity registered with compute control plane (F16).
7. Soft launch — partner capacity receives small allocation (e.g., 10% of own DC).
8. Monitor for 2-4 weeks (utilization, SLA hits, alerts).
9. Full activation — partner capacity in regular scheduling pool.
10. Monthly payout cycle begins (F18).
```

Schema (per Phase 6 §9.2):

```sql
CREATE TABLE dc_partners (
    partner_id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    primary_contact_email TEXT NOT NULL,
    region TEXT NOT NULL,
    state TEXT NOT NULL, -- 'pending' | 'verified' | 'active' | 'suspended'
    sla_tier TEXT NOT NULL,
    payout_terms JSONB NOT NULL,
    bond_amount NUMERIC(20, 2),                  -- staked bond
    bond_state TEXT,                              -- 'pending' | 'staked' | 'forfeited' | 'released'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE partner_capacity (
    capacity_id UUID PRIMARY KEY,
    partner_id UUID REFERENCES dc_partners(partner_id),
    gpu_type TEXT NOT NULL,
    gpu_count INT NOT NULL,
    pricing_floor NUMERIC(10, 4) NOT NULL,
    state TEXT NOT NULL, -- 'available' | 'allocated' | 'maintenance'
    last_seen_at TIMESTAMPTZ
);
```

## Owning agent

`settlement-trust`. `compute-platform` consumes the capacity registration.

## Contracts consumed / produced

### Produces
- `openapi/supply.yaml` — partner CRUD, capacity, ops dashboard.
- `events/partner.lifecycle.v1.yaml` — state changes.

### Consumes
- `events/compute.usage.v1.yaml` (drives payout calc).

## Dependencies

- F12, F16, F19.

## Sync points

- M4 day 5 — first partner contracted, agent deployed in staging.
- M4 day 15 — soft-launch traffic flowing.
- M4 day 25 — first monthly payout cycle (F18) wired.
- M5 — second partner DC onboarded; pipeline matures.

## Acceptance criteria

- [ ] Partner record created with full payout terms + bond reference.
- [ ] Agent connects via mTLS; capacity registered.
- [ ] Soft-launch allocation served real customer load.
- [ ] State transitions logged + auditable.

## Milestone

- M4 (Gate 4): first partner DC.
- M5 (Gate 5): second partner DC.
