#!/usr/bin/env bash
# Deploy the Exascale SANDBOX (paper mode, CPU-only — no GPU, no real Stripe, no KYC vendor) to a
# single-node k3s cluster, e.g. a VPS. Idempotent: safe to re-run.
#
#   SANDBOX_HOST=sandbox.example.com scripts/deploy-sandbox.sh          # HTTP, k3s already installed
#   INSTALL_K3S=1 SANDBOX_HOST=1.2.3.4 scripts/deploy-sandbox.sh        # also install k3s first
#   TLS=1 ACME_EMAIL=you@example.com SANDBOX_HOST=app.example.com scripts/deploy-sandbox.sh   # + HTTPS
#
# Env:
#   SANDBOX_HOST  (required)  DNS name (or IP) the app is served at.
#   TLS=0|1                   1 → cert-manager + Let's Encrypt (needs a real domain + ACME_EMAIL).
#   ACME_EMAIL                contact email for Let's Encrypt (TLS=1 only).
#   INSTALL_K3S=0|1           1 → install k3s if it's not present.
#   IMPORT=k3s|k3d|none       how locally-built images reach the cluster (default k3s).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"

SANDBOX_HOST="${SANDBOX_HOST:-}"
TLS="${TLS:-0}"; ACME_EMAIL="${ACME_EMAIL:-}"
INSTALL_K3S="${INSTALL_K3S:-0}"; IMPORT="${IMPORT:-k3s}"; TAG="${TAG:-sandbox}"
SVCS="platform-core credit-ledger inference-gateway compute-control inference-runtime-stub web"

[ -z "$SANDBOX_HOST" ] && { echo "ERROR: SANDBOX_HOST is required (e.g. SANDBOX_HOST=sandbox.example.com $0)"; exit 2; }
SCHEME=http; [ "$TLS" = 1 ] && SCHEME=https
URL="$SCHEME://$SANDBOX_HOST"
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

# 2. build the six images.
say "building images (:$TAG)"
docker build -t "exascale/platform-core:$TAG"          services/platform-core
docker build -t "exascale/credit-ledger:$TAG"          services/credit-ledger
docker build -t "exascale/inference-gateway:$TAG"      services/inference-gateway
docker build -t "exascale/compute-control:$TAG"        services/compute-control
docker build -t "exascale/inference-runtime-stub:$TAG" services/inference-runtime/stub
docker build -t "exascale/web:$TAG"                    "Exascale Frontend"

# 3. make the images visible to the cluster (k3s runs its own containerd).
case "$IMPORT" in
  k3s)  say "importing images into k3s containerd"
        for i in $SVCS; do docker save "exascale/$i:$TAG" | $SUDO k3s ctr images import -; done ;;
  k3d)  say "importing images into k3d"
        for i in $SVCS; do k3d image import "exascale/$i:$TAG" -c exascale; done ;;
  none) echo "IMPORT=none — images must be pullable from a registry the cluster can reach." ;;
esac

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

# 7. services + web + ingress (substitute the host/url placeholders into the rendered overlay).
say "deploying services + web + ingress (host=$SANDBOX_HOST url=$URL)"
kubectl kustomize deploy/k8s/overlays/sandbox \
  | sed -e "s|EXASCALE_SANDBOX_HOST|$SANDBOX_HOST|g" -e "s|EXASCALE_SANDBOX_URL|$URL|g" \
  | kubectl apply -f -
for d in platform-core credit-ledger inference-gateway compute-control inference-runtime web; do
  kubectl rollout status "deploy/$d" --timeout=300s
done

say "sandbox up → $URL"
echo "   • point an A record for $SANDBOX_HOST at this node's public IP (open ports 80/443)."
[ "$TLS" != 1 ] && echo "   • HTTP only. For HTTPS re-run with: TLS=1 ACME_EMAIL=you@example.com (needs a real domain)."
echo "   • flow: sign up → buy credits (MockStripe settles instantly) → run inference → see the ledger debit."
echo "   • captured email (verification/receipts) lands in Mailpit: kubectl -n data port-forward svc/mailpit 8025:8025"
