# Exascale — local dev (Tilt). Run from the repo root: `tilt up` (via `make up` / `make tilt`).
# Builds + deploys services into the k3d cluster with live-reload on save. The data plane +
# observability come from `make data-plane` (manifests/Helm), not Tilt. See docs/plans/LOCAL_DEV.md.

allow_k8s_contexts('k3d-exascale')  # guard: never act on a non-local cluster

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
             resource_deps=['credit-ledger-migrations'])

# Future services register their own docker_build + k8s_yaml + k8s_resource blocks here.
