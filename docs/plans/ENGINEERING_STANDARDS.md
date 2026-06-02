# Engineering Standards

> The cross-cutting rulebook **every agent follows** when writing code. This is how a 13-agent
> build stays coherent: same conventions, same quality bar, same security posture, same comment
> discipline, everywhere. Enforced by tooling (`.golangci.yml`, `.pre-commit-config.yaml`,
> `.editorconfig`, CI) and by the review skills (§9).
>
> If this conflicts with `CLAUDE.md`, `CLAUDE.md` wins. If it conflicts with a contract in
> `docs/contracts/`, the contract wins.

---

## 1. The non-negotiables (every language, every service)

1. **Every function carries a doc comment on the line(s) directly above it** explaining *what it
   does and why* — not how. See §3 for the per-language format. This is enforced in CI.
2. **No secrets in code.** Ever. Vault is the only source. `gitleaks` blocks the commit.
3. **All money/credit math is fixed-point.** `NUMERIC(20,6)` in SQL; a decimal library in Go
   (`github.com/shopspring/decimal` or equivalent — pinned by `tech-lead`); never `float64` for
   money. Lints flag float arithmetic in ledger/billing/trading paths.
4. **`is_paper` is threaded everywhere** a money/credit movement or order exists — request,
   row, and event. Never defaulted silently in prod.
5. **Idempotency keys on every mutating cross-service call.** Retries must be safe.
6. **Tests ship in the same PR as the code.** No PR merges new logic without tests.
7. **Audit hooks** on every operation touching credits, orders, or admin actions.
8. **Errors are handled, never swallowed.** No empty `catch`/`if err != nil {}` blocks.
9. **Inputs are validated at the boundary** (API handler, event consumer) before they reach
   domain logic.

---

## 2. Clean-code principles

- **Small functions, single responsibility.** If a function needs a paragraph of doc comment to
  explain, it's probably doing too much — split it.
- **Names say what they are.** `lockedCredits`, not `lc`. `chainHash`, not `h`. No abbreviations
  except the well-known (`ctx`, `id`, `db`, `tx`).
- **Domain logic is pure and IO-free** (`internal/domain/`). IO (DB, Redis, NATS, HTTP) lives at
  the edges (`internal/store/`, `internal/api/`). This keeps the core unit-testable without a
  database.
- **No dead code, no commented-out code.** Git is the history; delete it.
- **No magic numbers.** Thresholds (surveillance, risk limits, spreads, rate limits) come from
  config, not literals. Named constants for the rest.
- **Composition over cleverness.** Readable beats short. The next agent has zero context — write
  for them.
- **Match the surrounding code.** Comment density, naming, and idiom should look like the file
  it lives in.

---

## 3. The per-function comment mandate (with formats)

Every function — exported or not — gets a comment directly above it. Keep it to 1–3 lines for
simple functions; longer only where the *why* is non-obvious. State **what + why**, and call out
invariants, side effects, and failure modes when they matter.

### Go

Use a full-sentence comment starting with the function name (Go convention; `godot` + `revive`
enforce it).

```go
// DebitSubCredit atomically deducts `amount` of the given sub-credit from the tenant's balance
// and appends a hash-chained transaction row. It is idempotent on idempotencyKey: a repeated
// call with the same key returns the original result without double-debiting. Returns
// ErrInsufficientCredit if the balance would go negative.
func (l *Ledger) DebitSubCredit(ctx context.Context, tenantID uuid.UUID, ct CreditType, amount decimal.Decimal, idempotencyKey string) (Transaction, error) {
    ...
}
```

### TypeScript / Vue

Use a JSDoc block. For Vue composables and components, document the contract (inputs, returned
state, side effects).

```ts
/**
 * useWallet — reactive access to the signed-in tenant's credit balances.
 * Polls /v1/credits/balances and exposes { balances, loading, error, refresh }.
 * Switches data source by EXASCALE_API_MODE (mock | local | staging) with zero call-site change.
 */
export function useWallet() {
  ...
}
```

### Python (inference runtime)

Use a docstring (PEP 257). The first line is a summary; add why/invariants below if needed.

```python
def pack_models(gpu: GpuSpec, candidates: list[Model]) -> list[Model]:
    """Choose which models co-locate on one GPU without exceeding measured peak VRAM.

    Greedy by popularity; keeps >5% VRAM headroom so a burst doesn't OOM. Returns the
    subset that fits, most-popular first.
    """
    ...
```

### SQL (migrations)

A comment block at the top of every migration: what it changes, why, and the rollback note.

```sql
-- 0007_add_idempotency_key.sql
-- Adds idempotency_key to credit_transactions so retried debits don't double-apply.
-- Rollback: drop the column + unique index (forward-compensating, never destructive in prod).
```

---

## 4. Per-language conventions

### Go (services + CLI)

- Format with `gofmt` + `goimports`. Lint with `golangci-lint` (config: `.golangci.yml`).
- `context.Context` is the first parameter of any function doing IO or that can be cancelled.
- Wrap errors with context: `fmt.Errorf("debit sub-credit: %w", err)`. Sentinel errors for the
  domain (`var ErrInsufficientCredit = errors.New(...)`).
