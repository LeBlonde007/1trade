# Deployment

Dockerfiles, K8s manifests, CI/CD, and environments. Owned by `infra-sre`; every other agent
ships against the templates below.

---

## 0. Deployment platform (decision)

**Kubernetes is the single substrate — `k3s` in prod, `k3d` locally — kept lightweight and
hardened progressively.** No Dokploy/Swarm (it can't do GPU gang-scheduling, MIG, or
Kueue/Volcano — the compute layer's hard requirement). Full rationale + the rejected options:
`docs/plans/DECISIONS.md` **ADR-0001**.

The enterprise pieces arrive only when a milestone needs them (so M1 stays light):

| Concern | Start (M1) | Hardens to | When |
|---|---|---|---|
| Orchestration | k3s / k3d | (same — k3s scales) | — |
| Ingress | Traefik (k3s built-in) | Kong behind Cloudflare | M3 |
| Secrets | K8s Secrets + SOPS | Vault + Agent injector | M4 (SOC 2) |
| IaC | Helm + Kustomize | + Terraform | M3 |
| GPU scheduling | NVIDIA GPU Operator + Kueue + Volcano (on the GPU node pool) | (same) | M2→M5 |

Everything below (Dockerfiles, the K8s manifest shape, CI/CD) applies unchanged on k3s — k3s is
conformant Kubernetes, so the Kustomize bases/overlays and Helm charts are portable to managed
K8s later if the cloud footprint grows.

---

## 1. Environments

| Env | Hostname | Purpose | `is_paper` default | Real money? |
|---|---|---|---|---|
| `dev` | `*.dev.exascale.local` | Developer laptops + ephemeral CI | true | no |
| `staging` | `*.staging.exascale.io` | Pre-prod, mirrors prod | true | no |
| `prod-paper` | `*.paper.exascale.io` | Paper/sandbox for customers | true | no |
| `prod-real` | `*.exascale.io` | Real money. Phase 2 once licensed. | false | yes (Phase 2) |

Even though trading is paused, **`prod-paper` and `prod-real` are two separate Kubernetes
clusters with separate Postgres and separate secrets**, from day one. The pivot doesn't relax
that isolation — it just means `prod-real` runs only platform features (inference, compute,
prepaid credits) until the exchange is switched on.

---

## 2. Per-service Dockerfile (Go service template)

`deploy/docker/Dockerfile.go-service` — symlinked from each `services/*/Dockerfile`.

```dockerfile
# syntax=docker/dockerfile:1.6
ARG GO_VERSION=1.22

# --- build stage ---
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
ARG SERVICE
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" \
        -o /out/app ./cmd/${SERVICE}

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
ARG SERVICE
COPY --from=build /out/app /usr/local/bin/app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER nonroot:nonroot
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/app"]
```

Build: `docker build --build-arg SERVICE=credit-ledger -t exascale/credit-ledger:$(git rev-parse --short HEAD) .`

Rules:
- Distroless runtime; non-root user.
- Static binary; no CGO unless the service explicitly needs it (only `compute-control` may, for
  some K8s client libs — and we prefer pure-Go bindings).
- Each service exposes one HTTP port (`:8000`); Kong routes external traffic.
- Healthcheck path `/healthz` (liveness) and `/readyz` (readiness) — required.

---

## 3. Per-app Dockerfile (Nuxt frontend)

`deploy/docker/Dockerfile.nuxt`:

```dockerfile
# syntax=docker/dockerfile:1.6
FROM node:20-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY . .
ENV NODE_ENV=production
RUN npm run build

FROM node:20-alpine AS runtime
WORKDIR /app
COPY --from=build /src/.output ./.output
COPY --from=build /src/package.json ./
ENV NODE_ENV=production
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
```

Three serving modes for `apps/web/`:
- `dev`: `nuxt dev` (host process, hot reload).
- `prerendered marketing` + SPA app: `nuxt generate` then host on Cloudflare Pages.
- `node SSR`: the above Dockerfile (option for trading-dashboard pages that need server runtime).

`trading-frontend` picks per-route per `nuxt.config.ts` rules (already set up — see existing
config).

---

## 4. Per-runtime Dockerfile (vLLM inference worker)

`deploy/docker/Dockerfile.vllm`:

```dockerfile
# syntax=docker/dockerfile:1.6
FROM nvcr.io/nvidia/pytorch:24.10-py3 AS base
ARG VLLM_VERSION=0.6.3.post1
WORKDIR /app
RUN pip install --no-cache-dir vllm==${VLLM_VERSION} \
    && pip install --no-cache-dir prometheus-client fastapi uvicorn[standard]
COPY services/inference-runtime/python /app
EXPOSE 8000 8001
ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "-m", "exascale_runtime"]
```

This image is large (~12 GB); it is built once per vLLM release, pushed to the org registry, and
pulled by every inference pod. Model weights are mounted, not baked in (S3-backed PVC).

---

## 5. CLI build & distribution

`apps/cli/` builds via `make cli-release`:

- `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`, `windows/amd64`.
- Uploaded to GH Releases.
- Homebrew tap (`exascale/exascale`): published in CI on tag.
- `apt` repo (Debian/Ubuntu) hosted at `https://pkg.exascale.io/apt`.
- `pip install exascale` is a thin Python wrapper that vendors the binary (for `pip`-friendly users).
- Direct binary: `curl -sSL https://exascale.io/install.sh | sh`.

CLI release pipeline is owned by `platform-core` for the build content; `infra-sre` for the
distribution infra.

---

## 6. K8s deployment shape

`deploy/k8s/<service>/` per service, using Kustomize:

```
deploy/k8s/credit-ledger/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── hpa.yaml
│   ├── networkpolicy.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    ├── staging/
    ├── prod-paper/
    └── prod-real/
```

Standard deployment shape:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: credit-ledger
spec:
  replicas: 3                # minimum for HA; HPA scales up
  strategy:
    type: RollingUpdate
    rollingUpdate: { maxSurge: 1, maxUnavailable: 0 }
  template:
    spec:
      containers:
        - name: credit-ledger
          image: exascale/credit-ledger:$(VERSION)
          ports: [{ containerPort: 8000 }]
          env: # from configmap + Vault-injected secrets
            - { name: EXASCALE_ENV, valueFrom: { configMapKeyRef: { name: env, key: env } } }
            - { name: EXASCALE_PAPER, valueFrom: { configMapKeyRef: { name: env, key: paper } } }
          livenessProbe:  { httpGet: { path: /healthz, port: 8000 }, periodSeconds: 10 }
          readinessProbe: { httpGet: { path: /readyz,  port: 8000 }, periodSeconds: 5  }
          resources: # tuned per service
            requests: { cpu: 100m, memory: 256Mi }
            limits:   { cpu: 1,    memory: 1Gi   }
          securityContext:
            runAsNonRoot: true
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities: { drop: [ALL] }
```

Secrets via Vault Agent sidecar; no plaintext secrets in ConfigMaps.

---

## 7. Database deployment

PostgreSQL via CloudNativePG operator (Postgres 16):
- One cluster per environment.
- One database per service (`platform_core`, `credit_ledger`, ...). Shared schema types live in
  `docs/contracts/schemas/`; service-owned migrations live in `services/*/migrations/`.
- Read replicas for analytics queries.
- Continuous archiving (WAL-G to S3-compatible storage).
- Daily logical backup + monthly restore test (owned by `infra-sre`).

**Backup/restore drill (`make backup-restore-drill` · `scripts/backup-restore-drill.sh`).**
Proves the relational state — above all the **credit ledger** — survives a dump+restore intact.
It `pg_dump`s the `exascale` DB, restores it into a scratch DB, then asserts (1) **every table's row
count** matches the live DB and (2) the **ledger hash-chain digest** (`md5` over `credit_transactions.
chain_hash` ordered by `tx_id`) is **byte-identical** — any tamper/loss changes the digest. Postgres-only
(needs just the data plane). Verified live: 14 hash-chained txns across 3 tenants restored with an
unchanged digest. Run it after every backup-tooling change and as the monthly restore test.

TimescaleDB (separate cluster) for time-series:
- Index prints history.
- Candlestick data (kept for the keep-warm trading UI).
- Usage events feed (per-request inference logs, GPU-hour records).
- Hypertables with sensible chunk sizes per dataset.

Redis: order book sorted sets (kept for the matching-engine mock; used immediately for caching).
NATS: JetStream for trade/usage/payout events.

---

## 8. CI/CD pipeline

```
PR opened
   │
   ▼
[lint + unit tests per service]     ← <2 min, parallel across services
   │
   ▼
[contract check]                    ← OpenAPI lint, schema diff vs. main
   │
   ▼
[build images on push to main]
   │
   ▼
[deploy to dev]                     ← auto
   │
   ▼
[integration + e2e tests in dev]    ← <10 min
   │
   ▼
[deploy to staging]                 ← auto on main
   │
   ▼
[smoke tests]                       ← <5 min
   │
   ▼
[manual approval gate]
   │
   ▼
[deploy to prod-paper, then prod-real]
```

Each stage:
- Fails closed.
- Posts status to the PR.
- Records the SBOM + image digest to `deploy/sbom/`.

`security-compliance` adds gates:
- Secrets scan (gitleaks).
- Container image scan (Trivy) — no high-severity CVEs in base images.
- Audit-log presence check on any PR touching `credit-ledger`, `matching-engine`, or admin paths.

---

## 9. Observability stack (deployed once, consumed by all services)

`deploy/grafana/` includes preconfigured:
- **Prometheus** scraping `/metrics` from every service.
- **Loki** ingesting structured logs (JSON, one event per line, with `trace_id`).
- **Tempo** for OTLP traces.
- **Grafana** dashboards per service (templates in `deploy/grafana/dashboards/`).

SLO dashboards per the `infra-sre` agent file:
- Trading engine 99.95% / order match P99 <10ms (kept for keep-warm; reactivates with exchange).
- Inference API 99.95%, time-to-first-token P95.
- Compute 99.9%, instance-time-to-running P95.
- Ledger: hash-chain integrity must be 100% (any mismatch = page).
- Index publication: 100% (any miss = page).

---

## 10. Secrets management

- Vault is the source of truth for every secret.
- Services authenticate to Vault via Kubernetes Service Account (K8s auth method).
- CI uses a short-lived Vault token issued per pipeline run.
- Local dev: Vault in dev mode (`vault server -dev`), preloaded by `make seed`.
- Rotation: monthly for DB credentials, quarterly for OAuth client secrets.

`security-compliance` audits secret usage in every PR.

---

## 11. Per-agent deployment checklist

Definition of done for "deploy-ready" per service:

- [ ] `Dockerfile` follows the template (§2/3/4).
- [ ] Image builds in CI; SBOM produced; scan passes.
- [ ] `deploy/k8s/<service>/` Kustomize base + 4 overlays.
- [ ] Service registered with Kong (route added to gateway config).
- [ ] Liveness/readiness probes implemented and tested.
- [ ] Prometheus metrics exposed; one dashboard checked in to `deploy/grafana/dashboards/`.
- [ ] Alerts defined for the SLOs that apply to this service.
- [ ] Runbook in `services/<name>/RUNBOOK.md` (oncall playbook).
- [ ] Migration plan reviewed by `tech-lead` for shared schema impact.
- [ ] `is_paper` enforced at runtime (matches DB column / event field).

---

## 12. Roll-back

Every deployment is reversible:

```
kubectl -n exascale-prod-paper rollout undo deploy/<service>
```

For database migrations: every migration ships with a tested `down`. `credit-ledger` is the
hard case — it's append-only — so its "rollback" is a forward compensating-entry migration.
That is documented per migration.

`infra-sre` owns the roll-back drill: practiced quarterly.
