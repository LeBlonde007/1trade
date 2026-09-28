-- 0004_void.sql — the `void` journal command (SPEC.md §7.3): a tombstone for an order_id whose submit
-- never reached the journal although the ledger had already reserved its hold. Once an id is void it
-- can never be accepted, so the reconciler can release the orphaned reservation without racing a retry.
-- This only widens the kind CHECK; command_json carries the void like any other command.
-- Rollback: forward-only. An engine binary older than this migration cannot replay a journal that
-- holds a void: it refuses to start ("empty command") rather than rebuild different books.

ALTER TABLE engine_journal DROP CONSTRAINT IF EXISTS engine_journal_kind_check;
ALTER TABLE engine_journal ADD CONSTRAINT engine_journal_kind_check
    CHECK (kind IN ('submit', 'cancel', 'expire_day', 'void'));
