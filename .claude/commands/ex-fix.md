---
description: Fix a bug — create a fix branch, reproduce, fix with a regression test, and run the review gate.
argument-hint: "<short description of the bug>"  [feature-id or service]
---

You are fixing a bug in Exascale: **$ARGUMENTS**

## 1. Branch
- Run `git status`; make sure the working tree is clean (stash or ask if not).
- `git switch main && git pull --ff-only` (skip the pull if there's no upstream).
- Create a fix branch: `git switch -c fix/<kebab-slug-of-the-bug>`.

## 2. Reproduce first
- Find the affected code (Grep/Glob). If a feature/service is named in the args, start there;
  otherwise locate it from the symptom.
- **Write a failing test that reproduces the bug** before changing any logic. A fix without a
  regression test is not done (`ENGINEERING_STANDARDS.md` §6).

## 3. Fix
- Make the smallest correct change. Match the surrounding code's style and comment density.
- Keep the standards: doc-comment any new function, fixed-point money math, `is_paper` respected,
  idempotency preserved, errors handled (never swallowed), inputs validated.
- If the root cause is a contract mismatch, STOP — fixing the contract goes through `/ex-contract`,
  not a local workaround.

## 4. Verify
- Run the new regression test (it must now pass) plus the affected service's existing tests.
- If the bug touched credits / orders / auth / billing / customer data, run `/security-review`.
- Consider `/verify` to confirm the fix in the running app where the behavior is observable.

## 5. Commit
- Conventional message: `fix(<service>): <summary>` with a body explaining root cause + the test
  that now guards it.
- Update the relevant feature doc / MANAGEMENT_PLAN row if the bug changed a feature's status.

Report what was broken, the root cause, the fix, and the test that now prevents regression. Do not
push or merge unless asked.
