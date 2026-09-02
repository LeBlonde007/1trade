#!/usr/bin/env bash
# 1Trade — one-command, OS-aware bootstrap.
#
# Detects your OS, installs EVERY dependency the right way for that OS, then (with --up)
# brings the whole platform up locally via `make up`. One command from a clean machine to a
# running stack.
#
#   Debian / Ubuntu / WSL2  → delegates to scripts/install-toolchain.sh (apt + pinned installers)
#   macOS                   → auto-installs Homebrew if missing, then brew (+ pinned installers)
#   Fedora / RHEL / Arch    → native package manager + pinned curl installers
#
# Usage:
#   bash scripts/setup.sh              # install all requirements for this OS
#   bash scripts/setup.sh --up         # install, then `make up` (run the whole platform)
#   bash scripts/setup.sh --gpu        # also install the NVIDIA Container Toolkit (Linux only)
#   bash scripts/setup.sh --force      # reinstall tools even if already present
#   bash scripts/setup.sh --up --gpu   # flags combine
#
# Idempotent: re-running skips what's already installed. This is a DEV-MACHINE bootstrap; it uses
# official upstream installers — fine for local setup, not for CI (see ENGINEERING_STANDARDS §5).
set -euo pipefail

# ---- pinned versions (keep in sync with scripts/install-toolchain.sh + .pre-commit-config.yaml) ----
GOLANGCI_VERSION="${GOLANGCI_VERSION:-v2.12.2}"
GITLEAKS_VERSION="${GITLEAKS_VERSION:-8.18.4}"
SOPS_VERSION="${SOPS_VERSION:-3.9.0}"

# ---- flags ----------------------------------------------------------------------------------
RUN_UP=false
WANT_GPU=false
FORCE=false
for arg in "$@"; do
  case "$arg" in
    --up)    RUN_UP=true ;;
    --gpu)   WANT_GPU=true ;;
    --force) FORCE=true ;;
    -h|--help) sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown arg: $arg  (see --help)" >&2; exit 2 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ---- helpers --------------------------------------------------------------------------------
# step prints a section header.
step() { echo; echo "==> $*"; }
# have reports whether a command is already on PATH.
have() { command -v "$1" >/dev/null 2>&1; }
# want reports whether a tool should be (re)installed given --force and current presence.
want() { if [ "$FORCE" = true ]; then return 0; fi; if have "$1"; then echo "  ✓ $1 present"; return 1; fi; return 0; }
# arch_tag prints the Go/k8s-style architecture tag (amd64 or arm64).
arch_tag() { case "$(uname -m)" in x86_64|amd64) echo amd64 ;; aarch64|arm64) echo arm64 ;; *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;; esac; }
ARCH="$(arch_tag)"
SUDO=""; if [ "$(id -u 2>/dev/null || echo 0)" -ne 0 ]; then SUDO="sudo"; fi

# ---- OS detection ---------------------------------------------------------------------------
# detect_platform sets PLATFORM to one of: macos | debian | fedora | arch | linux-other | unknown.
detect_platform() {
  case "$(uname -s)" in
    Darwin) PLATFORM=macos ;;
    Linux)
      local id="" like=""
      if [ -f /etc/os-release ]; then . /etc/os-release; id="${ID:-}"; like="${ID_LIKE:-}"; fi
      case "$id $like" in
        *debian*|*ubuntu*) PLATFORM=debian ;;
        *fedora*|*rhel*|*centos*|*rocky*|*alma*) PLATFORM=fedora ;;
        *arch*|*manjaro*) PLATFORM=arch ;;
        *) PLATFORM=linux-other ;;
      esac ;;
    *) PLATFORM=unknown ;;
  esac
  IS_WSL=false; if grep -qiE 'microsoft|wsl' /proc/version 2>/dev/null; then IS_WSL=true; fi
  return 0   # never let the WSL probe's exit status leak out (grep returns 2 on macOS: no /proc/version)
}
detect_platform

# ---- shared pinned installers (curl-based; used by macOS + non-Debian Linux) ---------------
# These fetch the exact versions CI expects, independent of any package manager.
OSL="$(uname -s | tr '[:upper:]' '[:lower:]')"   # linux | darwin

