# Matching engine — specification (KW03)

Status: **engine core built and tested; not wired to order entry.** `POST /v1/trading/orders` still
returns `503 EXCHANGE_PAUSED`. Opening it is the licence-gated cutover (F22), not an engineering step.

Code: `internal/engine/`. Tests: `internal/engine/*_test.go` (run `go test -race ./internal/engine/`).

---

## 1. Model

- **Books.** One book per `(product_id, is_paper)`. Paper and real-money orders live in different
  books, so they cannot meet by construction.
- **Products.** Only products in the catalog (`internal/domain`) with `tradeable: true`. The dated
  forward is listed but not tradeable (v1.5).
- **Numbers.** Every price, quantity, notional and fee is `Fixed`: an `int64` count of micro-units,
  the same scale as the ledger's `NUMERIC(20,6)`. Strings with more than six decimals are rejected,
  not rounded. `price × quantity` is computed with `math/big` and rejected at entry if it overflows.
- **Purity.** The engine reads no clock, network or randomness. Timestamps come on commands. That is
  what makes replay exact (§5).

## 2. Order types

| `order_type` | `time_in_force` | Behaviour |
|---|---|---|
| `limit` | `gtc` (default) / `day` | Matches what crosses; the remainder rests at its limit. |
| `market` | `ioc` (implied) | Matches at any price; the remainder is cancelled (`unfilled_remainder`). No price is allowed. |
| `ioc` | `ioc` (implied) | Matches up to its limit; the remainder is cancelled. |
| `fok` | `fok` (implied) | Fills completely at or inside its limit, or is cancelled (`fok_unfilled`) with no trades. |

The contract carries both fields, so they must agree. `gtc` is the contract's declared default, so on
a market/ioc/fok order it is read as "unspecified" (a generated client may send it unasked). A real
conflict, such as `order_type: ioc` with `time_in_force: day`, is a validation error, not a guess.

## 3. Matching rules

1. **Price-time priority.** Best price first. Within a price, earliest arrival first.
2. **Trades execute at the resting (maker) price.**
3. **Self-trade prevention: cancel newest.** If the next resting order belongs to the incoming
   order's tenant, the incoming remainder is cancelled (`self_trade`) and the resting order is left
   alone. A tenant can never print a trade with itself.
4. **Insider-risk rule** (phase6_v2 §12.1). Internal accounts cannot trade with customer paper
   accounts. Books are split by paper/real, and an internal order on a paper book is refused at entry
   (`ErrInternalPaper`).
5. **Fees.** Taker 1%, maker 0.5% of notional, truncated to six places (`DefaultFees`, configurable).
6. **Pre-trade risk.** `Config.Risk` can reject an order with a reason (`insufficient_credit`,
   `position_limit`, …). This is the seam for ledger balance checks and surveillance holds. It runs
   once per order, and its answer is written into the journal.
7. **Idempotency.** Re-submitting an `order_id` for the same tenant returns the current order and
   emits nothing. Re-using another tenant's `order_id` is refused. Validation failures consume no id,
   so a corrected retry with the same id works.
8. **Cancel** is tenant-scoped. Another tenant's order reads as not found, so ids leak nothing.
9. **Day expiry.** `ExpireDayCmd` cancels every resting `day` order (`day_expired`). The caller
   issues it at the session boundary.

## 4. Outputs

Every command returns its events in emission order:

- **`OrderEvent`** is shaped for `events/orders.state.v1`. There is one event per transition:
  `accepted`, one per fill on each side, and the terminal `cancelled` / `rejected`.
- **`Trade`** is shaped for `events/trades.executed.v1`, with both counterparties, fees, liquidity
  roles and `is_internal`.
  - Trade ids are UUID v8s derived from `(book, sequence)`, so they are identical on replay. They are
    also the ledger's settlement idempotency key.
  - Trades on a book are hash-chained: `chain_hash = SHA-256(prev_chain_hash || canonical_json(trade))`,
    with a fixed key order. `VerifyTradeChain` checks a book's series.
- **Sequence.** Each book keeps one sequence across its order events and trades. It strictly
  increases, so a consumer can detect gaps.

## 5. Event sourcing and replay

The **journal** is the ordered list of every command that changed state: valid submits (including
risk-rejected ones), successful cancels, and every day-expiry. Each submit carries its recorded risk
decision. Validation failures and idempotent duplicates are not journaled; they change nothing.

