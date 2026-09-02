# Deploy 1Trade (sandbox) to a VPS — step by step

A start-to-finish runbook to host the **paper-mode** product on one Linux VPS. Sandbox = **CPU-only**:
no GPU, no real Stripe, no KYC vendor, no real money — every external dependency runs on its mock/stub
path. You get the full live journey: **sign up → buy credits (MockStripe settles instantly) → run
inference (CPU stub) → ledger debit → wallet / billing / audit → CLI**.

> Two paths through this doc: **A) one command** (`scripts/deploy-sandbox.sh` does everything) or
> **B) manual** (run each piece yourself). Most people want A. Total time ~20–30 min.

---

## 0. What you need

- A **VPS**: **4 vCPU / 8 GB RAM / 80 GB SSD, Ubuntu 22.04 or 24.04**. (2 vCPU / 4 GB works if you build
  images elsewhere — see §7.) Hetzner CPX31 (~€14/mo), DigitalOcean, Linode, Vultr all fine.
- A way to SSH in (the provider gives you `root@<ip>` + a password or key).
- **Optional, for HTTPS:** a domain you control (to add a DNS record) + an email for Let's Encrypt.

Nothing else. No Stripe / Hugging Face / NVIDIA / GPU accounts — those are the **prod** list
(`docs/plans/PROVISIONING.md`), not sandbox.

---

## 1. Create the VPS

1. In your provider's console, create a server: **Ubuntu 24.04 LTS**, the 4 vCPU / 8 GB plan, a region
   near you.
2. Add your SSH key (recommended) or note the root password.
3. Note the server's **public IP** (e.g. `203.0.113.10`).

---

## 2. First login + base setup

SSH in and update:

```bash
ssh root@203.0.113.10            # your VPS IP
apt-get update && apt-get -y upgrade
apt-get -y install git curl ca-certificates openssl
```

Open the firewall for web + SSH (if your provider uses a cloud firewall, do it there too):

```bash
# ufw is the simplest host firewall:
apt-get -y install ufw
ufw allow 22/tcp        # SSH
ufw allow 80/tcp        # HTTP (and Let's Encrypt ACME)
ufw allow 443/tcp       # HTTPS
ufw --force enable
```

> Working as `root` is fine for a sandbox. For a non-root user: `adduser deploy && usermod -aG sudo
> deploy && usermod -aG docker deploy`, then continue as `deploy`.

---

## 3. Install Docker

```bash
curl -fsSL https://get.docker.com | sh
docker run --rm hello-world      # sanity check → "Hello from Docker!"
```

(k3s is installed by the deploy script in §6 — you don't install it here.)

---

## 4. Get the code

```bash
git clone https://github.com/saadallahdev/ex-main.git 1trade
cd 1trade
```

> Private repo? Use a deploy key or `git clone https://<token>@github.com/...`.

---

## 5. DNS (only if you want HTTPS / a domain)

Add **two A records** — one for the web app, one for the `/v1` API — both pointing at the VPS IP:

| Type | Name              | Value          | Serves                          |
|------|-------------------|----------------|---------------------------------|
| A    | `sandbox`         | `203.0.113.10` | web app (browser)               |
| A    | `sandboxapi`      | `203.0.113.10` | JSON API (`1trade` CLI/clients) |

→ gives you `sandbox.yourdomain.com` (web) and `sandboxapi.yourdomain.com` (api). Wait for **both** to
resolve (`dig +short sandbox.yourdomain.com`) — for TLS, Let's Encrypt validates each host. **Skip this**
if you'll just use the raw IP over HTTP (the api host then defaults to `api.<host>`).

---

## 6. Deploy (Path A — one command)

From the repo root on the VPS. Pick the line that matches you:

```bash
# (a) HTTP, served at the raw IP — quickest:
INSTALL_K3S=1 SANDBOX_HOST=203.0.113.10 scripts/deploy-sandbox.sh

# (b) HTTPS on your domains (needs §5 done + ports 80/443 open):
INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
  SANDBOX_HOST=sandbox.yourdomain.com SANDBOX_API_HOST=sandboxapi.yourdomain.com \
  scripts/deploy-sandbox.sh
```

> **Deploying to a box you don't own (e.g. a client's droplet)?** Both (a) and (b) build the source on
> the box. Don't — use the **no-source** path in §7b (prebuilt images + a manifests-only bundle) so no
> `.go`/`.vue` ever lands there.

