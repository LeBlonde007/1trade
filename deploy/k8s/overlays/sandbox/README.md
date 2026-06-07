# env=sandbox — deploy the paper-mode product to a VPS

A **single-node k3s** deployment of the whole Exascale product in **sandbox / paper mode**: no GPU, no
real Stripe, no KYC vendor, no real money. Everything runs on its mock/stub path — `MockStripe` settles
purchases inline, the **CPU stub** serves inference, the **mock-GPU** scheduler runs instance
lifecycle, and KYC auto-approves. It's the full live journey (signup → buy credits → run inference →
ledger debit → wallet/billing/audit → CLI), hosted.

## What you need

- A VPS — **4 vCPU / 8 GB / 80 GB SSD, Ubuntu 22.04/24.04** is comfortable (2 vCPU / 4 GB works if you
  build images elsewhere). Open ports **80/443**.
- **Docker** + **kubectl** on the box. The deploy script can install **k3s** for you (`INSTALL_K3S=1`).
- (For HTTPS) a **domain** with an A record at the VPS IP, and an email for Let's Encrypt.
- Nothing else — no Stripe/HF/GPU/NVIDIA accounts (those are the *prod* list in `PROVISIONING.md`).

## Deploy

From the repo root on the VPS (or anywhere with `KUBECONFIG` pointing at it):

```bash
# HTTP, install k3s, serve at the node IP:
INSTALL_K3S=1 SANDBOX_HOST=<vps-ip> scripts/deploy-sandbox.sh

# HTTPS on a real domain:
INSTALL_K3S=1 TLS=1 ACME_EMAIL=you@example.com SANDBOX_HOST=sandbox.example.com scripts/deploy-sandbox.sh
```

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
(`deploy/k8s/web/base`), the public **Ingress** (web is the only public surface — the BFF proxies to
the Go services in-cluster), retags images to **`:sandbox`**, and points **`APP_BASE_URL`** at the
public URL. `EXASCALE_SANDBOX_HOST` / `EXASCALE_SANDBOX_URL` are placeholders the script substitutes.

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
