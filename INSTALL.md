# 1Trade — Install & Run Locally

Everything you need to install, and the exact steps to bring the platform up on your own machine.
This is the **dependency + local-run** guide. For the full deployment/CI map see [`DEPLOY.md`](DEPLOY.md);
for the deeper current-state runbook see [`docs/plans/RUN_LOCAL.md`](docs/plans/RUN_LOCAL.md).

> **TL;DR — one command, from a clean machine to a running stack:**
> ```bash
> make bootstrap        # OS-aware: installs every dependency, then `make up`
> ```
> `make bootstrap` detects your OS and installs the right way (macOS → Homebrew · Debian/Ubuntu/WSL →
> apt · Fedora/Arch → dnf/pacman), then brings the platform up. Add `GPU=1` to include the NVIDIA
> toolkit. Prefer the steps? Do it in parts:
> ```bash
> bash scripts/setup.sh   # or scripts/install-toolchain.sh on WSL2/Ubuntu — install deps only
> make up                 # k3d cluster + data plane + services (live-reload)
> cd "1Trade Frontend" && npm install && npm run dev   # web console → http://localhost:3000
> ```

---

## 1. System requirements

| Requirement | Detail |
|---|---|
| **OS** | Linux, macOS, or **Windows 11 + WSL2 (Ubuntu 22.04/24.04)** — the toolchain installer targets WSL2/Ubuntu/Debian |
| **CPU / RAM** | 4 vCPU / 8 GB minimum (whole stack in k3d). 2 vCPU / 4 GB works for a slim run |
| **Disk** | ~15 GB free (images + k3d + node_modules) |
| **Docker** | Docker Desktop for Windows with **WSL integration enabled**, or `docker.io` inside WSL. Must be running before `make up` |
| **GPU** | **Not required** — a CPU stub stands in for vLLM. A real GPU is optional (see §5) |

> **Windows note:** run all commands from inside the **WSL2 Ubuntu** shell, not PowerShell. The repo
> lives on the WSL filesystem (`\\wsl.localhost\Ubuntu\home\...`); k3d, Tilt, and the Go/Node toolchain
> run inside Linux.

---

## 2. Dependencies

Everything below is installed for you by **`bash scripts/install-toolchain.sh`** (idempotent — skips
what's already present; `--force` reinstalls; `--gpu` adds the NVIDIA toolkit).

### Core toolchain (required)

| Tool | Version | Purpose |
|---|---|---|
| **Docker** | current | container runtime that hosts the k3d cluster and builds images |
| **Go** | **1.25.x** | builds the five Go services + the `1trade` CLI |
| **Node.js + npm** | **Node 20+** | builds & runs the Nuxt 4 web console |
| **kubectl** | stable | talks to the local Kubernetes (k3d) cluster |
| **k3d** | current | runs k3s (Kubernetes) inside Docker — the local cluster |
| **helm** | v3 | installs the data plane (Postgres/TimescaleDB/Redis/NATS/Mailpit) |
| **tilt** | current | builds + deploys services into k3d with live-reload, holds port-forwards |

### Quality & secrets tooling (required for commits / CI parity)

| Tool | Version | Purpose |
|---|---|---|
| **golangci-lint** | v2.12.2 | Go linter (hard CI gate; matches `.golangci.yml`) |
| **gitleaks** | 8.18.4 | secret scanning (pre-commit + CI) |
| **sops** | 3.9.0 | decrypt/apply SOPS-sealed secrets |
| **age** | apt | encryption backend for sops |
| **pre-commit** | current | runs the hooks in `.pre-commit-config.yaml` |

### Optional

| Tool | When you need it |
|---|---|
| **NVIDIA Container Toolkit** | only for **real** local GPU inference (`make up GPU=1`). Install with `bash scripts/install-toolchain.sh --gpu` (needs an NVIDIA GPU + WSL2 GPU passthrough) |
| **Python 3.11+ + vLLM** | only to run the real (GPU) inference runtime instead of the CPU stub — `services/inference-runtime/requirements.txt` (`vllm`, `fastapi`, `uvicorn`). Not needed for the default local flow |

### Application dependencies (pulled automatically)

- **Go modules** — fetched on first `make build` / `make up` (per-service `go.mod`, all `go 1.25.0`).
- **Node packages** — installed with `npm install` in `1Trade Frontend/`. Key deps: `nuxt ^4`, `vue`,
  `@vueuse/core`, `chart.js`, `lightweight-charts`, `lucide-vue-next`, `docx`, `exceljs`, `jspdf`,
  `pptxgenjs`, `driver.js`.
- **Data plane** (Postgres · TimescaleDB · Redis · NATS · Mailpit) — installed into the cluster by
  `make up`; you don't install these on the host.

### Install manually (if you skip the script)

<details>
<summary>One-liners per tool</summary>

```bash
# Go (1.25.x)         → https://go.dev/dl/
# Node 20 + npm       → https://nodejs.org (or nvm: nvm install 20)
# kubectl             → curl -sSLo kubectl https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl
# k3d                 → curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
# helm                → curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
# tilt                → curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
# golangci-lint v2.12.2, gitleaks 8.18.4, sops 3.9.0, age, pre-commit
```
Prefer the script — it pins the exact versions CI expects.
</details>

After install, open a new shell (or `source ~/.profile`) so `go` and `$HOME/go/bin` are on `PATH`.

---

## 3. Bring the platform up

From the repo root, inside WSL:

```bash
make up
```

`make up` = **cluster → data-plane → tilt**:

1. **cluster** — creates (or starts) the k3d cluster `1trade` from `deploy/k8s/local/k3d.yaml`.
2. **data-plane** — applies Postgres · TimescaleDB · Redis · NATS · Mailpit into namespace `data`.
3. **tilt** — `tilt up`: generates the `platform-auth` secret, applies DB migrations, builds + deploys
   the services, and **holds the port-forwards open**. Edit source → Tilt live-reloads the pod.

Sanity check once pods are `Running`:

```bash
kubectl get pods -A          # or: make ps
curl -s localhost:8001/healthz   # → 200
```

### Ports Tilt exposes on localhost

| Port | Service |
|---|---|
| `:8001` | platform-core — auth · signup/login → JWT · billing · orgs/RBAC |
| `:8002` | credit-ledger — balances · convert · hash-chained debits |
| `:8085` | inference-gateway — OpenAI-compatible API · key auth · metering |
| `:8000` | inference-runtime — CPU stub (canned completions, real token counts) |
| `:8086` | compute-control — GPU control plane · catalog · mock-GPU scheduler |
| `:8025` | Mailpit web UI — captured email (`kubectl port-forward -n data deploy/mailpit 8025:8025`) |
| `:3000` | Nuxt web console (started separately, §4) |

### Optional tiers

| Command | Adds |
|---|---|
| `make up GPU=1` | NVIDIA GPU Operator — schedule a real host GPU (needs the NVIDIA Container Toolkit) |
| `make up SCHED=1` | Kueue + Volcano + mock-GPU queues |
| `make up OBS=1` | Prometheus + Loki + Tempo + Grafana (heavy) |
| `make up FULL=1` | SCHED + OBS together |

Default is **mock-GPU mode** — virtual GPUs, no driver, identical behavior on any machine.

---

## 4. Run the web console

The web app is **live-only** (no mock mode) — it needs the stack from §3 up and the Tilt port-forwards.

```bash
cd "1Trade Frontend"
npm install        # first time only
npm run dev        # → http://localhost:3000
```

No env needed — the BFF defaults its upstreams to `localhost:8001/8085/8002`. Override only if you
forwarded to different ports:

```bash
PLATFORM_CORE_URL=http://localhost:8001 \
CREDIT_LEDGER_URL=http://localhost:8002 \
INFERENCE_GATEWAY_URL=http://localhost:8085 \
npm run dev
```

---

## 5. Build & use the CLI

```bash
make cli                                                    # → bin/1trade
./bin/1trade signup --email you@dev.test --password 'pw'  # or: login
./bin/1trade whoami
./bin/1trade catalog
./bin/1trade credits balance
```

The CLI defaults to `localhost:8001/8085/8002`; override with `TRADE1_PLATFORM_URL` /
`TRADE1_GATEWAY_URL` / `TRADE1_LEDGER_URL`. The token is saved to `~/.1trade/config.json` (0600).

### Get credits on a fresh account (dev)

There's no `make seed` wired yet, so mint credits with the ledger **service token** (read from the
cluster secret):

