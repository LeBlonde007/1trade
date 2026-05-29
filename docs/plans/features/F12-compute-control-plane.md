# F12 — Compute control plane

> Ship in **Milestone 2** (v0, inference-facing) → **Milestone 3** (customer-facing).
> Owner: `compute-platform`. **The founder flagged this as the most important part.**

## Spec

`services/compute-control/` is a Go service that wraps the K8s control plane (Kueue + Volcano +
NVIDIA GPU Operator) and exposes a stable customer-facing API. It also owns the partner-DC
agent path (F16).

Internal layers:
- **Scheduling**: Kueue queues per workload class (inference, training-small, training-large),
  Volcano gang scheduling for multi-pod jobs, topology + locality-aware placement.
- **GPU lifecycle**: provision (Kubernetes-managed VM/pod), bootstrap, expose endpoints, tear down.
- **Quota**: per-tenant + per-sub-account quotas, read by `inference-gateway` for routing
  decisions.
- **Supply-source attribution**: every served request records `supply_source_id` so
  `settlement-trust` can compute partner payouts.

## Owning agent

`compute-platform`.

## Contracts consumed / produced

### Produces
- `openapi/compute.yaml` (instances, types, quota, jobs, clusters).
- `events/compute.usage.v1.yaml`.

### Consumes
- `openapi/credit.yaml` (debit GPU credits).
- `openapi/supply.yaml` (partner capacity registration shape).
- `credit-types.md`.

## Dependencies

- F01 (K8s, Kueue, Volcano, GPU Operator).
- F05 (ledger for debits).

## Sync points

- M2 day 5 — internal scheduling API serves `inference-runtime` pod placement.
- M3 — customer-facing instance API live; sub-90s start.
- M4 — partner-DC agent integration (F16).
- M5 — clusters + InfiniBand (F15).

## Acceptance criteria

- [ ] Inference runtime pods scheduled with topology + locality awareness.
- [ ] Customer instance provision <90s (P95).
- [ ] Per-second GPU-hour metering produces `compute.usage.v1` events.
- [ ] Reserved-capacity workloads pre-empt on-demand on contention.
- [ ] Mock-GPU mode for `kind` local dev works.
- [ ] Gang-scheduling correctness verified at 32+ GPU scale.

## Milestone

- M2 (Gate 2): inference scheduling.
- M3 (Gate 3): customer-facing GPU rental.
