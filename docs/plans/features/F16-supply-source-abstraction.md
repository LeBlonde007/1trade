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
