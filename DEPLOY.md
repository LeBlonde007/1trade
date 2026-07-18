# Exascale — Deployment Setup

The as-built deployment of this project: how it runs locally, how it ships to a sandbox box, and how
CI/CD builds the images. This is the practical "where does it run and how do I deploy it" map. For the
target-state design (4 prod environments, Vault, Kong, CloudNativePG, SLOs) see
[`docs/plans/DEPLOYMENT.md`](docs/plans/DEPLOYMENT.md); for the current-state local runbook see
[`docs/plans/RUN_LOCAL.md`](docs/plans/RUN_LOCAL.md).

## Platform decision

**Kubernetes is the single substrate — `k3d` locally, `k3s` in prod** (same k3s binary, so local == prod;
no second deployment model). Not Dokploy/Swarm: those can't do GPU gang-scheduling, MIG, or Kueue/Volcano.
Rationale: `docs/plans/DECISIONS.md` **ADR-0001**.

| Concern        | Now                                            | Hardens to                  |
| -------------- | ---------------------------------------------- | --------------------------- |
| Orchestration  | k3s / k3d                                      | same (k3s scales)           |
| Ingress        | Traefik (k3s built-in)                         | Kong behind Cloudflare (M3) |
| Secrets        | K8s Secrets + SOPS/age                         | Vault + agent injector (M4) |
| IaC            | Kustomize (+ Helm for data plane)              | + Terraform (M3)            |
| GPU scheduling | NVIDIA GPU Operator + Kueue + Volcano (opt-in) | same (M2→M5)                |

## Services that deploy (the six images)

| Image                    | Source                            | Local port | Role                                                      |
| ------------------------ | --------------------------------- | ---------- | --------------------------------------------------------- |
| `platform-core`          | `services/platform-core`          | `:8001`    | auth · signup/login → JWT · billing · orgs/RBAC           |
| `credit-ledger`          | `services/credit-ledger`          | `:8002`    | balances · convert · hash-chained debits                  |
| `inference-gateway`      | `services/inference-gateway`      | `:8085`    | API · key auth · metering                                 |
| `inference-runtime-stub` | `services/inference-runtime/stub` | `:8000`    | CPU stub (canned completions; real vLLM needs a GPU node) |
| `compute-control`        | `services/compute-control`        | `:8086`    | GPU control plane · catalog · quota · mock-GPU scheduler  |
| `web`                    | `Exascale Frontend/`              | `:3000`    | Nuxt console (BFF proxies to the services in-cluster)     |

The data plane (Postgres · TimescaleDB · Redis · NATS · Mailpit) deploys separately from the services in
namespace `data`. Optional tiers add scheduling (Kueue/Volcano) and observability (Prometheus/Loki/Tempo/
Grafana).

---

## 1. Local development (k3d + Tilt)

One command brings up the whole platform in a local k3d cluster with live-reload.

```bash
bash scripts/install-toolchain.sh   # first time: docker · k3d · kubectl · helm · tilt · go 1.25 · node
make up                             # create k3d cluster + data plane + `tilt up`
```

`make up` = `cluster` + `data-plane` + `tilt`:

1. **`cluster`** — creates (or starts) the k3d cluster `exascale` from `deploy/k8s/local/k3d.yaml`
   (1 server + 1 agent, Traefik LB on `:8080`/`:8443`, a managed local registry on `:5111`, the agent
   labelled `exascale.io/gpu=mock`).
2. **`data-plane`** — `deploy/k8s/local/install-data-plane.sh` applies the core data plane into namespace
   `data` (Postgres, TimescaleDB, Redis, NATS, Mailpit) and waits for rollouts.
3. **`tilt`** — `tilt up` (driven by the `Tiltfile`): builds each service image, applies its Kustomize
   base, generates the `platform-auth` secret, applies DB migration ConfigMaps, and **holds the
   port-forwards** open. Edit source → Tilt live-updates the pod.

Tilt is guarded to `allow_k8s_contexts('k3d-exascale')` — it will never act on a non-local cluster.

### Port-forwards Tilt holds open

| localhost | service                         |
| --------- | ------------------------------- |
| `:8001`   | platform-core                   |
| `:8002`   | credit-ledger                   |
| `:8085`   | inference-gateway               |
| `:8000`   | inference-runtime (CPU stub)    |
| `:8086`   | compute-control                 |
| `:8025`   | Mailpit web UI (captured email) |

Run the web console against these with `cd "Exascale Frontend" && make web` (or `npm run dev`) → `:3000`.
The BFF defaults its upstreams to `localhost:8001/8085/8002` — no env needed.

### Optional tiers

| Command                  | Adds                                                                                                             |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `make up GPU=1`          | NVIDIA GPU Operator — schedule the real host GPU (needs NVIDIA Container Toolkit in WSL2)                        |
| `make sched` / `SCHED=1` | Kueue + Volcano + mock-GPU resource + project Kueue config (queues: inference · training-small · training-large) |
| `OBS=1`                  | Prometheus + Loki + Tempo + Grafana (Helm, heavy)                                                                |
| `FULL=1`                 | SCHED + OBS together                                                                                             |

