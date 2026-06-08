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

// Model is one catalog entry, OpenAI-shaped (`id`/`object`/`owned_by`) plus a human display `name` and
// the Exascale pricing extension. `Created` is a fixed catalog-epoch timestamp (not per-request).
type Model struct {
	ID       string  `json:"id"`
	Object   string  `json:"object"`
	Name     string  `json:"name"`
	Created  int64   `json:"created"`
	OwnedBy  string  `json:"owned_by"`
	Exascale Pricing `json:"exascale"`
}

// catalogEpoch is a stable `created` value for catalog entries (2026-01-01T00:00:00Z).
const catalogEpoch int64 = 1767225600

// models is the curated catalog — the customer-facing menu. Every entry is served on Exascale infra
// (owned_by=exascale); the upstream that physically runs it is an implementation detail mapped at the
// gateway (INFERENCE_MODEL_MAP). All are chat/text and bill in `text` credits. Add a row here + its
// provider-slug to the model map and it just works — no API change.
var models = []Model{
	{
		ID: "llama-3.1-70b", Name: "Llama 3.1 70B", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "25.000000"},
	},
	{
		ID: "llama-3.1-8b", Name: "Llama 3.1 8B", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "5.000000"},
	},
	{
		ID: "claude-opus-4.8", Name: "Claude Opus 4.8", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "80.000000"},
	},
	{
		ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "30.000000"},
	},
	{
		ID: "gpt-5", Name: "GPT-5", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "40.000000"},
	},
	{
		ID: "gpt-4o", Name: "GPT-4o", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "25.000000"},
	},
	{
		ID: "deepseek-v3.2", Name: "DeepSeek V3.2", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "8.000000"},
	},
	{
		ID: "qwen3-32b", Name: "Qwen3 32B", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "6.000000"},
	},
	// Image generation (billed in `image` credits, per image). Inline playground is chat-only — these
	// run via the image API/SDK; the catalog surfaces them so the marketplace reads as multi-modal.
	{
		ID: "stable-diffusion-3.5", Name: "Stable Diffusion 3.5", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "80.000000"},
	},
	{
		ID: "flux-schnell", Name: "Flux Schnell", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "30.000000"},
	},
	{
		ID: "gpt-image-1.5", Name: "GPT Image 1.5", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "120.000000"},
	},
	// Speech / audio (text-to-speech), billed in `speech` credits per 1K characters.
	{
		ID: "elevenlabs-tts", Name: "ElevenLabs TTS Multilingual", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "speech", CreditType: "speech", Unit: "1K chars", Price: "2.000000"},
	},
	{
		ID: "qwen3-tts", Name: "Qwen3 TTS", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "speech", CreditType: "speech", Unit: "1K chars", Price: "1.000000"},
	},
	// Video (text-to-video), billed in `video` credits per clip.
	{
		ID: "wan-t2v", Name: "Wan 2.2 Text-to-Video", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "video", CreditType: "video", Unit: "1 video", Price: "600.000000"},
	},
	// Embeddings, billed in `embeddings` credits per 1M tokens.
	{
		ID: "bge-m3", Name: "BGE M3", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "embeddings", CreditType: "embeddings", Unit: "1M tokens", Price: "2.000000"},
	},
	{
		ID: "e5-large", Name: "E5 Large v2", Object: "model", Created: catalogEpoch, OwnedBy: "exascale",
		Exascale: Pricing{Modality: "embeddings", CreditType: "embeddings", Unit: "1M tokens", Price: "2.000000"},
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
