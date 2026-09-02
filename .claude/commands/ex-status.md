---
description: Show build progress — feature statuses from the plan vs. actual branches/tags, and what's unblocked to work on next.
argument-hint: [milestone, e.g. M2]  (optional — defaults to all)
---

Give a concise progress report for 1Trade. Milestone filter (if any): $1

## 1. Gather
- Read the feature tables in `docs/plans/MANAGEMENT_PLAN.md` (§6 Phase 1, §7 keep-warm) for the
  recorded Status / % per feature.
- Run `git branch -a` and `git tag --list 'v0.*'` to see what's actually been started/shipped.
- Run `git log --oneline -15` for recent movement.

## 2. Reconcile plan vs. reality
- For each feature, compare the doc's recorded status to the git evidence (a `feat/<id>-*` branch =
  in progress; a `v0.<m>.<patch>` tag for it = shipped). Flag mismatches (e.g. doc says ✅ but no
  tag, or a branch exists but the doc still says ⬜).

## 3. Report
Output three short sections:
1. **In flight** — features with an open branch, their % and what's left.
2. **Shipped** — features with a version tag.
3. **Unblocked next** — features whose dependencies (per their feature docs) are satisfied and that
   sit earliest on the critical path (`docs/plans/SEQUENCING.md`). Recommend the next 1–3 to start
   with `/ex-start`.

Keep it tight — this is a status glance, not a full audit. Note any plan/reality drift the user
should correct.