- No `panic` in request paths. `panic` only for truly unrecoverable init failures.
- Concurrency: prefer channels/`errgroup`; guard shared state with mutexes; run `go test -race`
  in CI. The matching engine is single-threaded per product **by design** — don't "optimize" it
  into a race.
- Table-driven tests; property-based tests (`testing/quick` or `rapid`) for ledger + matching
  invariants.
- Pinned dependencies; `go mod tidy` clean; no `replace` directives in main.

### TypeScript / Vue (frontend)

- **Conform to `docs/plans/DESIGN_SYSTEM.md`** — it is the canonical visual spec (tokens, type
  scale, spacing, number formatting, aesthetic guardrails). Every frontend change follows it.
- `<script setup lang="ts">` + Composition API. Strict TS (`strict: true`, already set).
- No hardcoded colors/sizes/fonts — `var(--token)` or a component (raw hex/rgb in components is
  blocked by the `design-tokens-guard` pre-commit hook). Numbers: mono + `tabular-nums`.
- One data layer (`useApi()` composable) hides mock-vs-real; switching backends = zero UI change.
- Charts in `.client.vue` (chart libs are SSR-unsafe).
- ESLint + Prettier; `vue-tsc` typecheck in CI (turn on once components stabilize, per nuxt.config).
- No `any` without a `// eslint-disable` + reason. Prefer generated types from the OpenAPI contracts.

### Python (inference runtime)

- Format with `ruff format`; lint with `ruff`. Type hints required; `mypy` in CI.
- FastAPI for the HTTP surface; Pydantic models validate every request.
- No blocking calls on the event loop; vLLM calls run in the appropriate executor.
- Pinned `requirements.txt` (or `uv` lock); reproducible image builds.

---

## 5. Security practices (baked in, not bolted on)

- **Auth at the edge.** Every handler validates the JWT/API key → resolves `tenant_id`,
  `sub_account_id`, scopes. No endpoint trusts an unauthenticated caller. Services behind Kong
  still re-verify — defense in depth.
- **Authorization per tenant.** Every query is scoped by `tenant_id` (and `sub_account_id` where
  applicable). Cross-tenant access is a security bug; `security-compliance` pen-tests it.
- **Input validation at the boundary.** Reject malformed input with 4xx before domain logic.
  Parameterized queries only — no string-built SQL.
- **Least privilege.** Service DB users get only the grants they need. K8s ServiceAccounts scoped.
  Network policies allow only the calls in `CONTRACTS.md` §6.
- **Secrets via Vault**, injected at runtime. `.env.example` documents names, never values.
- **PII minimization.** Don't log prompts, full names, or payment details. Usage events carry
  token *counts*, not prompt text. `security-compliance` flags PII in logs.
- **Encryption** at rest (DB, object storage) and in transit (TLS 1.3 external, mTLS internal).
- **Dependency hygiene.** `trivy` scans images; `govulncheck`/`npm audit`/`pip-audit` in CI.
  No high-severity CVEs ship.
- **Supply-chain.** Pinned deps + SBOM per image (`DEPLOYMENT.md` §8). No `curl | sh` in build
  steps without a checksum.
- **Rate limiting + abuse** at the gateway and per tenant (`surveillance` basic path).

The `/security-review` skill runs on every PR touching credits, orders, auth, or customer data —
see §9.

---

## 6. Testing standards

| Level | What | Where | Gate |
|---|---|---|---|
| Unit | Pure domain logic | `internal/domain/*_test.go`, Vue/TS unit | `make test` |
| Property | Ledger + matching invariants (no double-spend, balance == replay) | service test dirs | `make test` |
| Integration | Service against real Postgres/Redis/NATS | `make test-integ` (compose) | CI on PR |
| Contract | Service responses conform to its OpenAPI | per service | CI on PR |
| E2E | Signup → top up → first inference → debit → GPU job → debit (the 5-min flow) | `make test-e2e` (API harness, `scripts/e2e.sh`; Playwright variant TBD) | CI pre-staging |
| Load | Perf targets (match P99, instance start, ledger throughput) | `make test-load` | pre-prod, scheduled |

- **Coverage isn't a vanity metric** — but ledger, matching, conversion, and auth paths target
  high coverage because bugs there are financial or security incidents.
- **Fault injection** for the ledger (kill mid-transaction; assert nothing leaked) and idempotency
  (retry storms).
- **No flaky tests merged.** A flaky test is a broken test.

---

## 7. Observability hooks (every service, from day one)

- **Structured logs** — one JSON event per line with `trace_id`, `tenant_id`, `is_paper`,
  `service`, `level`, `msg`. Never log secrets or PII.
- **Metrics** — Prometheus `/metrics`; RED method (Rate, Errors, Duration) per endpoint + domain
  counters (debits/sec, matches/sec, prints published).
