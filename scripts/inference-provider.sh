#!/usr/bin/env bash
# Point the inference-gateway in the CURRENT kube-context cluster at a hosted OpenAI-compatible provider
# (OpenRouter by default) so it serves REAL model output instead of the keyless CPU stub — or revert to
# the stub. Works on the local k3d cluster AND the sandbox VPS (whatever `kubectl config current-context`
# resolves to). Idempotent: safe to re-run; only rolls the gateway when something changed.
#
#   INFERENCE_API_KEY=sk-or-... scripts/inference-provider.sh        # → OpenRouter, default Llama model map
#   INFERENCE_API_KEY=... VLLM_BASE_URL=https://api.groq.com/openai/v1 \
#     INFERENCE_MODEL_MAP='{"llama-3.1-8b":"llama-3.1-8b-instant"}' scripts/inference-provider.sh   # another provider
#   PROVIDER=local scripts/inference-provider.sh                     # → self-hosted small models on our own GPU (Ollama)
#   PROVIDER=stub scripts/inference-provider.sh                      # revert to the keyless in-cluster CPU stub
#
# The key is written ONLY into the in-cluster `platform-auth` Secret (the gateway loads it via envFrom);
# it is never committed. The non-secret URL + model map patch the `inference-gateway-env` ConfigMap.
# Env: NAMESPACE(default current) · VLLM_BASE_URL · LOCAL_LLM_URL · INFERENCE_MODEL_MAP · PROVIDER(provider|local|stub)
set -euo pipefail

PROVIDER="${PROVIDER:-provider}"
NS_ARG=""; [ -n "${NAMESPACE:-}" ] && NS_ARG="-n $NAMESPACE"
say(){ printf "\n\033[1;36m==> %s\033[0m\n" "$*"; }

command -v kubectl >/dev/null || { echo "ERROR: kubectl not found"; exit 1; }
kubectl get deploy inference-gateway $NS_ARG >/dev/null 2>&1 \
  || { echo "ERROR: inference-gateway not found in this cluster/context ($(kubectl config current-context 2>/dev/null))"; exit 1; }

if [ "$PROVIDER" = "local" ]; then
  # SELF-HOSTED: serve real models from our own GPU instead of a hosted provider. The server is any
  # OpenAI-compatible runtime on the host (Ollama by default, or vLLM on :8000); the cluster reaches
  # it through k3d's host alias. No API key — a local runtime does not authenticate, and the gateway
  # skips the Authorization header when the key is empty.
  #
  #   scripts/inference-provider.sh PROVIDER=local                      # Ollama on the host
  #   LOCAL_LLM_URL=http://host.k3d.internal:8000 PROVIDER=local ...    # a local vLLM instead
  #
  # NOTE the URL must NOT end in /v1 — the gateway appends /v1/chat/completions itself.
  LOCAL_URL="${LOCAL_LLM_URL:-http://host.k3d.internal:11434}"
  # Only the models we actually serve locally. Catalog ids map 1:1 to the runtime's own tags, so the
  # playground shows the true model — no pretending a 1B is a 70B. Ids absent here are simply not
  # served by this backend.
  LOCAL_DEFAULT_MAP='{"llama-3.2-1b":"llama3.2:1b","qwen2.5-1.5b":"qwen2.5:1.5b"}'
  LOCAL_MAP="${INFERENCE_MODEL_MAP:-$LOCAL_DEFAULT_MAP}"
  say "inference-gateway → self-hosted runtime $LOCAL_URL (real models, our GPU)"
  kubectl patch secret platform-auth $NS_ARG --type merge -p "$(cat <<'EOF'
stringData:
  INFERENCE_API_KEY: ''
EOF
)"
  kubectl patch configmap inference-gateway-env $NS_ARG --type merge -p "$(cat <<EOF
data:
  INFERENCE_BACKEND: "vllm"
  VLLM_BASE_URL: "$LOCAL_URL"
  INFERENCE_MODEL_MAP: '$LOCAL_MAP'
EOF
)"
elif [ "$PROVIDER" = "stub" ]; then
  # Revert: keyless in-cluster CPU stub. Empty key → gateway runs the keyless path; empty map → pass-through.
  say "reverting inference-gateway → keyless CPU stub (http://inference-runtime:8000)"
  kubectl patch secret platform-auth $NS_ARG --type merge -p "$(cat <<'EOF'
