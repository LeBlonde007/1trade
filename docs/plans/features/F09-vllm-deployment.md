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

---

## Status — self-hosted runtime reached, but NOT as specced (2026-09-06)

**The spec above is not met.** First real self-hosted inference is serving, but on a different
runtime, different models and different placement than this feature describes. Read the divergence
before assuming F09 is close to done.

| | Spec | Shipped today |
|---|---|---|
| Runtime | vLLM | **Ollama** (OpenAI-compatible, so the gateway is unchanged) |
| Models | Llama 70B · Llama 8B · Whisper | **Llama-3.2-1B · Qwen2.5-1.5B** |
| Placement | GPU nodes in-cluster | **Host process**, reached via `host.k3d.internal` |
| Hardware | H100/H200 class | RTX 3050, 8 GB |

**Why it diverged.** The goal was a real model answering a metered call end to end, on hardware we
own, today. An 8 GB card cannot hold Llama-70B, and vLLM needs far more host RAM than was free with
the cluster running. Ollama was already installed and speaks the same wire format, so the gateway's
`model.Backend` seam took it with no code change — which is the part worth keeping: **the runtime is
swappable and nothing above it cares.**

**Built**
- `scripts/inference-provider.sh PROVIDER=local` — points the gateway at a self-hosted runtime,
  keyless, default `http://host.k3d.internal:11434`. `LOCAL_LLM_URL` retargets it (e.g. a local vLLM
  on `:8000`, or the Tailscale GPU box) with no code change.
- Two catalog entries (F10) for the models actually served, under their real size.
- Verified: real completion through gateway → ledger debit from real token counts, model resident
  100% on GPU.

**Still open for F09 proper**
- [ ] vLLM as the runtime (this is the stated stack; Ollama is a stand-in).
- [ ] In-cluster GPU scheduling rather than a host process (`make up GPU=1` path).
- [ ] The three specced models — needs ≥24 GB for Llama-70B; the ~32 GB Tailscale box is the
      candidate, not this laptop.
- [ ] Whisper / audio transcription — not attempted.
- [ ] Weights from a PVC, restart/re-route behaviour, resource limits — none exercised.
