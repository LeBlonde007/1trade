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

## Build status (2026-09-28)

**Built and tested: placement, routing and gateway wiring.** Not built: process load/unload (F12), MIG,
and anything measured on real GPUs.

On this stack one runtime process serves one model (`services/inference-runtime`). So "several models
per GPU" means several single-model processes sharing a GPU's memory. The gateway decides placement
and routing. Starting and stopping processes belongs to the compute control plane (F12).

- **`internal/routing`** (pure, no I/O):
  - `Plan` pins static models, largest first, each on the GPU where it fits most tightly (best-fit
    decreasing). It fails if any static model can't be placed.
  - The `Router` tracks live replicas. For a warm model it routes to the replica with the fewest
    requests in flight, then the least recently used one.
  - For a cold model it returns a load plan: the tightest GPU with free room. Failing that, it picks
    the GPU needing the fewest evictions, evicting least recently used first.
  - VRAM rules:
    - every decision uses measured peak VRAM (`measured_at` required; an unmeasured model is refused);
    - every GPU keeps `headroom_ppm` free (default 5%);
    - a registration that would overcommit a GPU is refused.
  - Anti-thrash: a replica is never evicted if it is static, has requests in flight, or is younger than
    `min_residency` (default 2 min).
- **`internal/pool`** plus the `INFERENCE_POOL` env (JSON: `gpus`, `profiles`, `replicas`,
  `headroom_ppm`, `min_residency`; unknown fields are rejected).
  - Chat for pooled models is spread across the runtime pods. Other models and modalities use the
    existing backend.
  - A pooled model with no warm replica answers **503 `model_loading` with `Retry-After`**. It is not
    metered, and the load plan is logged for F12.
  - The gateway **refuses to boot** if the spec breaks a VRAM rule, or if a static model has no running
    replica.
- **Tests** (`go test -race ./internal/routing/ ./internal/pool/ ./internal/api/`):
  - packing and headroom, including 500 random fleets that never overcommit;
  - refusal of unmeasured models;
  - each eviction protection, the least-recently-used victim, and the fewest-victims choice;
  - 10 minutes of alternating demand for one slot: at most one swap per `min_residency`;
  - concurrent routing under `-race`;
  - real HTTP pods: 8 held requests split 4/4, with in-flight counts draining to zero.

  Mutation-checked: removing the static, in-flight or residency guard, reversing the eviction order,
  dropping headroom, or switching to worst-fit each fails a test.

### Acceptance status

- [~] Top-3 models always available: static models are pinned and the gateway won't boot without them.
  Cold-start latency is not measured yet (needs GPUs).
- [~] No thrashing: proven in simulation; the real load path waits on F12's loader.
- [~] Headroom above 5%: enforced on measured profiles. Profiles must come from real runs under load.
- [ ] Unit economics: needs production traffic.
- [ ] MIG for video: not started.