stringData:
  INFERENCE_API_KEY: ''
EOF
)"
  kubectl patch configmap inference-gateway-env $NS_ARG --type merge -p "$(cat <<'EOF'
data:
  INFERENCE_BACKEND: "vllm"
  VLLM_BASE_URL: "http://inference-runtime:8000"
  INFERENCE_MODEL_MAP: ''
EOF
)"
else
  [ -n "${INFERENCE_API_KEY:-}" ] || { echo "ERROR: set INFERENCE_API_KEY=... (or PROVIDER=stub to revert)"; exit 2; }
  # VLLM_BASE_URL is the part BEFORE /v1/chat/completions — the gateway appends that path itself. So it
  # must NOT already end in /v1 (that doubles to /v1/v1/... → 404). OpenRouter → https://openrouter.ai/api
  # DigitalOcean serverless inference → https://inference.do-ai.run · local vLLM runtime → http://host:8000
  PROVIDER_URL="${VLLM_BASE_URL:-https://openrouter.ai/api}"
  # JSON catalog-id→provider-slug map. The default is assigned to a var FIRST: writing the default
  # inline as ${INFERENCE_MODEL_MAP:-{...}} leaks a literal '}' past the parameter expansion when the
  # var is set (doubling the closing brace → invalid JSON → the gateway loads zero entries). When the
  # provider is DigitalOcean, default to its serverless slugs so the whole catalog works zero-config.
  case "$PROVIDER_URL" in
    # Media slugs target DigitalOcean's *base* serverless tier: only stable-diffusion-3.5-large (image),
    # qwen3-tts-voicedesign (speech), wan2-2-t2v-a14b (video) and the open text models are available
    # there. Flux / ElevenLabs aren't on DO, and openai-gpt-image is tier-gated (403), so those catalog
    # ids route to the available equivalents. On a higher DO tier, override via INFERENCE_MODEL_MAP.
    *do-ai.run*) DEFAULT_MAP='{"llama-3.1-8b":"llama-4-maverick","llama-3.1-70b":"llama3.3-70b-instruct","claude-opus-4.8":"anthropic-claude-opus-4.8","claude-sonnet-4.5":"anthropic-claude-4.5-sonnet","gpt-5":"openai-gpt-5","gpt-4o":"openai-gpt-4o","deepseek-v3.2":"deepseek-3.2","qwen3-32b":"alibaba-qwen3-32b","nemotron-vision":"nemotron-nano-12b-v2-vl","stable-diffusion-3.5":"stable-diffusion-3.5-large","flux-schnell":"stable-diffusion-3.5-large","gpt-image-1.5":"stable-diffusion-3.5-large","elevenlabs-tts":"qwen3-tts-voicedesign","qwen3-tts":"qwen3-tts-voicedesign","wan-t2v":"wan2-2-t2v-a14b","bge-m3":"bge-m3","e5-large":"e5-large-v2"}' ;;
    *)           DEFAULT_MAP='{"llama-3.1-70b":"meta-llama/llama-3.1-70b-instruct","llama-3.1-8b":"meta-llama/llama-3.1-8b-instruct"}' ;;
  esac
  MODEL_MAP="${INFERENCE_MODEL_MAP:-$DEFAULT_MAP}"
  say "inference-gateway → hosted provider $PROVIDER_URL (real model output; stub bypassed)"
  kubectl patch secret platform-auth $NS_ARG --type merge -p "$(cat <<EOF
stringData:
  INFERENCE_API_KEY: '$INFERENCE_API_KEY'
EOF
)"
  kubectl patch configmap inference-gateway-env $NS_ARG --type merge -p "$(cat <<EOF
data:
  INFERENCE_BACKEND: "vllm"
  VLLM_BASE_URL: "$PROVIDER_URL"
  INFERENCE_MODEL_MAP: '$MODEL_MAP'
EOF
)"
fi

# Restart so the new env is picked up (ConfigMap/Secret env changes don't roll pods on their own).
kubectl rollout restart deploy/inference-gateway $NS_ARG
kubectl rollout status  deploy/inference-gateway $NS_ARG --timeout=120s
say "done — current backend:"
kubectl get configmap inference-gateway-env $NS_ARG -o jsonpath='  INFERENCE_BACKEND={.data.INFERENCE_BACKEND}  VLLM_BASE_URL={.data.VLLM_BASE_URL}{"\n"}'
