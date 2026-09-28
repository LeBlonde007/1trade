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
  - Trades on a book are hash-chained: `chain_hash = SHA-256(prev_chain_hash || canonical_json(trade))`.
    The hashed row is exactly the `trades.executed.v1` payload fields, minus the two hash fields, in
    contract key order, with an empty `sub_account_id` rendered as `null`. So any consumer can
    re-verify the chain from the events alone (`TestTradeChainVerifiableFromPayloads`). `sequence`
    is not hashed: the contract does not carry it, and the prev links already fix the order.
- **Encoding.** `internal/events` encodes both contracts. `TestPayloadsMatchContracts` validates
  real engine output against the schemas read from `docs/contracts/events/*.yaml`, so contract drift
  fails the build. Nothing publishes yet.
- **Sequence.** Each book keeps one sequence across its order events and trades. It strictly
  increases, so a consumer can detect gaps.

## 5. Event sourcing and replay

The **journal** is the ordered list of every command that changed state: valid submits (including
risk-rejected ones), successful cancels, every day-expiry, and voids (§7.3). Each submit carries its
recorded risk decision. Validation failures and idempotent duplicates are not journaled; they change
nothing.

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
3. **Settlement and reservations — built.** Driven end to end against the real ledger.
   - **Holds.** At acceptance the engine computes each order's *hold* (`Order.Hold`):
     - a sell holds its credits;
     - a limit buy holds `floor(limit × qty)` plus the taker fee;
     - a market buy holds the exact cost of sweeping the current book, plus the taker fee.

     Property-tested: no order ever spends beyond its hold.
   - **Reserving.** `settle.ReserveRisk` is the risk hook: it reserves the hold via
     `/v1/credits/reserve` (credit.yaml v1.2). An order is rejected with `insufficient_credit` or
     `insufficient_cash`, or with `risk_unavailable` when the ledger can't answer — it fails closed.
     The decision is journaled.
   - **The worker.** `settle.Worker` applies the event stream in order: it settles each trade, and
     when a held order closes it releases the remainder. An order's terminal event always follows its
     trades, so fills settle before the leftover is freed.
     - Transient failures are retried.
     - A conflict halts at that event; it is idempotent to resume.
     - A refused trade is alerted, not stopped on. With reservations it should never happen.
   - **Epochs.** Trade and event ids mix in the journal's **epoch**, which is persisted write-once in
     `engine_meta` and loaded by `journal.Recover`. Without it, every fresh journal's first trade
     reused the same trade_id and the ledger refused it as a conflict. The cross-service test found
     this.
   - **Event publishing — built.** `internal/outbox`: **the journal is the outbox.** A relay follows
     the journal command by command (`journal.ReadAfter`, chain-verified). It re-derives each
     command's events on a shadow engine (`Engine.Apply` — replay is exact) and publishes them in
     emission order to the `TRADING_EVENTS` JetStream stream.
     - The Nats-Msg-Id is the event_id / trade_id, and the dedupe window is 10 min.
     - A Postgres cursor (`engine_outbox_cursor`, migration 0003) records (seq, idx) after each
       command.
     - Nothing in the order path waits on the network, and a crash loses nothing.
     - Tests:
       - the output is byte-identical to the live engine's emission;
       - resume after restart, and after a failure mid-command, gives no gap and no duplicate
         (mutation-checked);
       - the relay follows new entries;
       - the real path runs Postgres journal → relay → embedded JetStream in order, and a full
         republish after losing the cursor adds zero messages.
   - **Cutover wiring** (licence-gated):
     - `DATABASE_URL`, plus every migration (glob, in file order) in an initContainer;
     - the `ledger-settle` Secret mounted in the engine;
     - `journal.Recover` with `ReserveRisk`, and `OnUnjournaled: rec.Suspect` for a `Reconciler`
       running every 30 s (below), with `rec.Resume(e.Voided())` right after recovery and
       `Lister: client` so it sweeps the ledger's open reservations;
     - a `Worker` fed from the journal's events.
   - **Orphaned reservations — built (engine side).** The engine reserves an order's hold *before*
     writing the order to its journal. If that write fails, the ledger holds a reservation for an order
     that does not exist. Three parts close this:
     1. **Void** (journal kind `void`, migration `0004_void.sql`). A void is a durable tombstone for an
        order_id the journal has never seen.
        - A later submit with that id gets the tombstone back (rejected, reason `voided`). It never
          reaches the risk check, so nothing can claim the reservation again.
        - Voids emit no events. Voiding an id the engine knows changes nothing.
        - A void is write-ahead like any command. So an engine whose journal is behind the database
          (a write that committed without the engine seeing it) cannot void anything: its append
          collides on seq.
     2. **Reconciler** (`settle.Reconciler`). The engine reports every submit whose journal write
        failed after its risk check (`Config.OnUnjournaled`).
        - Grace period (default 2 min): a client retry with the same order_id reuses the reservation,
          and the order goes ahead.
        - After it, the reconciler voids the id, then releases the reservation. Either step is retried
          until it lands; once the id is void, the release cannot race an acceptance.
        - It never releases for an order that exists, or for an id another tenant owns.
     3. **The client checks what reserve returns.** The ledger answers a replay with the reservation's
        *current* state, so a replay of a released reservation comes back as a 200. The client now
        requires `state: open` and `remaining == amount == hold`. Otherwise the order is rejected with
        `order_id_reused` instead of being accepted unfunded. This is defence in depth: with voids,
        the engine never reserves a voided id again.

     Tests:
     - engine: void semantics, and submit-vs-void atomicity on 200 ids under `-race`. Voids are added
       to the 200-stream property run: a tombstone never appears in any event.
     - journal: a void survives restart, and a stale engine's void is refused on seq collision.
     - relay: steps over voids, both in memory and on Postgres → JetStream.
     - reconciler, each path:
       - void and release after the grace period;
       - a prompt retry keeps its reservation;
       - a journal outage delays the void;
       - a failed release is retried;
       - a rejected retry is released;
       - another tenant's id is left alone;
       - the replay of a closed reservation is refused.
     - a stress run: 8 clients with retries and cancels, a flapping journal, reserves that commit then
       fail, and a live reconciler. The journal is then replayed through the worker. Every live order
       ends with an open reservation, and no reservation is left without one.
     - the real credit-ledger binary.

     Mutation-checked. Each of these breaks a test:
     - releasing live orders, or another tenant's id;
     - releasing when the void failed;
     - dropping the response check, the grace period, the hook, or the void's journal write;
     - letting a submit through a tombstone;
     - restarting the grace period on a repeat report.

     **Crash recovery.** Suspects are held in memory, but tombstones are journaled. After recovery,
     `Reconciler.Resume(e.Voided())` re-releases every tombstone's reservation with no grace period.
     Release is idempotent, so one that landed before the crash is a no-op. That covers a crash
     between the void and the release.

     **Crash before the void — closed (credit.yaml v1.3).** The reconciler also sweeps the ledger's open
     reservations older than 10 minutes, at start and every 10 minutes (`Sweep`, `GET
     /v1/credits/reservations`). Every listed reservation goes through the same void-then-release
     path: one backing a live order is kept, and an orphan is voided and released. Releases carry an
     audit reason: `order_closed` from the worker, `orphaned` from the reconciler. Proven with a stress
     run where odd seeds recover with the suspects and even seeds "crash" and recover through the
     sweep alone, and against the real ledger.

     **Deployment invariant: one engine journal per ledger.** The sweep releases every reservation
     whose order this engine's journal does not know. Two engines (two journals) sharing one ledger
     would release each other's live reservations. A reset journal against a populated ledger is fine:
     its old orders no longer exist anywhere. Finding those needs the ledger to list open reservations. That is proposed
     in §8 (credit.yaml v1.3) and not yet approved.
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
- ~~**What if settlement is refused (402)?**~~ **Decided and built:** reserve at acceptance
  (credit.yaml v1.2). See §7.3.
