# Exascale — local dev (Tilt). Run from the repo root: `tilt up` (via `make up` / `make tilt`).
# Builds + deploys services into the k3d cluster with live-reload on save. The data plane +
# observability come from `make data-plane` (manifests/Helm), not Tilt. See docs/plans/LOCAL_DEV.md.

allow_k8s_contexts('k3d-exascale')  # guard: never act on a non-local cluster

# --- shared auth secret (F02/F08) — PLATFORM_JWT_SECRET (platform-core issues with it; credit-ledger
# and inference-gateway verify/mint with it) and SERVICE_TOKEN (the gateway uses it to introspect API
# keys at platform-core). Generated locally on first run, kept in the cluster, never committed. Prod
# sources them from SOPS/Vault.
local_resource(
    'platform-auth',
    cmd='kubectl get secret platform-auth >/dev/null 2>&1 || '
        'kubectl create secret generic platform-auth '
        '--from-literal=PLATFORM_JWT_SECRET=$(openssl rand -base64 48) '
        '--from-literal=SERVICE_TOKEN=$(openssl rand -base64 32)',
)

# --- credit-ledger (F05) — deployed to `default`; talks to the data plane in `data` via FQDN ---
# Migrations ConfigMap (shared types.sql + the service migration) — created out-of-band because it
# sources files from two repo locations; the Deployment's initContainer applies them.
local_resource(
    'credit-ledger-migrations',
    cmd='kubectl create configmap credit-ledger-migrations '
        '--from-file=types.sql=docs/contracts/schemas/types.sql '
        '--from-file=0001_init.sql=services/credit-ledger/migrations/0001_init.sql '
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['docs/contracts/schemas/types.sql', 'services/credit-ledger/migrations/0001_init.sql'],
)
docker_build('exascale/credit-ledger:dev', 'services/credit-ledger',
             dockerfile='services/credit-ledger/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/credit-ledger/base'))
k8s_resource('credit-ledger', port_forwards='8002:8002',
             resource_deps=['credit-ledger-migrations', 'platform-auth'])

# --- platform-core (F02) — identity for the fleet; issues the JWT credit-ledger verifies ---
local_resource(
    'platform-core-migrations',
    cmd='kubectl create configmap platform-core-migrations '
        '--from-file=0001_init.sql=services/platform-core/migrations/0001_init.sql '
        '--from-file=0002_billing.sql=services/platform-core/migrations/0002_billing.sql '
        '--from-file=0003_accounts.sql=services/platform-core/migrations/0003_accounts.sql '
        '--from-file=0004_gaps.sql=services/platform-core/migrations/0004_gaps.sql '
        '--dry-run=client -o yaml | kubectl apply -f -',
    deps=['services/platform-core/migrations/0001_init.sql',
          'services/platform-core/migrations/0002_billing.sql',
          'services/platform-core/migrations/0003_accounts.sql',
          'services/platform-core/migrations/0004_gaps.sql'],
)
docker_build('exascale/platform-core:dev', 'services/platform-core',
             dockerfile='services/platform-core/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/platform-core/base'))
k8s_resource('platform-core', port_forwards='8001:8001',
             resource_deps=['platform-core-migrations', 'platform-auth'])

# --- inference-runtime (F09) — CPU stub locally (real vLLM GPU worker on a GPU node) ---
docker_build('exascale/inference-runtime-stub:dev', 'services/inference-runtime/stub',
             dockerfile='services/inference-runtime/stub/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/inference-runtime/base'))
k8s_resource('inference-runtime', port_forwards='8000:8000')

# --- inference-gateway (F08) — the OpenAI-compatible API; authenticates customers, meters usage ---
docker_build('exascale/inference-gateway:dev', 'services/inference-gateway',
             dockerfile='services/inference-gateway/Dockerfile')
k8s_yaml(kustomize('deploy/k8s/inference-gateway/base'))
k8s_resource('inference-gateway', port_forwards='8085:8085',
             resource_deps=['platform-auth', 'inference-runtime'])  # gateway's vllm backend needs the runtime

# Future services register their own docker_build + k8s_yaml + k8s_resource blocks here.
