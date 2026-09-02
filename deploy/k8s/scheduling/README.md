# Scheduling — Kueue + Volcano (mock-GPU local mode)

The cluster-side scheduling layer the F12 compute control plane targets: **Kueue** for per-workload-
class quota admission, **Volcano** for gang (all-or-nothing) scheduling, over a **mock GPU** resource
so it all runs on k3d with no real GPU.

## Pieces

| File | What |
|---|---|
| `mock-gpu-resource.sh` | Advertises `1trade.io/gpu: 8` on the node labeled `1trade.io/gpu=mock` (extended resource via a node-status PATCH). Re-run after a cluster restart. |
| `kueue-config.yaml` | `ResourceFlavor`s (`default-flavor`, `mock-gpu`) + one `ClusterQueue` per workload class (`cq-inference` 4 GPU, `cq-training-small` 2, `cq-training-large` 2) in a shared `1trade` cohort + a `LocalQueue` per class in `default`. |
| `examples/kueue-inference-job.yaml` | A suspended Job in the `inference` queue → Kueue admits + unsuspends (quota path). |
| `examples/volcano-gang-job.yaml` | A Volcano `minAvailable: 2` job → gang-scheduled all-or-nothing (gang path). |

## Install

Kueue + Volcano come from the FULL/SCHED tier of the data-plane installer, which then advertises the
mock GPU resource and applies `kueue-config.yaml`:

```bash
make data-plane SCHED=1     # core + Kueue + Volcano + mock-GPU resource + kueue-config
#   or the whole FULL tier (also observability): make data-plane FULL=1
```

## Verify

```bash
kubectl apply -f deploy/k8s/scheduling/examples/kueue-inference-job.yaml
kubectl get workloads -n default                 # Admitted=True
kubectl get pods -n default -l job-name=mock-infer-job -o wide   # on k3d-1trade-agent-0

kubectl apply -f deploy/k8s/scheduling/examples/volcano-gang-job.yaml
kubectl get podgroups -n default                 # phase Running (both pods bound at once)
```

## Mock → real (M3)

In M3 the NVIDIA GPU Operator advertises `nvidia.com/gpu` and the `mock-gpu` flavor splits into per-
tier flavors (`gpu_h100`, `gpu_h200`); the queue topology here is unchanged. The F12 service grows a
`k8s` scheduler backend (behind the existing `scheduler.Scheduler` interface) that creates Kueue
Workloads + Volcano PodGroups in these queues instead of the in-memory mock.
