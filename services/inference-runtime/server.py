#!/usr/bin/env python3
"""Production inference-runtime worker (F09): vLLM behind an OpenAI-compatible HTTP API.

One pod serves one text model (catalog id via MODEL_ID; weights from a PVC mount via MODEL_PATH).
It implements the gateway↔runtime contract in ../README.md: POST /v1/chat/completions plus
/healthz, /readyz, /metrics. Run on a GPU node (see deploy/docker/Dockerfile.vllm). For GPU-free
local dev use stub/server.py, which speaks the same contract.

Speech models (Whisper) are served by a separate runtime variant; this worker is the text path the
gateway's chat route uses today.
"""

import os
import time

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse
from vllm import AsyncEngineArgs, AsyncLLMEngine, SamplingParams
from vllm.utils import random_uuid

# MODEL_PATH is the PVC-mounted weights dir; MODEL_ID is the catalog id the gateway addresses.
MODEL_PATH = os.getenv("MODEL_PATH", "/models/current")
MODEL_ID = os.getenv("MODEL_ID", "llama-3.1-8b")
MAX_MODEL_LEN = int(os.getenv("MAX_MODEL_LEN", "8192"))

app = FastAPI()
_engine: AsyncLLMEngine | None = None
_ready = False
_started = time.time()


@app.on_event("startup")
async def _load_model() -> None:
    """Load the model into vLLM at startup; flip readiness once weights are resident."""
    global _engine, _ready
    args = AsyncEngineArgs(
        model=MODEL_PATH,
        served_model_name=MODEL_ID,
        max_model_len=MAX_MODEL_LEN,
        gpu_memory_utilization=float(os.getenv("GPU_MEM_UTIL", "0.90")),
        dtype=os.getenv("DTYPE", "auto"),
    )
    _engine = AsyncLLMEngine.from_engine_args(args)
    _ready = True


@app.get("/healthz")
async def healthz() -> JSONResponse:
    """Liveness: the process is up."""
    return JSONResponse({"status": "ok"})


@app.get("/readyz")
async def readyz() -> JSONResponse:
    """Readiness: the model weights are loaded and the engine can serve."""
    if not _ready:
        return JSONResponse({"status": "loading"}, status_code=503)
    return JSONResponse({"status": "ok", "model": MODEL_ID})


@app.get("/metrics")
async def metrics() -> PlainTextResponse:
    """Prometheus metrics (vLLM exposes its own; this adds uptime/identity)."""
    body = (
        "# HELP inference_runtime_uptime_seconds Uptime.\n"
        "# TYPE inference_runtime_uptime_seconds gauge\n"
        f"inference_runtime_uptime_seconds {time.time() - _started:.0f}\n"
        f'inference_runtime_ready{{model="{MODEL_ID}"}} {1 if _ready else 0}\n'
    )
    return PlainTextResponse(body, media_type="text/plain; version=0.0.4")


def _render_prompt(tokenizer, messages: list[dict]) -> str:
    """Render chat messages to a prompt via the model's chat template."""
    return tokenizer.apply_chat_template(
        messages, tokenize=False, add_generation_prompt=True
    )


@app.post("/v1/chat/completions")
async def chat_completions(request: Request) -> JSONResponse:
    """Generate a completion for the OpenAI chat request and return it with exact token usage."""
    if _engine is None:
        return JSONResponse({"error": "model not ready"}, status_code=503)
    body = await request.json()
    messages = body.get("messages", [])
    sampling = SamplingParams(
        max_tokens=int(body.get("max_tokens") or 512),
        temperature=float(body.get("temperature", 1.0)),
    )

    tokenizer = await _engine.get_tokenizer()
    prompt = _render_prompt(tokenizer, messages)

    request_id = random_uuid()
    final = None
    async for out in _engine.generate(prompt, sampling, request_id):
        final = out  # keep the last (cumulative) output
    if final is None or not final.outputs:
        return JSONResponse({"error": "generation failed"}, status_code=500)

    text = final.outputs[0].text
    prompt_tokens = len(final.prompt_token_ids or [])
    completion_tokens = len(final.outputs[0].token_ids or [])
    finish = final.outputs[0].finish_reason or "stop"

    return JSONResponse(
        {
            "id": "chatcmpl_" + request_id,
            "object": "chat.completion",
            "created": int(time.time()),
            "model": MODEL_ID,
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": text},
                    "finish_reason": finish,
                }
            ],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
        }
    )
