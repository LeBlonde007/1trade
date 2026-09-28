-- 0011_payments.sql — pay by card, ACH bank debit or wire transfer; US dollars only
-- (platform-core.yaml v1.9). ACH settles days after checkout, so a purchase waits in 'processing'
-- until the bank confirms; a wire waits in 'pending' until treasury records the money, matched by a
-- unique reference. Credits are booked at most once per purchase. Rollback: forward-only in prod.
ALTER TABLE purchases ADD COLUMN IF NOT EXISTS method TEXT NOT NULL DEFAULT 'card';
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS purchases_method_check;
ALTER TABLE purchases ADD CONSTRAINT purchases_method_check CHECK (method IN ('card', 'ach', 'wire'));

ALTER TABLE purchases DROP CONSTRAINT IF EXISTS purchases_status_check;
ALTER TABLE purchases ADD CONSTRAINT purchases_status_check CHECK (status IN ('pending', 'processing', 'paid', 'failed'));

-- USD only from now on (older rows are left as they were).
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS purchases_usd_only;
ALTER TABLE purchases ADD CONSTRAINT purchases_usd_only CHECK (currency = 'usd') NOT VALID;

-- Wire transfers: the reference the customer puts on the wire, and what treasury recorded.
ALTER TABLE purchases ADD COLUMN IF NOT EXISTS wire_reference      TEXT UNIQUE;
ALTER TABLE purchases ADD COLUMN IF NOT EXISTS wire_received_cents BIGINT CHECK (wire_received_cents > 0);
ALTER TABLE purchases ADD COLUMN IF NOT EXISTS bank_reference      TEXT;
ALTER TABLE purchases ADD COLUMN IF NOT EXISTS failure_reason      TEXT;
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS purchases_wire_shape;
ALTER TABLE purchases ADD CONSTRAINT purchases_wire_shape CHECK (
    (method = 'wire') = (wire_reference IS NOT NULL) AND
    (method <> 'wire' OR charged_usd_cents IS NOT NULL) AND
    (wire_received_cents IS NULL OR wire_received_cents = charged_usd_cents));
