#!/usr/bin/env bash
# Exascale — install the local data plane into the k3d cluster. Tiers (compose them):
#   default : the CORE (Postgres, TimescaleDB, Redis, NATS, Mailpit) from data-plane.yaml — fast.
#   SCHED=1 : also scheduling — Kueue + Volcano + the mock-GPU resource + the project Kueue config.
#   OBS=1   : also observability — Prometheus + Loki + Tempo + Grafana (heavy, Helm).
#   FULL=1  : SCHED + OBS.
#   GPU=1   : also the NVIDIA GPU Operator (needs a real GPU).
# Idempotent. Invoked by `make data-plane` / `make up` / `make sched`. See DECISIONS.md ADR-0001.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
GPU="${GPU:-0}"; FULL="${FULL:-0}"; SCHED="${SCHED:-0}"; OBS="${OBS:-0}"
# FULL is the convenience switch for SCHED + OBS.
[ "$FULL" = "1" ] && { SCHED=1; OBS=1; }

echo "==> core data plane (Postgres · TimescaleDB · Redis · NATS · Mailpit)"
kubectl apply -f "$HERE/data-plane.yaml"
echo "   waiting for rollouts..."
for d in postgres timescaledb redis nats mailpit; do
  kubectl -n data rollout status "deploy/$d" --timeout=180s
done
echo "   core ready."

ns() { kubectl get ns "$1" >/dev/null 2>&1 || kubectl create ns "$1"; }

if [ "$SCHED" = "1" ]; then
  echo "==> scheduling — Kueue first"
  kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/latest/download/manifests.yaml
  # Kueue registers a Fail-policy mutating webhook on ALL Deployments/Jobs cluster-wide. Volcano (and
  # every later app deploy) is blocked until that webhook has endpoints, so wait for Kueue's
  # controller BEFORE applying anything else — otherwise Volcano's own Deployments fail to create.
  echo "   waiting for the Kueue controller + webhook endpoints..."
  kubectl -n kueue-system rollout status deploy/kueue-controller-manager --timeout=300s
  for _ in $(seq 1 30); do
    [ -n "$(kubectl -n kueue-system get endpoints kueue-webhook-service -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)" ] && break
    sleep 2
  done
  echo "==> scheduling — Volcano (now that the Kueue webhook is live)"
  kubectl apply -f https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml
  kubectl -n volcano-system rollout status deploy/volcano-scheduler  --timeout=240s || true
  kubectl -n volcano-system rollout status deploy/volcano-admission  --timeout=240s || true
  kubectl -n volcano-system rollout status deploy/volcano-controllers --timeout=240s || true
  echo "==> mock-GPU resource + Kueue config"
  bash "$HERE/../scheduling/mock-gpu-resource.sh"
  # server-side apply (the ClusterQueue webhook can briefly 500 right after the controller reports
  # Ready, and SSA avoids client-side last-applied annotation clashes across Kueue API versions).
  for i in 1 2 3 4 5; do
    kubectl apply --server-side --force-conflicts -f "$HERE/../scheduling/kueue-config.yaml" && break
    echo "   kueue-config apply failed (attempt $i) — webhook warming up, retrying in 5s"; sleep 5
  done
  echo "   scheduling ready (queues: inference · training-small · training-large)."
fi

if [ "$OBS" = "1" ]; then
  echo "==> observability (Prometheus + Loki + Tempo + Grafana)"
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
  helm repo add grafana https://grafana.github.io/helm-charts >/dev/null 2>&1 || true
  helm repo update >/dev/null
  ns observability
  helm upgrade --install kube-prom prometheus-community/kube-prometheus-stack -n observability --wait
  helm upgrade --install loki grafana/loki-stack -n observability --wait || true
  helm upgrade --install tempo grafana/tempo -n observability --wait || true
fi

if [ "$GPU" = "1" ]; then
  echo "==> NVIDIA GPU Operator (GPU=1)"
  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia >/dev/null 2>&1 || true
  helm repo update >/dev/null
  helm upgrade --install gpu-operator nvidia/gpu-operator -n gpu-operator --create-namespace --wait
fi
echo "==> data plane ready (SCHED=$SCHED OBS=$OBS GPU=$GPU)."
