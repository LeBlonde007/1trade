---
description: Pre-merge gate — run the review-skill workflow + the definition-of-done checklist, then tag the version and update the changelog.
argument-hint: [feature-id]  (defaults to the current branch's feature)
---

You are running the pre-merge review gate for the current branch. Feature (if given): $1

## 1. Establish scope
- `git branch --show-current` and `git status`. Determine the feature id from the branch name
  (`feat/<id>-...`) or from `$1`. Read its `docs/plans/features/<id>-*.md` for acceptance criteria.
- `git diff main...HEAD --stat` to see what's actually changed.

## 2. Run the review skills in order (per ENGINEERING_STANDARDS.md §9)
1. `/code-review` — fix the correctness findings it surfaces.
2. `/simplify` — apply the reuse/simplification/altitude cleanups (quality only).
3. `/security-review` — **mandatory** if the diff touches credits, orders, auth, billing, or
   customer data. Any blocker it finds must be resolved before proceeding.
4. `/verify` — where the change is observably runnable, confirm it actually works (not just tests).

## 3. Definition-of-done checklist (ENGINEERING_STANDARDS.md §10)
Walk the checklist explicitly and report each as pass/fail with evidence:
- [ ] Builds / typechecks. Tests pass; new logic has tests.
- [ ] **Every function has a doc comment.**
- [ ] Linters clean (`golangci-lint` / `ruff` / `eslint`); formatted.
- [ ] Contracts honored — no undocumented drift (contract changes went through `/ex-contract`).
- [ ] `is_paper` threaded; fixed-point money math; idempotency keys on mutating calls.
- [ ] Audit hooks where credits/orders/admin are involved.
- [ ] No secrets (`gitleaks`); no high-sev CVEs.
- [ ] Observability hooks present; Dockerfile / k8s / dashboard / runbook updated if a new surface shipped.
- [ ] Feature acceptance criteria met; feature doc + MANAGEMENT_PLAN row updated.
- [ ] **If the diff touches `apps/web/` or `Exascale Frontend/`:** conforms to
      `docs/plans/DESIGN_SYSTEM.md` — `design-tokens-guard` passes (no raw hex/rgb in components),
      numbers are mono + `tabular-nums`, sentence case, semantic ▲/▼, institutional aesthetic
      (no anti-references), realistic mock data (no Lorem). Run the "Larry Fink" gut check.

If anything fails, fix it (or hand it back) and re-run the relevant step. Do not pass a failing gate.

## 4. Version + changelog (only after the gate is green)
- Compute the target version the way `/ex-start` did: `v0.<milestone>.<patch>` for this feature's
  milestone, next free patch from `git tag --list 'v0.<m>.*'`.
- Add an entry to root `CHANGELOG.md` (create it Keep-a-Changelog style if absent):
  `## v0.<m>.<patch> — <feature id> <title>` with a short bullet list of what shipped.
- Tag the tip: `git tag -a v0.<m>.<patch> -m "<id> <title>"` (tag now; the merge itself and any
  push happen only when the user asks).

## 5. Report
Summarize: the green/red checklist, what the review skills changed, the version tagged, and the
exact merge command the user can run when ready (e.g. `git switch main && git merge --no-ff <branch>`).
Do not merge or push automatically.