Default is **mock-GPU mode** — virtual GPUs, no driver, so the control plane behaves identically on any
machine.

### Key Makefile targets

| Target                                     | Does                                                                      |
| ------------------------------------------ | ------------------------------------------------------------------------- |
| `make up`                                  | cluster + data plane + Tilt (the one-command bring-up)                    |
| `make down`                                | delete the local cluster                                                  |
| `make cli`                                 | build the `exascale` CLI → `bin/exascale`                                 |
| `make build` / `make test`                 | build / race-test every Go module (`scripts/go-all.sh`)                   |
| `make lint` / `make fmt`                   | golangci-lint / gofmt across modules                                      |
| `make test-e2e`                            | timed sub-5-min time-to-first-action loop (`scripts/e2e.sh`)              |
| `make backup-restore-drill`                | dump+restore the ledger DB; assert row counts + hash-chain digest survive |
| `make secrets-apply` / `make secrets-edit` | decrypt+apply / edit the SOPS secret                                      |
| `make sandbox-bundle`                      | build the no-source deploy bundle                                         |
| `make ps`                                  | `kubectl get pods -A`                                                     |

> Note: a few `LOCAL_DEV.md` conveniences (`make seed`, `:8080` ingress, Grafana, `exascale login --dev`)
> are **not wired yet** — follow `RUN_LOCAL.md` for the accurate current-state path (reach services via
> the Tilt port-forwards; create credits the real way).

---

## 2. Sandbox deploy (single-node k3s on a VPS)

A hosted, full-journey **paper-mode** deployment of the whole product: no GPU, no real Stripe, no KYC, no
real money — every path runs on its mock/stub. Driven by `scripts/deploy-sandbox.sh` + the Kustomize
overlay `deploy/k8s/overlays/sandbox/`.

**Box:** 4 vCPU / 8 GB / 80 GB, Ubuntu 22.04/24.04, ports 80/443. (2 vCPU/4 GB works for the no-source
path.) For HTTPS: two A records — `SANDBOX_HOST` (web) and `SANDBOX_API_HOST` (the `/v1` API for the CLI).

```bash
# HTTP, install k3s, serve at the node IP:
INSTALL_K3S=1 SANDBOX_HOST=<vps-ip> scripts/deploy-sandbox.sh

# HTTPS on real domains (web + api), Let's Encrypt:
INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
  SANDBOX_HOST=sandbox.example.com SANDBOX_API_HOST=sandboxapi.example.com \
  scripts/deploy-sandbox.sh
```

The script (idempotent) installs k3s, builds the six images and imports them into k3s containerd, creates
the `platform-auth` secret + migration ConfigMaps, applies the data plane, (TLS=1) installs cert-manager +
a Let's Encrypt issuer, renders the overlay (substituting host/URL), applies services + web + ingress,
waits for rollouts, and prints the URL.

The overlay is thin over the bases (which are already paper-mode): it adds the web frontend, one public
Ingress, retags images `:sandbox`, points `APP_BASE_URL` at the public URL, and **hardens
`EXASCALE_ENV=sandbox`** so the `X-Dev-Tenant` auth shortcut is off and a Traefik `strip-dev-headers`
middleware deletes dev headers at the edge. Credit minting needs the internal `SERVICE_TOKEN` (never
public) — so on the sandbox you get credits the real way (sign up → buy credits → MockStripe settles
instantly).

