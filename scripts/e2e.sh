#!/usr/bin/env bash
# Exascale — end-to-end "time-to-first-action" acceptance test (Decision Gate 2 + immutable
# commitment #3, "sub-5-minute time-to-first-action"). Drives the REAL services over their HTTP APIs
# and asserts the whole first-value loop completes under a 5-minute budget:
#
#   signup → top up credits → first inference (text debit) → first GPU job (gpu_* debit)
#
# This is the API-level harness (fast, CI-friendly, no browser). It assumes the platform stack is up
# (`make up` in another terminal) and opens its own kubectl port-forwards for any service that isn't
# already reachable, cleaning up only the ones it started. A browser-level Playwright variant over the
# Nuxt console can layer on top later (ENGINEERING_STANDARDS §E2E).
#
#   make up         # terminal 1 — cluster + data plane + Tilt
#   make test-e2e   # terminal 2 — this harness
#
# Exit 0 = the loop passed within budget; non-zero = a step failed or the budget was blown.
set -uo pipefail

# ─── config ──────────────────────────────────────────────────────────────────────────────────────
BUDGET_S="${E2E_BUDGET_S:-300}"          # the sub-5-minute contract, in seconds
PLATFORM_CORE_URL="${PLATFORM_CORE_URL:-http://localhost:8001}"
CREDIT_LEDGER_URL="${CREDIT_LEDGER_URL:-http://localhost:8002}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8085}"
COMPUTE_URL="${COMPUTE_URL:-http://localhost:8086}"
PASSWORD="${E2E_PASSWORD:-e2epass123}"
MODEL="${E2E_MODEL:-llama-3.1-8b}"       # a text chat model from the curated catalog
NAMESPACE="${E2E_NAMESPACE:-default}"

PIDS_TO_KILL=()                          # port-forwards we started (cleaned up on exit)
START_NS=$(date +%s%N)

# ─── helpers ─────────────────────────────────────────────────────────────────────────────────────
c_red() { printf '\033[31m%s\033[0m\n' "$1"; }
c_grn() { printf '\033[32m%s\033[0m\n' "$1"; }
c_dim() { printf '\033[2m%s\033[0m\n' "$1"; }

# elapsed seconds since START_NS (integer)
elapsed() { echo $(( ($(date +%s%N) - START_NS) / 1000000000 )); }

