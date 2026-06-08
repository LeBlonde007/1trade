#!/usr/bin/env bash
# Deploy the Exascale SANDBOX (paper mode, CPU-only — no GPU, no real Stripe, no KYC vendor) to a
# single-node k3s cluster, e.g. a VPS. Idempotent: safe to re-run.
#
#   SANDBOX_HOST=sandbox.example.com scripts/deploy-sandbox.sh          # HTTP, k3s already installed
#   INSTALL_K3S=1 SANDBOX_HOST=1.2.3.4 scripts/deploy-sandbox.sh        # also install k3s first
#   TLS=1 ACME_EMAIL=you@example.com SANDBOX_HOST=app.example.com scripts/deploy-sandbox.sh   # + HTTPS
#
# Env:
#   SANDBOX_HOST     (required) DNS name (or IP) the WEB app is served at (e.g. sandbox.example.com).
#   SANDBOX_API_HOST            DNS name the JSON API (/v1/*, for the CLI) is served at. Default
#                              api.$SANDBOX_HOST — override for a custom host (e.g. sandboxapi.example.com).
#   TLS=0|1                    1 → cert-manager + Let's Encrypt (needs real domains + ACME_EMAIL).
#   ACME_EMAIL                 contact email for Let's Encrypt (TLS=1 only).
#   INSTALL_K3S=0|1            1 → install k3s if it's not present.
#   IMPORT=k3s|k3d|none        how locally-built images reach the cluster (default k3s; ignored in no-source).
#   IMAGE_REGISTRY             NO-SOURCE deploy: pull prebuilt images from here (e.g. ghcr.io/<owner>)
#                              instead of building from source on this box. + REGISTRY_USER/REGISTRY_TOKEN
#                              for a private registry. See `make sandbox-bundle` to ship only manifests/SQL.
#   INFERENCE_API_KEY         OPTIONAL. Set it to serve REAL model output via a hosted OpenAI-compatible
#                             provider (OpenRouter by default) instead of the echo-style CPU stub. The key
#                             is injected into the `platform-auth` Secret at deploy — never committed.
#   VLLM_BASE_URL             provider base URL (default https://openrouter.ai/api/v1 when a key is set).
#   INFERENCE_MODEL_MAP       JSON catalog-id→provider-slug map (default maps the two Llama text models).
#   EMAIL_API_TOKEN           OPTIONAL (preferred). Send REAL verification email via an HTTP Email API
#                             (Mailtrap-style, HTTPS/443 — works where the VPS blocks SMTP ports) and gate
#                             login on a verified email. EMAIL_API_URL = send URL (default Mailtrap live
#                             https://send.api.mailtrap.io/api/send; sandbox inbox:
#                             https://sandbox.api.mailtrap.io/api/send/<inbox_id>). EMAIL_FROM default
#                             hello@exascale.ai. Token → platform-auth Secret; never committed.
#   STORAGE_SECRET_KEY        OPTIONAL. Set it (+ STORAGE_ACCESS_KEY, STORAGE_BUCKET) to enable file
#                             uploads to DigitalOcean Spaces (S3-compatible). STORAGE_ENDPOINT (default
#                             nyc3.digitaloceanspaces.com), STORAGE_REGION (nyc3). STORAGE_CDN=true serves
#                             file URLs via the Space's CDN edge (enable the CDN on the Space first);
#                             STORAGE_PUBLIC_BASE overrides the URL base (e.g. a custom CDN subdomain).
#                             Keys → platform-auth Secret. The Space needs CORS to allow PUT from the web
#                             origin ($SANDBOX_HOST), and objects must be public-read for the link to load.
#   SMTP_PASS                 OPTIONAL (SMTP fallback; many VPS block 25/465/587). With it: SMTP_ADDR
#                             (default mail.privateemail.com:465), SMTP_TLS (implicit|starttls), EMAIL_FROM
#                             + SMTP_USER (default hello@exascale.ai). Neither set → no gate (paper demo).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"

SANDBOX_HOST="${SANDBOX_HOST:-}"   # the WEB host (browser); the /v1 API gets its own host (below)
TLS="${TLS:-0}"; ACME_EMAIL="${ACME_EMAIL:-}"
INSTALL_K3S="${INSTALL_K3S:-0}"; IMPORT="${IMPORT:-k3s}"; TAG="${TAG:-sandbox}"
SVCS="platform-core credit-ledger inference-gateway compute-control inference-runtime-stub web"
# NO-SOURCE MODE: set IMAGE_REGISTRY (e.g. ghcr.io/saadallahdev) to PULL prebuilt images instead of
# building from source on this box — so no .go/.vue source ever lands here. For a private registry,
# also set REGISTRY_USER + REGISTRY_TOKEN (a read:packages token) to create an imagePullSecret. Pair
# with `make sandbox-bundle` (ship only manifests + SQL, not the repo). Default empty = build locally.
IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"; REGISTRY_USER="${REGISTRY_USER:-}"; REGISTRY_TOKEN="${REGISTRY_TOKEN:-}"

