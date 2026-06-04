# The way of working — a portable build playbook

> This file is **not** a feature list. It is the *method* — how work is planned, implemented, proven,
> and shipped here — written so it can be lifted into a different project. Where something is specific
> to this stack, it's marked **[adapt]** with what to swap. If you copy one file to a new repo, copy
> this one and delete the bits you don't need.

---

## 0. The one-paragraph version

Build **vertically, one thin slice at a time**: pick the smallest feature that a user can actually
*do something* with, write the shared contract first, implement it back-to-front (schema → domain →
store → API → enforcement → tests), wire exactly one screen/CLI path to it **live** (no mock), prove
it by running the real app and looking at the result, run the review gate, then **commit → branch →
merge --no-ff → tag (per-milestone SemVer) → push**. Never leave finished work uncommitted, never
ship a screen wired to fake data, never trade away the project's few immutable commitments.

---

## 1. Principles (these never bend)

1. **Contract-first.** Services/modules couple **only** through written contracts (API specs, DB
   schema, event schemas, shared enums) in one `contracts/` directory. Code consumes a contract as
   read-only truth. If the contract you need doesn't exist, **author it first** — don't build against
   an assumption. One owner approves contract changes; everyone else adapts. This is what lets
   parallel work (or parallel agents) not collide.
2. **Live or it doesn't count.** A screen wired to mock data is a lie you'll have to undo. Every UI
   path proxies to a real backend; every backend path hits a real store. The only allowed placeholder
   is a surface whose backend *genuinely does not exist yet* — and it must be **labelled as such**.
3. **Honest over impressive.** If a feature is half-built, the screen says so (tag it `M3`/`M4`/
   "in review"). Never fabricate a customer, a balance, an invoice, a "verified" badge. A reviewer
   should never be unsure whether something is real.
4. **Prove it by running it.** "Compiles" and "tests pass" are necessary, not sufficient. Launch the
   actual app, drive the real path, and **look at the output** (screenshot the screen, read the JSON,
   check the audit row). A blank screenshot is a failed launch.
5. **Fail closed.** Security/authorization/enforcement checks default to *deny*. A store error on a
   gate returns "no", not "yes". The client check is convenience; the server is authoritative.
6. **Ship the slice.** Done work gets committed, tagged, and pushed the same session. Uncommitted
   verified work is waste and risk.

