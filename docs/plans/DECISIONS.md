# Decision log (ADRs)

Short architecture-decision records. Each one is the *why* behind a choice the plan assumes, so
it isn't re-litigated. Add a new ADR (next number) when a significant, hard-to-reverse decision
is made; never edit a decided ADR's substance — supersede it with a new one.

Status values: **Accepted** · **Superseded by ADR-NNNN** · **Proposed** · **Open**.

---

## ADR-0001 — Deployment platform: Kubernetes (k3s/k3d), lightweight + progressively hardened

**Status:** Accepted (2026-05-29) · **Owner:** infra-sre + founder

**Context.** We need a deployment platform for ~9 Go services + a Nuxt frontend + a data plane,
*and* for GPU orchestration (multi-model packing/MIG, reserved-capacity pre-emption, multi-node
InfiniBand gang-scheduled clusters). Docker Desktop + WSL2 is the local environment. Dokploy
(a Docker-Swarm PaaS) was considered for its simplicity.

**Decision.** **Kubernetes is the single substrate.** `k3s` (single-binary, production-grade) in
prod; `k3d` (k3s-in-Docker) locally — so local == prod. Hardened progressively: Traefik→Kong (M3),
K8s Secrets+SOPS→Vault (M4), Helm/Kustomize→+Terraform (M3). GPU nodes run the NVIDIA GPU Operator
+ Kueue + Volcano.

**Why not Dokploy / Docker Swarm.** The compute layer (the founder's "most important part")
*requires* Kubernetes — Swarm cannot gang-schedule a 256-GPU job, do MIG partitioning, or run
Kueue/Volcano. So K8s is in the stack regardless. Adding Dokploy would mean maintaining two
deployment models and migrating the app plane onto K8s later anyway — more drift, not less.

**Why not full managed K8s + Terraform + Kong + Vault from day one.** That was the original
"locked" reading, but it front-loads heavy ops onto M1. k3s/k3d gives the same Kubernetes API at a
fraction of the setup, and the enterprise pieces are added exactly when a milestone needs them.

**Why not `kind` locally.** `kind` is a test harness, not a prod runtime. k3d runs the *same* k3s
that prod runs, so there's no behavioral gap between laptop and production.

**Consequences.**
- Local dev: `make up` → k3d + Helm data plane + `tilt up` live-reload (see `LOCAL_DEV.md`).
- Prod: k3s on owned-DC GPU servers + partner DCs; managed K8s optional for stateless nodes.
- GPU on WSL2: NVIDIA Container Toolkit; mock-GPU mode for machines without a GPU.
- `CLAUDE.md`'s locked stack (K8s + Kueue + Volcano + GPU Operator; Kong; Vault) stays valid —
  k3s/k3d is an *implementation* of that stack, and Kong/Vault are deferred, not dropped.
- Affected docs: `LOCAL_DEV.md` §2–§3, `DEPLOYMENT.md` §0, `agents/infra-sre.md` §3.

---

## ADR-0002 — AI ↔ sub-credit conversion direction *(Provisional — applied, pending counsel)*

**Status:** Provisional (default applied 2026-05-29) · **Owner:** tech-lead + counsel (security-compliance)

**Context.** `credit-types.md` and `F07` need to fix whether AI credits convert to/from sub-credits
**bidirectionally with a house spread** or **one-way** (sub→AI or AI→sub only). This also touches
the regulatory framing (redeemable-only must stay clear of "tradeable").

**Options.**
- **A — Bidirectional, 1% house spread.** Best UX; spread funds the house. (`ai_index` ↔ sub/gpu;
  sub↔sub and gpu↔gpu route through `ai_index`.)
- **B — One-way (sub/gpu → `ai_index` only).** Simplest regulatory story (pure redemption, no buy-back).

**Decision (provisional).** **Option A applied** in `credit-types.md` §3 and `openapi/credit.yaml`
(`convert`): bidirectional, 1% spread, rate from a ledger config source. **Flagged for counsel
sign-off (F22)** — if counsel requires redemption-only, switch to B (a `tech-lead` change to
`credit-types.md` + `convert`). Until counsel confirms, build against A but keep the spread + rate
in config so flipping to B is a policy change, not a code rewrite.

---

## ADR-0003 — Design system is canonical; tokens are placeholders pending brand book

**Status:** Accepted (2026-05-29) · **Owner:** trading-frontend

**Context.** The frontend needs one enforced visual language so screens don't drift and the product
reads as institutional (Bloomberg/Polymarket), not crypto-flashy. The brand book hasn't landed yet.

**Decision.** `docs/plans/DESIGN_SYSTEM.md` is the **single source of visual truth** (tokens, type
scale, spacing, number formatting, mock-data realism, aesthetic guardrails, mockup-vs-production
output rules). Token *values* are **v1 placeholders**; the *structure* (names, scale, rules) is
stable. Implementation lives in `apps/web/app/assets/css/tokens.css` — a design change is a one-line
edit there, never in a page/component. Every frontend change conforms; enforced by
`ENGINEERING_STANDARDS.md` §4, the `design-tokens-guard` pre-commit hook, and the `/ex-review` gate.

**Why.** Tokens-only + a written spec means the brand-book swap is a `tokens.css` value change with
zero component edits, and reviewers have an objective bar ("conforms to DESIGN_SYSTEM.md").

**Consequences.** When the brand book arrives: update `tokens.css` (+ the placeholder values in the
doc), re-verify WCAG-AA contrast, and record an ADR only if the *structure* changes. Affected docs:
`DESIGN_SYSTEM.md`, `ENGINEERING_STANDARDS.md` §4, `agents/trading-frontend.md`, the `/ex-*` commands.

---

> Template for new ADRs:
> `## ADR-NNNN — <title>` · **Status / Owner** · **Context** · **Decision** · **Why not <alt>** ·
> **Consequences (affected docs)**.
