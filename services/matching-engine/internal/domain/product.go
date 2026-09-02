// Package domain holds the matching engine's core types: the tradeable product catalog and the
// exchange-status value that tells clients why order entry is unavailable.
//
// The catalog is fixed in Phase 1 and derived from the canonical credit-type enum in
// docs/contracts/credit-types.md §1 — the exchange introduces no new credit type (§5), so every
// product maps onto a type the ledger already knows how to move. Prices here are *reference* levels
// the market-data simulator walks around; they are not quotes and nothing settles against them.
package domain

// Credit types this exchange trades. Mirrors docs/contracts/credit-types.md §1 — never invent one.
const (
	CreditAIIndex    = "ai_index"
	CreditText       = "text"
	CreditSpeech     = "speech"
	CreditImage      = "image"
	CreditVideo      = "video"
	CreditEmbeddings = "embeddings"
	CreditH100       = "gpu_h100"
	CreditH200       = "gpu_h200"
)

// Product types from the spec's products table (phase6_v2.md §9.1).
const (
	TypeSpot    = "spot"
	TypeForward = "forward"
)

// Product is one tradeable instrument, matching the Product schema in openapi/trading.yaml.
type Product struct {
	ID              string
	ProductType     string
	CreditType      string
	Name            string
	Description     string
	Family          string
	UnderlyingAsset string
	DeliveryDate    string // ISO date, forwards only; empty on spot
	ContractSize    string // fixed-point string, forwards only; empty on spot
	TickSize        string // fixed-point string
	QuotePrecision  int    // decimal places the client should display
	Tradeable       bool   // the product's own flag — forwards stay false until v1.5

	// Reference is the level the simulator's price walk mean-reverts to, and Vol is its relative
	// amplitude. Simulation inputs only — never a quote, never a settlement price.
	Reference float64
	Vol       float64
	// BaseVolume24h anchors the simulated 24h volume for this product.
	BaseVolume24h float64
}

// catalog is the Phase 1 product set: the index spot, one spot per sub-credit, one spot per GPU tier,
// and a dated H100 forward (not tradeable — forwards are v1.5 per phase6_v2.md §non-goals).
var catalog = []Product{
	{
		ID: "EAI-IDX", ProductType: TypeSpot, CreditType: CreditAIIndex,
		Name: "AI Index", Description: "AI Index · spot", Family: "Index",
		TickSize: "0.000001", QuotePrecision: 6, Tradeable: true,
		Reference: 0.001005, Vol: 0.012, BaseVolume24h: 12_400_000,
	},
	{
		ID: "TEXT-SPOT", ProductType: TypeSpot, CreditType: CreditText,
		Name: "Text credit", Description: "Text credit · spot", Family: "Text",
		TickSize: "0.000001", QuotePrecision: 6, Tradeable: true,
		Reference: 0.001210, Vol: 0.018, BaseVolume24h: 8_120_000,
	},
	{
		ID: "SPEECH-SPOT", ProductType: TypeSpot, CreditType: CreditSpeech,
		Name: "Speech credit", Description: "Speech credit · spot", Family: "Speech",
		TickSize: "0.000001", QuotePrecision: 6, Tradeable: true,
		Reference: 0.001200, Vol: 0.016, BaseVolume24h: 2_140_000,
	},
	{
		ID: "IMAGE-SPOT", ProductType: TypeSpot, CreditType: CreditImage,
		Name: "Image credit", Description: "Image credit · spot", Family: "Image",
		TickSize: "0.000001", QuotePrecision: 6, Tradeable: true,
		Reference: 0.008000, Vol: 0.022, BaseVolume24h: 5_810_000,
	},
	{
		ID: "VIDEO-SPOT", ProductType: TypeSpot, CreditType: CreditVideo,
		Name: "Video credit", Description: "Video credit · spot", Family: "Video",
		TickSize: "0.000100", QuotePrecision: 4, Tradeable: true,
		Reference: 0.250000, Vol: 0.020, BaseVolume24h: 1_840_000,
	},
	{
		ID: "EMBED-SPOT", ProductType: TypeSpot, CreditType: CreditEmbeddings,
		Name: "Embeddings credit", Description: "Embeddings credit · spot", Family: "Embeddings",
		TickSize: "0.000001", QuotePrecision: 6, Tradeable: true,
		Reference: 0.000400, Vol: 0.024, BaseVolume24h: 420_000,
	},
	{
		ID: "H100-SPOT", ProductType: TypeSpot, CreditType: CreditH100,
		Name: "H100 GPU-hour", Description: "H100 GPU-hour · spot", Family: "GPU",
		TickSize: "0.010000", QuotePrecision: 2, Tradeable: true,
		Reference: 2.990000, Vol: 0.014, BaseVolume24h: 9_800_000,
	},
	{
		ID: "H200-SPOT", ProductType: TypeSpot, CreditType: CreditH200,
		Name: "H200 GPU-hour", Description: "H200 GPU-hour · spot", Family: "GPU",
		TickSize: "0.010000", QuotePrecision: 2, Tradeable: true,
		Reference: 3.840000, Vol: 0.015, BaseVolume24h: 6_180_000,
	},
	{
		ID: "H100-FWD-30D", ProductType: TypeForward, CreditType: CreditH100,
		Name: "H100 30-day forward", Description: "H100 GPU-hour · 30-day forward", Family: "GPU",
		UnderlyingAsset: CreditH100, ContractSize: "1.000000",
		TickSize: "0.010000", QuotePrecision: 2, Tradeable: false,
		Reference: 3.040000, Vol: 0.016, BaseVolume24h: 2_410_000,
	},
}

// Catalog returns every product in listing order. The slice is copied so a caller cannot mutate the
// package-level catalog.
func Catalog() []Product {
	out := make([]Product, len(catalog))
	copy(out, catalog)
	return out
}

// Find returns the product with the given id and whether it exists. Lookup is exact and
// case-sensitive: product ids are contract identifiers, not user input to be normalized.
func Find(id string) (Product, bool) {
	for _, p := range catalog {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}
