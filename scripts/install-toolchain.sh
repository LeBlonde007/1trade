#!/usr/bin/env bash
# 1Trade — developer toolchain installer for WSL2 / Ubuntu / Debian.
#
# Installs everything needed to build + run the platform locally per docs/plans/DECISIONS.md
# (ADR-0001: Kubernetes via k3s/k3d) and ENGINEERING_STANDARDS.md:
#   go · kubectl · k3d · helm · tilt · golangci-lint · gitleaks · sops · age · pre-commit
# Optional: NVIDIA Container Toolkit (--gpu) for real local GPU inference.
#
# Idempotent: skips anything already present (use --force to reinstall). Safe to re-run.
# Usage:
#   bash scripts/install-toolchain.sh            # core toolchain
#   bash scripts/install-toolchain.sh --gpu      # + NVIDIA Container Toolkit
#   bash scripts/install-toolchain.sh --force    # reinstall even if present
#
# Note: this is a DEV-MACHINE bootstrap. It uses official upstream installers (some are
# curl|bash) — acceptable for local setup, not for CI/build steps (see ENGINEERING_STANDARDS §5).
set -euo pipefail

# ---- config (override via env) -------------------------------------------------------------
GOLANGCI_VERSION="${GOLANGCI_VERSION:-v2.12.2}"   # v2.x — Go 1.25 compatible. .golangci.yml is v2 format.
GITLEAKS_VERSION="${GITLEAKS_VERSION:-8.18.4}"    # keep in sync with .pre-commit-config.yaml
SOPS_VERSION="${SOPS_VERSION:-3.9.0}"
WANT_GPU=false
FORCE=false
for arg in "$@"; do
  case "$arg" in
    --gpu) WANT_GPU=true ;;
    --force) FORCE=true ;;
    *) echo "unknown arg: $arg" >&2; exit 2 ;;
  esac
done

# ---- helpers -------------------------------------------------------------------------------
# arch_tag prints the Go/k8s-style architecture tag for this machine (amd64 or arm64).
arch_tag() { case "$(uname -m)" in x86_64) echo amd64 ;; aarch64|arm64) echo arm64 ;; *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;; esac; }

# have reports whether a command is already on PATH.
have() { command -v "$1" >/dev/null 2>&1; }

# need_install decides whether to (re)install a tool given --force and current presence.
need_install() { local bin="$1"; if [ "$FORCE" = true ]; then return 0; fi; if have "$bin"; then echo "  ✓ $bin already installed ($("$bin" --version 2>/dev/null | head -1 || true))"; return 1; fi; return 0; }

# step prints a section header.
step() { echo; echo "==> $*"; }

ARCH="$(arch_tag)"
SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"

# ---- prerequisites (apt) -------------------------------------------------------------------
step "Base packages (curl, git, ca-certificates, jq)"
$SUDO apt-get update -y -qq
$SUDO apt-get install -y -qq curl git ca-certificates jq tar gzip pipx >/dev/null
pipx ensurepath >/dev/null 2>&1 || true

# ---- Docker (verify only; Docker Desktop provides it on WSL2) -------------------------------
step "Docker"
if have docker; then echo "  ✓ docker present ($(docker --version))"; else
  echo "  ! docker NOT found. Install Docker Desktop for Windows and enable WSL integration,"
  echo "    or 'sudo apt-get install docker.io'. Re-run after Docker is available."
fi

# ---- Go ------------------------------------------------------------------------------------
# install_go fetches the current stable Go toolchain to /usr/local/go and wires PATH.
install_go() {
  local ver tarball
  ver="$(curl -sSL https://go.dev/VERSION?m=text | head -1)"   # e.g. go1.23.5
  tarball="${ver}.linux-${ARCH}.tar.gz"
  echo "  installing ${ver}"
  curl -sSLo "/tmp/${tarball}" "https://go.dev/dl/${tarball}"
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf "/tmp/${tarball}"
  rm -f "/tmp/${tarball}"
  if ! grep -q '/usr/local/go/bin' "$HOME/.profile" 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$HOME/.profile"
  fi
  export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
}
step "Go"
if need_install go; then install_go; echo "  ✓ $(/usr/local/go/bin/go version)"; fi
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"

# ---- kubectl -------------------------------------------------------------------------------
step "kubectl"
if need_install kubectl; then
  v="$(curl -L -s https://dl.k8s.io/release/stable.txt)"
  curl -sSLo /tmp/kubectl "https://dl.k8s.io/release/${v}/bin/linux/${ARCH}/kubectl"
  $SUDO install -m 0755 /tmp/kubectl /usr/local/bin/kubectl && rm -f /tmp/kubectl
  echo "  ✓ kubectl ${v}"
fi

