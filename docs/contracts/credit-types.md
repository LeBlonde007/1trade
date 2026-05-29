# Credit types — the canonical enum (v1.0)

> **This is the single source of truth for every credit type in Exascale.** Every service reads
> it; no service hardcodes a credit type that isn't here. Adding/removing a type is a `tech-lead`
> change + a coordinated rollout across `credit-ledger` + `inference-ml` + `compute-platform` +
> `platform-core` (billing/UI) + `trading-frontend`. Owned by `tech-lead`.
>
> Status: **v1.0**, 2026-05-29. Consumed by: all services. See `docs/plans/CONTRACTS.md`.

---

## 1. The enum

`credit_type` is a lowercase string from exactly this set:

| `credit_type` | kind | unit | consumed by | tradeable (Phase 2) |
|---|---|---|---|---|
| `ai_index` | index | 1 AI credit | the umbrella unit; converts into any sub/GPU credit | yes (the index spot) |
| `text` | sub | 1 text credit | text + code LLM inference | yes |
| `speech` | sub | 1 speech credit | STT + TTS inference | yes |
| `image` | sub | 1 image credit | image generation + understanding | yes |
| `video` | sub | 1 video credit | video generation | yes |
| `embeddings` | sub | 1 embeddings credit | embeddings + reranker inference | yes |
| `gpu_h100` | gpu | 1 H100-80GB-GPU-hour | H100 compute (on-demand + reserved) | yes (per-tier spot) |
| `gpu_h200` | gpu | 1 H200-GPU-hour | H200 compute (on-demand + reserved) | yes (per-tier spot) |

Notes:
- **`kind`** ∈ {`index`, `sub`, `gpu`} — used by the ledger + UI to group balances.
- **Tradeable is Phase 2 only.** In Phase 1 every credit is **prepaid + redeemable, not tradeable**
  (regulatory framing — `docs/plans/features/F22-licensing-track.md`). The `tradeable` column
  documents the *eventual* exchange surface; it is dormant until the license clears.
- New GPU tiers (e.g. `gpu_b200`) are out of v1 scope; add here via `tech-lead` when introduced.

## 2. Modality → sub-credit mapping (for inference debits)

`inference-ml` debits the sub-credit for the model's modality. This table is the mapping the
inference gateway uses; it must stay consistent with the catalog.

| modality | debits `credit_type` |
|---|---|
| text generation, chat, code | `text` |
| speech-to-text, text-to-speech | `speech` |
| image generation, image understanding (vision) | `image` |
| video generation | `video` |
| embeddings, reranking | `embeddings` |

## 3. Conversion policy (ADR-0002 — provisional)

> **Provisional default, pending counsel sign-off (F22).** See `docs/plans/DECISIONS.md` ADR-0002.

- **Direction:** **bidirectional** — `ai_index` ↔ any `sub`, and `ai_index` ↔ any `gpu`.
- **Spread:** a **1% house spread** is applied on every conversion (the house keeps 1% of the
  converted value). The spread is configuration, not a hardcoded constant (`credit-ledger` reads
  it from the conversion-rate source).
- **Sub ↔ sub** and **gpu ↔ gpu** are **not** direct — they route through `ai_index`
  (two conversions, spread applied on each leg). This keeps a single price anchor.
- **Rates:** the per-pair conversion rate is published via `GET /v1/credits/conversion-rates`
  (`openapi/credit.yaml`). In Phase 1 the rate source is a ledger-owned config table; it may later
  be anchored to the index.
- **Framing:** customer-facing copy says **"convert" / "redeem,"** never "trade" / "exchange,"
  while the exchange is paused. `security-compliance` enforces this.

If counsel requires the simpler one-way model (sub/gpu → `ai_index` only, no buy-back), that is a
`tech-lead` change here + an update to `openapi/credit.yaml` `convert`.

## 4. Invariants every service must honor

- **Fixed-point only.** Amounts are `NUMERIC(20,6)` in SQL and decimal in Go/JSON (serialized as
  decimal strings, never floats). See `schemas/types.sql`.
- **`is_paper` travels with every balance, transaction, and event** carrying a credit type.
- **GPU credits are backed.** `settlement-trust` keeps outstanding `gpu_*` credits ≤ attested
  reserved capacity for that class (≥100% backing). Minting beyond backing is refused.
- **Append-only ledger.** Movements are hash-chained; corrections are compensating entries, never
  edits.

## 5. Phase 2 (dormant) types

No additional credit types are introduced for the exchange — the same enum trades. The only change
at switch-on is the `tradeable` flag going live per product (`matching-engine` + `index-service`).
