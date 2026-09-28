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

---

## Status — cluster API and console, all-or-nothing on one fabric (2026-09-28)

**Built** (`compute.yaml` v1.2):
- `POST/GET/DELETE /v1/compute/clusters`: whole 8-GPU nodes, 16–256 GPUs, `network: infiniband`,
  `topology: fat-tree | rail-optimized`.
- **Gang placement:** the whole cluster goes on **one** datacenter (one fabric) at once, or the request
  is refused (402) with nothing held. It is never split across sites and never partly placed. Reserved
  GPUs are used first.
- **Sales path:** tenants self-serve up to 32 GPUs; larger gets 403 `contact_sales`, and operations
  create it for the tenant (service token + X-Tenant-Id).
- A cluster is metered like an instance (`compute.usage.v1`), is deleted as a whole, has no
  stop/start, and is invisible to the instance endpoints. Each node has its own SSH address; node-0 is
  the head node.
- **Console:** `/compute/clusters` creates (16 / 24 / 32), lists nodes, and terminates.

**Verified:**
- **API tests:** lifecycle, replay, isolation, and kept apart from instances; 40 GPUs over two 32-GPU
  sites refused with nothing held, and 32 placed on one site; shape validation (whole nodes, one node,
  ethernet, topology, tier, unknown fields); self-serve over 32 refused; a 33-GPU instance refused.
- **Live stack:** a 24-GPU cluster comes up with 3 nodes; 40 GPUs self-serve gets `contact_sales`; a
  crafted id gets 404 at the BFF. Screenshot: `docs/screenshots/compute-clusters.png`.

**Not built (needs hardware):** a real InfiniBand fabric and all-reduce benchmarks, Volcano gang
scheduling on Kubernetes (the mock scheduler enforces the same all-or-nothing rule), Slurm-on-K8s
submission (`/clusters/{id}/jobs`), topology-aware placement inside a fabric, and fabric metadata on
partner sources.
