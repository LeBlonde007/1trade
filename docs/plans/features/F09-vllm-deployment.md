# F09 — vLLM deployment (first 3 models)

> Ship in **Milestone 2**. Owner: `inference-ml`.

## Spec

`services/inference-runtime/` — Python + vLLM workers, containerized via
`deploy/docker/Dockerfile.vllm`.

Milestone 2 model set (chosen to balance demand, VRAM fit, and showcasing):

| Model | Category | VRAM (H100 80GB) | Notes |
|---|---|---|---|
| Llama 3.1 70B (INT4) | text — general | ~40 GB | Fits with room to share |
| Llama 3.1 8B | text — small/fast | ~16 GB | Latency leader |
| Whisper Large v3 | speech — STT | ~3 GB | Distil-Whisper later |

Runtime contract:
- Each pod is one model + one (or multiple, see F11) tenant served concurrently.
- Pod registers with the gateway at startup and heartbeats.
- Pod exposes `/healthz`, `/readyz`, `/metrics`.
- Pod streams responses via internal HTTP (gateway buffers + re-emits to customer).

## Owning agent

`inference-ml`. Scheduling owned by `compute-platform`.

## Contracts consumed / produced

- Internal gateway↔runtime contract (not exposed as OpenAPI; lives in
  `services/inference-runtime/README.md`).
- Catalog entry shape (`openapi/inference.yaml` `/models` endpoint).

## Dependencies

- F01, F08 (gateway), F12 (control plane to schedule pods + GPU Operator for drivers).

## Sync points

- M2 day 5 — model weights cached to PVC; first pod boots successfully on staging GPU.
- M2 day 10 — gateway-to-runtime routing works; first cross-pod request flows end-to-end.

## Acceptance criteria

- [ ] All 3 models load and serve.
- [ ] Documented latency per model (P50, P95).
- [ ] Pod restart on crash; gateway re-routes within seconds.
- [ ] Model weights mounted from PVC (not baked into image — image stays reusable).
- [ ] Pod resource requests/limits accurate (no OOM under load).

## Milestone

- M2 (Gate 2): first inference dollar.