The script is **idempotent** (safe to re-run) and does, in order:

1. installs **k3s** (single node; ships Traefik as the ingress controller);
2. builds the six images (`platform-core`, `credit-ledger`, `inference-gateway`, `compute-control`,
   `inference-runtime-stub`, `web`) and imports them into k3s;
3. creates the `platform-auth` Secret (random JWT + service token) + the migration ConfigMaps;
4. applies the **data plane** (Postgres · TimescaleDB · Redis · NATS · Mailpit);
5. (TLS=1) installs **cert-manager** + a Let's Encrypt issuer;
6. applies the **services + web + ingress** (the `env=sandbox` overlay);
7. waits for every rollout and prints the URL.

It finishes with `==> sandbox up → http(s)://<host>`. First run takes ~10–15 min (image builds).

---

## 7. Deploy (Path B — manual, or low-RAM box)

If you'd rather run the steps yourself, or your VPS is small and you build images on your laptop:

```bash
# On the VPS — install k3s, then point kubectl at it:
curl -sfL https://get.k3s.io | sh -
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# Build + import the six images (on the box):
for s in platform-core credit-ledger inference-gateway compute-control; do
  docker build -t 1trade/$s:sandbox services/$s
done
docker build -t 1trade/inference-runtime-stub:sandbox services/inference-runtime/stub
docker build -t 1trade/web:sandbox "1Trade Frontend"
for i in platform-core credit-ledger inference-gateway compute-control inference-runtime-stub web; do
  docker save 1trade/$i:sandbox | k3s ctr images import -
done

# Secret + migrations + data plane:
kubectl create secret generic platform-auth \
  --from-literal=PLATFORM_JWT_SECRET=$(openssl rand -base64 48) \
  --from-literal=SERVICE_TOKEN=$(openssl rand -base64 32)
kubectl create configmap platform-core-migrations --from-file=services/platform-core/migrations/
kubectl create configmap credit-ledger-migrations \
  --from-file=types.sql=docs/contracts/schemas/types.sql \
  --from-file=0001_init.sql=services/credit-ledger/migrations/0001_init.sql \
  --from-file=0002_conversion.sql=services/credit-ledger/migrations/0002_conversion.sql
kubectl apply -f deploy/k8s/local/data-plane.yaml

# Services + web + ingress (substitute your web host, api host, and url):
HOST=sandbox.yourdomain.com; API_HOST=sandboxapi.yourdomain.com; URL=https://$HOST
kubectl kustomize deploy/k8s/overlays/sandbox \
  | sed -e "s|TRADE1_SANDBOX_API_HOST|$API_HOST|g" \
        -e "s|TRADE1_SANDBOX_HOST|$HOST|g" -e "s|TRADE1_SANDBOX_URL|$URL|g" \
  | kubectl apply -f -
```

> **Build on your laptop instead** (for a 4 GB box): build the images locally, `docker save` each to a
> tar, `scp` to the VPS, and `k3s ctr images import < file.tar`. Or push to GHCR and set the overlay
> images to your registry.

---

## 7b. No-source deploy — pull prebuilt images (skip on-box builds)

**Use this for a box you don't fully control (a client's droplet), or any 2 GB VPS.** It never builds on
the box, so no `.go`/`.vue` source ever lands there — the compiled product arrives as images.

The `release` GitHub Actions workflow (`.github/workflows/release.yml`) builds + pushes all six images to
**`ghcr.io/<owner>/1trade-<service>`** on every push to `main` (tag `:sandbox`) and every `vX.Y.Z` tag.
Make those packages **Public** (repo → Packages → each → Settings) — or keep them private and mint a
`read:packages` token. Then:

