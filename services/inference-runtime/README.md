# inference-runtime (F09 — vLLM workers)

Python + vLLM workers that actually run the models. The gateway (F08) routes customer requests to a
runtime pod; the pod serves them. Owner: `inference-ml`. Scheduling: `compute-platform` (F12).

## The gateway↔runtime contract (internal — not customer-facing OpenAPI)

A runtime pod is an **OpenAI-compatible HTTP server** (this is vLLM's native API), so the gateway's
`VLLMBackend` talks to it with the same request/response shapes it exposes to customers. One pod
serves **one model** (see F11 for multi-model packing).

### Endpoints the pod exposes
```
POST /v1/chat/completions   # OpenAI-compatible; the gateway calls this for text models
GET  /healthz               # process up
GET  /readyz                # model weights loaded + ready to serve
GET  /metrics               # Prometheus (tokens/s, queue depth, GPU mem, latency histograms)
```

### Request (gateway → runtime)
```json
{ "model": "llama-3.1-8b", "messages": [{"role":"user","content":"..."}],
  "max_tokens": 512, "stream": false }
```
`model` is the **catalog id** (the runtime is started with vLLM `--served-model-name <catalog-id>`).

### Response (runtime → gateway)
Standard OpenAI `chat.completion`:
```json
{ "id":"...", "object":"chat.completion", "model":"llama-3.1-8b",
  "choices":[{"index":0,"message":{"role":"assistant","content":"..."},"finish_reason":"stop"}],
  "usage":{"prompt_tokens":312,"completion_tokens":188,"total_tokens":500} }
```
The gateway bills from `usage` (exact tokens from the real tokenizer); if `usage` is absent it falls
back to its own estimate. The gateway never trusts the runtime for auth, balance, or `is_paper` —
those are resolved gateway-side before the call.

### Routing & lifecycle (M2 → F11/F12)
- M2: the gateway targets a single runtime URL (`VLLM_BASE_URL`) for text. A model→pod map and
  pod **registration/heartbeat** (pod announces `{model, url, capacity}` at startup, heartbeats
  liveness) land with F11/F12; until then routing is static config.
- Pods crash-restart (k8s); the gateway re-routes to a healthy pod within seconds (readiness-gated).

## Models (M2 set)
| Catalog id | Model | VRAM (H100 80GB) |
|---|---|---|
| `llama-3.1-70b` | Llama 3.1 70B (INT4) | ~40 GB |
| `llama-3.1-8b`  | Llama 3.1 8B | ~16 GB |
| `whisper-large-v3` | Whisper Large v3 (STT) | ~3 GB |

Weights are mounted from a **PVC** (not baked into the image) so the image stays small + reusable.

## Local dev (no GPU)
The M2 models need a 40 GB+ GPU node (this repo's dev box / CI have none). For local wiring, a CPU
**stub runtime** (`stub/`) serves the same contract deterministically, so the gateway's `VLLMBackend`
is exercised end-to-end in k3d without a GPU. Real weights swap in on a GPU node with no gateway
change. Run on a GPU node: `docker build -f deploy/docker/Dockerfile.vllm` then the k8s manifest in
`deploy/k8s/inference-runtime/`.
