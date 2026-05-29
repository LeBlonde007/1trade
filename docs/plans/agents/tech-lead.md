# Agent plan — `tech-lead`

> Active, continuous. The coordination brain — the substitute for agents being able to talk to
> each other.

## 1. Scope under the GTM pivot

Unchanged. Owns `docs/contracts/`, decomposes work, sequences agents, reviews cross-cutting
changes for contract drift, `is_paper` correctness, audit coverage, and fixed-point math. Under
the pivot, the sequencing emphasis shifts to platform-first; the contract-first rule is even more
important because Phase 2 switch-on must be a flag flip, not a redesign — so Phase 2 contracts
get written (specced, not implemented) alongside Phase 1.

## 2. Features owned

| Feature / artifact | Status |
|---|---|
| `docs/contracts/openapi/*` | active, continuous |
| `docs/contracts/schemas/types.sql` | active, continuous |
| `docs/contracts/events/*` | active, continuous |
| `docs/contracts/credit-types.md` | active, M1 v1 → versioned thereafter |
| `docs/plans/` (this plan) — maintenance | active, continuous |
| Task decomposition for orchestrator | active, every multi-service task |

## 3. Milestone-by-milestone

### Milestone 1 — Author the seed contracts
- `openapi/platform-core.yaml` v1.0.0 (auth, accounts, RBAC, billing skeleton).
- `openapi/credit.yaml` v1.0.0 (balances, transactions, purchase, convert).
- `openapi/inference.yaml` v0.9.0 (OpenAI-compatible skeleton; iterates with `inference-ml`).
- `openapi/compute.yaml` v0.9.0 (instances, types, quota).
- `openapi/supply.yaml` v0.9.0 (capacity registration, partner ops).
- `openapi/index.yaml` v0.9.0 (current, history, methodology — methodology is real even in keep-warm).
- `openapi/trading.yaml` v0.9.0 (specced; backed by mock data adapter — Phase 2 surface).
- `schemas/types.sql` (tenant_id, credit_type enum, audit columns, is_paper).
- `credit-types.md` v1.0 (full enum: ai_index, text, speech, image, video, embeddings, gpu_h100, gpu_h200).
- `events/credit.tx.v1.yaml`, `events/inference.usage.v1.yaml`, `events/compute.usage.v1.yaml`,
  `events/partner.capacity.v1.yaml`.
- Phase 2 events specced (not consumed yet): `events/trades.executed.v1.yaml`,
  `events/orders.state.v1.yaml`, `events/surveillance.alert.v1.yaml`.

### Milestone 2 — Iteration as services land
- Finalize `openapi/inference.yaml` v1.0.0 with `inference-ml`.
- Finalize `openapi/credit.yaml` purchase + convert shapes once Stripe webhook flow is wired.
- Review every PR touching a contract.

### Milestone 3 — Compute + supply finalization
- Finalize `openapi/compute.yaml` v1.0.0 (instances, reserved capacity, clusters spec).
- `openapi/supply.yaml` v0.9.x for owned-DC anchor.

### Milestone 4 — Partner DC + enterprise
- `openapi/supply.yaml` v1.0.0 for first partner DC.
- `openapi/platform-core.yaml` MINOR bump for SAML/SCIM/sub-accounts.
- Verify partner.capacity.v1 event consumers.

### Milestone 5 — Catalog + index
- Finalize index methodology document (`docs/index-methodology.md`).
- Coordinate the index keep-warm contract: published methodology must be the methodology that
  Phase 2 uses — no surprise changes at switch-on.

### Milestone 6 — Phase 2 readiness
- Verify Phase 2 contracts are complete and stub-implemented:
  - `openapi/trading.yaml` v1.0.0 (matching engine + market maker surface).
  - `events/trades.executed.v1.yaml`.
  - `events/orders.state.v1.yaml`.
  - `events/surveillance.alert.v1.yaml`.
- Plan the Phase 2 switch-on as a contract-version coordination (which OpenAPI versions go live,
  which event subjects start being consumed, which is_paper handling changes).

## 4. Workflow

For every multi-service task arriving from the orchestrator:

```
1. Read CLAUDE.md, docs/re/exascale_gtm_focus_update.md, docs/re/phase6_v2.md, docs/plans/README.md.

2. If the task touches a shared interface:
   - Look at the existing contract.
   - Decide whether: PATCH (clarify), MINOR (add), MAJOR (break) — and justify.
   - Author the change in docs/contracts/ FIRST.
   - List every downstream agent that must adapt.

3. Produce the task list:
   - One row per implementing agent.
   - Dependency order.
   - Which contract version each agent consumes.

4. After implementations land:
   - Review each service PR for drift.
   - Verify is_paper correctness.
   - Verify audit coverage on credit/order/admin operations.
   - Verify fixed-point money math (no float).

5. Update docs/plans/ when contracts change (the feature doc and the sequencing doc).
```

## 5. Tools restriction

Read / Grep / Glob / Write / Edit only (no Bash, no service work) — per the agent definition.
That keeps `tech-lead` from accidentally writing implementation code.

## 6. Definition of done (per task)

- Contracts are unambiguous and versioned.
- The task breakdown names owner + dependency + contract version for each piece.
- The four immutable commitments are preserved (re-checked at the end).
- The orchestrator can hand the task list to specialist agents without ambiguity.
