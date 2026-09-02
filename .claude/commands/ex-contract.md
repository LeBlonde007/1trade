---
description: Propose or author a shared-contract change the contract-first way (tech-lead owns docs/contracts/), then list every downstream agent that must adapt.
argument-hint: <service|contract> — <what needs to change and why>
---

You are handling a contract change for 1Trade: **$ARGUMENTS**

Contracts are the ONLY way isolated services couple. They are owned by `tech-lead`. Changing one
carelessly is the single biggest source of cross-service breakage — so follow this exactly
(`docs/plans/CONTRACTS.md`).

## 1. Identify the contract
- Determine which artifact changes: an `openapi/<service>.yaml`, `schemas/types.sql`, an
  `events/<subject>.yaml`, or `credit-types.md`. Read the current version in `docs/contracts/`.
- If it doesn't exist yet, this is a *new* contract — note that.

## 2. Classify the change (semver, CONTRACTS.md §3)
- **PATCH** = clarification, no shape change.
- **MINOR** = additive, backward-compatible.
- **MAJOR** = breaking — requires dual-serving the old + new version for one cycle.
State which it is and justify. For events, a breaking change = a new versioned subject
(`*.v2`), with both fanning out until consumers cut over.

## 3. Author the change (this is a tech-lead action)
- Make the edit in `docs/contracts/` only. Keep `is_paper` on every money/credit/order surface,
  idempotency keys on mutating calls, pagination on lists, and the auth scope documented.
- Bump `info.version` (OpenAPI) or add the new event subject. Update `credit-types.md` if a credit
  type is added (and remember: that triggers a coordinated rollout across ledger + inference +
  platform-core + frontend).

## 4. List the blast radius
Using the sync table in `CONTRACTS.md` §6, list **every** downstream service/agent that consumes
this contract and must adapt. For each, say what specifically changes for them.

## 5. Gate + record
- Run `/security-review` if the change touches credits, orders, or auth.
- Update the affected feature doc(s) in `docs/plans/features/` so the plan and the contract agree.
- Commit on a branch with `contracts(<service>): <summary>` (conventional commits use
  `contracts(propose): ...` if you are only proposing, not yet authoring).

Report: the contract + version, the classification, the diff summary, and the ordered list of
downstream agents to update next. Do not push or merge unless asked.
