-- 0005_reservations_open_index.sql — credit.yaml v1.3: the engine's reconciler pages through open
-- reservations in order_id order (GET /v1/credits/reservations). A partial index keeps that scan to the
-- open rows, however many released ones accumulate.
-- Rollback: DROP INDEX idx_reservations_open_order (safe: it only serves the listing).
CREATE INDEX IF NOT EXISTS idx_reservations_open_order ON reservations (order_id) WHERE state = 'open';
