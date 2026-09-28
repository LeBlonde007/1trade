# 1Trade — local dev (Tilt). Run from the repo root: `tilt up` (via `make up` / `make tilt`).
# Builds + deploys services into the k3d cluster with live-reload on save. The data plane +
# observability come from `make data-plane` (manifests/Helm), not Tilt. See docs/plans/LOCAL_DEV.md.

allow_k8s_contexts('k3d-1trade')  # guard: never act on a non-local cluster

# --- shared auth secret (F02/F08) — PLATFORM_JWT_SECRET (platform-core issues with it; credit-ledger
# and inference-gateway verify/mint with it) and SERVICE_TOKEN (the gateway uses it to introspect API
# keys at platform-core). Generated locally on first run, kept in the cluster, never committed. The
# committed/prod source of truth is SOPS (deploy/secrets/ — `make secrets-apply`); see its README.
local_resource(
    'platform-auth',
    cmd='kubectl get secret platform-auth >/dev/null 2>&1 || ' +
        'kubectl create secret generic platform-auth ' +
        '--from-literal=PLATFORM_JWT_SECRET=$(openssl rand -base64 48) ' +
        '--from-literal=SERVICE_TOKEN=$(openssl rand -base64 32)',
)

# --- credit-ledger (F05) — deployed to `default`; talks to the data plane in `data` via FQDN ---
# Migrations ConfigMap (shared types.sql + the service migration) — created out-of-band because it
# sources files from two repo locations; the Deployment's initContainer applies them.
local_resource(
    'credit-ledger-migrations',
    # The whole migrations directory, not a per-file list — a new NNNN_*.sql is picked up without
    # editing this file (the initContainer globs them).
    cmd='kubectl create configmap credit-ledger-migrations ' +
        '--from-file=types.sql=docs/contracts/schemas/types.sql ' +
        '--from-file=services/credit-ledger/migrations/ ' +
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['docs/contracts/schemas/types.sql', 'services/credit-ledger/migrations'],
)
# --- ledger-settle — the matching-engine's own token for /v1/credits/settle-trade (ADR-0004). Kept
# out of the shared platform-auth so no other service can settle trades between tenants. ---
local_resource(
    'ledger-settle',
    cmd='kubectl get secret ledger-settle >/dev/null 2>&1 || ' +
        'kubectl create secret generic ledger-settle ' +
        '--from-literal=SETTLE_SERVICE_TOKEN=$(openssl rand -base64 32)',
)
docker_build('1trade/credit-ledger:dev', 'services/credit-ledger',
             dockerfile='services/credit-ledger/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/credit-ledger/base'))
k8s_resource('credit-ledger', port_forwards='8002:8002',
             resource_deps=['credit-ledger-migrations', 'platform-auth', 'ledger-settle'])

# --- platform-core (F02) — identity for the fleet; issues the JWT credit-ledger verifies ---
local_resource(
    'platform-core-migrations',
    # The whole migrations directory (as for credit-ledger) — a new NNNN_*.sql is picked up without
    # editing this file; the initContainer applies them in name order.
    cmd='kubectl create configmap platform-core-migrations ' +
        '--from-file=services/platform-core/migrations/ ' +
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['services/platform-core/migrations'],
)
docker_build('1trade/platform-core:dev', 'services/platform-core',
             dockerfile='services/platform-core/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/platform-core/base'))
k8s_resource('platform-core', port_forwards='8001:8001',
             resource_deps=['platform-core-migrations', 'platform-auth'])

# --- inference-runtime (F09) — CPU stub locally (real vLLM GPU worker on a GPU node) ---
docker_build('1trade/inference-runtime-stub:dev', 'services/inference-runtime/stub',
             dockerfile='services/inference-runtime/stub/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/inference-runtime/base'))
k8s_resource('inference-runtime', port_forwards='8000:8000')

# --- inference-gateway (F08) — the OpenAI-compatible API; authenticates customers, meters usage ---
docker_build('1trade/inference-gateway:dev', 'services/inference-gateway',
             dockerfile='services/inference-gateway/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/inference-gateway/base'))
k8s_resource('inference-gateway', port_forwards='8085:8085',
             resource_deps=['platform-auth', 'inference-runtime'])  # gateway's vllm backend needs the runtime

# --- compute-control (F12) — GPU control plane: catalog + quota + the internal scheduling surface.
# M2 runs the in-memory mock-GPU scheduler; it emits compute.usage.v1 → credit-ledger debits gpu_*. ---
docker_build('1trade/compute-control:dev', 'services/compute-control',
             dockerfile='services/compute-control/Dockerfile')
local_resource(
    'compute-control-migrations',
    cmd='kubectl create configmap compute-control-migrations ' +
        '--from-file=services/compute-control/migrations/ ' +
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['services/compute-control/migrations'],
)
k8s_yaml(kustomize('deploy/k8s/compute-control/base'))
k8s_resource('compute-control', port_forwards='8086:8086',
             resource_deps=['platform-auth', 'compute-control-migrations'])

# --- data-plane UIs — deployed by `make data-plane` (not Tilt), so Tilt can't port-forward them
# itself. These long-running local_resources hold the forwards open for the life of `tilt up` and
# surface clickable links in the Tilt UI. --address 0.0.0.0 so a Windows browser reaches WSL2. ---
local_resource(
    'mailpit-ui',  # captured outbound email (F02 verification) — SMTP :1025, web UI :8025
    serve_cmd='kubectl port-forward -n data --address 0.0.0.0 svc/mailpit 8025:8025',
    links=['http://localhost:8025'],
)

# Future services register their own docker_build + k8s_yaml + k8s_resource blocks here.

# --- surveillance (KW05) — replays + follows the engine's order/trade stream, raises
# surveillance.alert.v1, serves the internal review API. Detects and recommends; never acts. ---
local_resource(
    'surveillance-migrations',
    cmd='kubectl create configmap surveillance-migrations ' +
        '--from-file=services/surveillance/migrations/ ' +
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['services/surveillance/migrations'],
)
docker_build('1trade/surveillance:dev', 'services/surveillance',
             dockerfile='services/surveillance/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/surveillance/base'))
k8s_resource('surveillance', port_forwards='8088:8088',
             resource_deps=['surveillance-migrations', 'platform-auth'])

# --- matching-engine (KW03) — the exchange's order book + reference index. Phase 2 moved from
# keep-warm to ACTIVE (2026-09): the service is built for real, but stays paper-only and the venue
# does not open until the F22 licence clears. Writes are expected to refuse with EXCHANGE_PAUSED. ---
docker_build('1trade/matching-engine:dev', 'services/matching-engine',
             dockerfile='services/matching-engine/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/matching-engine/base'))
k8s_resource('matching-engine', port_forwards='8087:8087',
             resource_deps=['platform-auth'])   # verifies tenant JWTs with the shared secret
