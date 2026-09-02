# Repo Layout

This documents the monorepo layout each agent builds into. It matches `CLAUDE.md`'s declared
structure. The existing `1Trade Frontend/` directory stays where it is for now; it migrates
to `apps/web/` in a planned step (§3 below).

---

## 1. Target structure

```
/home/dministrator/projects/ex/
├── CLAUDE.md                            ← global ground truth (already exists)
├── README.md                            ← repo entry (create in F01)
├── Makefile                             ← `make up`, `make test`, `make build` (create in F01)
│
├── docs/
│   ├── re/                              ← source docs (PRD, GTM update, settlement arch) [exists]
│   ├── plans/                           ← THIS PLAN (already created)
│   │   ├── README.md                    ← master plan
│   │   ├── REPO_LAYOUT.md               ← this file
│   │   ├── LOCAL_DEV.md
│   │   ├── DEPLOYMENT.md
│   │   ├── CONTRACTS.md
│   │   ├── SEQUENCING.md
│   │   ├── agents/                      ← one .md per agent
│   │   └── features/                    ← one .md per feature
│   ├── contracts/                       ← THE SHARED CONTRACTS (already created, see CONTRACTS.md)
│   │   ├── openapi/                     ← one yaml per service
│   │   ├── schemas/                     ← SQL migrations + shared DB types
│   │   ├── events/                      ← NATS subjects + payload schemas
│   │   └── credit-types.md              ← canonical credit-type enum
│   └── htmls/                           ← legacy HTML mockups (source for trading-frontend) [exists]
│
├── services/                            ← Go services (one dir each)
│   ├── platform-core/                   ← auth, accounts, RBAC, billing, gateway config
│   ├── credit-ledger/                   ← balances + tx + hash chain
│   ├── inference-gateway/               ← OpenAI-compatible router
│   ├── inference-runtime/               ← Python + vLLM workers (containerized)
│   ├── compute-control/                 ← K8s control plane bindings
│   ├── supply-service/                  ← settlement-trust home (DC onboarding, payout, attestation)
│   ├── index-service/                   ← private reference index (keep-warm)
│   ├── matching-engine/                 ← (paused / mock adapter only)
│   ├── market-maker/                    ← (paused, spec only)
│   └── surveillance/                    ← (basic abuse only)
│
├── apps/
│   ├── web/                             ← Nuxt 4 frontend (after migration from 1Trade Frontend/)
│   └── cli/                             ← `1trade` Go CLI
│
├── deploy/
│   ├── docker/                          ← Dockerfile patterns + per-service Dockerfiles
│   ├── k8s/                             ← Kustomize/Helm: per-service manifests + overlays
│   │   └── local/                       ← k3d cluster config + Tiltfile for local dev
│   ├── terraform/                       ← IaC (added M3; env-scoped: staging, prod-paper, prod-real)
│   └── ci/                              ← GH Actions pipelines (build → Helm/Kustomize apply)
│
├── 1Trade Frontend/                   ← CURRENT frontend (own git repo, to be merged) [exists]
│
└── .claude/
    ├── agents/                          ← agent definitions [exists]
    └── settings.local.json
```

---

## 2. Per-service shape (Go)

Every Go service follows the same skeleton. This is what every backend agent scaffolds first.

```
services/<name>/
├── cmd/
│   └── <name>/
│       └── main.go                      ← thin: parse env, wire deps, start server
├── internal/                            ← implementation (not importable from other services)
│   ├── api/                             ← HTTP/gRPC handlers
│   ├── domain/                          ← business logic (testable, no IO)
│   ├── store/                           ← Postgres/Redis/NATS bindings
│   └── config/                          ← env-driven config
├── migrations/                          ← service-owned migrations; shared types in docs/contracts/schemas/
├── go.mod
├── go.sum
├── Dockerfile                           ← see DEPLOYMENT.md for the template
└── README.md                            ← run-locally + endpoints + envs
```

**Forbidden:** importing another service's `internal/`. Cross-service coupling goes through
`docs/contracts/` only.

---

## 3. Frontend migration plan ("1Trade Frontend/" → "apps/web/")

The existing frontend is a separate git repo. To merge it cleanly while preserving history:

```
# from repo root
git remote add frontend-orig "/home/dministrator/projects/ex/1Trade Frontend"
git fetch frontend-orig
git merge --allow-unrelated-histories frontend-orig/main \
  -m "Merge 1Trade Frontend repo as apps/web/"

# then rewrite paths so all frontend files land under apps/web/
git filter-repo --to-subdirectory-filter apps/web   # (run in a temp clone of just the frontend repo first)
```

Owner of this migration step: `trading-frontend` agent, with `tech-lead` approval.
Trigger: scheduled for end of Milestone 1, **once** the foundation infra and CI are green so the
move doesn't break the dev experience for the frontend engineer.

Until then, the frontend stays in `1Trade Frontend/` and CI runs against that path. Plan docs
refer to both `apps/web/` (target) and `1Trade Frontend/` (current); after migration the
plan docs are updated in one PR.

---

## 4. Naming conventions

- Go module path: `github.com/trade1/<name>` (subject to `tech-lead` finalizing the org name).
- Container image name: `1trade/<service>` published to the org registry.
- K8s namespace per environment: `1trade-dev`, `1trade-staging`, `1trade-prod`.
- **Paper / real isolation also at the infra layer:** `1trade-paper-*` and `1trade-real-*`
  namespaces, separate secrets, separate Postgres clusters (even though trading is paused, the
  ledger keeps `is_paper` so the split is real today).

---

## 5. Where each agent lives

| Agent | Owns these paths |
|---|---|
| `tech-lead` | `docs/contracts/`, `docs/plans/` |
| `infra-sre` | `deploy/`, `Makefile`, root README |
| `platform-core` | `services/platform-core/`, `apps/cli/` |
| `credit-ledger` | `services/credit-ledger/` |
| `inference-ml` | `services/inference-gateway/`, `services/inference-runtime/` |
| `compute-platform` | `services/compute-control/`, `deploy/k8s/compute/` |
| `settlement-trust` | `services/supply-service/`, attestation deploys under `deploy/k8s/supply/` |
| `index-service` | `services/index-service/` |
| `matching-engine` | `services/matching-engine/` (mock adapter only in Phase 1) |
| `market-maker` | `services/market-maker/` (spec only) |
| `surveillance` | `services/surveillance/` (abuse-only in Phase 1) |
| `trading-frontend` | `apps/web/` (target), `1Trade Frontend/` (current) |
| `security-compliance` | reviewer; touches nothing directly; comments on PRs |

---

## 6. What's checked in vs. ignored

- **In:** source, tests, migrations, Dockerfiles, K8s manifests, OpenAPI contracts, docs.
- **Out (`.gitignore`):** `node_modules/`, `.nuxt/`, `.output/`, `bin/`, `vendor/` (Go),
  `.env*` (except `.env.example`), Terraform state, secrets of any kind.

`security-compliance` gates: any PR touching credentials gets blocked at review.
