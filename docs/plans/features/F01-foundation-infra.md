# F01 — Foundation infra

> Ship in **Milestone 1**. Owner: `infra-sre`. **Blocks everything.**

## Spec

The platform must run end-to-end on a developer laptop (`make up`), in staging, and in
`prod-paper` from Milestone 1. `prod-real` cluster exists but is paused (no customer traffic yet).

Platform per `DECISIONS.md` ADR-0001: **k3s/k3d**, lightweight, progressively hardened.

Concrete deliverables:

1. **Kubernetes platform**: `k3s` clusters for `staging` + `prod-paper` (+ paused `prod-real`),
   with Kueue, Volcano, NVIDIA GPU Operator (real or mock-GPU mode), cert-manager, and the
   built-in Traefik ingress behind Cloudflare. (Kong arrives M3; Terraform arrives M3.)
2. **Data plane (Helm, in-cluster)**: CloudNativePG (multi-DB), TimescaleDB, Redis, NATS JetStream.
3. **Local dev**: `k3d` cluster config (`deploy/k8s/local/k3d.yaml`) + Helm data plane + a
   `Tiltfile` for live-reload; `Makefile` targets (`make up`, `make seed`, `make web`, `make cli`,
   `make test`). Same k3s as prod — no local/prod drift.
4. **CI/CD**: GH Actions → build images → Helm/Kustomize apply (per `DEPLOYMENT.md` §8).
5. **Observability**: Prometheus + Loki + Tempo + Grafana with preconfigured dashboards,
   service-discovery for every service.
6. **Secrets**: Kubernetes Secrets + SOPS (sealed in git) for M1; Vault + Agent injector at M4.
7. **Repo scaffolding**: `services/`, `apps/`, `deploy/`, `docs/contracts/`, root `Makefile`,
   `.env.example`, `.gitignore` updated.

## Owning agent

`infra-sre`. Coordinates with `tech-lead` (who creates the seed contracts in parallel).

## Contracts consumed / produced

- Produces: the env conventions (`DEPLOYMENT.md` §2, §4 here), the Prometheus metrics scrape
  conventions, the structured-log JSON shape, the OTLP exporter convention.
- Consumes: nothing — this is the bottom of the stack.

## Dependencies

- None upstream.
- Blocks: all other features.

## Sync points

- Day 1 — repo scaffolding pushed; agents can start creating service directories.
- Day 5 — `kind` local dev works; agents can run their service against the local data plane.
- Day 10 — staging cluster green; first service-side deploys begin.
- Day 20 — `prod-paper` cluster green; ready for Milestone 2 service launches.

## Acceptance criteria

- [ ] `git clone && make up` works on a clean dev laptop in <15 min.
- [ ] CI runs lint + unit + integration in <15 min total.
- [ ] Staging cluster receives a deploy of every scaffolded service (`hello-world` if needed).
- [ ] `prod-paper` cluster exists, locked down with allow-list, ready to receive its first
      Milestone 2 deploy.
- [ ] `prod-real` cluster exists, isolated, gated.
- [ ] Grafana dashboards render for the data plane and the ingress.
- [ ] Secrets work via Kubernetes Secrets + SOPS (sealed in git); no plaintext secrets committed.
      (Vault migration is M4 — ADR-0001.)
- [ ] Ledger backup/restore drill passes (using an empty ledger as the test).
- [ ] `make test-e2e` skeleton runs (will fill in as services ship).

## Milestone

End of Milestone 1 — Decision Gate 1.
