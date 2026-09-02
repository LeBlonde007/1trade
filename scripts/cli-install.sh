#!/usr/bin/env sh
# 1Trade CLI installer. Anyone runs:
#
#   curl -fsSL https://sandbox.1trade.ai/cli/install.sh | sh
#
# It detects the OS/arch, downloads the matching prebuilt `1trade` binary (the sandbox api URL is
# baked in, so no config is needed), and installs it to ~/.local/bin. Override the source with
# TRADE1_CLI_BASE and the install dir with TRADE1_BIN_DIR.
set -e

BASE="${TRADE1_CLI_BASE:-https://sandbox.1trade.ai/cli}"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "1trade: unsupported architecture '$arch'" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "1trade: unsupported OS '$os' — on Windows download $BASE/1trade-windows-amd64.exe" >&2; exit 1 ;;
esac

bin="1trade-$os-$arch"
dest="${TRADE1_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$dest"
echo "1trade: downloading $BASE/$bin"
curl -fsSL "$BASE/$bin" -o "$dest/1trade"
chmod +x "$dest/1trade"

echo "1trade: installed to $dest/1trade"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "1trade: add it to your PATH →  export PATH=\"$dest:\$PATH\"" ;;
esac
echo "1trade: try it →  1trade catalog        (then: 1trade login --email you@example.com)"