> **[adapt] Immutable commitments.** This repo has 3–4 "never trade these away" rules (e.g. "the
> headline product must feel real even in sandbox", "audit integrity from day one"). Define yours up
> front in the project's root context file and let them outrank everything below.

---

## 2. Repository shape that makes the above possible

```
<repo>/
├── <ROOT_CONTEXT>.md          ← loaded into every session: stack, conventions, the immutable rules
├── docs/
│   ├── <spec>.md              ← product source of truth
│   ├── contracts/             ← THE ONLY coupling surface
│   │   ├── openapi/           ← one API spec per service
│   │   ├── schemas/           ← SQL migrations / shared types
│   │   ├── events/            ← async message subjects + payloads
│   │   └── <shared-enums>.md  ← canonical enums referenced everywhere
│   └── plans/
│       ├── README.md          ← the build plan: per-domain + per-feature, ordered
│       ├── ENGINEERING_STANDARDS.md
│       └── WAY_OF_WORKING.md  ← this file
├── services/ | apps/ | <code>
├── deploy/                    ← IaC, k8s, CI
└── <agent/role definitions>   ← optional: one file per domain owner
```

- **A root context file** (here `CLAUDE.md`) is auto-loaded every session and is ground truth. Put the
  locked tech stack, the global conventions, and the immutable commitments there. Keep it short enough
  that it's actually read.
- **One build-plan index** (`docs/plans/README.md`) lists features in **dependency order** with status,
  so "what's next" is never a guess.

---

## 3. The per-feature loop (run this every time)

```
EXPLORE → PLAN → CONTRACT → IMPLEMENT (back-to-front) → TEST → WIRE ONE PATH LIVE →
PROVE LIVE → REVIEW → SHIP (branch → commit → merge --no-ff → tag → push)
```

### 3.1 Explore first, always
Before writing anything, read the code that already exists for this area: the enforcement point you'll
hook, the store patterns, the migration style, the handler conventions, the nearest sibling feature.
Match the house style exactly — naming, error shape, doc-comment density. Grep widely; read the few
files that matter fully. **Time spent here is the cheapest time in the loop.**

### 3.2 Plan the slice
Decide the *thinnest* end-to-end path. For a gate/feature that means: what's the smallest thing a user
can do and see work? Write the steps down (a task list for anything ≥3 steps). Identify the single
enforcement/seam point.

### 3.3 Contract before code
Author the API/schema/event/enum changes in `contracts/` first. This is the interface everyone else
(or your future self, or the frontend) builds against. Validate it parses. Only then implement.

### 3.4 Implement back-to-front (server features)
The reliable order — each layer is testable before the next exists:
1. **Migration** — schema change, idempotent (`IF NOT EXISTS`), minimal columns, comment *why*.
2. **Domain** — pure logic (enums, state machines, predicates, money math). **No IO.** Unit-test it
   in isolation — this is where correctness lives and where tests are cheap.
3. **Store** — the IO layer. Parameterized queries only. Guard state transitions in the `WHERE`
   clause so concurrent/duplicate calls are no-ops, not corruption.
4. **API/handler** — decode → validate input (whitelist, lengths, types) → authz → call store →
   audit → respond with the contract's shapes. Generic error bodies; real detail logged server-side.
5. **Enforcement** — hook the one seam (e.g. the checkout/order/admin path). Fail closed.
6. **Wiring** — propagate any new field through to the read models the client needs (e.g. add it to
   the `/me`-equivalent so the UI can read it without a second call).

### 3.5 Test with the code, not after
- **Unit** for domain logic (fast, no DB).
- **Integration** for store/API against a real store, but **skip cleanly when the dependency is
  absent** (`if DATABASE_URL == "" { skip }`) so the suite still runs in a bare CI.
- Cover the **security cases explicitly**: the thing is blocked when it should be, allowed when it
  should be, and the exempt path stays exempt.

### 3.6 Wire exactly one live path
Frontend/CLI: add the BFF/proxy route, the small data hook/composable, and wire **one** screen or
command to the real endpoint. Read real state; render loading/empty/error states; no mock arrays.

### 3.7 Prove it live (non-negotiable)
- Bring up the real app against the real backend.
- Drive the actual path end-to-end (signup → action → see the effect).
- **Look at the artifact**: screenshot the screen and read it; dump the JSON; check the audit row
  shows the expected sequence. Keep a tiny screenshot/e2e harness in the repo for this.
- If you had to fight the environment to get it running (packages, env vars, a driver), capture that
  so the next person doesn't rediscover it.

### 3.8 Review gate
Run the review skills before merge: a general **code review**, a **simplify** pass, and a
**security review** on any diff touching credits/orders/auth/billing/customer-data. Trace data flow
from untrusted input to sensitive sink. Confirm fail-closed, parameterized queries, tenant/owner
scoping, minimal PII in logs/audit.

### 3.9 Ship
See §5.

---

## 4. Engineering standards (the always-on rules)

- **Every function gets a doc comment** on the line(s) directly above it: *what* it does and *why*.
  CI-enforce it if you can.
- **Money/credits use fixed-point**, never floats (`NUMERIC(20,6)` in SQL, a decimal lib in code).
- **Mutations are idempotent.** Key writes on a stable id so a retry/replay can't double-apply.
  Append-only ledgers with a hash chain where integrity matters.
- **Isolation flag is sacred.** If you have a paper/real (or test/prod, draft/published) split, every
  record carries the flag and the two states **never mix**. Internal/system accounts never transact
  against customer state.
- **Audit everything** that touches money, auth, or admin actions: actor, action, target, before/
  after, timestamp, isolation flag. Keep PII *out* of logs and audit `after` blobs — store only what's
  needed (e.g. country/type, not full legal name).
- **No secrets in code.** Config from env; secrets from a vault/sealed store.
- **Generic errors to clients, detailed logs server-side** (no info disclosure / enumeration oracles).
- **Tests + migrations ship in the same change as the code**, never "later".

> **[adapt]** Encode these in a linter config + pre-commit + a CI job so they're enforced, not
> aspirational. Put the per-language formatting rules in `ENGINEERING_STANDARDS.md`.

---

## 5. Versioning, git, and what not to commit

- **Per-milestone SemVer:** `v0.<milestone>.<patch>`. A feature within a milestone bumps the patch;
  closing a milestone is the minor. Tag every shippable slice.
- **Workflow:** never commit straight to the default branch. `branch → commit → checkout main →
  merge --no-ff → tag -a → push origin main → push origin <tag> → delete branch`. The `--no-ff`
  keeps each feature a legible bubble in history.
- **Commit messages** describe the change + the *why* + how it was verified. End with the standard
  co-author trailer if pair-built with an assistant.
- **Stage explicitly.** Add only the files for *this* change. Leave the user's working-notes docs,
  spreadsheets, and scratch files uncommitted — know your repo's "don't touch" set and never sweep
  them in with `git add -A`.
- **One CHANGELOG entry per tag**, in Keep-a-Changelog style: what changed, why, and a **Verified**
  line stating how you proved it (tests + the live check).

---

## 6. Deploy & local-prove loop (container/k8s flavour)

- **Migrations run before the service starts** (an init step over an ordered file list). Keep them
  idempotent so re-runs are safe. When you add a migration, add it to **both** the file list **and**
  the bundle the init step reads (the ConfigMap/volume) — a half-update crashloops the init.
- **Local cluster (k3d/kind/minikube):** build image → import into the cluster → point the deployment
  at it → restart → wait for rollout. **[gotcha]** if a dev-tool (Tilt/Skaffold) originally created
  the deployment, a plain `rollout restart` reuses its pinned image digest — you must explicitly
  `set image` to your freshly-imported tag.
- **Reach services via port-forward** for local e2e; health-check before driving.
- Verify the live endpoint exists+auth-guards (a `401` from an unauth'd call is a *good* sign the
  route is wired and protected).

> **[adapt]** Swap in your platform's equivalents (compose, serverless, a PaaS). The invariant is:
> migration-before-start, idempotent, and a repeatable "build → run the real thing → drive it" path.

---

## 7. Definition of done (check every item)

- [ ] Compiles; linters clean.
- [ ] Tests for new logic pass (unit + integration); they shipped *with* the code.
- [ ] Contract honored — no undocumented interface drift.
- [ ] Every new function documented.
- [ ] Money is fixed-point; mutations idempotent; isolation flag respected.
- [ ] Audit/log hooks present where money/auth/admin involved; no PII/secrets leaked.
- [ ] Security review clean on sensitive surfaces; enforcement fails closed.
- [ ] Wired **live** (no mock) and **proven by running the real app** (screenshot/JSON/audit checked).
- [ ] Committed, merged `--no-ff`, tagged, pushed; CHANGELOG updated with a Verified line.

---

## 8. Continuity & memory (so the method survives across sessions)

- Keep a small, durable **memory** of *non-obvious* facts: gotchas, environment quirks, decisions and
  their *why*, and corrections you've received. Don't record what the code/git history already says.
- Convert relative dates to absolute when you write them down.
- Re-verify a remembered detail (a file path, a flag) before relying on it — memories reflect the
  moment they were written.

---

## 9. Anti-patterns this method exists to prevent

- **Mock screens that demo well and rot.** → live-only + honest labels.
- **Big-bang horizontal layers** (all the schemas, then all the APIs, then all the UI) that never meet.
  → thin vertical slices that are usable at each tag.
- **Interface drift** between two parts built in parallel. → contract-first, single owner.
- **"It compiles, ship it."** → drive the real path and look at the result.
- **Allow-on-error gates.** → fail closed.
- **Lost work.** → commit/tag/push the slice the same session.
- **Floating-point money / non-idempotent writes / mixed test+real state.** → the always-on rules in §4.

---

*Copy this file into a new project, rewrite §0's domain nouns and the **[adapt]** blocks for your
stack, and the loop in §3 carries over unchanged.*
