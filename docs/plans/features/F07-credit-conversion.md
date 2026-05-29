# F07 — Credit conversion

> Ship in **Milestone 2** (AI ↔ text) → **Milestone 3** (all modalities + GPU tiers). Owner: `credit-ledger`.

## Spec

Convert between credit types at the published rate:

- AI credit ↔ sub-credit (text / speech / image / video / embeddings).
- AI credit ↔ GPU credit tiers (`gpu_h100`, `gpu_h200`).

Open questions to resolve with `tech-lead` (per Phase 6 §17 q1):

1. **Bidirectional with spread** (default): convert in either direction; house takes a 1% spread.
2. **One-way (sub-credit → AI only)** to simplify regulatory framing.

Recommended default: bidirectional with 1% spread, awaiting `tech-lead` final call. Decision
recorded in `docs/contracts/credit-types.md`.

Conversion is an atomic ledger op:

```
POST /v1/credits/convert { from: "ai_index", to: "text", amount: 100, idempotency_key }
  → credit-ledger:
    BEGIN;
    SELECT balance FOR UPDATE WHERE credit_type='ai_index';
    rate = SELECT rate FROM credit_conversion_rates WHERE from='ai_index' AND to='text';
    new_text = floor(amount * rate * (1 - spread));
    UPDATE balance ai_index -= amount;
    UPDATE balance text += new_text;
    INSERT tx { operation: 'conversion', ... } (one row per side, chained);
    COMMIT;
```

## Owning agent

`credit-ledger`. Conversion **policy** (rates, spread, direction) is a contract maintained by
`tech-lead`; ledger executes against it.

## Contracts consumed / produced

### Produces
- `openapi/credit.yaml` — `POST /v1/credits/convert`, `GET /v1/credits/conversion-rates`.
- Conversion-rate publication shape (lives in `credit.yaml` for now).

### Consumes
- `credit-types.md`.
- Conversion-rate source (a ledger-owned table for v1; later possibly an oracle).

## Dependencies

- F05 (ledger).
- `tech-lead` decision on the open question.

## Sync points

- M2 — AI ↔ text conversion live; trading-frontend wallet shows convert UI.
- M3 — all modalities + GPU tier conversions live; CLI `credits convert` works.

## Acceptance criteria

- [ ] Conversion is atomic (two transaction rows, one balance update per side, all in one DB tx).
- [ ] Hash chain extended correctly across both rows.
- [ ] Rate read from a single source — no hardcoding.
- [ ] Idempotent under retry.
- [ ] `security-compliance`: the conversion language in customer-facing copy stays "redeem" /
      "convert," not "trade" / "exchange," to preserve the prepaid-service-units framing.

## Milestone

- M2 (Gate 2): AI ↔ text.
- M3 (Gate 3): all modalities + GPU tiers.
