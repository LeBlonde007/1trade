# Contracts — How agents stay in sync

Agents run in isolated contexts and cannot see each other's work. They coordinate through
written contracts in `docs/contracts/`. This file is the rulebook for that.

---

## 1. The four contract types

```
docs/contracts/
├── openapi/             ← REST surface, one yaml per service
├── schemas/             ← SQL types shared across services, plus a registry of migrations
├── events/              ← NATS subjects + JSON payload schemas (event-sourced surfaces)
└── credit-types.md      ← the canonical credit-type enum — referenced everywhere
```

- **`openapi/<service>.yaml`** — every external (gateway-facing) HTTP route the service exposes.
  Internal-only Go interfaces don't live here; only what another service or the gateway calls.
- **`schemas/`** — SQL types shared across services (e.g., the `credit_type` enum, the `tenant_id`
  shape, common audit-log columns). Per-service tables live in `services/*/migrations/`. The shared
  registry exists so two services can't disagree on a shared shape.
- **`events/<subject>.yaml`** — NATS subject name + payload JSON schema + version. Every event
  consumer reads the schema as truth.
- **`credit-types.md`** — the enum of every credit type (`ai_index`, `text`, `speech`, `image`,
  `video`, `embeddings`, `gpu_h100`, `gpu_h200`, ...). Adding a new type is a `tech-lead`-authored
  PR and a coordinated rollout across `credit-ledger` + `inference-ml` + `platform-core`
  (billing/UI) + `trading-frontend`.

---

## 2. The contract-first rule

Restated from `CLAUDE.md` because it is the single most-violated rule:

1. **An agent never edits a shared contract on its own.** Contracts are owned by `tech-lead`.
   Other agents propose changes; `tech-lead` writes them.
2. **If you need an interface and it isn't in `docs/contracts/`, STOP.** Don't invent it. Ask
   the orchestrator to have `tech-lead` author it first. Building against assumed shapes is the
   single biggest source of inter-agent breakage.
3. **An agent consumes contracts as read-only truth.** If the contract is wrong, you propose a
   change; you don't shadow-implement around it.

---

## 3. Versioning

- `openapi/<service>.yaml` carries `info.version` (semver, MAJOR.MINOR.PATCH).
- Breaking changes bump MAJOR; additions bump MINOR; doc/clarification PATCH.
- Both old and new MAJOR versions are served in parallel for one release cycle (Kong routes
  `/v1/...` and `/v2/...`).
- Event subjects carry a version in the subject (`trades.executed.v1`); new versions = new
  subjects, both fan out until consumers cut over.
- Schema migrations are append-only history; the live shape is the result of applying them all.

---

## 4. The contract-change workflow

When an agent needs a contract change:

```
1. The implementing agent opens a PR titled:
     contracts(propose): <service> — <one-line summary>
   with: rationale, proposed change, list of affected agents.

2. `tech-lead` reviews, authors the actual contract change in docs/contracts/, and lists every
   downstream agent that needs to adapt.

3. The PR merges only with `tech-lead` approval + a checklist of downstream agents to ping.

4. Orchestrator queues the downstream-agent updates in dependency order.

5. `security-compliance` reviews any contract change that touches credits, orders, or auth.
```

---

## 5. The seed contracts (these get written first)

Tracked under feature [F01](features/F01-foundation-infra.md) and authored in Milestone 1 by
`tech-lead`:

