-- 0002_conversion.sql — F07 credit conversion. The conversion-rate source (ADR-0002: bidirectional,
-- 1% house spread). `rate` is `to`-units per 1 `from`-unit, pre-spread; the ledger reads it here so
-- nothing is hardcoded. M2 ships ai_index↔text; M3 adds the other modalities + GPU tiers. Rates are
-- anchored on the relative credit prices in credit-types.md.

CREATE TABLE IF NOT EXISTS conversion_rates (
    from_type  TEXT NOT NULL,
    to_type    TEXT NOT NULL,
    rate       NUMERIC(20,6) NOT NULL CHECK (rate > 0),
    spread     NUMERIC(10,6) NOT NULL DEFAULT 0.010000 CHECK (spread >= 0 AND spread < 1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (from_type, to_type)
);

-- ai_index ≈ $0.001005, text ≈ $0.001210 → 1 AI buys 0.001005/0.001210 ≈ 0.830579 text; inverse ≈ 1.203980.
INSERT INTO conversion_rates (from_type, to_type, rate) VALUES
    ('ai_index', 'text', 0.830579),
    ('text', 'ai_index', 1.203980)
ON CONFLICT (from_type, to_type) DO NOTHING;