[ -z "$SANDBOX_HOST" ] && { echo "ERROR: SANDBOX_HOST is required (e.g. SANDBOX_HOST=sandbox.example.com $0)"; exit 2; }
SANDBOX_API_HOST="${SANDBOX_API_HOST:-api.$SANDBOX_HOST}"   # the /v1 API host (CLI/external clients)
SCHEME=http; [ "$TLS" = 1 ] && SCHEME=https
URL="$SCHEME://$SANDBOX_HOST"
API_URL="$SCHEME://$SANDBOX_API_HOST"
SUDO=""; [ "$(id -u)" != 0 ] && SUDO=sudo
say(){ printf "\n\033[1;36m==> %s\033[0m\n" "$*"; }

# 1. (optional) install k3s — single node, ships Traefik as the ingress controller.
if [ "$INSTALL_K3S" = 1 ] && ! command -v k3s >/dev/null 2>&1; then
  say "installing k3s"
  curl -sfL https://get.k3s.io | $SUDO sh -
  $SUDO chmod 644 /etc/rancher/k3s/k3s.yaml || true
fi
[ -f /etc/rancher/k3s/k3s.yaml ] && export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
command -v kubectl >/dev/null || { echo "ERROR: kubectl not found"; exit 1; }
kubectl get nodes >/dev/null 2>&1 || { echo "ERROR: cannot reach a cluster (set KUBECONFIG)"; exit 1; }

# 2 + 3. images. Two mutually-exclusive paths:
#   • NO-SOURCE (IMAGE_REGISTRY set): pull prebuilt images — no `docker build`, no source on this box.
#   • default: build the six images from source here and import them into the cluster's containerd.
if [ -n "$IMAGE_REGISTRY" ]; then
  say "no-source mode — pulling prebuilt images from $IMAGE_REGISTRY (no build on this box)"
  if [ -n "$REGISTRY_TOKEN" ]; then
    # Private registry → imagePullSecret on the namespace's default ServiceAccount, so every pod uses it.
    kubectl create secret docker-registry exascale-pull \
      --docker-server="${IMAGE_REGISTRY%%/*}" \
      --docker-username="${REGISTRY_USER:?REGISTRY_USER is required with REGISTRY_TOKEN}" \
      --docker-password="$REGISTRY_TOKEN" \
      --dry-run=client -o yaml | kubectl apply -f -
    kubectl patch serviceaccount default -p '{"imagePullSecrets":[{"name":"exascale-pull"}]}'
  else
    echo "   (no REGISTRY_TOKEN — assuming the $IMAGE_REGISTRY packages are public)"
  fi
else
  say "building images (:$TAG)"
  docker build -t "exascale/platform-core:$TAG"          services/platform-core
  docker build -t "exascale/credit-ledger:$TAG"          services/credit-ledger
  docker build -t "exascale/inference-gateway:$TAG"      services/inference-gateway
  docker build -t "exascale/compute-control:$TAG"        services/compute-control
  docker build -t "exascale/inference-runtime-stub:$TAG" services/inference-runtime/stub
  # Bundle the CLI binaries into the web image (served at /cli/) — bake the api host into them.
  CLI_API_URL="$API_URL" bash scripts/build-cli-dist.sh
  docker build -t "exascale/web:$TAG"                    "Exascale Frontend"

  case "$IMPORT" in
    k3s)  say "importing images into k3s containerd"
          for i in $SVCS; do docker save "exascale/$i:$TAG" | $SUDO k3s ctr images import -; done ;;
    k3d)  say "importing images into k3d"
          for i in $SVCS; do k3d image import "exascale/$i:$TAG" -c exascale; done ;;
    none) echo "IMPORT=none — images must be pullable from a registry the cluster can reach." ;;
  esac
fi

# 4. out-of-band Secret + migration ConfigMaps (mirrors the Tiltfile; the initContainers consume them).
say "auth secret + migration config"
kubectl get secret platform-auth >/dev/null 2>&1 || kubectl create secret generic platform-auth \
  --from-literal=PLATFORM_JWT_SECRET="$(openssl rand -base64 48)" \
  --from-literal=SERVICE_TOKEN="$(openssl rand -base64 32)"
