package catalog_test

import (
	"testing"

	"github.com/trade1/inference-gateway/internal/catalog"
)

// TestListServableFiltersToRoutableModels pins the rule that keeps the catalogue honest per
// deployment: the static list is shared by every environment, but each can only route what its
// INFERENCE_MODEL_MAP covers. Advertising the rest meant a customer could pick a model with no
// upstream and receive a 500 from the provider's 404.
func TestListServableFiltersToRoutableModels(t *testing.T) {
	// The self-hosted GPU deployment: two small models, nothing else.
	local := map[string]string{"llama-3.2-1b": "llama3.2:1b", "qwen2.5-1.5b": "qwen2.5:1.5b"}

	got := catalog.ListServable(local)
	if len(got) != 2 {
		t.Fatalf("servable = %d models, want 2", len(got))
	}
	for _, m := range got {
		if _, ok := local[m.ID]; !ok {
			t.Errorf("returned %q, which is not in the model map", m.ID)
		}
	}

	// Entries keep their catalogue metadata — filtering must not change pricing or modality.
	for _, m := range got {
		if m.ID == "llama-3.2-1b" {
			if m.Trade1.CreditType != "text" || m.Trade1.Price != "1.000000" {
				t.Errorf("llama-3.2-1b pricing = %s %s, want text 1.000000", m.Trade1.Price, m.Trade1.CreditType)
			}
		}
	}
}

// TestEmptyMapMeansPassThrough — an empty map is the CPU stub (or a provider whose slugs already
// equal our ids), where everything is routable. Filtering to nothing there would empty the
// catalogue and break the default local stack.
func TestEmptyMapMeansPassThrough(t *testing.T) {
	full := len(catalog.List())
	if got := len(catalog.ListServable(nil)); got != full {
		t.Errorf("nil map → %d models, want the full %d", got, full)
	}
	if got := len(catalog.ListServable(map[string]string{})); got != full {
		t.Errorf("empty map → %d models, want the full %d", got, full)
	}
	if !catalog.IsServable("gpt-5", nil) {
		t.Error("IsServable must be permissive when no map is configured")
	}
}

// TestIsServableGuardsUnroutableIDs is what turns an upstream 500 into a clear 404.
func TestIsServableGuardsUnroutableIDs(t *testing.T) {
	local := map[string]string{"llama-3.2-1b": "llama3.2:1b"}
	if !catalog.IsServable("llama-3.2-1b", local) {
		t.Error("mapped id must be servable")
	}
	if catalog.IsServable("gpt-5", local) {
		t.Error("unmapped id must NOT be servable — it has no upstream on this deployment")
	}
	// An id absent from the catalogue entirely is likewise not servable.
	if catalog.IsServable("no-such-model", local) {
		t.Error("unknown id must not be reported servable")
	}
}