`Replay(cfg, journal)` rebuilds the engine and re-emits every event. Tests prove replay reproduces the
original event stream and final books exactly, including after a JSON round trip of the journal
(`TestReplayIsDeterministic`). No iteration in the matching path uses map order.

### Durable journal (`internal/journal`, migration `migrations/0001_journal.sql`)

- **Write-ahead.** `Config.Persist` is called with the command and its 1-based seq *before* any state
  changes. If the write fails, the command is refused with `ErrJournal` and the engine is untouched
  (not even an empty book is created). The books can never hold state the journal cannot reproduce.
- **Table.** `engine_journal(seq PK, kind, command_json, prev_hash, chain_hash, recorded_at)`.
  - Append-only: triggers refuse UPDATE, DELETE and TRUNCATE.
  - Hash-chained over the exact stored text: `chain_hash = SHA-256(prev_hash || command_json)`. The
    column is TEXT, not JSONB, so the hashed bytes are the stored bytes.
- **Single writer.** The seq primary key makes a second engine appending the same seq fail. Its
  command is refused rather than forking the books.
- **Recovery.** `journal.Recover` verifies the whole journal before replaying: contiguous seq, every
  link, every hash. A gap or edit returns `ErrCorrupt`, and the engine refuses to start rather than
  build different books. After recovery, new commands continue at n+1.
- **Cost.** A synchronous append is about 0.4 ms per order on local Postgres (fsync on;
  `BenchmarkSubmitJournaled`). That is within the 10 ms acceptance budget; a networked database adds
  its round trip. Group commit is the lever if it is ever needed.
- **Tests.** Integration tests run against real Postgres when `DATABASE_URL` is set, and skip
  otherwise. They cover:
  - crash then recover, giving identical books, orders and journal, with new work surviving a second
    restart;
  - DB-level refusal of UPDATE, DELETE and TRUNCATE;
  - detection of a superuser edit and of a gap;
  - a second writer being refused;
  - a database outage refusing commands.

## 6. Invariants (property-tested, 200 random streams × 400 commands)

- No book is ever crossed after a command.
- No trade has the same tenant on both sides.
- No trade mixes paper and real, and no paper trade has an internal counterparty.
- Every trade price is within both orders' limits.
- Each order's filled quantity equals the sum of its trades and never exceeds its quantity.
- Non-limit orders never rest.
- Per-book sequences strictly increase, and every book's trade chain verifies.
- Concurrent callers (8 goroutines, `-race`) leave a replayable journal and uncrossed books.

The suite was checked for teeth by breaking the engine on purpose: disabling self-trade prevention,
and executing at the taker's price. Both mutations were caught.

## 7. Not built yet (in order)

1. ~~**Durable journal.**~~ Done (§5). **Not yet deployed.** The service does not open a database
   until order entry is wired. At that point, add `DATABASE_URL`, the migration initContainer, and a
   `journal.Recover` call before the listener starts.
2. **Snapshots.** Every 5 minutes, snapshot books and sequences so replay starts from the snapshot
   rather than genesis. Redis as a read cache of depth for the API.
3. **Settlement.** On each trade, call credit-ledger to settle both sides atomically, idempotent on
   `trade_id`. Then publish `trades.executed.v1` and `orders.state.v1` to NATS.
4. **Risk hook implementation.** Balance and position-limit checks against the ledger, plus
   surveillance holds (KW05).
5. **API wiring.** Behind the licence gate: order entry, cancel, and orders/fills reads served from
   the engine instead of the Phase 1 simulator. Paper-only first.
6. **Performance.** Measure against the budget: acceptance P99 < 10 ms, match P99 < 5 ms. A single
   mutex serialises all books today. Per-book locking is the known optimisation, and must not change
   the order of events within a book.

## 8. Open questions for tech-lead (contracts not edited)

- **Sequence scope.** `orders.state.v1` says the sequence is "per-product". Because paper and real
  are separate books, the engine sequences per `(product_id, is_paper)`. Consumers should key gap
  detection on both. Proposed: clarify the contract wording.
- **Market maker on paper books.** The insider-risk rule stops internal accounts quoting into
  customer paper books. So paper liquidity cannot come from the internal market maker as specced.
  Options: a non-internal paper liquidity account that is clearly labelled, or accept thin paper
  books. KW04 needs this decided before it builds paper quoting.
- **Cancel reasons.** The engine emits `user_cancel`, `self_trade`, `unfilled_remainder`,
  `fok_unfilled` and `day_expired`. The contract's `reason` is free text with examples; proposed:
  enumerate these.