kubectl create configmap platform-core-migrations --from-file=services/platform-core/migrations/ \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap credit-ledger-migrations \
  --from-file=types.sql=docs/contracts/schemas/types.sql \
  --from-file=0001_init.sql=services/credit-ledger/migrations/0001_init.sql \
  --from-file=0002_conversion.sql=services/credit-ledger/migrations/0002_conversion.sql \
  --dry-run=client -o yaml | kubectl apply -f -

# 5. data plane (namespace `data`).
say "data plane (Postgres · TimescaleDB · Redis · NATS · Mailpit)"
kubectl apply -f deploy/k8s/local/data-plane.yaml
for d in postgres timescaledb redis nats mailpit; do kubectl -n data rollout status "deploy/$d" --timeout=180s; done

# 6. (optional) TLS via cert-manager + Let's Encrypt.
if [ "$TLS" = 1 ]; then
  [ -z "$ACME_EMAIL" ] && { echo "ERROR: TLS=1 needs ACME_EMAIL=you@example.com"; exit 2; }
  say "cert-manager + Let's Encrypt ClusterIssuer"
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
  kubectl -n cert-manager rollout status deploy/cert-manager-webhook --timeout=180s
  kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata: { name: letsencrypt }
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: $ACME_EMAIL
    privateKeySecretRef: { name: letsencrypt-acct }
    solvers: [{ http01: { ingress: { class: traefik } } }]
EOF
fi

# 7. services + web + ingress. Substitute the host/url placeholders; in no-source mode also rewrite the
#    local image names (exascale/<svc>) to the registry ones (<registry>/exascale-<svc>) so the cluster
#    pulls the prebuilt images. Build the sed program in an array so the registry rule is optional.
say "deploying services + web + ingress (web=$SANDBOX_HOST api=$SANDBOX_API_HOST url=$URL)"
SED_ARGS=(-e "s|EXASCALE_SANDBOX_API_HOST|$SANDBOX_API_HOST|g" \
          -e "s|EXASCALE_SANDBOX_HOST|$SANDBOX_HOST|g" \
          -e "s|EXASCALE_SANDBOX_URL|$URL|g")
[ -n "$IMAGE_REGISTRY" ] && SED_ARGS+=(-e "s|image: exascale/|image: $IMAGE_REGISTRY/exascale-|g")
kubectl kustomize deploy/k8s/overlays/sandbox \
  | sed "${SED_ARGS[@]}" \
  | kubectl apply -f -
# No-source redeploys reuse the MOVING :sandbox tag, which k3s caches (pullPolicy defaults to
# IfNotPresent for a non-:latest tag) — so a re-deploy would silently keep the OLD images. Force a
# fresh pull: set imagePullPolicy=Always, then rollout restart so new pods pull the just-built images.
if [ -n "$IMAGE_REGISTRY" ]; then
  say "forcing image re-pull (moving :$TAG tag)"
  for d in platform-core credit-ledger inference-gateway compute-control web; do
    kubectl patch deploy "$d" --type=json \
      -p '[{"op":"add","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Always"}]' >/dev/null 2>&1 || true
    kubectl rollout restart "deploy/$d" >/dev/null 2>&1 || true
  done
fi
for d in platform-core credit-ledger inference-gateway compute-control inference-runtime web; do
  kubectl rollout status "deploy/$d" --timeout=300s
done

# 7b. (optional) hosted inference provider — REAL model output instead of the echo CPU stub.
# Opt-in: only when INFERENCE_API_KEY is set. Delegates to scripts/inference-provider.sh (same flip works
# against the local cluster), which injects the key into the platform-auth Secret + patches the ConfigMap.
if [ -n "${INFERENCE_API_KEY:-}" ]; then
  bash scripts/inference-provider.sh
fi

# 7c. (optional) real transactional email + verify-before-login gate. Two transports; default off keeps
# the paper demo unbricked (no mailer → no gate, immediate login). The verification link points at $URL.
#   • EMAIL_API_TOKEN set → HTTP Email API (Mailtrap-style, HTTPS/443) — the right call when the host
#     blocks SMTP egress ports (most cloud VPS do). EMAIL_API_URL is the send URL.
#   • else SMTP_PASS set  → SMTP (PrivateEmail by default). Note: many VPS block 25/465/587.
if [ -n "${EMAIL_API_TOKEN:-}" ]; then
  EMAIL_API_URL_V="${EMAIL_API_URL:-https://send.api.mailtrap.io/api/send}"
  EMAIL_FROM_V="${EMAIL_FROM:-hello@exascale.ai}"
  say "transactional email via HTTP API $EMAIL_API_URL_V (from $EMAIL_FROM_V) — login gated on email verification"
  kubectl patch secret platform-auth --type merge -p "$(cat <<EOF
