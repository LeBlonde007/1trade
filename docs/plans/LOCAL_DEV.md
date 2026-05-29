# Local Development

Goal: a developer clones the repo, runs **one command**, and within ~15 minutes has the whole
platform running locally with mock data. Every agent's service must support this same flow.

---

## 1. One-command bring-up

```
make up         # creates the k3d cluster + deploys data plane + all services (tilt up)
make seed       # seeds tenants, demo credits, mock catalog
make web        # starts Nuxt dev server (apps/web/ or Exascale Frontend/)
make cli        # builds + symlinks `exascale` into ~/.local/bin
```

After `make up && make seed`:

- `http://localhost:3000` — Platform Console (Nuxt).
- `http://localhost:8080` — API ingress (Traefik in k3s; Kong arrives as prod hardening, M3).
- `http://localhost:9000` — Grafana (preconfigured dashboards from `deploy/grafana/`).
- `exascale login --dev` — logs in as a seeded tenant.
- `exascale credits balance` — shows seeded prepaid balance.
- `exascale infer chat -m llama-3.1-8b` — hits the mock inference backend (or real vLLM if GPU available).

If any of these don't work, the developer file an issue against `infra-sre`.

---

## 2. What runs where, locally

`make up` stands up a local **k3d** cluster (k3s-in-Docker) and deploys the whole stack into it —
the **same Kubernetes as production** (k3s), so there's no local/prod divergence. See
`docs/plans/DECISIONS.md` ADR-0001 for why k3s/k3d over kind or Dokploy.

```
k3d cluster `exascale` (created by `make up`) runs:
  data plane (Helm):   postgres :5432 · timescaledb :5433 · redis :6379 · nats :4222
  observability:       grafana :9000 · prometheus :9090 · loki :3100
  ingress:             Traefik (k3s built-in) on :8080 — routes /v1/* to the services
  services:            platform-core · credit-ledger · inference-gateway · compute-control ·
                       supply-service · index-service · matching-engine (mock) · surveillance
  frontend:            apps/web (Nuxt) on :3000

Inner loop: `make up` runs `tilt up` — Tilt watches source and live-updates the pods on save,
so iteration is fast AND what runs locally is exactly what runs in prod (no host-process drift).
```

Secrets locally are plain Kubernetes Secrets (no Vault needed in dev). Kong/Vault/Terraform are
prod hardening that arrive in later milestones — local stays light.

---

## 3. The cluster (k3d locally, k3s in prod)

Local and prod run the **same** Kubernetes. `make up` does:

```
k3d cluster create exascale --config deploy/k8s/local/k3d.yaml
helm install:  Kueue + Volcano + NVIDIA GPU Operator (mock-GPU mode by default)
               + data plane (CloudNativePG, Redis, NATS) + observability stack
tilt up:       builds + deploys every service into the cluster, live-reload on save
```

**GPU on Docker Desktop + WSL2:**
- Default = **mock-GPU mode** — registers virtual GPUs (no driver), so the control plane behaves
  identically without hardware. Anyone can run the full stack.
- Real GPU = `make up GPU=1` — uses the **NVIDIA Container Toolkit** (install it in your WSL2
  distro; Docker Desktop exposes the host GPU) to schedule real vLLM pods. Llama-3.1-8B fits on
  most dev GPUs; larger models route to the mock/stub unless you have the VRAM.

**Why not `kind` or Dokploy:** `kind` is test-only (not a prod runtime); Dokploy/Docker-Swarm
can't do GPU gang-scheduling, MIG, or Kueue/Volcano. k3s is single-binary and production-grade,
so local k3d == prod k3s with no second deployment model to maintain.

---

## 4. Service env conventions

Every service reads only env vars (no config files in source). The standard set:

```
EXASCALE_ENV=dev|staging|prod
EXASCALE_PAPER=true|false        # is_paper at the runtime level; dev defaults to true
DATABASE_URL=postgres://...
REDIS_URL=redis://...
NATS_URL=nats://...
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-root             # only in dev
LOG_LEVEL=debug|info|warn|error
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

Service-specific envs are documented in each service's `README.md`. A `.env.example` lives at
the repo root and per service; `make up` loads them.

---

## 5. Seeded data (`make seed`)

`make seed` creates a deterministic dev fixture:

- 3 tenants: `acme-ai` (AI-startup), `f500-co` (enterprise w/ SAML), `internal-mm` (internal market maker — kept for Phase 2).
- Each tenant gets $10,000 paper AI credits + sample sub-credit + GPU-credit balances.
- Mock model catalog populated (Llama-70B/8B, Whisper, FLUX) — `inference-runtime` runs a stub
  unless `GPU=1`.
- Mock partner-DC capacity registered (`tier=H100-80GB`, `count=8`).
- Mock daily index prints for the last 30 days (synthetic Brownian motion w/ mean reversion).

Reset: `make seed-reset` drops and re-seeds.

---

## 6. Frontend dev (mock vs. wired)

`apps/web/` (and current `Exascale Frontend/`) supports two modes via a single env var:

```
EXASCALE_API_MODE=mock           # uses apps/web/server/api/* mocks (default)
EXASCALE_API_MODE=local          # proxies to localhost:8080 (real gateway)
EXASCALE_API_MODE=staging        # proxies to https://staging.exascale.local
```

Switching `mock → local` must require **zero UI changes**. This is enforced in
`trading-frontend`'s definition of done.

---

## 7. CLI dev

`apps/cli/` builds with `make cli` to `bin/exascale` and symlinks. Local-dev defaults:

```
exascale config set api-url http://localhost:8080
exascale config set tenant acme-ai
exascale login --dev          # bypasses OAuth, uses a dev token
```

`exascale --help` is the contract; any new command needs a corresponding OpenAPI route in
`docs/contracts/openapi/`.

---

## 8. Per-agent local-dev checklist

Each agent ships their service ready for `make up` to bring it online. Definition of done for
"local-dev ready" is:

- [ ] Service has a `Dockerfile` (or is a Nuxt/Python app with documented run command).
- [ ] Service has a Helm chart / Kustomize entry under `deploy/k8s/<service>/` and is wired into
      the local `Tiltfile` so `make up` deploys it into the k3d cluster.
- [ ] Service reads only the standard env vars (§4) plus its own documented service-specific envs.
- [ ] `make seed` creates any fixtures the service needs to start in a usable state.
- [ ] Service-level integration tests run with `make test-<name>` and pass against the
      compose data plane.
- [ ] Service emits Prometheus metrics, OTLP traces, and structured logs.
- [ ] Service-level README documents: ports, envs, endpoints, how to call it from `curl`.

---

## 9. Cross-cutting tests

```
make test         # unit tests across all Go services + Nuxt unit
make test-integ   # spins up compose, runs integration tests, tears down
make test-e2e     # spins up everything + seeds + runs Playwright across signup → first infer
```

`make test-e2e` is the **sub-5-minute time-to-first-action** contract translated to a test:
signup → CLI login → buy credits → first inference call → wallet shows debit. All in <5 min
on CI hardware.

---

## 10. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `make up` hangs on `compute-control` | `k3d` cluster slow to start | wait or `k3d cluster delete exascale && make up` |
| Inference returns 402 | Tenant has no sub-credit | `exascale credits convert --from ai_index --to text 100` |
| Web shows "API unreachable" | `EXASCALE_API_MODE=local` but a service is down | `docker ps` to check, `make logs-<svc>` |
| Postgres fails to start on port 5432 | host Postgres already running | `pg_ctl stop` or set `EXASCALE_PG_PORT=5433` |

`infra-sre` owns this section and keeps it current.
