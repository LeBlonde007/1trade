#!/usr/bin/env sh
# Exascale CLI installer. Anyone runs:
#
#   curl -fsSL https://sandbox.exascale.ai/cli/install.sh | sh
#
# It detects the OS/arch, downloads the matching prebuilt `exascale` binary (the sandbox api URL is
# baked in, so no config is needed), and installs it to ~/.local/bin. Override the source with
# EXASCALE_CLI_BASE and the install dir with EXASCALE_BIN_DIR.
set -e

BASE="${EXASCALE_CLI_BASE:-https://sandbox.exascale.ai/cli}"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "exascale: unsupported architecture '$arch'" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "exascale: unsupported OS '$os' — on Windows download $BASE/exascale-windows-amd64.exe" >&2; exit 1 ;;
esac

bin="exascale-$os-$arch"
dest="${EXASCALE_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$dest"
echo "exascale: downloading $BASE/$bin"
curl -fsSL "$BASE/$bin" -o "$dest/exascale"
chmod +x "$dest/exascale"

echo "exascale: installed to $dest/exascale"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "exascale: add it to your PATH →  export PATH=\"$dest:\$PATH\"" ;;
esac
echo "exascale: try it →  exascale catalog        (then: exascale login --email you@example.com)"