```bash
SVC=$(kubectl get secret platform-auth -o jsonpath='{.data.SERVICE_TOKEN}' | base64 -d)
TENANT=…   # your tenant_id from `1trade whoami`

curl -s -X POST localhost:8002/v1/credits/purchase \
  -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: seed-$TENANT" \
  -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"text\",\"amount\":\"50\",\"is_paper\":true}"
```

### Drive the end-to-end loop

```bash
./bin/1trade infer chat -m llama-3.1-8b "Say hi in three words"   # stub reply + token usage
./bin/1trade credits balance                                      # text drops ~1s later (async debit)
./bin/1trade credits convert --from ai_index --to text --amount 100
```

The same loop runs in the browser at `:3000`: buy/convert in the wallet → inference playground → live
balances update.

---

## 6. Everyday commands

| Command | Does |
|---|---|
| `make bootstrap` | OS-aware install of every dependency, then `make up` (`GPU=1` adds NVIDIA toolkit) |
| `make setup` | OS-aware install of every dependency (no bring-up) |
| `make up` | cluster + data plane + Tilt (one-command bring-up) |
| `make down` | delete the local k3d cluster |
| `make ps` | `kubectl get pods -A` |
| `make cli` | build the `1trade` CLI → `bin/1trade` |
| `make build` / `make test` | build / race-test every Go module |
| `make lint` / `make fmt` | golangci-lint / gofmt across modules |
| `make test-e2e` | timed sub-5-min signup→top-up→infer→GPU-debit loop |
| `make web` | run the Nuxt dev server (`:3000`) |
| `pre-commit install` | wire the commit hooks (run once after cloning) |

---

## 7. Troubleshooting

| Symptom | Fix |
|---|---|
| `docker` not found / `make up` hangs | start Docker Desktop and enable WSL integration; run from the WSL shell |
| Web shows empty data / 502 | the stack or forwards are down — `tilt up`, or re-run the `kubectl port-forward`s |
| `port-forward` "address already in use" | that port is already forwarded — leave it |
| `make up` → `connection refused` | cluster is stopped — `make up` restarts it, or `k3d cluster start 1trade` |
| k3d LB `Bind for 0.0.0.0:8080 failed` | another project holds `:8080` — free it, then start; if wedged: `k3d cluster delete 1trade && make up` (dev data is disposable) |
| Inference returns `402` | tenant has no `text` credits — mint via §5 or `credits convert` into `text` |
| Balance didn't change after `infer` | debits are async (~1s via NATS) — re-check a moment later |
| `go` / `golangci-lint` not found | open a new shell or `source ~/.profile` (PATH picks up `$HOME/go/bin`) |

More detail: [`docs/plans/RUN_LOCAL.md`](docs/plans/RUN_LOCAL.md) · [`DEPLOY.md`](DEPLOY.md).
