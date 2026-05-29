# F10 — Curated SoTA model catalog

> Ship in **Milestone 3** (expansion) → **Milestone 5** (full). Owner: `inference-ml`.

## Spec

The catalog is the **differentiation** for AI startups. Curated, current, top 3–5 per category:

| Category | Top 3-5 |
|---|---|
| Text — general | Llama 3.3 70B, DeepSeek-R1 distill 70B, Qwen 2.5 72B, Mixtral 8x22B |
| Text — small/fast | Llama 3.1 8B, Phi-4, Qwen 2.5 7B |
| Text — code | Qwen 2.5 Coder 32B, DeepSeek Coder V2.5 |
| Speech — STT | Whisper Large v3, Distil-Whisper |
| Speech — TTS | XTTS v2, F5-TTS |
| Image — generation | FLUX.1-dev, FLUX.1-schnell, SDXL |
| Image — understanding | Llama 3.2 Vision 90B, Qwen2-VL |
| Video — generation | Hunyuan Video, CogVideoX (gated to v1.5 if capacity tight) |
| Embeddings | bge-m3, jina-embeddings-v3 |
| Niche — reranker | bge-reranker-v2 |

**Refresh discipline:**
- Quarterly review.
- Promote new SoTA within 2–4 weeks of release.
- Deprecate with 30-day customer notice (email + console banner + CLI warning).

## Owning agent

`inference-ml`.

## Contracts consumed / produced

### Produces
- `GET /v1/inference/models` (catalog listing with metadata: category, context length, pricing,
  latency, status, deprecation_date).
- `docs/catalog-refresh.md` (the refresh process, kept current).

### Consumes
- `credit-types.md` (which sub-credit each model debits).

## Dependencies

- F08, F09, F11.

## Sync points

- M3 — embeddings, code, image (FLUX), small/fast all live.
- M4 — vision models live.
- M5 — full catalog at scale; video if capacity allows.

## Acceptance criteria

- [ ] All listed models serve through the gateway.
- [ ] Per-model latency documented and exposed via `/v1/inference/models`.
- [ ] Deprecation flow tested (30-day banner; CLI warning; final cutoff).
- [ ] Quarterly catalog review held; promotion of one new model demonstrated in CI.

## Milestone

- M3 (Gate 3): catalog expansion.
- M5 (Gate 5): full catalog.
