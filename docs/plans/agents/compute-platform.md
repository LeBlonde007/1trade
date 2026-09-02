# Agent plan — `compute-platform`

> **Critical-path under the GTM pivot.** The control plane the founder flagged as "the most
> important part." Schedules inference pods AND customer workloads on 1Trade-owned + partner
> capacity as one fabric.

## 1. Scope

Unchanged from the agent definition. Kubernetes + Kueue + Volcano + NVIDIA GPU Operator;
topology-aware placement; GPU lifecycle (on-demand + reserved + clusters); supply-source
abstraction (owned + partner as one pool).

## 2. Features owned

| Feature | Status |
|---|---|
| [F12 — Compute control plane](../features/F12-compute-control-plane.md) | active, M2 |
| [F13 — GPU instance lifecycle (on-demand)](../features/F13-gpu-instance-lifecycle.md) | active, M3 |
| [F14 — Reserved capacity (1/6/12 mo)](../features/F14-reserved-capacity.md) | active, M3 (co-owned with `credit-ledger`) |
| [F15 — Multi-node clusters (InfiniBand)](../features/F15-clusters-infiniband.md) | active, M5 |
| [F16 — Supply-source abstraction](../features/F16-supply-source-abstraction.md) | active, M3 (owned DC) → M4 (partner DC) |

## 3. Milestone-by-milestone

### Milestone 2 — Control plane v0 (inference-facing)
- `services/compute-control/` (Go) — thin wrapper over K8s + Kueue + Volcano + NVIDIA GPU Operator.
- Enough to schedule `inference-runtime` pods for `inference-ml` (single-tenant per GPU first).
- Mock-GPU mode for `kind` local dev (NVIDIA GPU Operator in mock mode registers virtual GPUs).
- Customer-facing API minimal: not yet exposed; inference is the only consumer this milestone.

### Milestone 3 — Customer GPU rental
- `POST /v1/compute/instances` → provision an H100/H200 instance, return SSH/connection details.
- Target: **<90s time-to-running** (P95).
- CLI: `1trade gpu create/list/stop` works end-to-end.
- Reserved capacity: 1mo / 6mo / 12mo terms with 17% / 27% / 33% discounts; represented as
  GPU credits (purchased through `credit-ledger`).
- Per-second GPU-hour metering → `events/compute.usage.v1.yaml` → `credit-ledger` debit.
- Supply-source abstraction begins with 1Trade-owned DC only — but the data model already
  carries `supply_source_id` so partner DCs slot in seamlessly.

### Milestone 4 — Partner DC integration
- Partner-DC agent (lives in `services/compute-control/agent/`): runs at partner DC, registers
  capacity (`POST /v1/supply/partners/{id}/capacity`), heartbeats utilization, reports SLA.
- Scheduler treats partner capacity equivalently for placement decisions.
- Billing-attribution pipeline: every served request records `supply_source_id` (own vs. partner),
  feeding `settlement-trust`'s payout calc.
- mTLS for agent-to-control-plane traffic.

### Milestone 5 — Clusters + InfiniBand
- Multi-node H100/H200 clusters with InfiniBand fabric.
- **Gang scheduling** (Volcano): a 256-GPU job needs all-or-nothing placement. **Heavy testing
  here** — the founder flagged this explicitly.
- Sales-engaged for 32+ GPU clusters (the one exception to self-serve).
- Slurm-on-K8s submission for training-heavy customers; K8s-native job submission for the rest.

### Milestone 6 — Reliability + scale
- Reliability: 99.9% compute uptime over a 30-day window.
- Locality-aware scheduling for inference pods (model weight cache locality).
- Documented placement quality (the SUNK-style differentiation at ~30% of full-custom effort).

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/compute.yaml`.
- `docs/contracts/openapi/supply.yaml` (the partner-capacity registration endpoints).
- `docs/contracts/events/compute.usage.v1.yaml`.
- `docs/contracts/events/partner.capacity.v1.yaml`.

### Consumed
- `credit-types.md` — GPU credit tiers.
- `openapi/credit.yaml` — for GPU-hour debits.
- `openapi/supply.yaml` (the parts owned by `settlement-trust`) — partner contractual data.

## 5. Local dev

- `services/compute-control/` runs on `:8004`.
- `k3d` local cluster with mock GPU resources registered.
- `make up GPU=1` enables real-GPU scheduling for local development against a host GPU.
- `1trade gpu create --type mock-h100` works in mock mode.

## 6. Dockerfile

`services/compute-control/Dockerfile` → `deploy/docker/Dockerfile.go-service`.

Note: this is the one service that may need a small amount of CGO (some K8s client libraries
use cgo internally). Prefer pure-Go bindings; if CGO is needed, document and use a glibc base
image with proper static linking.

## 7. Deploy

- K8s Deployment, 3 replicas (leader-elected for write paths to the scheduling state).
- Talks to the K8s API on the same cluster (in-cluster ServiceAccount).
- Vault-injected: cluster admin kubeconfigs for cross-cluster scheduling if applicable.
- Network policy: only Kong → control-plane, control-plane → K8s API server, control-plane →
  credit-ledger.

## 8. Conventions

- **Gang scheduling correctness is the hard part.** Property-based and chaos tests for
  all-or-nothing placement.
- **Inference favored over training in v1** for reliability — when capacity is tight, training
  is pre-empted before inference.
- **Single fabric** — control plane never branches on `supply_source` for scheduling logic.
  The split shows up only in billing/payout attribution.
- Partner DC agents send capacity reports; the control plane never trusts them blindly — it
  cross-checks with `settlement-trust` attestation data.

## 9. Hard boundaries

- Don't own balances (that's `credit-ledger`).
- Don't own the inference catalog (that's `inference-ml`).
- Don't build partner commercial/onboarding logic (that's `settlement-trust`).
- Don't expose the trading API.

## 10. Definition of done

- Reliable gang scheduling proven at 32, 64, 128, 256 GPU scale.
- <90s instance start (P95) for on-demand.
- Partner + owned capacity scheduled as one pool.
- Correct attribution of which capacity served each request (drives partner payouts).
- 99.9% compute uptime.
- CLI `gpu create/list/stop` is the documented surface.
