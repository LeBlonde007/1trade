#!/usr/bin/env bash
# Exascale — seed dev fixtures: a few tenants/users (platform-core, F02) each topped up with paper
# credits across the catalog credit-types (credit-ledger, F05). Idempotent: re-running logs in to an
# existing user instead of failing, and mint is anchored so repeats don't double-credit.
#
# Talks to the services over their Tilt port-forwards, so `make up` (Tilt running) must be live in
# another terminal first. Override the URLs via env if you forward them elsewhere.
#
#   make up      # terminal 1 — cluster + data plane + Tilt (keeps port-forwards open)
#   make seed    # terminal 2 — this script
#
# The catalog itself is static (hardcoded in inference-gateway), so there is nothing to seed there.
set -euo pipefail

PLATFORM_CORE_URL="${PLATFORM_CORE_URL:-http://localhost:8001}"
CREDIT_LEDGER_URL="${CREDIT_LEDGER_URL:-http://localhost:8002}"
PASSWORD="${SEED_PASSWORD:-devpass123}"

# --- json helper: prefer jq, fall back to python3 so the script runs on a bare toolchain ----------
if command -v jq >/dev/null 2>&1; then
  json_get() { jq -r "$1" 2>/dev/null; }            # $1 = jq filter, e.g. .token
elif command -v python3 >/dev/null 2>&1; then
  json_get() { python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get(sys.argv[1].lstrip("."),""))' "$1"; }
else
  echo "seed: need jq or python3 to parse responses" >&2; exit 1
fi

# --- wait for the services to answer (port-forwards may still be warming up) ----------------------
wait_for() { # $1 = url, $2 = name
  local i
  for i in $(seq 1 30); do
    if curl -fsS -o /dev/null "$1" 2>/dev/null; then return 0; fi
    sleep 1
  done
  echo "seed: $2 not reachable at $1 — is \`make up\` (Tilt) running? Aborting." >&2
  exit 1
}
wait_for "$PLATFORM_CORE_URL/readyz" "platform-core"
wait_for "$CREDIT_LEDGER_URL/healthz" "credit-ledger"

# --- signup (or login if the email already exists) → returns a bearer token ----------------------
get_token() { # $1 = email, $2 = tenant_name; echoes the JWT
  local email="$1" tenant="$2" resp code
  resp=$(curl -sS -w '\n%{http_code}' -X POST "$PLATFORM_CORE_URL/v1/auth/signup" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\",\"tenant_name\":\"$tenant\"}")
  code=$(printf '%s' "$resp" | tail -n1)
  if [ "$code" = "201" ]; then
    printf '%s' "$resp" | sed '$d' | json_get .token
    return 0
  fi
  # 409 (email_taken) or anything else → fall back to login so re-runs are idempotent.
  curl -sS -X POST "$PLATFORM_CORE_URL/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}" | json_get .token
}

# --- resolve the tenant_id for a token (mint needs it in the body) -------------------------------
tenant_of() { # $1 = token
  curl -sS "$PLATFORM_CORE_URL/v1/auth/me" -H "Authorization: Bearer $1" | json_get .tenant_id
}

# --- mint paper credits of one type into a tenant ------------------------------------------------
# Uses the dev-only X-Dev-Tenant shortcut to satisfy the ledger's service auth without a service
# token. reference_id + Idempotency-Key make the movement safe to repeat across re-runs.
mint() { # $1 = tenant_id, $2 = credit_type, $3 = amount
  curl -sS -o /dev/null -w "    %{http_code}  $2  $3\n" -X POST "$CREDIT_LEDGER_URL/v1/credits/mint" \
    -H 'Content-Type: application/json' \
    -H "X-Dev-Tenant: $1" \
    -H "Idempotency-Key: seed-$1-$2" \
    -d "{\"tenant_id\":\"$1\",\"credit_type\":\"$2\",\"amount\":\"$3\",\"is_paper\":true,\"reference_id\":\"seed-$1-$2\"}"
}

# --- seed one tenant with a spread of credits ----------------------------------------------------
seed_tenant() { # $1 = email, $2 = tenant_name, then pairs: credit_type amount ...
  local email="$1" tenant="$2"; shift 2
  echo "==> $tenant <$email>"
  local token tid
  token=$(get_token "$email" "$tenant")
  if [ -z "$token" ] || [ "$token" = "null" ]; then
    echo "    failed to obtain token (signup+login both failed) — skipping" >&2
    return 1
  fi
  tid=$(tenant_of "$token")
  if [ -z "$tid" ] || [ "$tid" = "null" ]; then
    echo "    failed to resolve tenant_id — skipping mints" >&2
    return 1
  fi
  echo "    tenant_id=$tid  password=$PASSWORD"
  while [ "$#" -ge 2 ]; do
    mint "$tid" "$1" "$2"
    shift 2
  done
}

# Credit types come from docs/contracts/credit-types.md: ai_index, text, speech, image, video,
# embeddings, gpu_h100, gpu_h200. Amounts are paper (sandbox) credits, fixed-point strings.
seed_tenant "alice@exascale.local" "Acme AI" \
  text 5000000 embeddings 2000000 image 500000

seed_tenant "bob@exascale.local" "Globex Labs" \
  gpu_h100 1000 gpu_h200 500 text 1000000

seed_tenant "demo@exascale.local" "Exascale Demo" \
  ai_index 1000000 text 5000000 speech 1000000 image 1000000 \
  video 200000 embeddings 5000000 gpu_h100 2000 gpu_h200 1000

echo
echo "seed: done. Log in at the web app with any of the emails above / password '$PASSWORD'."
