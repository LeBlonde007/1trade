# Provisioning — external accounts & services to create

**As of 2026‑05‑30.** The third‑party accounts/services a human must create for the platform to run
in production. The code already expects these. Internal secrets (`PLATFORM_JWT_SECRET`,
`SERVICE_TOKEN`) are auto‑generated into the `platform-auth` k8s Secret and are **not** in this list.
Pair with [PLATFORM_GAPS.md](PLATFORM_GAPS.md).

**Legend:** 🔴 needed now (M2 go‑live) · 🟠 needed soon (M3) · 🟡 later (M4–M6) · 🟢 already have.

---

## 🔴 Needed NOW — take real money + serve real inference (M2)

| Account / service | Gives you (env / config) | What it's for |
|---|---|---|
| **Stripe** | `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET` | Credit‑card purchases (F06). Wired today on a mock. |
| **Domain + Cloudflare** | DNS, edge, WAF | `api.exascale.io` + `app.exascale.io` are already in the code (OpenAPI servers, buy‑credits link). |
| **TLS certs** | — | `https://` (via Cloudflare or Let's Encrypt). |
| **Container registry** | image host | GHCR / Docker Hub / cloud. GPU manifest refs `registry.exascale.io/inference-runtime:vllm`. |
| **Managed K8s or bare‑metal k3s** | `KUBECONFIG` | Real cluster (local is k3d only). |
| **GPU node(s)** | a 40 GB+ GPU | F09 to actually serve the M2 models (Lambda/CoreWeave/RunPod or owned). |
| **Hugging Face + Llama license** | `HF_TOKEN` | Pull gated Llama 3.1 weights to the PVC (Whisper is open; accept Meta's license). |
| **NVIDIA NGC** (free) | NGC API key | `Dockerfile.vllm` base `nvcr.io/nvidia/pytorch`. *(Or switch base to `vllm/vllm-openai` on Docker Hub → skip NGC.)* |
| **Transactional email** | `SMTP_*` / provider API key | Email verification + receipts (SES / Postmark / SendGrid). The "Verify" gap. |
| **Managed Postgres (+ Redis/NATS)** *(optional)* | `DATABASE_URL` | Or keep in‑cluster (already manifested); managed is safer for prod data. |

**Critical‑path minimum to be "live and taking money":** Stripe · domain/Cloudflare · registry · one
GPU node · HF token (Llama license) · email provider.

---

## 🟠 Needed SOON — M3 (real customers, enterprise, scale)

| Account / service | Gives you | What it's for |
|---|---|---|
| **KYC/AML provider** (Stripe Identity / Persona) | verification API | Required before real‑money purchases (F22 gate). |
| **OAuth apps** (Google / GitHub / Microsoft) | client id + secret each | SSO login (F02 OAuth; stubbed today). |
| **Secrets manager** (Vault or cloud KMS + SOPS) | encrypted secret store | See "Why" below. |
| **Observability** (Grafana Cloud or self‑host) | metrics/logs/traces/alerts | See "Why" below. |
| **ACH/wire banking** + Stripe ACH | bank account | $10K+ enterprise purchases (F06‑M3). |

---

## 🟡 LATER — M4–M6 (supply, compliance, exchange)

| Account / service | What it's for |
|---|---|
| **Vanta** | SOC 2 Type I automation (F21). |
| **SAML IdP** (Okta / Entra) | Enterprise SSO (F02‑M4). |
| **Stripe Connect or banking rails** | Pay datacenter partners (F18 settlement/payouts). |
| **Legal/licensing counsel + entity** | Money‑transmitter/exchange licensing (F22) before the trading layer switches on. |
| **NVIDIA cert + DCGM / attestation** | Partner‑DC GPU onboarding (F17/F19). |

---

## 🟢 Already have / auto‑handled
- ✅ **GitHub** (`saadallahdev/ex-main`) — code + branches + tags pushed.
- ✅ **SSH key** — push auth works.
- ✅ **PCI** — handled by Stripe; we never touch card data.
- ✅ **JWT / service secrets** — generated into the `platform-auth` k8s Secret (dev). Prod moves them
  into the secrets manager below.

---

## Why the two prod‑hardening items matter (M3, not day‑1)

### 🔐 Secrets manager (Vault / cloud KMS + SOPS)
Today the real secrets (`PLATFORM_JWT_SECRET`, `SERVICE_TOKEN`, `STRIPE_SECRET_KEY`,
`STRIPE_WEBHOOK_SECRET`, DB password, `HF_TOKEN`) live in a plain k8s `Secret` — **base64, not
encrypted**. Needed for:
- **Protection of the crown jewels** — the JWT secret authenticates *every tenant fleet‑wide*; the
  Stripe keys move real money; the DB URL guards customer data.
- **GitOps** — SOPS encrypts secrets so the encrypted blob can be committed; only the cluster/KMS
  decrypts.
- **Rotation** — rotate the JWT/Stripe/DB keys on schedule or after a leak, without hand‑editing.
- **Access control + audit** — who/what can read which secret, and a log of access.
- **SOC 2** — a control requirement.
> Without it, one `kubectl` slip or an etcd backup leaks auth + payment keys for the whole platform.

### 📈 Observability (Grafana + Prometheus + Loki + Tempo)
Today it's `kubectl logs` one pod. Needed for:
- **Metrics** — request/error rates, the `latency_ms` already emitted, GPU utilization, NATS
  consumer lag, ledger throughput (runtime already exposes `/metrics`).
- **Alerting on the failures that cost money** — the `inference.usage.v1` consumer falling behind
  (**inference served but not billed**), the Stripe webhook failing (**money taken, credits not
  granted**), P95 latency past SLA, a GPU node down.
- **Logs (Loki)** — the structured `slog` + `audit:` trail, searchable and retained across restarts.
- **Traces (Tempo)** — follow one request gateway → runtime → ledger to find where it broke.
- **Dashboards (Grafana)** — operator view + the public `status` page.
> Without it you're blind in prod — you learn about a billing gap or outage from a customer, not a page.

Already tracked as backlog tasks: observability dashboards/SLO alerts, and Prometheus `/metrics` on
every service.
