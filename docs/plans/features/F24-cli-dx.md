# F24 — CLI developer experience (Claude-Code-grade UX)

> Owner: `platform-core` (owns the `exascale` CLI, built in F04). Milestone: **M3→M6** (incremental —
> ships in slices on top of the live F04 binary). Builds on F04, F08 (streaming), F13 (`gpu`).

## Spec

Make the `exascale` CLI feel like a modern, conversational developer tool (Claude Code / `gh` / `stripe`
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

### Phase 1 — "feels alive" (no heavy deps, highest impact-per-effort)
- [ ] **Streaming inference** — set `stream:true` and render the gateway's SSE token-by-token (the
      typewriter feel). Non-stream path kept for `--json` / pipes.
- [ ] **Spinners** for in-flight work ("Thinking…", "Provisioning…", "Minting key…"); auto-off when not a TTY.
- [ ] **Semantic colour** — green `▲` / red `▼` on credit deltas, dim secondary, bold headers; honour `NO_COLOR`.
- [ ] **Actionable errors** — map upstream codes to next steps (`402` → "buy credits", `401` → "run `exascale login`").
- [ ] **`--json`** global flag — machine-readable output for scripting/CI (`… --json | jq`).
- [ ] **Per-command help + examples**; did-you-mean on unknown commands.
- [ ] **Confirm destructive ops** — `gpu delete`, `keys revoke` → `[y/N]` (skippable with `--yes`).

### Phase 2 — the interactive chat REPL (the headline Claude-Code-like feature)
- [ ] `exascale chat` — a persistent, multi-turn, **streaming** session.
- [ ] Status prompt with context: `llama-3.1-8b · 982 text ▸`.
- [ ] **Slash commands**: `/model`, `/balance`, `/cost`, `/clear`, `/save`, `/exit`.
- [ ] Per-turn live cost meter (`+12.5 text · 982 left`); light markdown rendering of model output.
- [ ] Built on **charm `bubbletea` + `lipgloss` + `bubbles`** (restrained styling) — *or* stdlib-only
      (decision below). Single static binary either way.

### Phase 3 — polish & productivity
- [ ] **Shell completion** — `exascale completion {bash|zsh|fish}`.
- [ ] **Browser/device login** (`exascale login` opens a browser + polls), pairing with F02 OAuth — no typed password.
- [ ] **Config profiles** — `exascale --profile staging …` (multi-environment).
- [ ] **`exascale status`** — one-line account · mode · balances · running GPUs; richer `whoami`.

## Open decisions

- **TUI library:** charm (`bubbletea`/`lipgloss`) for the Phase-2 REPL (heavier dep, polished) **vs.**
  stdlib-only (lighter binary, plainer). Ships as one binary regardless.
- **Distribution** (out of scope here, tracked in F04): brew / apt / `go install` / static release
  binaries land M6.

## Acceptance criteria

- [ ] `exascale infer chat` streams tokens live; `--json` returns the full non-streamed object.
- [ ] Credit deltas render green `▲` / red `▼`; colour disabled off-TTY and under `NO_COLOR`.
- [ ] Errors carry a next step (402/401/404 mapped); destructive commands confirm.
- [ ] `exascale chat` holds a multi-turn streaming conversation with slash commands + a live cost meter.
- [ ] Every interactive feature has a non-interactive/`--json` equivalent (CI-safe).
- [ ] No new service contract; per-function doc comments; tests for the formatter + SSE parser.

## Milestone

- M3→M6, incremental. Phase 1 is the recommended first slice (transforms the feel, low risk, no big deps).
