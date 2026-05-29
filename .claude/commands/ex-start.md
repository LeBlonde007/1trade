---
description: Start a feature — create its branch, set the target version, load its plan doc + contracts + standards, and produce a task breakdown.
argument-hint: <feature-id> (e.g. F01, F08, KW03)  [optional notes]
---

You are starting work on Exascale feature **$1**. Notes (if any): $ARGUMENTS

Follow this runbook exactly. Stop and ask the user only if a step genuinely blocks.

## 1. Locate the feature
- Normalize `$1` to uppercase (e.g. `f08` → `F08`).
- Find its plan doc with Glob: `docs/plans/features/$1-*.md` (case-insensitive on the ID).
- If none matches, list `docs/plans/features/` and ask which feature; do not guess.
- Read the doc fully. Extract: **title**, **owning agent(s)**, **milestone (M#)**,
  **dependencies**, **contracts consumed / produced**, **acceptance criteria**, **milestone gate**.

## 2. Confirm the ground rules (read, don't re-derive)
- Read `CLAUDE.md` (the four immutable commitments + the GTM-pivot banner) and
  `docs/plans/ENGINEERING_STANDARDS.md` (per-function doc comments, fixed-point money math,
  `is_paper` everywhere, idempotency, tests-with-code, audit hooks, the review-skill workflow).
- Read `docs/plans/CONTRACTS.md` §2 (contract-first rule). If the feature needs a contract that
  is **not yet** in `docs/contracts/`, STOP — that contract must be authored by `tech-lead` first
  (`/ex-contract`). Do not build against an assumed shape.
- **If this feature touches the frontend** (owner `trading-frontend`, or any change under
  `apps/web/` or `Exascale Frontend/`), also read `docs/plans/DESIGN_SYSTEM.md` and build to it:
  tokens only (no raw hex/rgb), mono + `tabular-nums` numbers, institutional aesthetic, realistic
  mock data. A design value with no token = add the token to `tokens.css`, never inline it.

## 3. Check dependencies
- For each dependency the feature doc lists, check whether it looks done: run
  `git tag --list 'v0.*'` and `git branch -a` and look for the dependency's branch/tag.
- If a dependency is clearly unmet, warn the user and ask whether to proceed anyway. Surface it;
  don't silently build on missing foundations.

## 4. Create the branch (off the latest main)
- Run: `git switch main` then `git pull --ff-only` (if a remote/upstream exists; if it errors
  because there's no upstream, continue on local main — note that you did).
- Build a slug from the feature title (kebab-case, ≤4 words).
- Create and switch: `git switch -c feat/<id-lower>-<slug>` (e.g. `feat/f08-inference-gateway`).

## 5. Compute the target version (announce, do NOT tag yet)
- Milestone number `m` = the M# from the feature doc.
- Look at existing tags: `git tag --list 'v0.'$m'.*'`. Next patch = (highest patch seen) + 1,
  else 0. Target version = `v0.<m>.<patch>`.
- **Do not create the tag now.** Tagging happens at merge (`/ex-review`) so parallel feature
  branches never collide. Just announce: "Target version for this feature: `v0.<m>.<patch>`."

## 6. Lay out the plan
Print a short brief:
- **Feature:** id + title.
- **Owner agent:** from the doc.
- **Branch:** the one you created. **Target version:** the one you computed.
- **Contracts to consume (read-only):** list them; note any you must NOT call (per the sync table).
- **Contracts to produce:** list; remind that authoring/altering them goes through `/ex-contract`.
- **Definition of done:** point at `ENGINEERING_STANDARDS.md` §10 + the feature's acceptance criteria.

## 7. Break it down and begin
- Create tasks (TaskCreate) for the feature's acceptance criteria, in dependency order, including
  "write tests", "doc-comment every function", and a final "run `/ex-review`".
- If the feature's owning agent is a specialist in the roster, delegate implementation to that
  agent via the Agent tool, handing it: the feature doc path, the branch name, the target version,
  the contracts list, and the standards file. Otherwise implement inline.
- Make the first commit only when there's something real to commit, using a conventional message:
  `feat(<service>): <id> — <summary>`.

Do **not** push or open a PR unless the user asks. End by telling the user the branch, the target
version, and the next command they'll likely want (`/ex-update-feature $1` to continue, or
`/ex-review` before merge).
