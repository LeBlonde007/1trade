# F13 — GPU instance lifecycle (on-demand)

> Ship in **Milestone 3**. Owner: `compute-platform`.

## Spec

Customer-facing on-demand GPU instances. Both CLI-first and web-available.

```
POST /v1/compute/instances
  body: { type: "h100", count: 1, image: "exascale-ml-stack-2026.05", region: "us-east-1" }
  response: { instance_id, state: "provisioning", connect: { ssh, jupyter, http } }

GET  /v1/compute/instances?state=running
GET  /v1/compute/instances/{id}
POST /v1/compute/instances/{id}/stop
POST /v1/compute/instances/{id}/start
DELETE /v1/compute/instances/{id}
```

Instance types in v1:
- H100 80GB SXM5 (single or up to 32 self-serve).
- H200 (added M3 if hardware available; else M4).

Pre-built **Exascale ML Stack images** (Ubuntu + CUDA + PyTorch + common libs) — versioned;
customers can choose stable / latest / pinned.

Target: <90s P95 time-to-running.

## Owning agent

`compute-platform`.

## Contracts consumed / produced

### Produces
- `openapi/compute.yaml` — instance endpoints.

### Consumes
- `openapi/credit.yaml` — debit GPU credit per second.

## Dependencies

- F12, F05.

## Sync points

- M3 — CLI `gpu create/list/stop` works; web compute page shows instances.

## Acceptance criteria

- [x] `exascale gpu create --type h100` returns connection info — instant on mock-GPU; the <90s-P95
  real-hardware timing is GPU-node-gated (the mock backend proves the lifecycle end-to-end).
- [x] Per-second GPU-hour debits applied to the right GPU credit tier — a metering ticker emits
  `compute.usage.v1` per interval (units = GPU-hours × count, `credit_type` = the instance tier);
  credit-ledger's `Compute` consumer (v0.2.12) debits the `gpu_*` credit, idempotent on `usage_id`.
- [x] Stop releases the GPU back to the pool within seconds — instances + scheduler jobs share one
  `pool.Pool`; stop/delete `Release()` the GPUs and meter the final partial interval.
- [x] Image versioning works; default = latest stable — `domain.ResolveImage` maps `stable`/`latest`/
  pinned ids; create defaults to `stable`, unknown pinned ids are rejected.
- [~] Idle auto-stop policy — `idle_stop_minutes` is captured on the instance + plumbed through the
  API/CLI/web; the enforcement loop lands with the F06 idle-policy integration (not in this increment).

**Delivered (v0.3.0):** `compute.yaml` v1.1.0 (instance lifecycle) · `compute-control` instance manager
(create/list/get/stop/start/delete + shared GPU pool + per-interval metering) · `exascale gpu …` CLI ·
live web `/compute` list + provision (BFF + `useCompute`). Mock-GPU backend; real K8s provisioner +
<90s-P95 timing land on a GPU node.

**Live-verified on k3d (v0.3.0):** signup → mint `gpu_h100` → create instance (running + ssh) → list →
run 8s → stop → async `compute.usage.v1` debit `gpu_h100 100.000000 → 99.997778` (8s ÷ 3600 = 0.002222
GPU-h, exact). Tenant-scoped, `is_paper` respected.

## Milestone

- M3 (Gate 3): on-demand GPU rental live.
