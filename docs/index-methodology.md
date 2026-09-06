# 1TRADE AI Credit Index — Methodology

**Version 1.2 · effective 2026-04-01 · published document**

> **Status: the index is not yet in production.** Prints served today are **simulated** and say so
> in the payload — every print carries `provisional: true` and `source: "mock"`, and every
> collection carries `is_mock: true`. Do not quote a value from this index as a market price.
>
> What *is* real today is the integrity machinery in §6: prints are hash-chained exactly as
> described, and the chain can be verified independently. That is deliberate — the audit chain is
> the part that is worthless if bolted on later, so it runs from day one (immutable commitment #2).
>
> This document is published now, before the index is live, so external review can begin against
> the methodology rather than against a fait accompli.

---

## 1. What the index measures

The **AI Credit Index** is a reference price for one unit of general-purpose AI compute, expressed
as a fixed-point USD value per AI credit.

It is a **reference** index, not a settlement price. Nothing on the 1TRADE platform is currently
settled, margined or liquidated against it.

## 2. Phase 1 inputs — why not trades

The exchange is not licensed and order entry is refused, so **no trades exist**. Computing an index
from trade prints is therefore impossible today, and inventing one would be dishonest.

Phase 1 observes real platform transactions instead:

| Observation source | What it is |
|---|---|
| Realized consumption rates | USD-equivalent actually charged per consumed sub-credit |
| Prepaid purchase prices | What customers paid per credit class, at purchase |
| Reserved-capacity transaction prices | Agreed prices on dated capacity commitments |

These are transactions the platform genuinely executes. When the venue opens, live trade prints and
`surveillance` exclusion flags join the observation set (§9) — the methodology version will change
and the change will be disclosed.

## 3. Constituents and weights

Every `credit_type` is drawn from the canonical enum in `docs/contracts/credit-types.md` §1. The
index never invents an input.

| Constituent | credit_type | Weight | Source |
|---|---|---:|---|
| Text credit consumption | `text` | 0.4200 | Realized consumption rates |
| H100 capacity cost | `gpu_h100` | 0.2000 | Reserved-capacity transaction prices |
| Image credit consumption | `image` | 0.1500 | Realized consumption rates |
| Embeddings credit consumption | `embeddings` | 0.1000 | Prepaid purchase prices |
| Speech credit consumption | `speech` | 0.0800 | Realized consumption rates |
| Video credit consumption | `video` | 0.0500 | Realized consumption rates |
| | | **1.0000** | |

Weights sum to exactly 1.0000. The live schedule, with each constituent's observation timestamp, is
served at `GET /v1/index/methodology` — the document and the API are the same schedule, so a
disagreement between them is a bug, not a matter of interpretation.

## 4. Observation window

Each print observes a **120-minute window** ending at publication. Observations outside the window
are not carried forward; a constituent with no observation in its window is excluded from that
print and counted in `excluded_count`.

## 5. Computation

1. Collect observations per constituent within the window (§4).
2. Apply the **volume floor**: an observation below `25000.000000` in notional is discarded as too
   thin to be price-forming. Discards count toward `excluded_count`.
3. Take a **95% trimmed mean** per constituent, removing the extreme tails before averaging, so a
   single unusual transaction cannot move the print.
4. Combine constituents by the §3 weights.
5. Round to 6 decimal places, fixed-point. **All index arithmetic is fixed-point** — never floating
   point — matching the ledger's money rule.

Every print reports `observation_count` and `excluded_count` so the sample behind a value is
visible, not implied.

## 6. Audit chain — how to verify a print yourself

Prints are immutable and hash-chained. For each print:

```
chain_hash = SHA256( prev_chain_hash || canonical_json(print) )
```

`canonical_json` is built with an **explicit fixed key order** and fixed-point values — never a
language's map iteration order or default float formatting, either of which would make the hash
irreproducible on another implementation:

```json
{"print_id":"…","value":"0.001003","published_at":"2026-09-06T16:00:00Z",
 "methodology_version":"1.2","observation_count":7978,"excluded_count":101,
 "provisional":true,"source":"mock"}
```

- `value` is formatted to exactly 6 decimals.
- `published_at` is RFC 3339, UTC.
- The genesis print (2025-10-15 16:00 UTC) chains from the empty string.

**To verify:** fetch a series from `GET /v1/index/history`, re-derive each `chain_hash` from the
preceding print's hash and the canonical form above, and compare. Altering any print breaks every
hash after it. The chain for a given day is identical regardless of how much history you request.

This is the same construction the credit ledger uses for its transaction chain, deliberately — one
verification routine covers both.

## 7. Publication

- **Daily at 16:00 UTC.** `next_publication_at` on every response states the next time.
- A print is marked `provisional: true` until it is finalised. Today *all* prints are provisional,
  because the index is not in production.
- Prints are never edited. A correction is issued as a **new print** citing the one it supersedes;
  the superseded print stays in the chain.

## 8. Changing this methodology

The methodology version is a published, committee-reviewed act — not a code change.

To change it: the change is proposed in writing with its rationale and expected effect on the
index, reviewed by the index committee, published here with a new version number and effective
date, and only then does `MethodologyVersion` move in
`services/matching-engine/internal/refindex/refindex.go`.

Version history:

| Version | Effective | Change |
|---|---|---|
| 1.2 | 2026-04-01 | Current. Six constituents, 120-minute window, 95% trim, 25,000 volume floor. |

**A version bump must never be silent.** A print's `methodology_version` field states which
methodology produced it, so a series spanning a change remains interpretable.

## 9. Known limitations

Stated plainly, because a methodology that hides its weaknesses is not auditable:

- **Values are simulated today.** See the banner above. `source: "mock"` marks every one.
- **No trade prints.** Phase 1 observes platform transactions only (§2); the exchange is unlicensed.
- **Thin history.** The platform is young, so early observation counts are small and the volume
  floor excludes proportionally more.
- **No independent audit yet.** The chain is verifiable by construction, but no third party has
  attested to it.
- **Single operator.** 1TRADE computes an index over its own platform's transactions. That is a
  genuine conflict; it is mitigated by publishing this methodology, the constituent weights, the
  observation counts and the audit chain — not eliminated by them.
- **Administered prices are not discovered prices — the most important limitation here.** Every
  Phase 1 observation is a transaction at a price 1TRADE set, from a published reference table
  (`billing.refUSDPerCredit`), not a price two parties negotiated. An index computed over them is
  therefore close to a restatement of that table: it would move only when we move it, and it would
  look stable for the wrong reason. **Publishing such a value as a market price would be
  misleading**, which is why the index stays labelled `mock` until either prices vary by
  transaction (negotiated, tiered, auctioned) or real trade prints exist. Storing the price at each
  transaction (below) is the prerequisite, not the solution.
- **Price history begins 2026-09-06.** Until then, purchases recorded how many credits were bought
  but never what was paid: the USD was computed at checkout and discarded. Migration 0007 records
  `unit_price_usd` and `charged_usd_cents` per purchase. Earlier rows are **not** back-filled —
  an inferred price is not an observation, and seeding the index with reconstructed history is
  precisely the failure this document exists to prevent.

## 10. Where it is served

| | |
|---|---|
| `GET /v1/index/latest` | Current print + `next_publication_at` |
| `GET /v1/index/history` | Print series (hash-chained) |
| `GET /v1/index/methodology` | Live constituent + weight schedule |

Contract: `docs/contracts/openapi/index.yaml`. Events: `docs/contracts/events/index.print.v1.yaml`.

Implementation today is `services/matching-engine/internal/refindex` — a keep-warm mock that
labels itself. Ownership moves to `index-service` (KW01) when the real computation ships; the
contract and this document do not change with it.
