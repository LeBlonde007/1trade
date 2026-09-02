# 1Trade

A commodity market for AI compute — built **platform-first**: inference + GPU compute + prepaid,
redeemable credits for AI startups now; the tradeable **exchange** is designed and kept warm,
switched on once licensed. See the GTM pivot in `docs/re/1trade_gtm_focus_update.md`.

> **Start here:** [`docs/plans/README.md`](docs/plans/README.md) — the master build plan
> (sequencing, agents, features). Ground rules live in [`CLAUDE.md`](CLAUDE.md) and
> [`docs/plans/ENGINEERING_STANDARDS.md`](docs/plans/ENGINEERING_STANDARDS.md).

## Quick start (local dev)

```bash
# 1. install the toolchain (Go, kubectl, k3d, helm, tilt, sops, …) — WSL2/Ubuntu
bash scripts/install-toolchain.sh

# 2. bring up the platform locally (k3d cluster + data plane + live-reload)
make up          # add GPU=1 to schedule a host GPU via the NVIDIA Container Toolkit
make seed        # dev tenants + demo credits + mock catalog

# 3. frontend
make web         # Nuxt dev server on :3000
```

`make help` lists all targets. Full walkthrough: [`docs/plans/LOCAL_DEV.md`](docs/plans/LOCAL_DEV.md).

## Layout

```
docs/plans/      the build plan (start at README.md) + ENGINEERING_STANDARDS + DESIGN_SYSTEM + DECISIONS
docs/contracts/  the shared contracts (OpenAPI / SQL / events / credit-types) — owned by tech-lead
services/        Go services (one dir each) — scaffolded per feature
apps/            web (Nuxt) + cli (1trade)
deploy/          docker templates · k8s (k3d local + overlays) · ci
scripts/         dev tooling (install-toolchain.sh, …)
.claude/         agent roster + /ex-* workflow commands
```

Full structure + conventions: [`docs/plans/REPO_LAYOUT.md`](docs/plans/REPO_LAYOUT.md).

## Platform

Kubernetes everywhere — **k3s** in prod, **k3d** locally — kept lightweight and progressively
hardened (Traefik→Kong, K8s Secrets+SOPS→Vault, Helm→+Terraform). Rationale:
[`docs/plans/DECISIONS.md`](docs/plans/DECISIONS.md) ADR-0001.

## Working in this repo

Use the `/ex-*` workflow commands (see `.claude/commands/`):
`/ex-start <feature>` · `/ex-update-feature` · `/ex-fix` · `/ex-review` · `/ex-contract` · `/ex-status`.

Every change follows the definition of done in `ENGINEERING_STANDARDS.md` §10 (tests with code,
a doc comment on every function, fixed-point money math, `is_paper` threaded, idempotent mutations,
audit hooks, `/security-review` on sensitive surfaces).