**No-source deploy** (a box you don't control — e.g. a client droplet): don't build on the box. Publish
images via CI (`ghcr.io/<owner>/exascale-<svc>:sandbox`), build a manifests-only bundle with
`make sandbox-bundle` → `sandbox-bundle.tar.gz`, then on the box deploy in pull-mode:

```bash
tar xzf sandbox-bundle.tar.gz
IMAGE_REGISTRY=ghcr.io/<owner> \
  REGISTRY_USER=<owner> REGISTRY_TOKEN=<read:packages PAT> \   # omit both if packages are public
  INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
  SANDBOX_HOST=sandbox.example.com SANDBOX_API_HOST=sandboxapi.example.com \
  scripts/deploy-sandbox.sh
```

**Real inference output** (optional, no GPU): pass `INFERENCE_API_KEY=sk-or-...` to point the gateway at a
hosted OpenAI-compatible provider (OpenRouter by default) — metering and ledger debit are unchanged; only
the upstream producing tokens differs.

Full detail + security notes: [`deploy/k8s/overlays/sandbox/README.md`](deploy/k8s/overlays/sandbox/README.md).

---

## 3. CI/CD (GitHub Actions)

**`.github/workflows/ci.yml`** — gates correctness on every PR + push to main (fails closed):

- **hygiene** — gitleaks (no secrets) + pre-commit (`golangci-lint` skipped here; it's the hard gate below).
- **contracts** — OpenAPI lint (advisory) + structural YAML parse of event/k8s manifests (hard).
- **go** — `scripts/go-all.sh build` then `test -race`, then `golangci-lint v2.12.2` per module (hard gate; Go 1.25).

**`.github/workflows/release.yml`** — builds + pushes the six images to GHCR (`ghcr.io/<owner>/exascale-<svc>`):

| Trigger           | Tags                                        |
| ----------------- | ------------------------------------------- |
| push to `main`    | `:latest`, `:sandbox`, `:sha-<short>`       |
| push tag `vX.Y.Z` | `:X.Y.Z`, `:X.Y`, `:sha-<short>`            |
| pull_request      | builds only, no push (packaging smoke test) |
| workflow_dispatch | manual                                      |

A matrix builds each service via Buildx with GHA layer cache. The `web` image additionally cross-compiles
the CLI into its `public/cli` dir (served at `/cli/install.sh`). Images are **private by default** — make
the GHCR packages public or add an `imagePullSecret` for a cluster to pull. Pushing needs no extra secret
(the built-in `GITHUB_TOKEN` has `packages:write`).

> The multi-environment promotion pipeline (dev → staging → prod-paper/prod-real, Trivy/SBOM gates) in
> `docs/plans/DEPLOYMENT.md` §8 is the target design; the live workflows above are CI + image publish.

---

## 4. Container images

Build templates in `deploy/docker/` (per-service `Dockerfile`s live in each service dir):

- **Go services** (`Dockerfile.go-service`) — multi-stage, distroless `static-debian12:nonroot`, static
  CGO-free binary, one HTTP port, `/healthz` + `/readyz` probes.
- **Nuxt web** (`Dockerfile.nuxt`) — `node:20-alpine` build → node SSR runtime on `:3000`.
- **vLLM runtime** (`Dockerfile.vllm`) — `nvcr.io/nvidia/pytorch` + vLLM (~12 GB; built per vLLM release,
  weights mounted not baked). Locally the CPU stub stands in for this.

## 5. Kubernetes manifests (`deploy/k8s/`)

Kustomize, one dir per service with a `base/` (deployment · service · configmap · kustomization):

```
deploy/k8s/
├── local/            k3d.yaml · data-plane.yaml · install-data-plane.sh · seed.sh
├── platform-core/base
├── credit-ledger/base
├── inference-gateway/base
├── inference-runtime/base   (+ runtime-gpu.yaml)
├── compute-control/base
├── web/base
├── overlays/sandbox/  the single-node k3s paper-mode overlay + Ingress
├── scheduling/        Kueue config · mock-GPU resource · Volcano/Kueue job examples
└── observability/     ServiceMonitors
```

## 6. Secrets

- **Local/sandbox:** plain Kubernetes Secret `platform-auth` (`PLATFORM_JWT_SECRET` + `SERVICE_TOKEN`),
  generated on first run (Tilt / deploy script), never committed.
- **Committed/rotatable:** SOPS + age — `deploy/secrets/platform-auth.sops.yaml`, applied with
  `make secrets-apply` (decrypts in memory, never writes plaintext to disk) and edited with
  `make secrets-edit`. Config in `.sops.yaml`; key handling in `deploy/secrets/README.md`.
- **Target:** Vault (K8s auth method, short-lived CI tokens) — M4 hardening.

## 7. Data plane & backups

Core data plane in namespace `data`: **Postgres** (state) · **TimescaleDB** (prints/candles/usage) ·
**Redis** (cache, order-book sets for the kept-warm matching mock) · **NATS** JetStream (usage/trade/payout
events) · **Mailpit** (captured outbound email — verification, receipts).

DB migrations ship with the code (`services/*/migrations/`, shared types in
`docs/contracts/schemas/types.sql`) and are applied via ConfigMaps by an initContainer (locally Tilt
creates them).

`make backup-restore-drill` (`scripts/backup-restore-drill.sh`) proves the relational state — above all the
append-only credit ledger — survives a `pg_dump`+restore: it asserts every table's row count matches and
the ledger **hash-chain digest** is byte-identical (any tamper/loss changes it).

---

## Quick reference

```bash
make up                  # local: k3d + data plane + Tilt (live-reload)
make down                # delete local cluster
make cli && ./bin/exascale signup --email you@dev.test --password pw
make test-e2e            # timed signup→topup→infer→gpu-debit loop

# sandbox VPS (paper mode):
INSTALL_K3S=1 SANDBOX_HOST=<vps-ip> scripts/deploy-sandbox.sh
```

| Path                        | What                                                   |
| --------------------------- | ------------------------------------------------------ |
| `Makefile`                  | developer entrypoint (targets above)                   |
| `Tiltfile`                  | local build + deploy + port-forwards                   |
| `deploy/k8s/`               | Kustomize bases + sandbox overlay + data plane         |
| `deploy/docker/`            | Dockerfile templates                                   |
| `deploy/secrets/`           | SOPS/age secrets                                       |
| `scripts/deploy-sandbox.sh` | single-node k3s VPS deploy                             |
| `.github/workflows/`        | `ci.yml` (lint/test) · `release.yml` (build+push GHCR) |
| `docs/plans/DEPLOYMENT.md`  | full target-state design                               |
| `docs/plans/RUN_LOCAL.md`   | accurate current-state local runbook                   |

</content>
</invoke>
