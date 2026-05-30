#!/usr/bin/env python3
"""CPU stub runtime for local dev (F09).

Speaks the same gateway↔runtime contract as the real vLLM worker (OpenAI-compatible
/v1/chat/completions + /healthz /readyz /metrics) but runs on CPU with zero dependencies, so the
gateway's VLLMBackend can be exercised end-to-end in k3d without a GPU. Output is deterministic; the
real vLLM server (../server.py) swaps in on a GPU node with no gateway change.
"""
import json
import os
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

# MODEL_ID is the catalog id this pod claims to serve (mirrors vLLM --served-model-name).
MODEL_ID = os.getenv("MODEL_ID", "llama-3.1-8b")
_started = time.time()
_requests = 0


def count_tokens(text: str) -> int:
    """Rough token estimate (~4 chars/token), matching the gateway's fallback so billing is stable."""
    return max(1, len(text) // 4)


class Handler(BaseHTTPRequestHandler):
    """Serves the runtime contract; one model, deterministic completions."""

    def log_message(self, *_args):
        """Silence default request logging (the gateway logs what matters)."""

    def _send(self, code: int, body: bytes, content_type: str = "application/json"):
        """Write a response with the given status, body, and content type."""
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        """Health, readiness, and Prometheus metrics."""
        if self.path in ("/healthz", "/readyz"):
            self._send(200, b'{"status":"ok"}')
        elif self.path == "/metrics":
            metrics = (
                f"# HELP inference_runtime_requests_total Requests served.\n"
                f"# TYPE inference_runtime_requests_total counter\n"
                f"inference_runtime_requests_total {{model=\"{MODEL_ID}\"}} {_requests}\n"
                f"# HELP inference_runtime_uptime_seconds Uptime.\n"
                f"# TYPE inference_runtime_uptime_seconds gauge\n"
                f"inference_runtime_uptime_seconds {time.time() - _started:.0f}\n"
            )
            self._send(200, metrics.encode(), "text/plain; version=0.0.4")
        else:
            self._send(404, b'{"error":"not found"}')

    def do_POST(self):
        """OpenAI-compatible chat completion (deterministic echo) with token usage."""
        global _requests
        if self.path != "/v1/chat/completions":
            self._send(404, b'{"error":"not found"}')
            return
        length = int(self.headers.get("Content-Length", 0))
        try:
            req = json.loads(self.rfile.read(length) or b"{}")
        except json.JSONDecodeError:
            self._send(400, b'{"error":"bad json"}')
            return
        _requests += 1

        messages = req.get("messages", [])
        last_user = next((m.get("content", "") for m in reversed(messages) if m.get("role") == "user"), "")
        prompt = "\n".join(m.get("role", "") + ": " + m.get("content", "") for m in messages)
        content = f"[stub:{req.get('model', MODEL_ID)}] You said: {last_user[:200]}"

        resp = {
            "id": "chatcmpl_stub_" + str(int(time.time() * 1000)),
            "object": "chat.completion",
            "created": int(time.time()),
            "model": req.get("model", MODEL_ID),
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": content},
                "finish_reason": "stop",
            }],
            "usage": {
                "prompt_tokens": count_tokens(prompt),
                "completion_tokens": count_tokens(content),
                "total_tokens": count_tokens(prompt) + count_tokens(content),
            },
        }
        self._send(200, json.dumps(resp).encode())


def main():
    """Start the threaded HTTP server on :8000."""
    port = int(os.getenv("PORT", "8000"))
    print(json.dumps({"msg": "inference-runtime stub listening", "port": port, "model": MODEL_ID}), flush=True)
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    main()
