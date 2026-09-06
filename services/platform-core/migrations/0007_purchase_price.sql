-- 0007 — persist the price a purchase was actually charged at.
--
-- `purchases` recorded how many credits were bought but never what was paid for them: the USD was
-- derived at checkout from billing.refUSDPerCredit and handed to Stripe, then discarded. Two
-- consequences:
--
--   1. If that reference table ever changes, past purchases become unreconstructable — "what did
--      this customer pay per credit?" has no answer from the database.
--   2. KW01's index is specified to observe "prepaid purchase prices". There were none stored to
--      observe, so an index built today could only restate the current price table back to itself.
--
-- Recording the price AT the transaction makes the observation real and the history auditable.
-- Both columns are nullable: rows written before this migration genuinely have no recorded price,
-- and back-filling them from today's table would fabricate history — exactly what the index must
-- not be built on.
ALTER TABLE purchases
    ADD COLUMN IF NOT EXISTS unit_price_usd    NUMERIC(20,6),  -- USD per credit, at purchase time
    ADD COLUMN IF NOT EXISTS charged_usd_cents BIGINT;         -- integer cents actually charged

COMMENT ON COLUMN purchases.unit_price_usd IS
    'USD per credit at the moment of purchase. NULL for rows created before migration 0007 — never back-filled, because an inferred price is not an observation.';
COMMENT ON COLUMN purchases.charged_usd_cents IS
    'Integer USD cents charged. Integer, not NUMERIC: this is the exact amount sent to the payment processor.';

-- Index observations select priced, settled purchases within a window.
CREATE INDEX IF NOT EXISTS idx_purchases_priced
    ON purchases (credit_type, paid_at DESC)
    WHERE unit_price_usd IS NOT NULL AND status = 'paid';
