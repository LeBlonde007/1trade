# 1Trade — Claude Code Agent System

A roster of specialized subagents, one per domain in the Phase 6 v2 PRD/SSD, plus an
orchestration model that makes them work **together** without stepping on each other.

> **Verify the format before relying on it.** This uses the established Claude Code subagent
> format (`.claude/agents/*.md`, YAML frontmatter `name` / `description` / optional `tools` /
> optional `model`, markdown body = the agent's system prompt). The feature evolves — confirm
> field names against the current docs map:
> `https://docs.anthropic.com/en/docs/claude-code/claude_code_docs_map.md`

---

## The core idea (and the constraint that shapes everything)

Claude Code subagents each run in their **own isolated context window** and **cannot call one
another**. So "agents working together" does **not** mean agents talking to agents. It means:

1. **The main `claude` session is the orchestrator/conductor.** It reads `CLAUDE.md` + the roadmap,
   breaks a goal into tasks, and delegates each task to the right specialist agent. Delegation is
   automatic (based on each agent's `description`) or explicit (`use the matching-engine agent to …`).
2. **Agents collaborate through written contracts**, not conversation. Service A publishes an
   OpenAPI spec / SQL schema / event schema in `docs/contracts/`; service B consumes it as
   read-only truth. The contract is the hand-off.
3. **`tech-lead` is the coordination brain.** It plans, decomposes, authors/owns the shared
   contracts, and reviews cross-cutting changes. It writes specs and docs, not feature code.

This is why the contract-first rule in `CLAUDE.md` exists: it's the substitute for agents being
able to talk to each other.

```
                         You (a goal)
                              │
                  ┌───────────▼────────────┐
                  │  Main claude session     │   ← orchestrator
                  │  (reads CLAUDE.md +       │
                  │   roadmap, delegates)     │
                  └───────────┬────────────┘
                              │ delegates tasks
        ┌──────────┬──────────┼──────────┬───────────┐
        ▼          ▼          ▼          ▼           ▼
   tech-lead  trading-   matching-  credit-   …all specialists
   (plan +    frontend   engine     ledger
    contracts)
        │          │          │          │
        └──────────┴────┬─────┴──────────┘
                        ▼
              docs/contracts/  ← the shared truth they all read
```

---

## The roster

### Coordination
| Agent | Owns | Writes code? |
|---|---|---|
| `tech-lead` | Decomposition, shared contracts (OpenAPI/SQL/events), cross-service review, sequencing | No — specs/docs/review only |

### Trading domain (the spine — ship first)
| Agent | Owns |
|---|---|
| `trading-frontend` | Nuxt 4 trading UI, design system, mock-data mode, all screens (incl. HTML→Nuxt migration) |
| `matching-engine` | Go order book, price-time matching, order lifecycle, event sourcing |
| `market-maker` | Go automated two-sided quoting, spread/skew logic, MM risk limits |
| `credit-ledger` | Go balances, append-only tx, conversions, hash chain, settlement integration |
| `index-service` | Go index computation + daily publication + methodology integrity + audit chain |
| `surveillance` | Go manipulation detection (wash/spoof/layering/marking-the-close), position limits |

### Compute / inference / supply
| Agent | Owns |
|---|---|
| `inference-ml` | vLLM, model catalog, multi-model-per-GPU, inference gateway, credit debit on usage |
| `compute-platform` | K8s + Kueue + Volcano + GPU Operator, control plane, supply-source abstraction |
| `settlement-trust` | Attestation stack, escrow/streamed payout, backing ratio, proof-of-reserves, DC supply onboarding |

### Platform / cross-cutting
| Agent | Owns |
|---|---|
| `platform-core` | Auth, accounts/orgs, RBAC, billing (Stripe/ACH/wire), multi-currency, API gateway, CLI client |
| `infra-sre` | IaC, K8s ops, CI/CD, observability, secrets, deploy, reliability |
| `security-compliance` | SOC 2 controls, KYC/AML, regulatory framing review, audit-trail + security review of changes |

---

## How a real task flows through the system

**Example: "Add AI-credit → text-credit conversion."**

1. **Orchestrator** recognizes this touches a shared interface → routes to `tech-lead` first.
2. **`tech-lead`** authors/updates the contract: the `POST /v1/credits/convert` OpenAPI entry, the
   conversion-rate source, and the credit-types doc. Commits to `docs/contracts/`.
3. Orchestrator delegates in dependency order:
   - **`credit-ledger`** implements the atomic deduct/add + hash-chain extension against the contract.
   - **`platform-core`** wires the gateway route + auth scope.
   - **`trading-frontend`** builds the convert UI against the now-fixed OpenAPI.
   - **`security-compliance`** reviews the audit trail + that `is_paper` is respected.
4. Each agent worked in isolation but against the **same contract**, so the pieces fit.

The orchestrator never lets two agents edit the same contract concurrently. Contracts change in
one place (`tech-lead`), then implementations follow.

---

## Build sequencing (under the 2026-05 GTM pivot)

> **Authoritative source:** `docs/plans/SEQUENCING.md` (milestone-by-milestone) and
> `docs/plans/README.md`. The older trading-first ordering from the Phase 6 roadmap is
> **superseded** — the build is platform-first; the exchange is paused/kept-warm.

Dependency order the orchestrator should follow — each milestone unblocks the next dollar
(milestones are stages, not calendar months):

```
M1  infra-sre (foundation) + platform-core (auth/accounts/CLI) + credit-ledger (skeleton)
        + trading-frontend (platform-console shell, mock data) ; tech-lead authors first contracts
M2  inference-ml (gateway + vLLM, first 3 models) ─ credit-ledger (debits) ─ compute-platform (v0)
        ─ platform-core (Stripe billing) ─ trading-frontend (wired wallet + catalog)
M3  compute-platform (on-demand GPU + reserved) ─ inference-ml (catalog + packing)
        ─ platform-core (ACH/wire + JPY) ─ settlement-trust (owned-DC backing)
M4  settlement-trust (partner DC + attestation + payouts) ─ platform-core (SAML/SCIM/sub-accounts)
        ─ security-compliance (SOC 2 kickoff)
M5  inference-ml (full catalog) ─ compute-platform (clusters/InfiniBand) ─ index-service (private index)
M6  hardening across all ─ security-compliance (SOC 2 Type I + license decision)

Keep-warm in parallel (paused exchange): matching-engine (mock + spec), market-maker (spec),
surveillance (basic abuse only), index-service (private), trading-frontend (`/trade` demo).
```

**Critical-path agents under the pivot:** `infra-sre` and `platform-core` first, then
`inference-ml` and `compute-platform` (the revenue machine), then `settlement-trust`.

---

## Setup

1. Drop `CLAUDE.md` at the repo root and the `agents/` files into `.claude/agents/`.
2. Create `docs/contracts/` and `docs/phase6-prd-ssd.md` (paste the Phase 6 v2 doc there).
3. From the repo root run `claude`. Confirm agents are loaded with `/agents`.
4. Start with: *"Read CLAUDE.md, docs/re/1trade_gtm_focus_update.md, and docs/plans/README.md.
   Use tech-lead to produce the Milestone-1 task breakdown and author the initial contracts,
   then begin delegating."*

## Notes on tuning

- **`description` drives auto-delegation.** Keep each one sharp and action-oriented ("Use proactively
  for X"). Vague descriptions = the orchestrator picks the wrong agent.
- **`tools`** is optional; omit to inherit all. We restrict `tech-lead` and `security-compliance` to
  read/review-mostly so they can't accidentally rewrite implementations.
- **`model`** is optional. The hardest agents (`tech-lead`, `matching-engine`, `index-service`,
  `settlement-trust`) are set to the strongest model; routine ones can run lighter to save cost.
- This is **12 specialists + 1 coordinator** — comprehensive because Phase 6 is. You can start with
  just the critical-path four (`tech-lead`, `trading-frontend`, `matching-engine`, `platform-core`)
  and add the rest as the team/scope grows.
