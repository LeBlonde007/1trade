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

---

## Status — five-layer attestation gates activation (2026-09-28)

**Built** (`supply.yaml` v1.2, compute-control migration `0003_attestation.sql`):
- **Five layers**, each result appended to `attestation_records` (append-only, with evidence and actor):
  - `kyb` and `bond`: recorded by operations with their evidence (document references, bond amount and
    custody reference);
  - `hardware`: the agent asks for a nonce and submits a GPU identity report signed by the attestation
    root. It must be for this source, bound to this nonce, list at least the registered number of
    **distinct** GPU UUIDs, and name the registered model. The same GPUs cannot back two live sources
    (`attested_gpus`; a retired source frees them);
  - `challenge`: a nonce answered with SHA-256 applied 100,000 times, within 30 s. It shows the agent
    is present and responsive;
  - `telemetry`: taken from heartbeats. Uncorrectable ECC errors are **drift**; a clean report with
    healthy GPUs is a pass.
- **Nonces:** fresh random 32 bytes; **one answer per challenge** (a replay or a late answer is 409 and
  records nothing); at most 10 open per source (429).
- **Gate:** `activate` and resuming a suspended source return 409 ATTESTATION_INCOMPLETE until every
  layer's latest result is pass. When the last layer passes, a pending source **activates by itself**;
  any later fail or drift **suspends** an active source at once (the pool drains it). Both are audited
  with actor `attestation`.
- **Console:** `/datacenter` shows "n / 5" per source and names what is missing or why a layer failed.

**Stand-in, stated plainly:** the hardware verifier sits behind the `HardwareVerifier` interface. Until
NVIDIA attestation (NRAS, with real GPUs) is wired, it trusts Ed25519 keys from
`ATTESTATION_TRUST_KEYS`. With no key configured the layer cannot pass (fail closed). Everything around
it is real: nonce binding, replay protection, the GPU model and count checks, the activation gate and
automatic suspension. We attest genuineness, presence and delivery — not the correctness of arbitrary
computation.

**Verified:**
- **Unit:** the challenge answer, trust-key parsing, signature verification, and that only kyb or bond
  can be recorded by hand.
- **Postgres end to end:**
  - the gate: activation is refused with 2 of 5 layers passing, and a pending source is never in the
    pool;
  - activation by attestation, with the audit;
  - report rejections: untrusted signer, another source, a stale nonce, too few GPUs, the wrong model,
    not JSON, duplicated UUIDs;
  - challenges: a replay, a late answer, GPUs claimed by a live source (then freed by retiring it), a
    retired source;
  - no trust root fails closed;
  - a wrong, late or right challenge answer, and the open-challenge cap;
  - drift suspends, and resuming is refused until telemetry is clean; a failed bond suspends;
  - access: another partner gets 404; a partner cannot record KYB.
- **Live:** the stack runs with a generated trust key. Activation is refused at 2 / 5
  (`docs/screenshots/datacenter-attesting.png`); then bond, a signed report and the challenge activate
  the source, and a paper job lands on it.
- **Mutation-checked:** 18 mutations, all caught. Each removes one check: the nonce binding, source
  binding, GPU count, model, the replay and deadline checks, the GPU claim, the retired exemption,
  drift suspension, the challenge cap, the missing-root check, UUID de-duplication, the activation gate,
  auto-activation completeness, the retired-source refusal, or manual-layer validation.

**Acceptance:** ✅ a partner passes all five layers before its capacity activates · ✅ drift in any layer
suspends the capacity · ✅ documented copy (genuineness, presence, delivery — not computation
correctness) · ⬜ the owned DC through the same stack (it is configured capacity, not a registry
source) · ⬜ real NVIDIA/NRAS verification and DCGM ingest (needs real GPUs) · ⬜ `attestation.*` events
and alerting · ⬜ periodic re-challenge scheduling.
