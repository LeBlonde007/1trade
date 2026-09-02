# Run 1Trade locally — what's built so far

**This is the accurate, current-state runbook** (verified against the `Tiltfile`, `Makefile`, and the
live services). `LOCAL_DEV.md` describes the *intended* end-state (ingress `:8080`, Grafana, `make
seed`, `1trade login --dev`); several of those aren't wired yet. When in doubt, follow this file.

Last verified: 2026-05-31 (tags up to `v0.2.8`). To **verify the shipped features** once it's up, see
**[TESTING.md](./TESTING.md)** (automated tests + curl + CLI + web, per feature).

---

## What you can actually run today

**Built & live:**

- `platform-core` (auth · signup/login → JWT · billing · orgs/RBAC) — `:8001`
- `credit-ledger` (balances · convert · hash-chained debits) — `:8002`
- `inference-gateway` (OpenAI-compatible API · API-key auth · metering) — `:8085`
- `inference-runtime` (**CPU stub** — canned completions with real token counts; no GPU) — `:8000`
- the **Nuxt web console** (`1Trade Frontend/`) — `:3000`
- the **`1trade` CLI** (`apps/cli/`) → `bin/1trade`

**Not wired yet (don't follow `LOCAL_DEV.md` for these):** the `:8080` Traefik ingress, Grafana/
Prometheus dashboards, `make seed`, `1trade login --dev`, and the compute/exchange layers. You
reach services through the **Tilt port-forwards** and create accounts/credits the real way
(signup + mint).

---

## Prerequisites

**docker · k3d · kubectl · helm · tilt · go 1.25 · node/npm.** From scratch:
`bash scripts/install-toolchain.sh`.

---

## 1. Bring up the platform (one command)

```bash
make up      # creates the k3d cluster + data plane (Postgres/NATS via Helm) + `tilt up`
```

`tilt up` is what makes it usable: it generates the `platform-auth` secret (`PLATFORM_JWT_SECRET` +
`SERVICE_TOKEN`), applies the DB migrations, builds + deploys the services, and **holds the
port-forwards for you**:

| Service | localhost | purpose |
|---|---|---|
| platform-core    | `:8001` | auth, billing, orgs |
| credit-ledger    | `:8002` | balances, convert, debits |
| inference-gateway| `:8085` | OpenAI-compatible API |
| inference-runtime| `:8000` | CPU stub |

> **If the cluster is already up but Tilt isn't running** (e.g. after a manual deploy), either run
> `tilt up` to reconcile + restore the forwards, or forward manually:
>
> ```bash
> kubectl port-forward -n default deploy/platform-core    8001:8001 &
> kubectl port-forward -n default deploy/credit-ledger    8002:8002 &
> kubectl port-forward -n default deploy/inference-gateway 8085:8085 &
> ```
>
> "address already in use" just means that port is already forwarded — leave it.

Sanity check: `kubectl get pods -A` (expect `data/{postgres,nats}` + the four services `Running`),
then `curl -s localhost:8001/healthz` etc. should return `200`.

---

## 2. Run the web console

The web app is **always live** (there is no mock mode) — it needs the stack up + the forwards from §1.

```bash
cd "1Trade Frontend"
npm run dev        # → http://localhost:3000
```

No env needed — the BFF's upstream URLs default to `localhost:8001/8085/8002` (the Tilt forwards).
Override only if you forwarded to different ports:

```bash
PLATFORM_CORE_URL=http://localhost:8001 \
CREDIT_LEDGER_URL=http://localhost:8002 \
INFERENCE_GATEWAY_URL=http://localhost:8085 \
npm run dev
```

---

## 3. Build & run the CLI

```bash
make cli                                              # → bin/1trade
./bin/1trade signup --email you@dev.test --password 'pw'   # or: login --email … --password …
./bin/1trade whoami
./bin/1trade catalog
./bin/1trade credits balance
```

The CLI defaults to `localhost:8001/8085/8002`; override with `TRADE1_PLATFORM_URL` /
`TRADE1_GATEWAY_URL` / `TRADE1_LEDGER_URL`. The token is saved to `~/.1trade/config.json`
(0600). Password can come from `TRADE1_PASSWORD` or the hidden prompt instead of `--password`.

---

## 4. Get credits on a fresh account (dev)

There's no `make seed` yet, so grant credits via the ledger **service token** (read from the cluster
secret) — this is the reliable dev path:

