#!/usr/bin/env bash
# Run a go subcommand across every Go module in the monorepo (each services/* and apps/* is its own
# module, so a single `go ... ./...` from the root can't reach them). One source of truth for
# `make build`, `make test`, and CI. Auto-discovers modules so new services are picked up.
#
#   scripts/go-all.sh build              # go build ./...   in each module
#   scripts/go-all.sh test -race         # go test -race ./... in each module
#   scripts/go-all.sh vet                # go vet ./...     in each module
#
# GOWORK=off pins module mode: each module builds against its OWN go.mod/go.sum, deterministically,
# regardless of any stray local go.work (which would force workspace mode + a go.work.sum).
set -uo pipefail
export GOWORK=off
CMD="${1:-test}"; shift || true

mapfile -t MODULES < <(find services apps -maxdepth 2 -name go.mod -printf '%h\n' 2>/dev/null | sort)
if [ "${#MODULES[@]}" -eq 0 ]; then
  echo "go-all: no Go modules found under services/ or apps/"; exit 0
fi

fail=0
for m in "${MODULES[@]}"; do
  echo "==> $m: go $CMD $* ./..."
  ( cd "$m" && go "$CMD" "$@" ./... ) || fail=1
done
[ "$fail" -eq 0 ] && echo "go-all: all ${#MODULES[@]} modules OK" || echo "go-all: FAILURES above"
exit $fail
