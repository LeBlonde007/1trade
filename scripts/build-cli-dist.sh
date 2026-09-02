#!/usr/bin/env bash
# Cross-compile the 1trade CLI for the common platforms with the sandbox api URL baked in, and stage
# them + the installer into a dir the web image serves at /cli/. After this runs, the web build bundles
# 1Trade Frontend/public/cli/* → served at https://<host>/cli/ → `curl …/cli/install.sh | sh` works.
#
#   scripts/build-cli-dist.sh                                  # → 1Trade Frontend/public/cli, sandbox URL
#   CLI_API_URL=https://api.1trade.ai OUT_DIR=/tmp/cli scripts/build-cli-dist.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_URL="${CLI_API_URL:-https://sandboxapi.1trade.ai}"
OUT="${OUT_DIR:-$ROOT/1Trade Frontend/public/cli}"
VERSION="${CLI_VERSION:-$(git -C "$ROOT" describe --tags --always 2>/dev/null || echo dev)}"

mkdir -p "$OUT"
LDFLAGS="-s -w -X github.com/trade1/cli/internal/config.DefaultAPIURL=$API_URL -X main.Version=$VERSION"
echo "==> building 1trade CLI (api=$API_URL version=$VERSION) → $OUT"
cd "$ROOT/apps/cli"
for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  os="${t%/*}"; arch="${t#*/}"; ext=""; [ "$os" = windows ] && ext=".exe"
  echo "   $os/$arch"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 GOWORK=off \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/1trade-$os-$arch$ext" ./cmd/1trade
done
cp "$ROOT/scripts/cli-install.sh" "$OUT/install.sh"
echo "==> done: $(ls "$OUT" | tr '\n' ' ')"