- **Cancel and reject reasons.** The engine emits:
  - cancel reasons: `user_cancel`, `self_trade`, `unfilled_remainder`, `fok_unfilled` and `day_expired`;
  - reject reasons: `insufficient_credit`, `insufficient_cash`, `risk_unavailable` and
    `order_id_reused`.

  `voided` marks a tombstone. It is returned on a duplicate submit and never emitted as an event. The
  contract's `reason` is free text with examples; proposed: enumerate these.
- ~~**Proposed: credit.yaml v1.3**~~ **Authored and implemented 2026-09-28** (owner-approved). The
  changes, as proposed:
  - `GET /v1/credits/reservations?state=open&min_age_seconds=N&after=<order_id>&limit=N`.
    - matching-engine only (the settle token). Keyset-paginated; the age is measured on the ledger's
      clock.
    - Returns `{data: [Reservation], next_cursor}`; `Reservation` gains `created_at`.
    - The reconciler would page through it at start and periodically, and `Suspect` every open
      reservation whose order the engine does not know. The void-then-release path then handles it
      unchanged.
  - An optional `reason` on `/v1/credits/release` (`order_closed` | `orphaned`), recorded on the
    reservation's audit event. A repair is then distinguishable from a normal close in the ledger's
    own audit trail.
  - Make a reserve replay of a released reservation `409 RESERVATION_CLOSED`, instead of a 200
    carrying the released state. The client already refuses that 200 (§7.3), so this is about
    clarity, not safety.

  Alternative without a contract change: the engine writes an intent row to its own database before
  every reserve, and reconciles intents the journal never took. The cost is a second synchronous
  write per order in the acceptance path. The listing is cheaper, and it keeps the ledger as the
  source of truth for locks.

  Affected: credit-ledger (implements it), matching-engine (consumes it), and security-compliance
  (review; it touches credits).
