#!/usr/bin/env bash
# Exascale — install the local data plane into the k3d cluster. Two tiers:
#   default : the CORE (Postgres, TimescaleDB, Redis, NATS) from data-plane.yaml — fast + reliable.
#   FULL=1  : also scheduling (Kueue, Volcano) + observability (Prometheus, Loki, Tempo) via Helm.
#   GPU=1   : also the NVIDIA GPU Operator.
# Idempotent. Invoked by `make data-plane` / `make up`. See docs/plans/DECISIONS.md ADR-0001.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
GPU="${GPU:-0}"; FULL="${FULL:-0}"

echo "==> core data plane (Postgres · TimescaleDB · Redis · NATS · Mailpit)"
kubectl apply -f "$HERE/data-plane.yaml"
echo "   waiting for rollouts..."
for d in postgres timescaledb redis nats mailpit; do
  kubectl -n data rollout status "deploy/$d" --timeout=180s
done
echo "   core ready."

if [ "$FULL" != "1" ]; then
  echo "==> skipping scheduling + observability (set FULL=1 to install them via Helm)."
  exit 0
fi

ns() { kubectl get ns "$1" >/dev/null 2>&1 || kubectl create ns "$1"; }
echo "==> helm repos"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
helm repo add grafana https://grafana.github.io/helm-charts >/dev/null 2>&1 || true
helm repo update >/dev/null

echo "==> scheduling (Kueue + Volcano)"
ns scheduling
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/latest/download/manifests.yaml || \
  echo "   (kueue manifests unavailable — install later)"
kubectl apply -f https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml || \
  echo "   (volcano manifests unavailable — install later)"

echo "==> observability (Prometheus + Loki + Tempo + Grafana)"
ns observability
helm upgrade --install kube-prom prometheus-community/kube-prometheus-stack -n observability --wait
helm upgrade --install loki grafana/loki-stack -n observability --wait || true
helm upgrade --install tempo grafana/tempo -n observability --wait || true

if [ "$GPU" = "1" ]; then
  echo "==> NVIDIA GPU Operator (GPU=1)"
  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia >/dev/null 2>&1 || true
  helm repo update >/dev/null
  helm upgrade --install gpu-operator nvidia/gpu-operator -n gpu-operator --create-namespace --wait
fi
echo "==> data plane ready (FULL)."
