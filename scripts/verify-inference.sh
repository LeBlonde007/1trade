#!/usr/bin/env bash
# Live inference smoke test — drive ONE real chat completion through the gateway and show (a) the model's
# actual text, (b) the credit-ledger text debit it produced, and (c) which backend served it. Backend-
# agnostic: run it against the keyless CPU stub (echo-style output) or a hosted provider (OpenRouter →
# real Llama). Pairs with scripts/inference-provider.sh, which flips the backend.
#
#   scripts/verify-inference.sh                 # uses the running local cluster (make up) via port-forwards
#   PROMPT="Explain MoE in one sentence." scripts/verify-inference.sh
#
# Assumes the platform stack is up (platform-core, credit-ledger, inference-gateway). Opens its own
# kubectl port-forwards for anything not already reachable and cleans up only the ones it started.
set -uo pipefail

PLATFORM_CORE_URL="${PLATFORM_CORE_URL:-http://localhost:8001}"
CREDIT_LEDGER_URL="${CREDIT_LEDGER_URL:-http://localhost:8002}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8085}"
MODEL="${MODEL:-llama-3.1-8b}"
PROMPT="${PROMPT:-In one short sentence, what is a GPU?}"
PASSWORD="${PASSWORD:-verifypass123}"
EMAIL="${EMAIL:-verify+$(date +%s)@exascale.local}"   # throwaway tenant per run (re-runnable)
NAMESPACE="${NAMESPACE:-default}"
PIDS=(); trap 'for p in "${PIDS[@]:-}"; do kill "$p" 2>/dev/null || true; done' EXIT

red(){ printf '\033[31m%s\033[0m\n' "$1"; }; grn(){ printf '\033[32m%s\033[0m\n' "$1"; }
dim(){ printf '\033[2m%s\033[0m\n' "$1"; }; bold(){ printf '\033[1m%s\033[0m\n' "$1"; }
fail(){ red "✗ $1"; exit 1; }

