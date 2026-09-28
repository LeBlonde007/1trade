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

**Built and tested: placement, routing, gateway wiring, and (2026-09-28, below) the load/unload path.**
Not built: MIG, and anything measured on real GPUs.

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

### Load / unload path (2026-09-28)

- **Router:** `BeginLoad` commits a load plan under the lock. It claims the new model's VRAM (a
  `Loading` replica that takes no requests) and marks the eviction victims `Draining` (no new requests).
  One load per model runs at a time (`ErrLoading`). `FinishLoad`, `AbortLoad` (releases the claimed
  VRAM) and `Undrain` (a victim that could not be stopped goes back into service) complete it. A route
  to a model whose load is underway says so rather than planning another.
- **`Loader` interface; `ProcessLoader`** starts each replica as a real process (vLLM, or the CPU stub
  in dev):
  - argv with `{model}` `{port}` `{gpu}` `{device}`, no shell, and `MODEL_ID` / `PORT` /
    `CUDA_VISIBLE_DEVICES`;
  - waits for `/readyz` (`ready_timeout`), and kills a process that exits or times out while loading;
  - unloads with SIGTERM, then SIGKILL after 10 s;
  - reports a process that dies unexpectedly, and the pool stops routing to it.
- **Pool:**
  - a request for a cold pooled model starts the load (single-flight) and waits up to `load_wait`,
    then is served; otherwise it gets 503 `model_loading` while the load continues in the background;
  - victims are stopped before the new model starts;
  - static models missing at boot are loaded before the gateway serves (largest first), or the
    gateway refuses to start;
  - processes are stopped on shutdown.
- **Cold-start latency is measured:** `inference_model_load_seconds` (histogram) is recorded, alongside
  `inference_model_loads_total{result}` and `inference_model_evictions_total`, and each load logs
  `cold_start_ms`. With the CPU stub runtime a load takes about 0.2 s; real vLLM numbers need GPUs.
- **Config** (`INFERENCE_POOL`): `"loader": {"kind": "process", "command": [...], "ready_timeout":
  "10m"}`, `"load_wait": "30s"`, and a per-GPU `"device"`. On Kubernetes the same `Loader` interface
  is backed by the compute control plane.
- **Tests** (`-race`):
  - real runtime processes: load, serve, unload (the process is gone); a crash is reported; a command
    that exits fails fast;
  - hot swap with real processes: the static model starts at boot; a cold tail model loads and
    serves; a second tail model evicts the first, whose process stops;
  - 8 concurrent cold requests start exactly 1 load and are all served;
  - `load_wait` 0 answers cold and loads in the background;
  - a failed load releases its VRAM; a failed eviction abandons the load and the victim serves again;
  - draining and loading replicas take no requests.

  Mutation-checked: 14 mutations, all caught.

### Acceptance status

- [~] Top-3 models always available: static models are pinned, loaded at boot, and the gateway won't
  serve without them. Cold-start latency is now measured (`inference_model_load_seconds`); the real
  figure needs GPUs.
- [x] No thrashing: proven in simulation and on the real load path (residency, static, in-flight and
  draining guards; single-flight loads).
- [~] Headroom above 5%: enforced on measured profiles. Profiles must come from real runs under load.
- [ ] Unit economics: needs production traffic.
- [ ] MIG for video: not started.
