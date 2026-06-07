#!/usr/bin/env bash
# Build sandbox-bundle.tar.gz — the NO-SOURCE deploy bundle for a box you don't fully control (e.g. a
# client's droplet). It contains ONLY what `scripts/deploy-sandbox.sh` needs to deploy from PREBUILT
# images: k8s manifests, the data plane, the migration SQL, and the deploy scripts. It deliberately
# EXCLUDES all application source (no services/*/internal, no Exascale Frontend/app, no .go/.vue/.ts) —
# the compiled product reaches the box as images pulled from your private registry, never as source.
#
# Use:
#   scripts/make-sandbox-bundle.sh                 # → ./sandbox-bundle.tar.gz
#   scp sandbox-bundle.tar.gz root@HOST:/root/     # ship only this
#   # on the box:  tar xzf sandbox-bundle.tar.gz && \
#   #   IMAGE_REGISTRY=ghcr.io/<owner> REGISTRY_USER=<u> REGISTRY_TOKEN=<read:packages PAT> \
#   #   INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@x.com \
#   #   SANDBOX_HOST=sandbox.example.com SANDBOX_API_HOST=sandboxapi.example.com \
#   #   scripts/deploy-sandbox.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
OUT="${OUT:-sandbox-bundle.tar.gz}"

# Allowlist — only these paths go in the bundle. (deploy-sandbox.sh reads each of them.)
PATHS=(
  deploy/k8s                                          # all bases + the sandbox overlay + local/data-plane.yaml
  services/platform-core/migrations                   # platform-core schema (SQL only)
  services/credit-ledger/migrations                   # ledger schema (SQL only)
  docs/contracts/schemas/types.sql                    # shared credit-type enum / base types
  scripts/deploy-sandbox.sh                           # the deployer
  scripts/inference-provider.sh                       # optional: flip to a hosted inference provider
  deploy/k8s/overlays/sandbox/README.md               # operator notes
)
for p in "${PATHS[@]}"; do
  [ -e "$p" ] || { echo "ERROR: missing $p (run from a full checkout)"; exit 1; }
done

tar -czf "$OUT" "${PATHS[@]}"

# Safety gate: the bundle must contain ZERO application source. Fail loudly if any slipped in.
if tar tzf "$OUT" | grep -qiE '\.(go|vue|ts|tsx|jsx|py)$'; then
  echo "ERROR: source files found in $OUT — refusing to ship. Offending entries:"
  tar tzf "$OUT" | grep -iE '\.(go|vue|ts|tsx|jsx|py)$'
  rm -f "$OUT"; exit 1
fi

echo "✓ $OUT  ($(du -h "$OUT" | cut -f1), $(tar tzf "$OUT" | wc -l) entries, no source)"
echo "  ship it: scp $OUT root@<box>:/root/   then on the box: tar xzf $OUT && IMAGE_REGISTRY=… scripts/deploy-sandbox.sh"
