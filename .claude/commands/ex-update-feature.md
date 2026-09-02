---
description: Continue an in-progress feature — re-load its plan + branch state, advance the work, and keep the feature doc's status/% current.
argument-hint: <feature-id> (e.g. F08)  [what to do next]
---

You are continuing work on 1Trade feature **$1**. Direction (if any): $ARGUMENTS

## 1. Re-orient
- Find and read `docs/plans/features/$1-*.md` (the spec + acceptance criteria + milestone).
- Run `git status` and `git branch --show-current`. The expected branch is
  `feat/<id-lower>-<slug>`.
  - If you're **not** on that branch and it exists, `git switch` to it.
  - If the branch doesn't exist yet, tell the user to run `/ex-start $1` first — don't invent one.
- Skim what's already implemented for this feature so you continue rather than duplicate.

## 2. Advance the work
- Pick up the next incomplete acceptance criterion (check the TaskList; create tasks if missing).
- Honor the standards every time you write code: **doc-comment on top of every function**,
  fixed-point money math, `is_paper` threaded through, idempotent mutating calls, input validated
  at the boundary, tests in the same change, audit hooks where credits/orders/admin are involved.
- Consume contracts from `docs/contracts/` as read-only truth. If you discover you need a contract
  change, STOP and use `/ex-contract` — never shadow-edit a shared contract.
- Commit incrementally with conventional messages (`feat(<service>): $1 — <summary>`).

## 3. Keep the tracker honest
- Update the feature doc's acceptance-criteria checkboxes that are now satisfied.
- Update the matching row in `docs/plans/MANAGEMENT_PLAN.md` (§6/§7): **Status** (🔵/✅) and **%**.
  Keep these truthful — partial is 🔵, not ✅.

## 4. Close out the turn
- Run tests for what you changed; report pass/fail honestly with output.
- Tell the user what's done, what's left, and whether it's ready for `/ex-review`.

Do not push or merge unless asked.