# json extractor — jq if present, else python3 (matches seed.sh's bare-toolchain rule).
if command -v jq >/dev/null 2>&1; then jget(){ jq -r "$1" 2>/dev/null; }
elif command -v python3 >/dev/null 2>&1; then jget(){ python3 -c 'import sys,json
try: d=json.load(sys.stdin)
except Exception: print(""); sys.exit(0)
cur=d
for k in sys.argv[1].lstrip(".").split("."):
  if k.endswith("]"):
    k,i=k[:-1].split("["); cur=cur.get(k,[]) if isinstance(cur,dict) else cur; cur=cur[int(i)] if isinstance(cur,list) and len(cur)>int(i) else ""
  else:
    cur=cur.get(k,"") if isinstance(cur,dict) else ""
print(cur if isinstance(cur,(str,int,float)) else "")' "$1"; }
else fail "need jq or python3"; fi

# ensure_reachable URL NAME LOCALPORT — port-forward svc/NAME if the URL isn't already answering.
ensure_reachable(){ local url="$1" name="$2" port="$3"
  if curl -fsS -o /dev/null "$url" 2>/dev/null; then return 0; fi
  dim "  port-forward svc/$name $port:$port"
  kubectl -n "$NAMESPACE" port-forward "svc/$name" "$port:$port" >/dev/null 2>&1 & PIDS+=("$!")
  for _ in $(seq 1 30); do curl -fsS -o /dev/null "$url" 2>/dev/null && return 0; sleep 0.3; done
  fail "$name not reachable at $url"
}

bold "== live inference verify =="
# which backend is the gateway on right now? (informational)
BK=$(kubectl -n "$NAMESPACE" get configmap inference-gateway-env -o jsonpath='{.data.INFERENCE_BACKEND}' 2>/dev/null)
URL=$(kubectl -n "$NAMESPACE" get configmap inference-gateway-env -o jsonpath='{.data.VLLM_BASE_URL}' 2>/dev/null)
KEYED=$(kubectl -n "$NAMESPACE" get secret platform-auth -o jsonpath='{.data.INFERENCE_API_KEY}' 2>/dev/null | base64 -d 2>/dev/null)
dim "  backend=$BK  vllm_base_url=${URL:-<unset>}  hosted_key=$([ -n "$KEYED" ] && echo yes || echo no)"

ensure_reachable "$PLATFORM_CORE_URL/healthz" platform-core 8001
ensure_reachable "$CREDIT_LEDGER_URL/healthz" credit-ledger 8002
ensure_reachable "$GATEWAY_URL/healthz"       inference-gateway 8085

# 1. signup → JWT (the gateway verifies this tenant JWT locally; no API key needed).
RESP=$(curl -sS -w '\n%{http_code}' -X POST "$PLATFORM_CORE_URL/v1/auth/signup" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"tenant_name\":\"Verify Co\"}")
CODE=$(printf '%s' "$RESP" | tail -n1)
if [ "$CODE" = "201" ]; then TOKEN=$(printf '%s' "$RESP" | sed '$d' | jget .token)
else TOKEN=$(curl -sS -X POST "$PLATFORM_CORE_URL/v1/auth/login" -H 'Content-Type: application/json' \
       -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | jget .token); fi
[ -n "$TOKEN" ] && [ "$TOKEN" != "null" ] || fail "could not obtain a tenant JWT"
TID=$(curl -sS "$PLATFORM_CORE_URL/v1/auth/me" -H "Authorization: Bearer $TOKEN" | jget .tenant_id)
[ -n "$TID" ] || fail "could not resolve tenant_id"
grn "  ✓ tenant $TID"

# 2. mint text credits (dev X-Dev-Tenant shortcut) so the gateway's preflight passes.
curl -sS -o /dev/null -X POST "$CREDIT_LEDGER_URL/v1/credits/mint" -H 'Content-Type: application/json' \
  -H "X-Dev-Tenant: $TID" -H "Idempotency-Key: verify-$TID-text" \
  -d "{\"tenant_id\":\"$TID\",\"credit_type\":\"text\",\"amount\":\"1000000\",\"is_paper\":true,\"reference_id\":\"verify-$TID-text\"}"
bal(){ curl -sS "$CREDIT_LEDGER_URL/v1/credits/balances" -H "Authorization: Bearer $1" | jget '.balances[0].balance' 2>/dev/null || true; }
BEFORE=$(curl -sS "$CREDIT_LEDGER_URL/v1/credits/balances" -H "Authorization: Bearer $TOKEN" \
  | (command -v jq >/dev/null && jq -r '.balances[]|select(.credit_type=="text")|.balance' || cat) 2>/dev/null | head -n1)
grn "  ✓ text balance before: ${BEFORE:-?}"

# 3. the actual inference call.
bold "  → POST /v1/chat/completions  model=$MODEL"
dim  "    prompt: $PROMPT"
INF=$(curl -sS -w '\n%{http_code}' -X POST "$GATEWAY_URL/v1/chat/completions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"$PROMPT\"}]}")
ICODE=$(printf '%s' "$INF" | tail -n1); BODY=$(printf '%s' "$INF" | sed '$d')
[ "$ICODE" = "200" ] || fail "inference returned $ICODE: $(printf '%s' "$BODY" | head -c 300)"
CONTENT=$(printf '%s' "$BODY" | jget '.choices[0].message.content')
[ -n "$CONTENT" ] || fail "200 but empty content: $(printf '%s' "$BODY" | head -c 300)"
echo; bold "  ── model response ─────────────────────────────"; echo "  $CONTENT"; bold "  ───────────────────────────────────────────────"; echo

# 4. confirm the ledger debited text credits (event-driven; poll up to 30s).
DEBITED=""
for _ in $(seq 1 30); do
  NOW=$(curl -sS "$CREDIT_LEDGER_URL/v1/credits/balances" -H "Authorization: Bearer $TOKEN" \
    | (command -v jq >/dev/null && jq -r '.balances[]|select(.credit_type=="text")|.balance' || cat) 2>/dev/null | head -n1)
  if [ -n "$NOW" ] && [ -n "$BEFORE" ] && awk "BEGIN{exit !($NOW<$BEFORE)}"; then DEBITED="$NOW"; break; fi
  sleep 1
done
[ -n "$DEBITED" ] || fail "text balance never dropped from $BEFORE (debit did not land)"
grn "  ✓ ledger debit: text $BEFORE → $DEBITED"
echo; grn "PASS — live inference + debit verified (backend=$BK${KEYED:+, hosted provider})"
