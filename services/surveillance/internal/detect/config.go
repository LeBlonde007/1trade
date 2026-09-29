package detect

import "time"

// Config holds every threshold. Nothing in a rule is a magic number; defaults are a conservative v1
// starting point meant to be tuned on real flow (KW05 acceptance: "tunable thresholds").
type Config struct {
	// BeneficialOwner maps a tenant to its beneficial owner (linked orgs, a trader's several accounts).
	// Unmapped tenants are their own owner.
	BeneficialOwner map[string]string

	// Wash trades: round trip A→B then B→A within WashWindow, quantities within WashQtyTolerancePPM of
	// each other and prices within WashPriceBps.
	// An exact round trip (same size, same price) alerts at once; a near one only when the same pair
	// repeats it WashMinNearPairs times in the window — one near match happens by chance in busy
	// honest flow (the benign-market test found one in ~20k trades).
	WashWindow          time.Duration
	WashQtyTolerancePPM int64
	WashPriceBps        int64
	WashMinNearPairs    int

	// Spoofing: an order at least SpoofSizeMultiple × the product's recent average order size (with at
	// least SpoofMinSamples orders seen), cancelled within SpoofMaxLifetime having filled at most
	// SpoofMaxFillPPM of its quantity, while the tenant traded the opposite side during its life.
	SpoofSizeMultiple int64
	SpoofMinSamples   int
	SpoofMaxLifetime  time.Duration
	SpoofMaxFillPPM   int64

	// Layering: within LayerWindow, at least LayerMinOrders unfilled orders cancelled on one side at at
	// least LayerMinLevels distinct prices, while the tenant traded the opposite side.
	LayerWindow    time.Duration
	LayerMinOrders int
	LayerMinLevels int

	// Marking the close: in the CloseWindow before the daily index print at CloseHourUTC, a tenant's
	// share of a product's volume at least CloseSharePPM, with window volume at least CloseMinVolume.
	CloseHourUTC   int
	CloseWindow    time.Duration
	CloseSharePPM  int64
	CloseMinVolume Fixed

	// Cross-product: Related[A] lists products whose value depends on A. Within CrossWindow, a tenant
	// that is at least CrossSharePPM of A's aggressor volume and moved A's price at least
	// CrossMinMoveBps in its direction, while holding a same-direction position in a related product.
	Related         map[string][]string
	CrossWindow     time.Duration
	CrossSharePPM   int64
	CrossMinMoveBps int64

	// Excessive cancellation: within OTRWindow, at least OTRMinOrders orders on a product and an
	// order-to-trade ratio of at least OTRThreshold.
	OTRWindow    time.Duration
	OTRMinOrders int
	OTRThreshold Fixed

	// Position limits: |net position| per product (MaxPosition[product], else DefaultMaxPosition) and
	// gross exposure Σ|position| × last price (MaxGrossNotional). Zero disables a limit.
	MaxPosition        map[string]Fixed
	DefaultMaxPosition Fixed
	MaxGrossNotional   Fixed
	LimitBucket        time.Duration // one position-limit alert per tenant/product per bucket

	// PaperQuoters are accounts that quote paper books by design — the matching engine's paper
	// liquidity account, which re-quotes a ladder on both sides of every book. On PAPER books only,
	// their order patterns (spoofing, layering, excessive cancellation) are not alerted: that is how
	// quoting looks, and flagging it would bury real alerts. Every trade-based rule (wash trades,
	// marking the close, cross-product, position limits) still applies to them, and on real books
	// they get no exemption at all.
	PaperQuoters map[string]bool
}

// DefaultConfig is the v1 starting point.
func DefaultConfig() Config {
	return Config{
		// Round trips: a wash nets the two parties back to (near) flat, so size must match within 0.1%.
		// Looser (1% / 20 bps / 10 min) raised 15 false alerts in the benign-market test.
		WashWindow: 5 * time.Minute, WashQtyTolerancePPM: 1_000, WashPriceBps: 5, WashMinNearPairs: 2,
		SpoofSizeMultiple: 10, SpoofMinSamples: 5, SpoofMaxLifetime: 30 * time.Second, SpoofMaxFillPPM: 100_000,
		LayerWindow: time.Minute, LayerMinOrders: 4, LayerMinLevels: 3,
		CloseHourUTC: 16, CloseWindow: 30 * time.Minute, CloseSharePPM: 500_000, CloseMinVolume: 1000 * unit,
		Related: map[string][]string{
			"TEXT-SPOT": {"EAI-IDX"}, "SPEECH-SPOT": {"EAI-IDX"}, "IMAGE-SPOT": {"EAI-IDX"},
			"VIDEO-SPOT": {"EAI-IDX"}, "EMBED-SPOT": {"EAI-IDX"}, "H100-SPOT": {"H100-FWD-30D"},
		},
		CrossWindow: 10 * time.Minute, CrossSharePPM: 600_000, CrossMinMoveBps: 100,
		OTRWindow: 5 * time.Minute, OTRMinOrders: 20, OTRThreshold: 20 * unit,
		LimitBucket: 24 * time.Hour,
	}
}

// maxWindow is how long any rule needs history — state older than this is pruned.
func (c Config) maxWindow() time.Duration {
	m := c.WashWindow
	for _, d := range []time.Duration{c.SpoofMaxLifetime, c.LayerWindow, c.CloseWindow, c.CrossWindow, c.OTRWindow} {
		m = max(m, d)
	}
	return m
}