- **Traces** — OTLP spans across service hops; propagate `trace_id` from the gateway.
- **Health** — `/healthz` (liveness) + `/readyz` (readiness, checks deps).
- **Alerts** — wired to the SLOs (`infra-sre` owns). Ledger hash-chain mismatch and missed index
  print are zero-tolerance pages.

---

## 8. Git, PR, and "nothing breaks" discipline

- **Branch per change**, never commit straight to `main`. Conventional-commit messages
  (`feat(credit-ledger): ...`, `fix(inference): ...`, `contracts(propose): ...`).
- **Small PRs.** One concern. Easier to review, safer to revert.
- **Every PR**: green CI, tests for new logic, doc comments present, contracts honored, owning
  agent + `tech-lead` (if a contract moved) review.
- **CODEOWNERS** (`.github/CODEOWNERS`) routes each path to its owning agent — changes can't land
  in someone's service without their review. This is the tracking layer.
- **Backwards compatibility.** A contract change is MAJOR/MINOR/PATCH (`CONTRACTS.md` §3); breaking
  changes dual-serve for one cycle. Migrations ship a tested rollback.
- **Feature flags** for anything risky or Phase-2-adjacent. The exchange switch-on is flags, not
  a rewrite — keep it that way: gate dormant code behind a flag, default off.
- **Revertable always.** `kubectl rollout undo` works; ledger "rollback" is a forward compensating
  migration (it's append-only).

---

## 9. The review-skill workflow (use the tools, every PR)

The orchestrator and agents use these skills as standard practice — they are the "best-practice +
security" layer the build runs through:

| Skill | When | What it does |
|---|---|---|
| `/code-review` | Before opening a PR, and on the PR diff | Correctness bugs + reuse/simplify/efficiency findings at chosen effort. Use `--comment` to post inline, `--fix` to apply. |
| `/security-review` | **Mandatory** on any diff touching credits, orders, auth, billing, or customer data | Full security review of pending changes on the branch. |
| `/simplify` | After a feature works, before merge | Applies reuse/simplification/altitude cleanups (quality only). |
| `/verify` | After a change that should be observable | Runs the app and confirms the behavior actually happens (not just tests pass). |

Recommended per-PR loop:

```
1. Write code + tests (with doc comments).          → §1–§7
2. /code-review   (fix correctness findings)
3. /simplify      (tidy without changing behavior)
4. /security-review   (if credits/orders/auth/PII touched)   ← BLOCKER if it finds one
5. /verify        (confirm it works in the running app, where applicable)
6. Open PR → CODEOWNERS review → tech-lead if a contract moved → merge.
```

`security-compliance` (the agent) independently reviews the same surfaces and can block. The
skill is the fast self-check; the agent is the gate.

---

## 10. Definition of done (the full checklist)

A change is done when **all** of these hold (superset of `CLAUDE.md`'s DoD):

- [ ] Compiles / typechecks / builds.
- [ ] Tests pass — unit + integration + (where relevant) contract/e2e. New logic has tests.
- [ ] **Every function has a doc comment** (CI-enforced).
- [ ] Linters clean (`golangci-lint` / `ruff` / `eslint`); formatted.
- [ ] Contracts honored — no undocumented interface drift; contract changes went through `tech-lead`.
- [ ] `is_paper` threaded correctly through requests, rows, and events.
- [ ] Money/credit math is fixed-point — no floats.
- [ ] Idempotency keys on mutating cross-service calls.
- [ ] Audit hooks present where credits/orders/admin are involved.
- [ ] No secrets in code (`gitleaks` clean); no high-severity CVEs (`trivy`/`govulncheck`).
- [ ] Observability hooks present (logs/metrics/traces/health).
- [ ] `/security-review` clean (if it touched credits/orders/auth/PII).
- [ ] Dockerfile + K8s manifests + dashboard + runbook updated if a new surface shipped
      (`DEPLOYMENT.md` §11).
- [ ] Matches the spec (`docs/re/phase6_v2.md`) and the GTM pivot
      (`docs/re/exascale_gtm_focus_update.md`).
- [ ] The feature doc in `docs/plans/features/` updated if the surface changed.

---

## 11. How tooling enforces this (so it isn't just prose)

| Rule | Enforced by |
|---|---|
| Per-function doc comments (Go) | `.golangci.yml` → `revive` (exported), `godot`, custom config |
| Formatting | `gofmt`/`goimports`, `prettier`, `ruff format`, `.editorconfig` |
| Lint | `golangci-lint`, `eslint`, `ruff` (in `.pre-commit-config.yaml` + CI) |
| No secrets | `gitleaks` (pre-commit + CI) |
| Vulnerabilities | `govulncheck`, `npm audit`, `pip-audit`, `trivy` (CI) |
| No floats in money paths | `golangci-lint` custom + `security-compliance` review |
| `is_paper` present | schema check + `security-compliance` `make audit-review` |
| Banned regulatory words (frontend) | `make audit-review` lint (`F22`) |
| Ownership routing | `.github/CODEOWNERS` |
| Tests required | CI gate (coverage diff + new-logic check) |

`infra-sre` owns the CI wiring; `tech-lead` owns the contract-check stage; `security-compliance`
owns the security gates.
