# services/

Go services, one directory each (skeleton in `docs/plans/REPO_LAYOUT.md` §2). Scaffolded per
feature by the owning agent — **not** all at once.

| dir | owner | feature |
|---|---|---|
| `platform-core/` | platform-core | F02/F03/F06 |
| `credit-ledger/` | credit-ledger | F05/F07 |
| `inference-gateway/` · `inference-runtime/` | inference-ml | F08/F09 |
| `compute-control/` | compute-platform | F12 |
| `supply-service/` | settlement-trust | F17/F18 |
| `index-service/` | index-service | KW01 (keep-warm) |
| `matching-engine/` · `market-maker/` · `surveillance/` | trading agents | KW03/04/05 (paused) |

Start one with `/ex-start <feature>`. No cross-service imports of another service's `internal/` —
couple only through `docs/contracts/`.
