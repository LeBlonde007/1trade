# env=sandbox — deploy the paper-mode product to a VPS

A **single-node k3s** deployment of the whole 1Trade product in **sandbox / paper mode**: no GPU, no
real Stripe, no KYC vendor, no real money. Everything runs on its mock/stub path — `MockStripe` settles
purchases inline, the **CPU stub** serves inference, the **mock-GPU** scheduler runs instance
lifecycle, and KYC auto-approves. It's the full live journey (signup → buy credits → run inference →
ledger debit → wallet/billing/audit → CLI), hosted.

## What you need

- A VPS — **4 vCPU / 8 GB / 80 GB SSD, Ubuntu 22.04/24.04** is comfortable (2 vCPU / 4 GB works in the
  **no-source** path below, since the box doesn't build images). Open ports **80/443**.
- **kubectl** on the box (+ **Docker** only if you build on the box). The deploy script can install
  **k3s** for you (`INSTALL_K3S=1`).
- (For HTTPS) **two A records** — `SANDBOX_HOST` (web) and `SANDBOX_API_HOST` (the `/v1` API) — both at
  the VPS IP, and an email for Let's Encrypt. One SAN cert covers both.
- Nothing else — no Stripe/HF/GPU/NVIDIA accounts (those are the *prod* list in `PROVISIONING.md`).

## Deploy

From the repo root on the VPS (or anywhere with `KUBECONFIG` pointing at it):

```bash
# HTTP, install k3s, serve at the node IP (api host defaults to api.<host>):
INSTALL_K3S=1 SANDBOX_HOST=<vps-ip> scripts/deploy-sandbox.sh

# HTTPS on real domains (web + api):
INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
  SANDBOX_HOST=sandbox.example.com SANDBOX_API_HOST=sandboxapi.example.com \
  scripts/deploy-sandbox.sh
```

> **Deploying to a box you don't own (a client's droplet)?** Don't run the above — it builds source on
> the box. Use the **No-source deploy** path below (prebuilt images + a manifests-only bundle).

The script (idempotent — safe to re-run):
1. installs k3s (optional) — single node, ships Traefik as the ingress controller;
2. builds the six images (`platform-core`, `credit-ledger`, `inference-gateway`, `compute-control`,
   `inference-runtime-stub`, `web`) and imports them into k3s containerd;
3. creates the `platform-auth` Secret (fresh random JWT + service token) and the migration ConfigMaps;
4. applies the data plane (Postgres · TimescaleDB · Redis · NATS · Mailpit, namespace `data`);
5. (TLS=1) installs cert-manager + a Let's Encrypt `ClusterIssuer`;
6. renders this overlay (substituting the host/URL) and applies the services + web + ingress;
7. waits for every rollout and prints the URL.

Then point DNS at the node and open `http(s)://$SANDBOX_HOST`.

## What this overlay does (vs. the bases)

The service **bases are already paper-mode**, so the overlay is thin: it adds the **web** frontend
(`deploy/k8s/web/base`), a single public **Ingress**, retags images to **`:sandbox`**, points
**`APP_BASE_URL`** at the public URL, and **hardens `TRADE1_ENV=sandbox`** (see security note below).
`TRADE1_SANDBOX_HOST` / `TRADE1_SANDBOX_URL` are placeholders the script substitutes.

## Two hosts: web + API (for the `1trade` CLI)

The sandbox uses **two subdomains**, both A-records at the node IP:

- **`SANDBOX_HOST`** (e.g. `sandbox.1trade.ai`) — the **web app** (browser). Its Nuxt BFF (`/api/*`)
  talks to the services in-cluster, so the browser only needs this host.
- **`SANDBOX_API_HOST`** (e.g. `sandboxapi.1trade.ai`) — the **JSON API** for the `1trade` CLI +
  external clients: `/v1/*` routed by path prefix to the Go services (`/v1/auth`,`/v1/account`,
  `/v1/billing` → platform-core · `/v1/credits` → credit-ledger · `/v1/models`,`/v1/chat` →
  inference-gateway · `/v1/compute` → compute-control). It also serves the web app at `/` so a browser
  hitting it gets a page, not a 404.

Point the CLI at the API host (one env var sets all four service URLs):

```bash
TRADE1_API_URL=https://sandboxapi.1trade.ai 1trade login     # then: balance · catalog · infer · gpu
```

### Security: going public turns off the dev auth shortcut

