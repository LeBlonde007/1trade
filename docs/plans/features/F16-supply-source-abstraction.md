# F16 — Supply-source abstraction

> Ship in **Milestone 3** (owned DC) → **Milestone 4** (partner DC). Co-owners: `compute-platform` +
> `settlement-trust`.

## Spec

The control plane treats 1Trade-owned and partner-supplied GPUs as **one scheduling pool**.
The distinction surfaces only in:

- **Billing attribution**: every served request records which capacity served it (`supply_source_id`).
- **Payouts**: `settlement-trust` aggregates per-source consumption to compute partner payouts.

Partner DC integration:

```
Partner DC operator → installs 1Trade agent (provided binary; mTLS bootstrap)
  → agent registers capacity:
    POST /v1/supply/partners/{id}/capacity
      { gpu_type: "h100-80gb-sxm5", count: 128, nic: "infiniband-ndr",
        location: "us-east-1a", sla_tier: "gold" }
  → agent heartbeats utilization, throttling, ECC errors via DCGM telemetry
  → control plane registers the capacity equivalently to owned GPUs
  → scheduler treats it equivalently for placement
  → on serve: emit compute.usage.v1 with supply_source_id = partner_id
```

## Owning agents

- `compute-platform`: scheduler + agent protocol + attribution.
- `settlement-trust`: attestation + payout pipeline.

## Contracts consumed / produced

### Produces
- `openapi/supply.yaml` — partner-capacity registration.
- `events/partner.capacity.v1.yaml`.
- `events/compute.usage.v1.yaml` — extended with `supply_source_id`.

## Dependencies

- F12, F17 (partner onboarding), F19 (attestation).

## Sync points

- M3 — `supply_source_id` field on usage events; owned DC reports as the source.
- M4 — first partner DC live in production.

## Acceptance criteria

- [ ] Owned and partner capacity scheduled by the same scheduler with no special branches.
- [ ] Every usage event records the supply source.
- [ ] Partner payout calc matches sum of partner-attributed `compute.usage.v1` events.
- [ ] Withdrawing a partner's capacity (state=suspended) drains in-flight work cleanly.

## Milestone

- M3 (Gate 3): owned-DC anchor.
- M4 (Gate 4): first partner DC.

## Status — one pool, partner registry, drain (2026-09-28)

**Built**
- **One pool across supply sources** (`compute-control/internal/pool`):
  - capacity is held per source, and placement picks the source with the most free GPUs of the tier;
  - owned and partner capacity are never special-cased; ties go by id;
  - a gang is never split across datacenters.

  Every job and instance records the source that served it, which flows into
  `compute.usage.v1.supply_source_id`. A restarted instance is re-attributed to wherever it lands.
- **Partner registry** (`supply.yaml` v1.0, migration `0001_supply.sql`):
  - endpoints: register (idempotent), list, get, heartbeat, suspend / resume, retire, usage;
  - **activation by operations only** (attestation, F19, will do it);
  - the admin or engineer role manages a tenant's own sources; another tenant's source reads as 404;
  - an append-only audit of every state change.
- **Schedulable** means active, a heartbeat in the last 2 minutes, and healthy GPUs above 0. Capacity
  is the healthy count, never more than registered. The Syncer mirrors the registry into the pool
  after every mutation and every 15 seconds.
- **Drain:** suspended, retired, silent or pending sources take no new work. Running work keeps its
  GPUs, and a retired source leaves the pool once it has drained.
- **Usage per source:** every metered interval is recorded, append-only and keyed on `usage_id`.
  This is the payout basis for F18.
- **Deploy:** `DATABASE_URL` and a migrations initContainer (glob), plus a Tilt migrations ConfigMap.

**Tests:**
- the pool: blind placement, no gang split, drain, shrink, pinned reservations, concurrency with no
  overbooking;
- end to end on Postgres: register, pending, operations activate, heartbeat, placement on the
  partner, suspend (new work goes to owned, the running job stays), usage attributed, retire,
  removed;
- access control: roles, isolation, operations' view, validation;
- stale heartbeat.

Mutation-checked: cross-tenant read, the role check, partner self-activation, the stale heartbeat,
and unhealthy GPUs all fail a test.

**Acceptance:** ✅ same scheduler, no special branches · ✅ every usage event records its source ·
✅ suspension drains in-flight work cleanly · ⬜ payout equals the sum of partner usage (F18).

**Open:** mTLS agent bootstrap (heartbeats use the partner's own token today);
`partner.capacity.v1` event (nothing consumes it yet); real DCGM telemetry.
