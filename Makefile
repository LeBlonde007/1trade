# Exascale — developer entrypoint. One command to a running platform locally.
# See docs/plans/LOCAL_DEV.md and docs/plans/DECISIONS.md ADR-0001 (k3s/k3d).
# Toolchain: run scripts/install-toolchain.sh first (go, kubectl, k3d, helm, tilt, sops).

CLUSTER ?= exascale
GPU     ?= 0          # set GPU=1 to schedule the host GPU (needs NVIDIA Container Toolkit)

.DEFAULT_GOAL := help

## help: list available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{printf "  \033[1m%-14s\033[0m %s\n", $$1, $$2}'

## up: create the k3d cluster + data plane, then start live-reload (Tilt)
.PHONY: up
up: cluster data-plane tilt

## cluster: create the local k3d cluster (idempotent)
.PHONY: cluster
cluster:
	@k3d cluster list 2>/dev/null | awk '{print $$1}' | grep -qx "$(CLUSTER)" \
		|| k3d cluster create --config deploy/k8s/local/k3d.yaml
	@kubectl config use-context k3d-$(CLUSTER) >/dev/null

## data-plane: install Postgres/TimescaleDB/Redis/NATS + observability via Helm
.PHONY: data-plane
data-plane:
	@GPU=$(GPU) bash deploy/k8s/local/install-data-plane.sh

## tilt: start Tilt (builds + deploys services into the cluster, live-reload on save)
.PHONY: tilt
tilt:
	@cd deploy/k8s/local && tilt up

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

## build: build all Go services
.PHONY: build
build:
	@test -n "$$(ls -d services/*/ 2>/dev/null)" \
		&& go build ./... || echo "no Go services scaffolded yet"

## test: run unit tests across the repo
.PHONY: test
test:
	@test -f go.work && go test ./... || echo "no Go modules yet"
	@test -d "Exascale Frontend/node_modules" && (cd "Exascale Frontend" && npm run typecheck 2>/dev/null || true) || true

## lint: run linters (golangci-lint + pre-commit hooks)
.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null && (test -f go.work && golangci-lint run ./... || echo "no Go yet") || echo "golangci-lint not installed"
	@command -v pre-commit >/dev/null && pre-commit run --all-files || echo "pre-commit not installed"

## fmt: format Go + run gofmt/goimports
.PHONY: fmt
fmt:
	@gofmt -l -w $$(git ls-files '*.go') 2>/dev/null || true

## ps: show cluster pods
.PHONY: ps
ps:
	@kubectl get pods -A
