#!/usr/bin/env bash
# Exascale — install the local data plane + scheduling + observability into the k3d cluster.
# Idempotent (helm upgrade --install). Pulls several images; first run takes a few minutes.
# Invoked by `make data-plane` / `make up`. GPU=1 also installs the NVIDIA GPU Operator.
set -euo pipefail

GPU="${GPU:-0}"
ns() { kubectl get ns "$1" >/dev/null 2>&1 || kubectl create ns "$1"; }

echo "==> helm repos"
helm repo add cnpg https://cloudnative-pg.github.io/charts >/dev/null 2>&1 || true
helm repo add bitnami https://charts.bitnami.com/bitnami >/dev/null 2>&1 || true
helm repo add nats https://nats-io.github.io/k8s/helm/charts >/dev/null 2>&1 || true
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
helm repo add grafana https://grafana.github.io/helm-charts >/dev/null 2>&1 || true
helm repo update >/dev/null

echo "==> data plane"
ns data
# PostgreSQL (state) — CloudNativePG operator; a Cluster CR is added per-env later.
helm upgrade --install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace --wait
# Redis (order books / cache) + NATS JetStream (events).
helm upgrade --install redis bitnami/redis -n data --set auth.enabled=false --wait
helm upgrade --install nats nats/nats -n data --set config.jetstream.enabled=true --wait

echo "==> batch scheduling (Kueue + Volcano)"
ns scheduling
helm upgrade --install kueue oci://registry.k8s.io/kueue/charts/kueue -n scheduling --wait || \
  echo "  (kueue chart unavailable offline — skip; install later)"
helm upgrade --install volcano bitnami/volcano -n scheduling --wait 2>/dev/null || \
  echo "  (volcano via bitnami unavailable — install from volcano.sh manifests later)"

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

echo "==> data plane ready. (Mock-GPU mode unless GPU=1.)"
