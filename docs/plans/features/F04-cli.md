# F04 — `exascale` CLI

> Ship in **Milestone 1** (v0) → **Milestone 6** (v1.0 release). Owner: `platform-core`.

## Spec

The `exascale` CLI is the **primary platform-side interface** for AI engineers and the unified
client surface for everything a customer can do.

Commands (final v1.0 set):

```
exascale login | logout | whoami
exascale config get|set <key> [value]

exascale credits balance | purchase --amount $X | convert --from <ai> --to <text> --amount X
                | transactions

exascale infer chat -m <model> | completions | embeddings | audio transcribe | image generate
exascale catalog list | inspect <model>

exascale gpu create --type h100 --count N
exascale gpu list | stop <id> | start <id> | rm <id>
exascale train submit --image ... --gpus 256 --slurm-script ./run.sh
exascale cluster create --gpus 64 --network infiniband

exascale billing today | this-month | alerts set --limit $X
exascale keys create | list | revoke <id>
exascale audit log
exascale trade quote | buy | sell | orders         # Phase 2 — returns "trading paused" stub in Phase 1
```

Distribution: brew (`brew install exascale/tap/exascale`), apt
(`apt install exascale`), pip (`pip install exascale`), direct binary
(`curl -sSL https://exascale.io/install.sh | sh`).

## Owning agent

`platform-core` (CLI code + release). `infra-sre` owns the distribution infra (apt repo, Homebrew
tap, install script CDN).

## Contracts consumed / produced

### Produces
- None new — CLI is a client.
- `exascale --help` is the documented surface; every new command needs a corresponding OpenAPI
  route in `docs/contracts/openapi/`.

### Consumes
- `openapi/platform-core.yaml` (auth, billing).
- `openapi/credit.yaml`, `openapi/inference.yaml`, `openapi/compute.yaml`, `openapi/index.yaml`,
  `openapi/trading.yaml` (Phase 2 — but the stub commands are wired).

## Dependencies

- F01 (infra: release CDN + apt repo + brew tap).
- F02 (auth).
- F03 (orgs, optional sub-account context switch).
- F05 (ledger balance/transactions endpoints).
- F08/F12 (inference + compute commands wire to those APIs).

## Sync points

- M1 — `login`, `whoami`, `credits balance` work end-to-end.
- M2 — `infer chat`, `credits purchase`, sub-5-min target met.
- M3 — `gpu create/list/stop`, `billing today`, `credits convert`.
- M4 — `keys`, `audit log`, sub-account context switch (`--sub-account <name>`).
- M6 — v1.0 release across all distribution channels.

## Acceptance criteria

- [ ] Installable via brew, apt, pip, binary.
- [ ] `exascale login` opens browser OAuth and lands a token at `~/.exascale/credentials`.
- [ ] Every command has `--help` matching the OpenAPI documentation.
- [ ] Sub-5-min flow: install → login → buy credits → first inference call (Playwright via
      headless CLI in CI).
- [ ] OpenAPI drift check: every command maps to a documented endpoint.
- [ ] Phase 2 commands return a clean "trading paused" message in Phase 1.

## Milestone

- M1 (Gate 1): v0 (login + whoami + credits balance).
- M2 (Gate 2): infer + credits purchase.
- M3 (Gate 3): gpu + billing + credits convert.
- M6 (Gate 6): v1.0 release.