# ---- k3d (k3s in Docker — our local cluster, ADR-0001) -------------------------------------
step "k3d"
if need_install k3d; then
  curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | $SUDO bash
  echo "  ✓ $(k3d version | head -1)"
fi

# ---- helm ----------------------------------------------------------------------------------
step "helm"
if need_install helm; then
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | $SUDO bash
  echo "  ✓ $(helm version --short)"
fi

# ---- Tilt (local live-reload into k3d) -----------------------------------------------------
step "tilt"
if need_install tilt; then
  curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
  echo "  ✓ tilt installed"
fi

# ---- golangci-lint (pinned; matches .pre-commit-config.yaml) -------------------------------
step "golangci-lint (${GOLANGCI_VERSION})"
if need_install golangci-lint; then
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$HOME/go/bin" "${GOLANGCI_VERSION}"
  echo "  ✓ golangci-lint ${GOLANGCI_VERSION}"
fi

# ---- gitleaks (pinned; matches .pre-commit-config.yaml) ------------------------------------
step "gitleaks (${GITLEAKS_VERSION})"
if need_install gitleaks; then
  curl -sSLo /tmp/gitleaks.tar.gz \
    "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_${ARCH/amd64/x64}.tar.gz" \
    || curl -sSLo /tmp/gitleaks.tar.gz \
    "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_${ARCH}.tar.gz"
  tar -C /tmp -xzf /tmp/gitleaks.tar.gz gitleaks
  $SUDO install -m 0755 /tmp/gitleaks /usr/local/bin/gitleaks && rm -f /tmp/gitleaks /tmp/gitleaks.tar.gz
  echo "  ✓ $(gitleaks version)"
fi

# ---- sops + age (secrets at rest for M1 — ADR-0001 progressive hardening) ------------------
step "sops (${SOPS_VERSION}) + age"
if need_install sops; then
  curl -sSLo /tmp/sops "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-v${SOPS_VERSION}.linux.${ARCH}"
  $SUDO install -m 0755 /tmp/sops /usr/local/bin/sops && rm -f /tmp/sops
  echo "  ✓ sops ${SOPS_VERSION}"
fi
if need_install age; then $SUDO apt-get install -y -qq age >/dev/null && echo "  ✓ age installed"; fi

# ---- pre-commit (runs the hooks in .pre-commit-config.yaml) --------------------------------
step "pre-commit"
if need_install pre-commit; then pipx install pre-commit >/dev/null && echo "  ✓ pre-commit installed"; fi

# ---- optional: NVIDIA Container Toolkit (real local GPU inference) -------------------------
# install_nvidia_toolkit wires the NVIDIA apt repo and installs the container toolkit so Docker
# (and k3d) can schedule the host GPU. Requires an NVIDIA GPU + WSL2 GPU passthrough.
install_nvidia_toolkit() {
  curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
    | $SUDO gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
  curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
    | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
    | $SUDO tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null
  $SUDO apt-get update -y -qq
  $SUDO apt-get install -y -qq nvidia-container-toolkit >/dev/null
  $SUDO nvidia-ctk runtime configure --runtime=docker || true
  echo "  ✓ NVIDIA Container Toolkit installed (restart Docker; verify: docker run --rm --gpus all ubuntu nvidia-smi)"
}
if [ "$WANT_GPU" = true ]; then step "NVIDIA Container Toolkit (--gpu)"; install_nvidia_toolkit; fi

# ---- summary -------------------------------------------------------------------------------
step "Summary"
printf '%-16s %s\n' go        "$(go version 2>/dev/null || echo MISSING)"
printf '%-16s %s\n' kubectl   "$(kubectl version --client -o yaml 2>/dev/null | grep -m1 gitVersion | awk '{print $2}' || echo MISSING)"
printf '%-16s %s\n' k3d       "$(k3d version 2>/dev/null | head -1 || echo MISSING)"
printf '%-16s %s\n' helm      "$(helm version --short 2>/dev/null || echo MISSING)"
printf '%-16s %s\n' tilt      "$(tilt version 2>/dev/null || echo MISSING)"
printf '%-16s %s\n' golangci  "$(golangci-lint --version 2>/dev/null | head -1 || echo MISSING)"
printf '%-16s %s\n' gitleaks  "$(gitleaks version 2>/dev/null || echo MISSING)"
printf '%-16s %s\n' sops      "$(sops --version 2>/dev/null | head -1 || echo MISSING)"
printf '%-16s %s\n' pre-commit "$(pre-commit --version 2>/dev/null || echo MISSING)"
echo
echo "Done. Open a new shell (or 'source ~/.profile') so PATH picks up go + \$HOME/go/bin."
echo "Next: 'pre-commit install' in the repo, then 'make up' once F01 scaffolding exists."
