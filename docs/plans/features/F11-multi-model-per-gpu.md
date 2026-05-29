# F11 — Multi-model-per-GPU packing

> Ship in **Milestone 3**. Owner: `inference-ml`. Critical for unit economics.

## Spec

Three strategies, chosen per model class:

1. **Static co-location** — top-3 most-popular models per GPU class always resident.
   E.g., one H100 80GB pod hosts Llama 70B INT4 (40 GB) + Llama 8B (16 GB) + Whisper (3 GB).
2. **Hot-swap (LRU eviction)** — tail of catalog. Pods can load/unload models on demand based
   on inflight requests; cold-start probability factored into routing (gateway can route to
   warmer pods).
3. **MIG partitioning** — for video gen where a smaller VRAM slice suffices.

Routing rules:
- If a popular model is requested → prefer a statically co-loc'd pod.
- If a tail model is requested → check pod cache; if warm, use it; if cold and demand is
  spiking, evict another tail model.
- VRAM math is real, not "should fit": packing decisions use measured peak VRAM, not nominal.

## Owning agent

`inference-ml`.

## Contracts consumed / produced

- No public contract change — `openapi/inference.yaml` already abstracts away placement.
- Internal routing config in `services/inference-gateway/internal/routing/`.

## Dependencies

- F09 (vLLM), F12 (compute control plane for pod scheduling decisions).

## Sync points

- M3 day 5 — static co-loc design ready; first 3 co-loc'd models live.
- M3 day 15 — hot-swap LRU live; cold-start latency measured.
- M3 day 25 — MIG partitioning tested for video.

## Acceptance criteria

- [ ] Top-3 popular models always available with <documented> cold-start.
- [ ] Hot-swap evicts cleanly under demand spikes (no thrashing).
- [ ] Measured VRAM headroom > 5% under realistic load (no OOM).
- [ ] Unit economics: $/1M tokens improves by >X% vs. single-tenant pods (target set with
      founder in M3).

## Milestone

- M3 (Gate 3): packing in production.
