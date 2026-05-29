# Agent plan — `infra-sre`

> Active, Milestone 1 foundation owner. **Ships first** so every other agent has ground to stand on.

## 1. Scope under the GTM pivot

Unchanged from the agent definition. Foundation, IaC, K8s, data plane, CI/CD, observability,
secrets, deployment, SLA hardening. With the pivot, the trading-engine SLO target moves to
Phase 2; the focused-now SLOs are inference, compute, ledger integrity, and (private) index
publication reliability.

## 2. Features owned

| Feature | Status |
|---|---|
| [F01 — Foundation infra](../features/F01-foundation-infra.md) | active, Milestone 1 |
| [F21 — SOC 2 Type I](../features/F21-soc2-type1.md) | active (co-owner with `security-compliance`), M4–M6 |
| Cross-cutting: observability stack, CI/CD, secrets | active, continuous |

## 3. Milestone-by-milestone

> **Platform decision:** Kubernetes is the single substrate — **k3s** in prod, **k3d** (k3s-in-Docker)
> locally — kept lightweight and hardened progressively (see `docs/plans/DECISIONS.md` ADR-0001).
> The heavy enterprise pieces (Kong, Vault, Terraform) arrive only when a milestone needs them.

### Milestone 1 — Foundation (CRITICAL)
- **k3s** clusters: `staging` + `prod-paper`; separate `prod-real` cluster (empty, paused). On the
  GPU fleet (owned DC) k3s runs directly; managed K8s is an option for stateless-only nodes.
- **k3d** local cluster config (`deploy/k8s/local/k3d.yaml`) — same k3s, so local == prod.
- Data plane (Helm, in-cluster): CloudNativePG operator + TimescaleDB + Redis + NATS JetStream.
- Ingress: **Traefik** (k3s built-in) + Cloudflare edge. (Kong arrives M3 — see ladder below.)
- Secrets: **Kubernetes Secrets + SOPS** (sealed in git). (Vault arrives M4 for SOC 2.)
- IaC: **Helm + Kustomize** + a thin bootstrap script. (Terraform arrives M3 as envs multiply.)
- Grafana stack (Prometheus + Loki + Tempo).
- CI/CD: GH Actions → build images → Helm/Kustomize apply (per `DEPLOYMENT.md` §8).
- Repo skeleton: root `Makefile`, `.env.example`, `docs/contracts/` scaffolding.

**Progressive-hardening ladder** (add each only when its milestone demands it):
`ingress` Traefik (M1) → Kong behind Cloudflare (M3) · `secrets` K8s Secrets + SOPS (M1) →
Vault + Agent injector (M4) · `IaC` Helm/Kustomize (M1) → + Terraform (M3).

### Milestone 2 — Wired observability + inference data plane
- Service dashboards for `platform-core`, `credit-ledger`, `inference-gateway` (the services going
  live this milestone).
- Alert routing (PagerDuty / equivalent) for ledger hash-chain mismatch, gateway 5xx, K8s.
- TimescaleDB hypertables for `inference.usage.v1` events.

### Milestone 3 — Compute scale + gateway/IaC hardening
- GPU nodepool sizing for `prod-paper` (H100 + H200 anchor); NVIDIA GPU Operator + Kueue + Volcano
  on the GPU nodes.
- **Kong** gateway behind Cloudflare (replaces raw Traefik routing as API-gateway features —
  per-tenant rate tiers, auth plugins — are needed).
- **Terraform** introduced for multi-environment provisioning as envs multiply.
- Network: InfiniBand groundwork for clusters (lands fully in M5).
- Backups: ledger hourly logical + WAL archive; monthly restore drill scheduled.

### Milestone 4 — Partner DC integration + SOC 2 controls
- Network topology for partner DC agents (mTLS + private connectivity).
- **Vault** + Agent injector stood up; migrate secrets off SOPS to Vault (SOC 2 driver).
- SOC 2 controls landing: SSO into every infra tool, audit log shipping, change management gate.
- `prod-real` cluster receives its first deploys (still trading-paused, just for real-money
  inference and compute consumption customers).

### Milestone 5 — InfiniBand + multi-cluster
- InfiniBand fabric live; gang-scheduling tested at 32–64 GPU scale.
- Second partner DC's connectivity onboarded.
- Backup + restore drills passed.

### Milestone 6 — Reliability hardening + SOC 2 Type I
- Reliability targets met (see §6 below).
- SOC 2 audit completes with `security-compliance`.
- Runbooks for every service collected in `services/*/RUNBOOK.md`.

## 4. Contracts owned / consumed

Owned: none (infra doesn't own application contracts). Provides:
- The Prometheus metrics convention every service must follow.
- The OTLP tracing convention.
- The structured-log JSON shape (`trace_id`, `tenant_id`, `is_paper`, `service`, `level`, `msg`, ...).
- The Kong route registration pattern.

Consumed: every service's `Dockerfile`, K8s manifests, env requirements.

## 5. Local dev

- `make up` creates the `k3d` cluster and deploys the data plane + services (see `LOCAL_DEV.md`).
- All preconfigured dashboards loaded at `http://localhost:9000`.
- `infra-sre` owns `LOCAL_DEV.md` and keeps it current.

## 6. SLOs owned

| SLO | Target | Notes |
|---|---|---|
| Inference API uptime | 99.95% | F08; M2 target onwards |
| Compute API uptime | 99.9% | F12/F13; M3 onwards |
| Credit ledger atomicity & hash integrity | 100% mismatch-free | F05; M1 onwards |
| Private index daily publication | 100% (never miss) | KW01; M5 onwards |
| Trading engine (Phase 2) | 99.95% / P99 <10ms match | inactive until switch-on |

## 7. Dockerfile / deploy

`infra-sre` doesn't ship a service Dockerfile per se. It does ship:
- `deploy/docker/Dockerfile.go-service` (template; see `DEPLOYMENT.md` §2).
- `deploy/docker/Dockerfile.nuxt` (frontend).
- `deploy/docker/Dockerfile.vllm` (inference runtime — shared with `inference-ml`).
- Per-env Kustomize overlays for every service (the base lives with the service).
- Terraform modules for: cluster, Postgres, TimescaleDB, Redis, NATS, Kong, Vault, observability stack.

## 8. Definition of done

- One-command reproducible environments (`make up` works clean-clone).
- CI/CD green for every service.
- Dashboards + alerts cover the SLOs in §6.
- Data plane HA + backed up; restore drill passed.
- No secrets in code; Vault is sole source of truth.
- `prod-real` cluster exists, isolated, ready (even though gated).
