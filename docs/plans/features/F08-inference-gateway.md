# F08 — Inference gateway

> Ship in **Milestone 2**. Owner: `inference-ml`. **The fastest revenue path.**

## Spec

**1Trade's own inference API** (Go gateway) for text, code, speech, image, and video. It is
wire-compatible with the OpenAI request/response format, so a customer migrates by changing a single
base URL — a migration convenience, not a dependency. Lead with "the 1Trade API."

Endpoints:

```
POST /v1/chat/completions
POST /v1/completions
POST /v1/embeddings
POST /v1/audio/transcriptions
POST /v1/audio/translations
POST /v1/audio/speech
POST /v1/images/generations
GET  /v1/models
```

Flow per request:

```
Customer (Authorization: Bearer <api_key>)
  → gateway: validate API key → tenant_id + sub_account_id + scopes
  → gateway: check quota (rate limit + per-tier limits)
  → gateway: check credit balance for the target sub-credit (text/speech/image/...)
            ← if insufficient → 402 Payment Required + error code
  → gateway: route to inference-runtime pool (model-aware; static co-loc or hot-swap)
  → gateway: stream response back (SSE for streaming endpoints)
  → on stream close:
      emit inference.usage.v1 { tenant_id, sub_account_id, model, input_tokens,
                                output_tokens, latency_ms, served_by_pod, supply_source_id }
  → credit-ledger consumes the event → atomic debit
```

## Owning agent

`inference-ml`.

## Contracts consumed / produced

### Produces
- `openapi/inference.yaml`.
- `events/inference.usage.v1.yaml`.
- `GET /v1/inference/models` catalog shape.

### Consumes
- Auth (`openapi/platform-core.yaml`).
- `credit-types.md`.
- `openapi/credit.yaml` — balance check + debit.
- `compute.usage.v1` indirectly (pod placement attribution flows through compute-platform).

## Dependencies

- F01, F02, F05, F12 (control plane schedules runtime pods).

## Sync points

- M2 day 5 — inference.usage.v1 event shape locked with `credit-ledger`.
- M2 end — end-to-end test green: API key → inference → debit visible in wallet.
- M3 — packing + multi-model behind same gateway (no API change).

## Acceptance criteria

- [ ] Major OpenAI SDKs work via base_url swap: Python, JS, LangChain, LlamaIndex.
- [ ] Streaming endpoints stream correctly (no buffering surprises).
- [ ] Insufficient-credit returns 402 with an actionable error message (and a link to the
      buy-credits page).
- [ ] Usage events are emitted exactly once per request; idempotent ledger debits.
- [ ] P95 time-to-first-token under documented target per model.
- [ ] Multi-tenant: one tenant cannot starve others (per-tenant fair scheduling).

## Milestone

- M2 (Gate 2): first inference dollar.

---

## Status — serving real models; catalogue is now deployment-scoped (2026-09-06)

The gateway serves live model output through the `model.Backend` seam against any OpenAI-compatible
runtime — a hosted provider, or our own GPU (F09). No gateway code changed to switch between them:
`INFERENCE_BACKEND` + `VLLM_BASE_URL` + `INFERENCE_MODEL_MAP` cover it.

**Built this pass**
- `catalog.ListServable(modelMap)` / `catalog.IsServable(id, modelMap)` — `GET /v1/models` now
  advertises only what the configured backend can actually route. The catalogue is one static list
  shared by every deployment, but each can serve a different subset; advertising all of it let a
  customer pick a model with no upstream and receive a **500 from the provider's 404**. An empty map
  means pass-through (CPU stub) and still lists everything — never filter to nothing.
- Chat + all media handlers reject an unroutable id with **`404 model_not_available`** instead of
  forwarding a doomed request.
- The built-in media map (`defaultMediaMap`) is DigitalOcean slugs; it now merges **only** when the
  backend is DO. It was merging everywhere, so a text-only self-hosted runtime advertised
  `wan-t2v` / `stable-diffusion-3.5` and failed at the upstream.
- Tests: servable filtering, the empty-map pass-through rule, the 404 guard, DO-only media scoping.

**⚠ Contract drift — needs `tech-lead`, do not treat as done**
Two interface changes are NOT reflected in `docs/contracts/openapi/inference.yaml`:
- the new `404 model_not_available` error code (only `INSUFFICIENT_CREDIT` is specified), and
- `/v1/models` is described as "List the curated catalog" but is now deployment-scoped.

Per the contract-first rule these must be authored by `tech-lead` before this counts as shipped.
