# F19 — GPU attestation stack

> Ship in **Milestone 4**. Owner: `settlement-trust`.

## Spec

Five attestation layers. None alone proves trustworthy supply; together they do.

1. **KYB + facility certificates** (manual, operations + legal).
   - Corporate documents, beneficial ownership, datacenter SOC 2 or ISO 27001.

2. **NVIDIA hardware attestation** (cryptographic).
   - Agent reports NVIDIA-signed GPU identity claims (GPU UUID, manufacturer signature).
   - Validates GPUs are genuine NVIDIA H100/H200, not relabeled.

3. **Randomized nonce-seeded challenge-response** (performance proof).
   - Periodic small workload sent to the partner GPU; expected output known; latency + throughput
     compared to reference performance for the GPU class.
   - Catches relabeling, throttling, or substitution attacks.

4. **Continuous DCGM telemetry** (operational health).
   - Utilization, temperature, throttling reasons, ECC errors streamed continuously.
   - Drift alerts (e.g., GPU clock dropped, suggesting underclock).

5. **Staked bond** (economic).
   - Partner stakes a USD bond in a custodial account.
   - Forfeitable on proven fraud (legal proof + counsel sign-off).

Recorded in `attestation_records`:

```sql
CREATE TABLE attestation_records (
    record_id UUID PRIMARY KEY,
    partner_id UUID REFERENCES dc_partners(partner_id),
    capacity_id UUID REFERENCES partner_capacity(capacity_id),
    layer TEXT NOT NULL, -- 'kyb' | 'nvidia_hardware' | 'challenge_response' | 'dcgm' | 'bond'
    state TEXT NOT NULL, -- 'pending' | 'pass' | 'fail' | 'drift'
    evidence JSONB NOT NULL,
    attested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Owning agent

`settlement-trust`.

## Contracts consumed / produced

### Produces
- `events/attestation.*.yaml` — one event per attestation result.
- Internal attestation API (within `supply-service`).

### Consumes
- Partner agent telemetry (mTLS).

## Dependencies

- F17, F16.

## Sync points

- M4 day 5 — KYB + hardware attestation flow tested on first partner.
- M4 day 15 — challenge-response automated; DCGM ingest live.
- M4 day 25 — bond staked + capacity activated.

## Acceptance criteria

- [ ] Owned DC attests trivially via the same stack (sanity check).
- [ ] First partner DC passes all five layers before capacity activates.
- [ ] Drift in any layer triggers an alert + capacity suspension.
- [ ] Documented copy: we prove genuineness + presence + performance-at-class + delivery —
      **not** verifiable arbitrary-computation correctness.

## Milestone

- M4 (Gate 4).