```bash
# 1) On your machine (full checkout): build the manifests-only bundle — no source goes in it.
make sandbox-bundle                                  # → sandbox-bundle.tar.gz
scp sandbox-bundle.tar.gz root@203.0.113.10:/root/

# 2) On the box: extract + deploy in pull-mode. IMAGE_REGISTRY set ⇒ no build, no source.
tar xzf sandbox-bundle.tar.gz
IMAGE_REGISTRY=ghcr.io/<owner> \
  REGISTRY_USER=<owner> REGISTRY_TOKEN=<read:packages PAT> \   # omit both if the packages are public
  INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com \
  SANDBOX_HOST=sandbox.yourdomain.com SANDBOX_API_HOST=sandboxapi.yourdomain.com \
  scripts/deploy-sandbox.sh
```

`deploy-sandbox.sh` then skips the build, rewrites the overlay's `1trade/<svc>` image names to
`ghcr.io/<owner>/1trade-<svc>`, and (if a token is given) creates an `imagePullSecret` on the default
ServiceAccount so the cluster can pull. Re-run after the workflow publishes a newer `:sandbox` to update.

> **Honest limit:** whoever controls the box can still copy a *running* image (the web bundle is minified
> JS, recoverable-ish; Go binaries less so). No-source removes the easy `cat *.go` leak, not physical
> access. Pre-payment, hosting on **your** infra and giving the client only a URL is the real protection.

---

## 8. Verify it's up

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get pods           # all Running/Ready (default ns)
kubectl get pods -n data   # postgres/redis/nats/etc Running
curl -I http://<host>/     # 200 from the web app
```

Then open **`http(s)://<host>`** in a browser and run the smoke flow:

1. **Sign up** (pick "AI Company") → lands on the console.
2. **Buy credits** → MockStripe settles instantly, balance jumps.
3. **Run inference** in the playground → response + a real per-token **debit** on the ledger.
4. Check **Wallet / Billing / Audit** → the movement is recorded.

(Verification emails go to Mailpit, not your inbox — but verification only gates *real-money*, so you
don't need it in sandbox. To read captured mail: `kubectl -n data port-forward svc/mailpit 8025:8025`,
then open `http://localhost:8025` through an SSH tunnel.)

---

## 9. Operate

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

kubectl get pods                                   # status
kubectl logs deploy/platform-core --tail=100 -f    # logs (swap the service name)
kubectl rollout restart deploy/web                 # restart a service

# Update to new code:
git pull
scripts/deploy-sandbox.sh   # re-run with the same SANDBOX_HOST (+ TLS/ACME_EMAIL) — rebuilds & rolls

# Postgres backup (paper data, but still):
kubectl -n data exec deploy/postgres -- pg_dump -U 1trade 1trade | gzip > backup-$(date +%F).sql.gz
```

---

## 10. Troubleshooting

| Symptom | Fix |
|---|---|
| `kubectl: connection refused` | `export KUBECONFIG=/etc/rancher/k3s/k3s.yaml` (k3s writes it there). |
| A pod stuck `ImagePullBackOff` | the image wasn't imported into k3s — re-run the `docker save \| k3s ctr images import -` step (or the script). |
| `platform-core` initContainer crashloops | the `platform-core-migrations` ConfigMap is missing a file — recreate it `--from-file=services/platform-core/migrations/`. |
| HTTPS cert not issued | DNS must resolve to the VPS **before** TLS=1, and ports 80/443 open; check `kubectl describe certificate web-tls`. |
| Web build OOMs (small VPS) | build images on your laptop and `scp` them in (§7 note), or resize to 8 GB. |
| Can't reach the site | confirm the firewall (`ufw status`) + the provider's cloud firewall allow 80/443, and DNS points at the IP. |

---

## 11. What this is **not**

Paper/sandbox only. **Not** included (these need the prod path — see `docs/plans/PROVISIONING.md`):
real GPU inference, real Stripe payments, a real KYC/AML vendor, the (paused) trading exchange, managed
Postgres/HA, and full observability. Data lives on k3s local-path — fine for a demo, not durable across
a node rebuild.

For the design of the overlay itself, see `deploy/k8s/overlays/sandbox/README.md`.
