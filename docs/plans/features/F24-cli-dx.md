# F24 — CLI developer experience (Claude-Code-grade UX)

> Owner: `platform-core` (owns the `1trade` CLI, built in F04). Milestone: **M3→M6** (incremental —
> ships in slices on top of the live F04 binary). Builds on F04, F08 (streaming), F13 (`gpu`).

## Spec

Make the `1trade` CLI feel like a modern, conversational developer tool (Claude Code / `gh` / `stripe`
calibre) — **streaming, interactive, legible** — without leaving the design system. It stays
**institutional / Bloomberg, not flashy**: semantic colour (green `▲` / red `▼`), mono + tabular
numbers, restraint. Colour auto-disables off-TTY and under `NO_COLOR`; every interactive feature has a
non-interactive equivalent for scripts/CI.

Today the CLI (F04, v0.1.4) is a thin stdlib `flag` + `fmt` client: correct and live, but spartan —
plain tables, blocking requests, no streaming, terse errors. F24 is the **DX layer** over it.

## Owning agent

`platform-core`.

## Contracts consumed / produced

- **Consumes** `openapi/inference.yaml` (the gateway **SSE** stream — `stream:true`), `credit.yaml`,
  `compute.yaml`, `platform-core.yaml`. **Produces no new service contract** — pure client-side DX.
- May propose a small `platform-core` device/browser-login endpoint for Phase 3 auth (via `/ex-contract`).

## Dependencies

- F04 (CLI base), F08 (streaming inference), F13 (`gpu`), F02 (auth; OAuth for Phase 3 browser login).

## Phases (each ships independently)

### Phase 1 — "feels alive" (no heavy deps, highest impact-per-effort) — **shipped (v0.3.x)**
- [x] **Streaming inference** — `infer chat` sets `stream:true` and renders the gateway SSE token-by-token
      (`internal/client/stream.go`); `--json` keeps the full non-streamed object for pipes/CI.
- [x] **Spinners** — `internal/ui.StartSpinner` ("Provisioning…" on `gpu create`); auto-off when not a TTY.
- [x] **Semantic colour** — `internal/ui` (green `▲` / red `▼` via `Delta`, bold headers, dim secondary);
      auto-off off-TTY and under `NO_COLOR` (`SetColor` for tests).
- [x] **Actionable errors** — `ui.Hint` maps codes to next steps (`402`→top up, `401`→login,
      `email_unverified`→verify, 404/429/5xx…), printed under the error in `main`.
- [x] **`--json`** — on `infer chat` (the streaming command); other commands' tabular output stays pipe-friendly.
- [ ] **Per-command help + examples**; did-you-mean on unknown commands. *(next)*
- [x] **Confirm destructive ops** — `gpu delete`, `keys revoke` → `[y/N]` (skippable with `--yes`; non-TTY aborts).
- Tests: SSE parser (`stream_test.go`) + formatter/hint (`ui_test.go`). golangci-lint clean.

### Phase 2 — the interactive chat REPL (the headline Claude-Code-like feature) — **shipped (v0.3.x)**
- [x] `1trade chat` — a persistent, multi-turn, **streaming** session (`cmd/1trade/chat.go`).
- [x] Status prompt with context: `llama-3.1-8b · 982 text ▸` (model cyan, balance dim).
- [x] **Slash commands**: `/model`, `/balance`, `/cost`, `/clear`, `/save`, `/exit` (+ `/help`).
- [x] Per-turn live cost meter (`▼ 12.5 text · 982 left · N tok`), balance tracked locally + reconciled by `/balance`.
- [ ] Light markdown rendering of model output. *(deferred — raw streamed text for now)*
- [x] **Decision: stdlib-only** (no charm dep) — `bufio` input + the F24 streaming client; single static binary.
      Tests: slash-parser + price parser.

### Phase 3 — polish & productivity
- [ ] **Shell completion** — `1trade completion {bash|zsh|fish}`.
- [ ] **Browser/device login** (`1trade login` opens a browser + polls), pairing with F02 OAuth — no typed password.
- [ ] **Config profiles** — `1trade --profile staging …` (multi-environment).
- [ ] **`1trade status`** — one-line account · mode · balances · running GPUs; richer `whoami`.

## Open decisions

- **TUI library:** ✅ **resolved → stdlib-only** for the Phase-2 REPL — no charm/bubbletea dep, lighter
  static binary, and the streaming SSE client already gives the live feel. Revisit only if Phase-3 needs
  full-screen TUI (e.g. a model picker).
- **Distribution** — ✅ **done early**: one-line install `curl -fsSL https://sandbox.1trade.ai/cli/install.sh | sh`.
  Cross-compiled static binaries (linux/darwin/windows × amd64/arm64, sandbox api URL baked in via ldflags)
  are built into the web image and served at `/cli/` (`scripts/build-cli-dist.sh` + `scripts/cli-install.sh`,
  staged by the release pipeline). brew/apt remain a later nicety.

## Acceptance criteria

- [ ] `1trade infer chat` streams tokens live; `--json` returns the full non-streamed object.
- [ ] Credit deltas render green `▲` / red `▼`; colour disabled off-TTY and under `NO_COLOR`.
- [ ] Errors carry a next step (402/401/404 mapped); destructive commands confirm.
- [ ] `1trade chat` holds a multi-turn streaming conversation with slash commands + a live cost meter.
- [ ] Every interactive feature has a non-interactive/`--json` equivalent (CI-safe).
- [ ] No new service contract; per-function doc comments; tests for the formatter + SSE parser.

## Milestone

- M3→M6, incremental. Phase 1 is the recommended first slice (transforms the feel, low risk, no big deps).