# ct_go installs the current stable Go toolchain into /usr/local/go and wires PATH.
ct_go() {
  want go || return 0
  local ver tb; ver="$(curl -sSL https://go.dev/VERSION?m=text | head -1)"
  tb="${ver}.${OSL}-${ARCH}.tar.gz"
  echo "  installing ${ver}"
  curl -sSLo "/tmp/${tb}" "https://go.dev/dl/${tb}"
  $SUDO rm -rf /usr/local/go && $SUDO tar -C /usr/local -xzf "/tmp/${tb}" && rm -f "/tmp/${tb}"
  grep -q '/usr/local/go/bin' "$HOME/.profile" 2>/dev/null || \
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$HOME/.profile"
  export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
}
# ct_kubectl installs the stable kubectl CLI.
ct_kubectl() { want kubectl || return 0; local v; v="$(curl -L -s https://dl.k8s.io/release/stable.txt)"; curl -sSLo /tmp/kubectl "https://dl.k8s.io/release/${v}/bin/${OSL}/${ARCH}/kubectl"; $SUDO install -m 0755 /tmp/kubectl /usr/local/bin/kubectl && rm -f /tmp/kubectl; }
# ct_k3d installs k3d (k3s-in-Docker, the local cluster).
ct_k3d() { want k3d || return 0; curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | $SUDO bash; }
# ct_helm installs Helm 3.
ct_helm() { want helm || return 0; curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | $SUDO bash; }
# ct_tilt installs Tilt (local live-reload).
ct_tilt() { want tilt || return 0; curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash; }
# ct_golangci installs the pinned golangci-lint into $HOME/go/bin (matches CI hard gate).
ct_golangci() { want golangci-lint || return 0; curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$HOME/go/bin" "${GOLANGCI_VERSION}"; }
# ct_gitleaks installs the pinned gitleaks binary.
ct_gitleaks() {
  want gitleaks || return 0
  curl -sSLo /tmp/gitleaks.tar.gz "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_${OSL}_${ARCH/amd64/x64}.tar.gz" \
    || curl -sSLo /tmp/gitleaks.tar.gz "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_${OSL}_${ARCH}.tar.gz"
  tar -C /tmp -xzf /tmp/gitleaks.tar.gz gitleaks
  $SUDO install -m 0755 /tmp/gitleaks /usr/local/bin/gitleaks && rm -f /tmp/gitleaks /tmp/gitleaks.tar.gz
}
# ct_sops installs the pinned sops binary.
ct_sops() { want sops || return 0; curl -sSLo /tmp/sops "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-v${SOPS_VERSION}.${OSL}.${ARCH}"; $SUDO install -m 0755 /tmp/sops /usr/local/bin/sops && rm -f /tmp/sops; }

# ---- Docker check (shared) ------------------------------------------------------------------
check_docker() {
  step "Docker"
  if have docker; then echo "  ✓ docker present ($(docker --version))"; return; fi
  case "$PLATFORM" in
    macos)  echo "  ! docker not found — install Docker Desktop:  brew install --cask docker  (then launch it)" ;;
    *)      echo "  ! docker not found — install Docker Desktop (WSL integration) or 'sudo apt-get install docker.io', then re-run" ;;
  esac
}

# =============================================================================================
#  Per-OS installation
# =============================================================================================

# --- Debian / Ubuntu / WSL2: reuse the maintained toolchain installer ------------------------
install_debian() {
  step "Debian/Ubuntu/WSL detected → scripts/install-toolchain.sh"
  local args=(); [ "$WANT_GPU" = true ] && args+=(--gpu); [ "$FORCE" = true ] && args+=(--force)
  bash "$REPO_ROOT/scripts/install-toolchain.sh" "${args[@]}"
}

# --- macOS: Homebrew for most, pinned curl installers for the CI-gated tools -----------------
install_macos() {
  step "macOS detected → Homebrew"
  if ! have brew; then
    step "Homebrew not found → installing it (this may ask for your Mac password once)"
    # NONINTERACTIVE skips Homebrew's RETURN prompt; sudo may still prompt for your password.
    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    # The installer does NOT put brew on PATH (esp. Apple Silicon /opt/homebrew) — do it for this
    # run, and persist it to the login shell's profile so future shells + `make up` find it.
    for b in /opt/homebrew/bin/brew /usr/local/bin/brew; do
      if [ -x "$b" ]; then eval "$("$b" shellenv)"; break; fi
    done
    if ! have brew; then
      echo "  ! Homebrew install did not complete — see the output above, then re-run 'make bootstrap'." >&2
      exit 1
    fi
    local prof="$HOME/.zprofile"
    case "$(basename "${SHELL:-/bin/zsh}")" in bash) prof="$HOME/.bash_profile" ;; esac
    if ! grep -q 'brew shellenv' "$prof" 2>/dev/null; then
      echo "eval \"\$($(command -v brew) shellenv)\"" >> "$prof"
      echo "  ✓ added brew to $prof (open a new terminal later and it'll be on PATH)"
    fi
    echo "  ✓ $(brew --version | head -1)"
  fi
  check_docker
  # brew_install installs a formula only if it is not already present.
  brew_install() { brew list --versions "$1" >/dev/null 2>&1 || brew install "$1"; }
  step "Core tools (brew)"
  for f in go node kubectl k3d helm sops age jq pre-commit; do echo "  • $f"; brew_install "$f"; done
  echo "  • tilt"; brew list --versions tilt >/dev/null 2>&1 || brew install tilt-dev/tap/tilt
  export PATH="$PATH:$HOME/go/bin"
  step "golangci-lint (${GOLANGCI_VERSION}, pinned) + gitleaks (${GITLEAKS_VERSION}, pinned)"
  ct_golangci
  ct_gitleaks
  if [ "$WANT_GPU" = true ]; then echo "  (note: --gpu is a no-op on macOS; local GPU inference is Linux/WSL2 only)"; fi
}

