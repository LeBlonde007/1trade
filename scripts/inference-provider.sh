#!/usr/bin/env bash
# Point the inference-gateway in the CURRENT kube-context cluster at a hosted OpenAI-compatible provider
# (OpenRouter by default) so it serves REAL model output instead of the keyless CPU stub — or revert to
# the stub. Works on the local k3d cluster AND the sandbox VPS (whatever `kubectl config current-context`
# resolves to). Idempotent: safe to re-run; only rolls the gateway when something changed.
#
#   INFERENCE_API_KEY=sk-or-... scripts/inference-provider.sh        # → OpenRouter, default Llama model map
#   INFERENCE_API_KEY=... VLLM_BASE_URL=https://api.groq.com/openai/v1 \
#     INFERENCE_MODEL_MAP='{"llama-3.1-8b":"llama-3.1-8b-instant"}' scripts/inference-provider.sh   # another provider
#   PROVIDER=stub scripts/inference-provider.sh                      # revert to the keyless in-cluster CPU stub
#
# The key is written ONLY into the in-cluster `platform-auth` Secret (the gateway loads it via envFrom);
# it is never committed. The non-secret URL + model map patch the `inference-gateway-env` ConfigMap.
# Env: NAMESPACE(default current) · VLLM_BASE_URL · INFERENCE_MODEL_MAP · PROVIDER(provider|stub)
set -euo pipefail

PROVIDER="${PROVIDER:-provider}"
NS_ARG=""; [ -n "${NAMESPACE:-}" ] && NS_ARG="-n $NAMESPACE"
say(){ printf "\n\033[1;36m==> %s\033[0m\n" "$*"; }

command -v kubectl >/dev/null || { echo "ERROR: kubectl not found"; exit 1; }
kubectl get deploy inference-gateway $NS_ARG >/dev/null 2>&1 \
  || { echo "ERROR: inference-gateway not found in this cluster/context ($(kubectl config current-context 2>/dev/null))"; exit 1; }

if [ "$PROVIDER" = "stub" ]; then
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
  # JSON catalog-id→provider-slug map; YAML-single-quoted below so its double quotes need no escaping.
  # Defaults are OpenRouter's PAID Llama slugs (the ":free" ones were retired → 404). For another
  # provider set VLLM_BASE_URL + INFERENCE_MODEL_MAP to its slugs (run GET <base>/v1/models to list them).
  MODEL_MAP="${INFERENCE_MODEL_MAP:-{\"llama-3.1-70b\":\"meta-llama/llama-3.1-70b-instruct\",\"llama-3.1-8b\":\"meta-llama/llama-3.1-8b-instruct\"}}"
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
