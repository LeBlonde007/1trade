#!/usr/bin/env bash
# Run golangci-lint across every Go module in the monorepo (golangci-lint at the repo root reaches
# nothing — each services/* and apps/cli is its own module). One source of truth for `make lint` and
# CI. Requires golangci-lint v2.x on PATH (scripts/install-toolchain.sh installs the pinned version).
# Extra args pass through, e.g. `scripts/lint.sh --fix`.
set -uo pipefail
export GOWORK=off

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "lint: golangci-lint not installed — run scripts/install-toolchain.sh"; exit 1
fi

mapfile -t MODULES < <(find services apps -maxdepth 2 -name go.mod -printf '%h\n' 2>/dev/null | sort)
fail=0
for m in "${MODULES[@]}"; do
  echo "==> golangci-lint: $m"
  ( cd "$m" && golangci-lint run "$@" ./... ) || fail=1
done
[ "$fail" -eq 0 ] && echo "lint: all ${#MODULES[@]} modules clean" || echo "lint: FAILURES above"
exit $fail
