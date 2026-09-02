#!/usr/bin/env bash
# 1Trade — advertise a MOCK GPU extended resource on the local k3d node so Kueue can quota and
# Volcano can gang-schedule against a countable resource with NO real GPU (the F12 "mock-GPU mode").
#
# It PATCHes `1trade.io/gpu: <count>` onto status.capacity of the node labeled 1trade.io/gpu=mock.
# k3d resets node status when the cluster restarts, so re-run this after `make up` (the FULL/SCHED
# installer calls it for you). Idempotent — a JSON-patch "add" replaces an existing value.
#
#   MOCK_GPU_COUNT=8 bash deploy/k8s/scheduling/mock-gpu-resource.sh
set -euo pipefail

COUNT="${MOCK_GPU_COUNT:-8}"
NODE=$(kubectl get nodes -l 1trade.io/gpu=mock -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -z "$NODE" ]; then
  echo "mock-gpu: no node labeled 1trade.io/gpu=mock — is the k3d cluster up?" >&2
  exit 1
fi

# extended resources live under status.capacity; "/" in the name is escaped as ~1 in a JSON-patch path
kubectl patch node "$NODE" --subresource=status --type=json \
  -p="[{\"op\":\"add\",\"path\":\"/status/capacity/1trade.io~1gpu\",\"value\":\"$COUNT\"}]" >/dev/null

echo "mock-gpu: advertised 1trade.io/gpu=$COUNT on $NODE"
kubectl get node "$NODE" -o jsonpath='{.status.capacity.1trade\.io/gpu}{"\n"}'
