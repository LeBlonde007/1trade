// Package catalog is the curated model catalog the gateway serves at GET /v1/models. It is the
// single source of truth for which models exist, what modality/sub-credit each bills, and the
// fixed-point price per unit. Real vLLM-backed models (F09) register here; the shape never changes.
package catalog

// Pricing is the 1Trade billing extension on a catalog entry (the `trade1` object in the
// OpenAPI Model schema). Price is a fixed-point decimal string (credits per unit) — never a float.
type Pricing struct {
	Modality   string `json:"modality"`
	CreditType string `json:"credit_type"`
	Unit       string `json:"unit"`
	Price      string `json:"price"`
}

// Model is one catalog entry, OpenAI-shaped (`id`/`object`/`owned_by`) plus a human display `name` and
// the 1Trade pricing extension. `Created` is a fixed catalog-epoch timestamp (not per-request).
type Model struct {
	ID      string  `json:"id"`
	Object  string  `json:"object"`
	Name    string  `json:"name"`
	Created int64   `json:"created"`
	OwnedBy string  `json:"owned_by"`
	Trade1  Pricing `json:"trade1"`
}

// catalogEpoch is a stable `created` value for catalog entries (2026-01-01T00:00:00Z).
const catalogEpoch int64 = 1767225600

// models is the curated catalog — the customer-facing menu. Every entry is served on 1Trade infra
// (owned_by=1trade); the upstream that physically runs it is an implementation detail mapped at the
// gateway (INFERENCE_MODEL_MAP). All are chat/text and bill in `text` credits. Add a row here + its
// provider-slug to the model map and it just works — no API change.
var models = []Model{
	{
		ID: "llama-3.1-70b", Name: "Llama 3.1 70B", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "25.000000"},
	},
	{
		ID: "llama-3.1-8b", Name: "Llama 3.1 8B", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "5.000000"},
	},
	// Self-hosted small models — the first entries actually served on our own GPUs rather than a
	// hosted provider. They are listed under their real size so the playground never labels a 1B
	// model as something larger; priced below the hosted tiers because they cost us far less to run.
	{
		ID: "llama-3.2-1b", Name: "Llama 3.2 1B", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "1.000000"},
	},
	{
		ID: "qwen2.5-1.5b", Name: "Qwen2.5 1.5B", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "1.500000"},
	},
	{
		ID: "claude-opus-4.8", Name: "Claude Opus 4.8", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "80.000000"},
	},
	{
		ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "30.000000"},
	},
	{
		ID: "gpt-5", Name: "GPT-5", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "40.000000"},
	},
	{
		ID: "gpt-4o", Name: "GPT-4o", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "25.000000"},
	},
	{
		ID: "deepseek-v3.2", Name: "DeepSeek V3.2", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "8.000000"},
	},
	{
		ID: "qwen3-32b", Name: "Qwen3 32B", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "text", CreditType: "text", Unit: "1K tokens", Price: "6.000000"},
	},
	// Vision (image/screen understanding → text). Bills in `text` credits — the output is text. Served
	// via POST /v1/chat/vision with an image_url content part.
	{
		ID: "nemotron-vision", Name: "Nemotron Vision", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "vision", CreditType: "text", Unit: "1K tokens", Price: "10.000000"},
	},
	// Document writer (text → Word / Excel / PowerPoint / PDF / Markdown). Generates the content over the
	// chat path; the client renders + downloads the file. Bills in `text` credits — the output is text.
	{
		ID: "doc-writer", Name: "Document Writer", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "docs", CreditType: "text", Unit: "1K tokens", Price: "6.000000"},
	},
	// Code generation (text → raw code: HTML/CSS/JS/TS/Python). Runs over the chat path, billed in text
	// credits; the console renders it with a ▶ Run + live preview.
	{
		ID: "code-writer", Name: "Code Writer", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "code", CreditType: "text", Unit: "1K tokens", Price: "8.000000"},
	},
	// Autonomous computer-use agent (text instruction → browser/computer actions). Listed for the catalog;
	// the runner is not live yet, so the console surfaces it as "coming soon".
	{
		ID: "agent-operator", Name: "Operator Agent", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "agent", CreditType: "text", Unit: "task", Price: "20.000000"},
	},
	// Image generation (billed in `image` credits, per image). Inline playground is chat-only — these
	// run via the image API/SDK; the catalog surfaces them so the marketplace reads as multi-modal.
	{
		ID: "stable-diffusion-3.5", Name: "Stable Diffusion 3.5", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "80.000000"},
	},
	{
		ID: "flux-schnell", Name: "Flux Schnell", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "30.000000"},
	},
	{
		ID: "gpt-image-1.5", Name: "GPT Image 1.5", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "image", CreditType: "image", Unit: "1 image", Price: "120.000000"},
	},
	// Speech / audio (text-to-speech), billed in `speech` credits per 1K characters.
	{
		ID: "elevenlabs-tts", Name: "ElevenLabs TTS Multilingual", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "speech", CreditType: "speech", Unit: "1K chars", Price: "2.000000"},
	},
	{
		ID: "qwen3-tts", Name: "Qwen3 TTS", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "speech", CreditType: "speech", Unit: "1K chars", Price: "1.000000"},
	},
	// Video (text-to-video), billed in `video` credits per clip.
	{
		ID: "wan-t2v", Name: "Wan 2.2 Text-to-Video", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "video", CreditType: "video", Unit: "1 video", Price: "600.000000"},
	},
	// Embeddings, billed in `embeddings` credits per 1M tokens.
	{
		ID: "bge-m3", Name: "BGE M3", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "embeddings", CreditType: "embeddings", Unit: "1M tokens", Price: "2.000000"},
	},
	{
		ID: "e5-large", Name: "E5 Large v2", Object: "model", Created: catalogEpoch, OwnedBy: "1trade",
		Trade1: Pricing{Modality: "embeddings", CreditType: "embeddings", Unit: "1M tokens", Price: "2.000000"},
	},
}

// List returns the full curated catalog (defensive copy so callers can't mutate the source).
func List() []Model {
	out := make([]Model, len(models))
	copy(out, models)
	return out
}

// ListServable returns only the entries the configured backend can actually route, given the
// gateway's INFERENCE_MODEL_MAP. The catalog is one static list shared by every deployment, but
// what each can serve differs — a self-hosted GPU runs two small models while a hosted provider
// covers most of the list. Advertising the whole catalog everywhere meant a customer could pick a
// model that had no upstream and get a 500 from the provider's 404.
//
// An EMPTY map means pass-through (the CPU stub, or a provider whose slugs already equal our ids),
// so everything is servable and the full list is returned — never filter down to nothing.
func ListServable(modelMap map[string]string) []Model {
	if len(modelMap) == 0 {
		return List()
	}
	out := make([]Model, 0, len(modelMap))
	for _, m := range models {
		if _, ok := modelMap[m.ID]; ok {
			out = append(out, m)
		}
	}
	return out
}

// IsServable reports whether this deployment can route a model id. Same empty-map rule as
// ListServable. Used to refuse an unroutable model with a clear 404 rather than letting the
// request reach an upstream that will reject it.
func IsServable(id string, modelMap map[string]string) bool {
	if len(modelMap) == 0 {
		return true
	}
	_, ok := modelMap[id]
	return ok
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
