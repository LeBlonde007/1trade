// Package catalog is the curated model catalog the gateway serves at GET /v1/models. It is the
// single source of truth for which models exist, what modality/sub-credit each bills, and the
// fixed-point price per unit. Real vLLM-backed models (F09) register here; the shape never changes.
package catalog

// Pricing is the Exascale billing extension on a catalog entry (the `exascale` object in the
// OpenAPI Model schema). Price is a fixed-point decimal string (credits per unit) — never a float.
type Pricing struct {
	Modality   string `json:"modality"`
	CreditType string `json:"credit_type"`
	Unit       string `json:"unit"`
	Price      string `json:"price"`
}

// Model is one catalog entry, OpenAI-shaped (`id`/`object`/`owned_by`) plus the Exascale pricing
// extension. `Created` is a fixed catalog-epoch timestamp (catalog entries are not per-request).
type Model struct {
	ID       string  `json:"id"`
	Object   string  `json:"object"`
	Created  int64   `json:"created"`
	OwnedBy  string  `json:"owned_by"`
	Exascale Pricing `json:"exascale"`
}

// catalogEpoch is a stable `created` value for catalog entries (2026-01-01T00:00:00Z).
const catalogEpoch int64 = 1767225600

// models is the curated catalog. M2 launches the first three (Llama-70B/8B + Whisper) per F09;
// more register here behind the same shape with no API change.
var models = []Model{
	{
		ID: "llama-3.1-70b", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "25.000000"},
	},
	{
		ID: "llama-3.1-8b", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "5.000000"},
	},
	{
		ID: "whisper-large-v3", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "speech", CreditType: "speech", Unit: "1 minute", Price: "10.000000"},
	},
}

// List returns the full curated catalog (defensive copy so callers can't mutate the source).
func List() []Model {
	out := make([]Model, len(models))
	copy(out, models)
	return out
}

// Lookup returns the catalog entry for a model id and whether it exists.
func Lookup(id string) (Model, bool) {
	for _, m := range models {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}