| Contract | First version writes | Owners (read) |
|---|---|---|
| `openapi/platform-core.yaml` | auth (login, OAuth, SAML), tenants/orgs, users, RBAC | every service authenticates inbound requests; CLI; web |
| `openapi/credit.yaml` | balances, transactions, purchase, convert | inference-gateway (debit), compute-control (debit), platform-core (purchase), web (wallet), CLI |
| `openapi/inference.yaml` | OpenAI-compatible chat/embeddings/audio/image | customer code; web; CLI |
| `openapi/compute.yaml` | instances, types, quota, jobs | CLI; web; supply-service (capacity registration) |
| `openapi/supply.yaml` | partner capacity registration, utilization, payout | compute-control; partner-DC agents |
| `openapi/index.yaml` | current, history, methodology — even keep-warm has methodology published | web; CLI; future tradeable services |
| `schemas/types.sql` | tenant_id, credit_type, audit columns | all services |
| `events/credit.tx.v1.yaml` | one event per ledger movement | observability; settlement-trust (payout) |
| `events/inference.usage.v1.yaml` | one event per request (after stream close) | credit-ledger (debit); analytics; web dashboard |
| `events/compute.usage.v1.yaml` | one event per GPU-hour consumed | credit-ledger (debit); settlement-trust (payout) |
| `events/partner.capacity.v1.yaml` | capacity heartbeat from partner-DC agent | compute-control |
| `credit-types.md` | the enum: ai_index, text, speech, image, video, embeddings, gpu_h100, gpu_h200 | every service |

### Phase 2 / keep-warm contracts (specced, not yet wired)

These are authored alongside Phase 1 so the switch-on is a config flip, not a redesign:

| Contract | Status |
|---|---|
| `openapi/trading.yaml` | specced; backed by mock data adapter in `matching-engine` |
| `events/trades.executed.v1.yaml` | specced |
| `events/orders.state.v1.yaml` | specced |
| `events/surveillance.alert.v1.yaml` | specced |

---

## 6. The sync table — who consumes what

This table is the explicit cross-service coupling map. Every cell is a contract.

| Producer ↓ / Consumer → | platform-core | credit-ledger | inference-ml | compute-platform | settlement-trust | index-service | trading-frontend | CLI |
|---|---|---|---|---|---|---|---|---|
| `platform-core` (auth) | — | jwt | jwt | jwt | jwt | jwt | jwt | jwt |
| `credit-ledger` | balance API | — | debit API | debit API | mint/burn API | (none) | balance API | balance API |
| `inference-ml` | usage events | inference.usage.v1 → debit | — | (none) | (none) | (none) | catalog API | catalog API |
| `compute-platform` | quota API | compute.usage.v1 → debit | (schedules pods) | — | partner.capacity.v1 | (none) | instances API | instances API |
| `settlement-trust` | partner DC mgmt | mint/burn calls | (none) | capacity attestation | — | (none) | partner portal | (none) |
| `index-service` | (none) | (none) | (none) | (none) | (none) | — | index API | index API |
| `matching-engine` (Phase 2) | (none) | settle on fill | (none) | (none) | (none) | trades feed | trading API | trading API |
| `market-maker` (Phase 2) | (none) | (none) | (none) | (none) | (none) | (none) | (none) | (none) |
| `surveillance` (Phase 2) | (none) | (none) | (none) | (none) | (none) | mark-to-close flags | alert events | (none) |

A cell that says "none" means the producer **must not** make calls into the consumer; if it
needs to, that's a contract change request to `tech-lead`.

---

## 7. Where `is_paper` lives in contracts

Every contract that carries a tenant action or a money/credit movement carries `is_paper`:

- OpenAPI: every request body field that creates a transaction or order has `is_paper: boolean`.
- Schemas: every relevant table has an `is_paper boolean not null` column with a default of
  `true` in dev/staging and explicit in prod.
- Events: every event payload carries `is_paper`. Consumers MUST filter.

Even with trading paused, `is_paper` lives in the platform schemas because Phase 2 demands
it and adding it later is a painful migration.

---

## 8. Contract review checklist

Every contract PR is reviewed against:

- [ ] Backwards compatible? If not, MAJOR bump + dual-serving plan.
- [ ] `is_paper` carried wherever applicable?
- [ ] Idempotency key on every mutating call?
- [ ] Pagination on every list endpoint?
- [ ] Authentication scope documented?
- [ ] PII surface justified?
- [ ] Audit log produced by this operation? (For ledger / orders / admin.)
- [ ] Downstream agents listed?
- [ ] `security-compliance` reviewed if it touches credits, orders, auth, or PII?

`tech-lead` runs this checklist; `security-compliance` countersigns.
