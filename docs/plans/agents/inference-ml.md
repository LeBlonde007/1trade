# Agent plan — `inference-ml`

> **Critical-path under the GTM pivot.** Fastest revenue path: AI startups pay-per-use against
> **Exascale's own platform API** (text, code, speech, image, video).

> **Framing (positioning):** this is *our* platform and *our* API. We keep it **wire-compatible
> with the OpenAI request/response format** purely as a drop-in-migration convenience (a customer
> switches by changing one base URL) — that's a feature, not our identity. Lead with "the Exascale
> API"; treat OpenAI-wire-compatibility as a migration detail, never as what we are.

## 1. Scope under the GTM pivot

The inference layer goes from "important" to "the headline product for Phase 1." Everything in
the original agent definition stays; the emphasis shifts to **DX for AI-startup builders**:
the API must be flawless, drop-in migration must be frictionless, latency competitive, catalog current.

## 2. Features owned

| Feature | Status |
|---|---|
| [F08 — Inference gateway](../features/F08-inference-gateway.md) | active, M2 |
| [F09 — vLLM deployment (first 3 models)](../features/F09-vllm-deployment.md) | active, M2 |
| [F10 — Curated SoTA catalog (full)](../features/F10-model-catalog.md) | active, M3 (expand) → M5 (full) |
| [F11 — Multi-model-per-GPU packing](../features/F11-multi-model-per-gpu.md) | active, M3 |

## 3. Milestone-by-milestone

### Milestone 2 — First inference dollar
- `services/inference-gateway/` (Go) — auth check, quota, balance check, route, stream response,
  emit usage event. OpenAI-compatible: `/v1/chat/completions`, `/v1/completions`,
  `/v1/embeddings`, `/v1/audio/transcriptions`.
- `services/inference-runtime/` (Python + vLLM) — first 3 models: Llama 3.1 70B (INT4 quantized
  for an H100 80GB fit), Llama 3.1 8B, Whisper Large v3.
- Single-tenant per GPU first — no packing yet, no MIG. Capacity is the bottleneck this milestone;
  optimize next.
- Usage event → `events/inference.usage.v1.yaml` → consumed by `credit-ledger` for text/speech
  sub-credit debit.
- CLI: `exascale infer chat -m llama-3.1-8b "hi"` works end-to-end.

### Milestone 3 — Catalog expansion + packing
- Add: text/code (Qwen 2.5 Coder 32B, DeepSeek Coder V2.5), text/small (Phi-4, Qwen 2.5 7B),
  embeddings (bge-m3, jina-embeddings-v3), reranker (bge-reranker-v2).
- **Multi-model-per-GPU packing**:
  - Static co-location for top-3 most-popular models per GPU class.
  - Hot-swap (LRU eviction) for the long tail; cold-start probability factored into routing.
  - Realistic VRAM math: a 70B INT4 + 8B + Whisper must actually fit on H100 80GB.
- Image (FLUX.1-dev, FLUX.1-schnell, SDXL) and vision (Llama 3.2 Vision 90B, Qwen2-VL) added.
- Video (Hunyuan, CogVideoX) gated to M5 unless capacity allows.

### Milestone 4 — Reliability + enterprise
- HA gateway: zero-downtime deploys verified.
- Multi-tenant quotas and per-tenant rate limits (consumed by `platform-core` auth tier).
- Custom-prompt-cache mode (vLLM prefix caching) tuned per model.

### Milestone 5 — Full catalog
- Video gen if capacity allows.
- Reranker + niche modality models per the catalog table in Phase 6 §4.4.
- Documented latency per model in `docs/inference-latency.md`.

### Milestone 6 — Polish
- SDK convenience wrappers (Python OpenAI client config snippet, JS, Go).
- Catalog refresh discipline: quarterly review; promote SoTA within 2–4 weeks; 30-day deprecation
  notices.

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/inference.yaml`.
- `docs/contracts/events/inference.usage.v1.yaml`.
- The catalog shape (`GET /v1/inference/models`).

### Consumed
- `credit-types.md` — which sub-credit each modality debits.
- `openapi/credit.yaml` — for the debit call.
- `openapi/compute.yaml` indirectly — runtime pods are scheduled by `compute-platform`.

## 5. Local dev

- `services/inference-gateway/` runs on `:8003`.
- `services/inference-runtime/` runs as a K8s Deployment in the local `k3d` cluster, or as a
  Python process in CPU-only stub mode (returns a deterministic stub response so the gateway
  flow is testable without GPUs).
- `make up GPU=1` switches to real vLLM on the host GPU for the dev-sized models (Llama-8B fits
  on most dev GPUs).
- `exascale infer chat -m llama-3.1-8b` from the local CLI hits the local gateway.

## 6. Dockerfile

Two:
- `services/inference-gateway/Dockerfile` → uses `deploy/docker/Dockerfile.go-service`.
- `services/inference-runtime/Dockerfile` → uses `deploy/docker/Dockerfile.vllm` (see `DEPLOYMENT.md` §4).
  Model weights mounted from PVC, not baked in.

## 7. Deploy

- Gateway: K8s Deployment, 5+ replicas, HPA on QPS.
- Runtime pods: scheduled by `compute-platform` (Kueue queues per model class), pinned to GPU
  nodes via taints/tolerations, locality-aware placement.
- Static co-loc pods are pinned; hot-swap pods rotate based on demand telemetry.
- Network policy: only Kong → gateway; gateway → runtime pods (over private mesh); gateway →
  credit-ledger.

## 8. Conventions

- **OpenAI compatibility is sacred** — drop-in `OPENAI_BASE_URL` swap must work for the major
  SDKs (Python `openai`, JS `openai`, LangChain, LlamaIndex).
- Check balance **before** serving; stream response; emit usage event after stream closes.
- Idempotent usage events (one event per request, with `request_id`).
- Catalog refresh: documented in `docs/catalog-refresh.md`.
- Realistic VRAM math for packing — no "should fit" guesses.

## 9. Hard boundaries

- Don't own GPU scheduling (that's `compute-platform`).
- Don't own balances (that's `credit-ledger`).
- Don't hardcode pricing — read it from `credit-types.md` / conversion rates.

## 10. Definition of done

- OpenAI-compatible API serves all top-3–5 per category.
- Multi-model packing fits VRAM in production.
- Usage correctly meters and debits the right sub-credit.
- Cold-starts handled gracefully (P95 first-token under documented target per model).
- Catalog refresh discipline running (auto-issues to add new SoTA models).
- 99.95% uptime over a rolling 30-day window.