```bash
SVC=$(kubectl get secret platform-auth -o jsonpath='{.data.SERVICE_TOKEN}' | base64 -d)
TENANT=…   # your tenant_id from `1trade whoami` / GET :8001/v1/auth/me

curl -s -X POST localhost:8002/v1/credits/purchase \
  -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: seed-$TENANT" \
  -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"text\",\"amount\":\"50\",\"is_paper\":true}"
```

Mint `ai_index` the same way (`"credit_type":"ai_index"`) if you want to exercise conversion.

---

## 5. The loop you can drive end-to-end

```bash
./bin/1trade infer chat -m llama-3.1-8b "Say hi in three words"   # stub reply + token usage
./bin/1trade credits balance                                      # text drops ~1s later (debit)
./bin/1trade credits convert --from ai_index --to text --amount 100
```

…and the same loop in the browser at `:3000`:
**buy/convert in the wallet → run the inference playground (shows real credit cost + a live session
meter) → wallet balances + recent-movements update live.**

---

## Things to know

- **No GPU →** `inference-runtime` is a CPU stub returning canned completions with real token counts
  — enough to drive the entire credit loop. Real vLLM needs `make up GPU=1` + a fitting GPU
  (Llama-3.1-8B fits most dev cards; larger models route to the stub).
- **Debits are async** (NATS `inference.usage.v1` → ledger): balance updates ~1s after a run, not
  instantly. (The consumer that makes this flow was fixed in `v0.2.7`.)
- **Buying credits via the UI** uses *mock* Stripe; the mint only completes when the webhook fires,
  so for dev prefer the service-token mint in §4.
- **Secrets** are plain Kubernetes Secrets locally (`platform-auth`) — no Vault in dev.

---

## Ports summary

| localhost | what |
|---|---|
| `:3000` | Nuxt web console (`npm run dev`) |
| `:8001` | platform-core (auth/billing/orgs) |
| `:8002` | credit-ledger (credits/convert/debits) |
| `:8085` | inference-gateway (OpenAI API) |
| `:8000` | inference-runtime (CPU stub) |
| `:8025` | Mailpit web UI — captured emails (`kubectl port-forward -n data deploy/mailpit 8025:8025`) |

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Web shows empty data | the stack/forwards aren't up | start the stack + forwards (§1); the app is live-only |
| Web "upstream error" / 502 | a forward is down | `tilt up`, or re-run the `kubectl port-forward`s (§1) |
| `port-forward` "address already in use" | already forwarded | leave it — that port is up |
| Inference returns 402 | tenant has no `text` credits | mint via §4, or `credits convert` into `text` |
| Balance didn't change after `infer` | debit is async (~1s) | re-check `credits balance` a moment later |
| `curl :8001/healthz` fails | services not up / no forward | `kubectl get pods -A`; `tilt up` |
| `make up` → `connection refused` on `…:43407` | cluster is **stopped** (was: Makefile skipped starting it — fixed) | `make up` now starts it; or `k3d cluster start 1trade` |
| `k3d cluster start` → serverlb `Bind for 0.0.0.0:8080 failed: port is already allocated` | another local project holds `:8080` (k3d's LB also fronts the kube API) | free `:8080` (`docker ps`, stop the holder) **then** start; if the LB got wedged from a failed start, `k3d cluster delete 1trade && make up` (dev data is disposable; images are cached so it's quick) |
