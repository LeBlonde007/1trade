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

- [ ] `exascale gpu create --type h100` returns connection info in <90s P95.
- [ ] Per-second GPU-hour debits applied to the right GPU credit tier.
- [ ] Stop releases the GPU back to the pool within seconds.
- [ ] Image versioning works; default = latest stable.
- [ ] Idle auto-stop policy honored (configured per F06).

## Milestone

- M3 (Gate 3): on-demand GPU rental live.
