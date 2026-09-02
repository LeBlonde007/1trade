# 1Trade workflow commands

Project slash commands that encode the build workflow from `docs/plans/`, so every feature is
started, advanced, fixed, and merged the same way — branch + version + standards + contracts, no
drift. Each `.md` file here is one `/command`; type `/` in Claude Code to see them.

| Command | Use it to |
|---|---|
| `/ex-start <feature-id>` | Begin a feature: creates `feat/<id>-<slug>`, computes the target version `v0.<milestone>.<patch>`, loads the feature's plan doc + contracts + standards, and lays out a task breakdown. e.g. `/ex-start F08`. |
| `/ex-update-feature <feature-id>` | Continue an in-progress feature: re-loads the spec + branch, advances the next acceptance criterion, keeps the feature doc + management tracker status honest. |
| `/ex-fix "<bug>"` | Fix a bug the disciplined way: `fix/<slug>` branch, reproduce with a failing test first, minimal fix, run the review gate. |
| `/ex-review [feature-id]` | Pre-merge gate: runs `/code-review` → `/simplify` → `/security-review` → `/verify`, walks the definition-of-done checklist, then tags `v0.<m>.<patch>` and updates `CHANGELOG.md`. |
| `/ex-contract <service> — <change>` | Change a shared contract the contract-first way (tech-lead owns `docs/contracts/`); classifies semver impact and lists the downstream blast radius. |
| `/ex-status [milestone]` | Progress glance: plan status vs. actual branches/tags, and what's unblocked to start next. |

## The normal loop

```
/ex-start F08           → branch feat/f08-inference-gateway, target v0.2.x, plan loaded
  …build with /ex-update-feature F08 as needed…
/ex-review F08          → review skills + DoD + tag v0.2.x + CHANGELOG
  …user merges when ready…
```

Bugs branch off the same way via `/ex-fix`. Anything that needs a shared interface to change goes
through `/ex-contract` first — never edit `docs/contracts/` mid-feature.

## Conventions these commands enforce

- **Branches:** `feat/<id>-<slug>`, `fix/<slug>`, `contracts/<service>-<slug>`.
- **Versioning (pre-GA):** `v0.<milestone>.<patch>`. Milestone = the M# in the feature doc; patch =
  next free for that milestone. **Tagged at merge (`/ex-review`), not at start**, so parallel
  feature branches never collide on a version.
- **Standards:** every command points at `docs/plans/ENGINEERING_STANDARDS.md` (per-function doc
  comments, fixed-point money math, `is_paper`, idempotency, tests-with-code, audit hooks).
- **No auto-push / no auto-merge.** The commands stop at "ready"; the user runs the merge.

These are prompts, not scripts — tune any file in this folder to taste; changes take effect next
time the command runs.
