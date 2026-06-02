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

- [ ] Inference runtime pods scheduled with topology + locality awareness. *(real Kueue+Volcano — M3)*
- [ ] Customer instance provision <90s (P95). *(M3)*
- [x] Per-second GPU-hour metering produces `compute.usage.v1` events.
- [ ] Reserved-capacity workloads pre-empt on-demand on contention. *(M3; `reserved` flag plumbed)*
- [x] Mock-GPU mode for `kind`/k3d local dev works.
- [ ] Gang-scheduling correctness verified at 32+ GPU scale. *(GPU-node gated — M3)*

## Milestone

- M2 (Gate 2): inference scheduling.
- M3 (Gate 3): customer-facing GPU rental.

## Status — M2 v0 delivered (2026-06-02, tag v0.2.12)

`services/compute-control/` ships and is **live in k3d** behind the scheduler-interface seam, with
the in-memory **mock-GPU** backend (`COMPUTE_SCHEDULER=mock`); the real Kueue+Volcano backend
implements the same `scheduler.Scheduler` interface in M3.

**Built**
- Service: `config` · `domain` (GPU tiers, workload classes, exact fixed-point GPU-hour math) ·
  `auth` (tenant JWT for reads, service token for scheduling, `is_paper` from the principal never the
  body) · `events` (NATS JetStream `compute.usage.v1` publisher + noop) · `scheduler` (MockScheduler:
  all-or-nothing gang capacity, idempotent submit on `Idempotency-Key`, supply-source attribution,
  tenant scoping, quota) · `api` (`/v1/compute/types|quota|jobs|jobs/{id}|instances`) · `cmd`.
- Contract: `openapi/compute.yaml` honored; emits `events/compute.usage.v1.yaml`.
- **credit-ledger** consumer generalized to drain **both** `inference.usage.v1` and
  `compute.usage.v1` → idempotent `gpu_*` debit (idempotent on `usage_id`).
- Deploy: distroless non-root Dockerfile · `deploy/k8s/compute-control/base` (kustomize) · Tilt wiring.
- Tests: scheduler (gang capacity, idempotency, tenant scoping, cancel→meter, validation, quota) +
  API (public catalog, JWT-gated quota, service-token submit→get→cancel lifecycle, 402 on capacity).

**Proven live (k3d e2e)** — `GET /v1/compute/types` (8×H100, 0×H200) → submit a 2-pod×2-GPU H100 gang
(service token + `X-Tenant-Id`) → 202 running, `placement: dc-owned-1/mock`, availability 8→4, quota
`used 4 / remaining 4`, idempotent re-submit returns the same job → cancel frees capacity (→8) and
emits `compute.usage.v1` → credit-ledger debits **gpu_h100 1000 → 999.976667** (a hash-chained
`consumption` txn keyed on `usage_id`, `is_paper=true`).

**Deferred to M3** (GPU-node / cluster-gated): real Kueue admission + Volcano gang scheduling with
topology/locality, customer instance lifecycle (<90s start), reserved-capacity pre-emption, and the
32+ GPU correctness run. The `k8s` scheduler backend is the only new code those need behind the
existing interface.
