# Exascale — developer entrypoint. One command to a running platform locally.
# See docs/plans/LOCAL_DEV.md and docs/plans/DECISIONS.md ADR-0001 (k3s/k3d).
# Toolchain: run scripts/install-toolchain.sh first (go, kubectl, k3d, helm, tilt, sops).

CLUSTER ?= exascale
GPU     ?= 0          # set GPU=1 to schedule the host GPU (needs NVIDIA Container Toolkit)
FULL    ?= 0          # set FULL=1 to also install scheduling + observability
SCHED   ?= 0          # set SCHED=1 to also install Kueue + Volcano + mock-GPU queues
OBS     ?= 0          # set OBS=1 to also install Prometheus/Loki/Tempo/Grafana

.DEFAULT_GOAL := help

## help: list available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{printf "  \033[1m%-14s\033[0m %s\n", $$1, $$2}'

## up: create the k3d cluster + data plane, then start live-reload (Tilt)
.PHONY: up
up: cluster data-plane tilt

## cluster: create the local k3d cluster, or start it if it already exists but is stopped (idempotent)
.PHONY: cluster
cluster:
	@if k3d cluster list 2>/dev/null | awk '{print $$1}' | grep -qx "$(CLUSTER)"; then \
		echo "==> starting existing k3d cluster '$(CLUSTER)'"; k3d cluster start "$(CLUSTER)"; \
	else \
		echo "==> creating k3d cluster '$(CLUSTER)'"; k3d cluster create --config deploy/k8s/local/k3d.yaml; \
	fi
	@kubectl config use-context k3d-$(CLUSTER) >/dev/null

## data-plane: install the data plane (core; FULL=1/SCHED=1/OBS=1/GPU=1 add tiers)
.PHONY: data-plane
data-plane:
	@GPU=$(GPU) FULL=$(FULL) SCHED=$(SCHED) OBS=$(OBS) bash deploy/k8s/local/install-data-plane.sh

## sched: install Kueue + Volcano + mock-GPU queues (the F12 scheduling stack) into the cluster
.PHONY: sched
sched:
	@SCHED=1 bash deploy/k8s/local/install-data-plane.sh

## tilt: start Tilt (builds + deploys services into the cluster, live-reload on save)
.PHONY: tilt
tilt:
	@tilt up

## down: delete the local cluster
.PHONY: down
down:
	k3d cluster delete $(CLUSTER)

## seed: seed dev tenants, demo credits, mock catalog
.PHONY: seed
seed:
	@bash deploy/k8s/local/seed.sh

## web: run the Nuxt frontend dev server
.PHONY: web
web:
	@cd "Exascale Frontend" && npm run dev

## cli: build the exascale CLI (apps/cli — platform-core / F04; not scaffolded yet)
.PHONY: cli
cli:
	@test -d apps/cli && (cd apps/cli && go build -o ../../bin/exascale ./cmd/exascale) \
		|| echo "apps/cli not scaffolded yet (F04)"

## build: build every Go module (services + CLI)
.PHONY: build
build:
	@bash scripts/go-all.sh build

## test: race-test every Go module (+ Nuxt typecheck if installed)
.PHONY: test
test:
	@bash scripts/go-all.sh test -race
	@test -d "Exascale Frontend/node_modules" && (cd "Exascale Frontend" && npm run typecheck 2>/dev/null || true) || true

## test-e2e: timed sub-5-min time-to-first-action loop (signup→top-up→infer→GPU debit). Needs `make up`.
.PHONY: test-e2e
test-e2e:
	@bash scripts/e2e.sh

## lint: golangci-lint across every Go module (scripts/lint.sh); add --fix to auto-fix
.PHONY: lint
lint:
	@bash scripts/lint.sh $(if $(FIX),--fix,)

## fmt: format Go + run gofmt/goimports
.PHONY: fmt
fmt:
	@gofmt -l -w $$(git ls-files '*.go') 2>/dev/null || true

## ps: show cluster pods
.PHONY: ps
ps:
	@kubectl get pods -A