stringData:
  EMAIL_API_TOKEN: '$EMAIL_API_TOKEN'
EOF
)"
  kubectl patch configmap platform-core-env --type merge -p "$(cat <<EOF
data:
  EMAIL_API_URL: "$EMAIL_API_URL_V"
  EMAIL_FROM: "$EMAIL_FROM_V"
  REQUIRE_EMAIL_VERIFICATION: "true"
EOF
)"
  kubectl rollout restart deploy/platform-core
  kubectl rollout status deploy/platform-core --timeout=120s
elif [ -n "${SMTP_PASS:-}" ]; then
  SMTP_ADDR_V="${SMTP_ADDR:-mail.privateemail.com:465}"
  SMTP_TLS_V="${SMTP_TLS:-implicit}"
  EMAIL_FROM_V="${EMAIL_FROM:-hello@exascale.ai}"
  SMTP_USER_V="${SMTP_USER:-$EMAIL_FROM_V}"
  say "transactional email via SMTP $SMTP_ADDR_V (from $EMAIL_FROM_V) — login gated on email verification"
  kubectl patch secret platform-auth --type merge -p "$(cat <<EOF
stringData:
  SMTP_USER: '$SMTP_USER_V'
  SMTP_PASS: '$SMTP_PASS'
EOF
)"
  kubectl patch configmap platform-core-env --type merge -p "$(cat <<EOF
data:
  SMTP_ADDR: "$SMTP_ADDR_V"
  SMTP_TLS: "$SMTP_TLS_V"
  EMAIL_FROM: "$EMAIL_FROM_V"
  REQUIRE_EMAIL_VERIFICATION: "true"
EOF
)"
  kubectl rollout restart deploy/platform-core
  kubectl rollout status deploy/platform-core --timeout=120s
fi

# 7d. (optional) object storage for uploads — DigitalOcean Spaces (S3-compatible). Opt-in: only when
# STORAGE_SECRET_KEY is set. Keys go into the platform-auth Secret; endpoint/region/bucket are config.
# Presigned uploads need the Space's CORS to allow PUT from $URL (the web origin).
if [ -n "${STORAGE_SECRET_KEY:-}" ]; then
  STORAGE_ENDPOINT_V="${STORAGE_ENDPOINT:-nyc3.digitaloceanspaces.com}"
  STORAGE_REGION_V="${STORAGE_REGION:-nyc3}"
  say "object storage (Spaces) — bucket ${STORAGE_BUCKET:?set STORAGE_BUCKET} @ $STORAGE_ENDPOINT_V"
  kubectl patch secret platform-auth --type merge -p "$(cat <<EOF
stringData:
  STORAGE_ACCESS_KEY: '${STORAGE_ACCESS_KEY:?set STORAGE_ACCESS_KEY}'
  STORAGE_SECRET_KEY: '$STORAGE_SECRET_KEY'
EOF
)"
  kubectl patch configmap platform-core-env --type merge -p "$(cat <<EOF
data:
  STORAGE_ENDPOINT: "$STORAGE_ENDPOINT_V"
  STORAGE_REGION: "$STORAGE_REGION_V"
  STORAGE_BUCKET: "$STORAGE_BUCKET"
  STORAGE_PUBLIC_BASE: "${STORAGE_PUBLIC_BASE:-}"
  STORAGE_CDN: "${STORAGE_CDN:-false}"
EOF
)"
  kubectl rollout restart deploy/platform-core
  kubectl rollout status deploy/platform-core --timeout=120s
fi

say "sandbox up → web $URL · api $API_URL"
echo "   • DNS: point A records for BOTH $SANDBOX_HOST and $SANDBOX_API_HOST at this node's IP (open 80/443)."
[ "$TLS" != 1 ] && echo "   • HTTP only. For HTTPS re-run with: TLS=1 ACME_EMAIL=you@example.com (both domains must resolve first)."
echo "   • web flow: sign up → buy credits (MockStripe settles instantly) → run inference → see the ledger debit."
echo "   • CLI (the api host, /v1/* → services):  EXASCALE_API_URL=$API_URL exascale login   → balance / catalog / infer / gpu."
echo "   • captured email (verification/receipts) lands in Mailpit: kubectl -n data port-forward svc/mailpit 8025:8025"