The credit-ledger trusts an `X-Dev-Tenant` header as an auth shortcut **only when `TRADE1_ENV=dev`**
(it lets you mint credits / act as any tenant with no token — fine on a laptop, fatal if public). The
overlay therefore sets **`TRADE1_ENV=sandbox`** on all four services so the server **never** trusts
that header, and a Traefik **`strip-dev-headers`** middleware deletes `X-Dev-Tenant`/`X-Dev-Paper` at
the edge as defence-in-depth. Credit creation (`/v1/credits/mint|burn|debit|purchase`) needs the
internal `SERVICE_TOKEN`, which is never public — those endpoints 401 from the internet. **Consequence:**
the local `seed.sh`/`verify-inference.sh` X-Dev-Tenant minting does **not** work against the sandbox; on
the sandbox you get credits the real way — sign up → **buy credits** (MockStripe settles instantly).

## No-source deploy (a box you don't control — e.g. a client's droplet)

The default deploy **builds the images on the box**, so it needs the full source there. On a machine you
don't own that hands your unpaid work to whoever controls the disk. Avoid it: deploy from **prebuilt
images** instead, and ship only manifests + SQL — never `.go`/`.vue`.

1. **Publish images once** (from your machine or CI). The `release` GitHub Action pushes
   `ghcr.io/<owner>/1trade-<svc>:sandbox` on every push to `main`. Make those packages **public**
   (simplest), or keep them private and mint a `read:packages` token for the box.
2. **Make the bundle** (no source): `scripts/make-sandbox-bundle.sh` → `sandbox-bundle.tar.gz`
   (k8s manifests + migration SQL + deploy scripts only; it refuses to ship if any source slips in).
3. **On the box** — extract and deploy in pull-mode (`IMAGE_REGISTRY` set ⇒ no build, no source):

   ```bash
   tar xzf sandbox-bundle.tar.gz
   IMAGE_REGISTRY=ghcr.io/<owner> \
     REGISTRY_USER=<owner> REGISTRY_TOKEN=<read:packages PAT> \   # omit both if packages are public
     INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
     SANDBOX_HOST=sandbox.example.com SANDBOX_API_HOST=sandboxapi.example.com \
     scripts/deploy-sandbox.sh
   ```

   `REGISTRY_TOKEN` becomes an `imagePullSecret` on the namespace's default ServiceAccount. The cluster
   pulls the compiled images; the manifests/SQL are config, not the product. (Still: anyone with the box
   can copy a running image — the web bundle is minified JS, recoverable-ish; pre-payment, hosting on
   **your** box behind a URL remains the only true protection.)

## Real inference output (optional — hosted provider)

By default the sandbox serves inference from the **CPU stub** (echo-style output — fine for proving the
plumbing, the ledger debit, and the UI, but not real model answers). To get **real model output** for a
demo without a GPU, point the gateway at a hosted **OpenAI-compatible** provider (OpenRouter by default)
— just pass a key at deploy:

```bash
INFERENCE_API_KEY=sk-or-... INSTALL_K3S=1 SANDBOX_HOST=<vps-ip> scripts/deploy-sandbox.sh
```

The script injects the key into the `platform-auth` Secret (never committed), flips
`INFERENCE_BACKEND=vllm` at `VLLM_BASE_URL=https://openrouter.ai/api/v1`, and sets a default
`INFERENCE_MODEL_MAP` translating our catalog ids to provider slugs
(`llama-3.1-70b`/`llama-3.1-8b` → `meta-llama/llama-3.1-…-instruct`). Override `VLLM_BASE_URL`
(e.g. a Groq endpoint) or `INFERENCE_MODEL_MAP` to use a different provider/model set. Nothing else
changes — the customer API, metering, and ledger debit are identical; only the upstream that produces
tokens differs. Leave `INFERENCE_API_KEY` unset to keep the keyless CPU stub.

> **Key hygiene:** the provider key lives only in the cluster Secret. Rotate it from the provider
> dashboard if it ever leaks; re-run the deploy with the new key to roll it.

## Notes & limits

- **Paper mode only.** Real GPU serving, real Stripe, real KYC, and the (paused) exchange are *not*
  here — see `PROVISIONING.md` for the prod list.
- **Email** (verification, receipts) is captured by **Mailpit** in-cluster, not delivered externally.
  Verification only gates *real-money* purchases, so it isn't needed to use the sandbox; to read it:
  `kubectl -n data port-forward svc/mailpit 8025:8025`.
- **Persistence** uses k3s local-path (Postgres data survives pod restarts, not a node rebuild). Add a
  managed Postgres or a volume backup before you treat sandbox data as durable.
- **Secrets:** the script generates a random `platform-auth`. For a committed/rotatable secret, use the
  SOPS flow in `deploy/secrets/README.md` (`make secrets-apply`) instead.
- **Scale:** one replica each. It's a demo/validation tier, not a load target.