cleanup() {
  for pid in "${PIDS_TO_KILL[@]:-}"; do kill "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT

fail() { c_red "✗ FAIL: $1"; c_red "  (after $(elapsed)s)"; exit 1; }

# json field extractor — prefers jq, falls back to python3 (matches seed.sh's bare-toolchain rule)
if command -v jq >/dev/null 2>&1; then
  jget() { jq -r "$1" 2>/dev/null; }
elif command -v python3 >/dev/null 2>&1; then
  jget() { python3 -c 'import sys,json
try: d=json.load(sys.stdin)
except Exception: print(""); sys.exit(0)
k=sys.argv[1].lstrip(".")
print(d.get(k,"") if isinstance(d,dict) else "")' "$1"; }
else
  echo "e2e: need jq or python3"; exit 1
fi

# balance_of <token> <credit_type> → the numeric balance string (or "" if absent)
balance_of() {
  curl -sS "$CREDIT_LEDGER_URL/v1/credits/balances" -H "Authorization: Bearer $1" \
    | python3 -c 'import sys,json
ct=sys.argv[1]
try: d=json.load(sys.stdin)
except Exception: sys.exit(0)
for b in d.get("balances",[]):
    if b.get("credit_type")==ct: print(b.get("balance","")); break' "$2"
}

# ensure <url> <name> reachable; if not, open a port-forward we own. <svc> <localport>
ensure_reachable() { # $1=healthurl $2=name $3=svc $4=port
  if curl -fsS -o /dev/null "$1" 2>/dev/null; then c_dim "  $2 already reachable"; return 0; fi
  c_dim "  port-forwarding $3 → :$4"
  kubectl port-forward -n "$NAMESPACE" "svc/$3" "$4:$4" >/dev/null 2>&1 &
  PIDS_TO_KILL+=("$!")
  for _ in $(seq 1 20); do curl -fsS -o /dev/null "$1" 2>/dev/null && return 0; sleep 1; done
  fail "$2 not reachable at $1 — is \`make up\` running?"
}

# ─── step 0: preflight (not counted against the user-facing TTFA, but bounded) ─────────────────────
echo "── e2e preflight ───────────────────────────────────────────"
command -v kubectl >/dev/null 2>&1 || fail "kubectl not found"
command -v python3 >/dev/null 2>&1 || fail "python3 not found"
ensure_reachable "$PLATFORM_CORE_URL/readyz"  "platform-core"     platform-core     8001
ensure_reachable "$CREDIT_LEDGER_URL/healthz" "credit-ledger"     credit-ledger     8002
ensure_reachable "$GATEWAY_URL/healthz"       "inference-gateway" inference-gateway 8085
ensure_reachable "$COMPUTE_URL/healthz"       "compute-control"   compute-control   8086
SVC_TOKEN=$(kubectl get secret platform-auth -n "$NAMESPACE" -o jsonpath='{.data.SERVICE_TOKEN}' 2>/dev/null | base64 -d)
[ -n "$SVC_TOKEN" ] || fail "could not read SERVICE_TOKEN from the platform-auth secret"

# ── the timed loop starts here ─────────────────────────────────────────────────────────────────────
START_NS=$(date +%s%N)
EMAIL="e2e+$(date +%s)@exascale.local"
echo
echo "── time-to-first-action loop (budget ${BUDGET_S}s) ─────────"
c_dim "  tenant: $EMAIL"

# step 1 — signup → JWT
RESP=$(curl -sS -w '\n%{http_code}' -X POST "$PLATFORM_CORE_URL/v1/auth/signup" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"account_type\":\"business\"}")
CODE=$(printf '%s' "$RESP" | tail -n1)
[ "$CODE" = "201" ] || fail "signup returned $CODE (want 201): $(printf '%s' "$RESP" | sed '$d' | head -c 200)"
TOKEN=$(printf '%s' "$RESP" | sed '$d' | jget .token)
[ -n "$TOKEN" ] && [ "$TOKEN" != "null" ] || fail "signup gave no token"
TID=$(curl -sS "$PLATFORM_CORE_URL/v1/auth/me" -H "Authorization: Bearer $TOKEN" | jget .tenant_id)
[ -n "$TID" ] || fail "could not resolve tenant_id"
c_grn "  ✓ [$(elapsed)s] signup → JWT (tenant $TID)"

# step 2 — top up credits (paper). Mint stands in for the Stripe purchase; the real checkout→webhook
# path is covered by F06's integration test. Idempotency-Key makes a re-run safe.
mint() { # $1=credit_type $2=amount
  local hc
  hc=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$CREDIT_LEDGER_URL/v1/credits/mint" \
    -H 'Content-Type: application/json' -H "X-Dev-Tenant: $TID" -H "Idempotency-Key: e2e-$TID-$1" \
    -d "{\"tenant_id\":\"$TID\",\"credit_type\":\"$1\",\"amount\":\"$2\",\"is_paper\":true,\"reference_id\":\"e2e-$TID-$1\"}")
  [ "$hc" = "200" ] || fail "mint $1 returned $hc"
}
mint text 100000
mint gpu_h100 1000
TEXT_BEFORE=$(balance_of "$TOKEN" text)
GPU_BEFORE=$(balance_of "$TOKEN" gpu_h100)
[ -n "$TEXT_BEFORE" ] && [ -n "$GPU_BEFORE" ] || fail "balances not visible after top-up"
c_grn "  ✓ [$(elapsed)s] top up → text=$TEXT_BEFORE gpu_h100=$GPU_BEFORE"

# step 3 — first inference (text). Gateway verifies the tenant JWT locally.
INF=$(curl -sS -w '\n%{http_code}' -X POST "$GATEWAY_URL/v1/chat/completions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Say hello in one word.\"}]}")
ICODE=$(printf '%s' "$INF" | tail -n1)
[ "$ICODE" = "200" ] || fail "inference returned $ICODE (want 200): $(printf '%s' "$INF" | sed '$d' | head -c 200)"
c_grn "  ✓ [$(elapsed)s] first inference ($MODEL) → 200"

# step 4 — wallet shows the text debit (event-driven; poll until it lands or 30s)
DEBITED=""
for _ in $(seq 1 30); do
  NOW=$(balance_of "$TOKEN" text)
  if [ -n "$NOW" ] && python3 -c "import sys; sys.exit(0 if float('$NOW')<float('$TEXT_BEFORE') else 1)"; then
    DEBITED="$NOW"; break
  fi
  sleep 1
done
[ -n "$DEBITED" ] || fail "text balance never dropped below $TEXT_BEFORE (debit did not land)"
c_grn "  ✓ [$(elapsed)s] wallet debit: text $TEXT_BEFORE → $DEBITED"

# step 5 — first GPU job (F12): schedule a gang, then cancel → compute.usage.v1 → gpu_* debit
JOB=$(curl -sS -X POST "$COMPUTE_URL/v1/compute/jobs" \
  -H "Authorization: Bearer $SVC_TOKEN" -H "X-Tenant-Id: $TID" -H "Idempotency-Key: e2e-$TID-job" \
  -H 'Content-Type: application/json' \
  -d '{"workload_class":"inference","gpu_type":"gpu_h100","gpus":1,"pods":1,"reference_id":"e2e"}')
JOBID=$(printf '%s' "$JOB" | jget .id)
[ -n "$JOBID" ] || fail "compute job submit gave no id: $(printf '%s' "$JOB" | head -c 200)"
sleep 2  # accrue a couple of GPU-seconds
DC=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "$COMPUTE_URL/v1/compute/jobs/$JOBID" -H "Authorization: Bearer $TOKEN")
[ "$DC" = "200" ] || fail "compute job cancel returned $DC"
GPU_DEBITED=""
for _ in $(seq 1 30); do
  NOW=$(balance_of "$TOKEN" gpu_h100)
  if [ -n "$NOW" ] && python3 -c "import sys; sys.exit(0 if float('$NOW')<float('$GPU_BEFORE') else 1)"; then
    GPU_DEBITED="$NOW"; break
  fi
  sleep 1
done
[ -n "$GPU_DEBITED" ] || fail "gpu_h100 balance never dropped below $GPU_BEFORE (compute debit did not land)"
c_grn "  ✓ [$(elapsed)s] GPU job → cancel → debit: gpu_h100 $GPU_BEFORE → $GPU_DEBITED"

# ─── verdict ─────────────────────────────────────────────────────────────────────────────────────
TOTAL=$(elapsed)
echo
echo "─────────────────────────────────────────────────────────────"
if [ "$TOTAL" -le "$BUDGET_S" ]; then
  c_grn "✓ PASS — time-to-first-action ${TOTAL}s ≤ ${BUDGET_S}s budget"
  echo "  signup → top up → inference (text debit) → GPU job (gpu_h100 debit), all live."
  exit 0
else
  fail "time-to-first-action ${TOTAL}s exceeded the ${BUDGET_S}s budget"
fi