# --- Fedora/RHEL/Arch/other Linux: native base packages + pinned curl installers -------------
install_generic_linux() {
  step "Linux (${PLATFORM}) detected → native base packages + pinned installers"
  case "$PLATFORM" in
    fedora) $SUDO dnf install -y curl git ca-certificates jq tar gzip age python3-pip >/dev/null || true ;;
    arch)   $SUDO pacman -Sy --noconfirm curl git ca-certificates jq tar gzip age python-pipx >/dev/null || true ;;
    *)      echo "  ! unrecognized distro — ensure curl, git, jq, tar, age are installed" ;;
  esac
  check_docker
  step "Go";           ct_go
  export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
  step "kubectl";      ct_kubectl
  step "k3d";          ct_k3d
  step "helm";         ct_helm
  step "tilt";         ct_tilt
  step "golangci-lint (${GOLANGCI_VERSION})"; ct_golangci
  step "gitleaks (${GITLEAKS_VERSION})";      ct_gitleaks
  step "sops (${SOPS_VERSION})";              ct_sops
  step "pre-commit"
  if want pre-commit; then have pipx && pipx install pre-commit >/dev/null || pip3 install --user pre-commit >/dev/null; fi
  if [ "$WANT_GPU" = true ]; then echo "  (note: NVIDIA toolkit auto-install is wired for Debian/Ubuntu only — install manually on ${PLATFORM})"; fi
}

case "$PLATFORM" in
  debian)      install_debian ;;
  macos)       install_macos ;;
  fedora|arch|linux-other) install_generic_linux ;;
  *) echo "Unsupported OS ($(uname -s)). On Windows, run this inside WSL2 (Ubuntu)." >&2; exit 1 ;;
esac

# ---- summary --------------------------------------------------------------------------------
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
step "Installed"
printf '  %-14s %s\n' docker    "$(docker --version 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' go        "$(go version 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' node      "$(node --version 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' kubectl   "$(kubectl version --client -o yaml 2>/dev/null | grep -m1 gitVersion | awk '{print $2}' || echo MISSING)"
printf '  %-14s %s\n' k3d       "$(k3d version 2>/dev/null | head -1 || echo MISSING)"
printf '  %-14s %s\n' helm      "$(helm version --short 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' tilt      "$(tilt version 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' golangci  "$(golangci-lint --version 2>/dev/null | head -1 || echo MISSING)"
printf '  %-14s %s\n' gitleaks  "$(gitleaks version 2>/dev/null || echo MISSING)"
printf '  %-14s %s\n' sops      "$(sops --version 2>/dev/null | head -1 || echo MISSING)"
printf '  %-14s %s\n' pre-commit "$(pre-commit --version 2>/dev/null || echo MISSING)"

# ---- bring the platform up (--up) -----------------------------------------------------------
if [ "$RUN_UP" = true ]; then
  step "make up — creating the k3d cluster + data plane + services (live-reload)"
  if ! have docker || ! docker info >/dev/null 2>&1; then
    echo "  ! Docker isn't running. Start Docker Desktop (or the docker daemon), then: make up" >&2
    exit 1
  fi
  ( cd "$REPO_ROOT" && make up )
else
  echo
  echo "Done. Open a new shell (or 'source ~/.profile') so PATH picks up go + \$HOME/go/bin."
  echo "Next:  make up      # bring the whole platform up   (or re-run: bash scripts/setup.sh --up)"
fi
