# F15 — Multi-node clusters (InfiniBand)

> Ship in **Milestone 5**. Owner: `compute-platform`.

## Spec

Sales-engaged for 32+ GPU clusters (the one exception to self-serve). Smaller clusters (≤32 GPU)
self-serve via the same API as F13. The differentiator is:

- **InfiniBand fabric** for low-latency multi-node training.
- **Gang scheduling** (Volcano) — all-or-nothing placement. **Heavy testing here** — a partial
  allocation on a 256-GPU job wastes massive money.
- **Slurm or K8s-native submission** — Slurm-on-K8s adapter for ML researchers; K8s job
  submission for the rest.

```
POST /v1/compute/clusters
  { gpus: 64, type: "h100", network: "infiniband", topology: "fat-tree" }
  → control plane: reserve, place, bring up, expose connection
POST /v1/compute/clusters/{id}/jobs
  { image, command, slurm_script? }
```

## Owning agent

`compute-platform`.

## Contracts consumed / produced

- `openapi/compute.yaml` — cluster endpoints.

## Dependencies

- F01 (InfiniBand fabric), F12, F13, F14.

## Sync points

- M5 — first 32-GPU and 64-GPU clusters demoed to design partner.

## Acceptance criteria

- [ ] Gang scheduling: 64-GPU job either starts fully or queues — never partial.
- [ ] Slurm-on-K8s submission tested with one real training workload.
- [ ] InfiniBand bandwidth meets documented spec under all-reduce benchmarks.
- [ ] Topology-aware placement keeps allreduce-heavy patterns on tight topology.

## Milestone

- M5 (Gate 5).
